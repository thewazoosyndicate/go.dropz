package manager

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/manager/discovery"
	"github.com/dropz/dropz/internal/manager/pairing"
	syncpkg "github.com/dropz/dropz/internal/manager/sync"
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/store"
	"tinygo.org/x/bluetooth"
)

// GoProManager is the main service that coordinates all GoPro operations
type GoProManager struct {
	// Core dependencies
	db       *store.Store
	ble      *ble.Manager
	log      *slog.Logger
	logLevel *slog.LevelVar
	notifier func()

	// Runtime control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.RWMutex

	// Serializes pairing (BLE adapter is single-threaded)
	pairingMu sync.Mutex

	// Channels for the deviceManager select loop
	immediateSyncTrigger chan struct{}
	statusCheckRequest   chan string

	// Component managers
	syncCoordinator    *syncpkg.Coordinator
	discoveryProcessor *discovery.Processor
	pairingManager     *pairing.Manager
}

// NewGoProManager creates a new GoPro manager instance
func NewGoProManager(dbPath, destinationDir string, log *slog.Logger, logLevel *slog.LevelVar) (*GoProManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Built before the callback closures below so they carry the component attr
	mlog := log.With("component", "manager")

	db, err := store.New(dbPath, log)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Enable BLE adapter directly (not in goroutine - required by platform BLE stacks).
	// Failure is not fatal: the library and config stay usable without Bluetooth,
	// and CI runners have no adapter at all.
	adapter := bluetooth.DefaultAdapter
	if adapter != nil {
		if err := adapter.Enable(); err != nil {
			mlog.Warn("Bluetooth unavailable, camera features disabled", "err", err)
			adapter = nil
		}
	} else {
		mlog.Warn("No default Bluetooth adapter, camera features disabled")
	}

	bleManager := ble.NewManager(adapter, log)

	// Set up metadata update callback to update database whenever camera connects
	bleManager.SetMetadataCallback(func(metadata ble.CameraMetadata) {
		cam, found := db.FindCameraByBLEAddress(metadata.BLEAddress)
		if !found {
			return
		}
		cameraID := cam.CameraState.Camera.ID

		_ = db.UpdateCameraByID(cameraID, func(cs *model.CameraWithState) {
			if metadata.WiFiSSID != "" && metadata.WiFiPassword != "" {
				cs.Camera.WiFiSSID = metadata.WiFiSSID
				cs.Camera.WiFiPassword = metadata.WiFiPassword
			}
			cs.Metadata.Model = metadata.ModelName
			cs.Metadata.ModelID = metadata.ModelID
			cs.Metadata.FirmwareVersion = metadata.FirmwareVersion
			cs.Metadata.SerialNumber = metadata.SerialNumber
			if metadata.BatteryLevel > 0 {
				cs.Metadata.BatteryLevel = int32(metadata.BatteryLevel)
			}
		})

		// Re-key from BLE address to serial number once serial is known
		if metadata.SerialNumber != "" && cam.DBKey != metadata.SerialNumber {
			if err := db.RekeyCamera(cam.DBKey, metadata.SerialNumber); err != nil {
				mlog.Warn("Failed to re-key camera to serial", "serial", metadata.SerialNumber, "err", err)
			}
		}
	})

	// If destination dir was provided via CLI and DB doesn't have one, store it
	config := db.GetConfig()
	if config.DestinationFolder == "" && destinationDir != "" {
		config.DestinationFolder = destinationDir
		_ = db.UpdateConfig(config)
	}

	manager := &GoProManager{
		ctx:                  ctx,
		cancel:               cancel,
		ble:                  bleManager,
		db:                   db,
		log:                  mlog,
		logLevel:             logLevel,
		immediateSyncTrigger: make(chan struct{}, 1),
		statusCheckRequest:   make(chan string, 8),
	}

	// Initialize component managers (notify/BLEOperation passed as method values)
	manager.syncCoordinator = syncpkg.NewCoordinator(ctx, db, bleManager, log, manager.notify, manager.BLEOperation)
	manager.discoveryProcessor = discovery.NewProcessor(db, log)
	manager.pairingManager = pairing.NewManager(ctx, db, bleManager, log, manager.BLEOperation, manager.notify)

	// Handle real-time status push notifications from camera (battery level)
	bleManager.SetStatusCallback(func(bleAddress string, statusID byte, value []byte) {
		if len(value) < 1 {
			return
		}

		cam, found := db.FindCameraByBLEAddress(bleAddress)
		if !found {
			return
		}

		switch statusID {
		case ble.StatusBatteryPercentage:
			if err := db.UpdateCameraByID(cam.CameraState.Camera.ID, func(cs *model.CameraWithState) {
				cs.Metadata.BatteryLevel = int32(value[0])
			}); err != nil {
				mlog.Error("Failed to update battery from push notification", "err", err)
			}
			manager.notify()
		}
	})

	// Auto-pair when a managed, unpaired camera shows its pairing UI.
	// The camera advertises the pairing flag, so no connection is needed
	// to detect it, and pairing attempts are no longer fired blind.
	manager.discoveryProcessor.SetOnPairingModeDetected(func(cameraID string) {
		if !db.GetConfig().PairModeEnabled {
			return
		}
		cs, ok := db.GetCameraByID(cameraID)
		if !ok || !cs.Status.IsManaged || cs.Status.IsPaired || cs.Status.IsPairing {
			return
		}
		go func() {
			if _, err := manager.PairCamera(cameraID); err != nil {
				mlog.Warn("Advertised pairing failed", "camera", cs.Camera.Name, "err", err)
			}
		}()
	})

	// A camera advertising new media gets a status check; the count
	// comparison there decides whether a sync is queued.
	manager.discoveryProcessor.SetOnNewMediaAdvertised(func(cameraID string) {
		cam, ok := db.GetManagedCameraByID(cameraID)
		if !ok || cam.CameraState.Status.IsSyncing {
			return
		}
		select {
		case manager.statusCheckRequest <- cameraID:
		default:
			mlog.Debug("Status check request dropped, queue full", cam.CameraState.LogAttrs()...)
		}
	})

	// Trigger BLE status check when a managed camera reappears
	manager.discoveryProcessor.SetOnCameraReappeared(func(cameraID string, wasGoneFor time.Duration) {
		if !db.GetConfig().CheckOnReturn {
			return
		}
		cam, ok := db.GetManagedCameraByID(cameraID)
		if !ok || cam.CameraState.Status.IsSyncing {
			return
		}
		select {
		case manager.statusCheckRequest <- cameraID:
		default:
			mlog.Debug("Status check request dropped, queue full", cam.CameraState.LogAttrs()...)
		}
	})

	return manager, nil
}

