package syncer

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/wifi"
)

// The media catalog lives next to the downloads, so the Electron renderer
// loads thumbnails straight from disk. Dotted names keep the library
// folder scan from picking them up as media.
const (
	catalogFileName  = ".catalog.json"
	thumbnailDirName = ".thumbnails"
	previewDirName   = ".previews"
)

type catalogEntry struct {
	Name       string    `json:"name"`
	CameraPath string    `json:"camera_path"`
	SizeBytes  int64     `json:"size_bytes"`
	CreatedAt  time.Time `json:"created_at"`
}

type catalogFile struct {
	UpdatedAt time.Time      `json:"updated_at"`
	Files     []catalogEntry `json:"files"`
}

// ThumbnailPath returns where a file's preview is stored locally.
func ThumbnailPath(cameraFolder, name string) string {
	return filepath.Join(cameraFolder, thumbnailDirName, name+".jpg")
}

// PreviewPath returns where a video's cached LRV proxy lives. The .mp4
// extension keeps the system player association; LRV is a plain MP4.
func PreviewPath(cameraFolder, localName string) string {
	base := strings.TrimSuffix(localName, filepath.Ext(localName))
	return filepath.Join(cameraFolder, previewDirName, base+".lrv.mp4")
}

// WriteCatalog persists the camera's media list (tmp + rename).
func WriteCatalog(cameraFolder string, files []wifi.MediaFile) error {
	if err := os.MkdirAll(cameraFolder, 0755); err != nil {
		return err
	}
	cat := catalogFile{UpdatedAt: time.Now(), Files: make([]catalogEntry, 0, len(files))}
	for _, f := range files {
		cat.Files = append(cat.Files, catalogEntry{
			Name:       f.Name,
			CameraPath: f.CameraPath,
			SizeBytes:  f.Size,
			CreatedAt:  f.CreatedAt,
		})
	}
	data, err := json.MarshalIndent(cat, "", " ")
	if err != nil {
		return err
	}
	path := filepath.Join(cameraFolder, catalogFileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadCatalog loads the cached catalog and resolves each entry's local
// state: thumbnail present, file already downloaded.
func ReadCatalog(cameraFolder string) ([]model.CameraMediaItem, time.Time, error) {
	data, err := os.ReadFile(filepath.Join(cameraFolder, catalogFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, time.Time{}, nil // never synced: empty catalog, not an error
		}
		return nil, time.Time{}, err
	}
	var cat catalogFile
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, time.Time{}, err
	}

	// Downloads and thumbnails live under the local name, which carries a
	// directory prefix when the card duplicates a name; mirror that here.
	dupes := make(map[string]int, len(cat.Files))
	for _, f := range cat.Files {
		dupes[f.Name]++
	}

	items := make([]model.CameraMediaItem, 0, len(cat.Files))
	for _, f := range cat.Files {
		item := model.CameraMediaItem{
			Name:       f.Name,
			CameraPath: f.CameraPath,
			SizeBytes:  f.SizeBytes,
			CreatedAt:  f.CreatedAt,
		}
		local := wifi.LocalMediaName(f.CameraPath, f.Name, dupes[f.Name] > 1)
		if thumb := ThumbnailPath(cameraFolder, local); fileExists(thumb) {
			item.ThumbnailPath = thumb
		}
		if preview := PreviewPath(cameraFolder, local); fileExists(preview) {
			item.PreviewPath = preview
		}
		if localPath := filepath.Join(cameraFolder, local); fileExists(localPath) {
			item.Downloaded = true
			item.LocalPath = localPath
		}
		items = append(items, item)
	}
	return items, cat.UpdatedAt, nil
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Size() > 0
}

// refreshThumbnails fetches previews the local cache is missing.
// Best effort per file; a camera mid-capture can refuse some.
func refreshThumbnails(ctx context.Context, wm wifiClient, cameraFolder string, files []wifi.MediaFile, log *slog.Logger) {
	thumbDir := filepath.Join(cameraFolder, thumbnailDirName)
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		log.Warn("Cannot create thumbnail dir", "dir", thumbDir, "err", err)
		return
	}
	dupes := wifi.DuplicateNames(files)
	fetched, failed := 0, 0
	for _, f := range files {
		if ctx.Err() != nil {
			return
		}
		out := ThumbnailPath(cameraFolder, wifi.LocalMediaName(f.CameraPath, f.Name, dupes[f.Name]))
		if fileExists(out) {
			continue
		}
		if err := wm.DownloadThumbnail(ctx, f.CameraPath, out); err != nil {
			failed++
			continue
		}
		fetched++
	}
	if fetched > 0 || failed > 0 {
		log.Info("Thumbnails refreshed", "fetched", fetched, "failed", failed)
	}
}
