package wifi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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
	MediaListURL      = "/gopro/media/list"
	MediaInfoURL      = "/gopro/media/info"
)

// WiFiManager handles WiFi operations for GoPro devices
type WiFiManager struct {
	log *logrus.Logger
}

// NewWiFiManager creates a new WiFi manager
func NewWiFiManager(log *logrus.Logger) *WiFiManager {
	return &WiFiManager{
		log: log,
	}
}

// DownloadVideos downloads videos from a GoPro device
func (m *WiFiManager) DownloadVideos(ctx context.Context, destDir string, daysInPast int) ([]string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %v", err)
	}

	mediaFiles, err := m.getMediaList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get media list: %v", err)
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

	m.log.Infof("Found %d media files (%d within last %d days)", len(mediaFiles), len(filteredMedia), daysInPast)

	// Set up parallel download
	var wg sync.WaitGroup
	maxWorkers := 1
	semaphore := make(chan struct{}, maxWorkers)
	var mu sync.Mutex
	downloadedFiles := make([]string, 0, len(filteredMedia))
	skippedCount := 0

	// Track files currently being downloaded to prevent race conditions
	downloadingFiles := make(map[string]bool)
	var downloadMutex sync.Mutex

	// Process each media file
	for _, media := range filteredMedia {
		select {
		case <-ctx.Done():
			// Wait for ongoing downloads to complete
			wg.Wait()
			return downloadedFiles, ctx.Err()
		default:
			// Continue processing
		}

		outputPath := filepath.Join(destDir, media.Name)

		// Check if file already exists with correct size
		exists, _ := m.fileExistsWithSize(outputPath, media.Size)
		if exists {
			mu.Lock()
			downloadedFiles = append(downloadedFiles, outputPath)
			skippedCount++
			mu.Unlock()
			continue
		}

		downloadMutex.Lock()
		if downloadingFiles[media.Name] {
			downloadMutex.Unlock()
			continue
		}
		downloadingFiles[media.Name] = true
		downloadMutex.Unlock()

		// File doesn't exist or has wrong size - proceed with download
		// Acquire semaphore slot
		semaphore <- struct{}{}
		wg.Add(1)

		// Launch download in goroutine
		go func(media MediaFile, outputPath string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			defer func() {
				// Remove from downloading files map when done
				downloadMutex.Lock()
				delete(downloadingFiles, media.Name)
				downloadMutex.Unlock()
			}()

			// Double-check: verify file doesn't exist just before downloading
			// This prevents race conditions where the file was created between
			// the initial check and when this goroutine starts
			existsDouble, _ := m.fileExistsWithSize(outputPath, media.Size)
			if existsDouble {
				mu.Lock()
				downloadedFiles = append(downloadedFiles, outputPath)
				skippedCount++
				mu.Unlock()
				return
			}

			// Download the file - use chunked download for large files
			var err error
			if media.Size > 10*1024*1024 { // 10MB threshold
				err = m.downloadChunked(ctx, media.URL, outputPath, media.CreatedAt, media.Size)
			} else {
				err = m.downloadFileWithResume(ctx, media.URL, outputPath, media.CreatedAt, media.Size)
			}

			if err != nil {
				m.log.Errorf("Failed to download %s: %v", media.Name, err)
				return
			}

			mu.Lock()
			downloadedFiles = append(downloadedFiles, outputPath)
			mu.Unlock()
		}(media, outputPath)
	}

	// Wait for all downloads to complete
	wg.Wait()

	actualDownloads := len(downloadedFiles) - skippedCount
	m.log.Infof("Download complete: %d downloaded, %d skipped", actualDownloads, skippedCount)
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
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get camera status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("camera status request failed with status: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode camera status: %v", err)
	}
	return result, nil
}

