package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/logger"
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
	PairModeEnabled          bool      `json:"pair_mode_enabled"`
	SyncEnabled              bool      `json:"sync_enabled"`
	ScanIntervalSeconds      int32     `json:"scan_interval_seconds"`
	ConnectTimeoutSeconds    int32     `json:"connect_timeout_seconds"`
	DaysThreshold            int32     `json:"days_threshold"`
	DestinationFolder        string    `json:"destination_folder"`
	InactivityTimeoutSeconds int32     `json:"inactivity_timeout_seconds"`
	SetTimeEnabled           bool      `json:"set_time_enabled"`
	LogLevel                 string    `json:"log_level"`
	DebugMode                bool      `json:"debug_mode"`
	LastUpdated              time.Time `json:"last_updated"`
}

// Database represents the in-memory database with JSON persistence
type Database struct {
	CameraStates   map[string]*CameraWithState `json:"camera_states"`
	SyncQueue      []*SyncQueueEntry           `json:"sync_queue"`
	Groups         []*Group                    `json:"groups"`
	Videos         []*VideoFile                `json:"videos"`
	Logs           []*LogEntry                 `json:"logs"`
	Config         Config                      `json:"config"`
	filePath       string
	mutex          sync.RWMutex
	log            logger.Logger
	changeCounters struct {
		discovered int64
		managed    int64
		syncQueue  int64
	}
}

var instance *Database
var once sync.Once

// GetDatabase returns a singleton database instance
func GetDatabase() *Database {
	once.Do(func() {
		instance = &Database{
			CameraStates: make(map[string]*CameraWithState),
			SyncQueue:    make([]*SyncQueueEntry, 0),
			Groups:       make([]*Group, 0),
			Videos:       make([]*VideoFile, 0),
			Logs:         make([]*LogEntry, 0),
			Config: Config{
				PairModeEnabled:          true,
				SyncEnabled:              true,
				ScanIntervalSeconds:      30,
				ConnectTimeoutSeconds:    60,
				DaysThreshold:            1,
				DestinationFolder:        "videos",
				InactivityTimeoutSeconds: 600,
				SetTimeEnabled:           true,
				LogLevel:                 "info",
				DebugMode:                false,
				LastUpdated:              time.Now(),
			},
			log: logger.GetLogger(),
		}
	})
	return instance
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
		db.log.Info("Created new database file")
		return nil
	}

	// Load existing database
	return db.loadFromFile()
}

// loadFromFile loads the database from the JSON file
func (db *Database) loadFromFile() error {
	data, err := ioutil.ReadFile(db.filePath)
	if err != nil {
		return fmt.Errorf("failed to read database file: %v", err)
	}

	if err := json.Unmarshal(data, db); err != nil {
		return fmt.Errorf("failed to unmarshal database: %v", err)
	}

	db.log.Info("Loaded database from file")
	return nil
}

// saveToFile saves the database to the JSON file
func (db *Database) saveToFile() error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal database: %v", err)
	}

	if err := ioutil.WriteFile(db.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write database file: %v", err)
	}

	return nil
}

// SaveChanges persists changes to the database file
func (db *Database) SaveChanges() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.log.Tracef("Saving database changes to %s", db.filePath)

	startTime := time.Now()
	err := db.saveToFile()

	if err != nil {
		db.log.Errorf("Failed to save database: %v", err)
	} else {
		db.log.Debugf("Database saved to %s in %s", db.filePath, time.Since(startTime))
	}

	return err
}

// AddOrUpdateDiscoveredCamera adds or updates a discovered camera
func (db *Database) AddOrUpdateDiscoveredCamera(camera *DiscoveredCamera) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if this is a new camera or an update to an existing one
	existingCam, exists := db.CameraStates[camera.CameraState.Camera.MACAddress]

	// Always increment the change counter for GoPro cameras
	// But only if it's a new camera or if there are meaningful changes
	if !exists ||
		existingCam.Camera.RSSI != camera.CameraState.Camera.RSSI ||
		existingCam.Status.IsPairing != camera.CameraState.Status.IsPairing ||
		existingCam.Status.IsManaged != camera.CameraState.Status.IsManaged ||
		existingCam.Status.IsPaired != camera.CameraState.Status.IsPaired {
		db.changeCounters.discovered++
		db.log.Debugf("Incremented discovered change counter to %d for camera %s",
			db.changeCounters.discovered, camera.CameraState.Camera.Name)
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

// GetAllDiscoveredCameras returns all discovered cameras
func (db *Database) GetAllDiscoveredCameras() []*DiscoveredCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*DiscoveredCamera, 0, len(db.CameraStates))
	for _, cameraState := range db.CameraStates {
		// Only include cameras that are not managed or paired
		if !cameraState.Status.IsManaged || !cameraState.Status.IsPaired {
			cameras = append(cameras, &DiscoveredCamera{CameraState: cameraState})
		}
	}
	return cameras
}

