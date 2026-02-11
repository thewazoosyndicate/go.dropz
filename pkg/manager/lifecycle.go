package manager

import (
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/manager/discovery"
	syncpkg "github.com/dropz/dropz/pkg/manager/sync"
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

// deviceManager periodically checks managed devices
func (m *GoProManager) deviceManager() {
	defer m.wg.Done()
	defer m.log.Debug("Device manager goroutine finished.")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Ticker for marking unreachable devices (needed since we moved to live processing)
	unreachableTicker := time.NewTicker(15 * time.Second)
	defer unreachableTicker.Stop()

	m.log.Info("Starting device manager")

	for {
		select {
		case <-m.ctx.Done():
			m.log.Debug("Device manager stopping due to context cancellation.")
			return
		case <-ticker.C:
			m.checkManagedDevices()
			m.syncCoordinator.ProcessSyncQueue()
		case <-m.immediateSyncTrigger:
			m.log.Debug("Immediate sync triggered - processing sync queue")
			m.syncCoordinator.ProcessSyncQueue()
		case <-unreachableTicker.C:
			config := m.db.GetConfig()
			timeout := time.Duration(config.InactivityTimeoutSeconds) * time.Second
			m.discoveryProcessor.MarkUnreachableDevicesBackground(timeout, m.notifier)
		}
	}
}

// checkManagedDevices checks all managed devices and queues tasks
func (m *GoProManager) checkManagedDevices() {
	// Use GetCamerasForSyncQueue which properly validates reachable + paired + managed + not synced
	syncCandidates := m.db.GetCamerasForSyncQueue()
	config := m.db.GetConfig()

	// Log the current sync configuration for debugging
	for _, camera := range syncCandidates {
		// Never change status for cameras that are actively syncing or pairing
		if camera.CameraState.Status.IsSyncing || camera.CameraState.Status.IsPairing {
			m.log.Tracef("Camera %s is actively syncing or pairing, preserving status during check",
				camera.CameraState.Camera.Name)
			continue
		}

		// Check if sync is needed based on timestamp
		if !config.SyncEnabled {
			m.log.Tracef("Sync is disabled, not checking camera %s", camera.CameraState.Camera.Name)
			continue
		}

		if !camera.CameraState.Status.IsPaired {
			m.log.Debugf("Camera %s is not paired, skipping sync", camera.CameraState.Camera.Name)
			continue
		}

		// Check if we should sync based on last sync time
		// If LastSynced is zero time or more than threshold days ago, queue for sync
		shouldSync := camera.CameraState.Status.LastSynced.IsZero() ||
			time.Since(camera.CameraState.Status.LastSynced) > time.Duration(config.DaysThreshold)*24*time.Hour

		if shouldSync {
			// Add to sync queue
			syncEntry := &database.SyncQueueEntry{
				CameraID:         camera.CameraState.Camera.ID,
				QueuedAt:         time.Now(),
				Priority:         syncpkg.SyncPriorityAuto,
				ProgressPercent:  0,
				CurrentOperation: "Waiting to start",
			}

			m.log.Infof("Adding camera %s to sync queue (threshold: %d days)",
				camera.CameraState.Camera.Name, config.DaysThreshold)

			if err := m.db.AddSyncQueueEntry(syncEntry); err != nil {
				m.log.Errorf("Failed to add camera %s to sync queue: %v",
					camera.CameraState.Camera.Name, err)
			}
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
		m.discoveryProcessor.ProcessDiscoveredDeviceLive(device, m.notifier)
	})
}
