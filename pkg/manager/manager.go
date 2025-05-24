package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/logger"
	"github.com/dropz/dropz/pkg/queue"
	"github.com/google/uuid"
	"tinygo.org/x/bluetooth"
)

// Ensure the uuid dependency is in go.mod
// run: go get github.com/google/uuid

// Update the Device type to match what's in the ble package
// Note: If the ble package has a different structure, adjust this code accordingly

// Assuming this is the Device struct in the ble package:
type Device struct {
	Name        string
	MACAddress  string
	RSSI        int32
	IsConnected bool
}

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
)

// SyncTask represents a camera sync task
type SyncTask struct {
	CameraID     string
	MACAddress   string
	CameraName   string
	StartedAt    time.Time
	CompletedAt  time.Time
	Success      bool
	ErrorMessage string
}

// GoProManager is the main service that coordinates all GoPro operations
type GoProManager struct {
	db             *database.Database
	ble            ble.BLEInterface
	queue          *queue.TaskQueue
	log            logger.Logger
	scanInterval   time.Duration
	connectTimeout time.Duration
	inactivityTime time.Duration
	setTimeEnabled bool
	daysThreshold  int
	destinationDir string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mutex          sync.RWMutex
	isRunning      bool
	// Add a worker pool for I/O operations
	ioWorkerPool chan struct{}
	// Notifier for device updates
	notifier common.UpdateNotifier
	// Active sync tasks
	activeSyncTasks map[string]*SyncTask
	// Processing devices
	processingDevices map[string]struct{}
}

// NewGoProManager creates a new GoPro manager instance
func NewGoProManager(dbPath, destinationDir string) (*GoProManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	db := database.GetDatabase()
	if err := db.Initialize(dbPath); err != nil {
		cancel() // Clean up the context
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

	bleManager := ble.NewManager(ble.ManagerConfig{
		Logger:  logger.GetBLEFilteredLogger(),
		Adapter: adapter,
	})

	if err := bleManager.Start(); err != nil {
		cancel() // Clean up the context
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
		destinationDir: destinationDir, // Use parameter value as default, but it can be overridden later
		queue:          taskQueue,
		// Allow up to 10 concurrent I/O operations
		ioWorkerPool: make(chan struct{}, 10),
		// Initialize active sync tasks
		activeSyncTasks: make(map[string]*SyncTask),
		// Initialize processing devices
		processingDevices: make(map[string]struct{}),
	}

	return manager, nil
}

// SetNotifier sets the update notifier component
func (m *GoProManager) SetNotifier(notifier common.UpdateNotifier) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.notifier = notifier
}

