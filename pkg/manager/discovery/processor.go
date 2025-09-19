package discovery

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/google/uuid"
)

// Processor handles discovery processing operations
type Processor struct {
	db  *database.Database
	log *logrus.Logger
}

// NewProcessor creates a new processor
func NewProcessor(db *database.Database, log *logrus.Logger) *Processor {
	return &Processor{
		db:  db,
		log: log,
	}
}

// ProcessDiscoveredDevices handles the discovered devices from a BLE scan
func (p *Processor) ProcessDiscoveredDevices(devices []ble.Device, notifier common.UpdateNotifier) {
	p.log.Tracef("Processing %d discovered devices", len(devices))

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
			p.log.Trace("Notifying observers about device updates")
			notifier()
		}
	}
}

// ProcessDiscoveredDeviceLive handles a single discovered device immediately (for live updates)
func (p *Processor) ProcessDiscoveredDeviceLive(device ble.Device, notifier common.UpdateNotifier) {
	p.log.Tracef("Processing live discovered device: %s (%s) RSSI:%d", device.Name, device.MACAddress, device.RSSI)

	// Use a mutex to protect database operations
	var dbMutex sync.Mutex
	// Track if any changes were made that require a notification
	var changesMade atomic.Bool

	// Process the device
	p.processDevice(device, &dbMutex, &changesMade, notifier)

	// For live discovery, we want to be more aggressive about sending notifications
	// to ensure the UI stays responsive and shows real-time RSSI updates
	if notifier != nil {
		p.log.Tracef("Notifying observers about live device update for %s (RSSI: %d)", device.MACAddress, device.RSSI)
		notifier()
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
			ID:              uuid.New().String(),
			FirmwareVersion: "", // Will be populated when connected
			Model:           "", // Will be populated when connected
			SerialNumber:    "", // Will be populated when connected
			BatteryLevel:    0,  // Will be populated when connected
			HardwareVersion: "", // Will be populated when connected
		},
	}

	discoveredCamera := &database.DiscoveredCamera{
		CameraState: cameraState,
	}

	// Database will log the new camera discovery

	// Add to database
	dbMutex.Lock()
	err := p.db.AddOrUpdateDiscoveredCamera(discoveredCamera)
	dbMutex.Unlock()

	if err != nil {
		p.log.Errorf("Failed to add discovered camera: %v", err)
		return
	}

	// Mark changes made - notification will be sent by ProcessDiscoveredDevices()
	// This avoids duplicate notifications and batches updates efficiently
	changesMade.Store(true)
}

