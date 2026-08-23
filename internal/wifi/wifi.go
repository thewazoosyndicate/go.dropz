package wifi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dropz/dropz/internal/logging"
)

// Shared HTTP client for downloads — transport-level timeouts only,
// no body-read timeout so large transfers aren't killed mid-stream.
var downloadTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 15 * time.Second,
}

var downloadClient = &http.Client{Transport: downloadTransport}

// GoPro HTTP API constants
const (
	GoProBaseURL = "http://10.5.5.9:8080"
	// Status endpoints
	StatusURL = "/gopro/camera/state"
	// Media endpoints
	MediaListURL = "/gopro/media/list"
	MediaInfoURL = "/gopro/media/info"
	ThumbnailURL = "/gopro/media/thumbnail"
	TurboURL     = "/gopro/media/turbo_transfer"
)

// WiFiManager handles WiFi operations for GoPro devices
type WiFiManager struct {
	log *slog.Logger
}

// NewWiFiManager creates a new WiFi manager
func NewWiFiManager(log *slog.Logger) *WiFiManager {
	return &WiFiManager{
		log: log.With("component", "wifi"),
	}
}

// DownloadVideos downloads videos from a GoPro device.
// A non-empty fileNames selection downloads exactly those files (media
// browser); empty falls back to the date-threshold window.
func (m *WiFiManager) DownloadVideos(ctx context.Context, destDir string, daysInPast int, fileNames []string) ([]string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	mediaFiles, err := m.getMediaList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get media list: %w", err)
	}

	if len(mediaFiles) == 0 {
		m.log.Info("No media files found on camera")
		return nil, nil
	}

	var filteredMedia []MediaFile
	if len(fileNames) > 0 {
		selected := make(map[string]bool, len(fileNames))
		for _, n := range fileNames {
			selected[n] = true
		}
		for _, media := range mediaFiles {
			if selected[media.Name] {
				filteredMedia = append(filteredMedia, media)
			}
		}
		m.log.Info("Media list fetched", "total", len(mediaFiles), "selected", len(filteredMedia))
	} else {
		if daysInPast <= 0 {
			daysInPast = 7
		}
		cutoffTime := time.Now().AddDate(0, 0, -daysInPast)
		filteredMedia = filterMediaByDate(mediaFiles, cutoffTime)
		m.log.Info("Media list fetched", "total", len(mediaFiles), "in_window", len(filteredMedia), "days", daysInPast)
	}

	var downloadedFiles []string
	skippedCount := 0
	failedCount := 0
	var totalBytes int64
	var totalSeconds float64

	for _, media := range filteredMedia {
		select {
		case <-ctx.Done():
			return downloadedFiles, ctx.Err()
		default:
		}

		outputPath := filepath.Join(destDir, media.Name)

		if exists, _ := m.fileExistsWithSize(outputPath, media.Size); exists {
			downloadedFiles = append(downloadedFiles, outputPath)
			skippedCount++
			continue
		}

		dlStart := time.Now()
		dlErr := m.downloadFileWithResume(ctx, media.URL, outputPath, media.CreatedAt, media.Size)
		if dlErr != nil {
			// Per-file, skipped and continued; the summary below counts them
			m.log.Warn("Failed to download media file", "file", media.Name, "err", dlErr)
			failedCount++
			continue
		}

		// Throughput per file: the number that settles turbo-vs-not debates
		if fi, statErr := os.Stat(outputPath); statErr == nil {
			elapsed := time.Since(dlStart).Seconds()
			totalBytes += fi.Size()
			totalSeconds += elapsed
			if elapsed > 0.5 {
				m.log.Info("File downloaded", "file", media.Name,
					"mb", fmt.Sprintf("%.1f", float64(fi.Size())/1e6),
					"mb_per_s", fmt.Sprintf("%.1f", float64(fi.Size())/1e6/elapsed))
			}
		}

		downloadedFiles = append(downloadedFiles, outputPath)
	}

	// A healthy 5GHz link does 20+ MB/s; sustained sub-2 usually means the
	// host WiFi is degraded, not the camera. Known case: mt7925 on kernel
	// 7.1.x pins the rate after an AP switch (docs/conformance.md).
	if totalBytes > 32<<20 && totalSeconds > 0 && float64(totalBytes)/1e6/totalSeconds < 2 {
		m.log.Warn("WiFi throughput unusually low for the whole sync",
			"mb_per_s", fmt.Sprintf("%.1f", float64(totalBytes)/1e6/totalSeconds),
			"hint", "host wifi driver or signal problem; see docs/conformance.md")
	}

	actualDownloads := len(downloadedFiles) - skippedCount
	if failedCount > 0 {
		m.log.Warn("Download finished with failures", "downloaded", actualDownloads, "skipped", skippedCount, "failed", failedCount)
	} else {
		m.log.Info("Download finished", "downloaded", actualDownloads, "skipped", skippedCount)
	}
	return downloadedFiles, nil
}