// GetDiscoveredChangeCounter returns the current counter for discovered cameras
func (db *Database) GetDiscoveredChangeCounter() int64 {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.changeCounters.discovered
}

// AddOrUpdateManagedCamera adds or updates a managed camera
func (db *Database) AddOrUpdateManagedCamera(camera *ManagedCamera) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.CameraStates[camera.CameraState.Camera.MACAddress] = camera.CameraState
	db.changeCounters.managed++
	return db.saveToFile()
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

// GetAllManagedCameras returns all managed cameras
func (db *Database) GetAllManagedCameras() []*ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*ManagedCamera, 0)
	for _, cameraState := range db.CameraStates {
		// Only include cameras that are both managed and paired
		if cameraState.Status.IsManaged && cameraState.Status.IsPaired {
			cameras = append(cameras, &ManagedCamera{CameraState: cameraState})
		}
	}
	return cameras
}

// GetManagedChangeCounter returns the current counter for managed cameras
func (db *Database) GetManagedChangeCounter() int64 {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.changeCounters.managed
}

// RemoveManagedCamera removes a camera from the managed pool
func (db *Database) RemoveManagedCamera(macAddress string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if exists {
		// Just update the camera status, don't remove it from the database
		cameraState.Status.IsManaged = false
		db.changeCounters.managed++
		return db.saveToFile()
	}
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
			db.changeCounters.syncQueue++
			return db.saveToFile()
		}
	}

	// Add to queue
	db.SyncQueue = append(db.SyncQueue, entry)
	db.changeCounters.syncQueue++
	return db.saveToFile()
}

// GetSyncQueue returns the current sync queue
func (db *Database) GetSyncQueue() []*SyncQueueEntry {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	queue := make([]*SyncQueueEntry, len(db.SyncQueue))
	copy(queue, db.SyncQueue)
	return queue
}

// GetSyncQueueChangeCounter returns the current counter for the sync queue
func (db *Database) GetSyncQueueChangeCounter() int64 {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.changeCounters.syncQueue
}

// UpdateSyncQueueEntry updates an existing sync queue entry
func (db *Database) UpdateSyncQueueEntry(entry *SyncQueueEntry) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, existing := range db.SyncQueue {
		if existing.CameraID == entry.CameraID {
			db.SyncQueue[i] = entry
			db.changeCounters.syncQueue++
			return db.saveToFile()
		}
	}
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
			db.changeCounters.syncQueue++
			return db.saveToFile()
		}
	}
	return nil // Not found, not an error
}

// AddOrUpdateGroup adds or updates a group
func (db *Database) AddOrUpdateGroup(group *Group) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, existing := range db.Groups {
		if existing.ID == group.ID {
			db.Groups[i] = group
			return db.saveToFile()
		}
	}

	// Add new group
	db.Groups = append(db.Groups, group)
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
			return db.saveToFile()
		}
	}
	return fmt.Errorf("group with ID %s not found", id)
}

// AddVideo adds a video file
func (db *Database) AddVideo(video *VideoFile) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.Videos = append(db.Videos, video)
	return db.saveToFile()
}

// GetAllVideos returns all videos
func (db *Database) GetAllVideos() []*VideoFile {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*VideoFile, len(db.Videos))
	copy(result, db.Videos)
	return result
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
		PairModeEnabled:          c.PairModeEnabled,
		SyncEnabled:              c.SyncEnabled,
		ScanIntervalSeconds:      c.ScanIntervalSeconds,
		ConnectTimeoutSeconds:    c.ConnectTimeoutSeconds,
		DaysThreshold:            c.DaysThreshold,
		DestinationFolder:        c.DestinationFolder,
		InactivityTimeoutSeconds: c.InactivityTimeoutSeconds,
		SetTimeEnabled:           c.SetTimeEnabled,
		LogLevel:                 c.LogLevel,
		DebugMode:                c.DebugMode,
		LastUpdated:              timestamppb.New(c.LastUpdated),
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
	return Config{
		PairModeEnabled:          true,
		SyncEnabled:              true,
		ScanIntervalSeconds:      30,
		ConnectTimeoutSeconds:    60,
		DaysThreshold:            1,
		DestinationFolder:        "videos",
		InactivityTimeoutSeconds: 600,
		SetTimeEnabled:           true,
		LogLevel:                 "info",
		DebugMode:                false,
		LastUpdated:              time.Now(),
	}
}