// processDiscoveredDevices handles the discovered devices from a BLE scan
func (m *GoProManager) processDiscoveredDevices(devices []ble.Device) {
	m.log.Debugf("Processing %d discovered devices", len(devices))

	// Use a wait group to process devices concurrently
	var wg sync.WaitGroup
	// Use a mutex to protect database operations
	var dbMutex sync.Mutex

	// Track if any changes were made that require a notification
	var changesMade atomic.Bool

	// First, mark all cameras as unreachable that haven't been seen recently
	markUnreachableDevices(m, &dbMutex, &changesMade)

	for _, device := range devices {
		wg.Add(1)

		// Process each device in a separate goroutine
		go func(dev ble.Device) {
			defer wg.Done()

			// Since we know the exact structure of the Device struct, we can use it directly
			name := dev.Name
			macAddress := dev.MACAddress
			rssi := dev.RSSI

			// Skip non-GoPro devices - only process devices with GoPro in the name
			if name == "" || !strings.Contains(strings.ToLower(name), "gopro") {
				return
			}

			// Check if device is already in the database
			dbMutex.Lock()
			discoveredCamera, exists := m.db.GetDiscoveredCamera(macAddress)
			dbMutex.Unlock()

			// Create a new discovered camera if it doesn't exist
			if !exists {
				// Create a new camera with state
				cameraState := &database.CameraWithState{
					Camera: database.Camera{
						ID:           uuid.New().String(),
						Name:         name,
						MACAddress:   macAddress,
						WiFiSSID:     "", // Will be set during pairing
						WiFiPassword: "", // Will be set during pairing
						RSSI:         rssi,
					},
					Status: database.CameraStatus{
						LastSeen:    time.Now(),
						LastSynced:  time.Time{}, // Zero time to indicate never synced
						IsPairing:   false,
						IsPaired:    false,
						IsManaged:   false,
						IsReachable: true, // It's reachable since we just discovered it
						IsSynced:    false,
						IsSyncing:   false,
					},
					GroupID: "",
					Metadata: database.CameraMetadata{
						ID: uuid.New().String(),
					},
				}

				discoveredCamera = &database.DiscoveredCamera{
					CameraState: cameraState,
				}

				m.log.Infof("New camera discovered: %s (%s)", name, macAddress)

				// Add to database
				dbMutex.Lock()
				err := m.db.AddOrUpdateDiscoveredCamera(discoveredCamera)
				dbMutex.Unlock()

				if err != nil {
					m.log.Errorf("Failed to add discovered camera: %v", err)
					return
				}

				// Notify immediately for new device discovery
				m.mutex.RLock()
				notifier := m.notifier
				m.mutex.RUnlock()

				if notifier != nil {
					m.log.Debug("Notifying observers about new device discovery")
					notifier.NotifyUpdate()
				}

				changesMade.Store(true)
			} else {
				// Camera exists, update its basic properties and status
				cameraState := discoveredCamera.CameraState

				// Update RSSI and name if needed
				if cameraState.Camera.RSSI != rssi {
					cameraState.Camera.RSSI = rssi
					changesMade.Store(true)
				}

				// Ensure name is set if it was empty before
				if cameraState.Camera.Name == "" || !strings.Contains(cameraState.Camera.Name, "GoPro") {
					cameraState.Camera.Name = name
					changesMade.Store(true)
				}

				// Update reachability status
				dbMutex.Lock()
				m.db.UpdateCameraReachability(macAddress, true)
				dbMutex.Unlock()

				changesMade.Store(true)
			}
		}(device)
	}

	// Wait for all device processing to complete
	wg.Wait()

	// Notify about device updates if any changes were made
	// This ensures we don't send unnecessary notifications
	if changesMade.Load() {
		m.mutex.RLock()
		notifier := m.notifier
		m.mutex.RUnlock()

		if notifier != nil {
			m.log.Debug("Notifying observers about device updates")
			notifier.NotifyUpdate()
		}
	}
}

// markUnreachableDevices marks cameras as unreachable if they haven't been seen recently
func markUnreachableDevices(m *GoProManager, dbMutex *sync.Mutex, changesMade *atomic.Bool) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	unreachableThreshold := time.Now().Add(-30 * time.Second)

	// Get all cameras
	for _, cameraState := range m.db.CameraStates {
		// If the camera was last seen more than 30 seconds ago and is currently marked as reachable,
		// mark it as unreachable
		if cameraState.Status.LastSeen.Before(unreachableThreshold) && cameraState.Status.IsReachable {
			m.log.Debugf("Marking camera %s as unreachable (last seen: %s)",
				cameraState.Camera.MACAddress, cameraState.Status.LastSeen.Format(time.RFC3339))

			cameraState.Status.IsReachable = false
			changesMade.Store(true)
		}
	}

	// Save changes if any were made
	if changesMade.Load() {
		err := m.db.SaveChanges()
		if err != nil {
			m.log.Errorf("Failed to save camera state changes: %v", err)
		}
	}
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

