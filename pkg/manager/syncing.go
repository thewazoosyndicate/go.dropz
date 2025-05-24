package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/wifi"
)

// processSyncQueue checks the sync queue and processes cameras that need syncing
func (m *GoProManager) processSyncQueue() {
	if !m.db.GetConfig().SyncEnabled {
		m.log.Debug("Sync is disabled, skipping sync queue processing")
		return
	}

	m.log.Debug("Processing sync queue")
	syncQueue := m.db.GetSyncQueue()

	for _, entry := range syncQueue {
		// Check if we already have an active sync task for this camera
		m.mutex.RLock()
		_, exists := m.activeSyncTasks[entry.CameraID]
		m.mutex.RUnlock()

		if exists {
			// Skip this entry, it's already being synced
			m.log.Debugf("Camera %s is already being synced, skipping", entry.CameraID)
			continue
		}

		// Get the camera details
		camera, found := m.db.GetManagedCameraByID(entry.CameraID)
		if !found {
			m.log.Warnf("Camera %s not found, removing from sync queue", entry.CameraID)
			m.db.RemoveSyncQueueEntry(entry.CameraID)
			continue
		}

		// Check if the camera is suitable for syncing
		if !camera.CameraState.Status.IsReachable {
			m.log.Debugf("Camera %s is not reachable, skipping sync", entry.CameraID)
			continue
		}

		if camera.CameraState.Status.IsSyncing {
			m.log.Debugf("Camera %s is already marked as syncing, skipping", entry.CameraID)
			continue
		}

		// Set camera as syncing
		m.log.Infof("Starting sync for camera %s", camera.CameraState.Camera.Name)
		m.db.UpdateCameraSyncingStatus(camera.CameraState.Camera.MACAddress, true)

		// Create and start a sync task
		syncTask := &SyncTask{
			CameraID:   camera.CameraState.Camera.ID,
			MACAddress: camera.CameraState.Camera.MACAddress,
			CameraName: camera.CameraState.Camera.Name,
			StartedAt:  time.Now(),
		}

		m.mutex.Lock()
		m.activeSyncTasks[entry.CameraID] = syncTask
		m.mutex.Unlock()

		// Start sync in background
		go m.performCameraSync(syncTask)
	}
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (m *GoProManager) ForceSync(cameraID string) (*database.SyncQueueEntry, error) {
	// Find the camera
	var cameraState *database.CameraWithState

	// Check all cameras
	for _, state := range m.db.CameraStates {
		if state.Camera.ID == cameraID {
			cameraState = state
			break
		}
	}

	if cameraState == nil {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	// Check if camera is already in the sync queue
	for _, entry := range m.db.GetSyncQueue() {
		if entry.CameraID == cameraID {
			m.log.Infof("Camera %s is already in the sync queue", cameraState.Camera.Name)
			return entry, nil
		}
	}

	// Create a new sync queue entry
	syncEntry := &database.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		ProgressPercent:  0,
		CurrentOperation: "Waiting to start",
	}

	// Add to sync queue
	if err := m.db.AddSyncQueueEntry(syncEntry); err != nil {
		return nil, fmt.Errorf("failed to add camera to sync queue: %v", err)
	}

	// Mark the camera as not synced so it will appear in sync queue
	m.db.ResetSyncStatus(cameraState.Camera.MACAddress)

	m.log.Infof("Added camera %s to sync queue", cameraState.Camera.Name)

	// Notify immediately
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	return syncEntry, nil
}

// performCameraSync handles the actual syncing of a camera
func (m *GoProManager) performCameraSync(task *SyncTask) {
	// Remove task when done
	defer func() {
		m.mutex.Lock()
		delete(m.activeSyncTasks, task.CameraID)
		m.mutex.Unlock()
	}()

	m.log.Infof("Syncing camera %s (%s)", task.CameraName, task.MACAddress)

	// Get the camera details to work with
	camera, found := m.db.GetManagedCameraByID(task.CameraID)
	if !found {
		m.log.Errorf("Cannot sync camera %s: not found in database", task.CameraID)
		return
	}

	// Create sync queue entry if it doesn't exist
	syncEntry := &database.SyncQueueEntry{
		CameraID:         task.CameraID,
		QueuedAt:         time.Now(),
		ProgressPercent:  0,
		CurrentOperation: "Starting sync",
	}

	// Set camera as syncing in database
	if err := m.db.UpdateCameraSyncingStatus(task.MACAddress, true); err != nil {
		m.log.Errorf("Failed to mark camera as syncing: %v", err)
	}

	// Make sure the syncing flag is reset when done, regardless of the outcome
	defer func() {
		if err := m.db.UpdateCameraSyncingStatus(task.MACAddress, false); err != nil {
			m.log.Errorf("Failed to reset syncing status: %v", err)
		}
	}()

	// Get the current config
	config := m.db.GetConfig()

	// Create a context for the sync operation
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Minute)
	defer cancel()

	// Real implementation of camera sync process
	// Step 1: Connect to camera via BLE
	syncEntry.CurrentOperation = "Connecting via BLE"
	syncEntry.ProgressPercent = 10
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	// Notify about the status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Use BLEOperation to handle concurrency and semaphores correctly
	connErr := m.BLEOperation(ctx, "ConnectForSync", func() error {
		return m.ble.Connect(camera.CameraState.Camera.MACAddress)
	})

	if connErr != nil {
		syncEntry.CurrentOperation = "BLE Connection Failed"
		// Just update the entry without setting a status enum
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to connect to camera via BLE: %v", connErr)
		return
	}

	// Ensure BLE disconnection happens in all cases
	defer func() {
		disconnectErr := m.BLEOperation(ctx, "DisconnectAfterSync", func() error {
			return m.ble.Disconnect(camera.CameraState.Camera.MACAddress)
		})
		if disconnectErr != nil {
			m.log.Warnf("Failed to disconnect from BLE after sync: %v", disconnectErr)
		}
	}()

	// Step 2: Enable WiFi on the camera
	syncEntry.CurrentOperation = "Enabling WiFi"
	syncEntry.ProgressPercent = 30
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	wifiErr := m.BLEOperation(ctx, "EnableWifi", func() error {
		return m.ble.EnableWifi(camera.CameraState.Camera.MACAddress)
	})

	if wifiErr != nil {
		syncEntry.CurrentOperation = "WiFi Enabling Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to enable WiFi: %v", wifiErr)
		return
	}

	// Give the camera a moment to fully enable WiFi
	select {
	case <-ctx.Done():
		m.log.Warnf("Sync operation canceled: %v", ctx.Err())
		return
	case <-time.After(2 * time.Second):
		// Continue to next step
	}

	// Step 3: Connect to WiFi
	syncEntry.CurrentOperation = "Connecting to WiFi"
	syncEntry.ProgressPercent = 50
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Create WiFi manager instance
	wifiManager, err := wifi.NewWiFiManager()
	if err != nil {
		syncEntry.CurrentOperation = "WiFi Manager Creation Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to create WiFi manager: %v", err)
		return
	}

	// Connect to WiFi using stored credentials
	wifiCtx, wifiCancel := context.WithTimeout(ctx, time.Duration(config.ConnectTimeoutSeconds)*time.Second)
	defer wifiCancel()

	err = wifiManager.Connect(wifiCtx, camera.CameraState.Camera.WiFiSSID, camera.CameraState.Camera.WiFiPassword)
	if err != nil {
		syncEntry.CurrentOperation = "WiFi Connection Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to connect to camera WiFi: %v", err)
		return
	}

	// Ensure we disconnect from WiFi when done
	defer func() {
		disconnectErr := wifiManager.Disconnect()
		if disconnectErr != nil {
			m.log.Warnf("Failed to disconnect from WiFi: %v", disconnectErr)
		}
	}()

	// Step 4: Download media
	syncEntry.CurrentOperation = "Downloading media"
	syncEntry.ProgressPercent = 70
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Ensure destination directory exists
	if err := os.MkdirAll(config.DestinationFolder, 0755); err != nil {
		syncEntry.CurrentOperation = "Failed to create destination folder"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to create destination folder: %v", err)
		return
	}

	// Download videos from the last X days
	downloadCtx, downloadCancel := context.WithTimeout(ctx, 20*time.Minute)
	defer downloadCancel()

	downloadedFiles, err := wifiManager.DownloadVideos(downloadCtx, config.DestinationFolder, int(config.DaysThreshold))
	if err != nil {
		syncEntry.CurrentOperation = "Media Download Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to download media: %v", err)
		return
	}

	// Step 5: Process files (move to final location, add metadata, etc.)
	syncEntry.CurrentOperation = "Processing files"
	syncEntry.ProgressPercent = 90
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%), downloaded %d files",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent, len(downloadedFiles))

	// Create a folder for the camera if it doesn't exist
	cameraFolder := filepath.Join(config.DestinationFolder, camera.CameraState.Camera.Name)
	if err := os.MkdirAll(cameraFolder, 0755); err != nil {
		m.log.Warnf("Failed to create camera-specific folder: %v", err)
		// Continue anyway, use the main destination folder
	} else {
		// Move files to camera-specific folder if needed
		for _, file := range downloadedFiles {
			if filepath.Dir(file) != cameraFolder {
				newPath := filepath.Join(cameraFolder, filepath.Base(file))
				if err := os.Rename(file, newPath); err != nil {
					m.log.Warnf("Failed to move file %s to camera folder: %v", file, err)
				} else {
					m.log.Debugf("Moved file to camera folder: %s -> %s", file, newPath)
				}
			}
		}
	}

	// Mark sync as complete - set appropriate boolean flags instead of status enum
	m.db.MarkCameraSynced(task.MACAddress)
	m.log.Infof("Camera %s synced successfully", task.CameraName)

	// Update completion status in sync entry
	syncEntry.ProgressPercent = 100
	syncEntry.CurrentOperation = "Completed"
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry on completion: %v", err)
	}

	// Remove from sync queue
	m.db.RemoveSyncQueueEntry(task.CameraID)

	// Update completion status
	task.CompletedAt = time.Now()
	task.Success = true

	// Notify about the status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}
}

// GetVideosByCamera returns videos for a specific camera
func (m *GoProManager) GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*database.VideoFile, int) {
	videos := m.db.GetVideosByCamera(cameraID)

	// Filter by date range
	var filtered []*database.VideoFile
	for _, video := range videos {
		if !startDate.IsZero() && video.CreatedAt.Before(startDate) {
			continue
		}
		if !endDate.IsZero() && video.CreatedAt.After(endDate) {
			continue
		}
		filtered = append(filtered, video)
	}

	// Apply pagination
	totalCount := len(filtered)
	if offset >= totalCount {
		return []*database.VideoFile{}, totalCount
	}

	end := offset + limit
	if end > totalCount {
		end = totalCount
	}

	return filtered[offset:end], totalCount
}
