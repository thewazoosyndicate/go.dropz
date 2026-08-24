package syncer

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
	teardownTimeout    = 30 * time.Second // BLE/WiFi cleanup after cancel or failure
	keepAliveInterval  = 3 * time.Second
	apiReadyAttempts   = 10
	apiReadyDelay      = 2 * time.Second
	idleWaitTimeout    = 2 * time.Minute // max wait for busy/encoding to clear
	idleWaitPoll       = 5 * time.Second
	apReadyTimeout     = 10 * time.Second // max wait for AP Mode (status 69)
)

// bleClient is the slice of ble.Manager the sync flow uses; an interface so
// tests can drive PerformCameraSync without a radio.
type bleClient interface {
	AcquireSession(macAddress string, timeout time.Duration) (func(), error)
	Connect(macAddress string) error
	Disconnect(macAddress string) error
	KeepAlive(macAddress string) error
	QueryStatuses(macAddress string, statusIDs []byte) (map[byte][]byte, error)
	WaitForWiFiAPReady(macAddress string, timeout time.Duration) error
	SetAPControl(macAddress string, mode ble.WiFiAPMode) error
	ForgetDevice(macAddress string) error
}

// wifiClient is the slice of wifi.WiFiManager the sync flow uses.
type wifiClient interface {
	Connect(ctx context.Context, ssid, password string) error
	Disconnect() error
	GetCameraStatus(ctx context.Context) (map[string]interface{}, error)
	ListMedia(ctx context.Context) ([]wifi.MediaFile, error)
	SetTurboTransfer(ctx context.Context, enable bool) error
	DownloadVideos(ctx context.Context, destDir string, daysInPast int, fileNames []string, progress wifi.ProgressFunc) ([]string, error)
	DownloadThumbnail(ctx context.Context, cameraPath, outPath string) error
	DownloadLRV(ctx context.Context, cameraPath, outPath string) error
}

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
	ble          bleClient
	log          *slog.Logger
	activeTasks  map[string]*SyncTask
	// Open interactive preview sessions, keyed like activeTasks; a camera
	// appears in both while its preview link is up.
	previewSessions map[string]*previewSession
	previewIdle     time.Duration // tests shrink the linger window
	// Last establish attempt per camera, for the armed-session backoff
	previewAttempt map[string]time.Time
	// Preview transcoder, swappable so tests run without ffmpeg
	transcode func(ctx context.Context, src, dst string) error
	// In-flight library preview transcodes, keyed by source path
	previewGen map[string]*previewGen
	mutex           sync.RWMutex
	notifier     func()
	bleOperation ble.Operation
	ctx          context.Context
	syncSem      chan struct{}
	wifiFactory  func() wifiClient
	// Tracks sync goroutines so shutdown can wait for teardown instead of
	// killing the process mid-download.
	wg sync.WaitGroup
}

// NewCoordinator creates a new sync coordinator.
// wifiFactory may be nil; the real WiFi manager is used then.
func NewCoordinator(ctx context.Context, db *store.Store, bleManager bleClient, log *slog.Logger, notifier func(), bleOperation ble.Operation, wifiFactory func() wifiClient) *Coordinator {
	if wifiFactory == nil {
		// wifi.NewWiFiManager tags its own component, hence the root logger
		wifiFactory = func() wifiClient { return wifi.NewWiFiManager(log) }
	}
	return &Coordinator{
		db:              db,
		ble:             bleManager,
		log:             log.With("component", "sync"),
		activeTasks:     make(map[string]*SyncTask),
		previewSessions: make(map[string]*previewSession),
		previewIdle:     previewLinger,
		previewAttempt:  make(map[string]time.Time),
		transcode:       transcodePreview,
		notifier:        notifier,
		bleOperation:    bleOperation,
		ctx:             ctx,
		syncSem:         make(chan struct{}, 1),
		wifiFactory:     wifiFactory,
	}
}

