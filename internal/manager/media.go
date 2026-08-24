package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dropz/dropz/internal/manager/syncer"
	"github.com/dropz/dropz/internal/model"
)

// cameraFolder returns the per-camera download folder, empty when the
// camera has no WiFi SSID yet (never paired).
func (m *GoProManager) cameraFolder(cs *model.CameraWithState) string {
	dest := m.db.GetConfig().DestinationFolder
	if dest == "" || cs.Camera.WiFiSSID == "" {
		return ""
	}
	return filepath.Join(dest, cs.Camera.WiFiSSID)
}

// GetCameraMedia returns the camera's cached media catalog with local
// state resolved. Empty until the first sync captures a catalog.
func (m *GoProManager) GetCameraMedia(cameraID string) ([]model.CameraMediaItem, time.Time, error) {
	cs, ok := m.db.GetCameraByID(cameraID)
	if !ok {
		return nil, time.Time{}, fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}
	folder := m.cameraFolder(cs)
	if folder == "" {
		return nil, time.Time{}, nil
	}
	return syncer.ReadCatalog(folder)
}

// RequestMediaDownload queues a selection download (or catalog-only
// refresh when the selection is empty) for the media browser.
func (m *GoProManager) RequestMediaDownload(cameraID string, fileNames []string) (*model.SyncQueueEntry, error) {
	return m.syncCoordinator.RequestMediaDownload(cameraID, fileNames)
}

// PreviewMedia fetches a clip's LRV proxy into the preview cache.
func (m *GoProManager) PreviewMedia(cameraID, cameraPath string) error {
	return m.syncCoordinator.RequestPreview(cameraID, cameraPath)
}

// PreviewVideo generates (or reuses) the in-app playable preview of a
// library file. Synchronous; long clips take a while.
func (m *GoProManager) PreviewVideo(videoPath string) (string, error) {
	dest := m.db.GetConfig().DestinationFolder
	if dest == "" {
		return "", fmt.Errorf("no destination folder configured")
	}
	// The RPC hands us a path; only files inside the library may be read
	clean := filepath.Clean(videoPath)
	if !strings.HasPrefix(clean, filepath.Clean(dest)+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside the library: %s", videoPath)
	}
	if fi, err := os.Stat(clean); err != nil || fi.IsDir() {
		return "", fmt.Errorf("no such library file: %s", videoPath)
	}
	return m.syncCoordinator.GeneratePreviewFile(clean)
}
