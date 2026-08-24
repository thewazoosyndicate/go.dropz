// Package store persists the domain model as a single JSON file.
// All reads return copies; all writes go through the store's lock.
package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/logging"
	"github.com/dropz/dropz/internal/model"
)

// maxSyncHistory bounds the kept sessions; per-file lists make each one
// a few KB and the whole store rewrites on every save.
const maxSyncHistory = 200

// Store is the in-memory state with JSON persistence
type Store struct {
	cameraStates map[string]*model.CameraWithState
	syncQueue    []*model.SyncQueueEntry
	syncHistory  []*model.SyncSession // newest first
	groups       []*model.Group
	config       model.Config
	filePath     string
	log          *slog.Logger
	mutex        sync.RWMutex
}

// storeJSON is the JSON-serializable representation of Store.
// Config is a pointer so a file missing the key keeps the defaults
// instead of zeroing every interval.
type storeJSON struct {
	CameraStates map[string]*model.CameraWithState `json:"camera_states"`
	SyncQueue    []*model.SyncQueueEntry           `json:"sync_queue"`
	SyncHistory  []*model.SyncSession              `json:"sync_history"`
	Groups       []*model.Group                    `json:"groups"`
	Config       *model.Config                     `json:"config"`
}

func (db *Store) MarshalJSON() ([]byte, error) {
	return json.Marshal(storeJSON{
		CameraStates: db.cameraStates,
		SyncQueue:    db.syncQueue,
		SyncHistory:  db.syncHistory,
		Groups:       db.groups,
		Config:       &db.config,
	})
}

func (db *Store) UnmarshalJSON(data []byte) error {
	var aux storeJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	// A truncated or hand-edited file can omit any key; nil collections
	// must not survive loading, or the first write panics on a nil map.
	db.cameraStates = aux.CameraStates
	if db.cameraStates == nil {
		db.cameraStates = make(map[string]*model.CameraWithState)
	}
	db.syncQueue = aux.SyncQueue
	if db.syncQueue == nil {
		db.syncQueue = make([]*model.SyncQueueEntry, 0)
	}
	db.syncHistory = aux.SyncHistory
	if db.syncHistory == nil {
		db.syncHistory = make([]*model.SyncSession, 0)
	}
	db.groups = aux.Groups
	if db.groups == nil {
		db.groups = make([]*model.Group, 0)
	}
	if aux.Config != nil {
		db.config = *aux.Config
	}
	return nil
}

// New creates a store backed by the given JSON file, loading it if present.
// A nil log discards; the store logs its own write failures because many
// callers deliberately ignore the returned error.
func New(filePath string, log *slog.Logger) (*Store, error) {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	db := &Store{
		cameraStates: make(map[string]*model.CameraWithState),
		syncQueue:    make([]*model.SyncQueueEntry, 0),
		syncHistory:  make([]*model.SyncSession, 0),
		groups:       make([]*model.Group, 0),
		config:       model.DefaultConfig(),
		log:          log.With("component", "store"),
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
		db.log.Info("Store created", "path", filePath)
		return nil
	}

	// Load existing store
	if err := db.loadFromFile(); err != nil {
		return err
	}
	db.log.Info("Store loaded", "path", filePath,
		"cameras", len(db.cameraStates), "groups", len(db.groups), "queued", len(db.syncQueue))
	return nil
}

// loadFromFile loads the store from the JSON file
func (db *Store) loadFromFile() error {
	data, err := os.ReadFile(db.filePath)
	if err != nil {
		return fmt.Errorf("failed to read store file: %w", err)
	}

	if err := json.Unmarshal(data, db); err != nil {
		db.log.Error("Store file is corrupt", "path", db.filePath, "bytes", len(data), "err", err)
		return fmt.Errorf("failed to unmarshal store: %w", err)
	}

	// Migration: existing DBs won't have check_on_return set (zero value = false).
	// Default to true for users who had periodic checks enabled.
	if db.config.StatusCheckIntervalSeconds > 0 && !db.config.CheckOnReturn {
		db.config.CheckOnReturn = true
		db.log.Info("Store migration applied", "migration", "check_on_return")
	}

	return nil
}

// saveToFile saves the store to the JSON file atomically via temp file + rename.
// Failures are logged here as well as returned: persistence silently stopping
// is the worst store failure mode and most callers drop the error.
func (db *Store) saveToFile() error {
	start := time.Now()
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		db.log.Error("Store save failed", "path", db.filePath, "err", err)
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	tmpPath := db.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		db.log.Error("Store save failed", "path", db.filePath, "err", err)
		return fmt.Errorf("failed to write temp store file: %w", err)
	}
	if err := os.Rename(tmpPath, db.filePath); err != nil {
		db.log.Error("Store save failed", "path", db.filePath, "err", err)
		return fmt.Errorf("failed to rename temp store file: %w", err)
	}
	logging.Trace(db.log, "Store saved", "bytes", len(data), "elapsed", time.Since(start))
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