// Wait blocks until every in-flight sync goroutine has finished.
func (c *Coordinator) Wait() {
	c.wg.Wait()
}

// ProcessSyncQueue checks the sync queue and processes cameras that need syncing.
func (c *Coordinator) ProcessSyncQueue() {
	// Piggybacks on the sync cadence: an armed camera's preview session
	// (re)establishes here once it is in range and the radio is idle.
	defer c.ensureArmedSessions()

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
		// Only one sync at a time - BLE and WiFi can't handle concurrent cameras
		select {
		case c.syncSem <- struct{}{}:
			c.wg.Add(1)
			go func() {
				defer c.wg.Done()
				c.PerformCameraSync(task)
			}()
		default:
			c.log.Info("Sync deferred, another sync in progress",
				"camera_id", task.CameraID, "camera", task.CameraName)
			c.mutex.Lock()
			delete(c.activeTasks, task.CameraID)
			c.mutex.Unlock()
			_ = c.db.UpdateCameraSyncingStatusByID(task.CameraID, false)
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
		_ = c.db.RemoveSyncQueueEntry(entry.CameraID)
		return nil, false
	}

	if !camera.Status.IsManaged || !camera.Status.IsPaired {
		c.log.Warn("Camera not managed and paired, removing from sync queue", camera.LogAttrs()...)
		_ = c.db.RemoveSyncQueueEntry(entry.CameraID)
		return nil, false
	}

	// A paused rig never auto-syncs; a manual request still does
	if entry.Priority < SyncPriorityManual && c.db.IsCameraSyncPaused(entry.CameraID) {
		c.log.Debug("Auto sync dropped, group paused", camera.LogAttrs()...)
		_ = c.db.RemoveSyncQueueEntry(entry.CameraID)
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
	_ = c.db.UpdateCameraSyncingStatusByID(camera.Camera.ID, true)

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

	_ = c.db.ResetSyncStatusByID(cameraID)
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

	_ = c.db.RemoveSyncQueueEntry(cameraID)
	_ = c.db.UpdateCameraSyncingStatusByID(cameraID, false)
	c.log.Info("Sync cancelled", camera.LogAttrs()...)
	c.notifier()

	return nil
}

// syncRun carries state across the steps of one camera sync.
type syncRun struct {
	task        *SyncTask
	camera      *model.CameraWithState
	bleAddress  string
	config      model.Config
	ctx         context.Context // task ctx bounded by syncOverallTimeout
	downloadCtx context.Context // ctx bounded by downloadTimeout
	wifi        wifiClient
	folder      string
	downloaded  int
	cleanups    []func()
	// session is the history record being built; files and phases are
	// mirrored into the queue entry as they change and copied here at
	// the end, so a cancel that drops the entry loses nothing.
	session *model.SyncSession
	files   []model.SyncFile
	phases  []model.SyncPhaseTiming
}

// cleanup registers teardown to run when the sync ends, in reverse order.
func (r *syncRun) cleanup(f func()) {
	r.cleanups = append(r.cleanups, f)
}

// finish records how the sync ended; the first call wins so a cancel
// racing a step failure keeps the truthful outcome.
func (r *syncRun) finish(outcome model.SyncOutcome, errMsg string) {
	if r.session.Outcome != "" {
		return
	}
	r.session.Outcome = outcome
	r.session.Error = errMsg
}

// errCatalogOnly ends a catalog-only run successfully after the catalog step.
var errCatalogOnly = errors.New("catalog refreshed, no download requested")

// errUpToDate ends a full sync before the WiFi cycle: BLE proved nothing
// changed on the camera and the local library is complete.
var errUpToDate = errors.New("camera verified up to date over BLE")

type syncStep struct {
	label   string // progress label while the step runs
	percent int32
	failMsg string // user-visible last-sync error when the step fails
	phase   model.SyncPhase
	run     func(*syncRun) error
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
			_ = c.db.UpdateCameraSyncingStatusByID(task.CameraID, false)
			_ = c.db.RemoveSyncQueueEntry(task.CameraID)
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

	syncCtx, cancel := context.WithTimeout(task.Ctx, syncOverallTimeout)
	defer cancel()

	started := time.Now()
	run := &syncRun{
		task:       task,
		camera:     camera,
		bleAddress: bleAddress,
		config:     c.db.GetConfig(),
		ctx:        syncCtx,
		session: &model.SyncSession{
			ID:        fmt.Sprintf("%s-%d", task.CameraID, started.UnixMilli()),
			CameraID:  task.CameraID,
			StartedAt: started,
			Selection: task.CatalogOnly || len(task.FileNames) > 0,
		},
	}
	_, _ = c.db.MutateSyncQueueEntry(task.CameraID, func(entry *model.SyncQueueEntry) {
		entry.StartedAt = started
	})
	defer func() {
		for i := len(run.cleanups) - 1; i >= 0; i-- {
			run.cleanups[i]()
		}
		c.recordSession(run)
	}()

	// Serialize with status checks and settings sessions on this camera.
	release, sessionErr := c.ble.AcquireSession(bleAddress, 30*time.Second)
	if sessionErr != nil {
		c.updateProgress(task, "Camera busy", 0)
		c.log.Warn("Sync could not acquire BLE session", "camera", task.CameraName, "err", sessionErr)
		_ = c.db.SetLastSyncErrorByID(task.CameraID, "Camera busy with another operation")
		run.finish(model.SyncOutcomeFailed, "Camera busy with another operation")
		return
	}
	defer release()

	steps := []syncStep{
		{"Connecting via BLE", 10, "BLE connection failed", model.SyncPhaseConnect, c.stepConnectBLE},
		{"Checking for changes", 15, "", model.SyncPhaseConnect, c.stepSkipIfClean},
		{"Waiting for camera to be idle", 20, "Camera busy or recording", model.SyncPhaseConnect, c.stepWaitIdle},
		{"Waiting for camera WiFi AP", 25, "", model.SyncPhaseLink, c.stepWaitAP},
		{"Connecting to WiFi", 30, "WiFi connection failed", model.SyncPhaseLink, c.stepConnectWiFi},
		{"Waiting for GoPro API", 40, "GoPro API unreachable", model.SyncPhaseLink, c.stepAwaitAPI},
		{"Preparing download", 50, "Failed to create destination folder", model.SyncPhaseCatalog, c.stepPrepareFolders},
		{"Updating media catalog", 55, "", model.SyncPhaseCatalog, c.stepCatalog},
		{"Downloading media", 60, "Media download failed", model.SyncPhaseTransfer, c.stepDownload},
	}
	run.session.StepCount = int32(len(steps))

	for i, step := range steps {
		run.session.StepIndex = int32(i + 1)
		_, _ = c.db.MutateSyncQueueEntry(task.CameraID, func(entry *model.SyncQueueEntry) {
			entry.StepIndex = int32(i + 1)
			entry.StepCount = int32(len(steps))
		})
		c.startPhase(run, step.phase)
		c.updateProgress(task, step.label, step.percent)
		err := step.run(run)
		if err == nil {
			continue
		}
		if errors.Is(err, errCatalogOnly) {
			_ = c.db.SetLastSyncErrorByID(task.CameraID, "")
			c.log.Info("Media catalog refreshed, no download requested", "camera", task.CameraName)
			run.finish(model.SyncOutcomeCatalogRefreshed, "")
			return
		}
		if errors.Is(err, errUpToDate) {
			// Verified over BLE, so the synced timestamp is honest.
			// The progress label is streamed before the defer removes the
			// entry; the frontend turns it into a toast.
			_ = c.db.MarkCameraSyncedByID(task.CameraID)
			_ = c.db.SetLastSyncErrorByID(task.CameraID, "")
			c.updateProgress(task, "Already up to date", 100)
			c.log.Info("Camera already up to date, WiFi skipped", camera.LogAttrs()...)
			run.finish(model.SyncOutcomeUpToDate, "")
			return
		}
		if task.Ctx.Err() != nil {
			// User cancel: the error is a consequence, not a fault
			run.finish(model.SyncOutcomeCancelled, "")
			return
		}
		failMsg := step.failMsg
		if failMsg == "" {
			failMsg = "Sync failed"
		}
		if errors.Is(err, context.DeadlineExceeded) {
			failMsg = "Sync timed out"
		}
		if errors.Is(err, ble.ErrBondLost) {
			// The camera dropped its side of the bond (HERO13 without a
			// completed pairing finish); connects abort until both sides
			// forget it and the camera is paired again.
			failMsg = "Pairing lost, put the camera in pairing mode to pair again"
			c.log.Warn("BLE bond lost, clearing pairing", camera.LogAttrs()...)
			if forgetErr := c.ble.ForgetDevice(bleAddress); forgetErr != nil {
				c.log.Warn("Failed to remove stale bond", append(camera.LogAttrs(), "err", forgetErr)...)
			}
			_ = c.db.SetCameraPairedByID(task.CameraID, false)
		}
		c.updateProgress(task, failMsg, step.percent)
		c.log.Error("Sync failed", append(camera.LogAttrs(), "step", step.label, "err", err)...)
		_ = c.db.SetLastSyncErrorByID(task.CameraID, failMsg)
		run.session.FailedStep = step.label
		run.finish(model.SyncOutcomeFailed, failMsg)
		return
	}

	// Finalize - defer handles IsSyncing=false, queue cleanup, and notification.
	// A selection download is partial by definition: the camera is not
	// "synced", so the auto-sync eligibility must stay untouched.
	if len(task.FileNames) == 0 {
		_ = c.db.MarkCameraSyncedByID(task.CameraID)
		c.storeSyncedBaseline(run)
	}
	_ = c.db.SetLastSyncErrorByID(task.CameraID, "")
	// Streamed before the defer removes the entry; the frontend turns the
	// label into a success toast, like the up-to-date skip above.
	c.updateProgress(task, "Sync complete", 100)
	c.log.Info("Camera synced", append(camera.LogAttrs(), "files", run.downloaded)...)
	run.finish(model.SyncOutcomeComplete, "")
}

// startPhase closes the running phase timing and opens the next one when
// the step crosses a phase boundary; mirrored into the queue entry so the
// UI shows the stepper live.
func (c *Coordinator) startPhase(r *syncRun, phase model.SyncPhase) {
	now := time.Now()
	if n := len(r.phases); n > 0 {
		if r.phases[n-1].Phase == phase {
			return
		}
		r.phases[n-1].FinishedAt = now
	}
	r.phases = append(r.phases, model.SyncPhaseTiming{Phase: phase, StartedAt: now})
	phases := append([]model.SyncPhaseTiming(nil), r.phases...)
	_, _ = c.db.MutateSyncQueueEntry(r.task.CameraID, func(entry *model.SyncQueueEntry) {
		entry.Phase = phase
		entry.Phases = phases
	})
}

// recordSession closes the timings and persists the history record. A
// run that never reached finish (task cancelled between steps, or the
// camera vanished) counts as cancelled.
func (c *Coordinator) recordSession(r *syncRun) {
	s := r.session
	now := time.Now()
	if n := len(r.phases); n > 0 && r.phases[n-1].FinishedAt.IsZero() {
		r.phases[n-1].FinishedAt = now
	}
	if s.Outcome == "" {
		s.Outcome = model.SyncOutcomeCancelled
	}
	s.FinishedAt = now
	s.Files = r.files
	s.Phases = r.phases
	s.CountFiles()
	if err := c.db.AddSyncSession(s); err != nil {
		c.log.Warn("Sync history not saved", "camera", r.task.CameraName, "err", err)
	}
	c.log.Info("Sync recorded", "camera", r.task.CameraName, "outcome", s.Outcome,
		"downloaded", s.FilesDownloaded, "failed", s.FilesFailed, "skipped", s.FilesSkipped,
		"elapsed", now.Sub(s.StartedAt).Round(time.Second))
}

// stepConnectBLE connects, arranges disconnect, and starts the keep-alive.
func (c *Coordinator) stepConnectBLE(r *syncRun) error {
	if err := c.bleOperation(r.ctx, func() error {
		return c.ble.Connect(r.bleAddress)
	}); err != nil {
		return err
	}

	r.cleanup(func() {
		// Fresh ctx: teardown must run even after cancel or timeout,
		// otherwise the camera stays awake with the AP up.
		ctx, cancel := context.WithTimeout(context.Background(), teardownTimeout)
		defer cancel()
		if err := c.bleOperation(ctx, func() error {
			return c.ble.Disconnect(r.bleAddress)
		}); err != nil {
			c.log.Warn("Failed to disconnect from BLE after sync", "err", err)
		}
	})

	// Keep camera awake during the entire sync
	kaCtx, kaCancel := context.WithCancel(r.ctx)
	r.cleanup(kaCancel)
	go func() {
		ticker := time.NewTicker(keepAliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-kaCtx.Done():
				return
			case <-ticker.C:
				if err := c.ble.KeepAlive(r.bleAddress); err != nil {
					c.log.Debug("Keep-alive failed", "camera", r.task.CameraName, "err", err)
				}
			}
		}
	}()
	return nil
}