// getMediaList retrieves the list of media files from a GoPro device via HTTP API
func (m *WiFiManager) getMediaList(ctx context.Context) ([]MediaFile, error) {
	url := fmt.Sprintf("%s%s", GoProBaseURL, MediaListURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get media list: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("media list request failed with status: %s", resp.Status)
	}

	// Parse the response
	var mediaList GoProMediaList
	if err := json.NewDecoder(resp.Body).Decode(&mediaList); err != nil {
		return nil, fmt.Errorf("failed to decode media list: %v", err)
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
				m.log.Errorf("Failed to parse createdAt value %q: %v", file.CreatedAt, err)
				continue
			}
			createdAtTime := time.Unix(createdInt, 0)

			// Parse the file size
			sizeInt, err := file.Size.Int64()
			if err != nil {
				m.log.Errorf("Failed to parse size value %q: %v", file.Size, err)
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

	m.log.Debugf("Media list: %d files", len(result))
	return result, nil
}

// downloadFileWithResume downloads a file with resume capability
// This function assumes the caller has already checked if the file exists
func (m *WiFiManager) downloadFileWithResume(ctx context.Context, url, outputPath string, createdAt time.Time, totalSize int64) error {
	// Skip if file already exists with correct size
	exists, err := m.fileExistsWithSize(outputPath, totalSize)
	if err != nil {
		m.log.Warnf("Error checking file existence for %s: %v", filepath.Base(outputPath), err)
	}
	if exists {
		return nil // already downloaded
	}

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			m.log.Tracef("Retrying download (attempt %d/%d) for %s", attempt+1, maxRetries, outputPath)
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
		m.log.Warnf("Download failed for %s: %v", outputPath, err)
	}

	return fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// downloadFileWithResumeOnce performs a single attempt to download a file with resume capability
func (m *WiFiManager) downloadFileWithResumeOnce(ctx context.Context, url, outputPath string, createdAt time.Time, totalSize int64) error {
	tempFilePath := outputPath + ".partial"

	// Check if partial download exists
	var startOffset int64 = 0
	fileInfo, err := os.Stat(tempFilePath)
	if err == nil {
		startOffset = fileInfo.Size()
		m.log.Tracef("Resuming download of %s from offset %d", outputPath, startOffset)
	}

	// Create/open the file for appending
	var file *os.File
	if startOffset > 0 {
		file, err = os.OpenFile(tempFilePath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(tempFilePath)
	}

	if err != nil {
		return fmt.Errorf("failed to create/open output file: %v", err)
	}
	defer file.Close()

	// If file is already complete, just rename it
	if startOffset == totalSize {
		m.log.Tracef("File %s is already complete", outputPath)
		return os.Rename(tempFilePath, outputPath)
	}

	// Prepare the request with range header
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
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
		return fmt.Errorf("failed to copy data: %v", err)
	}

	// Close the file before moving it
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %v", err)
	}

	// Rename the temporary file to the final filename
	if err := os.Rename(tempFilePath, outputPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %v", err)
	}

	// After renaming the file, verify its size (silent check)
	{
		fi, err := os.Stat(outputPath)
		if err != nil {
			m.log.Warnf("Failed to stat downloaded file %s: %v", outputPath, err)
		} else if fi.Size() != totalSize {
			return fmt.Errorf("file size verification failed for %s: expected %d bytes, got %d bytes", outputPath, totalSize, fi.Size())
		}
	}

	// Set the file timestamps to match the creation time
	if err := os.Chtimes(outputPath, createdAt, createdAt); err != nil {
		m.log.Warnf("Failed to set file timestamps for %s: %v", outputPath, err)
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
	logger     *logrus.Logger
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	// Report progress every 3 seconds
	if time.Since(pr.reportTime) > 3*time.Second {
		percentComplete := float64(pr.current) / float64(pr.total) * 100
		pr.logger.Debugf("Downloading %s: %.1f%% complete (%.2f MB / %.2f MB)",
			pr.fileName,
			percentComplete,
			float64(pr.current)/(1024*1024),
			float64(pr.total)/(1024*1024))
		pr.reportTime = time.Now()
	}

	return n, err
}

// downloadChunk downloads a specific byte range of a file with retries
func (m *WiFiManager) downloadChunk(ctx context.Context, url, chunkPath string, startOffset, endOffset int64) error {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			m.log.Tracef("Retrying chunk download (attempt %d/%d) for range %d-%d", attempt+1, maxRetries, startOffset, endOffset)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second): // Exponential backoff
				// Continue with retry
			}
		}

		err := m.downloadChunkOnce(ctx, url, chunkPath, startOffset, endOffset)
		if err == nil {
			return nil // Success
		}

		lastErr = err
		m.log.Warnf("Chunk download failed for range %d-%d: %v", startOffset, endOffset, err)
	}

	return fmt.Errorf("failed after %d attempts: %v", maxRetries, lastErr)
}

