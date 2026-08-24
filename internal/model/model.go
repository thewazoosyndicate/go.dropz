// Package model holds the domain types shared by store, manager, and server.
// It must stay free of persistence and transport concerns.
package model

import (
	"os"
	"path/filepath"
	"time"
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
	LastSeen      time.Time `json:"last_seen"`
	LastSynced    time.Time `json:"last_synced"`
	IsPairing     bool      `json:"is_pairing"`
	IsPaired      bool      `json:"is_paired"`
	IsManaged     bool      `json:"is_managed"`
	IsReachable   bool      `json:"is_reachable"`
	IsSynced      bool      `json:"is_synced"`
	IsSyncing     bool      `json:"is_syncing"`
	LastSyncError string    `json:"last_sync_error,omitempty"`
	// User intent: keep a preview session up whenever the camera is in
	// range, so in-app previews skip the connection dance.
	PreviewEnabled bool `json:"preview_enabled,omitempty"`
	// InPairingMode mirrors the camera-side pairing UI flag from BLE
	// advertising data; ephemeral, valid only while the camera is seen.
	InPairingMode bool `json:"in_pairing_mode,omitempty"`
}

// CameraMetadata represents technical details about a camera
type CameraMetadata struct {
	ID               string `json:"id"` // References the camera ID
	FirmwareVersion  string `json:"firmware_version"`
	Model            string `json:"model"`
	ModelID          int    `json:"model_id"`
	SerialNumber     string `json:"serial_number"`
	BatteryLevel     int32  `json:"battery_level"` // percentage
	HardwareVersion  string `json:"hardware_version"`
	NumPhotos        int32  `json:"num_photos"`
	NumVideos        int32  `json:"num_videos"`
	SDCardStatusCode int32  `json:"sd_card_status"`
	RemainingSpaceKB int64  `json:"remaining_space_kb"`
	// Camera state snapshotted when the last full sync completed. The
	// sync skip guard compares against these, never the live values
	// above: a status check refreshes those the moment it detects new
	// media, which would make the new content read as "no change".
	SyncedNumPhotos int32 `json:"synced_num_photos,omitempty"`
	SyncedNumVideos int32 `json:"synced_num_videos,omitempty"`
	SyncedSpaceKB   int64 `json:"synced_space_kb,omitempty"`
	// Settings is the last-known snapshot from the camera. Mutators must
	// replace the slice, never edit elements in place (copies share it).
	Settings        []CameraSetting `json:"settings,omitempty"`
	SettingsUpdated time.Time       `json:"settings_updated,omitempty"`
}

// SettingOption is one selectable value for a camera setting.
type SettingOption struct {
	Value int64  `json:"value"`
	Name  string `json:"name"`
}

// CameraSetting is one setting's state as the camera reported it over BLE.
// Options list only what the camera says is available right now; per-model
// and per-state gating comes from the camera, never from static tables.
type CameraSetting struct {
	ID        int32           `json:"id"`
	Name      string          `json:"name"`
	Value     int64           `json:"value"`
	ValueName string          `json:"value_name,omitempty"`
	Options   []SettingOption `json:"options,omitempty"`
}

// SettingApplyResult reports one setting write on one camera.
type SettingApplyResult struct {
	ID    int32  `json:"id"`
	Error string `json:"error,omitempty"` // empty = applied
}

// GroupSettingsResult reports one camera's outcome of a settings apply.
type GroupSettingsResult struct {
	CameraID string
	Error    string // whole-camera failure (unreachable, busy); empty otherwise
	Results  []SettingApplyResult
}

// CameraWithState represents a complete camera with all its information
type CameraWithState struct {
	Camera Camera       `json:"camera"`
	Status CameraStatus `json:"status"`
	// GroupID is derived from Group.CameraIDs on every store read;
	// never write it directly.
	GroupID  string         `json:"group_id"`
	Metadata CameraMetadata `json:"metadata"`
}

// LogAttrs is the standard camera identity pair for slog calls: stable ID
// plus display name, so lines join across subsystems.
func (c *CameraWithState) LogAttrs() []any {
	return []any{"camera_id", c.Camera.ID, "camera", c.Camera.Name}
}

// NewMediaCount is how many files the card holds beyond the last completed
// sync's baseline: the "new on camera" signal. 0 when unknown or clean.
func (c *CameraWithState) NewMediaCount() int32 {
	md := c.Metadata
	if md.SyncedSpaceKB <= 0 {
		return 0
	}
	n := (md.NumPhotos - md.SyncedNumPhotos) + (md.NumVideos - md.SyncedNumVideos)
	if n < 0 {
		return 0
	}
	return n
}

