package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/logging"
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/store"
	"github.com/dropz/dropz/internal/wifi"
)

// Sync priority constants
const (
	SyncPriorityManual = 10 // High priority for manually requested syncs
	SyncPriorityAuto   = 5  // Normal priority for automatic syncs
)

// Operation timeouts
const (
	syncOverallTimeout = 30 * time.Minute
	downloadTimeout    = 20 * time.Minute
	keepAliveInterval  = 3 * time.Second
	apiReadyAttempts   = 10
	apiReadyDelay      = 2 * time.Second
	idleWaitTimeout    = 2 * time.Minute // max wait for busy/encoding to clear
	idleWaitPoll       = 5 * time.Second
	apReadyTimeout     = 10 * time.Second // max wait for AP Mode (status 69)
)

// BLEOperation runs a BLE operation with the manager's retry policy.
type BLEOperation func(ctx context.Context, critical bool, op func() error) error

// SyncTask represents a camera sync task
type SyncTask struct {
	CameraID   string
	BLEAddress string
	CameraName string
	Cancel     context.CancelFunc
	Ctx        context.Context
	// FileNames limits the download to a media-browser selection;
	// CatalogOnly refreshes catalog and thumbnails without downloading.
	FileNames   []string
	CatalogOnly bool
}

// Coordinator handles sync orchestration
type Coordinator struct {
	db           *store.Store
	ble          *ble.Manager
	log          *slog.Logger
	rootLog      *slog.Logger // for wifi.NewWiFiManager, which tags its own component
	activeTasks  map[string]*SyncTask
	mutex        sync.RWMutex
	notifier     func()
	bleOperation BLEOperation
	ctx          context.Context
	syncSem      chan struct{}
}

// NewCoordinator creates a new sync coordinator
func NewCoordinator(ctx context.Context, db *store.Store, ble *ble.Manager, log *slog.Logger, notifier func(), bleOperation BLEOperation) *Coordinator {
	return &Coordinator{
		db:           db,
		ble:          ble,
		log:          log.With("component", "sync"),
		rootLog:      log,
		activeTasks:  make(map[string]*SyncTask),
		notifier:     notifier,
		bleOperation: bleOperation,
		ctx:          ctx,
		syncSem:      make(chan struct{}, 1),
	}
}

// ProcessSyncQueue checks the sync queue and processes cameras that need syncing.
func (c *Coordinator) ProcessSyncQueue() {
	if !c.db.GetConfig().SyncEnabled {
		logging.Trace(c.log, "Sync disabled, skipping queue processing")
		return
	}

	logging.Trace(c.log, "Processing sync queue")
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
			c.log.Info("Sync deferred, another sync in progress",
				"camera_id", task.CameraID, "camera", task.CameraName)
			c.mutex.Lock()
			delete(c.activeTasks, task.CameraID)
			c.mutex.Unlock()
			c.db.UpdateCameraSyncingStatusByID(task.CameraID, false)
			task.Cancel()
		}
	}
}