// Start starts the GoPro manager service
func (m *GoProManager) Start() error {
	m.log.Info("Starting GoPro manager service")

	// Start BLE manager
	m.ble.Start()

	// Start task queue
	m.queue.Start()

	// Start background scanner
	m.wg.Add(1)
	go m.startBackgroundScanner()

	// Start device manager
	m.wg.Add(1)
	go m.deviceManager()

	m.isRunning = true

	// ResetTransientStates resets transient camera states (is_syncing, is_pairing) on startup
	m.ResetTransientStates()

	// Perform initial sync of device pairing states with database
	go func() {
		// Wait a bit for BLE to be fully ready
		time.Sleep(5 * time.Second)
		m.log.Info("Performing initial device pairing state synchronization")
		m.syncDevicePairingStates()
	}()

	return nil
}

// Stop stops the GoPro manager service
func (m *GoProManager) Stop() {
	m.log.Infof("GoProManager.Stop() called. isRunning: %v", m.isRunning)
	if !m.isRunning {
		m.log.Info("GoProManager.Stop() called but manager was not running.")
		return
	}

	m.log.Info("Stopping GoPro manager service...")

	m.log.Debug("Cancelling GoProManager context (for backgroundScanner, deviceManager)...")
	m.cancel() // Signal all internal goroutines to stop
	m.log.Debug("GoProManager context cancelled.")

	m.log.Debug("Attempting to stop task queue...")
	m.queue.Stop() // This should be idempotent and handle being called multiple times
	m.log.Debug("Task queue stop requested/completed.")

	m.log.Debug("Attempting to stop BLE manager...")
	m.ble.Stop() // This should also be idempotent
	m.log.Debug("BLE manager stop requested/completed.")

	m.log.Debug("Waiting for GoProManager internal goroutines (backgroundScanner, deviceManager) to finish...")
	m.wg.Wait() // Wait for backgroundScanner and deviceManager
	m.log.Debug("GoProManager internal goroutines finished.")

	m.isRunning = false
	m.log.Info("GoPro manager service stopped successfully.")
}

// startContinuousScan initiates a continuous BLE scan process
func (m *GoProManager) startContinuousScan(ctx context.Context, scanInProgress *atomic.Bool) {
	// Mark scan as in progress
	scanInProgress.Store(true)
	defer scanInProgress.Store(false)

	m.log.Debug("Starting continuous BLE scanning process")

	// Loop until context is canceled or other conditions stop the scan
	for {
		select {
		case <-ctx.Done():
			m.log.Debug("Continuous scan stopping due to context cancellation")
			return
		default:
			// Continue with the scan
		}

		// Create a scan context with timeout for this single scan cycle
		scanCtx, cancel := context.WithTimeout(ctx, 10*time.Second)

		m.log.Debug("Starting BLE scan cycle...")
		err := m.ble.StartScanning(scanCtx)
		if err != nil {
			cancel() // Always cancel the context
			m.log.Errorf("Failed to start BLE scan: %v", err)

			// Only short pause before retrying
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				// Continue with retry
			}
			continue
		}

		m.log.Debug("BLE scan started, waiting for scan to complete...")

		// Process devices immediately after scan completes or times out
		select {
		case <-scanCtx.Done():
			if scanCtx.Err() != context.Canceled {
				m.log.Debug("Scan cycle completed, processing discovered devices")
				// Process discovered devices
				devices := m.ble.GetDiscoveredDevices()
				m.processDiscoveredDevices(devices)
			}
		case <-ctx.Done():
			cancel()
			m.log.Debug("Parent context canceled during scan")
			return
		}

		cancel() // Always cancel the context

		// If we're in continuous mode, add a small pause between scans
		// to allow other BLE operations to occur
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
			// Short pause between scan cycles
		}
	}
}

// deviceManager periodically checks managed devices
func (m *GoProManager) deviceManager() {
	defer m.wg.Done()
	defer m.log.Debug("Device manager goroutine finished.")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	m.log.Info("Starting device manager")

	for {
		select {
		case <-m.ctx.Done():
			m.log.Debug("Device manager stopping due to context cancellation.")
			return
		case <-ticker.C:
			// Sync device pairing states with database
			m.syncDevicePairingStates()
			m.checkManagedDevices()
			// Process sync queue to start sync tasks for pending cameras
			m.processSyncQueue()
		}
	}
}

