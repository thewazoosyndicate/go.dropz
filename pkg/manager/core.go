package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/dropz/dropz/pkg/manager/discovery"
	"github.com/dropz/dropz/pkg/manager/pairing"
	syncpkg "github.com/dropz/dropz/pkg/manager/sync"
	"tinygo.org/x/bluetooth"
)


// GoProManager is the main service that coordinates all GoPro operations
type GoProManager struct {
	// Core dependencies
	db       *database.Database
	ble      *ble.Manager
	log      *logrus.Logger
	notifier func()

	// Runtime control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mutex  sync.RWMutex

	// Channels for the deviceManager select loop
	immediateSyncTrigger chan struct{}
	statusCheckRequest   chan string

	// Component managers
	syncCoordinator    *syncpkg.Coordinator
	discoveryProcessor *discovery.Processor
	pairingManager     *pairing.Manager
}

// NewGoProManager creates a new GoPro manager instance
func NewGoProManager(dbPath, destinationDir string, log *logrus.Logger) (*GoProManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	db := database.GetDatabase()
	if err := db.Initialize(dbPath); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	// Initialize BLE adapter
	adapter := bluetooth.DefaultAdapter
	if adapter == nil {
		cancel()
		return nil, fmt.Errorf("failed to get default Bluetooth adapter")
	}

	// Enable BLE adapter directly (not in goroutine - required by platform BLE stacks)
	if err := adapter.Enable(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to enable BLE adapter: %v", err)
	}

	bleManager := ble.NewManager(adapter, log)

	// Set up metadata update callback to update database whenever camera connects
	bleManager.SetMetadataCallback(func(metadata ble.CameraMetadata) {
		cam, found := db.FindCameraByBLEAddress(metadata.BLEAddress)
		if !found {
			return
		}
		cameraID := cam.CameraState.Camera.ID

		db.UpdateCameraByID(cameraID, func(cs *database.CameraWithState) {
			if metadata.WiFiSSID != "" && metadata.WiFiPassword != "" {
				cs.Camera.WiFiSSID = metadata.WiFiSSID
				cs.Camera.WiFiPassword = metadata.WiFiPassword
			}
			cs.Metadata.Model = metadata.ModelName
			cs.Metadata.FirmwareVersion = metadata.FirmwareVersion
			cs.Metadata.SerialNumber = metadata.SerialNumber
			if metadata.BatteryLevel > 0 {
				cs.Metadata.BatteryLevel = int32(metadata.BatteryLevel)
			}
		})

		// Re-key from BLE address to serial number once serial is known
		if metadata.SerialNumber != "" && cam.DBKey != metadata.SerialNumber {
			if err := db.RekeyCamera(cam.DBKey, metadata.SerialNumber); err != nil {
				log.Warnf("Failed to re-key camera to serial %s: %v", metadata.SerialNumber, err)
			}
		}
	})

	// If destination dir was provided via CLI and DB doesn't have one, store it
	config := db.GetConfig()
	if config.DestinationFolder == "" && destinationDir != "" {
		config.DestinationFolder = destinationDir
		db.UpdateConfig(config)
	}


	manager := &GoProManager{
		ctx:                  ctx,
		cancel:               cancel,
		ble:                  bleManager,
		db:                   db,
		log:                  log,
		immediateSyncTrigger: make(chan struct{}, 1),
		statusCheckRequest:   make(chan string, 8),
	}

	// Initialize component managers (coordinator gets notify/BLEOperation as method values)
	manager.syncCoordinator = syncpkg.NewCoordinator(db, bleManager, log, make(map[string]*syncpkg.SyncTask), manager.notify, manager.BLEOperation, ctx)
	manager.discoveryProcessor = discovery.NewProcessor(db, log)
	manager.pairingManager = pairing.NewManager(db, bleManager, log, ctx)

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
			if err := db.UpdateCameraByID(cam.CameraState.Camera.ID, func(cs *database.CameraWithState) {
				cs.Metadata.BatteryLevel = int32(value[0])
			}); err != nil {
				log.Errorf("Failed to update battery from push notification: %v", err)
			}
			manager.notify()
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
		log.Infof("Camera %s reappeared after %v, triggering status check", cam.CameraState.Camera.Name, wasGoneFor)
		select {
		case manager.statusCheckRequest <- cameraID:
		default:
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

// ManageCamera adds a camera to the managed camera pool
func (m *GoProManager) ManageCamera(cameraID string) (*database.ManagedCamera, error) {
	cs, found := m.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	if cs.Status.IsManaged {
		managedCamera, _ := m.db.GetManagedCameraByID(cameraID)
		return managedCamera, nil
	}

	m.db.UpdateCameraByID(cameraID, func(cs *database.CameraWithState) {
		cs.Status.IsManaged = true
	})

	m.log.Infof("Camera %s added to managed pool", cameraID)
	m.notify()

	config := m.db.GetConfig()
	if config.PairModeEnabled && !cs.Status.IsPaired {
		m.log.Infof("Pair mode enabled, starting pairing for camera %s", cs.Camera.Name)
		m.PairCamera(cameraID)
	} else if cs.Status.IsReachable && cs.Status.IsPaired {
		select {
		case m.statusCheckRequest <- cameraID:
		default:
		}
	}

	managedCamera, _ := m.db.GetManagedCameraByID(cameraID)
	return managedCamera, nil
}

// UnmanageCamera removes a camera from the managed pool
func (m *GoProManager) UnmanageCamera(cameraID string) error {
	_, found := m.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	m.db.UpdateCameraByID(cameraID, func(cs *database.CameraWithState) {
		cs.Status.IsManaged = false
	})

	m.log.Infof("Camera %s removed from managed pool", cameraID)
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

		if opErr == context.Canceled && critical {
			m.log.Infof("Waiting 3 seconds before retrying critical operation attempt %d/3", opTry+1)
			time.Sleep(3 * time.Second)
			continue
		} else if opErr == context.Canceled {
			return opErr
		}

		if isTransientBLEError(opErr) {
			m.log.Warnf("Transient BLE error: %v, retrying in %ds", opErr, (opTry+1)*3)
			time.Sleep(time.Duration(opTry+1) * 3 * time.Second)
			continue
		}

		return opErr
	}
	return opErr
}

// ResetTransientStates resets transient camera states (is_syncing, is_pairing) on startup
func (m *GoProManager) ResetTransientStates() {
	m.log.Info("Resetting transient camera states on startup")
	err := m.db.ResetTransientStates()
	if err != nil {
		m.log.Errorf("Failed to reset transient camera states: %v", err)
	}
}

// PairCamera pairs with a GoPro camera using BLE
func (m *GoProManager) PairCamera(cameraID string) (*database.ManagedCamera, error) {
	result, err := m.pairingManager.PairCamera(cameraID, m.BLEOperation, m.notify)
	if err != nil {
		return result, err
	}

	// After pairing, run a status check to gather battery/media info and sleep the camera
	if cs, ok := m.db.GetCameraByID(cameraID); ok && cs.Status.IsReachable {
		select {
		case m.statusCheckRequest <- cameraID:
		default:
		}
	}

	return result, nil
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (m *GoProManager) ForceSync(cameraID string) (*database.SyncQueueEntry, error) {
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

// GetVideosByCamera returns videos for a specific camera
func (m *GoProManager) GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*database.VideoFile, int) {
	return m.db.GetVideosByCamera(cameraID, startDate, endDate, limit, offset)
}
