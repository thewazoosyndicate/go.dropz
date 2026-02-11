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
	CameraID   string
	DBKey      string
	BLEAddress string
	CameraName string
	Cancel     context.CancelFunc
	Ctx        context.Context
}

// Coordinator handles sync orchestration
type Coordinator struct {
	db             *database.Database
	ble            *ble.Manager
	log            *logrus.Logger
	activeTasks    map[string]*SyncTask
	mutex          sync.RWMutex
	notifier       func()
	bleOperation   func(context.Context, string, func() error) error
	ctx            context.Context
}

// NewCoordinator creates a new sync coordinator
func NewCoordinator(db *database.Database, ble *ble.Manager, log *logrus.Logger, activeTasks map[string]*SyncTask, notifier func(), bleOperation func(context.Context, string, func() error) error, ctx context.Context) *Coordinator {
	return &Coordinator{
		db:           db,
		ble:          ble,
		log:          log,
		activeTasks:  activeTasks,
		notifier:     notifier,
		bleOperation: bleOperation,
		ctx:          ctx,
	}
}

// ProcessSyncQueue checks the sync queue and processes cameras that need syncing.
func (c *Coordinator) ProcessSyncQueue() {
	if !c.db.GetConfig().SyncEnabled {
		c.log.Debug("Sync is disabled, skipping sync queue processing")
		return
	}

	c.log.Trace("Processing sync queue")
	syncQueue := c.db.GetSyncQueue()

	for _, entry := range syncQueue {
		// Skip cameras that already have an active sync goroutine
		c.mutex.RLock()
		_, exists := c.activeTasks[entry.CameraID]
		c.mutex.RUnlock()
		if exists {
			c.log.Tracef("Camera %s is already being synced, skipping", entry.CameraID)
			continue
		}

		camera, found := c.db.GetCameraByID(entry.CameraID)
		if !found {
			c.log.Warnf("Camera %s not found, removing from sync queue", entry.CameraID)
			c.db.RemoveSyncQueueEntry(entry.CameraID)
			continue
		}

		if !camera.Status.IsManaged || !camera.Status.IsPaired {
			c.log.Warnf("Camera %s is not managed+paired, removing from sync queue", camera.Camera.Name)
			c.db.RemoveSyncQueueEntry(entry.CameraID)
			continue
		}

		if !camera.Status.IsReachable {
			c.log.Tracef("Camera %s is not reachable, skipping sync", entry.CameraID)
			continue
		}

		if camera.Status.IsSyncing {
			c.log.Tracef("Camera %s is already marked as syncing, skipping", entry.CameraID)
			continue
		}

		c.log.Infof("Starting sync for camera %s", camera.Camera.Name)
		// Look up the DB key for this camera
		dbKey, _, _, _ := c.db.GetCameraIdentifiers(camera.Camera.ID)
		c.db.UpdateCameraSyncingStatus(dbKey, true)

		taskCtx, taskCancel := context.WithCancel(c.ctx)
		syncTask := &SyncTask{
			CameraID:   camera.Camera.ID,
			DBKey:      dbKey,
			BLEAddress: camera.Camera.BLEAddress,
			CameraName: camera.Camera.Name,
			Cancel:     taskCancel,
			Ctx:        taskCtx,
		}

		c.mutex.Lock()
		c.activeTasks[entry.CameraID] = syncTask
		c.mutex.Unlock()

		go c.PerformCameraSync(syncTask)
	}
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (c *Coordinator) ForceSync(cameraID string) (*database.SyncQueueEntry, error) {
	camera, found := c.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}
	if !camera.Status.IsManaged || !camera.Status.IsPaired {
		return nil, fmt.Errorf("camera %s must be managed and paired before syncing", camera.Camera.Name)
	}

	dbKey, _, name, _ := c.db.GetCameraIdentifiers(cameraID)

	for _, entry := range c.db.GetSyncQueue() {
		if entry.CameraID == cameraID {
			c.log.Infof("Camera %s is already in the sync queue", name)
			return entry, nil
		}
	}

	syncEntry := &database.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		Priority:         SyncPriorityManual,
		ProgressPercent:  0,
		CurrentOperation: "Waiting to start",
	}

	if err := c.db.AddSyncQueueEntry(syncEntry); err != nil {
		return nil, fmt.Errorf("failed to add camera to sync queue: %v", err)
	}

	c.db.ResetSyncStatus(dbKey)
	c.log.Infof("Added camera %s to sync queue", name)
	c.notifier()

	return syncEntry, nil
}