// checkManagedDevices checks all managed devices and queues tasks
func (m *GoProManager) checkManagedDevices() {
	managedCameras := m.db.GetAllManagedCameras()
	config := m.db.GetConfig()

	// Log the current sync configuration for debugging
	m.log.Debugf("Current sync configuration: SyncEnabled=%v, DaysThreshold=%v",
		config.SyncEnabled, config.DaysThreshold)

	for _, camera := range managedCameras {
		// Never change status for cameras that are actively syncing or pairing
		if camera.CameraState.Status.IsSyncing || camera.CameraState.Status.IsPairing {
			m.log.Debugf("Camera %s is actively syncing or pairing, preserving status during check",
				camera.CameraState.Camera.Name)
			continue
		}

		// Check if the camera is reachable
		if !camera.CameraState.Status.IsReachable {
			m.log.Debugf("Camera %s is not reachable", camera.CameraState.Camera.Name)
			continue
		}

		// Skip sync check if the camera is already synced
		if camera.CameraState.Status.IsSynced {
			m.log.Debugf("Camera %s is already synced, skipping check", camera.CameraState.Camera.Name)
			continue
		}

		// Check if sync is needed based on timestamp
		if !config.SyncEnabled {
			m.log.Debugf("Sync is disabled, not checking camera %s", camera.CameraState.Camera.Name)
			continue
		}

		// Check if camera is paired before attempting sync
		// Query device directly for most accurate state
		macAddress := camera.CameraState.Camera.MACAddress
		devicePaired, err := m.ble.IsPaired(macAddress)
		if err != nil {
			// If we can't check device state, fall back to database state
			m.log.Debugf("Failed to check device pairing state for %s, using database state: %v",
				camera.CameraState.Camera.Name, err)
			devicePaired = camera.CameraState.Status.IsPaired
		}

		if !devicePaired {
			m.log.Debugf("Camera %s is not paired, skipping sync", camera.CameraState.Camera.Name)
			continue
		}

		// Check if we should sync based on last sync time
		// If LastSynced is zero time or more than threshold days ago, queue for sync
		shouldSync := camera.CameraState.Status.LastSynced.IsZero() ||
			time.Since(camera.CameraState.Status.LastSynced) > time.Duration(config.DaysThreshold)*24*time.Hour

		if shouldSync {
			// Add to sync queue
			syncEntry := &database.SyncQueueEntry{
				CameraID:         camera.CameraState.Camera.ID,
				QueuedAt:         time.Now(),
				ProgressPercent:  0,
				CurrentOperation: "Waiting to start",
			}

			m.log.Infof("Adding camera %s to sync queue (threshold: %d days)",
				camera.CameraState.Camera.Name, config.DaysThreshold)

			if err := m.db.AddSyncQueueEntry(syncEntry); err != nil {
				m.log.Errorf("Failed to add camera %s to sync queue: %v",
					camera.CameraState.Camera.Name, err)
			}
		}
	}
}

// GetConfig returns the current configuration
func (m *GoProManager) GetConfig() database.Config {
	return m.db.GetConfig()
}

// UpdateConfig updates the configuration
func (m *GoProManager) UpdateConfig(config database.Config) error {
	// Update local variables
	m.scanInterval = time.Duration(config.ScanIntervalSeconds) * time.Second
	m.connectTimeout = time.Duration(config.ConnectTimeoutSeconds) * time.Second
	m.inactivityTime = time.Duration(config.InactivityTimeoutSeconds) * time.Second
	m.setTimeEnabled = config.SetTimeEnabled
	m.daysThreshold = int(config.DaysThreshold)

	// Update database
	return m.db.UpdateConfig(config)
}