// SetNotifier sets the update notifier component
func (m *GoProManager) SetNotifier(notifier func()) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.notifier = notifier
}

// notify safely calls the notifier if set
func (m *GoProManager) notify() {
	m.mutex.RLock()
	notifier := m.notifier
	m.mutex.RUnlock()
	if notifier != nil {
		notifier()
	}
}

// GetDiscoveredCameras returns cameras in the Discovered pool
func (m *GoProManager) GetDiscoveredCameras() []*model.DiscoveredCamera {
	return m.db.GetCamerasForDiscoveredPool()
}

// GetManagedCameras returns cameras in the Managed pool
func (m *GoProManager) GetManagedCameras() []*model.ManagedCamera {
	return m.db.GetCamerasForManagedPool()
}

// GetSyncQueue returns the current sync queue
func (m *GoProManager) GetSyncQueue() []*model.SyncQueueEntry {
	return m.db.GetSyncQueue()
}

// GetGroups returns all groups
func (m *GoProManager) GetGroups() []*model.Group {
	return m.db.GetAllGroups()
}

// ManageCamera adds a camera to the managed camera pool
func (m *GoProManager) ManageCamera(cameraID string) (*model.ManagedCamera, error) {
	cs, found := m.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}

	if cs.Status.IsManaged {
		managedCamera, _ := m.db.GetManagedCameraByID(cameraID)
		return managedCamera, nil
	}

	_ = m.db.UpdateCameraByID(cameraID, func(cs *model.CameraWithState) {
		cs.Status.IsManaged = true
	})

	m.log.Info("Camera added to managed pool", cs.LogAttrs()...)
	m.notify()

	config := m.db.GetConfig()
	if config.PairModeEnabled && !cs.Status.IsPaired {
		m.log.Debug("Pair mode enabled, starting pairing", cs.LogAttrs()...)
		// Async so the gRPC handler doesn't block behind pairingMu for 30s+
		go func() {
			if _, err := m.PairCamera(cameraID); err != nil {
				m.log.Warn("Auto-pairing failed", append(cs.LogAttrs(), "err", err)...)
			}
		}()
	} else if cs.Status.IsReachable && cs.Status.IsPaired {
		select {
		case m.statusCheckRequest <- cameraID:
		default:
		}
	}

	// Pairing may still be running async, so fall back to the raw camera
	// state instead of requiring managed+paired.
	managedCamera, ok := m.db.GetManagedCameraByID(cameraID)
	if !ok {
		if updated, found := m.db.GetCameraByID(cameraID); found {
			managedCamera = &model.ManagedCamera{CameraState: updated}
		}
	}
	return managedCamera, nil
}