// InManagedPool reports whether the camera belongs in the Managed pool.
func (c *CameraWithState) InManagedPool() bool {
	return c.Status.IsPaired && c.Status.IsManaged
}

// InDiscoveredPool reports whether the camera belongs in the Discovered pool.
func (c *CameraWithState) InDiscoveredPool() bool {
	return !c.InManagedPool()
}

// DiscoveredCamera represents a camera that is not managed or paired
type DiscoveredCamera struct {
	CameraState *CameraWithState `json:"camera_state"`
}

// ManagedCamera represents a camera in the management pool
type ManagedCamera struct {
	CameraState *CameraWithState `json:"camera_state"`
}

// SyncFileState is one file's place in a transfer.
type SyncFileState string

const (
	SyncFileQueued      SyncFileState = "queued"
	SyncFileDownloading SyncFileState = "downloading"
	SyncFileDone        SyncFileState = "done"
	SyncFileFailed      SyncFileState = "failed"
	SyncFileSkipped     SyncFileState = "skipped" // already in the library
)

// SyncFile is one file the transfer step considered.
type SyncFile struct {
	Name       string        `json:"name"`
	CameraPath string        `json:"camera_path"`
	SizeBytes  int64         `json:"size_bytes"`
	State      SyncFileState `json:"state"`
	BytesDone  int64         `json:"bytes_done,omitempty"`
	Error      string        `json:"error,omitempty"`
	LocalPath  string        `json:"local_path,omitempty"`
	DurationMs int64         `json:"duration_ms,omitempty"`
}

// SyncPhase is the user-facing stage of a sync; the backend steps
// collapse onto these so the UI never shows nine labels.
type SyncPhase string

const (
	SyncPhaseWaiting  SyncPhase = "waiting"
	SyncPhaseConnect  SyncPhase = "connect"
	SyncPhaseLink     SyncPhase = "link"
	SyncPhaseCatalog  SyncPhase = "catalog"
	SyncPhaseTransfer SyncPhase = "transfer"
)