// GetSetting gets a specific setting value
func (m *GoProManager) GetSetting(settingName string) (interface{}, error) {
	config := m.db.GetConfig()

	switch settingName {
	case "sync_enabled":
		return config.SyncEnabled, nil
	case "scan_interval_seconds":
		return config.ScanIntervalSeconds, nil
	case "connect_timeout_seconds":
		return config.ConnectTimeoutSeconds, nil
	case "days_threshold":
		return config.DaysThreshold, nil
	case "destination_folder":
		return config.DestinationFolder, nil
	case "inactivity_timeout_seconds":
		return config.InactivityTimeoutSeconds, nil
	case "inactivity_sync_interval_seconds":
		return config.InactivitySyncIntervalSeconds, nil
	case "set_time_enabled":
		return config.SetTimeEnabled, nil
	case "log_level":
		return config.LogLevel, nil
	default:
		return nil, fmt.Errorf("unknown setting: %s", settingName)
	}
}

// UpdateSetting updates a specific setting
func (m *GoProManager) UpdateSetting(settingName string, value interface{}) (database.Config, error) {
	config := m.db.GetConfig()

	switch settingName {
	case "pair_mode_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for pair_mode_enabled: expected bool")
		}
		config.PairModeEnabled = boolVal
		m.log.Infof("Updated pair_mode_enabled to %v", boolVal)

	case "sync_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for sync_enabled: expected bool")
		}
		config.SyncEnabled = boolVal
		m.log.Infof("Updated sync_enabled to %v", boolVal)

	case "scan_interval_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for scan_interval_seconds: expected number")
		}
		if intVal < 5 {
			return config, fmt.Errorf("scan_interval_seconds must be at least 5 seconds")
		}
		config.ScanIntervalSeconds = intVal
		m.log.Infof("Updated scan_interval_seconds to %d", intVal)

	case "connect_timeout_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for connect_timeout_seconds: expected number")
		}
		if intVal < 5 || intVal > 60 {
			return config, fmt.Errorf("connect_timeout_seconds must be between 5 and 60 seconds")
		}
		config.ConnectTimeoutSeconds = intVal
		m.log.Infof("Updated connect_timeout_seconds to %d", intVal)

	case "days_threshold":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for days_threshold: expected number")
		}
		if intVal < 1 {
			return config, fmt.Errorf("days_threshold must be at least 1 day")
		}
		config.DaysThreshold = intVal
		m.log.Infof("Updated days_threshold to %d", intVal)

	case "destination_folder":
		strVal, ok := value.(string)
		if !ok {
			return config, fmt.Errorf("invalid value type for destination_folder: expected string")
		}
		config.DestinationFolder = strVal
		m.log.Infof("Updated destination_folder to %s", strVal)

	case "inactivity_timeout_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for inactivity_timeout_seconds: expected number")
		}
		if intVal < 30 || intVal > 3600 {
			return config, fmt.Errorf("inactivity_timeout_seconds must be between 30 and 3600 seconds")
		}
		config.InactivityTimeoutSeconds = intVal
		m.log.Infof("Updated inactivity_timeout_seconds to %d", intVal)

	case "inactivity_sync_interval_seconds":
		var intVal int32
		switch v := value.(type) {
		case float64:
			intVal = int32(v)
		case int:
			intVal = int32(v)
		case int32:
			intVal = v
		default:
			return config, fmt.Errorf("invalid value type for inactivity_sync_interval_seconds: expected number")
		}
		if intVal < 60 || intVal > 3600 {
			return config, fmt.Errorf("inactivity_sync_interval_seconds must be between 60 and 3600 seconds")
		}
		config.InactivitySyncIntervalSeconds = intVal
		m.log.Infof("Updated inactivity_sync_interval_seconds to %d", intVal)

	case "set_time_enabled":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for set_time_enabled: expected bool")
		}
		config.SetTimeEnabled = boolVal
		m.log.Infof("Updated set_time_enabled to %v", boolVal)

	case "log_level":
		strVal, ok := value.(string)
		if !ok {
			return config, fmt.Errorf("invalid value type for log_level: expected string")
		}
		validLogLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}
		if !validLogLevels[strings.ToLower(strVal)] {
			return config, fmt.Errorf("invalid log_level: expected one of debug, info, warn, error")
		}
		config.LogLevel = strVal
		m.log.Infof("Updated log_level to %s", strVal)

	default:
		return config, fmt.Errorf("unknown setting: %s", settingName)
	}

	// Update the database
	if err := m.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config in database: %w", err)
	}

	// Update local variables
	m.scanInterval = time.Duration(config.ScanIntervalSeconds) * time.Second
	m.connectTimeout = time.Duration(config.ConnectTimeoutSeconds) * time.Second
	m.inactivityTime = time.Duration(config.InactivityTimeoutSeconds) * time.Second
	m.setTimeEnabled = config.SetTimeEnabled
	m.daysThreshold = int(config.DaysThreshold)

	return config, nil
}