// stepSkipIfClean ends a full sync before the WiFi cycle when BLE proves
// nothing changed on the camera and the local library is complete.
// Any doubt (missing status, sentinel value, failed last sync, absent
// catalog, undownloaded file in the window) falls through to the full
// cycle: the WiFi dance is also the repair path, so only a provable
// no-op may skip it.
func (c *Coordinator) stepSkipIfClean(r *syncRun) error {
	// Explicit requests always get the WiFi cycle
	if r.task.CatalogOnly || len(r.task.FileNames) > 0 {
		return nil
	}
	if r.camera.Status.LastSyncError != "" || r.camera.Status.LastSynced.IsZero() {
		return nil
	}
	md := r.camera.Metadata
	if md.SyncedSpaceKB <= 0 {
		return nil // no baseline captured at a completed sync yet
	}
	if r.config.DestinationFolder == "" || r.camera.Camera.WiFiSSID == "" {
		return nil
	}

	items, updatedAt, err := ReadCatalog(filepath.Join(r.config.DestinationFolder, r.camera.Camera.WiFiSSID))
	if err != nil || updatedAt.IsZero() {
		return nil
	}
	days := int(r.config.DaysThreshold)
	if days <= 0 {
		days = 7 // mirror DownloadVideos's default window
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, item := range items {
		if item.CreatedAt.After(cutoff) && !item.Downloaded {
			return nil
		}
	}

	statuses, err := c.ble.QueryStatuses(r.bleAddress, []byte{
		ble.StatusNumTotalPhotos, ble.StatusNumTotalVideos, ble.StatusSDCardRemainingKB,
	})
	if err != nil {
		return nil
	}
	photos, photosOK := statusValue(statuses, ble.StatusNumTotalPhotos)
	videos, videosOK := statusValue(statuses, ble.StatusNumTotalVideos)
	space, spaceOK := statusValue64(statuses, ble.StatusSDCardRemainingKB)
	if !photosOK || !videosOK || !spaceOK {
		return nil
	}
	// Exact match against the last-sync snapshot, free space included: any
	// new file on the card changes free KB, so equality means the card is
	// untouched since the last completed sync.
	if photos != md.SyncedNumPhotos || videos != md.SyncedNumVideos || space != md.SyncedSpaceKB {
		return nil
	}
	return errUpToDate
}

// storeSyncedBaseline snapshots the camera-side state the library now
// mirrors, over the still-open BLE connection. Sentinel or missing values
// (HERO13 shortly after wake) leave no baseline, so the next sync does the
// full verify instead of a wrong skip.
func (c *Coordinator) storeSyncedBaseline(r *syncRun) {
	statuses, err := c.ble.QueryStatuses(r.bleAddress, []byte{
		ble.StatusNumTotalPhotos, ble.StatusNumTotalVideos, ble.StatusSDCardRemainingKB,
	})
	if err != nil {
		c.log.Debug("Baseline query failed, next sync will verify fully", "camera", r.task.CameraName, "err", err)
		return
	}
	photos, photosOK := statusValue(statuses, ble.StatusNumTotalPhotos)
	videos, videosOK := statusValue(statuses, ble.StatusNumTotalVideos)
	space, spaceOK := statusValue64(statuses, ble.StatusSDCardRemainingKB)
	if !photosOK || !videosOK || !spaceOK || space <= 0 {
		c.log.Debug("Baseline unavailable, next sync will verify fully", "camera", r.task.CameraName)
		return
	}
	_ = c.db.UpdateCameraByID(r.task.CameraID, func(cs *model.CameraWithState) {
		cs.Metadata.SyncedNumPhotos = photos
		cs.Metadata.SyncedNumVideos = videos
		cs.Metadata.SyncedSpaceKB = space
	})
}

// statusValue returns a status as int32, false when absent or invalid
// (HERO13+ post-wake sentinels parse as -1).
func statusValue(statuses map[byte][]byte, id byte) (int32, bool) {
	v, ok := statuses[id]
	if !ok {
		return 0, false
	}
	n := ble.ParseIntStatus(v)
	return n, n >= 0
}

func statusValue64(statuses map[byte][]byte, id byte) (int64, bool) {
	v, ok := statuses[id]
	if !ok {
		return 0, false
	}
	n := ble.ParseInt64Status(v)
	return n, n >= 0
}

// stepWaitIdle waits until the camera is neither busy nor encoding.
func (c *Coordinator) stepWaitIdle(r *syncRun) error {
	deadline := time.Now().Add(idleWaitTimeout)
	for {
		statuses, err := c.ble.QueryStatuses(r.bleAddress, []byte{ble.StatusSystemBusy, ble.StatusEncoding})
		if err != nil {
			return fmt.Errorf("busy query failed: %w", err)
		}
		if !ble.InUse(statuses) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("camera still busy after %v", idleWaitTimeout)
		}
		logging.Trace(c.log, "Camera busy or encoding, waiting", "ble_addr", r.bleAddress)
		select {
		case <-r.ctx.Done():
			return r.ctx.Err()
		case <-time.After(idleWaitPoll):
		}
	}
}