// downloadChunkOnce downloads a specific byte range of a file in a single attempt
func (m *WiFiManager) downloadChunkOnce(ctx context.Context, url, chunkPath string, startOffset, endOffset int64) error {
	// Check if partial chunk exists
	var resumeOffset int64 = startOffset
	if fi, err := os.Stat(chunkPath); err == nil {
		resumeOffset = startOffset + fi.Size()
		if resumeOffset > endOffset {
			// Chunk is already complete
			return nil
		}
	}

	// Create or open the chunk file
	var file *os.File
	var err error

	if resumeOffset > startOffset {
		file, err = os.OpenFile(chunkPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = os.Create(chunkPath)
	}

	if err != nil {
		return fmt.Errorf("failed to create/open chunk file: %v", err)
	}
	defer file.Close()

	// Prepare request with range header
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set range header
	rangeHeader := fmt.Sprintf("bytes=%d-%d", resumeOffset, endOffset)
	req.Header.Set("Range", rangeHeader)

	resp, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download chunk: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received invalid response for range request: %d", resp.StatusCode)
	}

	// Copy response body to file
	buf := make([]byte, 256*1024) // 256KB buffer
	_, err = io.CopyBuffer(file, resp.Body, buf)

	if err != nil {
		return fmt.Errorf("error while copying data: %v", err)
	}

	return nil
}

// downloadChunked downloads large files using chunked approach
// This function assumes the caller has already checked if the file exists
func (m *WiFiManager) downloadChunked(ctx context.Context, url, outputPath string, createdAt time.Time, totalSize int64) error {
	// Skip if file already exists with correct size
	exists, err := m.fileExistsWithSize(outputPath, totalSize)
	if err != nil {
		m.log.Warnf("Error checking file existence for %s: %v", filepath.Base(outputPath), err)
	}
	if exists {
		return nil // already downloaded
	}

	// Only use chunked download for files larger than 10MB
	if totalSize < 10*1024*1024 {
		return m.downloadFileWithResume(ctx, url, outputPath, createdAt, totalSize)
	}

	tempDir := outputPath + ".chunks"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Determine optimal chunk size and count
	chunkSize := int64(64 * 1024 * 1024) // 64MB chunks
	chunkCount := (totalSize + chunkSize - 1) / chunkSize

	m.log.Debugf("Downloading %s in %d chunks of size %d bytes", outputPath, chunkCount, chunkSize)

	var wg sync.WaitGroup
	errors := make(chan error, chunkCount)
	maxConcurrent := 1
	semaphore := make(chan struct{}, maxConcurrent)

	// Download each chunk
	for i := int64(0); i < chunkCount; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		startOffset := i * chunkSize
		endOffset := (i+1)*chunkSize - 1
		if endOffset >= totalSize {
			endOffset = totalSize - 1
		}

		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk-%03d", i))

		go func(i, startOffset, endOffset int64, chunkPath string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			chunkSize := endOffset - startOffset + 1
			m.log.Tracef("Downloading chunk %d/%d (%d bytes) to %s",
				i+1, chunkCount, chunkSize, filepath.Base(chunkPath))

			// Skip if chunk already exists and has the right size
			expectedSize := endOffset - startOffset + 1
			if fi, err := os.Stat(chunkPath); err == nil && fi.Size() == expectedSize {
				m.log.Tracef("Chunk %d/%d already exists with right size, skipping", i+1, chunkCount)
				return
			}

			err := m.downloadChunk(ctx, url, chunkPath, startOffset, endOffset)
			if err != nil {
				select {
				case errors <- fmt.Errorf("chunk %d/%d failed: %v", i+1, chunkCount, err):
				default:
				}
			} else {
				m.log.Tracef("Successfully downloaded chunk %d/%d", i+1, chunkCount)
			}
		}(i, startOffset, endOffset, chunkPath)
	}

	// Wait for all chunks to download
	wg.Wait()

	// Check for errors
	select {
	case err := <-errors:
		return err
	default:
		// No errors
	}

	// Create the final file
	finalFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %v", err)
	}
	defer finalFile.Close()

	// Combine chunks
	buffer := make([]byte, 1024*1024) // 1MB buffer
	for i := int64(0); i < chunkCount; i++ {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk-%03d", i))

		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to open chunk %d: %v", i, err)
		}

		_, err = io.CopyBuffer(finalFile, chunkFile, buffer)
		chunkFile.Close()

		if err != nil {
			return fmt.Errorf("failed to copy chunk %d to final file: %v", i, err)
		}
	}

	// After combining all chunks, verify the final file size (silent check)
	{
		fi, err := os.Stat(outputPath)
		if err != nil {
			m.log.Warnf("Failed to stat combined file %s: %v", outputPath, err)
		} else if fi.Size() != totalSize {
			return fmt.Errorf("file size verification failed for %s: expected %d bytes, got %d bytes", outputPath, totalSize, fi.Size())
		}
	}

	// Set the file timestamps to match the creation time
	if err := os.Chtimes(outputPath, createdAt, createdAt); err != nil {
		m.log.Warnf("Failed to set file timestamps for %s: %v", outputPath, err)
	}

	return nil
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
