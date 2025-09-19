package manager

import (
	"time"

	"github.com/dropz/dropz/pkg/database"
)

// Start starts the GoPro manager service
func (m *GoProManager) Start() error {
	m.log.Info("GoPro manager starting: mode=service version=1.0")

	// Start BLE manager
	m.ble.Start()

	// Start background scanner
	m.wg.Add(1)
	go m.startBackgroundScanner()

	// Start device manager
	m.wg.Add(1)
	go m.deviceManager()

	m.isRunning = true

	// ResetTransientStates resets transient camera states (is_syncing, is_pairing) on startup
	m.ResetTransientStates()

	// Perform initial sync of device pairing states with database
	go func() {
		// Wait a bit for BLE to be fully ready
		time.Sleep(5 * time.Second)
		m.log.Info("Device state synchronization: action=pairing_state_init phase=startup")
		m.syncDevicePairingStates()
	}()

	return nil
}

// Stop stops the GoPro manager service
func (m *GoProManager) Stop() {
	m.log.Infof("GoPro manager stopping: is_running=%t", m.isRunning)
	if !m.isRunning {
		m.log.Info("GoPro manager stop ignored: reason=not_running")
		return
	}

	m.log.Info("GoPro manager shutdown: phase=initiated")

	m.log.Debug("GoPro manager shutdown: phase=cancelling_context")
	m.cancel() // Signal all internal goroutines to stop
	m.log.Trace("GoPro manager shutdown: phase=context_cancelled")

	m.log.Debug("GoPro manager shutdown: phase=stopping_ble")
	m.ble.Stop() // This should also be idempotent
	m.log.Debug("GoPro manager shutdown: phase=ble_stopped")

	m.log.Debug("GoPro manager shutdown: phase=waiting_for_goroutines")
	m.wg.Wait() // Wait for backgroundScanner and deviceManager
	m.log.Debug("GoPro manager shutdown: phase=goroutines_completed")

	m.isRunning = false
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
			// Sync device pairing states with database
			m.syncDevicePairingStates()
			m.checkManagedDevices()
			// Process sync queue to start sync tasks for pending cameras
			m.processSyncQueue()
		case <-m.immediateSyncTrigger:
			// Immediate sync requested - process sync queue immediately
			m.log.Debug("Immediate sync triggered - processing sync queue")
			m.processSyncQueue()
		case <-unreachableTicker.C:
			// Mark unreachable devices (since we're no longer doing batch processing)
			m.markUnreachableDevicesBackground()
		}
	}
}

// checkManagedDevices checks all managed devices and queues tasks
func (m *GoProManager) checkManagedDevices() {
	// Use GetCamerasForSyncQueue which properly validates reachable + paired + managed + not synced
	syncCandidates := m.db.GetCamerasForSyncQueue()
	config := m.db.GetConfig()

	// Log the current sync configuration for debugging
	m.log.Debugf("Current sync configuration: SyncEnabled=%v, DaysThreshold=%v",
		config.SyncEnabled, config.DaysThreshold)

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

		// Double-check device pairing state for extra safety
		macAddress := camera.CameraState.Camera.MACAddress
		devicePaired, err := m.ble.IsPaired(macAddress)
		if err != nil {
			// If we can't check device state, fall back to database state
			m.log.Tracef("Failed to check device pairing state for %s, using database state: %v",
				camera.CameraState.Camera.Name, err)
			devicePaired = camera.CameraState.Status.IsPaired
		}

		if !devicePaired {
			m.log.Debugf("Camera %s is not paired on device, skipping sync", camera.CameraState.Camera.Name)
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
				Priority:         SyncPriorityAuto, // Normal priority for automatic syncs
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

// markUnreachableDevicesBackground marks cameras as unreachable in the background
func (m *GoProManager) markUnreachableDevicesBackground() {
	m.mutex.RLock()
	notifier := m.notifier
	discoveryProcessor := m.discoveryProcessor
	m.mutex.RUnlock()

	if discoveryProcessor != nil {
		m.log.Trace("Running background cleanup for unreachable devices")
		discoveryProcessor.MarkUnreachableDevicesBackground(notifier)
	} else {
		m.log.Warn("Discovery processor not available for background cleanup")
	}
}