// SyncPhaseTiming records when a phase ran; FinishedAt is zero while it runs.
type SyncPhaseTiming struct {
	Phase      SyncPhase `json:"phase"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}

// SyncOutcome is how a sync ended.
type SyncOutcome string

const (
	SyncOutcomeComplete         SyncOutcome = "complete"
	SyncOutcomeUpToDate         SyncOutcome = "up_to_date"
	SyncOutcomeCatalogRefreshed SyncOutcome = "catalog_refreshed"
	SyncOutcomeFailed           SyncOutcome = "failed"
	SyncOutcomeCancelled        SyncOutcome = "cancelled"
)

// SyncSession is one finished sync, kept so the outcome outlives the
// queue entry and the UI can show what arrived, or why nothing did.
type SyncSession struct {
	ID              string            `json:"id"`
	CameraID        string            `json:"camera_id"`
	StartedAt       time.Time         `json:"started_at"`
	FinishedAt      time.Time         `json:"finished_at"`
	Outcome         SyncOutcome       `json:"outcome"`
	Error           string            `json:"error,omitempty"`       // user-visible, failed only
	FailedStep      string            `json:"failed_step,omitempty"` // backend step label
	StepIndex       int32             `json:"step_index,omitempty"`
	StepCount       int32             `json:"step_count,omitempty"`
	FilesDownloaded int32             `json:"files_downloaded"`
	FilesFailed     int32             `json:"files_failed"`
	FilesSkipped    int32             `json:"files_skipped"`
	BytesDownloaded int64             `json:"bytes_downloaded"`
	Files           []SyncFile        `json:"files,omitempty"`
	Phases          []SyncPhaseTiming `json:"phases,omitempty"`
	// Selection marks a media browser pick or catalog refresh; those
	// never make the camera "synced".
	Selection bool `json:"selection,omitempty"`
}

// CountFiles fills the per-state totals from Files.
func (s *SyncSession) CountFiles() {
	s.FilesDownloaded, s.FilesFailed, s.FilesSkipped, s.BytesDownloaded = 0, 0, 0, 0
	for _, f := range s.Files {
		switch f.State {
		case SyncFileDone:
			s.FilesDownloaded++
			s.BytesDownloaded += f.SizeBytes
		case SyncFileFailed:
			s.FilesFailed++
		case SyncFileSkipped:
			s.FilesSkipped++
		}
	}
}

// SyncQueueEntry represents a record in the sync queue
type SyncQueueEntry struct {
	CameraID         string    `json:"camera_id"` // References the camera ID
	QueuedAt         time.Time `json:"queued_at"`
	Priority         int32     `json:"priority"` // Higher numbers = higher priority (manual sync = 10, auto sync = 5)
	ProgressPercent  int32     `json:"progress_percent"`
	CurrentOperation string    `json:"current_operation"`
	// Download detail, populated only while the download step runs.
	// Byte fields are 0 when the camera did not report sizes.
	FileIndex  int32  `json:"file_index,omitempty"` // 1-based
	FileCount  int32  `json:"file_count,omitempty"`
	FileName   string `json:"file_name,omitempty"`
	FileBytes  int64  `json:"file_bytes,omitempty"`
	FileTotal  int64  `json:"file_total,omitempty"`
	BytesDone  int64  `json:"bytes_done,omitempty"`  // whole sync
	BytesTotal int64  `json:"bytes_total,omitempty"`
	RateBps    int64  `json:"rate_bps,omitempty"`
	// Set when the sync leaves the queue's waiting state.
	StartedAt time.Time         `json:"started_at,omitempty"`
	Phase     SyncPhase         `json:"phase,omitempty"`
	Phases    []SyncPhaseTiming `json:"phases,omitempty"`
	// Every file the transfer considered, card order; nil before the
	// media list is known. Replace the slice, never edit in place.
	Files     []SyncFile `json:"files,omitempty"`
	StepIndex int32      `json:"step_index,omitempty"` // 1-based
	StepCount int32      `json:"step_count,omitempty"`
	// FileNames limits the download to exactly these files (media browser
	// selection); empty means the normal date-threshold sync.
	FileNames []string `json:"file_names,omitempty"`
	// CatalogOnly refreshes the media catalog and thumbnails without
	// downloading anything.
	CatalogOnly bool `json:"catalog_only,omitempty"`
}

// CameraMediaItem is one file in a camera's cached media catalog,
// captured during the last sync.
type CameraMediaItem struct {
	Name          string    `json:"name"`
	CameraPath    string    `json:"camera_path"` // <dir>/<name> on the camera
	SizeBytes     int64     `json:"size_bytes"`
	CreatedAt     time.Time `json:"created_at"`
	ThumbnailPath string    `json:"thumbnail_path,omitempty"` // absolute local path
	Downloaded    bool      `json:"downloaded"`
	LocalPath     string    `json:"local_path,omitempty"`   // set when Downloaded
	PreviewPath   string    `json:"preview_path,omitempty"` // cached LRV proxy
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
	PreviewPath     string    `json:"preview_path,omitempty"` // in-app playable WebM
}

// TrimResult describes the file a lossless trim produced. StartMs is the
// keyframe the cut actually starts on, which can precede what was asked.
type TrimResult struct {
	OutputPath    string
	Name          string
	ThumbnailPath string
	PreviewPath   string
	SizeBytes     int64
	StartMs       int64
	EndMs         int64
}

// Config represents the system-wide configuration settings
type Config struct {
	PairModeEnabled            bool   `json:"pair_mode_enabled"`
	SyncEnabled                bool   `json:"sync_enabled"`
	ScanIntervalSeconds        int32  `json:"scan_interval_seconds"`
	ConnectTimeoutSeconds      int32  `json:"connect_timeout_seconds"`
	DaysThreshold              int32  `json:"days_threshold"`
	DestinationFolder          string `json:"destination_folder"`
	InactivityTimeoutSeconds   int32  `json:"inactivity_timeout_seconds"`
	StatusCheckIntervalSeconds int32  `json:"status_check_interval_seconds"`
	CheckOnReturn              bool   `json:"check_on_return"`
	SetTimeEnabled             bool   `json:"set_time_enabled"`
	// TurboEnabled uses Turbo Transfer during offload. Off by default:
	// it is tuned for the GoPro app's parallel chunked downloads and
	// measured slower for our single sequential stream on a HERO11.
	TurboEnabled bool      `json:"turbo_enabled"`
	LogLevel     string    `json:"log_level"`
	LastUpdated  time.Time `json:"last_updated"`
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
		ConnectTimeoutSeconds:      120,
		DaysThreshold:              1,
		DestinationFolder:          filepath.Join(homeDir, "Videos", "dropz"),
		InactivityTimeoutSeconds:   60,
		StatusCheckIntervalSeconds: 300,
		CheckOnReturn:              true,
		SetTimeEnabled:             true,
		TurboEnabled:               false,
		LogLevel:                   "info",
		LastUpdated:                time.Now(),
	}
}
