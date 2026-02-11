package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/dropz/dropz/pkg/protocol"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Camera represents a camera's basic information
type Camera struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Alias        string `json:"alias"`
	MACAddress   string `json:"mac_address"`
	WiFiSSID     string `json:"wifi_ssid"`
	WiFiPassword string `json:"wifi_password"`
	RSSI         int32  `json:"rssi"`
}

// CameraStatus represents the status of a camera
type CameraStatus struct {
	LastSeen    time.Time `json:"last_seen"`
	LastSynced  time.Time `json:"last_synced"`
	IsPairing   bool      `json:"is_pairing"`
	IsPaired    bool      `json:"is_paired"`
	IsManaged   bool      `json:"is_managed"`
	IsReachable bool      `json:"is_reachable"`
	IsSynced    bool      `json:"is_synced"`
	IsSyncing   bool      `json:"is_syncing"`
}

// CameraMetadata represents technical details about a camera
type CameraMetadata struct {
	ID              string `json:"id"` // References the camera ID
	FirmwareVersion string `json:"firmware_version"`
	Model           string `json:"model"`
	SerialNumber    string `json:"serial_number"`
	BatteryLevel    int32  `json:"battery_level"` // percentage
	HardwareVersion string `json:"hardware_version"`
}

// CameraWithState represents a complete camera with all its information
type CameraWithState struct {
	Camera   Camera         `json:"camera"`
	Status   CameraStatus   `json:"status"`
	GroupID  string         `json:"group_id"`
	Metadata CameraMetadata `json:"metadata"`
}

// DiscoveredCamera represents a camera that is not managed or paired
type DiscoveredCamera struct {
	CameraState *CameraWithState `json:"camera_state"`
}

// ManagedCamera represents a camera in the management pool
type ManagedCamera struct {
	CameraState *CameraWithState `json:"camera_state"`
}

// SyncQueueEntry represents a record in the sync queue
type SyncQueueEntry struct {
	CameraID         string    `json:"camera_id"` // References the camera ID
	QueuedAt         time.Time `json:"queued_at"`
	Priority         int32     `json:"priority"` // Higher numbers = higher priority (manual sync = 10, auto sync = 5)
	ProgressPercent  int32     `json:"progress_percent"`
	CurrentOperation string    `json:"current_operation"`
}

// Group represents a collection of cameras that can be managed together
type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CameraIDs []string  `json:"camera_ids"` // References to managed cameras
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VideoFile represents metadata about a synchronized video file
type VideoFile struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Path            string    `json:"path"`
	SizeBytes       int64     `json:"size_bytes"`
	CreatedAt       time.Time `json:"created_at"`
	SyncedAt        time.Time `json:"synced_at"`
	CameraID        string    `json:"camera_id"`
	MimeType        string    `json:"mime_type"`
	DurationSeconds int32     `json:"duration_seconds"`
	ThumbnailPath   string    `json:"thumbnail_path"`
	HasProcessed    bool      `json:"has_processed"`
}

// LogEntry represents a system log entry
type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Level      string    `json:"level"`
	Message    string    `json:"message"`
	CameraID   string    `json:"camera_id"`
	Component  string    `json:"component"`
	SourceFile string    `json:"source_file"`
	LineNumber int32     `json:"line_number"`
}

// Config represents the system-wide configuration settings
type Config struct {
	PairModeEnabled               bool      `json:"pair_mode_enabled"`
	SyncEnabled                   bool      `json:"sync_enabled"`
	ScanIntervalSeconds           int32     `json:"scan_interval_seconds"`
	ConnectTimeoutSeconds         int32     `json:"connect_timeout_seconds"`
	DaysThreshold                 int32     `json:"days_threshold"`
	DestinationFolder             string    `json:"destination_folder"`
	InactivityTimeoutSeconds      int32     `json:"inactivity_timeout_seconds"`
	InactivitySyncIntervalSeconds int32     `json:"inactivity_sync_interval_seconds"`
	SetTimeEnabled                bool      `json:"set_time_enabled"`
	LogLevel                      string    `json:"log_level"`
	LastUpdated                   time.Time `json:"last_updated"`
}

