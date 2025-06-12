package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/logger"
	configpkg "github.com/dropz/dropz/pkg/manager/config"
	"github.com/dropz/dropz/pkg/manager/discovery"
	"github.com/dropz/dropz/pkg/manager/groups"
	"github.com/dropz/dropz/pkg/manager/pairing"
	syncpkg "github.com/dropz/dropz/pkg/manager/sync"
	"github.com/dropz/dropz/pkg/queue"
	"tinygo.org/x/bluetooth"
)

const (
	// Task types
	TaskTypeConnect       = "connect"
	TaskTypeSetDateTime   = "set_datetime"
	TaskTypeEnableWifi    = "enable_wifi"
	TaskTypeDownloadMedia = "download_media"

	// Task priorities
	PriorityHigh   = 3
	PriorityNormal = 2
	PriorityLow    = 1

	// Sync priorities
	SyncPriorityManual = 10 // Manual sync requests (ForceSync)
	SyncPriorityAuto   = 5  // Automatic sync requests
)

// Device represents a BLE device
type Device struct {
	Name        string
	MACAddress  string
	RSSI        int32
	IsConnected bool
}

// GoProManager is the main service that coordinates all GoPro operations
type GoProManager struct {
	// Core dependencies
	db       *database.Database
	ble      ble.BLEInterface
	queue    *queue.TaskQueue
	log      logger.Logger
	notifier common.UpdateNotifier

	// Configuration
	scanInterval   time.Duration
	connectTimeout time.Duration
	inactivityTime time.Duration
	setTimeEnabled bool
	daysThreshold  int
	destinationDir string

	// Runtime control
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mutex     sync.RWMutex
	isRunning bool

	// Worker pool for I/O operations
	ioWorkerPool chan struct{}

	// Active sync tasks
	activeSyncTasks map[string]*syncpkg.SyncTask

	// Processing devices
	processingDevices map[string]struct{}

	// Immediate sync trigger channel
	immediateSyncTrigger chan struct{}

	// Component managers
	syncCoordinator    *syncpkg.Coordinator
	groupManager       *groups.Manager
	discoveryProcessor *discovery.Processor
	pairingManager     *pairing.Manager
	configManager      *configpkg.Manager
}

// NewGoProManager creates a new GoPro manager instance
func NewGoProManager(dbPath, destinationDir string) (*GoProManager, error) {
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

	// Enable BLE adapter with timeout
	enableCtx, enableCancel := context.WithTimeout(ctx, 5*time.Second)
	defer enableCancel()

	enableCh := make(chan error, 1)
	go func() {
		enableCh <- adapter.Enable()
	}()

	select {
	case err := <-enableCh:
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to enable BLE adapter: %w", err)
		}
	case <-enableCtx.Done():
		cancel()
		return nil, fmt.Errorf("timeout while enabling BLE adapter")
	}

	bleManager := ble.NewManager(adapter, logger.GetLogger())

	if err := bleManager.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start BLE manager: %v", err)
	}

	// Initialize task queue with 5 workers and 30 second retry interval
	taskQueue := queue.NewTaskQueue(5, 30*time.Second)

	// Load configuration from database
	config := db.GetConfig()

	// Prefer database destination folder if it exists
	folderFromDb := config.DestinationFolder
	if folderFromDb != "" {
		destinationDir = folderFromDb
	}

	// Create the manager with configuration from database
	manager := &GoProManager{
		ctx:            ctx,
		cancel:         cancel,
		ble:            bleManager,
		db:             db,
		log:            logger.GetLogger(),
		scanInterval:   time.Duration(config.ScanIntervalSeconds) * time.Second,
		connectTimeout: time.Duration(config.ConnectTimeoutSeconds) * time.Second,
		inactivityTime: time.Duration(config.InactivityTimeoutSeconds) * time.Second,
		setTimeEnabled: config.SetTimeEnabled,
		daysThreshold:  int(config.DaysThreshold),
		destinationDir: destinationDir,
		queue:          taskQueue,
		// Allow up to 10 concurrent I/O operations
		ioWorkerPool: make(chan struct{}, 10),
		// Initialize active sync tasks
		activeSyncTasks: make(map[string]*syncpkg.SyncTask),
		// Initialize processing devices
		processingDevices: make(map[string]struct{}),
		// Initialize immediate sync trigger channel
		immediateSyncTrigger: make(chan struct{}, 1),
	}

	// Initialize component managers
	manager.configManager = configpkg.NewManager(db, logger.GetLogger())
	manager.syncCoordinator = syncpkg.NewCoordinator(db, bleManager, logger.GetLogger())
	manager.groupManager = groups.NewManager(db, logger.GetLogger())
	manager.discoveryProcessor = discovery.NewProcessor(db, logger.GetLogger())
	manager.pairingManager = pairing.NewManager(db, bleManager, logger.GetLogger(), nil, ctx)

	return manager, nil
}

// SetNotifier sets the update notifier component
func (m *GoProManager) SetNotifier(notifier common.UpdateNotifier) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.notifier = notifier
}

// ManageCamera adds a camera to the managed camera pool
func (m *GoProManager) ManageCamera(cameraID string) (*database.ManagedCamera, error) {
	// Find the camera
	var cameraState *database.CameraWithState

	// Check all cameras
	for _, state := range m.db.CameraStates {
		if state.Camera.ID == cameraID {
			cameraState = state
			break
		}
	}

	if cameraState == nil {
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

	// Notify immediately after adding to managed pool
	m.mutex.RLock()
	notifier := m.notifier
	m.mutex.RUnlock()
	if notifier != nil {
		m.log.Debug("Notifying observers about managed camera addition")
		notifier.NotifyUpdate()
	}

	// If we're in pair mode, queue pairing for the camera
	config := m.db.GetConfig()
	if config.PairModeEnabled {
		// Check actual device pairing state instead of just database state
		macAddress := cameraState.Camera.MACAddress
		devicePaired, err := m.ble.IsPaired(macAddress)

		if err != nil {
			// If we can't check device state, fall back to database state
			m.log.Debugf("Failed to check device pairing state for %s, using database state: %v",
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
			go m.PairCamera(cameraID)
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
	// Find the camera
	var cameraState *database.CameraWithState

	// Check all cameras
	for _, state := range m.db.CameraStates {
		if state.Camera.ID == cameraID {
			cameraState = state
			break
		}
	}

	if cameraState == nil {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	// Set camera as not managed
	m.db.ToggleCameraManaged(cameraState.Camera.MACAddress)

	m.log.Infof("Camera %s removed from managed pool", cameraID)

	// Notify immediately after updating
	m.mutex.RLock()
	notifier := m.notifier
	m.mutex.RUnlock()
	if notifier != nil {
		m.log.Debug("Notifying observers about camera pool change")
		notifier.NotifyUpdate()
	}

	return nil
}

// BLEOperation executes a BLE operation with retry logic
func (m *GoProManager) BLEOperation(ctx context.Context, operationName string, operation func() error) error {
	m.log.Debugf("Starting BLE operation: %s", operationName)

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
