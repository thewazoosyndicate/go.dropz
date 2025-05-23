package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/logger"
	"github.com/dropz/dropz/pkg/queue"
	"github.com/dropz/dropz/pkg/wifi"
	"github.com/google/uuid"
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
	ble            *ble.BLEManager
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

	bleManager, err := ble.NewBLEManager()
	if err != nil {
		cancel() // Clean up the context
		return nil, fmt.Errorf("failed to create BLE manager: %v", err)
	}

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

// processSyncQueue checks the sync queue and processes cameras that need syncing
func (m *GoProManager) processSyncQueue() {
	if !m.db.GetConfig().SyncEnabled {
		m.log.Debug("Sync is disabled, skipping sync queue processing")
		return
	}

	m.log.Debug("Processing sync queue")
	syncQueue := m.db.GetSyncQueue()

	for _, entry := range syncQueue {
		// Check if we already have an active sync task for this camera
		m.mutex.RLock()
		_, exists := m.activeSyncTasks[entry.CameraID]
		m.mutex.RUnlock()

		if exists {
			// Skip this entry, it's already being synced
			m.log.Debugf("Camera %s is already being synced, skipping", entry.CameraID)
			continue
		}

		// Get the camera details
		camera, found := m.db.GetManagedCameraByID(entry.CameraID)
		if !found {
			m.log.Warnf("Camera %s not found, removing from sync queue", entry.CameraID)
			m.db.RemoveSyncQueueEntry(entry.CameraID)
			continue
		}

		// Check if the camera is suitable for syncing
		if !camera.CameraState.Status.IsReachable {
			m.log.Debugf("Camera %s is not reachable, skipping sync", entry.CameraID)
			continue
		}

		if camera.CameraState.Status.IsSyncing {
			m.log.Debugf("Camera %s is already marked as syncing, skipping", entry.CameraID)
			continue
		}

		// Set camera as syncing
		m.log.Infof("Starting sync for camera %s", camera.CameraState.Camera.Name)
		m.db.UpdateCameraSyncingStatus(camera.CameraState.Camera.MACAddress, true)

		// Create and start a sync task
		syncTask := &SyncTask{
			CameraID:   camera.CameraState.Camera.ID,
			MACAddress: camera.CameraState.Camera.MACAddress,
			CameraName: camera.CameraState.Camera.Name,
			StartedAt:  time.Now(),
		}

		m.mutex.Lock()
		m.activeSyncTasks[entry.CameraID] = syncTask
		m.mutex.Unlock()

		// Start sync in background
		go m.performCameraSync(syncTask)
	}
}

