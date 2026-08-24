package syncer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/model"
)

func newPreviewCoordinator(t *testing.T) (*Coordinator, *fakeWiFi) {
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
	c.previewIdle = 50 * time.Millisecond
	// Stub transcoder: tests must not need ffmpeg
	c.transcode = func(_ context.Context, src, dst string) error {
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, append([]byte("webm:"), data...), 0644)
	}

	cfg := st.GetConfig()
	cfg.DestinationFolder = filepath.Join(t.TempDir(), "library")
	if err := st.UpdateConfig(cfg); err != nil {
		t.Fatal(err)
	}
	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: true, IsPaired: true, IsReachable: true})
	_ = st.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Camera.WiFiSSID = "GP12345678"
		cs.Camera.WiFiPassword = "secret"
	})
	return c, fw
}

func TestPreviewSessionFetchesLRVAndCleansUp(t *testing.T) {
	c, fw := newPreviewCoordinator(t)

	if err := c.RequestPreview("cam1", "100GOPRO/GX010001.MP4"); err != nil {
		t.Fatal(err)
	}
	c.Wait() // session ends after the shrunken linger window

	folder := filepath.Join(c.db.GetConfig().DestinationFolder, "GP12345678")
	preview := PreviewPath(folder, "GX010001.MP4")
	if _, err := os.Stat(preview); err != nil {
		t.Errorf("preview not cached at %s: %v", preview, err)
	}
	if fw.factoryCalls.Load() != 1 {
		t.Errorf("wifi sessions = %d, want 1", fw.factoryCalls.Load())
	}
	cs, _ := c.db.GetCameraByID("cam1")
	if cs.Status.IsSyncing {
		t.Error("IsSyncing not cleared after preview session")
	}
	if len(c.db.GetSyncQueue()) != 0 {
		t.Error("preview queue entry not removed")
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if len(c.activeTasks) != 0 || len(c.previewSessions) != 0 {
		t.Error("session maps not cleaned up")
	}
}

func TestPreviewFromLocalFileSkipsRadio(t *testing.T) {
	c, fw := newPreviewCoordinator(t)
	folder := filepath.Join(c.db.GetConfig().DestinationFolder, "GP12345678")
	if err := os.MkdirAll(folder, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "GX010001.MP4"), []byte("hevc"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := c.RequestPreview("cam1", "100GOPRO/GX010001.MP4"); err != nil {
		t.Fatal(err)
	}
	c.Wait()

	if fw.factoryCalls.Load() != 0 {
		t.Error("local file preview must not open a camera session")
	}
	if _, err := os.Stat(PreviewPath(folder, "GX010001.MP4")); err != nil {
		t.Errorf("preview not generated: %v", err)
	}
	if len(c.db.GetSyncQueue()) != 0 {
		t.Error("preview task entry not removed")
	}
}

// The sync fetches LRV sidecars; preview generation must transcode the
// small sidecar instead of the full-res original, then consume it.
func TestGeneratePreviewPrefersLRVSidecar(t *testing.T) {
	c, _ := newPreviewCoordinator(t)
	folder := filepath.Join(c.db.GetConfig().DestinationFolder, "GP12345678")
	src := filepath.Join(folder, "GX010001.MP4")
	sidecar := rawLRVPath(folder, "GX010001.MP4")
	if err := os.MkdirAll(filepath.Dir(sidecar), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("full-res"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, []byte("lrv"), 0644); err != nil {
		t.Fatal(err)
	}
	var gotSrc string
	c.transcode = func(_ context.Context, s, dst string) error {
		gotSrc = s
		return os.WriteFile(dst, []byte("webm"), 0644)
	}

	out, err := c.GeneratePreviewFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if gotSrc != sidecar {
		t.Errorf("transcoded %s, want the sidecar %s", gotSrc, sidecar)
	}
	if out != PreviewPath(folder, "GX010001.MP4") {
		t.Errorf("preview at %s", out)
	}
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Error("sidecar must be consumed after transcode")
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// Arming while out of range records intent; the session comes up via the
// sync-cadence maintainer once the camera appears, and disarming ends it.
func TestArmedPreviewSessionLifecycle(t *testing.T) {
	c, fw := newPreviewCoordinator(t)
	_ = c.db.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Status.IsReachable = false
	})

	if err := c.SetPreviewSession("cam1", true); err != nil {
		t.Fatal(err)
	}
	c.ProcessSyncQueue()
	if fw.factoryCalls.Load() != 0 {
		t.Fatal("session must not start while the camera is out of range")
	}

	_ = c.db.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Status.IsReachable = true
	})
	c.ProcessSyncQueue()
	waitFor(t, "armed session to establish", func() bool { return fw.factoryCalls.Load() == 1 })

	if err := c.SetPreviewSession("cam1", false); err != nil {
		t.Fatal(err)
	}
	c.Wait()
	cs, _ := c.db.GetCameraByID("cam1")
	if cs.Status.PreviewEnabled || cs.Status.IsSyncing {
		t.Error("disarm must clear the flag and end the session")
	}
	if len(c.db.GetSyncQueue()) != 0 {
		t.Error("session queue entry not removed")
	}
}

// One camera holds the preview slot; arming another moves it.
func TestArmedPreviewSingleSlot(t *testing.T) {
	c, _ := newPreviewCoordinator(t)
	_ = c.db.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Status.IsReachable = false
	})
	seedCamera(t, c.db, "cam2", model.CameraStatus{IsManaged: true, IsPaired: true, IsReachable: false})

	if err := c.SetPreviewSession("cam1", true); err != nil {
		t.Fatal(err)
	}
	if err := c.SetPreviewSession("cam2", true); err != nil {
		t.Fatal(err)
	}
	cs1, _ := c.db.GetCameraByID("cam1")
	cs2, _ := c.db.GetCameraByID("cam2")
	if cs1.Status.PreviewEnabled || !cs2.Status.PreviewEnabled {
		t.Errorf("slot did not move: cam1=%v cam2=%v", cs1.Status.PreviewEnabled, cs2.Status.PreviewEnabled)
	}
}

func TestPreviewRejectsNonVideo(t *testing.T) {
	c, _ := newPreviewCoordinator(t)
	if err := c.RequestPreview("cam1", "100GOPRO/G0010001.JPG"); err == nil {
		t.Error("photo must not open a preview session")
	}
}
