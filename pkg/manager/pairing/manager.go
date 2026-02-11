package pairing

import (
	"context"
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	db  *database.Database
	ble *ble.Manager
	log *logrus.Logger
	ctx context.Context
}

func NewManager(db *database.Database, ble *ble.Manager, log *logrus.Logger, ctx context.Context) *Manager {
	return &Manager{db: db, ble: ble, log: log, ctx: ctx}
}

// PairCamera pairs with a GoPro camera using BLE
func (pm *Manager) PairCamera(cameraID string, bleOperation func(context.Context, string, func() error) error, notifier func()) (*database.ManagedCamera, error) {
	macAddress, name, found := pm.db.GetCameraIdentifiers(cameraID)
	if !found {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	pm.log.Infof("Starting pairing process for camera %s", name)

	pm.db.UpdateCameraPairingStatus(macAddress, true)
	notifier()

	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	var verifiedPairingState bool

	bleErr := bleOperation(ctx, "PairCamera", func() error {
		pm.log.Infof("Connecting to device %s for pairing", macAddress)

		if err := pm.ble.ConnectForPairing(macAddress); err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}

		// Verify pairing succeeded by checking if WiFi credentials were read.
		// ConnectForPairing reads them after D-Bus bonding and the metadata
		// callback persists them to the database.
		cam, exists := pm.db.GetDiscoveredCamera(macAddress)
		isPaired := exists && cam.CameraState != nil &&
			cam.CameraState.Camera.WiFiSSID != "" && cam.CameraState.Camera.WiFiPassword != ""

		if isPaired {
			pm.log.Infof("Pairing verified — WiFi credentials received")
		} else {
			pm.log.Warnf("Pairing completed but WiFi credentials not available yet")
		}

		if err := pm.db.SetCameraPaired(macAddress, true); err != nil {
			pm.log.Errorf("Failed to set camera paired status: %v", err)
		}

		notifier()
		verifiedPairingState = isPaired

		if err := pm.ble.Disconnect(macAddress); err != nil {
			pm.log.Warnf("Failed to disconnect from camera: %v", err)
		}

		return nil
	})

	if bleErr != nil {
		pm.log.Errorf("BLE pairing operation failed: %v", bleErr)
		pm.db.UpdateCameraPairingStatus(macAddress, false)
		pm.db.SetCameraPaired(macAddress, false)
		notifier()
		return nil, fmt.Errorf("failed in BLE pairing operation: %v", bleErr)
	}

	// Complete pairing
	pm.db.UpdateCameraPairingStatus(macAddress, false)
	notifier()

	managedCamera, exists := pm.db.GetManagedCamera(macAddress)
	if !exists {
		cs, _ := pm.db.GetDiscoveredCamera(macAddress)
		if cs != nil {
			managedCamera = &database.ManagedCamera{CameraState: cs.CameraState}
		}
	}

	pm.log.Infof("Camera %s pairing completed. isPaired=%v", name, verifiedPairingState)
	return managedCamera, nil
}