// performCameraSync handles the actual syncing of a camera
func (m *GoProManager) performCameraSync(task *SyncTask) {
	// Remove task when done
	defer func() {
		m.mutex.Lock()
		delete(m.activeSyncTasks, task.CameraID)
		m.mutex.Unlock()
	}()

	m.log.Infof("Syncing camera %s (%s)", task.CameraName, task.MACAddress)

	// Get the camera details to work with
	camera, found := m.db.GetManagedCameraByID(task.CameraID)
	if !found {
		m.log.Errorf("Cannot sync camera %s: not found in database", task.CameraID)
		return
	}

	// Create sync queue entry if it doesn't exist
	syncEntry := &database.SyncQueueEntry{
		CameraID:         task.CameraID,
		QueuedAt:         time.Now(),
		ProgressPercent:  0,
		CurrentOperation: "Starting sync",
	}

	// Set camera as syncing in database
	if err := m.db.UpdateCameraSyncingStatus(task.MACAddress, true); err != nil {
		m.log.Errorf("Failed to mark camera as syncing: %v", err)
	}

	// Make sure the syncing flag is reset when done, regardless of the outcome
	defer func() {
		if err := m.db.UpdateCameraSyncingStatus(task.MACAddress, false); err != nil {
			m.log.Errorf("Failed to reset syncing status: %v", err)
		}
	}()

	// Get the current config
	config := m.db.GetConfig()

	// Create a context for the sync operation
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Minute)
	defer cancel()

	// Real implementation of camera sync process
	// Step 1: Connect to camera via BLE
	syncEntry.CurrentOperation = "Connecting via BLE"
	syncEntry.ProgressPercent = 10
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	// Notify about the status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Use BLEOperation to handle concurrency and semaphores correctly
	connErr := m.BLEOperation(ctx, "ConnectForSync", func() error {
		return m.ble.Connect(camera.CameraState.Camera.MACAddress)
	})

	if connErr != nil {
		syncEntry.CurrentOperation = "BLE Connection Failed"
		// Just update the entry without setting a status enum
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to connect to camera via BLE: %v", connErr)
		return
	}

	// Ensure BLE disconnection happens in all cases
	defer func() {
		disconnectErr := m.BLEOperation(ctx, "DisconnectAfterSync", func() error {
			return m.ble.Disconnect(camera.CameraState.Camera.MACAddress)
		})
		if disconnectErr != nil {
			m.log.Warnf("Failed to disconnect from BLE after sync: %v", disconnectErr)
		}
	}()

	// Step 2: Enable WiFi on the camera
	syncEntry.CurrentOperation = "Enabling WiFi"
	syncEntry.ProgressPercent = 30
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	wifiErr := m.BLEOperation(ctx, "EnableWifi", func() error {
		return m.ble.EnableWifi(camera.CameraState.Camera.MACAddress)
	})

	if wifiErr != nil {
		syncEntry.CurrentOperation = "WiFi Enabling Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to enable WiFi: %v", wifiErr)
		return
	}

	// Give the camera a moment to fully enable WiFi
	select {
	case <-ctx.Done():
		m.log.Warnf("Sync operation canceled: %v", ctx.Err())
		return
	case <-time.After(2 * time.Second):
		// Continue to next step
	}

	// Step 3: Connect to WiFi
	syncEntry.CurrentOperation = "Connecting to WiFi"
	syncEntry.ProgressPercent = 50
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Create WiFi manager instance
	wifiManager, err := wifi.NewWiFiManager()
	if err != nil {
		syncEntry.CurrentOperation = "WiFi Manager Creation Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to create WiFi manager: %v", err)
		return
	}

	// Connect to WiFi using stored credentials
	wifiCtx, wifiCancel := context.WithTimeout(ctx, time.Duration(config.ConnectTimeoutSeconds)*time.Second)
	defer wifiCancel()

	err = wifiManager.Connect(wifiCtx, camera.CameraState.Camera.WiFiSSID, camera.CameraState.Camera.WiFiPassword)
	if err != nil {
		syncEntry.CurrentOperation = "WiFi Connection Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to connect to camera WiFi: %v", err)
		return
	}

	// Ensure we disconnect from WiFi when done
	defer func() {
		disconnectErr := wifiManager.Disconnect()
		if disconnectErr != nil {
			m.log.Warnf("Failed to disconnect from WiFi: %v", disconnectErr)
		}
	}()

	// Step 4: Download media
	syncEntry.CurrentOperation = "Downloading media"
	syncEntry.ProgressPercent = 70
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%)",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent)

	// Ensure destination directory exists
	if err := os.MkdirAll(config.DestinationFolder, 0755); err != nil {
		syncEntry.CurrentOperation = "Failed to create destination folder"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to create destination folder: %v", err)
		return
	}

	// Download videos from the last X days
	downloadCtx, downloadCancel := context.WithTimeout(ctx, 20*time.Minute)
	defer downloadCancel()

	downloadedFiles, err := wifiManager.DownloadVideos(downloadCtx, config.DestinationFolder, int(config.DaysThreshold))
	if err != nil {
		syncEntry.CurrentOperation = "Media Download Failed"
		if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
			m.log.Errorf("Failed to update sync queue entry: %v", err)
		}
		m.log.Errorf("Failed to download media: %v", err)
		return
	}

	// Step 5: Process files (move to final location, add metadata, etc.)
	syncEntry.CurrentOperation = "Processing files"
	syncEntry.ProgressPercent = 90
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry progress: %v", err)
	}

	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	m.log.Infof("Sync for camera %s: %s (%d%%), downloaded %d files",
		camera.CameraState.Camera.Name, syncEntry.CurrentOperation, syncEntry.ProgressPercent, len(downloadedFiles))

	// Create a folder for the camera if it doesn't exist
	cameraFolder := filepath.Join(config.DestinationFolder, camera.CameraState.Camera.Name)
	if err := os.MkdirAll(cameraFolder, 0755); err != nil {
		m.log.Warnf("Failed to create camera-specific folder: %v", err)
		// Continue anyway, use the main destination folder
	} else {
		// Move files to camera-specific folder if needed
		for _, file := range downloadedFiles {
			if filepath.Dir(file) != cameraFolder {
				newPath := filepath.Join(cameraFolder, filepath.Base(file))
				if err := os.Rename(file, newPath); err != nil {
					m.log.Warnf("Failed to move file %s to camera folder: %v", file, err)
				} else {
					m.log.Debugf("Moved file to camera folder: %s -> %s", file, newPath)
				}
			}
		}
	}

	// Mark sync as complete - set appropriate boolean flags instead of status enum
	m.db.MarkCameraSynced(task.MACAddress)
	m.log.Infof("Camera %s synced successfully", task.CameraName)

	// Update completion status in sync entry
	syncEntry.ProgressPercent = 100
	syncEntry.CurrentOperation = "Completed"
	if err := m.db.UpdateSyncQueueEntry(syncEntry); err != nil {
		m.log.Errorf("Failed to update sync queue entry on completion: %v", err)
	}

	// Remove from sync queue
	m.db.RemoveSyncQueueEntry(task.CameraID)

	// Update completion status
	task.CompletedAt = time.Now()
	task.Success = true

	// Notify about the status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
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
	if config.PairModeEnabled && !cameraState.Status.IsPaired {
		m.log.Infof("Pair mode enabled, starting pairing for camera %s", cameraState.Camera.Name)
		go m.PairCamera(cameraID)
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

// startBackgroundScanner starts the background scanner
func (m *GoProManager) startBackgroundScanner() {
	m.log.Info("Starting background scanner")

	// Create a ticker for regular scan intervals
	ticker := time.NewTicker(m.scanInterval)
	defer ticker.Stop()

	// Create a watchdog ticker to ensure scanning is active
	watchdogTicker := time.NewTicker(1 * time.Minute)
	defer watchdogTicker.Stop()

	// Track scan state
	var scanInProgress atomic.Bool

	// Create initial scan context
	continuousScanCtx, cancelContinuousScan := context.WithCancel(m.ctx)
	defer cancelContinuousScan()

	// Start initial scan
	go m.startContinuousScan(continuousScanCtx, &scanInProgress)

	for {
		select {
		case <-m.ctx.Done():
			m.log.Debug("Background scanner stopping due to context cancellation.")
			cancelContinuousScan()
			return

		case <-ticker.C:
			// If we're not actively scanning, restart the continuous scanning process
			if !scanInProgress.Load() {
				m.log.Info("Regular scan interval triggered, restarting continuous scan")
				// Cancel any existing scan and create a new context
				cancelContinuousScan()
				continuousScanCtx, cancelContinuousScan = context.WithCancel(m.ctx)

				go m.startContinuousScan(continuousScanCtx, &scanInProgress)
			}

		case <-watchdogTicker.C:
			// Check if scanning is active, if not restart it
			if !scanInProgress.Load() {
				m.log.Info("Watchdog detected scan not running, restarting continuous scan")
				// Cancel any existing scan and create a new context
				cancelContinuousScan()
				continuousScanCtx, cancelContinuousScan = context.WithCancel(m.ctx)

				go m.startContinuousScan(continuousScanCtx, &scanInProgress)
			}
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

// PairCamera pairs with a GoPro camera using BLE
func (m *GoProManager) PairCamera(cameraID string) (*database.ManagedCamera, error) {
	// First check if we can find the camera
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

	// Mark camera as pairing
	m.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, true)
	m.log.Infof("Starting pairing process for camera %s", cameraState.Camera.Name)

	// Notify about the status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	// TODO: Implement actual pairing logic here
	// For now, just simulate a successful pairing
	// Create a context with a reasonable timeout for pairing operation
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	// Perform the BLE pairing operation
	bleErr := m.BLEOperation(ctx, "PairCamera", func() error {
		// Get the MAC address, which is used for BLE operations
		macAddress := cameraState.Camera.MACAddress

		// Connect to the device
		m.log.Infof("Connecting to device %s for pairing", macAddress)
		if err := m.ble.Connect(macAddress); err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}

		// Get WiFi credentials
		ssid, password, err := m.ble.GetWifiCredentials(macAddress)
		if err != nil {
			// Try to disconnect gracefully even if getting WiFi credentials failed
			_ = m.ble.Disconnect(macAddress)
			return fmt.Errorf("failed to get WiFi credentials: %v", err)
		}

		// Store WiFi credentials in the database
		m.log.Debugf("Obtained WiFi credentials for camera %s: SSID=%s", cameraID, ssid)

		// Update the camera's WiFi credentials in the cameraState object
		cameraState.Camera.WiFiSSID = ssid
		cameraState.Camera.WiFiPassword = password

		// Disconnect from the camera
		if err := m.ble.Disconnect(macAddress); err != nil {
			m.log.Warnf("Failed to disconnect from camera: %v", err)
			// This is not critical, we can continue
		}

		return nil
	})

	// Handle any BLE operation errors
	if bleErr != nil {
		m.log.Errorf("BLE pairing operation failed: %v", bleErr)
		// Make sure pairing flag is reset if BLE operation failed
		m.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)
		return nil, fmt.Errorf("failed in BLE pairing operation: %v", bleErr)
	}

	// Mark camera as paired and ensure pairing flag is reset
	pairErr := m.db.SetCameraPaired(cameraState.Camera.MACAddress, true)
	if pairErr != nil {
		m.log.Errorf("Error setting camera paired status: %v", pairErr)
		// Make sure pairing flag is reset even if there was an error
		m.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)
		return nil, fmt.Errorf("failed to set camera paired status: %v", pairErr)
	}

	// Double-check the paired status was correctly set
	updatedState, exists := m.db.CameraStates[cameraState.Camera.MACAddress]
	if exists {
		m.log.Infof("Camera %s paired successfully. isPaired=%v, isPairing=%v",
			cameraState.Camera.Name, updatedState.Status.IsPaired, updatedState.Status.IsPairing)
	} else {
		m.log.Warnf("Camera state not found after pairing for %s", cameraState.Camera.MACAddress)
	}

	// Notify about the status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	// Return the managed camera view
	managedCamera, exists := m.db.GetManagedCamera(cameraState.Camera.MACAddress)

	// If the camera is paired but not managed, it won't be returned by GetManagedCamera
	// In this case, we need to create a ManagedCamera wrapper around the CameraState
	// to return to the caller
	if !exists {
		m.log.Debugf("Camera %s is paired but not yet managed, creating managed view", cameraState.Camera.Name)
		managedCamera = &database.ManagedCamera{
			CameraState: cameraState,
		}
	}

	return managedCamera, nil
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (m *GoProManager) ForceSync(cameraID string) (*database.SyncQueueEntry, error) {
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

	// Check if camera is already in the sync queue
	for _, entry := range m.db.GetSyncQueue() {
		if entry.CameraID == cameraID {
			m.log.Infof("Camera %s is already in the sync queue", cameraState.Camera.Name)
			return entry, nil
		}
	}

	// Create a new sync queue entry
	syncEntry := &database.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		ProgressPercent:  0,
		CurrentOperation: "Waiting to start",
	}

	// Add to sync queue
	if err := m.db.AddSyncQueueEntry(syncEntry); err != nil {
		return nil, fmt.Errorf("failed to add camera to sync queue: %v", err)
	}

	// Mark the camera as not synced so it will appear in sync queue
	m.db.ResetSyncStatus(cameraState.Camera.MACAddress)

	m.log.Infof("Added camera %s to sync queue", cameraState.Camera.Name)

	// Notify immediately
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	return syncEntry, nil
}

