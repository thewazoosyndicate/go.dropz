package manager

import (
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/manager/discovery"
)

const (
	syncQueueInterval   = 30 * time.Second
	unreachableInterval = 15 * time.Second
)

// Start starts the GoPro manager service
func (m *GoProManager) Start() error {
	m.log.Info("GoPro manager starting")

	m.ResetTransientStates()

	m.wg.Add(3)
	go m.startBackgroundScanner()
	go m.deviceManager()
	go m.statusCheckWorker()

	return nil
}

// Stop stops the GoPro manager service
func (m *GoProManager) Stop() {
	m.log.Debug("GoPro manager stopping")

	// Stop BLE first so active operations finish cleanly before context cancel
	_ = m.ble.Stop()
	m.cancel()
	m.wg.Wait()

	m.log.Info("GoPro manager stopped")
}

// deviceManager periodically processes the sync queue and schedules status checks.
// BLE-heavy work runs in statusCheckWorker so a slow camera never stalls this loop.
func (m *GoProManager) deviceManager() {
	defer m.wg.Done()
	defer m.log.Debug("Device manager stopped")

	syncTicker := time.NewTicker(syncQueueInterval)
	defer syncTicker.Stop()

	config := m.db.GetConfig()
	statusInterval := time.Duration(config.StatusCheckIntervalSeconds) * time.Second
	if statusInterval == 0 {
		statusInterval = time.Hour // placeholder; tick is gated below
	}
	statusCheckTicker := time.NewTicker(statusInterval)
	defer statusCheckTicker.Stop()

	unreachableTicker := time.NewTicker(unreachableInterval)
	defer unreachableTicker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-syncTicker.C:
			m.syncCoordinator.ProcessSyncQueue()
			// Checked here (most frequent ticker) so config changes take effect within 30s
			newSeconds := m.db.GetConfig().StatusCheckIntervalSeconds
			newInterval := time.Duration(newSeconds) * time.Second
			if newInterval == 0 {
				newInterval = time.Hour
			}
			if newInterval != statusInterval {
				statusInterval = newInterval
				statusCheckTicker.Reset(statusInterval)
				// The setting change itself is logged where it happens (config.go)
				m.log.Debug("Status check ticker reset", "interval", statusInterval)
			}
		case <-statusCheckTicker.C:
			if m.db.GetConfig().StatusCheckIntervalSeconds > 0 {
				m.enqueueStatusChecks()
			}
		case <-m.immediateSyncTrigger:
			m.log.Debug("Immediate sync triggered - processing sync queue")
			m.syncCoordinator.ProcessSyncQueue()
		case <-unreachableTicker.C:
			config := m.db.GetConfig()
			timeout := time.Duration(config.InactivityTimeoutSeconds) * time.Second
			m.discoveryProcessor.MarkUnreachableDevicesBackground(timeout, m.notify)
		}
	}
}

// statusCheckWorker serializes BLE status checks off the deviceManager loop.
func (m *GoProManager) statusCheckWorker() {
	defer m.wg.Done()
	defer m.log.Debug("Status check worker stopped")

	for {
		select {
		case <-m.ctx.Done():
			return
		case cameraID := <-m.statusCheckRequest:
			m.checkSingleCameraStatusByID(cameraID)
		}
	}
}

// startBackgroundScanner starts the background scanner
func (m *GoProManager) startBackgroundScanner() {
	defer m.wg.Done()
	if !m.ble.Available() {
		// Already warned once at construction; this is just the consequence
		m.log.Debug("Background scanner disabled, bluetooth unavailable")
		return
	}
	config := m.db.GetConfig()
	scanInterval := time.Duration(config.ScanIntervalSeconds) * time.Second
	scanner := discovery.NewScanner(m.ble, m.log, scanInterval)
	scanner.StartBackgroundScanner(m.ctx, func(device ble.Device) {
		m.discoveryProcessor.ProcessDiscoveredDeviceLive(device, m.notify)
	})
}
