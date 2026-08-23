package discovery

import (
	"log/slog"
	"strings"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/store"
	"github.com/google/uuid"
)

// Processor handles discovery processing operations.
// Only the scan callback goroutine calls processDevice, so lastNewMedia
// needs no lock.
type Processor struct {
	db                    *store.Store
	log                   *slog.Logger
	onCameraReappeared    func(cameraID string, wasGoneFor time.Duration)
	onPairingModeDetected func(cameraID string)
	onNewMediaAdvertised  func(cameraID string)
	lastNewMedia          map[string]bool // rising-edge detection per camera ID
}

// NewProcessor creates a new processor
func NewProcessor(db *store.Store, log *slog.Logger) *Processor {
	return &Processor{
		db:           db,
		log:          log.With("component", "discovery"),
		lastNewMedia: make(map[string]bool),
	}
}

// SetOnCameraReappeared sets a callback invoked when a camera becomes reachable again
func (p *Processor) SetOnCameraReappeared(fn func(cameraID string, wasGoneFor time.Duration)) {
	p.onCameraReappeared = fn
}

// SetOnPairingModeDetected sets a callback invoked when a camera starts
// advertising its pairing UI (transition only, not on every advertisement).
func (p *Processor) SetOnPairingModeDetected(fn func(cameraID string)) {
	p.onPairingModeDetected = fn
}

// SetOnNewMediaAdvertised sets a callback invoked when a camera starts
// advertising unsynced media (transition only).
func (p *Processor) SetOnNewMediaAdvertised(fn func(cameraID string)) {
	p.onNewMediaAdvertised = fn
}

// ProcessDiscoveredDeviceLive handles a single discovered device immediately (for live updates)
func (p *Processor) ProcessDiscoveredDeviceLive(device ble.Device, notifier func()) {
	p.processDevice(device)
	if notifier != nil {
		notifier()
	}
}

// processDevice processes a single discovered device.
// Identity resolution order: serial number (stable across macOS's random
// BLE addresses), then BLE address, then exact name (older cameras whose
// advertisements carry no serial).
func (p *Processor) processDevice(dev ble.Device) {
	if dev.Name == "" || !strings.Contains(strings.ToLower(dev.Name), "gopro") {
		return
	}

	lookup, exists := p.db.FindCameraBySerial(dev.SerialNumber)
	if !exists {
		lookup, exists = p.db.FindCameraByBLEAddress(dev.BLEAddress)
	}
	if !exists {
		lookup, exists = p.db.FindCameraByName(dev.Name)
	}

	if !exists {
		p.createNewCamera(dev)
		return
	}
	p.updateExistingCamera(lookup.DBKey, dev)
}

// createNewCamera creates a new camera entry in the database
func (p *Processor) createNewCamera(dev ble.Device) {
	cameraState := &model.CameraWithState{
		Camera: model.Camera{
			ID:         uuid.New().String(),
			Name:       dev.Name,
			BLEAddress: dev.BLEAddress,
			RSSI:       dev.RSSI,
		},
		Status: model.CameraStatus{
			LastSeen:      time.Now(),
			IsReachable:   true,
			InPairingMode: dev.PairingMode,
		},
		Metadata: model.CameraMetadata{
			ID:           uuid.New().String(),
			SerialNumber: dev.SerialNumber,
			ModelID:      dev.ModelID,
		},
	}

	discoveredCamera := &model.DiscoveredCamera{
		CameraState: cameraState,
	}

	if err := p.db.AddOrUpdateDiscoveredCamera(discoveredCamera); err != nil {
		p.log.Error("Failed to add discovered camera", "camera", dev.Name, "ble_addr", dev.BLEAddress, "err", err)
		return
	}
	p.log.Info("Camera discovered", "camera_id", cameraState.Camera.ID, "camera", dev.Name,
		"ble_addr", dev.BLEAddress, "serial", dev.SerialNumber)
}

