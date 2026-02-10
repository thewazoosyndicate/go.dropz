package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/dropz/dropz/pkg/wifi"
)

// Sync priority constants
const (
	SyncPriorityManual = 10 // High priority for manually requested syncs
	SyncPriorityAuto   = 5  // Normal priority for automatic syncs
)

// SyncTask represents a camera sync task
type SyncTask struct {
	CameraID     string
	MACAddress   string
	CameraName   string
	StartedAt    time.Time
	CompletedAt  time.Time
	Success      bool
	ErrorMessage string
	Cancel       context.CancelFunc
	Ctx          context.Context
}

// Coordinator handles sync orchestration
type Coordinator struct {
	db  *database.Database
	ble *ble.Manager
	log *logrus.Logger
}

// NewCoordinator creates a new sync coordinator
func NewCoordinator(db *database.Database, ble *ble.Manager, log *logrus.Logger) *Coordinator {
	return &Coordinator{
		db:  db,
		ble: ble,
		log: log,
	}
}

// ProcessSyncQueue checks the sync queue and processes cameras that need syncing
func (c *Coordinator) ProcessSyncQueue(activeSyncTasks map[string]*SyncTask, mutex *sync.RWMutex, notifier func(), performSync func(*SyncTask), ctx context.Context) {
	if !c.db.GetConfig().SyncEnabled {
		c.log.Debug("Sync is disabled, skipping sync queue processing")
		return
	}

	c.log.Trace("Processing sync queue")
	syncQueue := c.db.GetSyncQueue()

	for _, entry := range syncQueue {
		// Check if we already have an active sync task for this camera
		mutex.RLock()
		_, exists := activeSyncTasks[entry.CameraID]
		mutex.RUnlock()

		if exists {
			// Skip this entry, it's already being synced
			c.log.Tracef("Camera %s is already being synced, skipping", entry.CameraID)
			continue
		}

		// Get the camera details
		camera, found := c.db.GetManagedCameraByID(entry.CameraID)
		if !found {
			c.log.Warnf("Camera %s not found, removing from sync queue", entry.CameraID)
			c.db.RemoveSyncQueueEntry(entry.CameraID)
			continue
		}

		// Check if the camera is suitable for syncing
		if !camera.CameraState.Status.IsReachable {
			c.log.Tracef("Camera %s is not reachable, skipping sync", entry.CameraID)
			continue
		}

		if camera.CameraState.Status.IsSyncing {
			c.log.Tracef("Camera %s is already marked as syncing, skipping", entry.CameraID)
			continue
		}

		// Set camera as syncing
		c.log.Infof("Starting sync for camera %s", camera.CameraState.Camera.Name)
		c.db.UpdateCameraSyncingStatus(camera.CameraState.Camera.MACAddress, true)

		// Create and start a sync task
		taskCtx, taskCancel := context.WithCancel(ctx)
		syncTask := &SyncTask{
			CameraID:   camera.CameraState.Camera.ID,
			MACAddress: camera.CameraState.Camera.MACAddress,
			CameraName: camera.CameraState.Camera.Name,
			StartedAt:  time.Now(),
			Cancel:     taskCancel,
			Ctx:        taskCtx,
		}

		mutex.Lock()
		activeSyncTasks[entry.CameraID] = syncTask
		mutex.Unlock()

		// Start sync in background
		go performSync(syncTask)
	}
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (c *Coordinator) ForceSync(cameraID string, notifier func()) (*database.SyncQueueEntry, error) {
	cameraState, found := c.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	// Check if camera is already in the sync queue
	for _, entry := range c.db.GetSyncQueue() {
		if entry.CameraID == cameraID {
			c.log.Infof("Camera %s is already in the sync queue", cameraState.Camera.Name)
			return entry, nil
		}
	}

	// Create a new sync queue entry
	syncEntry := &database.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		Priority:         SyncPriorityManual, // High priority for manual sync requests
		ProgressPercent:  0,
		CurrentOperation: "Waiting to start",
	}

	// Add to sync queue
	if err := c.db.AddSyncQueueEntry(syncEntry); err != nil {
		return nil, fmt.Errorf("failed to add camera to sync queue: %v", err)
	}

	// Mark the camera as not synced so it will appear in sync queue
	c.db.ResetSyncStatus(cameraState.Camera.MACAddress)

	c.log.Infof("Added camera %s to sync queue", cameraState.Camera.Name)

	// Notify immediately
	if notifier != nil {
		notifier()
	}

	return syncEntry, nil
}

