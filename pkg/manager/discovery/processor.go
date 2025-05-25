package discovery

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/logger"
	"github.com/google/uuid"
)

// Processor handles discovery processing operations
type Processor struct {
	db  *database.Database
	log logger.Logger
}

// NewProcessor creates a new processor
func NewProcessor(db *database.Database, log logger.Logger) *Processor {
	return &Processor{
		db:  db,
		log: log,
	}
}

// ProcessDiscoveredDevices handles the discovered devices from a BLE scan
func (p *Processor) ProcessDiscoveredDevices(devices []ble.Device, notifier common.UpdateNotifier) {
	p.log.Debugf("Processing %d discovered devices", len(devices))

	// Use a wait group to process devices concurrently
	var wg sync.WaitGroup
	// Use a mutex to protect database operations
	var dbMutex sync.Mutex

	// Track if any changes were made that require a notification
	var changesMade atomic.Bool

	// First, mark all cameras as unreachable that haven't been seen recently
	p.markUnreachableDevices(&dbMutex, &changesMade)

	for _, device := range devices {
		wg.Add(1)

		// Process each device in a separate goroutine
		go func(dev ble.Device) {
			defer wg.Done()
			p.processDevice(dev, &dbMutex, &changesMade, notifier)
		}(device)
	}

	// Wait for all device processing to complete
	wg.Wait()

	// Notify about device updates if any changes were made
	// This ensures we don't send unnecessary notifications
	if changesMade.Load() {
		if notifier != nil {
			p.log.Debug("Notifying observers about device updates")
			notifier.NotifyUpdate()
		}
	}
}

// processDevice processes a single discovered device
func (p *Processor) processDevice(dev ble.Device, dbMutex *sync.Mutex, changesMade *atomic.Bool, notifier common.UpdateNotifier) {
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
	discoveredCamera, exists := p.db.GetDiscoveredCamera(macAddress)
	dbMutex.Unlock()

	// Create a new discovered camera if it doesn't exist
	if !exists {
		p.createNewCamera(name, macAddress, rssi, dbMutex, changesMade, notifier)
	} else {
		p.updateExistingCamera(discoveredCamera, name, rssi, dbMutex, changesMade)
	}
}

// createNewCamera creates a new camera entry in the database
func (p *Processor) createNewCamera(name, macAddress string, rssi int32, dbMutex *sync.Mutex, changesMade *atomic.Bool, notifier common.UpdateNotifier) {
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

	discoveredCamera := &database.DiscoveredCamera{
		CameraState: cameraState,
	}

	p.log.Infof("New camera discovered: %s (%s)", name, macAddress)

	// Add to database
	dbMutex.Lock()
	err := p.db.AddOrUpdateDiscoveredCamera(discoveredCamera)
	dbMutex.Unlock()

	if err != nil {
		p.log.Errorf("Failed to add discovered camera: %v", err)
		return
	}

	// Notify immediately for new device discovery
	if notifier != nil {
		p.log.Debug("Notifying observers about new device discovery")
		notifier.NotifyUpdate()
	}

	changesMade.Store(true)
}

// updateExistingCamera updates an existing camera entry in the database
func (p *Processor) updateExistingCamera(discoveredCamera *database.DiscoveredCamera, name string, rssi int32, dbMutex *sync.Mutex, changesMade *atomic.Bool) {
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
	p.db.UpdateCameraReachability(cameraState.Camera.MACAddress, true)
	dbMutex.Unlock()

	changesMade.Store(true)
}

// markUnreachableDevices marks cameras as unreachable if they haven't been seen recently
func (p *Processor) markUnreachableDevices(dbMutex *sync.Mutex, changesMade *atomic.Bool) {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	unreachableThreshold := time.Now().Add(-30 * time.Second)

	// Get all cameras
	for _, cameraState := range p.db.CameraStates {
		// If the camera was last seen more than 30 seconds ago and is currently marked as reachable,
		// mark it as unreachable
		if cameraState.Status.LastSeen.Before(unreachableThreshold) && cameraState.Status.IsReachable {
			p.log.Debugf("Marking camera %s as unreachable (last seen: %s)",
				cameraState.Camera.MACAddress, cameraState.Status.LastSeen.Format(time.RFC3339))

			cameraState.Status.IsReachable = false
			changesMade.Store(true)
		}
	}

	// Save changes if any were made
	if changesMade.Load() {
		err := p.db.SaveChanges()
		if err != nil {
			p.log.Errorf("Failed to save camera state changes: %v", err)
		}
	}
}

// GetDiscoveredDevicesCount returns the number of discovered devices
func (p *Processor) GetDiscoveredDevicesCount() int {
	return len(p.db.CameraStates)
}

// GetReachableDevicesCount returns the number of reachable devices
func (p *Processor) GetReachableDevicesCount() int {
	count := 0
	for _, cameraState := range p.db.CameraStates {
		if cameraState.Status.IsReachable {
			count++
		}
	}
	return count
}

// GetDeviceDiscoveryStats returns discovery statistics
func (p *Processor) GetDeviceDiscoveryStats() map[string]interface{} {
	total := len(p.db.CameraStates)
	reachable := p.GetReachableDevicesCount()
	paired := 0
	managed := 0

	for _, cameraState := range p.db.CameraStates {
		if cameraState.Status.IsPaired {
			paired++
		}
		if cameraState.Status.IsManaged {
			managed++
		}
	}

	return map[string]interface{}{
		"total":     total,
		"reachable": reachable,
		"paired":    paired,
		"managed":   managed,
	}
}