// ResetSetting resets a specific setting to its default value
func (m *GoProManager) ResetSetting(settingName string) (database.Config, error) {
	config := m.db.GetConfig()
	defaultConfig := database.DefaultConfig()

	switch settingName {
	case "pair_mode_enabled":
		config.PairModeEnabled = defaultConfig.PairModeEnabled
		m.log.Infof("Reset pair_mode_enabled to default: %v", defaultConfig.PairModeEnabled)

	case "sync_enabled":
		config.SyncEnabled = defaultConfig.SyncEnabled
		m.log.Infof("Reset sync_enabled to default: %v", defaultConfig.SyncEnabled)

	case "scan_interval_seconds":
		config.ScanIntervalSeconds = defaultConfig.ScanIntervalSeconds
		m.log.Infof("Reset scan_interval_seconds to default: %d", defaultConfig.ScanIntervalSeconds)

	case "connect_timeout_seconds":
		config.ConnectTimeoutSeconds = defaultConfig.ConnectTimeoutSeconds
		m.log.Infof("Reset connect_timeout_seconds to default: %d", defaultConfig.ConnectTimeoutSeconds)

	case "days_threshold":
		config.DaysThreshold = defaultConfig.DaysThreshold
		m.log.Infof("Reset days_threshold to default: %d", defaultConfig.DaysThreshold)

	case "destination_folder":
		config.DestinationFolder = defaultConfig.DestinationFolder
		m.log.Infof("Reset destination_folder to default: %s", defaultConfig.DestinationFolder)

	case "inactivity_timeout_seconds":
		config.InactivityTimeoutSeconds = defaultConfig.InactivityTimeoutSeconds
		m.log.Infof("Reset inactivity_timeout_seconds to default: %d", defaultConfig.InactivityTimeoutSeconds)

	case "inactivity_sync_interval_seconds":
		config.InactivitySyncIntervalSeconds = defaultConfig.InactivitySyncIntervalSeconds
		m.log.Infof("Reset inactivity_sync_interval_seconds to default: %d", defaultConfig.InactivitySyncIntervalSeconds)

	case "set_time_enabled":
		config.SetTimeEnabled = defaultConfig.SetTimeEnabled
		m.log.Infof("Reset set_time_enabled to default: %v", defaultConfig.SetTimeEnabled)

	case "log_level":
		config.LogLevel = defaultConfig.LogLevel
		m.log.Infof("Reset log_level to default: %s", defaultConfig.LogLevel)

	case "all":
		// Reset all settings to default
		config = defaultConfig
		m.log.Info("Reset all settings to default values")

	default:
		return config, fmt.Errorf("unknown setting: %s", settingName)
	}

	// Update the database
	if err := m.db.UpdateConfig(config); err != nil {
		return config, fmt.Errorf("failed to update config in database: %w", err)
	}

	// Update local variables
	m.scanInterval = time.Duration(config.ScanIntervalSeconds) * time.Second
	m.connectTimeout = time.Duration(config.ConnectTimeoutSeconds) * time.Second
	m.inactivityTime = time.Duration(config.InactivityTimeoutSeconds) * time.Second
	m.setTimeEnabled = config.SetTimeEnabled
	m.daysThreshold = int(config.DaysThreshold)

	return config, nil
}

// BLEOperation executes a BLE operation
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
