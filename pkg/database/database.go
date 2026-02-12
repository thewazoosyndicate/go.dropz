package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/protocol"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Camera represents a camera's basic information
type Camera struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Alias        string `json:"alias"`
	BLEAddress   string `json:"ble_address"`
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
	IsSynced      bool      `json:"is_synced"`
	IsSyncing     bool      `json:"is_syncing"`
	LastSyncError string    `json:"last_sync_error,omitempty"`
}

// CameraMetadata represents technical details about a camera
type CameraMetadata struct {
	ID               string `json:"id"` // References the camera ID
	FirmwareVersion  string `json:"firmware_version"`
	Model            string `json:"model"`
	SerialNumber     string `json:"serial_number"`
	BatteryLevel     int32  `json:"battery_level"` // percentage
	HardwareVersion  string `json:"hardware_version"`
	NumPhotos        int32  `json:"num_photos"`
	NumVideos        int32  `json:"num_videos"`
	SDCardStatusCode int32  `json:"sd_card_status"`
	RemainingSpaceKB int64  `json:"remaining_space_kb"`
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

// Config represents the system-wide configuration settings
type Config struct {
	PairModeEnabled            bool      `json:"pair_mode_enabled"`
	SyncEnabled                bool      `json:"sync_enabled"`
	ScanIntervalSeconds        int32     `json:"scan_interval_seconds"`
	ConnectTimeoutSeconds      int32     `json:"connect_timeout_seconds"`
	DaysThreshold              int32     `json:"days_threshold"`
	DestinationFolder          string    `json:"destination_folder"`
	InactivityTimeoutSeconds   int32     `json:"inactivity_timeout_seconds"`
	StatusCheckIntervalSeconds int32     `json:"status_check_interval_seconds"`
	CheckOnReturn              bool      `json:"check_on_return"`
	SetTimeEnabled             bool      `json:"set_time_enabled"`
	LogLevel                   string    `json:"log_level"`
	LastUpdated                time.Time `json:"last_updated"`
}

// Database represents the in-memory database with JSON persistence
type Database struct {
	cameraStates map[string]*CameraWithState
	SyncQueue    []*SyncQueueEntry `json:"sync_queue"`
	Groups       []*Group          `json:"groups"`
	Config       Config            `json:"config"`
	filePath     string
	mutex        sync.RWMutex
}

// databaseJSON is the JSON-serializable representation of Database.
type databaseJSON struct {
	CameraStates map[string]*CameraWithState `json:"camera_states"`
	SyncQueue    []*SyncQueueEntry           `json:"sync_queue"`
	Groups       []*Group                    `json:"groups"`
	Config       Config                      `json:"config"`
}

func (db *Database) MarshalJSON() ([]byte, error) {
	return json.Marshal(databaseJSON{
		CameraStates: db.cameraStates,
		SyncQueue:    db.SyncQueue,
		Groups:       db.Groups,
		Config:       db.Config,
	})
}

func (db *Database) UnmarshalJSON(data []byte) error {
	var aux databaseJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	db.cameraStates = aux.CameraStates
	db.SyncQueue = aux.SyncQueue
	db.Groups = aux.Groups
	db.Config = aux.Config
	return nil
}

var instance *Database
var once sync.Once

// GetDatabase returns the singleton database instance
func GetDatabase() *Database {
	once.Do(func() {
		instance = &Database{
			cameraStates: make(map[string]*CameraWithState),
			SyncQueue:    make([]*SyncQueueEntry, 0),
			Groups:       make([]*Group, 0),
			Config:       DefaultConfig(),
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

	// Migration: existing DBs won't have check_on_return set (zero value = false).
	// Default to true for users who had periodic checks enabled.
	if db.Config.StatusCheckIntervalSeconds > 0 && !db.Config.CheckOnReturn {
		db.Config.CheckOnReturn = true
	}

	return nil
}

// saveToFile saves the database to the JSON file atomically via temp file + rename.
func (db *Database) saveToFile() error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal database: %v", err)
	}

	tmpPath := db.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp database file: %v", err)
	}
	if err := os.Rename(tmpPath, db.filePath); err != nil {
		return fmt.Errorf("failed to rename temp database file: %v", err)
	}
	return nil
}


// CameraLookupResult holds the result of a camera lookup by BLE address.
type CameraLookupResult struct {
	DBKey       string
	CameraState *CameraWithState
}

// FindCameraByBLEAddress finds a camera by its BLE address (linear scan).
// Returns the DB key and camera state regardless of whether the key is serial or BLE address.
func (db *Database) FindCameraByBLEAddress(bleAddress string) (*CameraLookupResult, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for key, cs := range db.cameraStates {
		if cs.Camera.BLEAddress == bleAddress {
			csCopy := *cs
			return &CameraLookupResult{DBKey: key, CameraState: &csCopy}, true
		}
	}
	return nil, false
}

// RekeyCamera moves a camera entry from oldKey to newKey.
func (db *Database) RekeyCamera(oldKey, newKey string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cs, exists := db.cameraStates[oldKey]
	if !exists {
		return fmt.Errorf("camera not found: %s", oldKey)
	}
	if oldKey == newKey {
		return nil
	}
	db.cameraStates[newKey] = cs
	delete(db.cameraStates, oldKey)
	return db.saveToFile()
}

// AddOrUpdateDiscoveredCamera adds or updates a discovered camera
func (db *Database) AddOrUpdateDiscoveredCamera(camera *DiscoveredCamera) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.cameraStates[camera.CameraState.Camera.BLEAddress] = camera.CameraState

	return db.saveToFile()
}