// DownloadThumbnail fetches the camera-generated preview JPEG for one file
// and writes it to outPath (tmp + rename so readers never see partials).
func (m *WiFiManager) DownloadThumbnail(ctx context.Context, cameraPath, outPath string) error {
	url := fmt.Sprintf("%s%s?path=%s", GoProBaseURL, ThumbnailURL, cameraPath)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("thumbnail request failed: %s", resp.Status)
	}
	tmp := outPath + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, outPath)
}

// SetTurboTransfer toggles Turbo Transfer, the faster WiFi offload mode.
// Spec: enable only during media offload, disable afterwards. Unsupported
// cameras answer 501; callers should treat failure as non-fatal.
func (m *WiFiManager) SetTurboTransfer(ctx context.Context, enable bool) error {
	p := "0"
	if enable {
		p = "1"
	}
	url := fmt.Sprintf("%s%s?p=%s", GoProBaseURL, TurboURL, p)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("turbo transfer request failed: %s", resp.Status)
	}
	return nil
}

// GoProMediaList represents the JSON response from the GoPro media list API
type GoProMediaList struct {
	ID    string `json:"id"`
	Media []struct {
		Directory string      `json:"d"`
		FS        []goProFile `json:"fs"`
	} `json:"media"`
}

// goProFile represents a file in the GoPro media list
type goProFile struct {
	// Basic fields
	Name       string      `json:"n"`
	CreatedAt  json.Number `json:"cre"`
	ModifiedAt json.Number `json:"mod"`
	Size       json.Number `json:"s"`

	// Optional fields
	RawFlag    json.Number `json:"raw,omitempty"`
	LastStatus json.Number `json:"ls,omitempty"`
	GLRVersion json.Number `json:"glrv,omitempty"`

	// Group file fields (for time lapse/burst photos)
	FirstID    string   `json:"b,omitempty"`
	LastID     string   `json:"l,omitempty"`
	GroupID    string   `json:"g,omitempty"`
	MissingIDs []string `json:"m,omitempty"`
	FileType   string   `json:"t,omitempty"`
}

// MediaFile represents a media file on the GoPro
type MediaFile struct {
	Name       string
	URL        string
	CameraPath string // <dir>/<name>, the camera API's path parameter
	CreatedAt  time.Time
	Size       int64
}

// GetCameraStatus retrieves the camera status via HTTP API
func (m *WiFiManager) GetCameraStatus(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s%s", GoProBaseURL, StatusURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get camera status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("camera status request failed with status: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode camera status: %w", err)
	}
	return result, nil
}

// getMediaList retrieves the list of media files from a GoPro device via HTTP API
// ListMedia returns the parsed media list without downloading anything.
// Used by diagnostic tools (cmd/ble-probe).
func (m *WiFiManager) ListMedia(ctx context.Context) ([]MediaFile, error) {
	return m.getMediaList(ctx)
}

