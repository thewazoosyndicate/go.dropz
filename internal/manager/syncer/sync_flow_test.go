package syncer

import (
	"context"
	"path/filepath"
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
}

func (f *fakeBLE) AcquireSession(string, time.Duration) (func(), error) { return func() {}, nil }
func (f *fakeBLE) Connect(string) error                                 { f.connects.Add(1); return nil }
func (f *fakeBLE) Disconnect(string) error                              { f.disconnects.Add(1); return nil }
func (f *fakeBLE) KeepAlive(string) error                               { return nil }
func (f *fakeBLE) QueryStatuses(string, []byte) (map[byte][]byte, error) {
	// idle: busy=0, encoding=0
	return map[byte][]byte{ble.StatusSystemBusy: {0}, ble.StatusEncoding: {0}}, nil
}
func (f *fakeBLE) WaitForWiFiAPReady(string, time.Duration) error { return nil }
func (f *fakeBLE) SetAPControl(string, ble.WiFiAPMode) error      { return nil }

// fakeWiFi serves one media file and records download selections.
type fakeWiFi struct {
	downloadedNames []string
	disconnects     atomic.Int32
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
func (f *fakeWiFi) DownloadVideos(_ context.Context, _ string, _ int, fileNames []string) ([]string, error) {
	f.downloadedNames = fileNames
	return []string{"GX010001.MP4"}, nil
}
func (f *fakeWiFi) DownloadThumbnail(context.Context, string, string) error { return nil }

func newFlowCoordinator(t *testing.T) (*Coordinator, *fakeBLE, *fakeWiFi, *SyncTask) {
	t.Helper()
	c, st := newTestCoordinator(t)
	fb := &fakeBLE{}
	fw := &fakeWiFi{}
	c.ble = fb
	c.wifiFactory = func() wifiClient { return fw }
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