// stepWaitAP waits for the camera's AP to actually be up (status 69); the
// Set AP Control response only acknowledges the request. On timeout, bounce
// the AP once (spec FAQ recovery for intermittent AP failures), then proceed
// either way and let the join retries have their chance. Never fails the sync.
func (c *Coordinator) stepWaitAP(r *syncRun) error {
	if err := c.ble.WaitForWiFiAPReady(r.bleAddress, apReadyTimeout); err != nil {
		// Expected recovery step; only a failing bounce is Warn-worthy
		c.log.Debug("Camera AP not ready, bouncing AP", "camera", r.task.CameraName, "err", err)
		if bounceErr := c.ble.SetAPControl(r.bleAddress, ble.WiFiAPModeBounce); bounceErr != nil {
			c.log.Warn("AP bounce failed", "camera", r.task.CameraName, "err", bounceErr)
		} else if err := c.ble.WaitForWiFiAPReady(r.bleAddress, apReadyTimeout); err != nil {
			c.log.Warn("Camera AP still not ready after bounce", "camera", r.task.CameraName, "err", err)
		}
	}
	return nil
}

// stepConnectWiFi joins the camera's AP and arranges the WiFi disconnect.
func (c *Coordinator) stepConnectWiFi(r *syncRun) error {
	r.wifi = c.wifiFactory()

	wifiCtx, cancel := context.WithTimeout(r.ctx, time.Duration(r.config.ConnectTimeoutSeconds)*time.Second)
	defer cancel()

	if err := r.wifi.Connect(wifiCtx, r.camera.Camera.WiFiSSID, r.camera.Camera.WiFiPassword); err != nil {
		return err
	}
	c.log.Info("WiFi connected to camera", "camera", r.camera.Camera.Name, "ssid", r.camera.Camera.WiFiSSID)

	r.cleanup(func() {
		if err := r.wifi.Disconnect(); err != nil {
			c.log.Warn("Failed to disconnect from WiFi", "camera", r.task.CameraName, "err", err)
		}
	})
	return nil
}