func (m *WiFiManager) getMediaList(ctx context.Context) ([]MediaFile, error) {
	url := fmt.Sprintf("%s%s", GoProBaseURL, MediaListURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get media list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("media list request failed with status: %s", resp.Status)
	}

	// Parse the response
	var mediaList GoProMediaList
	if err := json.NewDecoder(resp.Body).Decode(&mediaList); err != nil {
		return nil, fmt.Errorf("failed to decode media list: %w", err)
	}

	// Convert to our internal format
	result := make([]MediaFile, 0)
	for _, media := range mediaList.Media {
		for _, file := range media.FS {
			// Skip directories or special files
			if !strings.HasSuffix(strings.ToLower(file.Name), ".mp4") &&
				!strings.HasSuffix(strings.ToLower(file.Name), ".jpg") {
				continue
			}

			// Parse the creation time
			createdInt, err := file.CreatedAt.Int64()
			if err != nil {
				// Per-item, skipped and continued
				m.log.Warn("Failed to parse createdAt value", "file", file.Name, "value", file.CreatedAt, "err", err)
				continue
			}
			createdAtTime := time.Unix(createdInt, 0)

			// For grouped items (burst, time lapse) "s" is the member count,
			// not bytes (spec media list schema); per-member sizes are unknown.
			isGroup := file.GroupID != ""
			var sizeInt int64
			if !isGroup {
				sizeInt, err = file.Size.Int64()
				if err != nil {
					m.log.Warn("Failed to parse size value", "file", file.Name, "value", file.Size, "err", err)
					continue
				}
			}

			// Build the download URL - using directory from media and filename
			mediaURL := fmt.Sprintf("%s/videos/DCIM/%s/%s", GoProBaseURL, media.Directory, file.Name)

			result = append(result, MediaFile{
				Name:       file.Name,
				URL:        mediaURL,
				CameraPath: fmt.Sprintf("%s/%s", media.Directory, file.Name),
				CreatedAt:  createdAtTime,
				Size:       sizeInt,
			})

			// A grouped entry only lists its first member; the siblings must
			// be extrapolated from b..l skipping m (spec media list schema).
			if isGroup {
				for _, member := range expandGroupMembers(file.Name, file.FirstID, file.LastID, file.MissingIDs) {
					result = append(result, MediaFile{
						Name:       member,
						URL:        fmt.Sprintf("%s/videos/DCIM/%s/%s", GoProBaseURL, media.Directory, member),
						CameraPath: fmt.Sprintf("%s/%s", media.Directory, member),
						CreatedAt:  createdAtTime,
						Size:       0,
					})
				}
			}
		}
	}

	return result, nil
}

// expandGroupMembers returns the sibling filenames of a grouped media item.
// Names follow GXXXYYYY.EXT: chars 1-3 are the group id, chars 4-7 the member
// id. The listed item is skipped; missing ids (deleted members) are skipped.
func expandGroupMembers(name, firstID, lastID string, missingIDs []string) []string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if len(base) != 8 || (base[0] != 'G' && base[0] != 'g') {
		return nil
	}
	first, err1 := strconv.Atoi(firstID)
	last, err2 := strconv.Atoi(lastID)
	if err1 != nil || err2 != nil || last < first || last-first > 9999 {
		return nil
	}
	missing := make(map[int]bool, len(missingIDs))
	for _, id := range missingIDs {
		if v, err := strconv.Atoi(id); err == nil {
			missing[v] = true
		}
	}
	var members []string
	for id := first; id <= last; id++ {
		if missing[id] {
			continue
		}
		member := fmt.Sprintf("%s%04d%s", base[:4], id, ext)
		if member == name {
			continue
		}
		members = append(members, member)
	}
	return members
}

