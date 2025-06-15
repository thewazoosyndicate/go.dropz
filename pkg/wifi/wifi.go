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

	"github.com/dropz/dropz/pkg/logger"
)

// GoPro HTTP API constants
const (
	GoProBaseURL = "http://10.5.5.9:8080"
	// Status endpoints
	StatusURL     = "/gopro/camera/state"
	BatteryURL    = "/gopro/status/battery"
	CameraInfoURL = "/gopro/camera/info"
	// Control endpoints
	SleepURL   = "/gopro/camera/control/sleep"
	ShutterURL = "/gopro/camera/control/trigger"
	ModeURL    = "/gopro/camera/mode"
	PresetURL  = "/gopro/camera/presets"
	// Media endpoints
	MediaListURL    = "/gopro/media/list"
	MediaInfoURL    = "/gopro/media/info"
	MediaHiLightURL = "/gopro/media/hilight"
	MediaDeleteURL  = "/gopro/media/delete"
	// Webcam endpoints
	WebcamStartURL  = "/gopro/webcam/start"
	WebcamStopURL   = "/gopro/webcam/stop"
	WebcamStatusURL = "/gopro/webcam/status"
	WebcamExitURL   = "/gopro/webcam/exit"
)

// WiFiManager handles WiFi operations for GoPro devices
type WiFiManager struct {
	log logger.Logger
}

// NewWiFiManager creates a new WiFi manager
func NewWiFiManager() (*WiFiManager, error) {
	return &WiFiManager{
		log: logger.GetLogger(),
	}, nil
}

// Connect connects to a GoPro WiFi network
func (m *WiFiManager) Connect(ctx context.Context, ssid, password string) error {
	m.log.Infof("Connecting to WiFi network: ssid=%s", ssid)

	// Check if connection already exists
	cmd := exec.CommandContext(ctx, "nmcli", "-t", "connection", "show", ssid)
	if err := cmd.Run(); err == nil {
		// Connection exists, just activate it
		m.log.Infof("Existing WiFi connection found, activating: ssid=%s", ssid)
		cmd = exec.CommandContext(ctx, "nmcli", "connection", "up", ssid)
		if err := cmd.Run(); err != nil {
			m.log.Errorf("Failed to activate existing WiFi connection: ssid=%s error=%v", ssid, err)
			return fmt.Errorf("failed to activate existing connection %s: %v", ssid, err)
		}
	} else {
		// Connection doesn't exist, create it
		m.log.Infof("Creating new WiFi connection: ssid=%s", ssid)
		cmd = exec.CommandContext(ctx, "nmcli", "device", "wifi", "connect", ssid, "password", password)
		if err := cmd.Run(); err != nil {
			m.log.Errorf("Failed to create WiFi connection: ssid=%s error=%v", ssid, err)
			return fmt.Errorf("failed to connect to %s: %v", ssid, err)
		}
	}

	// Verify connection with detailed progress tracking
	m.log.Tracef("Verifying WiFi connection establishment: ssid=%s max_attempts=10", ssid)
	for i := 0; i < 10; i++ {
		if m.isConnectedTo(ssid) {
			m.log.Infof("WiFi connection established successfully: ssid=%s verification_attempts=%d", ssid, i+1)
			return nil
		}

		m.log.Tracef("WiFi connection verification attempt: ssid=%s attempt=%d/10", ssid, i+1)

		select {
		case <-ctx.Done():
			m.log.Warnf("WiFi connection verification cancelled: ssid=%s context_error=%v", ssid, ctx.Err())
			return ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue checking
		}
	}

	m.log.Errorf("WiFi connection verification timeout: ssid=%s timeout=10s", ssid)
	return fmt.Errorf("timed out waiting for connection to %s", ssid)
}

// Disconnect disconnects from the current WiFi network
func (m *WiFiManager) Disconnect() error {
	m.log.Info("Initiating WiFi network disconnection")

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
			m.log.Infof("Disconnecting from WiFi network: ssid=%s", ssid)

			// Disconnect
			cmd = exec.Command("nmcli", "connection", "down", ssid)
			if err := cmd.Run(); err != nil {
				m.log.Errorf("Failed to disconnect from WiFi network: ssid=%s error=%v", ssid, err)
				return fmt.Errorf("failed to disconnect from %s: %v", ssid, err)
			}

			m.log.Infof("Successfully disconnected from WiFi network: ssid=%s", ssid)
			disconnectedCount++
		}
	}

	if disconnectedCount == 0 {
		m.log.Debug("No active WiFi connections found to disconnect")
	} else {
		m.log.Infof("WiFi disconnection completed: networks_disconnected=%d", disconnectedCount)
	}

	return nil
}

