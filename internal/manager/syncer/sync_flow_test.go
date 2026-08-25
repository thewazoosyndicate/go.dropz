package syncer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/wifi"
)

// fakeBLE answers every call successfully and counts connects/disconnects.
type fakeBLE struct {
	connects    atomic.Int32
	disconnects atomic.Int32
	forgets     atomic.Int32
	// set to fail Connect with this error
	connectErr error
	// merged into QueryStatuses responses (busy/encoding stay idle)
	statuses map[byte][]byte
}

func (f *fakeBLE) AcquireSession(string, time.Duration) (func(), error) { return func() {}, nil }
func (f *fakeBLE) Connect(string) error {
	f.connects.Add(1)
	return f.connectErr
}
func (f *fakeBLE) Disconnect(string) error { f.disconnects.Add(1); return nil }
func (f *fakeBLE) KeepAlive(string) error  { return nil }
func (f *fakeBLE) QueryStatuses(string, []byte) (map[byte][]byte, error) {
	// idle: busy=0, encoding=0
	out := map[byte][]byte{ble.StatusSystemBusy: {0}, ble.StatusEncoding: {0}}
	for k, v := range f.statuses {
		out[k] = v
	}
	return out, nil
}
func (f *fakeBLE) WaitForWiFiAPReady(string, time.Duration) error { return nil }
func (f *fakeBLE) ForgetDevice(string) error                      { f.forgets.Add(1); return nil }
func (f *fakeBLE) SetAPControl(string, ble.WiFiAPMode) error      { return nil }

// fakeWiFi serves one media file and records download selections.
type fakeWiFi struct {
	downloadedNames []string
	disconnects     atomic.Int32
	factoryCalls    atomic.Int32
	// non-zero FileCount: reported to the progress callback once
	progress wifi.DownloadProgress
}

func (f *fakeWiFi) Connect(context.Context, string, string) error { return nil }
func (f *fakeWiFi) Disconnect() error                             { f.disconnects.Add(1); return nil }
func (f *fakeWiFi) GetCameraStatus(context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (f *fakeWiFi) ListMedia(context.Context) ([]wifi.MediaFile, error) {
	return []wifi.MediaFile{{Name: "GX010001.MP4", CameraPath: "100GOPRO/GX010001.MP4", Size: 42, CreatedAt: time.Now()}}, nil
}
func (f *fakeWiFi) SetTurboTransfer(context.Context, bool) error { return nil }
func (f *fakeWiFi) DownloadVideos(_ context.Context, destDir string, _ int, fileNames []string, progress wifi.ProgressFunc) ([]string, error) {
	f.downloadedNames = fileNames
	if progress != nil && f.progress.FileCount > 0 {
		progress(f.progress)
	}
	// The skip guard's catalog check stats this file; a fake download
	// must leave it on disk like the real one does.
	_ = os.WriteFile(filepath.Join(destDir, "GX010001.MP4"), []byte("x"), 0644)
	return []string{"GX010001.MP4"}, nil
}
func (f *fakeWiFi) DownloadThumbnail(context.Context, string, string) error { return nil }
func (f *fakeWiFi) DownloadLRV(_ context.Context, _, outPath string) error {
	return os.WriteFile(outPath, []byte("lrv"), 0644)
}

func newFlowCoordinator(t *testing.T) (*Coordinator, *fakeBLE, *fakeWiFi, *SyncTask) {
	t.Helper()
	c, st := newTestCoordinator(t)
	fb := &fakeBLE{}
	fw := &fakeWiFi{}
	c.ble = fb
	c.wifiFactory = func() wifiClient { fw.factoryCalls.Add(1); return fw }
	c.bleOperation = func(ctx context.Context, op func() error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return op()
	}

	cfg := st.GetConfig()
	cfg.DestinationFolder = filepath.Join(t.TempDir(), "library")
	cfg.SyncEnabled = true
	if err := st.UpdateConfig(cfg); err != nil {
		t.Fatal(err)
	}

	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: true, IsPaired: true, IsReachable: true})
	_ = st.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Camera.WiFiSSID = "GP12345678"
		cs.Camera.WiFiPassword = "secret"
	})
	if err := st.AddSyncQueueEntry(entry("cam1")); err != nil {
		t.Fatal(err)
	}
	task, ok := c.tryClaimCamera(entry("cam1"))
	if !ok {
		t.Fatal("claim failed")
	}
	return c, fb, fw, task
}