// downloadFileWithResume downloads a file with resume capability.
// The caller is responsible for skipping files that already exist.
func (m *WiFiManager) downloadFileWithResume(ctx context.Context, url, outputPath string, createdAt time.Time, totalSize int64) error {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			logging.Trace(m.log, "Retrying download", "attempt", attempt+1, "max", maxRetries, "file", outputPath)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second): // Exponential backoff
				// Continue with retry
			}
		}

		err := m.downloadFileWithResumeOnce(ctx, url, outputPath, createdAt, totalSize)
		if err == nil {
			return nil // Success
		}

		lastErr = err
		// Per-attempt at Trace; the per-file Warn happens once in the caller
		logging.Trace(m.log, "Download attempt failed", "file", outputPath, "attempt", attempt+1, "max", maxRetries, "err", err)
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// downloadFileWithResumeOnce performs a single attempt to download a file with resume capability
func (m *WiFiManager) downloadFileWithResumeOnce(ctx context.Context, url, outputPath string, createdAt time.Time, totalSize int64) error {
	tempFilePath := outputPath + ".partial"

	// Check if partial download exists
	var startOffset int64 = 0
	fileInfo, err := os.Stat(tempFilePath)
	if err == nil {
		startOffset = fileInfo.Size()
		logging.Trace(m.log, "Resuming download", "file", outputPath, "offset", startOffset)
	}

	// Create/open the file for appending
	var file *os.File
	if startOffset > 0 {
		file, err = os.OpenFile(tempFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(tempFilePath)
	}

	if err != nil {
		return fmt.Errorf("failed to create/open output file: %w", err)
	}
	defer file.Close()

	// If file is already complete, just rename it
	if totalSize > 0 && startOffset == totalSize {
		logging.Trace(m.log, "File already complete", "file", outputPath)
		return os.Rename(tempFilePath, outputPath)
	}

	// Prepare the request with range header
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	// Report progress periodically
	progressReader := &progressReader{
		reader:     resp.Body,
		total:      totalSize,
		current:    startOffset,
		reportTime: time.Now(),
		fileName:   filepath.Base(outputPath),
		logger:     m.log,
	}

	// Set up a pipe with buffer for faster downloads
	buf := make([]byte, 1024*1024) // 1MB buffer
	_, err = io.CopyBuffer(file, progressReader, buf)

	// Check copy error
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Close the file before moving it
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Rename the temporary file to the final filename
	if err := os.Rename(tempFilePath, outputPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// After renaming the file, verify its size (silent check)
	{
		fi, err := os.Stat(outputPath)
		if err != nil {
			m.log.Warn("Failed to stat downloaded file", "file", outputPath, "err", err)
		} else if totalSize > 0 && fi.Size() != totalSize {
			return fmt.Errorf("file size verification failed for %s: expected %d bytes, got %d bytes", outputPath, totalSize, fi.Size())
		}
	}

	// Set the file timestamps to match the creation time
	if err := os.Chtimes(outputPath, createdAt, createdAt); err != nil {
		m.log.Warn("Failed to set file timestamps", "file", outputPath, "err", err)
	}

	return nil
}

// progressReader is an io.Reader that reports download progress
type progressReader struct {
	reader     io.Reader
	total      int64
	current    int64
	reportTime time.Time
	fileName   string
	logger     *slog.Logger
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	// Report progress every 3 seconds; numeric attrs so the JSON handler
	// emits numbers the Electron host can use
	if time.Since(pr.reportTime) > 3*time.Second && pr.total > 0 {
		logging.Trace(pr.logger, "Downloading",
			"file", pr.fileName,
			"percent", int(float64(pr.current)/float64(pr.total)*100),
			"current_mb", pr.current/(1024*1024),
			"total_mb", pr.total/(1024*1024))
		pr.reportTime = time.Now()
	}

	return n, err
}

// filterMediaByDate filters media files by their creation date
func filterMediaByDate(mediaFiles []MediaFile, cutoffTime time.Time) []MediaFile {
	result := make([]MediaFile, 0)
	for _, media := range mediaFiles {
		if media.CreatedAt.After(cutoffTime) {
			result = append(result, media)
		}
	}
	return result
}

// fileExistsWithSize checks if a file exists and has the expected size.
// Size 0 means unknown (grouped media members): any non-empty file counts.
func (m *WiFiManager) fileExistsWithSize(filePath string, expectedSize int64) (bool, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if expectedSize <= 0 {
		return fi.Size() > 0, nil
	}
	return fi.Size() == expectedSize, nil
}
