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
	ble               ble.BLEInterface
	log               logger.Logger
	scanInterval      time.Duration
	processDeviceFunc func(ble.Device) // callback for processing individual devices
}

// NewScanner creates a new scanner
func NewScanner(ble ble.BLEInterface, log logger.Logger, scanInterval time.Duration) *Scanner {
	return &Scanner{
		ble:          ble,
		log:          log,
		scanInterval: scanInterval,
	}
}

// StartBackgroundScanner starts the background scanner with live device processing
func (s *Scanner) StartBackgroundScanner(ctx context.Context, processDeviceFunc func(ble.Device)) {
	s.log.Info("Starting background scanner with live device processing")
	s.processDeviceFunc = processDeviceFunc

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
	go s.startContinuousScan(continuousScanCtx, &scanInProgress)

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

				go s.startContinuousScan(continuousScanCtx, &scanInProgress)
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

				go s.startContinuousScan(continuousScanCtx, &scanInProgress)
			}
		}
	}
}

// startContinuousScan initiates a continuous BLE scan process with live device processing
func (s *Scanner) startContinuousScan(ctx context.Context, scanInProgress *atomic.Bool) {
	// Mark scan as in progress
	scanInProgress.Store(true)
	defer scanInProgress.Store(false)

	s.log.Debug("Starting continuous BLE scanning process with live device callbacks")

	// Loop until context is canceled or other conditions stop the scan
	for {
		select {
		case <-ctx.Done():
			s.log.Debug("Continuous scan stopping due to context cancellation")
			return
		default:
			// Continue with the scan
		}

		// For continuous RSSI monitoring, use much longer scan cycles (5 minutes)
		// This ensures we keep getting RSSI updates without frequent restarts
		scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		s.log.Trace("Starting BLE scan cycle with live device processing...")

		// Use the callback-based scanning for live updates
		err := s.ble.StartScanningWithCallback(scanCtx, func(device ble.Device) {
			// Process device immediately when discovered for real-time RSSI updates
			s.log.Debugf("Live discovery: %s (%s) RSSI:%d", device.Name, device.MACAddress, device.RSSI)
			if s.processDeviceFunc != nil {
				s.processDeviceFunc(device)
			}
		})

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

		s.log.Debug("BLE scan started with live processing, waiting for scan to complete...")

		// Wait for scan to complete or context cancellation
		select {
		case <-scanCtx.Done():
			if scanCtx.Err() != context.Canceled {
				s.log.Debug("Scan cycle completed (5 minute timeout)")
			}
		case <-ctx.Done():
			cancel()
			s.log.Debug("Parent context canceled during scan")
			return
		}

		cancel() // Always cancel the context

		// Minimal pause between scan cycles for continuous RSSI monitoring
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
			// Very short pause to ensure continuous scanning
		}
	}
}
