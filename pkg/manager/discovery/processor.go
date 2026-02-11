package discovery

import (
	"strings"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	macAddress := dev.MACAddress
	rssi := dev.RSSI

	if name == "" || !strings.Contains(strings.ToLower(name), "gopro") {
		return
	}

	discoveredCamera, exists := p.db.GetDiscoveredCamera(macAddress)

	if !exists {
		p.createNewCamera(name, macAddress, rssi)
	} else {
		p.updateExistingCamera(discoveredCamera, name, rssi)
	}
}

// createNewCamera creates a new camera entry in the database
func (p *Processor) createNewCamera(name, macAddress string, rssi int32) {
	cameraState := &database.CameraWithState{
		Camera: database.Camera{
			ID:         uuid.New().String(),
			Name:       name,
			MACAddress: macAddress,
			RSSI:       rssi,
		},
		Status: database.CameraStatus{
			LastSeen:    time.Now(),
			IsReachable: true,
		},
		Metadata: database.CameraMetadata{
			ID: uuid.New().String(),
		},
	}

	discoveredCamera := &database.DiscoveredCamera{
		CameraState: cameraState,
	}

	if err := p.db.AddOrUpdateDiscoveredCamera(discoveredCamera); err != nil {
		p.log.Errorf("Failed to add discovered camera: %v", err)
	}
}

// updateExistingCamera atomically updates an existing camera entry in the database.
func (p *Processor) updateExistingCamera(discoveredCamera *database.DiscoveredCamera, name string, rssi int32) {
	macAddress := discoveredCamera.CameraState.Camera.MACAddress

	err := p.db.UpdateCamera(macAddress, func(cs *database.CameraWithState) {
		if cs.Camera.Name == "" || (!strings.Contains(cs.Camera.Name, "GoPro") && strings.Contains(name, "GoPro")) {
			p.log.Debugf("Name updated for %s: '%s' -> '%s'", macAddress, cs.Camera.Name, name)
			cs.Camera.Name = name
		}

		if !cs.Status.IsReachable {
			p.log.Infof("Marking camera %s as reachable again (was unreachable)", cs.Camera.Name)
			cs.Status.IsReachable = true
		}

		cs.Camera.RSSI = rssi
		cs.Status.LastSeen = time.Now()
	})
	if err != nil {
		p.log.Errorf("Failed to update discovered camera: %v", err)
	}
}

// MarkUnreachableDevicesBackground marks cameras as unreachable from background tasks.
func (p *Processor) MarkUnreachableDevicesBackground(notifier func()) {
	cutoff := time.Now().Add(-45 * time.Second)
	changed, err := p.db.MarkCamerasUnreachableBefore(cutoff)
	if err != nil {
		p.log.Errorf("Failed to save camera state changes: %v", err)
	}
	if changed && notifier != nil {
		notifier()
	}
}