// UpdateCameraReachability updates a camera's reachability status and last_seen timestamp
func (db *Database) UpdateCameraReachability(macAddress string, isReachable bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	// Update last_seen timestamp if the camera is reachable
	if isReachable {
		cameraState.Status.LastSeen = time.Now()
	}

	// Only make changes if the state actually changes
	if cameraState.Status.IsReachable != isReachable {
		cameraState.Status.IsReachable = isReachable
		db.changeCounters.discovered++
		db.changeCounters.managed++
		return db.saveToFile()
	}

	return nil
}

// UpdateCameraPairingStatus updates a camera's pairing status
func (db *Database) UpdateCameraPairingStatus(macAddress string, isPairing bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	cameraState.Status.IsPairing = isPairing
	db.changeCounters.discovered++
	db.changeCounters.managed++
	return db.saveToFile()
}

// SetCameraPaired marks a camera as paired
func (db *Database) SetCameraPaired(macAddress string, isPaired bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	cameraState.Status.IsPaired = isPaired
	cameraState.Status.IsPairing = false // Always reset pairing status
	db.changeCounters.discovered++
	db.changeCounters.managed++
	return db.saveToFile()
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
	db.changeCounters.discovered++
	db.changeCounters.managed++
	return db.saveToFile()
}

// UpdateCameraSyncingStatus updates a camera's syncing status
func (db *Database) UpdateCameraSyncingStatus(macAddress string, isSyncing bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	cameraState.Status.IsSyncing = isSyncing
	db.changeCounters.managed++
	return db.saveToFile()
}

// MarkCameraSynced marks a camera as synced and updates last_synced timestamp
func (db *Database) MarkCameraSynced(macAddress string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	cameraState.Status.IsSynced = true
	cameraState.Status.IsSyncing = false // Reset syncing status
	cameraState.Status.LastSynced = time.Now()
	db.changeCounters.managed++
	return db.saveToFile()
}

// ResetSyncStatus marks a camera as not synced (to trigger re-sync)
func (db *Database) ResetSyncStatus(macAddress string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cameraState, exists := db.CameraStates[macAddress]
	if !exists {
		return fmt.Errorf("camera with MAC address %s not found", macAddress)
	}

	cameraState.Status.IsSynced = false
	db.changeCounters.managed++
	return db.saveToFile()
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
		db.changeCounters.discovered++
		db.changeCounters.managed++
		return db.saveToFile()
	}

	return nil
}

// GetCamerasForDiscoveredPool returns cameras that should appear in the Discovered pool
func (db *Database) GetCamerasForDiscoveredPool() []*DiscoveredCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*DiscoveredCamera, 0)
	for _, cameraState := range db.CameraStates {
		// Camera must be reachable and either not paired OR not managed
		if cameraState.Status.IsReachable && (!cameraState.Status.IsPaired || !cameraState.Status.IsManaged) {
			cameras = append(cameras, &DiscoveredCamera{CameraState: cameraState})
		}
	}

	// Log debug info about discovered pool criteria
	db.log.Debugf("GetCamerasForDiscoveredPool returning %d cameras", len(cameras))
	for _, camera := range cameras {
		db.log.Debugf("Camera in discovered pool: %s (Paired=%v, Managed=%v, Reachable=%v)",
			camera.CameraState.Camera.Name,
			camera.CameraState.Status.IsPaired,
			camera.CameraState.Status.IsManaged,
			camera.CameraState.Status.IsReachable)
	}

	return cameras
}

// GetCamerasForManagedPool returns cameras that should appear in the Managed pool
func (db *Database) GetCamerasForManagedPool() []*ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*ManagedCamera, 0)
	for _, cameraState := range db.CameraStates {
		// Camera must be reachable, paired, and managed
		if cameraState.Status.IsReachable &&
			cameraState.Status.IsPaired &&
			cameraState.Status.IsManaged {
			cameras = append(cameras, &ManagedCamera{CameraState: cameraState})
		}
	}

	// Log debug info about managed pool criteria
	db.log.Debugf("GetCamerasForManagedPool returning %d cameras", len(cameras))
	for _, camera := range cameras {
		db.log.Debugf("Camera in managed pool: %s (Paired=%v, Managed=%v, Reachable=%v)",
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

// GetVisualStatusForCamera determines the visual status to display in the UI
func GetVisualStatusForCamera(cameraState *CameraWithState) string {
	if !cameraState.Status.IsReachable {
		return "Unreachable"
	}

	if cameraState.Status.IsPairing {
		return "Pairing"
	}

	if cameraState.Status.IsSyncing {
		return "Syncing"
	}

	if !cameraState.Status.IsPaired && cameraState.Status.IsManaged {
		return "Unavailable"
	}

	if !cameraState.Status.IsPaired {
		return "Discovered"
	}

	if cameraState.Status.IsPaired && !cameraState.Status.IsManaged {
		return "Available"
	}

	if cameraState.Status.IsPaired && cameraState.Status.IsManaged {
		return "Managed"
	}

	return "Unknown"
}
