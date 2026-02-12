package manager

import (
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/manager/discovery"
)

// Start starts the GoPro manager service
func (m *GoProManager) Start() error {
	m.log.Info("GoPro manager starting: mode=service version=1.0")

	m.ResetTransientStates()

	m.wg.Add(2)
	go m.startBackgroundScanner()
	go m.deviceManager()

	return nil
}

// Stop stops the GoPro manager service
func (m *GoProManager) Stop() {
	m.log.Info("GoPro manager shutdown: phase=initiated")

	// Stop BLE first so active operations finish cleanly before context cancel
	m.log.Debug("GoPro manager shutdown: phase=stopping_ble")
	m.ble.Stop()

	m.log.Debug("GoPro manager shutdown: phase=cancelling_context")
	m.cancel()

	m.log.Debug("GoPro manager shutdown: phase=waiting_for_goroutines")
	m.wg.Wait()

	m.log.Info("GoPro manager shutdown: phase=completed status=success")
}

// deviceManager periodically processes sync queue and checks camera statuses via BLE.
func (m *GoProManager) deviceManager() {
	defer m.wg.Done()
	defer m.log.Debug("Device manager goroutine finished.")

	syncTicker := time.NewTicker(30 * time.Second)
	defer syncTicker.Stop()

	config := m.db.GetConfig()
	statusInterval := time.Duration(config.StatusCheckIntervalSeconds) * time.Second
	if statusInterval == 0 {
		statusInterval = time.Hour // placeholder; tick is gated below
	}
	statusCheckTicker := time.NewTicker(statusInterval)
	defer statusCheckTicker.Stop()

	unreachableTicker := time.NewTicker(15 * time.Second)
	defer unreachableTicker.Stop()

	m.log.Info("Starting device manager")

	for {
		select {
		case <-m.ctx.Done():
			m.log.Debug("Device manager stopping due to context cancellation.")
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
				m.log.Infof("Status check interval updated to %v", statusInterval)
			}
		case <-statusCheckTicker.C:
			if m.db.GetConfig().StatusCheckIntervalSeconds > 0 {
				m.checkCameraStatuses()
			}
		case <-m.immediateSyncTrigger:
			m.log.Debug("Immediate sync triggered - processing sync queue")
			m.syncCoordinator.ProcessSyncQueue()
		case cameraID := <-m.statusCheckRequest:
			m.checkSingleCameraStatusByID(cameraID)
		case <-unreachableTicker.C:
			config := m.db.GetConfig()
			timeout := time.Duration(config.InactivityTimeoutSeconds) * time.Second
			m.discoveryProcessor.MarkUnreachableDevicesBackground(timeout, m.notify)
		}
	}
}

// startBackgroundScanner starts the background scanner
func (m *GoProManager) startBackgroundScanner() {
	defer m.wg.Done()
	config := m.db.GetConfig()
	scanInterval := time.Duration(config.ScanIntervalSeconds) * time.Second
	scanner := discovery.NewScanner(m.ble, m.log, scanInterval)
	scanner.StartBackgroundScanner(m.ctx, func(device ble.Device) {
		m.discoveryProcessor.ProcessDiscoveredDeviceLive(device, m.notify)
	})
}
