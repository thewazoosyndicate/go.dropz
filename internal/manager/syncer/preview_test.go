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

func TestPreviewRejectsNonVideo(t *testing.T) {
	c, _ := newPreviewCoordinator(t)
	if err := c.RequestPreview("cam1", "100GOPRO/G0010001.JPG"); err == nil {
		t.Error("photo must not open a preview session")
	}
}