// tryClaimCamera validates a sync queue entry and atomically claims the camera for syncing.
func (c *Coordinator) tryClaimCamera(entry *model.SyncQueueEntry) (*SyncTask, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.activeTasks[entry.CameraID]; exists {
		logging.Trace(c.log, "Camera already being synced, skipping", "camera_id", entry.CameraID)
		return nil, false
	}

	camera, found := c.db.GetCameraByID(entry.CameraID)
	if !found {
		c.log.Warn("Camera not found, removing from sync queue", "camera_id", entry.CameraID)
		c.db.RemoveSyncQueueEntry(entry.CameraID)
		return nil, false
	}

	if !camera.Status.IsManaged || !camera.Status.IsPaired {
		c.log.Warn("Camera not managed and paired, removing from sync queue", camera.LogAttrs()...)
		c.db.RemoveSyncQueueEntry(entry.CameraID)
		return nil, false
	}

	if !camera.Status.IsReachable {
		logging.Trace(c.log, "Camera not reachable, skipping sync", "camera_id", entry.CameraID)
		return nil, false
	}

	if camera.Status.IsSyncing {
		logging.Trace(c.log, "Camera already marked syncing, skipping", "camera_id", entry.CameraID)
		return nil, false
	}

	// Started is logged in PerformCameraSync; a claim can still be
	// deferred when another sync holds the semaphore.
	c.log.Debug("Sync task claimed", camera.LogAttrs()...)
	c.db.UpdateCameraSyncingStatusByID(camera.Camera.ID, true)

	taskCtx, taskCancel := context.WithCancel(c.ctx)
	task := &SyncTask{
		CameraID:    camera.Camera.ID,
		BLEAddress:  camera.Camera.BLEAddress,
		CameraName:  camera.Camera.Name,
		Cancel:      taskCancel,
		Ctx:         taskCtx,
		FileNames:   entry.FileNames,
		CatalogOnly: entry.CatalogOnly,
	}

	c.activeTasks[entry.CameraID] = task
	return task, true
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (c *Coordinator) ForceSync(cameraID string) (*model.SyncQueueEntry, error) {
	camera, found := c.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}
	if !camera.Status.IsManaged || !camera.Status.IsPaired {
		return nil, fmt.Errorf("%w: %s", model.ErrNotManagedPaired, camera.Camera.Name)
	}

	for _, entry := range c.db.GetSyncQueue() {
		if entry.CameraID == cameraID {
			c.log.Debug("Camera already in sync queue", camera.LogAttrs()...)
			return entry, nil
		}
	}

	syncEntry := &model.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		Priority:         SyncPriorityManual,
		ProgressPercent:  0,
		CurrentOperation: "Waiting to start",
	}

	if err := c.db.AddSyncQueueEntry(syncEntry); err != nil {
		return nil, fmt.Errorf("failed to add camera to sync queue: %w", err)
	}

	c.db.ResetSyncStatusByID(cameraID)
	c.log.Info("Camera added to sync queue", camera.LogAttrs()...)
	c.notifier()

	return syncEntry, nil
}

// RequestMediaDownload queues a media-browser request: a selection of
// files to fetch, or (empty selection) a catalog-only refresh.
func (c *Coordinator) RequestMediaDownload(cameraID string, fileNames []string) (*model.SyncQueueEntry, error) {
	camera, found := c.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}
	if !camera.Status.IsManaged || !camera.Status.IsPaired {
		return nil, fmt.Errorf("%w: %s", model.ErrNotManagedPaired, camera.Camera.Name)
	}

	// Merge into an already-queued entry: union selections; any file
	// request upgrades a catalog-only entry; a queued full sync already
	// covers everything.
	var merged *model.SyncQueueEntry
	updated, err := c.db.MutateSyncQueueEntry(cameraID, func(entry *model.SyncQueueEntry) {
		if len(fileNames) > 0 && (entry.CatalogOnly || len(entry.FileNames) > 0) {
			seen := make(map[string]bool, len(entry.FileNames))
			for _, n := range entry.FileNames {
				seen[n] = true
			}
			for _, n := range fileNames {
				if !seen[n] {
					entry.FileNames = append(entry.FileNames, n)
				}
			}
			entry.CatalogOnly = false
		}
		entryCopy := *entry
		merged = &entryCopy
	})
	if err != nil {
		return nil, err
	}
	if updated {
		c.notifier()
		return merged, nil
	}

	syncEntry := &model.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		Priority:         SyncPriorityManual,
		CurrentOperation: "Waiting to start",
		FileNames:        fileNames,
		CatalogOnly:      len(fileNames) == 0,
	}
	if err := c.db.AddSyncQueueEntry(syncEntry); err != nil {
		return nil, fmt.Errorf("failed to queue media download: %w", err)
	}
	c.log.Info("Media download queued", "camera", camera.Camera.Name, "files", len(fileNames), "catalog_only", syncEntry.CatalogOnly)
	c.notifier()
	return syncEntry, nil
}

