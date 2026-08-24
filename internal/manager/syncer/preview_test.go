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

func TestPreviewRejectsNonVideo(t *testing.T) {
	c, _ := newPreviewCoordinator(t)
	if err := c.RequestPreview("cam1", "100GOPRO/G0010001.JPG"); err == nil {
		t.Error("photo must not open a preview session")
	}
}