// stepAwaitAPI polls until the GoPro HTTP API answers.
func (c *Coordinator) stepAwaitAPI(r *syncRun) error {
	for attempt := 1; ; attempt++ {
		_, err := r.wifi.GetCameraStatus(r.ctx)
		if err == nil {
			c.log.Debug("GoPro HTTP API reachable", "camera", r.task.CameraName, "attempt", attempt)
			return nil
		}
		if attempt == apiReadyAttempts {
			return fmt.Errorf("GoPro API not reachable after %d attempts: %w", attempt, err)
		}
		c.log.Debug("GoPro API not ready", "camera", r.task.CameraName, "attempt", attempt, "max", apiReadyAttempts, "err", err)
		select {
		case <-r.ctx.Done():
			return r.ctx.Err()
		case <-time.After(apiReadyDelay):
		}
	}
}

// stepPrepareFolders creates the destination folders and the download ctx.
func (c *Coordinator) stepPrepareFolders(r *syncRun) error {
	if err := os.MkdirAll(r.config.DestinationFolder, 0755); err != nil {
		return fmt.Errorf("cannot create destination folder %s: %w", r.config.DestinationFolder, err)
	}
	r.folder = filepath.Join(r.config.DestinationFolder, r.camera.Camera.WiFiSSID)
	if err := os.MkdirAll(r.folder, 0755); err != nil {
		return fmt.Errorf("cannot create camera folder %s: %w", r.folder, err)
	}

	dctx, dcancel := context.WithTimeout(r.ctx, downloadTimeout)
	r.downloadCtx = dctx
	r.cleanup(dcancel)
	return nil
}