// Database represents the in-memory database with JSON persistence
type Database struct {
	CameraStates map[string]*CameraWithState `json:"camera_states"`
	SyncQueue    []*SyncQueueEntry           `json:"sync_queue"`
	Groups       []*Group                    `json:"groups"`
	Videos       []*VideoFile                `json:"videos"`
	Logs         []*LogEntry                 `json:"logs"`
	Config       Config                      `json:"config"`
	filePath     string
	mutex        sync.RWMutex
	log          *logrus.Logger
}

var instance *Database
var once sync.Once

// GetDatabase returns the singleton database instance
func GetDatabase() *Database {
	once.Do(func() {
		instance = &Database{
			CameraStates: make(map[string]*CameraWithState),
			SyncQueue:    make([]*SyncQueueEntry, 0),
			Groups:       make([]*Group, 0),
			Videos:       make([]*VideoFile, 0),
			Logs:         make([]*LogEntry, 0),
			Config:       DefaultConfig(),
			log:          logrus.New(),
		}
	})
	return instance
}

// SetLogger sets the logger for the database
func (db *Database) SetLogger(log *logrus.Logger) {
	db.log = log
}

// Initialize sets up the database with the specified file path
func (db *Database) Initialize(filePath string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.filePath = filePath

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create a new file with default values
		if err := db.saveToFile(); err != nil {
			return fmt.Errorf("failed to create database file: %v", err)
		}
		db.log.WithFields(logrus.Fields{"path": filePath}).Info("Created new database file")
		return nil
	}

	// Load existing database
	return db.loadFromFile()
}

// loadFromFile loads the database from the JSON file
func (db *Database) loadFromFile() error {
	data, err := os.ReadFile(db.filePath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %v", err)
	}

	if err := json.Unmarshal(data, db); err != nil {
		return fmt.Errorf("failed to unmarshal database: %v", err)
	}

	db.log.WithFields(logrus.Fields{"path": db.filePath, "cameras_count": len(db.CameraStates), "groups_count": len(db.Groups), "videos_count": len(db.Videos)}).Info("Loaded database from file")
	return nil
}

// saveToFile saves the database to the JSON file
func (db *Database) saveToFile() error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal database: %v", err)
	}

	if err := os.WriteFile(db.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write database file: %v", err)
	}

	return nil
}

// SaveChanges persists changes to the database file
func (db *Database) SaveChanges() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.log.Trace("Saving database changes", "path", db.filePath)

	startTime := time.Now()
	err := db.saveToFile()

	if err != nil {
		db.log.Error("Failed to save database", "error", err, "path", db.filePath)
	} else {
		db.log.Debug("Database saved successfully", "path", db.filePath, "duration", time.Since(startTime))
	}

	return err
}

// AddOrUpdateDiscoveredCamera adds or updates a discovered camera
func (db *Database) AddOrUpdateDiscoveredCamera(camera *DiscoveredCamera) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if this is a new camera or an update to an existing one
	_, exists := db.CameraStates[camera.CameraState.Camera.MACAddress]

	// Log camera updates
	if !exists {
		db.log.Info("New discovered camera added",
			"camera_name", camera.CameraState.Camera.Name,
			"mac_address", camera.CameraState.Camera.MACAddress,
			"rssi", camera.CameraState.Camera.RSSI)
	} else {
		db.log.Trace("Discovered camera updated",
			"camera_name", camera.CameraState.Camera.Name,
			"mac_address", camera.CameraState.Camera.MACAddress,
			"rssi", camera.CameraState.Camera.RSSI)
	}

	// Store the camera in the database
	db.CameraStates[camera.CameraState.Camera.MACAddress] = camera.CameraState

	return db.saveToFile()
}

// GetDiscoveredCamera retrieves a discovered camera by MAC address
func (db *Database) GetDiscoveredCamera(macAddress string) (*DiscoveredCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return nil, false
	}
	return &DiscoveredCamera{CameraState: cameraState}, exists
}