// CancelSync cancels an ongoing sync operation and removes the camera from the sync queue
func (c *Coordinator) CancelSync(cameraID string) error {
	camera, found := c.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}

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
	c.log.Info("Sync cancelled", camera.LogAttrs()...)
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

	c.log.Info("Sync started", "camera_id", task.CameraID, "camera", task.CameraName, "ble_addr", task.BLEAddress)

	camera, found := c.db.GetCameraByID(task.CameraID)
	if !found {
		c.log.Warn("Camera not found for sync", "camera_id", task.CameraID)
		return
	}

	// Re-read the BLE address at sync time: the enqueue-time address can be
	// stale on macOS, which rotates BLE identifiers.
	bleAddress := camera.Camera.BLEAddress
	if bleAddress != task.BLEAddress {
		c.log.Debug("BLE address changed since enqueue", "camera", task.CameraName,
			"old", task.BLEAddress, "new", bleAddress)
	}

	syncEntry := &model.SyncQueueEntry{
		CameraID: task.CameraID,
	}

	updateProgress := func(operation string, percent int32) {
		syncEntry.CurrentOperation = operation
		syncEntry.ProgressPercent = percent
		c.updateSyncQueueEntrySafely(task, syncEntry)
		c.notifier()
		c.log.Debug("Sync progress", "camera", task.CameraName, "operation", operation, "percent", percent)
	}

	config := c.db.GetConfig()
	syncCtx, cancel := context.WithTimeout(task.Ctx, syncOverallTimeout)
	defer cancel()

	// Step 1: Connect to camera via BLE
	updateProgress("Connecting via BLE", 10)

	// Serialize with status checks and settings sessions on this camera.
	release, sessionErr := c.ble.AcquireSession(bleAddress, 30*time.Second)
	if sessionErr != nil {
		updateProgress("Camera busy", syncEntry.ProgressPercent)
		c.log.Warn("Sync could not acquire BLE session", "camera", task.CameraName, "err", sessionErr)
		c.db.SetLastSyncErrorByID(task.CameraID, "Camera busy with another operation")
		return
	}
	defer release()

	connErr := c.bleOperation(syncCtx, true, func() error {
		return c.ble.Connect(bleAddress)
	})
	if connErr != nil {
		updateProgress("BLE Connection Failed", syncEntry.ProgressPercent)
		c.log.Error("Sync failed, BLE connection", append(camera.LogAttrs(), "err", connErr)...)
		c.db.SetLastSyncErrorByID(task.CameraID, "BLE connection failed")
		return
	}

	defer func() {
		disconnectErr := c.bleOperation(syncCtx, false, func() error {
			return c.ble.Disconnect(bleAddress)
		})
		if disconnectErr != nil {
			c.log.Warn("Failed to disconnect from BLE after sync", "err", disconnectErr)
		}
	}()

	// Keep camera awake during the entire sync
	keepAliveCtx, keepAliveCancel := context.WithCancel(syncCtx)
	defer keepAliveCancel()
	go func() {
		ticker := time.NewTicker(keepAliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-keepAliveCtx.Done():
				return
			case <-ticker.C:
				if err := c.ble.KeepAlive(bleAddress); err != nil {
					c.log.Debug("Keep-alive failed", "camera", task.CameraName, "err", err)
				}
			}
		}
	}()

	// Step 1b: Wait until the camera is neither busy nor encoding
	// (OpenGoPro state_management: gate work on statuses 8 and 10).
	updateProgress("Waiting for camera to be idle", 20)
	if err := c.waitForCameraIdle(syncCtx, bleAddress); err != nil {
		updateProgress("Camera busy", syncEntry.ProgressPercent)
		c.log.Error("Sync failed, camera did not become idle", append(camera.LogAttrs(), "err", err)...)
		c.db.SetLastSyncErrorByID(task.CameraID, "Camera busy or recording")
		return
	}

	// Step 1c: Wait for the camera's AP to actually be up (status 69); the
	// Set AP Control response only acknowledges the request. On timeout,
	// bounce the AP once (spec FAQ recovery for intermittent AP failures),
	// then proceed either way and let the join retries have their chance.
	updateProgress("Waiting for camera WiFi AP", 25)
	if err := c.ble.WaitForWiFiAPReady(bleAddress, apReadyTimeout); err != nil {
		// Expected recovery step; only a failing bounce is Warn-worthy
		c.log.Debug("Camera AP not ready, bouncing AP", "camera", task.CameraName, "err", err)
		if bounceErr := c.ble.SetAPControl(bleAddress, ble.WiFiAPModeBounce); bounceErr != nil {
			c.log.Warn("AP bounce failed", "camera", task.CameraName, "err", bounceErr)
		} else if err := c.ble.WaitForWiFiAPReady(bleAddress, apReadyTimeout); err != nil {
			c.log.Warn("Camera AP still not ready after bounce", "camera", task.CameraName, "err", err)
		}
	}

	// Step 2: Connect to camera WiFi
	updateProgress("Connecting to WiFi", 30)

	wifiManager := wifi.NewWiFiManager(c.rootLog)

	wifiCtx, wifiCancel := context.WithTimeout(syncCtx, time.Duration(config.ConnectTimeoutSeconds)*time.Second)
	defer wifiCancel()

	connWifiErr := wifiManager.Connect(wifiCtx, camera.Camera.WiFiSSID, camera.Camera.WiFiPassword)
	if connWifiErr != nil {
		updateProgress("WiFi Connection Failed", syncEntry.ProgressPercent)
		c.log.Error("Sync failed, WiFi connection", append(camera.LogAttrs(), "err", connWifiErr)...)
		c.db.SetLastSyncErrorByID(task.CameraID, "WiFi connection failed")
		return
	}

	c.log.Info("WiFi connected to camera", "camera", camera.Camera.Name, "ssid", camera.Camera.WiFiSSID)

	defer func() {
		disconnectErr := wifiManager.Disconnect()
		if disconnectErr != nil {
			c.log.Warn("Failed to disconnect from WiFi", "camera", task.CameraName, "err", disconnectErr)
		}
	}()

	// Step 3: Verify GoPro HTTP API is reachable
	for attempt := 1; attempt <= apiReadyAttempts; attempt++ {
		_, err := wifiManager.GetCameraStatus(syncCtx)
		if err == nil {
			c.log.Debug("GoPro HTTP API reachable", "camera", task.CameraName, "attempt", attempt)
			break
		}
		if attempt == apiReadyAttempts {
			updateProgress("GoPro API unreachable", syncEntry.ProgressPercent)
			c.log.Error("Sync failed, GoPro HTTP API not reachable", append(camera.LogAttrs(), "attempts", attempt, "err", err)...)
			c.db.SetLastSyncErrorByID(task.CameraID, "GoPro API unreachable")
			return
		}
		c.log.Debug("GoPro API not ready", "camera", task.CameraName, "attempt", attempt, "max", apiReadyAttempts, "err", err)
		time.Sleep(apiReadyDelay)
	}

	// Step 4: Media catalog and download
	updateProgress("Preparing download", 50)

	if err := os.MkdirAll(config.DestinationFolder, 0755); err != nil {
		updateProgress("Failed to create destination folder", syncEntry.ProgressPercent)
		c.log.Error("Sync failed, cannot create destination folder", append(camera.LogAttrs(), "folder", config.DestinationFolder, "err", err)...)
		c.db.SetLastSyncErrorByID(task.CameraID, "Failed to create destination folder")
		return
	}

	cameraFolder := filepath.Join(config.DestinationFolder, camera.Camera.WiFiSSID)
	if err := os.MkdirAll(cameraFolder, 0755); err != nil {
		updateProgress("Failed to create camera folder", syncEntry.ProgressPercent)
		c.log.Error("Sync failed, cannot create camera folder", append(camera.LogAttrs(), "folder", cameraFolder, "err", err)...)
		c.db.SetLastSyncErrorByID(task.CameraID, "Failed to create camera folder")
		return
	}

	downloadCtx, downloadCancel := context.WithTimeout(syncCtx, downloadTimeout)
	defer downloadCancel()

	// Step 4a: capture the media catalog + thumbnails while the WiFi link
	// is up; the media browser reads them offline later.
	updateProgress("Updating media catalog", 55)
	if mediaFiles, listErr := wifiManager.ListMedia(downloadCtx); listErr != nil {
		c.log.Warn("Media catalog refresh failed", "camera", task.CameraName, "err", listErr)
	} else {
		if err := WriteCatalog(cameraFolder, mediaFiles); err != nil {
			c.log.Warn("Media catalog write failed", "camera", task.CameraName, "err", err)
		}
		refreshThumbnails(downloadCtx, wifiManager, cameraFolder, mediaFiles, c.log)
	}

	if task.CatalogOnly {
		c.db.SetLastSyncErrorByID(task.CameraID, "")
		c.log.Info("Media catalog refreshed, no download requested", "camera", task.CameraName)
		c.notifier()
		return
	}

	// Turbo Transfer is opt-in: it is tuned for the GoPro app's parallel
	// chunked downloads and measured slower for our single sequential
	// stream. `ble-probe validate -wifi -speed` measures both on hardware.
	if config.TurboEnabled {
		if err := wifiManager.SetTurboTransfer(downloadCtx, true); err != nil {
			c.log.Debug("Turbo transfer not enabled", "camera", task.CameraName, "err", err)
		} else {
			defer func() {
				if err := wifiManager.SetTurboTransfer(syncCtx, false); err != nil {
					c.log.Debug("Turbo transfer not disabled", "camera", task.CameraName, "err", err)
				}
			}()
		}
	}

	updateProgress("Downloading media", 60)
	downloadedFiles, err := wifiManager.DownloadVideos(downloadCtx, cameraFolder, int(config.DaysThreshold), task.FileNames)
	if err != nil {
		updateProgress("Media Download Failed", syncEntry.ProgressPercent)
		msg := "Sync failed, media download"
		if errors.Is(err, context.DeadlineExceeded) {
			msg = "Sync failed, media download timed out"
		}
		c.log.Error(msg, append(camera.LogAttrs(), "timeout", downloadTimeout, "err", err)...)
		c.db.SetLastSyncErrorByID(task.CameraID, "Media download failed")
		return
	}

	// Step 5: Finalize — defer handles IsSyncing=false, queue cleanup, and notification.
	// A selection download is partial by definition: the camera is not
	// "synced", so the auto-sync eligibility must stay untouched.
	if len(task.FileNames) == 0 {
		c.db.MarkCameraSyncedByID(task.CameraID)
	}
	c.db.SetLastSyncErrorByID(task.CameraID, "")
	c.log.Info("Camera synced", append(camera.LogAttrs(), "files", len(downloadedFiles))...)
}