// DownloadVideos downloads videos from a GoPro device
func (m *WiFiManager) DownloadVideos(ctx context.Context, destDir string, daysInPast int) ([]string, error) {
	m.log.Infof("Starting video download: destination=%s days_past=%d", destDir, daysInPast)

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		m.log.Errorf("Failed to create destination directory: path=%s error=%v", destDir, err)
		return nil, fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Get media list from GoPro
	m.log.Debug("Retrieving media list from GoPro device")
	mediaFiles, err := m.getMediaList(ctx)
	if err != nil {
		m.log.Errorf("Failed to retrieve media list from GoPro: error=%v", err)
		return nil, fmt.Errorf("failed to get media list: %v", err)
	}

	m.log.Infof("Media list retrieved: total_files=%d", len(mediaFiles))

	// If no media files found, return early
	if len(mediaFiles) == 0 {
		m.log.Info("No media files found on GoPro device")
		return nil, nil
	}
	// If no days specified, default to 7 days
	if daysInPast <= 0 {
		daysInPast = 7
		m.log.Debugf("No days specified, defaulting to %d days", daysInPast)
	}

	// Filter by date
	cutoffTime := time.Now().AddDate(0, 0, -daysInPast)
	filteredMedia := filterMediaByDate(mediaFiles, cutoffTime)

	m.log.Debugf("Found %d media files within the last %d days", len(filteredMedia), daysInPast)

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
		exists, err := m.fileExistsWithSize(outputPath, media.Size)
		if err != nil {
			m.log.Warnf("Error checking if file exists: file=%s error=%v", media.Name, err)
		}
		if exists {
			m.log.Infof("File already exists with correct size, skipping download: file=%s size=%d bytes",
				media.Name, media.Size)
			mu.Lock()
			downloadedFiles = append(downloadedFiles, outputPath)
			skippedCount++
			mu.Unlock()
			m.log.Debugf("Added existing file to results: total_files_so_far=%d skipped_so_far=%d",
				len(downloadedFiles), skippedCount)
			continue
		}

		// Check if this file is already being downloaded by another goroutine
		downloadMutex.Lock()
		if downloadingFiles[media.Name] {
			m.log.Infof("File %s is already being downloaded by another operation, skipping to avoid duplicate download", media.Name)
			downloadMutex.Unlock()
			continue
		}
		// Mark this file as being downloaded
		downloadingFiles[media.Name] = true
		m.log.Tracef("Marked file %s as being downloaded to prevent concurrent downloads", media.Name)
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
			existsDouble, errDouble := m.fileExistsWithSize(outputPath, media.Size)
			if errDouble != nil {
				m.log.Warnf("Error double-checking file existence for %s: %v", media.Name, errDouble)
			}
			if existsDouble {
				m.log.Infof("File already exists with correct size (double-check), skipping download: file=%s size=%d bytes",
					media.Name, media.Size)
				mu.Lock()
				downloadedFiles = append(downloadedFiles, outputPath)
				skippedCount++
				mu.Unlock()
				return
			}

			m.log.Debugf("Downloading %s to %s", media.Name, outputPath)

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
			m.log.Debugf("Successfully downloaded %s", media.Name)
		}(media, outputPath)
	}

	// Wait for all downloads to complete
	wg.Wait()

	actualDownloads := len(downloadedFiles) - skippedCount
	if skippedCount > 0 {
		m.log.Infof("Video download completed: total_files=%d downloaded=%d skipped=%d destination=%s",
			len(downloadedFiles), actualDownloads, skippedCount, destDir)
	} else {
		m.log.Infof("Video download completed: total_downloaded=%d destination=%s", len(downloadedFiles), destDir)
	}
	return downloadedFiles, nil
}

