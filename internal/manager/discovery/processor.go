package discovery

import (
	"strings"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/store"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor handles discovery processing operations
type Processor struct {
	db                 *store.Store
	log                *logrus.Logger
	onCameraReappeared func(cameraID string, wasGoneFor time.Duration)
}

// NewProcessor creates a new processor
func NewProcessor(db *store.Store, log *logrus.Logger) *Processor {
	return &Processor{
		db:  db,
		log: log,
	}
}

// SetOnCameraReappeared sets a callback invoked when a camera becomes reachable again
func (p *Processor) SetOnCameraReappeared(fn func(cameraID string, wasGoneFor time.Duration)) {
	p.onCameraReappeared = fn
}

// ProcessDiscoveredDeviceLive handles a single discovered device immediately (for live updates)
func (p *Processor) ProcessDiscoveredDeviceLive(device ble.Device, notifier func()) {
	p.processDevice(device)
	if notifier != nil {
		notifier()
	}
}

// processDevice processes a single discovered device
func (p *Processor) processDevice(dev ble.Device) {
	name := dev.Name
	bleAddress := dev.BLEAddress
	rssi := dev.RSSI

	if name == "" || !strings.Contains(strings.ToLower(name), "gopro") {
		return
	}

	// Look up by BLE address — finds the camera even if re-keyed by serial
	lookup, exists := p.db.FindCameraByBLEAddress(bleAddress)

	if !exists {
		p.createNewCamera(name, bleAddress, rssi)
	} else {
		p.updateExistingCamera(lookup.DBKey, bleAddress, name, rssi)
	}
}

// createNewCamera creates a new camera entry in the database
func (p *Processor) createNewCamera(name, bleAddress string, rssi int32) {
	cameraState := &model.CameraWithState{
		Camera: model.Camera{
			ID:         uuid.New().String(),
			Name:       name,
			BLEAddress: bleAddress,
			RSSI:       rssi,
		},
		Status: model.CameraStatus{
			LastSeen:    time.Now(),
			IsReachable: true,
		},
		Metadata: model.CameraMetadata{
			ID: uuid.New().String(),
		},
	}

	discoveredCamera := &model.DiscoveredCamera{
		CameraState: cameraState,
	}

	if err := p.db.AddOrUpdateDiscoveredCamera(discoveredCamera); err != nil {
		p.log.Errorf("Failed to add discovered camera: %v", err)
	}
}

// updateExistingCamera atomically updates an existing camera entry in the model.
func (p *Processor) updateExistingCamera(dbKey, bleAddress, name string, rssi int32) {
	var wasGoneFor time.Duration
	var cameraID string
	wasUnreachable := false

	// Only persist when identity fields change; RSSI, LastSeen, and
	// reachability are ephemeral and would otherwise rewrite the DB file
	// on every advertisement.
	err := p.db.UpdateCamera(dbKey, func(cs *model.CameraWithState) bool {
		cameraID = cs.Camera.ID
		durable := false

		// Update BLE address in case it changed (macOS assigns random UUIDs)
		if cs.Camera.BLEAddress != bleAddress {
			cs.Camera.BLEAddress = bleAddress
			durable = true
		}

		if cs.Camera.Name == "" || (!strings.Contains(cs.Camera.Name, "GoPro") && strings.Contains(name, "GoPro")) {
			p.log.Debugf("Name updated for %s: '%s' to '%s'", dbKey, cs.Camera.Name, name)
			cs.Camera.Name = name
			durable = true
		}

		if !cs.Status.IsReachable {
			wasGoneFor = time.Since(cs.Status.LastSeen)
			wasUnreachable = true
			p.log.Infof("Marking camera %s as reachable again (was gone for %v)", cs.Camera.Name, wasGoneFor)
			cs.Status.IsReachable = true
		}

		cs.Camera.RSSI = rssi
		cs.Status.LastSeen = time.Now()
		return durable
	})
	if err != nil {
		p.log.Errorf("Failed to update discovered camera: %v", err)
		return
	}

	if wasUnreachable && p.onCameraReappeared != nil {
		p.onCameraReappeared(cameraID, wasGoneFor)
	}
}

// MarkUnreachableDevicesBackground marks cameras as unreachable from background tasks.
func (p *Processor) MarkUnreachableDevicesBackground(inactivityTimeout time.Duration, notifier func()) {
	cutoff := time.Now().Add(-inactivityTimeout)
	changed, err := p.db.MarkCamerasUnreachableBefore(cutoff)
	if err != nil {
		p.log.Errorf("Failed to save camera state changes: %v", err)
	}
	if changed && notifier != nil {
		notifier()
	}
}