// GetManagedCamera retrieves a managed camera by MAC address
func (db *Database) GetManagedCamera(macAddress string) (*ManagedCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return nil, false
	}
	// Only return if it's both managed and paired
	if cameraState.Status.IsManaged && cameraState.Status.IsPaired {
		return &ManagedCamera{CameraState: cameraState}, true
	}
	return nil, false
}

// GetCameraByID retrieves a camera by its ID
func (db *Database) GetCameraByID(id string) (*CameraWithState, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, cameraState := range db.CameraStates {
		if cameraState.Camera.ID == id {
			return cameraState, true
		}
	}
	return nil, false
}

// GetManagedCameraByID retrieves a managed camera by ID
func (db *Database) GetManagedCameraByID(id string) (*ManagedCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, cameraState := range db.CameraStates {
		if cameraState.Camera.ID == id && cameraState.Status.IsManaged && cameraState.Status.IsPaired {
			return &ManagedCamera{CameraState: cameraState}, true
		}
	}
	return nil, false
}

// RemoveManagedCamera removes a camera from the managed pool
func (db *Database) RemoveManagedCamera(macAddress string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if exists {
		// Just update the camera status, don't remove it from the database
		cameraState.Status.IsManaged = false
		db.log.WithFields(logrus.Fields{"mac_address": macAddress, "camera_name": cameraState.Camera.Name}).Info("Camera removed from managed pool")
		return db.saveToFile()
	}
	db.log.Warn("Attempted to remove non-existent managed camera", "mac_address", macAddress)
	return fmt.Errorf("camera with MAC address %s not found", macAddress)
}

// AddSyncQueueEntry adds a camera to the sync queue
func (db *Database) AddSyncQueueEntry(entry *SyncQueueEntry) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if the camera is already in the queue
	for _, existing := range db.SyncQueue {
		if existing.CameraID == entry.CameraID {
			// Already in queue, update it
			*existing = *entry
			db.log.Debug("Sync queue entry updated", "camera_id", entry.CameraID, "operation", entry.CurrentOperation, "progress", entry.ProgressPercent)
			return db.saveToFile()
		}
	}

	// Add to queue
	db.SyncQueue = append(db.SyncQueue, entry)
	db.log.WithFields(logrus.Fields{"camera_id": entry.CameraID, "operation": entry.CurrentOperation, "queue_length": len(db.SyncQueue)}).Info("Camera added to sync queue")
	return db.saveToFile()
}

// GetSyncQueue returns the current sync queue, sorted by priority (highest first) then by queued time
func (db *Database) GetSyncQueue() []*SyncQueueEntry {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	queue := make([]*SyncQueueEntry, len(db.SyncQueue))
	copy(queue, db.SyncQueue)

	// Sort by priority (highest first), then by queued time (oldest first)
	sort.Slice(queue, func(i, j int) bool {
		if queue[i].Priority != queue[j].Priority {
			return queue[i].Priority > queue[j].Priority // Higher priority first
		}
		return queue[i].QueuedAt.Before(queue[j].QueuedAt) // Older entries first for same priority
	})

	return queue
}

// UpdateSyncQueueEntry updates an existing sync queue entry
func (db *Database) UpdateSyncQueueEntry(entry *SyncQueueEntry) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, existing := range db.SyncQueue {
		if existing.CameraID == entry.CameraID {
			db.SyncQueue[i] = entry
			db.log.Debug("Sync queue entry progress updated", "camera_id", entry.CameraID, "operation", entry.CurrentOperation, "progress", entry.ProgressPercent)
			return db.saveToFile()
		}
	}
	db.log.Warn("Attempted to update non-existent sync queue entry", "camera_id", entry.CameraID)
	return fmt.Errorf("sync queue entry for camera %s not found", entry.CameraID)
}

