package discovery

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/logger"
)

// Scanner handles background scanning operations
type Scanner struct {
	ble          ble.BLEInterface
	log          logger.Logger
	scanInterval time.Duration
}

// NewScanner creates a new scanner
func NewScanner(ble ble.BLEInterface, log logger.Logger, scanInterval time.Duration) *Scanner {
	return &Scanner{
		ble:          ble,
		log:          log,
		scanInterval: scanInterval,
	}
}

// StartBackgroundScanner starts the background scanner
func (s *Scanner) StartBackgroundScanner(ctx context.Context, processDevices func([]ble.Device)) {
	s.log.Info("Starting background scanner")

	// Create a ticker for regular scan intervals
	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	// Create a watchdog ticker to ensure scanning is active
	watchdogTicker := time.NewTicker(1 * time.Minute)
	defer watchdogTicker.Stop()

	// Track scan state
	var scanInProgress atomic.Bool

	// Create initial scan context
	continuousScanCtx, cancelContinuousScan := context.WithCancel(ctx)

	// Use a closure to ensure the current cancelContinuousScan is always called
	defer func() {
		if cancelContinuousScan != nil {
			cancelContinuousScan()
		}
	}()

	// Start initial scan
	go s.startContinuousScan(continuousScanCtx, &scanInProgress, processDevices)

	for {
		select {
		case <-ctx.Done():
			s.log.Debug("Background scanner stopping due to context cancellation.")
			if cancelContinuousScan != nil {
				cancelContinuousScan()
			}
			return

		case <-ticker.C:
			// If we're not actively scanning, restart the continuous scanning process
			if !scanInProgress.Load() {
				s.log.Info("Regular scan interval triggered, restarting continuous scan")
				// Cancel any existing scan and create a new context
				if cancelContinuousScan != nil {
					cancelContinuousScan()
				}
				continuousScanCtx, cancelContinuousScan = context.WithCancel(ctx)

				go s.startContinuousScan(continuousScanCtx, &scanInProgress, processDevices)
			}

		case <-watchdogTicker.C:
			// Check if scanning is active, if not restart it
			if !scanInProgress.Load() {
				s.log.Info("Watchdog detected scan not running, restarting continuous scan")
				// Cancel any existing scan and create a new context
				if cancelContinuousScan != nil {
					cancelContinuousScan()
				}
				continuousScanCtx, cancelContinuousScan = context.WithCancel(ctx)

				go s.startContinuousScan(continuousScanCtx, &scanInProgress, processDevices)
			}
		}
	}
}

// startContinuousScan initiates a continuous BLE scan process
func (s *Scanner) startContinuousScan(ctx context.Context, scanInProgress *atomic.Bool, processDevices func([]ble.Device)) {
	// Mark scan as in progress
	scanInProgress.Store(true)
	defer scanInProgress.Store(false)

	s.log.Debug("Starting continuous BLE scanning process")

	// Loop until context is canceled or other conditions stop the scan
	for {
		select {
		case <-ctx.Done():
			s.log.Debug("Continuous scan stopping due to context cancellation")
			return
		default:
			// Continue with the scan
		}

		// Create a scan context with timeout for this single scan cycle
		scanCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

		s.log.Trace("Starting BLE scan cycle...")
		err := s.ble.StartScanning(scanCtx)
		if err != nil {
			cancel() // Always cancel the context
			s.log.Errorf("Failed to start BLE scan: %v", err)

			// Only short pause before retrying
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				// Continue with retry
			}
			continue
		}

		s.log.Debug("BLE scan started, waiting for scan to complete...")

		// Process devices immediately after scan completes or times out
		select {
		case <-scanCtx.Done():
			if scanCtx.Err() != context.Canceled {
				s.log.Debug("Scan cycle completed, processing discovered devices")
				// Process discovered devices
				devices := s.ble.GetDiscoveredDevices()
				processDevices(devices)
			}
		case <-ctx.Done():
			cancel()
			s.log.Debug("Parent context canceled during scan")
			return
		}

		cancel() // Always cancel the context

		// If we're in continuous mode, add a small pause between scans
		// to allow other BLE operations to occur
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			// Short pause between scan cycles
		}
	}
}