// GetDiscoveredCamera retrieves a discovered camera by key (serial or BLE address)
func (db *Database) GetDiscoveredCamera(key string) (*DiscoveredCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameraState, exists := db.cameraStates[key]
	if !exists {
		return nil, false
	}
	csCopy := *cameraState
	return &DiscoveredCamera{CameraState: &csCopy}, true
}

// GetManagedCamera retrieves a managed camera by key (serial or BLE address)
func (db *Database) GetManagedCamera(key string) (*ManagedCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameraState, exists := db.cameraStates[key]
	if !exists {
		return nil, false
	}
	if cameraState.Status.IsManaged && cameraState.Status.IsPaired {
		csCopy := *cameraState
		return &ManagedCamera{CameraState: &csCopy}, true
	}
	return nil, false
}

// GetCameraByID retrieves a camera by its ID (returns a copy)
func (db *Database) GetCameraByID(id string) (*CameraWithState, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, cameraState := range db.cameraStates {
		if cameraState.Camera.ID == id {
			csCopy := *cameraState
			return &csCopy, true
		}
	}
	return nil, false
}

// GetManagedCameraByID retrieves a managed camera by ID (returns a copy)
func (db *Database) GetManagedCameraByID(id string) (*ManagedCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, cameraState := range db.cameraStates {
		if cameraState.Camera.ID == id && cameraState.Status.IsManaged && cameraState.Status.IsPaired {
			csCopy := *cameraState
			return &ManagedCamera{CameraState: &csCopy}, true
		}
	}
	return nil, false
}