// RemoveSyncQueueEntry removes an entry from the sync queue
func (db *Database) RemoveSyncQueueEntry(cameraID string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, entry := range db.SyncQueue {
		if entry.CameraID == cameraID {
			// Remove entry by replacing it with the last one and truncating
			db.SyncQueue[i] = db.SyncQueue[len(db.SyncQueue)-1]
			db.SyncQueue = db.SyncQueue[:len(db.SyncQueue)-1]
			db.log.WithFields(logrus.Fields{"camera_id": cameraID, "queue_length": len(db.SyncQueue)}).Info("Camera removed from sync queue")
			return db.saveToFile()
		}
	}
	db.log.Trace("Camera not found in sync queue for removal", "camera_id", cameraID)
	return nil // Not found, not an error
}

// AddOrUpdateGroup adds or updates a group
func (db *Database) AddOrUpdateGroup(group *Group) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, existing := range db.Groups {
		if existing.ID == group.ID {
			db.Groups[i] = group
			db.log.WithFields(logrus.Fields{"group_id": group.ID, "group_name": group.Name, "camera_count": len(group.CameraIDs)}).Info("Group updated")
			return db.saveToFile()
		}
	}

	// Add new group
	db.Groups = append(db.Groups, group)
	db.log.WithFields(logrus.Fields{"group_id": group.ID, "group_name": group.Name, "camera_count": len(group.CameraIDs), "total_groups": len(db.Groups)}).Info("Group created")
	return db.saveToFile()
}

// GetGroup returns a group by ID
func (db *Database) GetGroup(id string) (*Group, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, group := range db.Groups {
		if group.ID == id {
			return group, true
		}
	}
	return nil, false
}

// GetAllGroups returns all groups
func (db *Database) GetAllGroups() []*Group {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*Group, len(db.Groups))
	copy(result, db.Groups)
	return result
}

// RemoveGroup removes a group
func (db *Database) RemoveGroup(id string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, group := range db.Groups {
		if group.ID == id {
			// Remove group by replacing it with the last one and truncating
			db.Groups[i] = db.Groups[len(db.Groups)-1]
			db.Groups = db.Groups[:len(db.Groups)-1]
			db.log.WithFields(logrus.Fields{"group_id": id, "group_name": group.Name, "remaining_groups": len(db.Groups)}).Info("Group removed")
			return db.saveToFile()
		}
	}
	db.log.Warn("Attempted to remove non-existent group", "group_id", id)
	return fmt.Errorf("group with ID %s not found", id)
}

// AddVideo adds a video file
func (db *Database) AddVideo(video *VideoFile) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.Videos = append(db.Videos, video)
	db.log.WithFields(logrus.Fields{"video_name": video.Name, "camera_id": video.CameraID, "size_bytes": video.SizeBytes, "duration_seconds": video.DurationSeconds, "total_videos": len(db.Videos)}).Info("Video file added")
	return db.saveToFile()
}

// GetVideosByCamera returns videos for a specific camera
func (db *Database) GetVideosByCamera(cameraID string) []*VideoFile {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	var result []*VideoFile
	for _, video := range db.Videos {
		if video.CameraID == cameraID {
			result = append(result, video)
		}
	}
	return result
}

// AddLogEntry adds a log entry
func (db *Database) AddLogEntry(entry *LogEntry) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.Logs = append(db.Logs, entry)

	// Limit log entries to avoid excessive memory usage
	if len(db.Logs) > 10000 {
		db.Logs = db.Logs[len(db.Logs)-10000:]
		db.log.Trace("Log entries trimmed to limit memory usage", "max_entries", 10000, "current_entries", len(db.Logs))
	}

	return nil // Don't save to file for every log entry
}