// isConnectedTo checks if currently connected to the specified SSID
func (m *WiFiManager) isConnectedTo(ssid string) bool {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE,STATE", "connection", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		m.log.Errorf("Failed to check active WiFi connections: error=%v", err)
		return false
	}

	// Look for the SSID in active connections
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[0] == ssid && parts[2] == "activated" {
			m.log.Tracef("WiFi connection confirmed active: ssid=%s device=%s state=%s", ssid, parts[1], parts[2])
			return true
		}
	}

	m.log.Tracef("WiFi connection not found in active connections: ssid=%s", ssid)
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
	CreatedAt  json.Number `json:"cre,string"` // Support both string and number
	ModifiedAt json.Number `json:"mod,string"` // Support both string and number
	Size       json.Number `json:"s,string"`   // Support both string and number

	// Optional fields
	RawFlag    json.Number `json:"raw,string,omitempty"`
	LastStatus json.Number `json:"ls,string,omitempty"`
	GLRVersion json.Number `json:"glrv,string,omitempty"`

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
	m.log.Debug("Retrieving camera status via HTTP API")

	url := fmt.Sprintf("%s%s", GoProBaseURL, StatusURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.log.Errorf("Failed to create camera status request: error=%v", err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		m.log.Errorf("Camera status request failed: error=%v", err)
		return nil, fmt.Errorf("failed to get camera status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.log.Errorf("Camera status request returned error: status_code=%d status=%s", resp.StatusCode, resp.Status)
		return nil, fmt.Errorf("camera status request failed with status: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		m.log.Errorf("Failed to decode camera status response: error=%v", err)
		return nil, fmt.Errorf("failed to decode camera status: %v", err)
	}

	m.log.Tracef("Camera status retrieved successfully: status_fields=%d", len(result))
	return result, nil
}

// GetCameraInfo retrieves the camera information via HTTP API
func (m *WiFiManager) GetCameraInfo(ctx context.Context) (map[string]interface{}, error) {
	m.log.Debug("Retrieving camera information via HTTP API")

	url := fmt.Sprintf("%s%s", GoProBaseURL, CameraInfoURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.log.Errorf("Failed to create camera info request: error=%v", err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		m.log.Errorf("Camera info request failed: error=%v", err)
		return nil, fmt.Errorf("failed to get camera info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.log.Errorf("Camera info request returned error: status_code=%d status=%s", resp.StatusCode, resp.Status)
		return nil, fmt.Errorf("camera info request failed with status: %s", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		m.log.Errorf("Failed to decode camera info response: error=%v", err)
		return nil, fmt.Errorf("failed to decode camera info: %v", err)
	}

	m.log.Tracef("Camera info retrieved successfully: info_fields=%d", len(result))
	return result, nil
}

// TriggerShutter starts or stops capturing depending on the current mode
func (m *WiFiManager) TriggerShutter(ctx context.Context, start bool) error {
	m.log.Debugf("Trigger shutter: start=%v", start)

	url := fmt.Sprintf("%s%s", GoProBaseURL, ShutterURL)

	// Prepare the request body
	requestBody := fmt.Sprintf(`{"start": %t}`, start)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to trigger shutter: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("shutter request failed with status: %s", resp.Status)
	}

	return nil
}

// SetMode sets the camera mode
func (m *WiFiManager) SetMode(ctx context.Context, modeId int) error {
	m.log.Infof("Setting camera mode: mode_id=%d", modeId)

	url := fmt.Sprintf("%s%s", GoProBaseURL, ModeURL)

	// Prepare the request body
	requestBody := fmt.Sprintf(`{"mode_id": %d}`, modeId)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(requestBody))
	if err != nil {
		m.log.Errorf("Failed to create mode set request: mode_id=%d error=%v", modeId, err)
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		m.log.Errorf("Mode set request failed: mode_id=%d error=%v", modeId, err)
		return fmt.Errorf("failed to set mode: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.log.Errorf("Mode set request returned error: mode_id=%d status_code=%d status=%s", modeId, resp.StatusCode, resp.Status)
		return fmt.Errorf("set mode request failed with status: %s", resp.Status)
	}

	m.log.Infof("Camera mode set successfully: mode_id=%d", modeId)
	return nil
}

// SleepCamera puts the camera to sleep
func (m *WiFiManager) SleepCamera(ctx context.Context) error {
	m.log.Info("Putting camera to sleep via HTTP API")

	url := fmt.Sprintf("%s%s", GoProBaseURL, SleepURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		m.log.Errorf("Failed to create camera sleep request: error=%v", err)
		return fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		m.log.Errorf("Camera sleep request failed: error=%v", err)
		return fmt.Errorf("failed to put camera to sleep: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		m.log.Errorf("Camera sleep request returned error: status_code=%d status=%s", resp.StatusCode, resp.Status)
		return fmt.Errorf("sleep request failed with status: %s", resp.Status)
	}

	m.log.Info("Camera sleep command sent successfully")
	return nil
}

// getMediaList retrieves the list of media files from a GoPro device via HTTP API
func (m *WiFiManager) getMediaList(ctx context.Context) ([]MediaFile, error) {
	m.log.Trace("Getting media list from GoPro")

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

	// For debugging
	responseBytes, _ := json.Marshal(mediaList)
	m.log.Tracef("Received media list response: %s", string(responseBytes))

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

	m.log.Debugf("Found %d media files", len(result))
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
	logger     logger.Logger
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

// WebcamState represents the current state of the webcam
type WebcamState struct {
	Active bool                   `json:"active"`
	Status string                 `json:"status"`
	Error  string                 `json:"error,omitempty"`
	Info   map[string]interface{} `json:"info,omitempty"`
}

// StartWebcam starts the webcam mode
func (m *WiFiManager) StartWebcam(ctx context.Context) (*WebcamState, error) {
	m.log.Debug("Starting webcam mode")

	url := fmt.Sprintf("%s%s", GoProBaseURL, WebcamStartURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to start webcam: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("start webcam request failed with status: %s", resp.Status)
	}

	var state WebcamState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode webcam state: %v", err)
	}

	return &state, nil
}

// StopWebcam stops the webcam mode
func (m *WiFiManager) StopWebcam(ctx context.Context) (*WebcamState, error) {
	m.log.Debug("Stopping webcam mode")

	url := fmt.Sprintf("%s%s", GoProBaseURL, WebcamStopURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to stop webcam: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stop webcam request failed with status: %s", resp.Status)
	}

	var state WebcamState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode webcam state: %v", err)
	}

	return &state, nil
}

// GetWebcamStatus retrieves the current webcam status
func (m *WiFiManager) GetWebcamStatus(ctx context.Context) (*WebcamState, error) {
	m.log.Trace("Getting webcam status")

	url := fmt.Sprintf("%s%s", GoProBaseURL, WebcamStatusURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get webcam status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get webcam status request failed with status: %s", resp.Status)
	}

	var state WebcamState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return nil, fmt.Errorf("failed to decode webcam state: %v", err)
	}

	return &state, nil
}

// ExitWebcam exits webcam mode
func (m *WiFiManager) ExitWebcam(ctx context.Context) error {
	m.log.Debug("Exiting webcam mode")

	url := fmt.Sprintf("%s%s", GoProBaseURL, WebcamExitURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exit webcam mode: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("exit webcam request failed with status: %s", resp.Status)
	}

	return nil
}

// fileExistsWithSize checks if a file exists and has the expected size
func (m *WiFiManager) fileExistsWithSize(filePath string, expectedSize int64) (bool, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.log.Debugf("File does not exist: file=%s - WILL DOWNLOAD", filepath.Base(filePath))
			// Try searching in immediate subdirectories
			rootDir := filepath.Dir(filePath)
			pattern := filepath.Join(rootDir, "*", filepath.Base(filePath))
			matches, _ := filepath.Glob(pattern)
			for _, p := range matches {
				if info, e := os.Stat(p); e == nil && info.Size() == expectedSize {
					m.log.Infof("File exists in subfolder, skipping download: file=%s path=%s size=%d bytes",
						filepath.Base(filePath), p, expectedSize)
					return true, nil
				}
			}
			return false, nil // File doesn't exist
		}
		m.log.Warnf("Error checking file existence: file=%s error=%v", filepath.Base(filePath), err)
		return false, err
	}

	if fi.Size() == expectedSize {
		m.log.Infof("File exists with correct size: file=%s size=%d bytes - WILL SKIP",
			filepath.Base(filePath), expectedSize)
		return true, nil
	}

	m.log.Infof("File exists but size mismatch: file=%s actual_size=%d expected_size=%d - WILL DOWNLOAD",
		filepath.Base(filePath), fi.Size(), expectedSize)
	return false, nil
}