// AddSyncQueueEntry adds a camera to the sync queue
func (db *Database) AddSyncQueueEntry(entry *SyncQueueEntry) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if the camera is already in the queue
	for _, existing := range db.SyncQueue {
		if existing.CameraID == entry.CameraID {
			if entry.Priority > existing.Priority {
				*existing = *entry
				return db.saveToFile()
			}
			return nil
		}
	}

	db.SyncQueue = append(db.SyncQueue, entry)
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
			db.SyncQueue[i] = db.SyncQueue[len(db.SyncQueue)-1]
			db.SyncQueue = db.SyncQueue[:len(db.SyncQueue)-1]
			return db.saveToFile()
		}
	}
	return nil
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
			db.Groups[i] = db.Groups[len(db.Groups)-1]
			db.Groups = db.Groups[:len(db.Groups)-1]
			return db.saveToFile()
		}
	}
	return fmt.Errorf("group with ID %s not found", id)
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
			BleAddress:   c.Camera.BLEAddress,
			WifiSsid:     c.Camera.WiFiSSID,
			WifiPassword: c.Camera.WiFiPassword,
			Rssi:         c.Camera.RSSI,
		},
		Status: &protocol.CameraStatus{
			LastSeen:      timestamppb.New(c.Status.LastSeen),
			LastSynced:    timestamppb.New(c.Status.LastSynced),
			IsPairing:     c.Status.IsPairing,
			IsPaired:      c.Status.IsPaired,
			IsManaged:     c.Status.IsManaged,
			IsReachable:   c.Status.IsReachable,
			IsSynced:      c.Status.IsSynced,
			IsSyncing:     c.Status.IsSyncing,
			LastSyncError: c.Status.LastSyncError,
		},
		GroupId: c.GroupID,
		Metadata: &protocol.CameraMetadata{
			Id:               c.Metadata.ID,
			FirmwareVersion:  c.Metadata.FirmwareVersion,
			Model:            c.Metadata.Model,
			SerialNumber:     c.Metadata.SerialNumber,
			BatteryLevel:     c.Metadata.BatteryLevel,
			HardwareVersion:  c.Metadata.HardwareVersion,
			NumPhotos:        c.Metadata.NumPhotos,
			NumVideos:        c.Metadata.NumVideos,
			SdCardStatus:     c.Metadata.SDCardStatusCode,
			RemainingSpaceKb: c.Metadata.RemainingSpaceKB,
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
		PairModeEnabled:            c.PairModeEnabled,
		SyncEnabled:                c.SyncEnabled,
		ScanIntervalSeconds:        c.ScanIntervalSeconds,
		ConnectTimeoutSeconds:      c.ConnectTimeoutSeconds,
		DaysThreshold:              c.DaysThreshold,
		DestinationFolder:          c.DestinationFolder,
		InactivityTimeoutSeconds:   c.InactivityTimeoutSeconds,
		StatusCheckIntervalSeconds: c.StatusCheckIntervalSeconds,
		CheckOnReturn:              c.CheckOnReturn,
		SetTimeEnabled:             c.SetTimeEnabled,
		LogLevel:                   c.LogLevel,
		LastUpdated:                timestamppb.New(c.LastUpdated),
	}
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return Config{
		PairModeEnabled:            false,
		SyncEnabled:                true,
		ScanIntervalSeconds:        30,
		ConnectTimeoutSeconds:      60,
		DaysThreshold:              1,
		DestinationFolder:          filepath.Join(homeDir, "Videos", "dropz"),
		InactivityTimeoutSeconds:   60,
		StatusCheckIntervalSeconds: 300,
		CheckOnReturn:              true,
		SetTimeEnabled:             true,
		LogLevel:                   "info",
		LastUpdated:                time.Now(),
	}
}

// UpdateCamera atomically mutates a camera by key under the write lock.
func (db *Database) UpdateCamera(key string, mutate func(*CameraWithState)) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cs, exists := db.cameraStates[key]
	if !exists {
		return fmt.Errorf("camera not found: %s", key)
	}
	mutate(cs)
	return db.saveToFile()
}

// UpdateCameraByID atomically mutates a camera by ID under the write lock.
func (db *Database) UpdateCameraByID(cameraID string, mutate func(*CameraWithState)) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for _, cs := range db.cameraStates {
		if cs.Camera.ID == cameraID {
			mutate(cs)
			return db.saveToFile()
		}
	}
	return fmt.Errorf("camera not found: %s", cameraID)
}

// MarkCamerasUnreachableBefore marks all cameras last seen before cutoff as unreachable.
func (db *Database) MarkCamerasUnreachableBefore(cutoff time.Time) (bool, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	changed := false
	for _, cs := range db.cameraStates {
		if cs.Status.LastSeen.Before(cutoff) && cs.Status.IsReachable {
			cs.Status.IsReachable = false
			changed = true
		}
	}
	if changed {
		return true, db.saveToFile()
	}
	return false, nil
}