// GetLogs returns log entries filtered by criteria
func (db *Database) GetLogs(level, cameraID, component string, startTime, endTime time.Time, limit, offset int) ([]*LogEntry, int) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	var filtered []*LogEntry
	for _, entry := range db.Logs {
		if level != "" && entry.Level != level {
			continue
		}
		if cameraID != "" && entry.CameraID != cameraID {
			continue
		}
		if component != "" && entry.Component != component {
			continue
		}
		if !startTime.IsZero() && entry.Timestamp.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && entry.Timestamp.After(endTime) {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Calculate total
	total := len(filtered)

	// Apply pagination
	if offset >= total {
		return []*LogEntry{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return filtered[offset:end], total
}

// GetConfig returns the current configuration
func (db *Database) GetConfig() Config {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.Config
}

// UpdateConfig updates the system configuration
func (db *Database) UpdateConfig(config Config) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.Config = config
	db.Config.LastUpdated = time.Now()
	return db.saveToFile()
}

// ToProtoCameraWithState converts a CameraWithState to protocol CameraWithState
func (c *CameraWithState) ToProtoCameraWithState() *protocol.CameraWithState {
	return &protocol.CameraWithState{
		Camera: &protocol.Camera{
			Id:           c.Camera.ID,
			Name:         c.Camera.Name,
			Alias:        c.Camera.Alias,
			MacAddress:   c.Camera.MACAddress,
			WifiSsid:     c.Camera.WiFiSSID,
			WifiPassword: c.Camera.WiFiPassword,
			Rssi:         c.Camera.RSSI,
		},
		Status: &protocol.CameraStatus{
			LastSeen:    timestamppb.New(c.Status.LastSeen),
			LastSynced:  timestamppb.New(c.Status.LastSynced),
			IsPairing:   c.Status.IsPairing,
			IsPaired:    c.Status.IsPaired,
			IsManaged:   c.Status.IsManaged,
			IsReachable: c.Status.IsReachable,
			IsSynced:    c.Status.IsSynced,
			IsSyncing:   c.Status.IsSyncing,
		},
		GroupId: c.GroupID,
		Metadata: &protocol.CameraMetadata{
			Id:              c.Metadata.ID,
			FirmwareVersion: c.Metadata.FirmwareVersion,
			Model:           c.Metadata.Model,
			SerialNumber:    c.Metadata.SerialNumber,
			BatteryLevel:    c.Metadata.BatteryLevel,
			HardwareVersion: c.Metadata.HardwareVersion,
		},
	}
}

// ToProtoDiscoveredCamera converts a DiscoveredCamera to protocol DiscoveredCamera
func (c *DiscoveredCamera) ToProtoDiscoveredCamera() *protocol.DiscoveredCamera {
	return &protocol.DiscoveredCamera{
		CameraState: c.CameraState.ToProtoCameraWithState(),
	}
}

// ToProtoManagedCamera converts a ManagedCamera to protocol ManagedCamera
func (c *ManagedCamera) ToProtoManagedCamera() *protocol.ManagedCamera {
	return &protocol.ManagedCamera{
		CameraState: c.CameraState.ToProtoCameraWithState(),
	}
}

// ToProtoSyncQueueEntry converts a SyncQueueEntry to protocol SyncQueueEntry
func (e *SyncQueueEntry) ToProtoSyncQueueEntry() *protocol.SyncQueueEntry {
	return &protocol.SyncQueueEntry{
		CameraId:         e.CameraID,
		QueuedAt:         timestamppb.New(e.QueuedAt),
		Priority:         e.Priority,
		ProgressPercent:  e.ProgressPercent,
		CurrentOperation: e.CurrentOperation,
	}
}

// ToProtoGroup converts a Group to protocol Group
func (g *Group) ToProtoGroup() *protocol.Group {
	protoGroup := &protocol.Group{
		Id:        g.ID,
		Name:      g.Name,
		CameraIds: make([]string, len(g.CameraIDs)),
		CreatedAt: timestamppb.New(g.CreatedAt),
		UpdatedAt: timestamppb.New(g.UpdatedAt),
	}
	copy(protoGroup.CameraIds, g.CameraIDs)
	return protoGroup
}

// ToProtoVideoFile converts a VideoFile to protocol VideoFile
func (v *VideoFile) ToProtoVideoFile() *protocol.VideoFile {
	return &protocol.VideoFile{
		Id:              v.ID,
		Name:            v.Name,
		Path:            v.Path,
		SizeBytes:       v.SizeBytes,
		CreatedAt:       timestamppb.New(v.CreatedAt),
		SyncedAt:        timestamppb.New(v.SyncedAt),
		CameraId:        v.CameraID,
		MimeType:        v.MimeType,
		DurationSeconds: v.DurationSeconds,
		ThumbnailPath:   v.ThumbnailPath,
		HasProcessed:    v.HasProcessed,
	}
}

// ToProtoConfig converts a Config to protocol Config
func (c *Config) ToProtoConfig() *protocol.Config {
	return &protocol.Config{
		PairModeEnabled:               c.PairModeEnabled,
		SyncEnabled:                   c.SyncEnabled,
		ScanIntervalSeconds:           c.ScanIntervalSeconds,
		ConnectTimeoutSeconds:         c.ConnectTimeoutSeconds,
		DaysThreshold:                 c.DaysThreshold,
		DestinationFolder:             c.DestinationFolder,
		InactivityTimeoutSeconds:      c.InactivityTimeoutSeconds,
		InactivitySyncIntervalSeconds: c.InactivitySyncIntervalSeconds,
		SetTimeEnabled:                c.SetTimeEnabled,
		LogLevel:                      c.LogLevel,
		LastUpdated:                   timestamppb.New(c.LastUpdated),
	}
}

// ToProtoLogEntry converts a LogEntry to protocol LogEntry
func (l *LogEntry) ToProtoLogEntry() *protocol.LogEntry {
	return &protocol.LogEntry{
		Timestamp:  timestamppb.New(l.Timestamp),
		Level:      l.Level,
		Message:    l.Message,
		CameraId:   l.CameraID,
		Component:  l.Component,
		SourceFile: l.SourceFile,
		LineNumber: l.LineNumber,
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return Config{
		PairModeEnabled:               false,
		SyncEnabled:                   true,
		ScanIntervalSeconds:           30,
		ConnectTimeoutSeconds:         60,
		DaysThreshold:                 1,
		DestinationFolder:             filepath.Join(homeDir, "Videos", "dropz"),
		InactivityTimeoutSeconds:      60,
		InactivitySyncIntervalSeconds: 600,
		SetTimeEnabled:                true,
		LogLevel:                      "info",
		LastUpdated:                   time.Now(),
	}
}

// updateCameraStatus is a helper that looks up a camera by MAC and applies a mutation.
func (db *Database) updateCameraStatus(macAddress string, mutate func(*CameraStatus) bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	if mutate(&cameraState.Status) {
		return db.saveToFile()
	}
	return nil
}

// UpdateCameraPairingStatus updates a camera's pairing status
func (db *Database) UpdateCameraPairingStatus(macAddress string, isPairing bool) error {
	return db.updateCameraStatus(macAddress, func(s *CameraStatus) bool {
		s.IsPairing = isPairing
		return true
	})
}

// SetCameraPaired marks a camera as paired
func (db *Database) SetCameraPaired(macAddress string, isPaired bool) error {
	return db.updateCameraStatus(macAddress, func(s *CameraStatus) bool {
		s.IsPaired = isPaired
		s.IsPairing = false
		return true
	})
}

// SetCameraMetadata stores hardware metadata for a camera
func (db *Database) SetCameraMetadata(macAddress string, metadata CameraMetadata) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if state, exists := db.CameraStates[macAddress]; exists {
		state.Metadata = metadata
		return db.saveToFile()
	}
	return fmt.Errorf("camera not found: %s", macAddress)
}

// ToggleCameraManaged toggles a camera's managed status
func (db *Database) ToggleCameraManaged(macAddress string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	cameraState.Status.IsManaged = !cameraState.Status.IsManaged
	return db.saveToFile()
}

// UpdateCameraSyncingStatus updates a camera's syncing status
func (db *Database) UpdateCameraSyncingStatus(macAddress string, isSyncing bool) error {
	return db.updateCameraStatus(macAddress, func(s *CameraStatus) bool {
		s.IsSyncing = isSyncing
		return true
	})
}

// MarkCameraSynced marks a camera as synced and updates last_synced timestamp
func (db *Database) MarkCameraSynced(macAddress string) error {
	return db.updateCameraStatus(macAddress, func(s *CameraStatus) bool {
		s.IsSynced = true
		s.IsSyncing = false
		s.LastSynced = time.Now()
		return true
	})
}

// ResetSyncStatus marks a camera as not synced (to trigger re-sync)
func (db *Database) ResetSyncStatus(macAddress string) error {
	return db.updateCameraStatus(macAddress, func(s *CameraStatus) bool {
		s.IsSynced = false
		return true
	})
}

// ResetTransientStates resets transient states (is_syncing, is_pairing) on app start
func (db *Database) ResetTransientStates() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	changesMade := false
	for _, cameraState := range db.CameraStates {
		if cameraState.Status.IsSyncing || cameraState.Status.IsPairing {
			cameraState.Status.IsSyncing = false
			cameraState.Status.IsPairing = false
			changesMade = true
		}
	}

	if changesMade {
		return db.saveToFile()
	}

	return nil
}

// GetCamerasForDiscoveredPool returns cameras that should appear in the Discovered pool
func (db *Database) GetCamerasForDiscoveredPool() []*DiscoveredCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*DiscoveredCamera, 0)

	// Log all cameras and their states for debugging
	db.log.Tracef("GetCamerasForDiscoveredPool: examining %d total cameras", len(db.CameraStates))

	for macAddress, cameraState := range db.CameraStates {
		db.log.Tracef("Camera %s (%s): Reachable=%v, Paired=%v, Managed=%v",
			cameraState.Camera.Name, macAddress,
			cameraState.Status.IsReachable,
			cameraState.Status.IsPaired,
			cameraState.Status.IsManaged)

		// Camera must be reachable and NOT paired (managed cameras still appear until paired)
		// Camera appears in Discovered pool if either not paired or not managed (reachability not required)
		if !cameraState.Status.IsPaired || !cameraState.Status.IsManaged {
			cameras = append(cameras, &DiscoveredCamera{CameraState: cameraState})
			db.log.Tracef("✓ Camera %s INCLUDED in discovered pool", cameraState.Camera.Name)
		} else {
			db.log.Tracef("✗ Camera %s EXCLUDED from discovered pool (criteria not met)", cameraState.Camera.Name)
		}
	}

	db.log.Tracef("GetCamerasForDiscoveredPool returning %d cameras", len(cameras))
	return cameras
}

