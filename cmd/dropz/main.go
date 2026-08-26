package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dropz/dropz/internal/logging"
	"github.com/dropz/dropz/internal/manager"
	"github.com/dropz/dropz/internal/server"
)

// getDefaultPath expands home directory and returns the full path
func getDefaultPath(path string) string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, path)
	}
	return path // fallback to relative path if home dir can't be determined
}

var (
	dataDir     = flag.String("data-dir", getDefaultPath(".dropz/data"), "Directory for storing data")
	videoDir    = flag.String("video-dir", getDefaultPath("Videos"), "Directory for storing downloaded videos")
	logDir      = flag.String("log-dir", getDefaultPath(".dropz/logs"), "Directory for storing logs")
	logLevel    = flag.String("log-level", "info", "Log level (trace, debug, info, warn, error)")
	logFormat   = flag.String("log-format", "text", "Log format (text, json); json is meant for the Electron host")
	serverAddr  = flag.String("server-addr", "127.0.0.1:50051", "gRPC server address")
	showVersion = flag.Bool("version", false, "Show version and exit")
	syncEnabled = flag.Bool("sync-enabled", true, "Enable automatic content synchronization")
)

// Set via -ldflags "-X main.appVersion=... -X main.buildTime=..." (see Makefile).
// Must stay vars: -X cannot override consts.
var (
	appVersion = "dev"
	buildTime  = "unknown"
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("dropz version %s (built %s)\n", appVersion, buildTime)
		os.Exit(0)
	}

	// Create directories if they don't exist
	for _, dir := range []string{*dataDir, *videoDir, *logDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Initialize logger
	logFile := filepath.Join(*logDir, "dropz.log")
	log, levelVar, err := logging.New(logging.Options{
		Level:    *logLevel,
		Format:   *logFormat,
		FilePath: logFile,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	log = log.With("version", appVersion, "pid", os.Getpid())
	mainLog := log.With("component", "main")

	dbPath := filepath.Join(*dataDir, "dropz.db")
	mainLog.Info("Starting dropz",
		"built", buildTime,
		"data_dir", *dataDir,
		"video_dir", *videoDir,
		"log_file", logFile,
		"db", dbPath,
		"addr", *serverAddr,
		"level", logging.LevelName(levelVar.Level()),
	)

	// Initialize GoPro manager
	goProManager, err := manager.NewGoProManager(dbPath, *videoDir, log, levelVar)
	if err != nil {
		mainLog.Error("Failed to initialize GoPro manager", "err", err)
		os.Exit(1)
	}

	// Only override config settings if flags were explicitly set
	config := goProManager.GetConfig()

	// Track if any flag was explicitly set
	configChanged := false

	// Check which flags were explicitly set by user
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "sync-enabled":
			config.SyncEnabled = *syncEnabled
			configChanged = true
		case "log-level":
			config.LogLevel = *logLevel
			configChanged = true
		case "video-dir":
			config.DestinationFolder = *videoDir
			configChanged = true
		}
	})

	// Only update config if at least one setting was explicitly changed
	if configChanged {
		if err := goProManager.UpdateConfig(config); err != nil {
			mainLog.Warn("Failed to apply config settings from command line", "err", err)
		}
	}

	// Apply persisted log level (CLI flag takes priority if explicitly set)
	if level, err := logging.ParseLevel(config.LogLevel); err == nil {
		if level != levelVar.Level() {
			mainLog.Info("Log level set from persisted config", "level", config.LogLevel)
		}
		levelVar.Set(level)
	} else {
		mainLog.Warn("Invalid persisted log level, keeping flag level", "value", config.LogLevel, "err", err)
	}

	// Start GoPro manager
	if err := goProManager.Start(); err != nil {
		mainLog.Error("Failed to start GoPro manager", "err", err)
		os.Exit(1)
	}

	// Initialize and start gRPC server
	dropzServer := server.NewDropzServer(goProManager, log)
	if err := dropzServer.Start(*serverAddr); err != nil {
		mainLog.Error("Failed to start gRPC server", "err", err)
		os.Exit(1)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	receivedSignal := <-sigChan
	mainLog.Info("Shutting down on signal", "signal", receivedSignal.String())

	dropzServer.Stop()
	goProManager.Stop()
	mainLog.Info("dropz has been shut down")
}
