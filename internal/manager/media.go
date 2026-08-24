package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dropz/dropz/internal/manager/syncer"
	"github.com/dropz/dropz/internal/model"
)

func fileExistsNonEmpty(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}

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
