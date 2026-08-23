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

// DownloadVideos downloads videos from a GoPro device
func (m *WiFiManager) DownloadVideos(ctx context.Context, destDir string, daysInPast int) ([]string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	mediaFiles, err := m.getMediaList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get media list: %w", err)
	}

	if len(mediaFiles) == 0 {
		m.log.Info("No media files found on GoPro")
		return nil, nil
	}

	if daysInPast <= 0 {
		daysInPast = 7
	}

	cutoffTime := time.Now().AddDate(0, 0, -daysInPast)
	filteredMedia := filterMediaByDate(mediaFiles, cutoffTime)

	m.log.Info("Media list fetched", "total", len(mediaFiles), "in_window", len(filteredMedia), "days", daysInPast)

	var downloadedFiles []string
	skippedCount := 0

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

		dlErr := m.downloadFileWithResume(ctx, media.URL, outputPath, media.CreatedAt, media.Size)
		if dlErr != nil {
			m.log.Error("Failed to download media file", "file", media.Name, "err", dlErr)
			continue
		}

		downloadedFiles = append(downloadedFiles, outputPath)
	}

	actualDownloads := len(downloadedFiles) - skippedCount
	m.log.Info("Download complete", "downloaded", actualDownloads, "skipped", skippedCount)
	return downloadedFiles, nil
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
	Name      string
	URL       string
	CreatedAt time.Time
	Size      int64
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
				m.log.Error("Failed to parse createdAt value", "value", file.CreatedAt, "err", err)
				continue
			}
			createdAtTime := time.Unix(createdInt, 0)

			// Parse the file size
			sizeInt, err := file.Size.Int64()
			if err != nil {
				m.log.Error("Failed to parse size value", "value", file.Size, "err", err)
				continue
			}

			// Build the download URL - using directory from media and filename
			mediaURL := fmt.Sprintf("%s/videos/DCIM/%s/%s", GoProBaseURL, media.Directory, file.Name)

			result = append(result, MediaFile{
				Name:      file.Name,
				URL:       mediaURL,
				CreatedAt: createdAtTime,
				Size:      sizeInt,
			})
		}
	}

	m.log.Debug("Media list parsed", "files", len(result))
	return result, nil
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
		m.log.Warn("Download attempt failed", "file", outputPath, "err", err)
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
	if startOffset == totalSize {
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
		} else if fi.Size() != totalSize {
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

	// Report progress every 3 seconds
	if time.Since(pr.reportTime) > 3*time.Second {
		percentComplete := float64(pr.current) / float64(pr.total) * 100
		pr.logger.Debug("Downloading",
			"file", pr.fileName,
			"percent", fmt.Sprintf("%.1f", percentComplete),
			"mb", fmt.Sprintf("%.2f/%.2f", float64(pr.current)/(1024*1024), float64(pr.total)/(1024*1024)))
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

// fileExistsWithSize checks if a file exists and has the expected size
func (m *WiFiManager) fileExistsWithSize(filePath string, expectedSize int64) (bool, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.Size() == expectedSize, nil
}
