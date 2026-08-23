// Package store persists the domain model as a single JSON file.
// All reads return copies; all writes go through the store's lock.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/model"
)

// Store is the in-memory state with JSON persistence
type Store struct {
	cameraStates map[string]*model.CameraWithState
	syncQueue    []*model.SyncQueueEntry
	groups       []*model.Group
	config       model.Config
	filePath     string
	mutex        sync.RWMutex
}

// storeJSON is the JSON-serializable representation of Store.
type storeJSON struct {
	CameraStates map[string]*model.CameraWithState `json:"camera_states"`
	SyncQueue    []*model.SyncQueueEntry           `json:"sync_queue"`
	Groups       []*model.Group                    `json:"groups"`
	Config       model.Config                      `json:"config"`
}

func (db *Store) MarshalJSON() ([]byte, error) {
	return json.Marshal(storeJSON{
		CameraStates: db.cameraStates,
		SyncQueue:    db.syncQueue,
		Groups:       db.groups,
		Config:       db.config,
	})
}

func (db *Store) UnmarshalJSON(data []byte) error {
	var aux storeJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	db.cameraStates = aux.CameraStates
	db.syncQueue = aux.SyncQueue
	db.groups = aux.Groups
	db.config = aux.Config
	return nil
}

// New creates a store backed by the given JSON file, loading it if present.
func New(filePath string) (*Store, error) {
	db := &Store{
		cameraStates: make(map[string]*model.CameraWithState),
		syncQueue:    make([]*model.SyncQueueEntry, 0),
		groups:       make([]*model.Group, 0),
		config:       model.DefaultConfig(),
	}
	if err := db.initialize(filePath); err != nil {
		return nil, err
	}
	return db, nil
}

// initialize sets up the store with the specified file path
func (db *Store) initialize(filePath string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.filePath = filePath

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create a new file with default values
		if err := db.saveToFile(); err != nil {
			return fmt.Errorf("failed to create store file: %w", err)
		}
		return nil
	}

	// Load existing store
	return db.loadFromFile()
}

// loadFromFile loads the store from the JSON file
func (db *Store) loadFromFile() error {
	data, err := os.ReadFile(db.filePath)
	if err != nil {
		return fmt.Errorf("failed to read store file: %w", err)
	}

	if err := json.Unmarshal(data, db); err != nil {
		return fmt.Errorf("failed to unmarshal store: %w", err)
	}

	// Migration: existing DBs won't have check_on_return set (zero value = false).
	// Default to true for users who had periodic checks enabled.
	if db.config.StatusCheckIntervalSeconds > 0 && !db.config.CheckOnReturn {
		db.config.CheckOnReturn = true
	}

	return nil
}

// saveToFile saves the store to the JSON file atomically via temp file + rename.
func (db *Store) saveToFile() error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	tmpPath := db.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp store file: %w", err)
	}
	if err := os.Rename(tmpPath, db.filePath); err != nil {
		return fmt.Errorf("failed to rename temp store file: %w", err)
	}
	return nil
}

// groupIDFor derives a camera's group from the groups list, the single
// source of truth for membership; the GroupID stored on the camera is
// ignored on read. Caller must hold at least the read lock.
func (db *Store) groupIDFor(cameraID string) string {
	for _, g := range db.groups {
		for _, id := range g.CameraIDs {
			if id == cameraID {
				return g.ID
			}
		}
	}
	return ""
}

// copyCamera returns a copy with the derived GroupID filled in.
// Caller must hold at least the read lock.
func (db *Store) copyCamera(cs *model.CameraWithState) *model.CameraWithState {
	csCopy := *cs
	csCopy.GroupID = db.groupIDFor(cs.Camera.ID)
	return &csCopy
}

// CameraLookupResult holds the result of a camera lookup by BLE address.
type CameraLookupResult struct {
	DBKey       string
	CameraState *model.CameraWithState
}

// FindCameraByBLEAddress finds a camera by its BLE address (linear scan).
// Returns the DB key and camera state regardless of whether the key is serial or BLE address.
func (db *Store) FindCameraByBLEAddress(bleAddress string) (*CameraLookupResult, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for key, cs := range db.cameraStates {
		if cs.Camera.BLEAddress == bleAddress {
			return &CameraLookupResult{DBKey: key, CameraState: db.copyCamera(cs)}, true
		}
	}
	return nil, false
}

