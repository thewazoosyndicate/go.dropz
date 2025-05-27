package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/dropz/dropz/cmd/dropz/cli"
	"github.com/dropz/dropz/pkg/logger"
	"github.com/dropz/dropz/pkg/manager"
	"github.com/dropz/dropz/pkg/server"
)

var (
	dataDir     = flag.String("data-dir", "data", "Directory for storing data")
	videoDir    = flag.String("video-dir", "videos", "Directory for storing downloaded videos")
	logDir      = flag.String("log-dir", "logs", "Directory for storing logs")
	logLevel    = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	serverAddr  = flag.String("server-addr", "127.0.0.1:50051", "gRPC server address")
	showVersion = flag.Bool("version", false, "Show version and exit")
	pairMode    = flag.Bool("pair-mode", true, "Enable automatic pairing mode")
	syncEnabled = flag.Bool("sync-enabled", true, "Enable automatic content synchronization")
	// Commands
	scan  = flag.Bool("scan", false, "Run scanner once and exit")
	mac   = flag.String("mac", "", "GoPro Mac Address - required for commands")
	pair  = flag.Bool("pair", false, "Run and exit")
	sync  = flag.Bool("sync", false, "Run and exit")
	sleep = flag.Bool("exit", false, "Run and exit")
)

// CLI
//
// Commands:
// - scan
// - pair (args. macAddress)
// - connect (args. macAddress)
//   . enable-wifi (also returns SSID/Passwd)
//   . get-hw-info
//   . get-battery-level
// - sync (args. macAddress)
// - sleep (args. macAddress)

const (
	appVersion = "0.2.0"
)

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
	if err := logger.Initialize(*logLevel, logFile); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log := logger.GetLogger()
	log.Infof("Starting dropz version %s", appVersion)

	// Initialize GoPro manager
	dbPath := filepath.Join(*dataDir, "dropz.db")
	goProManager, err := manager.NewGoProManager(dbPath, *videoDir)
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

	// Check for CLI flags that require immediate action
	cli.Cli()

	// If no CLI args, start GoPro manager
	if err := goProManager.Start(); err != nil {
		log.Fatalf("Failed to start GoPro manager: %v", err)
	}

	// Initialize and start gRPC server
	dropzServer := server.NewDropzServer(goProManager)
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