// updateCameraStatusByID is like updateCameraStatus but looks up by Camera.ID (immune to re-keying).
func (db *Database) updateCameraStatusByID(cameraID string, mutate func(*CameraStatus) bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for _, cs := range db.cameraStates {
		if cs.Camera.ID == cameraID {
			if mutate(&cs.Status) {
				return db.saveToFile()
			}
			return nil
		}
	}
	return fmt.Errorf("camera not found: %s", cameraID)
}

func (db *Database) UpdateCameraPairingStatusByID(cameraID string, isPairing bool) error {
	return db.updateCameraStatusByID(cameraID, func(s *CameraStatus) bool {
		s.IsPairing = isPairing
		return true
	})
}

func (db *Database) SetCameraPairedByID(cameraID string, isPaired bool) error {
	return db.updateCameraStatusByID(cameraID, func(s *CameraStatus) bool {
		s.IsPaired = isPaired
		s.IsPairing = false
		return true
	})
}

func (db *Database) UpdateCameraSyncingStatusByID(cameraID string, isSyncing bool) error {
	return db.updateCameraStatusByID(cameraID, func(s *CameraStatus) bool {
		s.IsSyncing = isSyncing
		return true
	})
}

func (db *Database) MarkCameraSyncedByID(cameraID string) error {
	return db.updateCameraStatusByID(cameraID, func(s *CameraStatus) bool {
		s.IsSynced = true
		s.IsSyncing = false
		s.LastSynced = time.Now()
		return true
	})
}

func (db *Database) SetLastSyncErrorByID(cameraID string, errMsg string) error {
	return db.updateCameraStatusByID(cameraID, func(s *CameraStatus) bool {
		s.LastSyncError = errMsg
		return true
	})
}

func (db *Database) ResetSyncStatusByID(cameraID string) error {
	return db.updateCameraStatusByID(cameraID, func(s *CameraStatus) bool {
		s.IsSynced = false
		return true
	})
}

// ResetTransientStates resets transient states (is_syncing, is_pairing) on app start
func (db *Database) ResetTransientStates() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	changesMade := false
	for _, cameraState := range db.cameraStates {
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

// GetAllCameras returns all cameras (copies)
func (db *Database) GetAllCameras() []*CameraWithState {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*CameraWithState, 0, len(db.cameraStates))
	for _, cs := range db.cameraStates {
		csCopy := *cs
		cameras = append(cameras, &csCopy)
	}
	return cameras
}

// GetCamerasForDiscoveredPool returns cameras that should appear in the Discovered pool
func (db *Database) GetCamerasForDiscoveredPool() []*DiscoveredCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*DiscoveredCamera, 0)
	for _, cameraState := range db.cameraStates {
		if !cameraState.Status.IsPaired || !cameraState.Status.IsManaged {
			csCopy := *cameraState
			cameras = append(cameras, &DiscoveredCamera{CameraState: &csCopy})
		}
	}
	return cameras
}

// GetCamerasForManagedPool returns cameras that should appear in the Managed pool
func (db *Database) GetCamerasForManagedPool() []*ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*ManagedCamera, 0)
	for _, cameraState := range db.cameraStates {
		if cameraState.Status.IsPaired && cameraState.Status.IsManaged {
			csCopy := *cameraState
			cameras = append(cameras, &ManagedCamera{CameraState: &csCopy})
		}
	}
	return cameras
}

// GetCamerasForSyncQueue returns cameras that should appear in the Sync Queue
func (db *Database) GetCamerasForSyncQueue() []*ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*ManagedCamera, 0)
	for _, cameraState := range db.cameraStates {
		if cameraState.Status.IsReachable &&
			cameraState.Status.IsPaired &&
			cameraState.Status.IsManaged &&
			!cameraState.Status.IsSynced {
			csCopy := *cameraState
			cameras = append(cameras, &ManagedCamera{CameraState: &csCopy})
		}
	}
	return cameras
}