// CancelSync cancels an ongoing sync operation and removes the camera from the sync queue
func (c *Coordinator) CancelSync(cameraID string) error {
	dbKey, _, name, found := c.db.GetCameraIdentifiers(cameraID)
	if !found {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	c.log.Infof("Canceling sync for camera %s", name)

	c.mutex.Lock()
	task, exists := c.activeTasks[cameraID]
	if exists {
		if task.Cancel != nil {
			task.Cancel()
		}
		delete(c.activeTasks, cameraID)
	}
	c.mutex.Unlock()

	c.db.RemoveSyncQueueEntry(cameraID)
	c.db.UpdateCameraSyncingStatus(dbKey, false)
	c.log.Infof("Cancelled sync for camera %s", name)
	c.notifier()

	return nil
}

// PerformCameraSync handles the actual syncing of a camera
func (c *Coordinator) PerformCameraSync(task *SyncTask) {
	defer func() {
		c.mutex.Lock()
		delete(c.activeTasks, task.CameraID)
		c.mutex.Unlock()
		c.db.UpdateCameraSyncingStatus(task.DBKey, false)
		c.db.RemoveSyncQueueEntry(task.CameraID)
		c.notifier()
	}()

	c.log.Infof("Syncing camera %s (%s)", task.CameraName, task.BLEAddress)

	camera, found := c.db.GetCameraByID(task.CameraID)
	if !found {
		c.log.Errorf("Cannot sync camera %s: not found in database", task.CameraID)
		return
	}

	syncEntry := &database.SyncQueueEntry{
		CameraID: task.CameraID,
	}

	updateProgress := func(operation string, percent int32) {
		syncEntry.CurrentOperation = operation
		syncEntry.ProgressPercent = percent
		c.updateSyncQueueEntrySafely(task, syncEntry)
		c.notifier()
		c.log.Infof("Sync for camera %s: %s (%d%%)", task.CameraName, operation, percent)
	}

	config := c.db.GetConfig()
	syncCtx, cancel := context.WithTimeout(task.Ctx, 30*time.Minute)
	defer cancel()

	// Step 1: Connect to camera via BLE
	updateProgress("Connecting via BLE", 10)

	connErr := c.bleOperation(syncCtx, "ConnectForSync", func() error {
		return c.ble.Connect(task.BLEAddress)
	})
	if connErr != nil {
		updateProgress("BLE Connection Failed", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to connect to camera via BLE: %v", connErr)
		return
	}

	defer func() {
		disconnectErr := c.bleOperation(syncCtx, "DisconnectAfterSync", func() error {
			return c.ble.Disconnect(task.BLEAddress)
		})
		if disconnectErr != nil {
			c.log.Warnf("Failed to disconnect from BLE after sync: %v", disconnectErr)
		}
	}()

	// Keep camera awake during the entire sync
	keepAliveCtx, keepAliveCancel := context.WithCancel(syncCtx)
	defer keepAliveCancel()
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-keepAliveCtx.Done():
				return
			case <-ticker.C:
				if err := c.ble.KeepAlive(task.BLEAddress); err != nil {
					c.log.Debugf("Keep-alive failed: %v", err)
				}
			}
		}
	}()

	// Step 2: Connect to camera WiFi
	updateProgress("Connecting to WiFi", 30)

	wifiManager := wifi.NewWiFiManager(c.log)

	wifiCtx, wifiCancel := context.WithTimeout(syncCtx, time.Duration(config.ConnectTimeoutSeconds)*time.Second)
	defer wifiCancel()

	connWifiErr := wifiManager.Connect(wifiCtx, camera.Camera.WiFiSSID, camera.Camera.WiFiPassword)
	if connWifiErr != nil {
		updateProgress("WiFi Connection Failed", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to connect to camera WiFi: %v", connWifiErr)
		return
	}

	c.log.Infof("WiFi connected to camera %s (SSID: %s)", camera.Camera.Name, camera.Camera.WiFiSSID)

	defer func() {
		disconnectErr := wifiManager.Disconnect()
		if disconnectErr != nil {
			c.log.Warnf("Failed to disconnect from WiFi: %v", disconnectErr)
		}
	}()

	// Step 3: Verify GoPro HTTP API is reachable
	c.log.Info("Verifying GoPro HTTP API connectivity...")
	for attempt := 1; attempt <= 10; attempt++ {
		_, err := wifiManager.GetCameraStatus(syncCtx)
		if err == nil {
			c.log.Info("GoPro HTTP API is reachable")
			break
		}
		if attempt == 10 {
			updateProgress("GoPro API unreachable", syncEntry.ProgressPercent)
			c.log.Errorf("GoPro HTTP API not reachable after %d attempts: %v", attempt, err)
			return
		}
		c.log.Debugf("GoPro API not ready (attempt %d/10): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}

	// Step 4: Download media
	updateProgress("Downloading media", 60)

	if err := os.MkdirAll(config.DestinationFolder, 0755); err != nil {
		updateProgress("Failed to create destination folder", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to create destination folder: %v", err)
		return
	}

	cameraFolder := filepath.Join(config.DestinationFolder, camera.Camera.Name)
	if err := os.MkdirAll(cameraFolder, 0755); err != nil {
		updateProgress("Failed to create camera folder", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to create camera-specific folder: %v", err)
		return
	}

	downloadCtx, downloadCancel := context.WithTimeout(syncCtx, 20*time.Minute)
	defer downloadCancel()

	downloadedFiles, err := wifiManager.DownloadVideos(downloadCtx, cameraFolder, int(config.DaysThreshold))
	if err != nil {
		updateProgress("Media Download Failed", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to download media: %v", err)
		return
	}

	c.log.Infof("Download complete for camera %s: %d files downloaded", camera.Camera.Name, len(downloadedFiles))

	// Step 5: Finalize — defer handles IsSyncing=false, queue cleanup, and notification
	c.db.MarkCameraSynced(task.DBKey)
	c.log.Infof("Camera %s synced successfully, %d files downloaded", task.CameraName, len(downloadedFiles))
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