// updateExistingCamera atomically updates an existing camera entry.
func (p *Processor) updateExistingCamera(dbKey string, dev ble.Device) {
	var wasGoneFor time.Duration
	var cameraID, cameraName string
	wasUnreachable := false
	enteredPairingMode := false
	newMediaAppeared := false

	// Only persist when identity fields change; RSSI, LastSeen,
	// reachability, and the advertisement flags are ephemeral and would
	// otherwise rewrite the DB file on every advertisement.
	err := p.db.UpdateCamera(dbKey, func(cs *model.CameraWithState) bool {
		cameraID = cs.Camera.ID
		durable := false

		// Update BLE address in case it changed (macOS assigns random UUIDs)
		if cs.Camera.BLEAddress != dev.BLEAddress {
			// Routine on macOS, which rotates identifiers
			p.log.Debug("Camera BLE address changed", "camera", cs.Camera.Name,
				"old", cs.Camera.BLEAddress, "new", dev.BLEAddress)
			cs.Camera.BLEAddress = dev.BLEAddress
			durable = true
		}

		if cs.Camera.Name == "" || (!strings.Contains(cs.Camera.Name, "GoPro") && strings.Contains(dev.Name, "GoPro")) {
			p.log.Debug("Camera name updated", "camera_id", cs.Camera.ID, "old", cs.Camera.Name, "new", dev.Name)
			cs.Camera.Name = dev.Name
			durable = true
		}

		// Identity and model info learned from advertising data
		if dev.SerialNumber != "" && cs.Metadata.SerialNumber == "" {
			cs.Metadata.SerialNumber = dev.SerialNumber
			durable = true
		}
		if dev.ModelID > 0 && cs.Metadata.ModelID == 0 {
			cs.Metadata.ModelID = dev.ModelID
			durable = true
		}

		if !cs.Status.IsReachable {
			wasGoneFor = time.Since(cs.Status.LastSeen)
			wasUnreachable = true
			p.log.Info("Camera reachable again", append(cs.LogAttrs(), "gone_for", wasGoneFor)...)
			cs.Status.IsReachable = true
		}

		// Transition detection: fire callbacks once per flag rise
		if dev.PairingMode && !cs.Status.InPairingMode {
			enteredPairingMode = true
		}
		cs.Status.InPairingMode = dev.PairingMode

		cs.Camera.RSSI = dev.RSSI
		cs.Status.LastSeen = time.Now()
		cameraName = cs.Camera.Name
		return durable
	})
	if err != nil {
		p.log.Error("Failed to update discovered camera", "camera", dev.Name, "ble_addr", dev.BLEAddress, "err", err)
		return
	}

	// Rising edge of the advertised new-media flag. The flag reflects the
	// camera's own idea of unsynced media, so it only triggers a status
	// check; the count comparison there decides whether to sync.
	if dev.NewMedia && !p.lastNewMedia[cameraID] {
		newMediaAppeared = true
	}
	p.lastNewMedia[cameraID] = dev.NewMedia

	if wasUnreachable && p.onCameraReappeared != nil {
		p.onCameraReappeared(cameraID, wasGoneFor)
	}
	if enteredPairingMode {
		p.log.Info("Camera entered pairing mode", "camera_id", cameraID, "camera", cameraName)
		if p.onPairingModeDetected != nil {
			p.onPairingModeDetected(cameraID)
		}
	}
	if newMediaAppeared && p.onNewMediaAdvertised != nil {
		p.log.Info("Camera advertises new media", "camera_id", cameraID, "camera", cameraName)
		p.onNewMediaAdvertised(cameraID)
	}
}

// MarkUnreachableDevicesBackground marks cameras as unreachable from background tasks.
func (p *Processor) MarkUnreachableDevicesBackground(inactivityTimeout time.Duration, notifier func()) {
	cutoff := time.Now().Add(-inactivityTimeout)
	changed := p.db.MarkCamerasUnreachableBefore(cutoff)
	for _, cam := range changed {
		p.log.Info("Camera unreachable", "camera_id", cam.ID, "camera", cam.Name, "timeout", inactivityTimeout)
	}
	if len(changed) > 0 && notifier != nil {
		notifier()
	}
}
