package manager

import (
	"context"
	"sync/atomic"
	"time"
)

// startBackgroundScanner starts the background scanner
func (m *GoProManager) startBackgroundScanner() {
	m.log.Info("Starting background scanner")

	// Create a ticker for regular scan intervals
	ticker := time.NewTicker(m.scanInterval)
	defer ticker.Stop()

	// Create a watchdog ticker to ensure scanning is active
	watchdogTicker := time.NewTicker(1 * time.Minute)
	defer watchdogTicker.Stop()

	// Track scan state
	var scanInProgress atomic.Bool

	// Create initial scan context
	continuousScanCtx, cancelContinuousScan := context.WithCancel(m.ctx)

	// Use a closure to ensure the current cancelContinuousScan is always called
	defer func() {
		if cancelContinuousScan != nil {
			cancelContinuousScan()
		}
	}()

	// Start initial scan
	go m.startContinuousScan(continuousScanCtx, &scanInProgress)

	for {
		select {
		case <-m.ctx.Done():
			m.log.Debug("Background scanner stopping due to context cancellation.")
			if cancelContinuousScan != nil {
				cancelContinuousScan()
			}
			return

		case <-ticker.C:
			// If we're not actively scanning, restart the continuous scanning process
			if !scanInProgress.Load() {
				m.log.Info("Regular scan interval triggered, restarting continuous scan")
				// Cancel any existing scan and create a new context
				if cancelContinuousScan != nil {
					cancelContinuousScan()
				}
				continuousScanCtx, cancelContinuousScan = context.WithCancel(m.ctx)

				go m.startContinuousScan(continuousScanCtx, &scanInProgress)
			}

		case <-watchdogTicker.C:
			// Check if scanning is active, if not restart it
			if !scanInProgress.Load() {
				m.log.Info("Watchdog detected scan not running, restarting continuous scan")
				// Cancel any existing scan and create a new context
				if cancelContinuousScan != nil {
					cancelContinuousScan()
				}
				continuousScanCtx, cancelContinuousScan = context.WithCancel(m.ctx)

				go m.startContinuousScan(continuousScanCtx, &scanInProgress)
			}
		}
	}
}
