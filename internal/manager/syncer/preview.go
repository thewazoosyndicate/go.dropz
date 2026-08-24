package syncer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

// ffmpegPath finds the bundled ffmpeg (shipped next to the dropz binary
// in the packaged app) or falls back to PATH.
func ffmpegPath() string {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "ffmpeg")
		if _, statErr := os.Stat(p); statErr == nil {
			return p
		}
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	return ""
}

// transcodePreview turns a camera clip into a 480p VP9/Opus WebM.
// Chromium decodes VP9 in software everywhere; the camera's HEVC does
// not render at all on hosts whose Mesa lacks HEVC VAAPI.
func transcodePreview(ctx context.Context, src, dst string) error {
	ff := ffmpegPath()
	if ff == "" {
		return errors.New("ffmpeg not found next to the binary or in PATH")
	}
	tmp := dst + ".partial"
	cmd := exec.CommandContext(ctx, ff, "-y", "-v", "error", "-i", src,
		"-vf", "scale=-2:480", "-c:v", "libvpx-vp9", "-row-mt", "1",
		"-deadline", "realtime", "-cpu-used", "7", "-crf", "34", "-b:v", "0",
		"-c:a", "libopus", "-b:a", "64k", "-f", "webm", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg: %w: %s", err, out)
	}
	return os.Rename(tmp, dst)
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

	// A local source needs no radio: transcode straight from disk.
	config := c.db.GetConfig()
	folder := filepath.Join(config.DestinationFolder, camera.Camera.WiFiSSID)
	local := c.localNameFor(folder, cameraPath)
	if src := filepath.Join(folder, local); fileExists(src) {
		taskCtx, taskCancel := context.WithCancel(c.ctx)
		task := &SyncTask{CameraID: cameraID, CameraName: camera.Camera.Name, Cancel: taskCancel, Ctx: taskCtx}
		c.activeTasks[cameraID] = task
		c.mutex.Unlock()
		_ = c.db.AddSyncQueueEntry(&model.SyncQueueEntry{
			CameraID:         cameraID,
			QueuedAt:         time.Now(),
			Priority:         SyncPriorityManual,
			CurrentOperation: "Generating preview",
		})
		c.notifier()
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.performLocalPreview(task, src, folder, local)
		}()
		return nil
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

// fetchPreview downloads one LRV, transcodes it into the always-playable
// WebM cache, and streams "Preview ready", which the browser uses as its
// reload-and-open signal.
func (c *Coordinator) fetchPreview(run *syncRun, previewDir, cameraPath string) {
	c.updateProgress(run.task, "Fetching preview", 80)
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		c.log.Warn("Cannot create preview dir", "dir", previewDir, "err", err)
		c.updateProgress(run.task, "Preview failed", 100)
		return
	}
	local := c.localNameFor(run.folder, cameraPath)
	raw := rawLRVPath(run.folder, local)
	if err := run.wifi.DownloadLRV(run.ctx, cameraPath, raw); err != nil {
		c.log.Warn("Preview fetch failed", "camera", run.task.CameraName, "path", cameraPath, "err", err)
		c.updateProgress(run.task, "Preview failed", 100)
		return
	}
	c.updateProgress(run.task, "Converting preview", 90)
	out := PreviewPath(run.folder, local)
	err := c.transcode(run.ctx, raw, out)
	_ = os.Remove(raw)
	if err != nil {
		c.log.Warn("Preview transcode failed", "camera", run.task.CameraName, "path", cameraPath, "err", err)
		c.updateProgress(run.task, "Preview failed", 100)
		return
	}
	c.log.Info("Preview cached", "camera", run.task.CameraName, "path", cameraPath)
	c.updateProgress(run.task, "Preview ready", 100)
}

// performLocalPreview transcodes an already-downloaded clip; no radio.
func (c *Coordinator) performLocalPreview(task *SyncTask, src, folder, local string) {
	defer func() {
		c.mutex.Lock()
		delete(c.activeTasks, task.CameraID)
		c.mutex.Unlock()
		_ = c.db.RemoveSyncQueueEntry(task.CameraID)
		c.notifier()
	}()

	ctx, cancel := context.WithTimeout(task.Ctx, 10*time.Minute)
	defer cancel()
	out := PreviewPath(folder, local)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		c.log.Warn("Cannot create preview dir", "err", err)
		return
	}
	if err := c.transcode(ctx, src, out); err != nil {
		c.log.Warn("Local preview transcode failed", "camera", task.CameraName, "src", src, "err", err)
		c.updateProgress(task, "Preview failed", 100)
		return
	}
	c.log.Info("Preview generated from local file", "camera", task.CameraName, "src", src)
	// The label streams before the defer removes the entry; either signal
	// (label or entry removal) makes the browser reload and open it.
	c.updateProgress(task, "Preview ready", 100)
}

// localNameFor resolves a camera path to its local media name via the
// cached catalog, falling back to the bare file name.
func (c *Coordinator) localNameFor(folder, cameraPath string) string {
	name := filepath.Base(cameraPath)
	items, _, err := ReadCatalog(folder)
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
