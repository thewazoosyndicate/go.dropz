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

// SetPreviewSession arms or disarms a camera's standing preview session.
func (m *GoProManager) SetPreviewSession(cameraID string, enabled bool) error {
	return m.syncCoordinator.SetPreviewSession(cameraID, enabled)
}

// libraryFile validates a path the RPC handed us: only files inside the
// library may be read or written next to.
func (m *GoProManager) libraryFile(videoPath string) (string, error) {
	dest := m.db.GetConfig().DestinationFolder
	if dest == "" {
		return "", fmt.Errorf("no destination folder configured")
	}
	clean := filepath.Clean(videoPath)
	if !strings.HasPrefix(clean, filepath.Clean(dest)+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside the library: %s", videoPath)
	}
	if fi, err := os.Stat(clean); err != nil || fi.IsDir() {
		return "", fmt.Errorf("no such library file: %s", videoPath)
	}
	return clean, nil
}

// PreviewVideo generates (or reuses) the in-app playable preview of a
// library file. Synchronous; long clips take a while.
func (m *GoProManager) PreviewVideo(videoPath string) (string, error) {
	clean, err := m.libraryFile(videoPath)
	if err != nil {
		return "", err
	}
	return m.syncCoordinator.GeneratePreviewFile(clean)
}

// VideoKeyframes lists where a lossless trim of a library clip can start.
func (m *GoProManager) VideoKeyframes(videoPath string) ([]int64, int64, error) {
	clean, err := m.libraryFile(videoPath)
	if err != nil {
		return nil, 0, err
	}
	return m.syncCoordinator.VideoKeyframes(clean)
}

// TrimVideo writes a lossless cut of a library clip next to it.
func (m *GoProManager) TrimVideo(videoPath string, startMs, endMs int64) (*model.TrimResult, error) {
	clean, err := m.libraryFile(videoPath)
	if err != nil {
		return nil, err
	}
	return m.syncCoordinator.TrimVideo(clean, startMs, endMs)
}
