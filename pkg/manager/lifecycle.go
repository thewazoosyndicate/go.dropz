package manager

import (
	"time"

	"github.com/dropz/dropz/pkg/database"
)

// Start starts the GoPro manager service
func (m *GoProManager) Start() error {
	m.log.Info("Starting GoPro manager service")

	// Start BLE manager
	m.ble.Start()

	// Start task queue
	m.queue.Start()

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
		m.log.Info("Performing initial device pairing state synchronization")
		m.syncDevicePairingStates()
	}()

	return nil
}

// Stop stops the GoPro manager service
func (m *GoProManager) Stop() {
	m.log.Infof("GoProManager.Stop() called. isRunning: %v", m.isRunning)
	if !m.isRunning {
		m.log.Info("GoProManager.Stop() called but manager was not running.")
		return
	}

	m.log.Info("Stopping GoPro manager service...")

	m.log.Debug("Cancelling GoProManager context (for backgroundScanner, deviceManager)...")
	m.cancel() // Signal all internal goroutines to stop
	m.log.Debug("GoProManager context cancelled.")

	m.log.Debug("Attempting to stop task queue...")
	m.queue.Stop() // This should be idempotent and handle being called multiple times
	m.log.Debug("Task queue stop requested/completed.")

	m.log.Debug("Attempting to stop BLE manager...")
	m.ble.Stop() // This should also be idempotent
	m.log.Debug("BLE manager stop requested/completed.")

	m.log.Debug("Waiting for GoProManager internal goroutines (backgroundScanner, deviceManager) to finish...")
	m.wg.Wait() // Wait for backgroundScanner and deviceManager
	m.log.Debug("GoProManager internal goroutines finished.")

	m.isRunning = false
	m.log.Info("GoPro manager service stopped successfully.")
}

// deviceManager periodically checks managed devices
func (m *GoProManager) deviceManager() {
	defer m.wg.Done()
	defer m.log.Debug("Device manager goroutine finished.")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

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
		}
	}
}

// checkManagedDevices checks all managed devices and queues tasks
func (m *GoProManager) checkManagedDevices() {
	managedCameras := m.db.GetAllManagedCameras()
	config := m.db.GetConfig()

	// Log the current sync configuration for debugging
	m.log.Debugf("Current sync configuration: SyncEnabled=%v, DaysThreshold=%v",
		config.SyncEnabled, config.DaysThreshold)

	for _, camera := range managedCameras {
		// Never change status for cameras that are actively syncing or pairing
		if camera.CameraState.Status.IsSyncing || camera.CameraState.Status.IsPairing {
			m.log.Debugf("Camera %s is actively syncing or pairing, preserving status during check",
				camera.CameraState.Camera.Name)
			continue
		}

		// Check if the camera is reachable
		if !camera.CameraState.Status.IsReachable {
			m.log.Debugf("Camera %s is not reachable", camera.CameraState.Camera.Name)
			continue
		}

		// Skip sync check if the camera is already synced
		if camera.CameraState.Status.IsSynced {
			m.log.Debugf("Camera %s is already synced, skipping check", camera.CameraState.Camera.Name)
			continue
		}

		// Check if sync is needed based on timestamp
		if !config.SyncEnabled {
			m.log.Debugf("Sync is disabled, not checking camera %s", camera.CameraState.Camera.Name)
			continue
		}

		// Check if camera is paired before attempting sync
		// Query device directly for most accurate state
		macAddress := camera.CameraState.Camera.MACAddress
		devicePaired, err := m.ble.IsPaired(macAddress)
		if err != nil {
			// If we can't check device state, fall back to database state
			m.log.Debugf("Failed to check device pairing state for %s, using database state: %v",
				camera.CameraState.Camera.Name, err)
			devicePaired = camera.CameraState.Status.IsPaired
		}

		if !devicePaired {
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