// FindCameraBySerial finds a camera by serial number: the stable identity
// that survives macOS's per-boot random BLE addresses.
func (db *Store) FindCameraBySerial(serial string) (*CameraLookupResult, bool) {
	if serial == "" {
		return nil, false
	}
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for key, cs := range db.cameraStates {
		if key == serial || cs.Metadata.SerialNumber == serial {
			return &CameraLookupResult{DBKey: key, CameraState: db.copyCamera(cs)}, true
		}
	}
	return nil, false
}

// FindCameraByName finds a camera by its exact advertised name.
// Fallback identity for older cameras whose advertisements carry no
// serial, on platforms with unstable BLE addresses (macOS).
func (db *Store) FindCameraByName(name string) (*CameraLookupResult, bool) {
	if name == "" {
		return nil, false
	}
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for key, cs := range db.cameraStates {
		if cs.Camera.Name == name {
			return &CameraLookupResult{DBKey: key, CameraState: db.copyCamera(cs)}, true
		}
	}
	return nil, false
}

// RekeyCamera moves a camera entry from oldKey to newKey.
func (db *Store) RekeyCamera(oldKey, newKey string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cs, exists := db.cameraStates[oldKey]
	if !exists {
		return fmt.Errorf("%w: %s", model.ErrCameraNotFound, oldKey)
	}
	if oldKey == newKey {
		return nil
	}
	db.cameraStates[newKey] = cs
	delete(db.cameraStates, oldKey)
	return db.saveToFile()
}

// AddOrUpdateDiscoveredCamera adds or updates a discovered camera.
// Keyed by serial number when known (stable identity); BLE address otherwise.
func (db *Store) AddOrUpdateDiscoveredCamera(camera *model.DiscoveredCamera) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	key := camera.CameraState.Camera.BLEAddress
	if serial := camera.CameraState.Metadata.SerialNumber; serial != "" {
		key = serial
	}
	db.cameraStates[key] = camera.CameraState

	return db.saveToFile()
}

// GetDiscoveredCamera retrieves a discovered camera by key (serial or BLE address)
func (db *Store) GetDiscoveredCamera(key string) (*model.DiscoveredCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameraState, exists := db.cameraStates[key]
	if !exists {
		return nil, false
	}
	return &model.DiscoveredCamera{CameraState: db.copyCamera(cameraState)}, true
}

// GetManagedCamera retrieves a managed camera by key (serial or BLE address)
func (db *Store) GetManagedCamera(key string) (*model.ManagedCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameraState, exists := db.cameraStates[key]
	if !exists {
		return nil, false
	}
	if cameraState.InManagedPool() {
		return &model.ManagedCamera{CameraState: db.copyCamera(cameraState)}, true
	}
	return nil, false
}

// GetCameraByID retrieves a camera by its ID (returns a copy)
func (db *Store) GetCameraByID(id string) (*model.CameraWithState, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, cameraState := range db.cameraStates {
		if cameraState.Camera.ID == id {
			return db.copyCamera(cameraState), true
		}
	}
	return nil, false
}

// GetManagedCameraByID retrieves a managed camera by ID (returns a copy)
func (db *Store) GetManagedCameraByID(id string) (*model.ManagedCamera, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, cameraState := range db.cameraStates {
		if cameraState.Camera.ID == id && cameraState.InManagedPool() {
			return &model.ManagedCamera{CameraState: db.copyCamera(cameraState)}, true
		}
	}
	return nil, false
}

// AddSyncQueueEntry adds a camera to the sync queue
func (db *Store) AddSyncQueueEntry(entry *model.SyncQueueEntry) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if the camera is already in the queue
	for _, existing := range db.syncQueue {
		if existing.CameraID == entry.CameraID {
			if entry.Priority > existing.Priority {
				*existing = *entry
				return db.saveToFile()
			}
			return nil
		}
	}

	db.syncQueue = append(db.syncQueue, entry)
	return db.saveToFile()
}