// waitForCameraIdle polls busy (8) and encoding (10) until both clear.
func (c *Coordinator) waitForCameraIdle(ctx context.Context, bleAddress string) error {
	deadline := time.Now().Add(idleWaitTimeout)
	for {
		statuses, err := c.ble.QueryStatuses(bleAddress, []byte{ble.StatusSystemBusy, ble.StatusEncoding})
		if err != nil {
			return fmt.Errorf("busy query failed: %w", err)
		}
		if !statusesInUse(statuses) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("camera still busy after %v", idleWaitTimeout)
		}
		logging.Trace(c.log, "Camera busy or encoding, waiting", "ble_addr", bleAddress)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(idleWaitPoll):
		}
	}
}

// statusesInUse reports whether busy or encoding statuses are nonzero.
func statusesInUse(statuses map[byte][]byte) bool {
	for _, id := range []byte{ble.StatusSystemBusy, ble.StatusEncoding} {
		if v, ok := statuses[id]; ok && len(v) >= 1 && v[0] != 0 {
			return true
		}
	}
	return false
}

// updateSyncQueueEntrySafely updates the sync queue entry only if the task hasn't been cancelled
func (c *Coordinator) updateSyncQueueEntrySafely(task *SyncTask, syncEntry *model.SyncQueueEntry) error {
	// Check if the task has been cancelled
	if task.Ctx != nil {
		select {
		case <-task.Ctx.Done():
			// Task has been cancelled, don't update the database
			c.log.Debug("Skipping sync queue update for cancelled task", "camera_id", task.CameraID)
			return nil
		default:
			// Task is still active, proceed with update
		}
	}

	return c.db.UpdateSyncQueueEntry(syncEntry)
}
