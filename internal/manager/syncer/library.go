package syncer

import (
	"crypto/sha256"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dropz/dropz/internal/model"
)

// GetVideosByCamera scans the destination folder for media files and maps
// them to cameras via WiFi SSID. Lives here because this package owns the
// on-disk library layout (camera folders, catalog, thumbnails).
func (c *Coordinator) GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*model.VideoFile, int) {
	config := c.db.GetConfig()
	if config.DestinationFolder == "" {
		return []*model.VideoFile{}, 0
	}

	// Build WiFi SSID to camera ID map
	ssidToCamera := make(map[string]string)
	for _, cam := range c.db.GetAllCameras() {
		if cam.Camera.WiFiSSID != "" {
			ssidToCamera[cam.Camera.WiFiSSID] = cam.Camera.ID
		}
	}

	entries, err := os.ReadDir(config.DestinationFolder)
	if err != nil {
		c.log.Warn("Cannot read destination folder", "folder", config.DestinationFolder, "err", err)
		return []*model.VideoFile{}, 0
	}

	var allVideos []*model.VideoFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ssid := entry.Name()
		camID, known := ssidToCamera[ssid]
		if !known {
			continue
		}
		if cameraID != "" && camID != cameraID {
			continue
		}

		subdir := filepath.Join(config.DestinationFolder, ssid)
		files, err := os.ReadDir(subdir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(f.Name()))
			mimeType := mime.TypeByExtension(ext)
			if !strings.HasPrefix(mimeType, "video/") && !strings.HasPrefix(mimeType, "image/") {
				continue
			}

			fullPath := filepath.Join(subdir, f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}

			mtime := info.ModTime()
			if !startDate.IsZero() && mtime.Before(startDate) {
				continue
			}
			if !endDate.IsZero() && mtime.After(endDate) {
				continue
			}

			hash := sha256.Sum256([]byte(fullPath))
			video := &model.VideoFile{
				ID:        fmt.Sprintf("%x", hash[:8]),
				Name:      f.Name(),
				Path:      fullPath,
				SizeBytes: info.Size(),
				CreatedAt: mtime,
				CameraID:  camID,
				MimeType:  mimeType,
			}
			// Camera-generated preview cached by the media catalog sync
			if thumb := ThumbnailPath(subdir, f.Name()); fileExists(thumb) {
				video.ThumbnailPath = thumb
			}
			if preview := PreviewPath(subdir, f.Name()); fileExists(preview) {
				video.PreviewPath = preview
			}
			allVideos = append(allVideos, video)
		}
	}

	// Sort newest first
	sort.Slice(allVideos, func(i, j int) bool {
		return allVideos[i].CreatedAt.After(allVideos[j].CreatedAt)
	})

	totalCount := len(allVideos)
	if offset >= totalCount {
		return []*model.VideoFile{}, totalCount
	}
	end := offset + limit
	if limit == 0 || end > totalCount {
		end = totalCount
	}
	return allVideos[offset:end], totalCount
}