func TestPerformCameraSyncHappyPath(t *testing.T) {
	c, fb, fw, task := newFlowCoordinator(t)

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if fb.connects.Load() != 1 || fb.disconnects.Load() != 1 {
		t.Errorf("BLE connects=%d disconnects=%d, want 1/1", fb.connects.Load(), fb.disconnects.Load())
	}
	if fw.disconnects.Load() != 1 {
		t.Errorf("WiFi disconnects=%d, want 1", fw.disconnects.Load())
	}
	cs, _ := c.db.GetCameraByID("cam1")
	if cs.Status.IsSyncing {
		t.Error("IsSyncing not cleared")
	}
	if !cs.Status.IsSynced {
		t.Error("full sync must mark the camera synced")
	}
	if cs.Status.LastSyncError != "" {
		t.Errorf("unexpected sync error: %q", cs.Status.LastSyncError)
	}
	if len(c.db.GetSyncQueue()) != 0 {
		t.Error("queue entry not removed")
	}
}

func TestPerformCameraSyncSelectionStaysUnsynced(t *testing.T) {
	c, _, fw, task := newFlowCoordinator(t)
	task.FileNames = []string{"GX010001.MP4"}

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if len(fw.downloadedNames) != 1 || fw.downloadedNames[0] != "GX010001.MP4" {
		t.Errorf("selection not passed through, got %v", fw.downloadedNames)
	}
	cs, _ := c.db.GetCameraByID("cam1")
	if cs.Status.IsSynced {
		t.Error("selection download must not mark the camera synced")
	}
}

func TestPerformCameraSyncCancelledStillTearsDown(t *testing.T) {
	c, fb, _, task := newFlowCoordinator(t)
	task.Cancel()

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	// Connect is skipped (ctx already cancelled) so nothing to tear down,
	// and the failure must be recorded without wedging the task table.
	if fb.connects.Load() != 0 {
		t.Errorf("connect ran on a cancelled task")
	}
	c.mutex.RLock()
	_, active := c.activeTasks["cam1"]
	c.mutex.RUnlock()
	if active {
		t.Error("cancelled task still active")
	}
}

// seedCleanState makes cam1 a fully-synced camera whose BLE statuses match
// the stored baseline, with a complete catalog on disk.
func seedCleanState(t *testing.T, c *Coordinator, fb *fakeBLE) {
	t.Helper()
	_ = c.db.MarkCameraSyncedByID("cam1")
	_ = c.db.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Metadata.SyncedNumPhotos = 10
		cs.Metadata.SyncedNumVideos = 5
		cs.Metadata.SyncedSpaceKB = 1000 * 1024
	})
	fb.statuses = map[byte][]byte{
		ble.StatusNumTotalPhotos:    be32t(10),
		ble.StatusNumTotalVideos:    be32t(5),
		ble.StatusSDCardRemainingKB: be32t(1000 * 1024),
	}
	cfg := c.db.GetConfig()
	if err := WriteCatalog(filepath.Join(cfg.DestinationFolder, "GP12345678"), nil); err != nil {
		t.Fatal(err)
	}
}

func be32t(v uint32) []byte { return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)} }

func TestPerformCameraSyncSkipsWiFiWhenClean(t *testing.T) {
	c, fb, fw, task := newFlowCoordinator(t)
	seedCleanState(t, c, fb)

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if fw.factoryCalls.Load() != 0 {
		t.Error("clean camera must not get a WiFi cycle")
	}
	if fb.disconnects.Load() != 1 {
		t.Errorf("BLE disconnects=%d, want 1", fb.disconnects.Load())
	}
	cs, _ := c.db.GetCameraByID("cam1")
	if !cs.Status.IsSynced || cs.Status.LastSyncError != "" {
		t.Error("skip must record a clean synced state")
	}
	if len(c.db.GetSyncQueue()) != 0 {
		t.Error("queue entry not removed")
	}
}

func TestPerformCameraSyncRunsFullWhenSpaceChanged(t *testing.T) {
	c, fb, fw, task := newFlowCoordinator(t)
	seedCleanState(t, c, fb)
	// One new photo on the card: free space differs by a few KB
	fb.statuses[ble.StatusSDCardRemainingKB] = be32t(1000*1024 - 4)

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if fw.factoryCalls.Load() != 1 {
		t.Error("changed free space must force the WiFi cycle")
	}
}

func TestPerformCameraSyncSelectionNeverSkips(t *testing.T) {
	c, fb, fw, task := newFlowCoordinator(t)
	seedCleanState(t, c, fb)
	task.FileNames = []string{"GX010001.MP4"}

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if fw.factoryCalls.Load() != 1 {
		t.Error("explicit selection must always get the WiFi cycle")
	}
}

