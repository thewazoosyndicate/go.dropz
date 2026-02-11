package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dropz/dropz/pkg/manager"
	"github.com/dropz/dropz/pkg/server"
	"github.com/sirupsen/logrus"
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
	logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	serverAddr  = flag.String("server-addr", "127.0.0.1:50051", "gRPC server address")
	showVersion = flag.Bool("version", false, "Show version and exit")
	pairMode    = flag.Bool("pair-mode", true, "Enable automatic pairing mode")
	syncEnabled = flag.Bool("sync-enabled", true, "Enable automatic content synchronization")
)

const (
	appVersion = "0.2.0"
)

type logFormatter struct{}

func (f *logFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var levelPrefix string
	switch entry.Level {
	case logrus.TraceLevel:
		levelPrefix = "[TRACE] "
	case logrus.DebugLevel:
		levelPrefix = "[DEBUG] "
	case logrus.InfoLevel:
		levelPrefix = "[INFO] "
	case logrus.WarnLevel:
		levelPrefix = "[WARN] "
	case logrus.ErrorLevel:
		levelPrefix = "[ERROR] "
	case logrus.FatalLevel:
		levelPrefix = "[FATAL] "
	case logrus.PanicLevel:
		levelPrefix = "[PANIC] "
	}

	timestamp := entry.Time.Format("2006/01/02 15:04:05")
	fields := ""
	for k, v := range entry.Data {
		fields += " " + k + "=" + fmt.Sprintf("%v", v)
	}
	return []byte(timestamp + " " + levelPrefix + entry.Message + fields + "\n"), nil
}

func initLogger(level, filePath string) (*logrus.Logger, error) {
	log := logrus.New()
	log.SetFormatter(&logFormatter{})

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	log.SetLevel(logLevel)

	if filePath != "" {
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		log.SetOutput(io.MultiWriter(os.Stderr, file))
	}

	return log, nil
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("dropz version %s\n", appVersion)
		os.Exit(0)
	}

	// Create directories if they don't exist
	for _, dir := range []string{*dataDir, *videoDir, *logDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("Failed to create directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Initialize logger
	logFile := filepath.Join(*logDir, "dropz.log")
	log, err := initLogger(*logLevel, logFile)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	log.Infof("Starting dropz version %s", appVersion)

	// Initialize GoPro manager
	dbPath := filepath.Join(*dataDir, "dropz.db")
	goProManager, err := manager.NewGoProManager(dbPath, *videoDir, log)
	if err != nil {
		log.Fatalf("Failed to initialize GoPro manager: %v", err)
	}

	// Reset transient camera states (is_syncing, is_pairing) on startup
	goProManager.ResetTransientStates()

	// Only override config settings if flags were explicitly set
	config := goProManager.GetConfig()

	// Track if any flag was explicitly set
	configChanged := false

	// Check which flags were explicitly set by user
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "pair-mode":
			config.PairModeEnabled = *pairMode
			configChanged = true
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
			log.Warnf("Failed to apply config settings from command line: %v", err)
		}
	}

	// Start GoPro manager
	if err := goProManager.Start(); err != nil {
		log.Fatalf("Failed to start GoPro manager: %v", err)
	}

	// Initialize and start gRPC server
	dropzServer := server.NewDropzServer(goProManager, log)
	if err := dropzServer.Start(*serverAddr); err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}

	log.Infof("dropz is running. Press Ctrl+C to exit.")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	receivedSignal := <-sigChan
	log.Infof("Received signal: %s. Shutting down...", receivedSignal)

	log.Info("Attempting to stop Dropz server...")
	dropzServer.Stop()
	log.Info("Dropz server stop requested.")

	log.Info("Attempting to stop GoPro manager...")
	goProManager.Stop()
	log.Info("GoPro manager stop requested.")

	log.Info("dropz has been shut down.")
}