// GetSyncQueue returns the current sync queue, sorted by priority (highest first) then by queued time
func (db *Store) GetSyncQueue() []*model.SyncQueueEntry {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	queue := make([]*model.SyncQueueEntry, len(db.syncQueue))
	copy(queue, db.syncQueue)

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
func (db *Store) UpdateSyncQueueEntry(entry *model.SyncQueueEntry) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, existing := range db.syncQueue {
		if existing.CameraID == entry.CameraID {
			db.syncQueue[i] = entry
			return db.saveToFile()
		}
	}
	return fmt.Errorf("sync queue entry for camera %s not found", entry.CameraID)
}

// RemoveSyncQueueEntry removes an entry from the sync queue
func (db *Store) RemoveSyncQueueEntry(cameraID string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, entry := range db.syncQueue {
		if entry.CameraID == cameraID {
			db.syncQueue[i] = db.syncQueue[len(db.syncQueue)-1]
			db.syncQueue = db.syncQueue[:len(db.syncQueue)-1]
			return db.saveToFile()
		}
	}
	return nil
}

// copyGroup deep-copies a group so callers never share the stored pointer.
func copyGroup(g *model.Group) *model.Group {
	gc := *g
	gc.CameraIDs = append([]string(nil), g.CameraIDs...)
	return &gc
}

// AddOrUpdateGroup adds or updates a group (stores a copy)
func (db *Store) AddOrUpdateGroup(group *model.Group) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	stored := copyGroup(group)
	for i, existing := range db.groups {
		if existing.ID == group.ID {
			db.groups[i] = stored
			return db.saveToFile()
		}
	}

	db.groups = append(db.groups, stored)
	return db.saveToFile()
}

// GetGroup returns a group by ID (returns a copy)
func (db *Store) GetGroup(id string) (*model.Group, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	for _, group := range db.groups {
		if group.ID == id {
			return copyGroup(group), true
		}
	}
	return nil, false
}

// GetAllGroups returns all groups (returns copies)
func (db *Store) GetAllGroups() []*model.Group {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	result := make([]*model.Group, len(db.groups))
	for i, group := range db.groups {
		result[i] = copyGroup(group)
	}
	return result
}

// RemoveGroup removes a group
func (db *Store) RemoveGroup(id string) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for i, group := range db.groups {
		if group.ID == id {
			db.groups[i] = db.groups[len(db.groups)-1]
			db.groups = db.groups[:len(db.groups)-1]
			return db.saveToFile()
		}
	}
	return fmt.Errorf("%w: %s", model.ErrGroupNotFound, id)
}

// GetConfig returns the current configuration
func (db *Store) GetConfig() model.Config {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	return db.config
}

// UpdateConfig updates the system configuration
func (db *Store) UpdateConfig(config model.Config) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.config = config
	db.config.LastUpdated = time.Now()
	return db.saveToFile()
}

// UpdateCamera atomically mutates a camera by key under the write lock.
// The mutate func returns whether a durable field changed; ephemeral-only
// changes (RSSI, LastSeen, IsReachable) stay in memory and skip the file
// write, which otherwise happens on every BLE advertisement.
func (db *Store) UpdateCamera(key string, mutate func(*model.CameraWithState) bool) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	cs, exists := db.cameraStates[key]
	if !exists {
		return fmt.Errorf("%w: %s", model.ErrCameraNotFound, key)
	}
	if mutate(cs) {
		return db.saveToFile()
	}
	return nil
}

// UpdateCameraByID atomically mutates a camera by ID under the write lock.
func (db *Store) UpdateCameraByID(cameraID string, mutate func(*model.CameraWithState)) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for _, cs := range db.cameraStates {
		if cs.Camera.ID == cameraID {
			mutate(cs)
			return db.saveToFile()
		}
	}
	return fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
}

// MarkCamerasUnreachableBefore marks all cameras last seen before cutoff as unreachable.
// Reachability is ephemeral: memory only, no file write.
func (db *Store) MarkCamerasUnreachableBefore(cutoff time.Time) (bool, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	changed := false
	for _, cs := range db.cameraStates {
		if cs.Status.LastSeen.Before(cutoff) && cs.Status.IsReachable {
			cs.Status.IsReachable = false
			changed = true
		}
	}
	return changed, nil
}