func TestPerformCameraSyncBondRejectedKeepsPairing(t *testing.T) {
	c, fb, fw, task := newFlowCoordinator(t)
	fb.connectErr = fmt.Errorf("3 consecutive aborted connects while advertising: %w", ble.ErrBondRejected)

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	// Forgetting here manufactured the HERO13 bond-store bloat; the bond
	// and the paired flag must both survive a rejected connect.
	if fb.forgets.Load() != 0 {
		t.Error("bond rejection must never forget the bond")
	}
	cs, _ := c.db.GetCameraByID("cam1")
	if !cs.Status.IsPaired {
		t.Error("bond rejection must keep the paired flag")
	}
	if !strings.Contains(cs.Status.LastSyncError, "not ready") {
		t.Errorf("lastSyncError = %q, want not-ready message", cs.Status.LastSyncError)
	}
	if fw.factoryCalls.Load() != 0 {
		t.Error("no WiFi cycle on a failed connect")
	}
}

// The bench-found bug: a status check detects new media and refreshes the
// live counts; the queued sync must still run the full cycle because the
// camera differs from the last-sync baseline, not from the live values.
func TestSyncDoesNotSkipAfterStatusCheckMovedLiveCounts(t *testing.T) {
	c, fb, fw, task := newFlowCoordinator(t)
	seedCleanState(t, c, fb)
	// Camera recorded one clip: camera reports 6 videos, and a status
	// check already stored 6 in the live metadata.
	fb.statuses[ble.StatusNumTotalVideos] = be32t(6)
	_ = c.db.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Metadata.NumVideos = 6
	})

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if fw.factoryCalls.Load() != 1 {
		t.Error("new media behind a refreshed live count must force the WiFi cycle")
	}
}

func TestDownloadProgressStreamsToQueueEntry(t *testing.T) {
	c, _, fw, task := newFlowCoordinator(t)
	fw.progress = wifi.DownloadProgress{
		FileIndex: 3, FileCount: 17, FileName: "GX010003.MP4",
		FileBytes: 5, FileTotal: 10,
		BytesDone: 100, BytesTotal: 400, RateBps: 42,
	}
	// The entry is gone once the sync finishes; snapshot it from the
	// notifier that fires on every progress write.
	var got *model.SyncQueueEntry
	c.notifier = func() {
		for _, e := range c.db.GetSyncQueue() {
			if e.FileName != "" {
				snapshot := *e
				got = &snapshot
			}
		}
	}

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if got == nil {
		t.Fatal("no queue entry carried download detail")
	}
	if got.FileIndex != 3 || got.FileCount != 17 || got.FileName != "GX010003.MP4" {
		t.Errorf("file detail = %d/%d %q", got.FileIndex, got.FileCount, got.FileName)
	}
	if got.BytesDone != 100 || got.BytesTotal != 400 || got.RateBps != 42 {
		t.Errorf("byte detail = %d/%d at %d", got.BytesDone, got.BytesTotal, got.RateBps)
	}
	// 60 + 39 * 100/400
	if got.ProgressPercent != 69 {
		t.Errorf("percent = %d, want 69", got.ProgressPercent)
	}
	if got.CurrentOperation != "Downloading GX010003.MP4" {
		t.Errorf("operation = %q", got.CurrentOperation)
	}
}

func TestSyncRecordsCompletedSession(t *testing.T) {
	c, _, _, task := newFlowCoordinator(t)

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	history := c.db.GetSyncHistory(0, "")
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	s := history[0]
	if s.Outcome != model.SyncOutcomeComplete || s.Error != "" || s.Selection {
		t.Errorf("session = %+v, want complete full sync", s)
	}
	if s.FinishedAt.Before(s.StartedAt) {
		t.Error("finished before started")
	}
	want := []model.SyncPhase{model.SyncPhaseConnect, model.SyncPhaseLink, model.SyncPhaseCatalog, model.SyncPhaseTransfer}
	if len(s.Phases) != len(want) {
		t.Fatalf("phases = %v, want %v", s.Phases, want)
	}
	for i, p := range s.Phases {
		if p.Phase != want[i] {
			t.Errorf("phase %d = %s, want %s", i, p.Phase, want[i])
		}
		if p.FinishedAt.IsZero() {
			t.Errorf("phase %s left open", p.Phase)
		}
	}
}

