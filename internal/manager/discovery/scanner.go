package discovery

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/logging"
)

// Scanner handles background scanning operations
type Scanner struct {
	ble               *ble.Manager
	log               *slog.Logger
	scanInterval      time.Duration
	processDeviceFunc func(ble.Device) // callback for processing individual devices
}

// NewScanner creates a new scanner
func NewScanner(ble *ble.Manager, log *slog.Logger, scanInterval time.Duration) *Scanner {
	return &Scanner{
		ble:          ble,
		log:          log.With("component", "scanner"),
		scanInterval: scanInterval,
	}
}

// StartBackgroundScanner starts the background scanner with live device processing
func (s *Scanner) StartBackgroundScanner(ctx context.Context, processDeviceFunc func(ble.Device)) {
	s.log.Debug("Starting background scanner", "interval", s.scanInterval)
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
			s.log.Debug("Background scanner stopping due to context cancellation")
			if cancelContinuousScan != nil {
				cancelContinuousScan()
			}
			return

		case <-ticker.C:
			if !scanInProgress.Load() {
				s.log.Debug("Restarting scan after external stop")
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
				// The scan loop died without the ticker noticing: an anomaly,
				// unlike the routine post-connect restart above
				s.log.Warn("Scanner watchdog restarting scan")
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

	logging.Trace(s.log, "Starting continuous BLE scanning process")

	// First failure and recovery are logged once; retries back off silently
	// so a dead adapter cannot flood the rotating log.
	const maxRetryDelay = time.Minute
	retryDelay := 2 * time.Second
	scanFailing := false

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

		logging.Trace(s.log, "Starting BLE scan cycle")

		err := s.ble.StartScanningWithCallback(scanCtx, func(device ble.Device) {
			if s.processDeviceFunc != nil {
				s.processDeviceFunc(device)
			}
		})

		if err != nil {
			cancel() // Always cancel the context
			if !scanFailing {
				s.log.Warn("Failed to start BLE scan, retrying with backoff", "err", err)
				scanFailing = true
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(retryDelay):
			}
			if retryDelay *= 2; retryDelay > maxRetryDelay {
				retryDelay = maxRetryDelay
			}
			continue
		}
		if scanFailing {
			s.log.Info("BLE scan recovered")
			scanFailing = false
		}
		retryDelay = 2 * time.Second

		logging.Trace(s.log, "BLE scan started, waiting for completion")

		scanDone := s.ble.ScanDone()

		// Wait for scan to complete, external stop, or context cancellation
		select {
		case <-scanCtx.Done():
			if scanCtx.Err() != context.Canceled {
				logging.Trace(s.log, "Scan cycle completed (5 minute timeout)")
			}
		case <-ctx.Done():
			cancel()
			s.log.Debug("Parent context canceled during scan")
			return
		case <-scanDone:
			s.log.Debug("Scan stopped externally, waiting for connection to finish")
			// Wait for the BLE connection to complete before restarting scan
			if ch := s.ble.ConnectingDone(); ch != nil {
				select {
				case <-ch:
					s.log.Debug("Connection finished, will restart scan")
				case <-ctx.Done():
					cancel()
					return
				}
			}
		}

		cancel() // Always cancel the context

		// Minimal pause between scan cycles for continuous RSSI monitoring
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}
