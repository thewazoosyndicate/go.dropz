package wifi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// GoPro HTTP API constants
const (
	GoProBaseURL = "http://10.5.5.9:8080"
	// Status endpoints
	StatusURL     = "/gopro/camera/state"
	BatteryURL    = "/gopro/status/battery"
	// Media endpoints
	MediaListURL      = "/gopro/media/list"
	MediaInfoURL      = "/gopro/media/info"
	TurboTransferURL  = "/gopro/media/turbo_transfer"
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

// Connect connects to a GoPro WiFi network
func (m *WiFiManager) Connect(ctx context.Context, ssid, password string) error {
	m.log.Debugf("Connecting to WiFi network: ssid=%s", ssid)

	connName := fmt.Sprintf("dropz-%s", ssid)

	// Clean up any stale connection profiles for this SSID
	for _, name := range []string{ssid, connName} {
		exec.CommandContext(ctx, "nmcli", "connection", "delete", "id", name).Run()
	}

	// Create a proper connection profile
	addCmd := exec.CommandContext(ctx, "nmcli", "connection", "add",
		"type", "wifi",
		"con-name", connName,
		"ifname", "*",
		"ssid", ssid,
		"wifi-sec.key-mgmt", "wpa-psk",
		"wifi-sec.psk", password)

	if addOutput, addErr := addCmd.CombinedOutput(); addErr != nil {
		m.log.Errorf("Failed to create connection profile: error=%v nmcli_output=%s", addErr, string(addOutput))
		return fmt.Errorf("failed to create WiFi connection for %s: %v", ssid, addErr)
	}

	// Try to activate, retrying while the AP becomes visible
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		// Trigger a WiFi rescan so nmcli can discover the new AP
		exec.CommandContext(ctx, "nmcli", "device", "wifi", "rescan").Run()
		time.Sleep(2 * time.Second)

		upCmd := exec.CommandContext(ctx, "nmcli", "connection", "up", "id", connName)
		if output, err := upCmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%v (nmcli: %s)", err, strings.TrimSpace(string(output)))
			m.log.Debugf("WiFi activation attempt %d/10 failed: %v", attempt, lastErr)
		} else {
			m.log.Debugf("WiFi connection activated: connection=%s", connName)
			lastErr = nil
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to activate WiFi connection %s after 10 attempts: %v", connName, lastErr)
	}

	for i := 0; i < 10; i++ {
		if m.isConnectedTo(ssid) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return fmt.Errorf("timed out waiting for connection to %s", ssid)
}

// Disconnect disconnects from the current WiFi network
func (m *WiFiManager) Disconnect() error {
	// Get current active connection
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE", "connection", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		m.log.Errorf("Failed to get active WiFi connections: error=%v", err)
		return fmt.Errorf("failed to get active connections: %v", err)
	}

	// Find WiFi connections
	lines := strings.Split(string(output), "\n")
	disconnectedCount := 0

	for _, line := range lines {
		if strings.Contains(line, ":802-11-wireless") {
			parts := strings.Split(line, ":")
			if len(parts) < 1 {
				continue
			}

			ssid := parts[0]

			// Disconnect
			// Use "id" parameter to properly handle SSIDs with special characters
			cmd = exec.Command("nmcli", "connection", "down", "id", ssid)
			if output, err := cmd.CombinedOutput(); err != nil {
				// Also try with dropz- prefix
				altName := fmt.Sprintf("dropz-%s", ssid)
				cmd = exec.Command("nmcli", "connection", "down", "id", altName)
				if altOutput, altErr := cmd.CombinedOutput(); altErr != nil {
					m.log.Errorf("Failed to disconnect from WiFi network: ssid=%s error=%v nmcli_output=%s alt_error=%v alt_output=%s", 
						ssid, err, string(output), altErr, string(altOutput))
					return fmt.Errorf("failed to disconnect from %s: %v (nmcli: %s)", ssid, err, strings.TrimSpace(string(output)))
				}
			}

			disconnectedCount++
		}
	}
	_ = disconnectedCount

	return nil
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

	if err := m.SetTurboTransfer(ctx, true); err != nil {
		m.log.Warnf("Failed to enable turbo transfer: %v", err)
	} else {
		defer func() {
			if err := m.SetTurboTransfer(ctx, false); err != nil {
				m.log.Warnf("Failed to disable turbo transfer: %v", err)
			}
		}()
	}

	if daysInPast <= 0 {
		daysInPast = 7
	}

	cutoffTime := time.Now().AddDate(0, 0, -daysInPast)
	filteredMedia := filterMediaByDate(mediaFiles, cutoffTime)

	m.log.Infof("Found %d media files (%d within last %d days)", len(mediaFiles), len(filteredMedia), daysInPast)

	// Set up parallel download
	var wg sync.WaitGroup
	maxWorkers := 3 // Maximum concurrent downloads
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

// SetTurboTransfer enables or disables turbo transfer mode for faster media downloads
func (m *WiFiManager) SetTurboTransfer(ctx context.Context, enabled bool) error {
	p := "0"
	if enabled {
		p = "1"
	}
	url := fmt.Sprintf("%s%s?p=%s", GoProBaseURL, TurboTransferURL, p)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create turbo transfer request: %v", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set turbo transfer: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("turbo transfer request failed with status: %s", resp.Status)
	}
	return nil
}

// isConnectedTo checks if currently connected to the specified SSID
func (m *WiFiManager) isConnectedTo(ssid string) bool {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE,STATE", "connection", "show", "--active")
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.log.Errorf("Failed to check active WiFi connections: error=%v output=%s", err, string(output))
		return false
	}

	// Look for the SSID in active connections
	// Also check for dropz- prefixed connection name
	altName := fmt.Sprintf("dropz-%s", ssid)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && (parts[0] == ssid || parts[0] == altName) && parts[2] == "activated" {
			return true
		}
	}
	return false
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

	// Create client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Perform the request
	resp, err := client.Do(req)
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

	// Create a client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Perform the request
	resp, err := client.Do(req)
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
	chunkSize := int64(8 * 1024 * 1024) // 8MB chunks
	chunkCount := (totalSize + chunkSize - 1) / chunkSize

	// Limit to a reasonable number of chunks
	maxChunks := int64(10)
	if chunkCount > maxChunks {
		chunkSize = (totalSize + maxChunks - 1) / maxChunks
		chunkCount = (totalSize + chunkSize - 1) / chunkSize
	}

	m.log.Debugf("Downloading %s in %d chunks of size %d bytes", outputPath, chunkCount, chunkSize)

	var wg sync.WaitGroup
	errors := make(chan error, chunkCount)
	maxConcurrent := 3 // Maximum concurrent chunk downloads
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