// updateCameraStatusByID looks up by Camera.ID (immune to re-keying).
// The mutate func returns whether the change is durable and must be persisted.
func (db *Store) updateCameraStatusByID(cameraID string, mutate func(*model.CameraStatus) bool) error {
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
	return fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
}

func (db *Store) UpdateCameraPairingStatusByID(cameraID string, isPairing bool) error {
	return db.updateCameraStatusByID(cameraID, func(s *model.CameraStatus) bool {
		s.IsPairing = isPairing
		return false // transient UI state, reset at startup anyway
	})
}

func (db *Store) SetCameraPairedByID(cameraID string, isPaired bool) error {
	return db.updateCameraStatusByID(cameraID, func(s *model.CameraStatus) bool {
		s.IsPaired = isPaired
		s.IsPairing = false
		return true
	})
}

func (db *Store) UpdateCameraSyncingStatusByID(cameraID string, isSyncing bool) error {
	return db.updateCameraStatusByID(cameraID, func(s *model.CameraStatus) bool {
		s.IsSyncing = isSyncing
		return false // transient UI state, reset at startup anyway
	})
}

func (db *Store) MarkCameraSyncedByID(cameraID string) error {
	return db.updateCameraStatusByID(cameraID, func(s *model.CameraStatus) bool {
		s.IsSynced = true
		s.IsSyncing = false
		s.LastSynced = time.Now()
		return true
	})
}

func (db *Store) SetLastSyncErrorByID(cameraID string, errMsg string) error {
	return db.updateCameraStatusByID(cameraID, func(s *model.CameraStatus) bool {
		s.LastSyncError = errMsg
		return true
	})
}

func (db *Store) ResetSyncStatusByID(cameraID string) error {
	return db.updateCameraStatusByID(cameraID, func(s *model.CameraStatus) bool {
		s.IsSynced = false
		return true
	})
}

// ResetTransientStates resets transient states on app start.
// IsReachable is cleared too: the first advertisement re-marks the camera
// reachable, so stale reachability from the previous run never leaks through.
func (db *Store) ResetTransientStates() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	changesMade := false
	for _, cameraState := range db.cameraStates {
		if cameraState.Status.IsSyncing || cameraState.Status.IsPairing || cameraState.Status.IsReachable {
			cameraState.Status.IsSyncing = false
			cameraState.Status.IsPairing = false
			cameraState.Status.IsReachable = false
			changesMade = true
		}
	}

	if changesMade {
		return db.saveToFile()
	}

	return nil
}

// GetAllCameras returns all cameras (copies)
func (db *Store) GetAllCameras() []*model.CameraWithState {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*model.CameraWithState, 0, len(db.cameraStates))
	for _, cs := range db.cameraStates {
		cameras = append(cameras, db.copyCamera(cs))
	}
	return cameras
}

// GetCamerasForDiscoveredPool returns cameras that should appear in the Discovered pool
func (db *Store) GetCamerasForDiscoveredPool() []*model.DiscoveredCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*model.DiscoveredCamera, 0)
	for _, cameraState := range db.cameraStates {
		if cameraState.InDiscoveredPool() {
			cameras = append(cameras, &model.DiscoveredCamera{CameraState: db.copyCamera(cameraState)})
		}
	}
	return cameras
}

// GetCamerasForManagedPool returns cameras that should appear in the Managed pool
func (db *Store) GetCamerasForManagedPool() []*model.ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*model.ManagedCamera, 0)
	for _, cameraState := range db.cameraStates {
		if cameraState.InManagedPool() {
			cameras = append(cameras, &model.ManagedCamera{CameraState: db.copyCamera(cameraState)})
		}
	}
	return cameras
}

// GetCamerasForSyncQueue returns cameras that should appear in the Sync Queue
func (db *Store) GetCamerasForSyncQueue() []*model.ManagedCamera {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	cameras := make([]*model.ManagedCamera, 0)
	for _, cameraState := range db.cameraStates {
		if cameraState.EligibleForSyncQueue() {
			cameras = append(cameras, &model.ManagedCamera{CameraState: db.copyCamera(cameraState)})
		}
	}
	return cameras
}