// UnmanageCamera removes a camera from the managed pool
func (m *GoProManager) UnmanageCamera(cameraID string) error {
	cs, found := m.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}

	_ = m.db.UpdateCameraByID(cameraID, func(cs *model.CameraWithState) {
		cs.Status.IsManaged = false
	})

	m.log.Info("Camera removed from managed pool", cs.LogAttrs()...)
	m.notify()

	return nil
}

// BLEOperation executes a BLE operation with retry logic.
// Set critical=true for operations that should retry on context.Canceled (sync, connect).
func (m *GoProManager) BLEOperation(ctx context.Context, critical bool, operation func() error) error {
	var opErr error
	for opTry := 0; opTry < 3; opTry++ {
		opErr = operation()
		if opErr == nil {
			return nil
		}

		if errors.Is(opErr, context.Canceled) && critical {
			m.log.Debug("Retrying critical BLE operation", "attempt", opTry+1, "max", 3)
			time.Sleep(3 * time.Second)
			continue
		} else if errors.Is(opErr, context.Canceled) || errors.Is(opErr, context.DeadlineExceeded) {
			return opErr
		}

		if isTransientBLEError(opErr) {
			m.log.Warn("Transient BLE error, retrying", "err", opErr, "backoff_s", (opTry+1)*3)
			time.Sleep(time.Duration(opTry+1) * 3 * time.Second)
			continue
		}

		return opErr
	}
	// Exhaustion carries the retried-3-times context the caller cannot know
	m.log.Warn("BLE operation retries exhausted", "attempts", 3, "err", opErr)
	return opErr
}

// ResetTransientStates resets transient camera states (is_syncing, is_pairing) on startup
func (m *GoProManager) ResetTransientStates() {
	m.log.Debug("Resetting transient camera states on startup")
	err := m.db.ResetTransientStates()
	if err != nil {
		m.log.Error("Failed to reset transient camera states", "err", err)
	}
}