// stepCatalog captures the media catalog + thumbnails while the WiFi link
// is up; the media browser reads them offline later. Best effort, except
// that a catalog-only task ends the sync here.
func (c *Coordinator) stepCatalog(r *syncRun) error {
	if mediaFiles, listErr := r.wifi.ListMedia(r.downloadCtx); listErr != nil {
		c.log.Warn("Media catalog refresh failed", "camera", r.task.CameraName, "err", listErr)
	} else {
		if err := WriteCatalog(r.folder, mediaFiles); err != nil {
			c.log.Warn("Media catalog write failed", "camera", r.task.CameraName, "err", err)
		}
		refreshThumbnails(r.downloadCtx, r.wifi, r.folder, mediaFiles, c.log)
	}

	if r.task.CatalogOnly {
		return errCatalogOnly
	}
	return nil
}

// stepDownload toggles turbo when configured and downloads the media.
func (c *Coordinator) stepDownload(r *syncRun) error {
	// Turbo Transfer is opt-in: it is tuned for the GoPro app's parallel
	// chunked downloads and measured slower for our single sequential
	// stream. `ble-probe validate -wifi -speed` measures both on hardware.
	if r.config.TurboEnabled {
		if err := r.wifi.SetTurboTransfer(r.downloadCtx, true); err != nil {
			c.log.Debug("Turbo transfer not enabled", "camera", r.task.CameraName, "err", err)
		} else {
			r.cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), teardownTimeout)
				defer cancel()
				if err := r.wifi.SetTurboTransfer(ctx, false); err != nil {
					c.log.Debug("Turbo transfer not disabled", "camera", r.task.CameraName, "err", err)
				}
			})
		}
	}

	downloadedFiles, err := r.wifi.DownloadVideos(r.downloadCtx, r.folder, int(r.config.DaysThreshold), r.task.FileNames,
		func(p wifi.DownloadProgress) { c.updateDownloadProgress(r, p) })
	if err != nil {
		return err
	}
	r.downloaded = len(downloadedFiles)
	return nil
}