func TestSyncRecordsFailedSessionWithStep(t *testing.T) {
	c, fb, _, task := newFlowCoordinator(t)
	fb.connectErr = fmt.Errorf("radio off")

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	history := c.db.GetSyncHistory(0, "cam1")
	if len(history) != 1 {
		t.Fatalf("history len = %d, want 1", len(history))
	}
	s := history[0]
	if s.Outcome != model.SyncOutcomeFailed || s.Error != "BLE connection failed" {
		t.Errorf("outcome = %s %q, want failed with the step's user message", s.Outcome, s.Error)
	}
	if s.FailedStep != "Connecting via BLE" || s.StepIndex != 1 || s.StepCount != 9 {
		t.Errorf("failed step = %q at %d/%d", s.FailedStep, s.StepIndex, s.StepCount)
	}
}

func TestSyncRecordsUpToDateSession(t *testing.T) {
	c, fb, _, task := newFlowCoordinator(t)
	seedCleanState(t, c, fb)

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	history := c.db.GetSyncHistory(1, "")
	if len(history) != 1 || history[0].Outcome != model.SyncOutcomeUpToDate {
		t.Fatalf("history = %+v, want one up-to-date session", history)
	}
	// The BLE verify never reaches the link phase
	for _, p := range history[0].Phases {
		if p.Phase != model.SyncPhaseConnect {
			t.Errorf("unexpected phase %s on a skipped sync", p.Phase)
		}
	}
}

func TestCancelledSyncRecordsCancelled(t *testing.T) {
	c, _, _, task := newFlowCoordinator(t)
	task.Cancel()

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	history := c.db.GetSyncHistory(0, "")
	if len(history) != 1 || history[0].Outcome != model.SyncOutcomeCancelled {
		t.Fatalf("history = %+v, want one cancelled session", history)
	}
	if history[0].Error != "" {
		t.Errorf("a cancel is not a fault, got error %q", history[0].Error)
	}
}

func TestSyncFileStatesReachQueueEntryAndHistory(t *testing.T) {
	c, _, fw, task := newFlowCoordinator(t)
	fw.progress = wifi.DownloadProgress{
		FileIndex: 2, FileCount: 2, FileName: "GX010002.MP4",
		Files: []wifi.FileStatus{
			{Name: "GX010001.MP4", Size: 10, State: wifi.FileDone, LocalPath: "/lib/GX010001.MP4", Duration: 1500 * time.Millisecond},
			{Name: "GX010002.MP4", Size: 20, State: wifi.FileFailed, Err: "connection reset"},
			{Name: "GX010000.MP4", Size: 5, State: wifi.FileSkipped, LocalPath: "/lib/GX010000.MP4"},
		},
	}
	var streamed []model.SyncFile
	c.notifier = func() {
		for _, e := range c.db.GetSyncQueue() {
			if len(e.Files) > 0 {
				streamed = e.Files
			}
		}
	}

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task)

	if len(streamed) != 3 || streamed[0].State != model.SyncFileDone || streamed[1].Error != "connection reset" {
		t.Errorf("queue entry files = %+v", streamed)
	}
	s := c.db.GetSyncHistory(1, "")[0]
	if s.FilesDownloaded != 1 || s.FilesFailed != 1 || s.FilesSkipped != 1 || s.BytesDownloaded != 10 {
		t.Errorf("counts = %d/%d/%d %dB", s.FilesDownloaded, s.FilesFailed, s.FilesSkipped, s.BytesDownloaded)
	}
	if s.Files[0].DurationMs != 1500 || s.Files[0].LocalPath != "/lib/GX010001.MP4" {
		t.Errorf("file detail = %+v", s.Files[0])
	}
}

// A completed full sync snapshots the baseline; the next sync skips.
func TestSecondSyncSkipsAfterFullSync(t *testing.T) {
	c, fb, fw, task := newFlowCoordinator(t)
	fb.statuses = map[byte][]byte{
		ble.StatusNumTotalPhotos:    be32t(10),
		ble.StatusNumTotalVideos:    be32t(5),
		ble.StatusSDCardRemainingKB: be32t(1000 * 1024),
	}

	c.syncSem <- struct{}{}
	c.PerformCameraSync(task) // full: no baseline yet

	if fw.factoryCalls.Load() != 1 {
		t.Fatal("first sync must run the full cycle")
	}

	task2, ok := c.tryClaimCamera(entry("cam1"))
	if !ok {
		t.Fatal("second claim failed")
	}
	c.syncSem <- struct{}{}
	c.PerformCameraSync(task2)

	if fw.factoryCalls.Load() != 1 {
		t.Error("second sync must skip using the baseline stored by the first")
	}
}