// GetCamerasForManagedPool returns cameras that should appear in the Managed pool
func (db *Database) GetCamerasForManagedPool() []*ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*ManagedCamera, 0)
	for _, cameraState := range db.CameraStates {
		// Camera must be paired and managed (reachability is not required for managed pool)
		// Camera appears in Managed pool if paired and managed
		if cameraState.Status.IsPaired && cameraState.Status.IsManaged {
			cameras = append(cameras, &ManagedCamera{CameraState: cameraState})
		}
	}

	// Log debug info about managed pool criteria
	db.log.Tracef("GetCamerasForManagedPool returning %d cameras", len(cameras))
	for _, camera := range cameras {
		db.log.Tracef("Camera in managed pool: %s (Paired=%v, Managed=%v, Reachable=%v)",
			camera.CameraState.Camera.Name,
			camera.CameraState.Status.IsPaired,
			camera.CameraState.Status.IsManaged,
			camera.CameraState.Status.IsReachable)
	}

	return cameras
}

// GetCamerasForSyncQueue returns cameras that should appear in the Sync Queue
func (db *Database) GetCamerasForSyncQueue() []*ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*ManagedCamera, 0)
	for _, cameraState := range db.CameraStates {
		// Camera must be reachable, paired, managed, and not synced
		if cameraState.Status.IsReachable &&
			cameraState.Status.IsPaired &&
			cameraState.Status.IsManaged &&
			!cameraState.Status.IsSynced {
			cameras = append(cameras, &ManagedCamera{CameraState: cameraState})
		}
	}
	return cameras
}

