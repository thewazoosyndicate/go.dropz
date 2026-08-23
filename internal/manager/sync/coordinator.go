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
	bleOperation   func(context.Context, bool, func() error) error
	ctx            context.Context
	syncSem        chan struct{}
}

// NewCoordinator creates a new sync coordinator
func NewCoordinator(db *database.Database, ble *ble.Manager, log *logrus.Logger, activeTasks map[string]*SyncTask, notifier func(), bleOperation func(context.Context, bool, func() error) error, ctx context.Context) *Coordinator {
	return &Coordinator{
		db:           db,
		ble:          ble,
		log:          log,
		activeTasks:  activeTasks,
		notifier:     notifier,
		bleOperation: bleOperation,
		ctx:          ctx,
		syncSem:      make(chan struct{}, 1),
	}
}

// ProcessSyncQueue checks the sync queue and processes cameras that need syncing.
func (c *Coordinator) ProcessSyncQueue() {
	if !c.db.GetConfig().SyncEnabled {
		c.log.Debug("Sync is disabled, skipping sync queue processing")
		return
	}

	c.log.Trace("Processing sync queue")
	for _, entry := range c.db.GetSyncQueue() {
		task, ok := c.tryClaimCamera(entry)
		if !ok {
			continue
		}
		// Only one sync at a time — BLE and WiFi can't handle concurrent cameras
		select {
		case c.syncSem <- struct{}{}:
			go c.PerformCameraSync(task)
		default:
			c.mutex.Lock()
			delete(c.activeTasks, task.CameraID)
			c.mutex.Unlock()
			c.db.UpdateCameraSyncingStatusByID(task.CameraID, false)
			task.Cancel()
		}
	}
}

// tryClaimCamera validates a sync queue entry and atomically claims the camera for syncing.
func (c *Coordinator) tryClaimCamera(entry *database.SyncQueueEntry) (*SyncTask, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.activeTasks[entry.CameraID]; exists {
		c.log.Tracef("Camera %s is already being synced, skipping", entry.CameraID)
		return nil, false
	}

	camera, found := c.db.GetCameraByID(entry.CameraID)
	if !found {
		c.log.Warnf("Camera %s not found, removing from sync queue", entry.CameraID)
		c.db.RemoveSyncQueueEntry(entry.CameraID)
		return nil, false
	}

	if !camera.Status.IsManaged || !camera.Status.IsPaired {
		c.log.Warnf("Camera %s is not managed+paired, removing from sync queue", camera.Camera.Name)
		c.db.RemoveSyncQueueEntry(entry.CameraID)
		return nil, false
	}

	if !camera.Status.IsReachable {
		c.log.Tracef("Camera %s is not reachable, skipping sync", entry.CameraID)
		return nil, false
	}

	if camera.Status.IsSyncing {
		c.log.Tracef("Camera %s is already marked as syncing, skipping", entry.CameraID)
		return nil, false
	}

	c.log.Infof("Starting sync for camera %s", camera.Camera.Name)
	c.db.UpdateCameraSyncingStatusByID(camera.Camera.ID, true)

	taskCtx, taskCancel := context.WithCancel(c.ctx)
	task := &SyncTask{
		CameraID:   camera.Camera.ID,
		BLEAddress: camera.Camera.BLEAddress,
		CameraName: camera.Camera.Name,
		Cancel:     taskCancel,
		Ctx:        taskCtx,
	}

	c.activeTasks[entry.CameraID] = task
	return task, true
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

	for _, entry := range c.db.GetSyncQueue() {
		if entry.CameraID == cameraID {
			c.log.Infof("Camera %s is already in the sync queue", camera.Camera.Name)
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

	c.db.ResetSyncStatusByID(cameraID)
	c.log.Infof("Added camera %s to sync queue", camera.Camera.Name)
	c.notifier()

	return syncEntry, nil
}

// CancelSync cancels an ongoing sync operation and removes the camera from the sync queue
func (c *Coordinator) CancelSync(cameraID string) error {
	camera, found := c.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	c.log.Infof("Canceling sync for camera %s", camera.Camera.Name)

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
	c.db.UpdateCameraSyncingStatusByID(cameraID, false)
	c.log.Infof("Cancelled sync for camera %s", camera.Camera.Name)
	c.notifier()

	return nil
}

// PerformCameraSync handles the actual syncing of a camera
func (c *Coordinator) PerformCameraSync(task *SyncTask) {
	defer func() {
		<-c.syncSem
		// Only clean up if this goroutine still owns the task (CancelSync may have already claimed it)
		c.mutex.Lock()
		_, stillActive := c.activeTasks[task.CameraID]
		if stillActive {
			delete(c.activeTasks, task.CameraID)
		}
		c.mutex.Unlock()
		if stillActive {
			c.db.UpdateCameraSyncingStatusByID(task.CameraID, false)
			c.db.RemoveSyncQueueEntry(task.CameraID)
		}
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

	connErr := c.bleOperation(syncCtx, true, func() error {
		return c.ble.Connect(task.BLEAddress)
	})
	if connErr != nil {
		updateProgress("BLE Connection Failed", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to connect to camera via BLE: %v", connErr)
		c.db.SetLastSyncErrorByID(task.CameraID, "BLE connection failed")
		return
	}

	defer func() {
		disconnectErr := c.bleOperation(syncCtx, false, func() error {
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
		c.db.SetLastSyncErrorByID(task.CameraID, "WiFi connection failed")
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
			c.db.SetLastSyncErrorByID(task.CameraID, "GoPro API unreachable")
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
		c.db.SetLastSyncErrorByID(task.CameraID, "Failed to create destination folder")
		return
	}

	cameraFolder := filepath.Join(config.DestinationFolder, camera.Camera.WiFiSSID)
	if err := os.MkdirAll(cameraFolder, 0755); err != nil {
		updateProgress("Failed to create camera folder", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to create camera-specific folder: %v", err)
		c.db.SetLastSyncErrorByID(task.CameraID, "Failed to create camera folder")
		return
	}

	downloadCtx, downloadCancel := context.WithTimeout(syncCtx, 20*time.Minute)
	defer downloadCancel()

	downloadedFiles, err := wifiManager.DownloadVideos(downloadCtx, cameraFolder, int(config.DaysThreshold))
	if err != nil {
		updateProgress("Media Download Failed", syncEntry.ProgressPercent)
		c.log.Errorf("Failed to download media: %v", err)
		c.db.SetLastSyncErrorByID(task.CameraID, "Media download failed")
		return
	}

	c.log.Infof("Download complete for camera %s: %d files downloaded", camera.Camera.Name, len(downloadedFiles))

	// Step 5: Finalize — defer handles IsSyncing=false, queue cleanup, and notification
	c.db.MarkCameraSyncedByID(task.CameraID)
	c.db.SetLastSyncErrorByID(task.CameraID, "")
	c.log.Infof("Camera %s synced successfully, %d files downloaded", task.CameraName, len(downloadedFiles))
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
