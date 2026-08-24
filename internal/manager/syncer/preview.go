package syncer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/wifi"
)

const (
	// How long an idle preview session keeps the camera link up so the
	// next preview click skips the ~25s connection dance.
	previewLinger = 45 * time.Second
	previewPoll   = 2 * time.Second
)

// previewSession accepts follow-up preview requests while its camera
// link is up.
type previewSession struct {
	requests chan string // camera paths
}

// RequestPreview fetches a clip's LRV proxy into the preview cache.
// An open session takes the request immediately; otherwise a new session
// claims the radio like a sync does.
func (c *Coordinator) RequestPreview(cameraID, cameraPath string) error {
	camera, found := c.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}
	if !camera.Status.IsManaged || !camera.Status.IsPaired {
		return fmt.Errorf("%w: %s", model.ErrNotManagedPaired, camera.Camera.Name)
	}
	if _, ok := wifi.LRVCameraPath(cameraPath); !ok {
		return fmt.Errorf("no preview available for %s", cameraPath)
	}

	c.mutex.Lock()
	if s, ok := c.previewSessions[cameraID]; ok {
		select {
		case s.requests <- cameraPath:
		default:
			c.mutex.Unlock()
			return fmt.Errorf("preview queue full for %s", camera.Camera.Name)
		}
		c.mutex.Unlock()
		return nil
	}
	if _, busy := c.activeTasks[cameraID]; busy {
		c.mutex.Unlock()
		return fmt.Errorf("%s is busy syncing", camera.Camera.Name)
	}

	taskCtx, taskCancel := context.WithCancel(c.ctx)
	task := &SyncTask{
		CameraID:   cameraID,
		BLEAddress: camera.Camera.BLEAddress,
		CameraName: camera.Camera.Name,
		Cancel:     taskCancel,
		Ctx:        taskCtx,
	}
	s := &previewSession{requests: make(chan string, 16)}
	s.requests <- cameraPath
	c.activeTasks[cameraID] = task
	c.previewSessions[cameraID] = s
	c.mutex.Unlock()

	// Previews are interactive; never queue behind a long sync.
	select {
	case c.syncSem <- struct{}{}:
	default:
		c.mutex.Lock()
		delete(c.activeTasks, cameraID)
		delete(c.previewSessions, cameraID)
		c.mutex.Unlock()
		task.Cancel()
		return fmt.Errorf("another sync is using the radio")
	}

	_ = c.db.UpdateCameraSyncingStatusByID(cameraID, true)
	_ = c.db.AddSyncQueueEntry(&model.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		Priority:         SyncPriorityManual,
		CurrentOperation: "Connecting for preview",
	})
	c.notifier()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.performPreviewSession(task, s)
	}()
	return nil
}

func (c *Coordinator) performPreviewSession(task *SyncTask, s *previewSession) {
	defer func() {
		<-c.syncSem
		c.mutex.Lock()
		delete(c.activeTasks, task.CameraID)
		delete(c.previewSessions, task.CameraID)
		c.mutex.Unlock()
		_ = c.db.UpdateCameraSyncingStatusByID(task.CameraID, false)
		_ = c.db.RemoveSyncQueueEntry(task.CameraID)
		c.notifier()
	}()

	camera, found := c.db.GetCameraByID(task.CameraID)
	if !found {
		return
	}
	c.log.Info("Preview session started", camera.LogAttrs()...)

	syncCtx, cancel := context.WithTimeout(task.Ctx, syncOverallTimeout)
	defer cancel()
	run := &syncRun{
		task:       task,
		camera:     camera,
		bleAddress: camera.Camera.BLEAddress,
		config:     c.db.GetConfig(),
		ctx:        syncCtx,
	}
	defer func() {
		for i := len(run.cleanups) - 1; i >= 0; i-- {
			run.cleanups[i]()
		}
	}()

	release, err := c.ble.AcquireSession(run.bleAddress, 30*time.Second)
	if err != nil {
		c.log.Warn("Preview could not acquire BLE session", "camera", task.CameraName, "err", err)
		return
	}
	defer release()

	steps := []syncStep{
		{"Connecting via BLE", 20, "", c.stepConnectBLE},
		{"Waiting for camera WiFi AP", 30, "", c.stepWaitAP},
		{"Connecting to WiFi", 40, "", c.stepConnectWiFi},
		{"Waiting for GoPro API", 50, "", c.stepAwaitAPI},
		{"Preparing download", 60, "", c.stepPrepareFolders},
	}
	for _, step := range steps {
		c.updateProgress(task, step.label, step.percent)
		if err := step.run(run); err != nil {
			c.log.Warn("Preview session failed", "camera", task.CameraName, "step", step.label, "err", err)
			c.updateProgress(task, "Preview connection failed", step.percent)
			time.Sleep(2 * time.Second) // let the stream deliver the label
			return
		}
	}

	previewDir := filepath.Join(run.folder, previewDirName)
	poll := previewPoll
	if c.previewIdle < poll {
		poll = c.previewIdle
	}
	idle := time.Now()
	for {
		select {
		case <-run.ctx.Done():
			return
		case cameraPath := <-s.requests:
			c.fetchPreview(run, previewDir, cameraPath)
			idle = time.Now()
		case <-time.After(poll):
			if time.Since(idle) > c.previewIdle {
				c.log.Info("Preview session idle, closing", "camera", task.CameraName)
				return
			}
			// Hand the radio over as soon as real work is queued
			if len(c.db.GetSyncQueue()) > 1 {
				c.log.Info("Preview session yields to queued sync", "camera", task.CameraName)
				return
			}
			c.updateProgress(task, "Preview session active", 100)
		}
	}
}

// fetchPreview downloads one LRV into the cache and streams "Preview
// ready", which the browser uses as its reload-and-open signal.
func (c *Coordinator) fetchPreview(run *syncRun, previewDir, cameraPath string) {
	c.updateProgress(run.task, "Fetching preview", 80)
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		c.log.Warn("Cannot create preview dir", "dir", previewDir, "err", err)
		c.updateProgress(run.task, "Preview failed", 100)
		return
	}
	local := c.localNameFor(run, cameraPath)
	out := PreviewPath(run.folder, local)
	if err := run.wifi.DownloadLRV(run.ctx, cameraPath, out); err != nil {
		c.log.Warn("Preview fetch failed", "camera", run.task.CameraName, "path", cameraPath, "err", err)
		c.updateProgress(run.task, "Preview failed", 100)
		return
	}
	c.log.Info("Preview cached", "camera", run.task.CameraName, "path", cameraPath)
	c.updateProgress(run.task, "Preview ready", 100)
}

// localNameFor resolves a camera path to its local media name via the
// cached catalog, falling back to the bare file name.
func (c *Coordinator) localNameFor(run *syncRun, cameraPath string) string {
	name := filepath.Base(cameraPath)
	items, _, err := ReadCatalog(run.folder)
	if err != nil {
		return name
	}
	counts := make(map[string]int, len(items))
	for _, it := range items {
		counts[it.Name]++
	}
	for _, it := range items {
		if it.CameraPath == cameraPath {
			return wifi.LocalMediaName(it.CameraPath, it.Name, counts[it.Name] > 1)
		}
	}
	return name
}