// CancelSync cancels an ongoing sync operation and removes the camera from the sync queue
func (c *Coordinator) CancelSync(cameraID string, activeSyncTasks map[string]*SyncTask, mutex *sync.RWMutex, notifier func()) error {
	cameraState, found := c.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	c.log.Infof("Canceling sync for camera %s", cameraState.Camera.Name)

	// Check if there's an active sync task
	mutex.Lock()
	task, exists := activeSyncTasks[cameraID]
	if exists {
		// Cancel the sync task's context first
		if task.Cancel != nil {
			task.Cancel()
			c.log.Infof("Cancelled sync context for camera %s", cameraState.Camera.Name)
		}
		// Remove the active sync task
		delete(activeSyncTasks, cameraID)
		c.log.Infof("Removed active sync task for camera %s", cameraState.Camera.Name)
	}
	mutex.Unlock()

	// Remove from sync queue regardless
	c.db.RemoveSyncQueueEntry(cameraID)

	// Reset camera syncing status
	c.db.UpdateCameraSyncingStatus(cameraState.Camera.MACAddress, false)

	c.log.Infof("Cancelled sync for camera %s", cameraState.Camera.Name)

	// Notify immediately
	if notifier != nil {
		notifier()
	}

	return nil
}

// PerformCameraSync handles the actual syncing of a camera
func (c *Coordinator) PerformCameraSync(task *SyncTask, activeSyncTasks map[string]*SyncTask, mutex *sync.RWMutex, notifier func(), bleOperation func(context.Context, string, func() error) error, ctx context.Context) {
	// Remove task when done
	defer func() {
		mutex.Lock()
		delete(activeSyncTasks, task.CameraID)
		mutex.Unlock()
	}()

	c.log.Infof("Syncing camera %s (%s)", task.CameraName, task.MACAddress)

	// Get the camera details to work with
	camera, found := c.db.GetManagedCameraByID(task.CameraID)
	if !found {
		c.log.Errorf("Cannot sync camera %s: not found in database", task.CameraID)
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
	if err := c.db.UpdateCameraSyncingStatus(task.MACAddress, true); err != nil {
		c.log.Errorf("Failed to mark camera as syncing: %v", err)
	}

	// Make sure the syncing flag is reset when done, regardless of the outcome
	defer func() {
		if err := c.db.UpdateCameraSyncingStatus(task.MACAddress, false); err != nil {
			c.log.Errorf("Failed to reset syncing status: %v", err)
		}
	}()

	// Get the current config
	config := c.db.GetConfig()

	// Create a context for the sync operation - use the task's context if available, otherwise the provided context
	var syncCtx context.Context
	var cancel context.CancelFunc

	if task.Ctx != nil {
		// Create a timeout context that also respects the task's cancellation context
		syncCtx, cancel = context.WithTimeout(task.Ctx, 30*time.Minute)
	} else {
		// Fallback to the provided context
		syncCtx, cancel = context.WithTimeout(ctx, 30*time.Minute)
	}
	defer cancel()

	// Real implementation of camera sync process
	// Step 1: Connect to camera via BLE
	syncEntry.CurrentOperation = "Connecting via BLE"
	syncEntry.ProgressPercent = 10
	if err := c.updateSyncQueueEntrySafely(task, syncEntry); err != nil {
		c.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	// Notify about the status change
	if notifier != nil {
		notifier()
	}

	c.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Use BLEOperation to handle concurrency and semaphores correctly
	connErr := bleOperation(syncCtx, "ConnectForSync", func() error {
		return c.ble.Connect(camera.CameraState.Camera.MACAddress)
	})

	if connErr != nil {
		syncEntry.CurrentOperation = "BLE Connection Failed"
		// Just update the entry without setting a status enum
		if err := c.updateSyncQueueEntrySafely(task, syncEntry); err != nil {
			c.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		c.log.Errorf("Failed to connect to camera via BLE: %v", connErr)
		return
	}

	// Ensure BLE disconnection happens in all cases
	defer func() {
		disconnectErr := bleOperation(syncCtx, "DisconnectAfterSync", func() error {
			return c.ble.Disconnect(camera.CameraState.Camera.MACAddress)
		})
		if disconnectErr != nil {
			c.log.Warnf("Failed to disconnect from BLE after sync: %v", disconnectErr)
		}
	}()

	// Step 2: Connect to camera WiFi
	// WiFi AP is already enabled by ble.Connect()
	syncEntry.CurrentOperation = "Connecting to WiFi"
	syncEntry.ProgressPercent = 30
	if err := c.updateSyncQueueEntrySafely(task, syncEntry); err != nil {
		c.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if notifier != nil {
		notifier()
	}

	c.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Create WiFi manager instance
	wifiManager := wifi.NewWiFiManager(c.log)

	// Connect to WiFi using stored credentials
	wifiCtx, wifiCancel := context.WithTimeout(syncCtx, time.Duration(config.ConnectTimeoutSeconds)*time.Second)
	defer wifiCancel()

	connWifiErr := wifiManager.Connect(wifiCtx, camera.CameraState.Camera.WiFiSSID, camera.CameraState.Camera.WiFiPassword)
	if connWifiErr != nil {
		syncEntry.CurrentOperation = "WiFi Connection Failed"
		if err := c.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			c.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		c.log.Errorf("Failed to connect to camera WiFi: %v", connWifiErr)
		return
	}

	// Ensure we disconnect from WiFi when done
	defer func() {
		disconnectErr := wifiManager.Disconnect()
		if disconnectErr != nil {
			c.log.Warnf("Failed to disconnect from WiFi: %v", disconnectErr)
		}
	}()

	// Step 3: Verify GoPro HTTP API is reachable before downloading
	c.log.Info("Verifying GoPro HTTP API connectivity...")
	for attempt := 1; attempt <= 10; attempt++ {
		_, err := wifiManager.GetCameraStatus(syncCtx)
		if err == nil {
			c.log.Info("GoPro HTTP API is reachable")
			break
		}
		if attempt == 10 {
			syncEntry.CurrentOperation = "GoPro API unreachable"
			c.db.UpdateSyncQueueEntry(syncEntry)
			c.log.Errorf("GoPro HTTP API not reachable after %d attempts: %v", attempt, err)
			return
		}
		c.log.Debugf("GoPro API not ready (attempt %d/10): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}

	// Step 4: Download media
	syncEntry.CurrentOperation = "Downloading media"
	syncEntry.ProgressPercent = 60
	if err := c.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		c.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if notifier != nil {
		notifier()
	}

	c.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Ensure destination directory exists
	if err := os.MkdirAll(config.DestinationFolder, 0755); err != nil {
		syncEntry.CurrentOperation = "Failed to create destination folder"
		if err := c.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			c.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		c.log.Errorf("Failed to create destination folder: %v", err)
		return
	}

	// Create a folder for the camera if it doesn't exist
	cameraFolder := filepath.Join(config.DestinationFolder, camera.CameraState.Camera.Name)
	if err := os.MkdirAll(cameraFolder, 0755); err != nil {
		c.log.Errorf("Failed to create camera-specific folder: %v", err)
		syncEntry.CurrentOperation = "Failed to create camera folder"
		if err := c.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			c.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		return
	}

	// Download videos directly to the camera folder
	downloadCtx, downloadCancel := context.WithTimeout(syncCtx, 20*time.Minute)
	defer downloadCancel()

	downloadedFiles, err := wifiManager.DownloadVideos(downloadCtx, cameraFolder, int(config.DaysThreshold))
	if err != nil {
		syncEntry.CurrentOperation = "Media Download Failed"
		if err := c.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			c.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		c.log.Errorf("Failed to download media: %v", err)
		return
	}

	// Step 5: Process files (add metadata, etc.)
	syncEntry.CurrentOperation = "Processing files"
	syncEntry.ProgressPercent = 90
	if err := c.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		c.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if notifier != nil {
		notifier()
	}

	c.log.Infof("Sync for camera %s: %s (%d%%), downloaded %d files",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent, len(downloadedFiles))

	// Mark sync as complete - set appropriate boolean flags instead of status enum
	c.db.MarkCameraSynced(task.MACAddress)
	c.log.Infof("Camera %s synced successfully", task.CameraName)

	// Update completion status in sync entry
	syncEntry.ProgressPercent = 100
	syncEntry.CurrentOperation = "Completed"
	if err := c.updateSyncQueueEntrySafely(task, syncEntry); err != nil {
		c.log.Errorf("Failed to update sync queue entry on completion: %v", err)
	}

	// Remove from sync queue
	c.db.RemoveSyncQueueEntry(task.CameraID)

	// Update completion status
	task.CompletedAt = time.Now()
	task.Success = true

	// Notify about the status change
	if notifier != nil {
		notifier()
	}
}

// GetVideosByCamera returns videos for a specific camera
func (c *Coordinator) GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*database.VideoFile, int) {
	videos := c.db.GetVideosByCamera(cameraID)

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

// updateSyncQueueEntrySafely updates the sync queue entry only if the task hasn't been cancelled
func (c *Coordinator) updateSyncQueueEntrySafely(task *SyncTask, syncEntry *database.SyncQueueEntry) error {
	// Check if the task has been cancelled
	if task.Ctx != nil {
		select {
		case <-task.Ctx.Done():
			// Task has been cancelled, don't update the database
			c.log.Debugf("Skipping sync queue update for cancelled task: %s", task.CameraID)
			return nil
		default:
			// Task is still active, proceed with update
		}
	}

	return c.db.UpdateSyncQueueEntry(syncEntry)
}