// CreateGroup creates a new group with the specified cameras
func (m *GoProManager) CreateGroup(name string, cameraIDs []string) (*database.Group, error) {
	// Create a new group
	group := &database.Group{
		ID:        uuid.New().String(),
		Name:      name,
		CameraIDs: cameraIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add to database
	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create group: %v", err)
	}

	// Update camera records to associate with this group
	for _, cameraID := range cameraIDs {
		// Find the camera in database
		found := false
		for _, state := range m.db.CameraStates {
			if state.Camera.ID == cameraID {
				state.GroupID = group.ID
				found = true
				break
			}
		}

		if !found {
			m.log.Warnf("Camera %s not found when adding to group", cameraID)
		}
	}

	// Save changes
	if err := m.db.SaveChanges(); err != nil {
		m.log.Warnf("Failed to update camera group associations: %v", err)
	}

	m.log.Infof("Created group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// UpdateGroup updates an existing group
func (m *GoProManager) UpdateGroup(groupID, name string, cameraIDs []string) (*database.Group, error) {
	// Get the existing group
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("group with ID %s not found", groupID)
	}

	// Update group details
	group.Name = name

	// Get the original camera IDs for comparison
	originalCameraIDs := make(map[string]bool)
	for _, cameraID := range group.CameraIDs {
		originalCameraIDs[cameraID] = true
	}

	// Update camera list
	group.CameraIDs = cameraIDs
	group.UpdatedAt = time.Now()

	// Add to database
	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to update group: %v", err)
	}

	// Update camera records that are newly added to this group
	for _, cameraID := range cameraIDs {
		if !originalCameraIDs[cameraID] {
			// This is a newly added camera
			found := false
			for _, state := range m.db.CameraStates {
				if state.Camera.ID == cameraID {
					state.GroupID = group.ID
					found = true
					break
				}
			}

			if !found {
				m.log.Warnf("Camera %s not found when adding to group", cameraID)
			}
		}
	}

	// Remove the group association from cameras that were removed from the group
	for cameraID := range originalCameraIDs {
		stillInGroup := false
		for _, id := range cameraIDs {
			if id == cameraID {
				stillInGroup = true
				break
			}
		}

		if !stillInGroup {
			// This camera was removed from the group
			found := false
			for _, state := range m.db.CameraStates {
				if state.Camera.ID == cameraID && state.GroupID == groupID {
					state.GroupID = ""
					found = true
					break
				}
			}

			if !found {
				m.log.Warnf("Camera %s not found when removing from group", cameraID)
			}
		}
	}

	// Save changes
	if err := m.db.SaveChanges(); err != nil {
		m.log.Warnf("Failed to update camera group associations: %v", err)
	}

	m.log.Infof("Updated group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// DeleteGroup deletes a group
func (m *GoProManager) DeleteGroup(groupID string) error {
	// Get the existing group
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group with ID %s not found", groupID)
	}

	// Remove the group association from all cameras in the group
	for _, cameraID := range group.CameraIDs {
		found := false
		for _, state := range m.db.CameraStates {
			if state.Camera.ID == cameraID && state.GroupID == groupID {
				state.GroupID = ""
				found = true
				break
			}
		}

		if !found {
			m.log.Warnf("Camera %s not found when removing from group", cameraID)
		}
	}

	// Remove the group from the database
	if err := m.db.RemoveGroup(groupID); err != nil {
		return fmt.Errorf("failed to delete group: %v", err)
	}

	// Save changes
	if err := m.db.SaveChanges(); err != nil {
		m.log.Warnf("Failed to update camera group associations: %v", err)
	}

	m.log.Infof("Deleted group %s", group.Name)
	return nil
}

// GetAllVideos returns all videos
func (m *GoProManager) GetAllVideos() []*database.VideoFile {
	return m.db.GetAllVideos()
}

// GetVideosByCamera returns videos for a specific camera
func (m *GoProManager) GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*database.VideoFile, int) {
	videos := m.db.GetVideosByCamera(cameraID)

	// Filter by date range
	var filtered []*database.VideoFile
	for _, video := range videos {
		if !startDate.IsZero() && video.CreatedAt.Before(startDate) {
			continue
		}
		if !endDate.IsZero() && video.CreatedAt.After(endDate) {
			continue
		}
		filtered = append(filtered, video)
	}

	// Apply pagination
	totalCount := len(filtered)
	if offset >= totalCount {
		return []*database.VideoFile{}, totalCount
	}

	end := offset + limit
	if end > totalCount {
		end = totalCount
	}

	return filtered[offset:end], totalCount
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
	case "pair_mode_enabled":
		return config.PairModeEnabled, nil
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
	case "set_time_enabled":
		return config.SetTimeEnabled, nil
	case "log_level":
		return config.LogLevel, nil
	case "debug_mode":
		return config.DebugMode, nil
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
		// Handle both float64 (common from JSON) and int
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
		// Handle both float64 (common from JSON) and int
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
		// Handle both float64 (common from JSON) and int
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
		// Handle both float64 (common from JSON) and int
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
		if intVal < 10 || intVal > 300 {
			return config, fmt.Errorf("inactivity_timeout_seconds must be between 10 and 300 seconds")
		}
		config.InactivityTimeoutSeconds = intVal
		m.log.Infof("Updated inactivity_timeout_seconds to %d", intVal)

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
		// Validate log level
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

	case "debug_mode":
		boolVal, ok := value.(bool)
		if !ok {
			return config, fmt.Errorf("invalid value type for debug_mode: expected bool")
		}
		config.DebugMode = boolVal
		m.log.Infof("Updated debug_mode to %v", boolVal)

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

	case "set_time_enabled":
		config.SetTimeEnabled = defaultConfig.SetTimeEnabled
		m.log.Infof("Reset set_time_enabled to default: %v", defaultConfig.SetTimeEnabled)

	case "log_level":
		config.LogLevel = defaultConfig.LogLevel
		m.log.Infof("Reset log_level to default: %s", defaultConfig.LogLevel)

	case "debug_mode":
		config.DebugMode = defaultConfig.DebugMode
		m.log.Infof("Reset debug_mode to default: %v", defaultConfig.DebugMode)

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