// copyEntry deep-copies a queue entry so readers never share its slices
// with the progress writer.
func copyEntry(e *model.SyncQueueEntry) *model.SyncQueueEntry {
	ec := *e
	ec.Files = append([]model.SyncFile(nil), e.Files...)
	ec.Phases = append([]model.SyncPhaseTiming(nil), e.Phases...)
	ec.FileNames = append([]string(nil), e.FileNames...)
	return &ec
}

// copyCamera returns a copy with the derived GroupID filled in.
// Caller must hold at least the read lock.
func (db *Store) copyCamera(cs *model.CameraWithState) *model.CameraWithState {
	csCopy := *cs
	csCopy.GroupID = db.groupIDFor(cs.Camera.ID)
	if len(cs.Metadata.Settings) > 0 {
		csCopy.Metadata.Settings = make([]model.CameraSetting, len(cs.Metadata.Settings))
		copy(csCopy.Metadata.Settings, cs.Metadata.Settings)
	}
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
	if existing, taken := db.cameraStates[newKey]; taken {
		// The serial-keyed entry is the camera's durable identity (ID,
		// managed/paired flags, group membership). The oldKey entry is a
		// duplicate born from a rotated BLE address: take its fresh
		// address and drop it, instead of clobbering the identity.
		existing.Camera.BLEAddress = cs.Camera.BLEAddress
		delete(db.cameraStates, oldKey)
		return db.saveToFile()
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

// MutateSyncQueueEntry mutates a camera's queued entry under lock and
// persists it. Returns false when the camera is not queued.
func (db *Store) MutateSyncQueueEntry(cameraID string, mutate func(*model.SyncQueueEntry)) (bool, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	for _, existing := range db.syncQueue {
		if existing.CameraID == cameraID {
			mutate(existing)
			return true, db.saveToFile()
		}
	}
	return false, nil
}

// GetSyncQueue returns the current sync queue, sorted by priority (highest first) then by queued time
func (db *Store) GetSyncQueue() []*model.SyncQueueEntry {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	queue := make([]*model.SyncQueueEntry, len(db.syncQueue))
	for i, e := range db.syncQueue {
		queue[i] = copyEntry(e)
	}

	// Sort by priority (highest first), then by queued time (oldest first)
	sort.Slice(queue, func(i, j int) bool {
		if queue[i].Priority != queue[j].Priority {
			return queue[i].Priority > queue[j].Priority // Higher priority first
		}
		return queue[i].QueuedAt.Before(queue[j].QueuedAt) // Older entries first for same priority
	})

	return queue
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

// copySession deep-copies a session so callers never share the stored slices.
func copySession(s *model.SyncSession) *model.SyncSession {
	sc := *s
	sc.Files = append([]model.SyncFile(nil), s.Files...)
	sc.Phases = append([]model.SyncPhaseTiming(nil), s.Phases...)
	return &sc
}

// AddSyncSession records a finished sync at the head of the history,
// dropping the oldest beyond maxSyncHistory.
func (db *Store) AddSyncSession(session *model.SyncSession) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.syncHistory = append([]*model.SyncSession{copySession(session)}, db.syncHistory...)
	if len(db.syncHistory) > maxSyncHistory {
		db.syncHistory = db.syncHistory[:maxSyncHistory]
	}
	return db.saveToFile()
}

// GetSyncHistory returns finished syncs newest first, optionally for one
// camera; limit <= 0 returns everything kept.
func (db *Store) GetSyncHistory(limit int, cameraID string) []*model.SyncSession {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	out := make([]*model.SyncSession, 0, len(db.syncHistory))
	for _, s := range db.syncHistory {
		if cameraID != "" && s.CameraID != cameraID {
			continue
		}
		out = append(out, copySession(s))
		if limit > 0 && len(out) == limit {
			break
		}
	}
	return out
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

// MarkCamerasUnreachableBefore marks all cameras last seen before cutoff as
// unreachable and returns the identities that changed, so the caller can log
// the departure transition. Reachability is ephemeral: memory only, no file write.
func (db *Store) MarkCamerasUnreachableBefore(cutoff time.Time) []model.Camera {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	var changed []model.Camera
	for _, cs := range db.cameraStates {
		if cs.Status.LastSeen.Before(cutoff) && cs.Status.IsReachable {
			cs.Status.IsReachable = false
			changed = append(changed, cs.Camera)
		}
	}
	return changed
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

// SetCameraAliasByID stores the user's name for a camera; empty clears it.
func (db *Store) SetCameraAliasByID(cameraID, alias string) error {
	return db.UpdateCameraByID(cameraID, func(cs *model.CameraWithState) {
		cs.Camera.Alias = alias
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