// updateExistingCamera updates an existing camera entry in the database
func (p *Processor) updateExistingCamera(discoveredCamera *database.DiscoveredCamera, name string, rssi int32, dbMutex *sync.Mutex, changesMade *atomic.Bool) {
	// Camera exists, update its basic properties and status
	cameraState := discoveredCamera.CameraState

	// Store previous values for change detection and logging
	previousLastSeen := cameraState.Status.LastSeen
	previousRSSI := cameraState.Camera.RSSI
	previousName := cameraState.Camera.Name

	// Update LastSeen timestamp - this is crucial for tracking when the camera was last discovered
	now := time.Now()
	timeSinceLastSeen := now.Sub(previousLastSeen)
	cameraState.Status.LastSeen = now

	// Track that changes were made (this will ensure the database update logic runs)
	changesMade.Store(true)

	// Update RSSI if it has changed - crucial for real-time signal strength updates
	rssiChanged := cameraState.Camera.RSSI != rssi
	if rssiChanged {
		cameraState.Camera.RSSI = rssi
		p.log.Tracef("RSSI updated for %s (%s): %d -> %d (last seen %v ago)",
			cameraState.Camera.Name, cameraState.Camera.MACAddress, previousRSSI, rssi, timeSinceLastSeen.Truncate(time.Millisecond))
	}

	// Ensure name is set if it was empty before or update if it's different
	nameChanged := false
	if cameraState.Camera.Name == "" || (!strings.Contains(cameraState.Camera.Name, "GoPro") && strings.Contains(name, "GoPro")) {
		p.log.Debugf("Name updated for %s: '%s' -> '%s'",
			cameraState.Camera.MACAddress, previousName, name)
		cameraState.Camera.Name = name
		nameChanged = true
	}

	// Ensure device is marked as reachable (critical for live discovery)
	reachabilityChanged := false
	if !cameraState.Status.IsReachable {
		p.log.Infof("Marking camera %s as reachable again (was unreachable)",
			cameraState.Camera.Name)
		cameraState.Status.IsReachable = true
		reachabilityChanged = true
	}

	// Log the discovery with appropriate detail level based on what changed
	if rssiChanged || nameChanged || reachabilityChanged || timeSinceLastSeen > 5*time.Second {
		changes := []string{}
		if rssiChanged {
			changes = append(changes, fmt.Sprintf("RSSI:%d->%d", previousRSSI, rssi))
		}
		if nameChanged {
			changes = append(changes, fmt.Sprintf("Name:'%s'->'%s'", previousName, name))
		}
		if reachabilityChanged {
			changes = append(changes, "IsReachable:false->true")
		}
		if timeSinceLastSeen > 5*time.Second {
			changes = append(changes, fmt.Sprintf("LastSeen:%v ago", timeSinceLastSeen.Truncate(time.Millisecond)))
		}

		p.log.Tracef("Rediscovered camera %s (%s) with changes: %s",
			cameraState.Camera.Name, cameraState.Camera.MACAddress, strings.Join(changes, ", "))
	} else {
		p.log.Tracef("Rediscovered camera %s (%s) RSSI:%d (no significant changes)",
			cameraState.Camera.Name, cameraState.Camera.MACAddress, rssi)
	}

	// Save to database - the database layer will handle change counter increments
	// based on the new enhanced logic for RSSI and last_seen changes
	dbMutex.Lock()
	err := p.db.AddOrUpdateDiscoveredCamera(discoveredCamera)
	dbMutex.Unlock()

	if err != nil {
		p.log.Errorf("Failed to update discovered camera: %v", err)
		return
	}
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

// MarkUnreachableDevicesBackground marks cameras as unreachable if they haven't been seen recently
// This is designed to be called from background tasks to clean up stale devices
func (p *Processor) MarkUnreachableDevicesBackground(notifier common.UpdateNotifier) {
	var dbMutex sync.Mutex
	var changesMade atomic.Bool

	// Use a longer threshold for live discovery to avoid race conditions
	// Since live updates happen immediately, we give more time before marking as unreachable
	unreachableThreshold := time.Now().Add(-45 * time.Second)
	devicesChecked := 0
	devicesMarkedUnreachable := 0

	dbMutex.Lock()
	for _, cameraState := range p.db.CameraStates {
		devicesChecked++

		// If the camera was last seen more than the threshold and is currently marked as reachable,
		// mark it as unreachable
		if cameraState.Status.LastSeen.Before(unreachableThreshold) && cameraState.Status.IsReachable {
			timeSinceLastSeen := time.Since(cameraState.Status.LastSeen)
			p.log.Infof("Background cleanup: marking camera %s (%s) as unreachable (last seen %v ago)",
				cameraState.Camera.Name,
				cameraState.Camera.MACAddress,
				timeSinceLastSeen.Truncate(time.Second))

			cameraState.Status.IsReachable = false
			changesMade.Store(true)
			devicesMarkedUnreachable++
		}
	}

	// Save changes if any were made
	if changesMade.Load() {
		err := p.db.SaveChanges()
		if err != nil {
			p.log.Errorf("Failed to save camera state changes during background cleanup: %v", err)
		} else {
			p.log.Debugf("Background cleanup completed: checked %d devices, marked %d as unreachable",
				devicesChecked, devicesMarkedUnreachable)
		}
	}
	dbMutex.Unlock()

	// Notify observers if changes were made
	if changesMade.Load() && notifier != nil {
		notifier()
	}
}