// toSyncFiles converts a transfer snapshot for the queue entry and history.
func toSyncFiles(files []wifi.FileStatus) []model.SyncFile {
	if files == nil {
		return nil
	}
	out := make([]model.SyncFile, len(files))
	for i, f := range files {
		out[i] = model.SyncFile{
			Name:       f.Name,
			CameraPath: f.CameraPath,
			SizeBytes:  f.Size,
			State:      model.SyncFileState(f.State),
			BytesDone:  f.BytesDone,
			Error:      f.Err,
			LocalPath:  f.LocalPath,
			DurationMs: f.Duration.Milliseconds(),
		}
	}
	return out
}

// updateDownloadProgress streams byte-level download state into the queue
// entry. Bytes map onto the 60..99 tail of the step scale; file counts
// carry it when the camera reported no sizes. The file list is kept on
// the run too, for the history record.
func (c *Coordinator) updateDownloadProgress(r *syncRun, p wifi.DownloadProgress) {
	task := r.task
	if p.Files != nil {
		r.files = toSyncFiles(p.Files)
	}
	if task.Ctx != nil && task.Ctx.Err() != nil {
		return
	}
	percent := int32(60)
	switch {
	case p.BytesTotal > 0:
		percent += int32(float64(p.BytesDone) / float64(p.BytesTotal) * 39)
	case p.FileCount > 0 && p.FileIndex > 0:
		percent += int32((p.FileIndex - 1) * 39 / p.FileCount)
	}
	if percent > 99 {
		percent = 99
	}
	operation := "Downloading media"
	if p.FileName != "" {
		operation = "Downloading " + p.FileName
	}
	files := r.files
	_, _ = c.db.MutateSyncQueueEntry(task.CameraID, func(entry *model.SyncQueueEntry) {
		entry.CurrentOperation = operation
		entry.ProgressPercent = percent
		entry.FileIndex = int32(p.FileIndex)
		entry.FileCount = int32(p.FileCount)
		entry.FileName = p.FileName
		entry.FileBytes = p.FileBytes
		entry.FileTotal = p.FileTotal
		entry.BytesDone = p.BytesDone
		entry.BytesTotal = p.BytesTotal
		entry.RateBps = p.RateBps
		if files != nil {
			entry.Files = files
		}
	})
	c.notifier()
}