// PairCamera pairs with a GoPro camera using BLE
func (m *GoProManager) PairCamera(cameraID string) (*model.ManagedCamera, error) {
	// Mark as pairing immediately so UI shows spinner even while queued
	_ = m.db.UpdateCameraPairingStatusByID(cameraID, true)
	m.notify()

	m.pairingMu.Lock()
	defer m.pairingMu.Unlock()

	result, err := m.pairingManager.PairCamera(cameraID)
	if err != nil {
		return result, err
	}

	// After pairing, run a status check to gather battery/media info and sleep the camera
	if cs, ok := m.db.GetCameraByID(cameraID); ok && cs.Status.IsReachable {
		select {
		case m.statusCheckRequest <- cameraID:
		default:
			m.log.Debug("Status check request dropped, queue full", cs.LogAttrs()...)
		}
	}

	return result, nil
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (m *GoProManager) ForceSync(cameraID string) (*model.SyncQueueEntry, error) {
	syncEntry, err := m.syncCoordinator.ForceSync(cameraID)
	if err != nil {
		return nil, err
	}

	select {
	case m.immediateSyncTrigger <- struct{}{}:
		m.log.Debug("Immediate sync trigger sent for manual sync request")
	default:
		m.log.Debug("Immediate sync trigger channel full, sync will be picked up by next cycle")
	}

	return syncEntry, nil
}

// CancelSync cancels an ongoing sync operation and removes the camera from the sync queue
func (m *GoProManager) CancelSync(cameraID string) error {
	return m.syncCoordinator.CancelSync(cameraID)
}

// GetVideosByCamera scans the destination folder for media files and maps them to cameras via WiFi SSID.
func (m *GoProManager) GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*model.VideoFile, int) {
	config := m.db.GetConfig()
	if config.DestinationFolder == "" {
		return []*model.VideoFile{}, 0
	}

	// Build WiFi SSID → camera ID map
	ssidToCamera := make(map[string]string)
	for _, cam := range m.db.GetAllCameras() {
		if cam.Camera.WiFiSSID != "" {
			ssidToCamera[cam.Camera.WiFiSSID] = cam.Camera.ID
		}
	}

	entries, err := os.ReadDir(config.DestinationFolder)
	if err != nil {
		m.log.Warn("Cannot read destination folder", "folder", config.DestinationFolder, "err", err)
		return []*model.VideoFile{}, 0
	}

	var allVideos []*model.VideoFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		ssid := entry.Name()
		camID, known := ssidToCamera[ssid]
		if !known {
			continue
		}
		if cameraID != "" && camID != cameraID {
			continue
		}

		subdir := filepath.Join(config.DestinationFolder, ssid)
		files, err := os.ReadDir(subdir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(f.Name()))
			mimeType := mime.TypeByExtension(ext)
			if !strings.HasPrefix(mimeType, "video/") && !strings.HasPrefix(mimeType, "image/") {
				continue
			}

			fullPath := filepath.Join(subdir, f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}

			mtime := info.ModTime()
			if !startDate.IsZero() && mtime.Before(startDate) {
				continue
			}
			if !endDate.IsZero() && mtime.After(endDate) {
				continue
			}

			hash := sha256.Sum256([]byte(fullPath))
			video := &model.VideoFile{
				ID:        fmt.Sprintf("%x", hash[:8]),
				Name:      f.Name(),
				Path:      fullPath,
				SizeBytes: info.Size(),
				CreatedAt: mtime,
				CameraID:  camID,
				MimeType:  mimeType,
			}
			// Camera-generated preview cached by the media catalog sync
			if thumb := syncpkg.ThumbnailPath(subdir, f.Name()); fileExistsNonEmpty(thumb) {
				video.ThumbnailPath = thumb
			}
			allVideos = append(allVideos, video)
		}
	}

	// Sort newest first
	sort.Slice(allVideos, func(i, j int) bool {
		return allVideos[i].CreatedAt.After(allVideos[j].CreatedAt)
	})

	totalCount := len(allVideos)
	if offset >= totalCount {
		return []*model.VideoFile{}, totalCount
	}
	end := offset + limit
	if limit == 0 || end > totalCount {
		end = totalCount
	}
	return allVideos[offset:end], totalCount
}
