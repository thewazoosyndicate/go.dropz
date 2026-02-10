package manager

import (
	"context"
	"fmt"
	"strings"
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
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mutex     sync.RWMutex
	isRunning bool

	// Active sync tasks
	activeSyncTasks map[string]*syncpkg.SyncTask

	// Immediate sync trigger channel
	immediateSyncTrigger chan struct{}

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

	// Enable BLE adapter directly (not in goroutine - breaks DBus thread affinity)
	if err := adapter.Enable(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to enable BLE adapter: %v", err)
	}

	bleManager := ble.NewManager(adapter, log)

	// Set up metadata update callback to update database whenever camera connects
	bleManager.SetMetadataCallback(func(metadata ble.CameraMetadata) {
		log.Debugf("Metadata callback invoked for camera %s", metadata.MACAddress)
		
		// Update camera WiFi credentials if they've changed
		if state, exists := db.CameraStates[metadata.MACAddress]; exists {
			if metadata.WiFiSSID != "" && metadata.WiFiPassword != "" {
				state.Camera.WiFiSSID = metadata.WiFiSSID
				state.Camera.WiFiPassword = metadata.WiFiPassword
			}
		}
		
		// Update camera metadata in database
		dbMetadata := database.CameraMetadata{
			Model:           metadata.ModelName,
			FirmwareVersion: metadata.FirmwareVersion,
			SerialNumber:    metadata.SerialNumber,
			BatteryLevel:    int32(metadata.BatteryLevel),
		}
		
		if err := db.SetCameraMetadata(metadata.MACAddress, dbMetadata); err != nil {
			log.Errorf("Failed to update camera metadata in database: %v", err)
		} else {
			log.Infof("Updated metadata for camera %s: battery=%d%%, model=%s", 
				metadata.MACAddress, metadata.BatteryLevel, metadata.ModelName)
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
		activeSyncTasks:      make(map[string]*syncpkg.SyncTask),
		immediateSyncTrigger: make(chan struct{}, 1),
	}

	// Initialize component managers
	manager.syncCoordinator = syncpkg.NewCoordinator(db, bleManager, log)
	manager.discoveryProcessor = discovery.NewProcessor(db, log)
	manager.pairingManager = pairing.NewManager(db, bleManager, log, ctx)

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
	cameraState, found := m.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	// Check if camera is already managed
	if cameraState.Status.IsManaged {
		// Already managed, return the managed view
		managedCamera, _ := m.db.GetManagedCamera(cameraState.Camera.MACAddress)
		return managedCamera, nil
	}

	// Set camera as managed
	cameraState.Status.IsManaged = true
	m.db.SaveChanges()

	m.log.Infof("Camera %s added to managed pool", cameraID)

	m.notify()

	// If we're in pair mode, queue pairing for the camera
	config := m.db.GetConfig()
	if config.PairModeEnabled {
		// Check actual device pairing state instead of just database state
		macAddress := cameraState.Camera.MACAddress
		devicePaired, err := m.ble.IsPaired(macAddress)

		if err != nil {
			// If we can't check device state, fall back to database state
			m.log.Tracef("Failed to check device pairing state for %s, using database state: %v",
				cameraState.Camera.Name, err)
			devicePaired = cameraState.Status.IsPaired
		} else if devicePaired != cameraState.Status.IsPaired {
			// Update database to match device state
			m.log.Infof("Updating database pairing state for camera %s: database=%v, device=%v",
				cameraState.Camera.Name, cameraState.Status.IsPaired, devicePaired)
			if updateErr := m.db.SetCameraPaired(macAddress, devicePaired); updateErr != nil {
				m.log.Warnf("Failed to update pairing state: %v", updateErr)
			}
		}

		if !devicePaired {
			m.log.Infof("Pair mode enabled, starting pairing for camera %s", cameraState.Camera.Name)
			m.PairCamera(cameraID)
		} else {
			m.log.Infof("Camera %s is already paired on device", cameraState.Camera.Name)
		}
	}

	// Return the managed camera view
	managedCamera, _ := m.db.GetManagedCamera(cameraState.Camera.MACAddress)
	return managedCamera, nil
}

// UnmanageCamera removes a camera from the managed pool
func (m *GoProManager) UnmanageCamera(cameraID string) error {
	cameraState, found := m.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	// Set camera as not managed
	m.db.ToggleCameraManaged(cameraState.Camera.MACAddress)

	m.log.Infof("Camera %s removed from managed pool", cameraID)

	m.notify()

	return nil
}

// BLEOperation executes a BLE operation with retry logic
func (m *GoProManager) BLEOperation(ctx context.Context, operationName string, operation func() error) error {
	m.log.Tracef("Starting BLE operation: %s", operationName)

	// Execute the operation with retry logic for specific errors
	var opErr error
	for opTry := 0; opTry < 3; opTry++ {
		opErr = operation()

		// If operation succeeded, return success
		if opErr == nil {
			return nil
		}

		// Check for context.Canceled which may happen when trying to scan after stopping
		if opErr == context.Canceled {
			m.log.Warnf("Context canceled during operation (likely due to scan being forcibly stopped)")

			// For important operations, wait and retry
			if strings.Contains(operationName, "Sync") || strings.HasPrefix(operationName, "Connect") ||
				strings.Contains(operationName, "EnableWifi") {
				m.log.Infof("Waiting 3 seconds before retrying critical operation attempt %d/3", opTry+1)
				time.Sleep(3 * time.Second)
				continue
			}

			// For non-critical operations, just return the error
			return opErr
		}

		// Check for BlueZ DBus interface errors that might be transient
		if strings.Contains(opErr.Error(), "Properties.GetAll") {
			m.log.Warnf("BlueZ DBus interface error detected during %s: %v", operationName, opErr)
			m.log.Infof("Waiting %d seconds before retry attempt %d/3", (opTry+1)*3, opTry+1)
			time.Sleep(time.Duration(opTry+1) * 3 * time.Second)
			continue
		}

		// For other errors, no retry, just return
		return opErr
	}

	// If we get here, we've tried multiple times but still have an error
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
	return m.pairingManager.PairCamera(cameraID, m.BLEOperation, m.notifier)
}

// syncDevicePairingStates checks actual device pairing states and updates database accordingly
func (m *GoProManager) syncDevicePairingStates() {
	m.pairingManager.SyncDevicePairingStates(m.notifier)
}

// VerifyAndFixPairingState checks a specific camera's pairing state and fixes database if needed
func (m *GoProManager) VerifyAndFixPairingState(cameraID string) error {
	return m.pairingManager.VerifyAndFixPairingState(cameraID, m.notifier)
}