// updateProgress mutates only the progress fields of the queue entry.
// Overwriting the whole entry destroyed FileNames/Priority/QueuedAt and made
// a later selection request merge into an empty entry and get dropped.
func (c *Coordinator) updateProgress(task *SyncTask, operation string, percent int32) {
	if task.Ctx != nil && task.Ctx.Err() != nil {
		// Cancelled: the queue entry may already belong to a new request
		c.log.Debug("Skipping sync queue update for cancelled task", "camera_id", task.CameraID)
		return
	}
	_, _ = c.db.MutateSyncQueueEntry(task.CameraID, func(entry *model.SyncQueueEntry) {
		entry.CurrentOperation = operation
		entry.ProgressPercent = percent
		// Byte detail belongs to the download step only; a step-level
		// update means that step is over, so stale bytes must not linger.
		// The file list stays: the final label ("Sync complete") is
		// streamed with it so the UI can show what arrived.
		entry.FileIndex = 0
		entry.FileCount = 0
		entry.FileName = ""
		entry.FileBytes = 0
		entry.FileTotal = 0
		entry.BytesDone = 0
		entry.BytesTotal = 0
		entry.RateBps = 0
	})
	c.notifier()
	c.log.Debug("Sync progress", "camera", task.CameraName, "operation", operation, "percent", percent)
}
