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
func (pm *Manager) PairCamera(cameraID string, bleOperation func(context.Context, bool, func() error) error, notifier func()) (*database.ManagedCamera, error) {
	cs, found := pm.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}
	bleAddress := cs.Camera.BLEAddress
	name := cs.Camera.Name

	pm.log.Infof("Starting pairing process for camera %s", name)

	pm.db.UpdateCameraPairingStatusByID(cameraID, true)
	notifier()

	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	var verifiedPairingState bool

	bleErr := bleOperation(ctx, true, func() error {
		pm.log.Infof("Connecting to device %s for pairing", bleAddress)

		if err := pm.ble.ConnectForPairing(bleAddress); err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}

		// Verify pairing succeeded by checking if WiFi credentials were read.
		// ConnectForPairing reads them after D-Bus bonding and the metadata
		// callback persists them to the database.
		cam, exists := pm.db.GetCameraByID(cameraID)
		isPaired := exists && cam.Camera.WiFiSSID != "" && cam.Camera.WiFiPassword != ""

		if isPaired {
			pm.log.Infof("Pairing verified — WiFi credentials received")
		} else {
			pm.log.Warnf("Pairing completed but WiFi credentials not available yet")
		}

		pm.db.SetCameraPairedByID(cameraID, true)
		notifier()
		verifiedPairingState = isPaired

		if err := pm.ble.DisconnectQuietly(bleAddress); err != nil {
			pm.log.Warnf("Failed to disconnect from camera: %v", err)
		}

		return nil
	})

	if bleErr != nil {
		pm.log.Errorf("BLE pairing operation failed: %v", bleErr)
		pm.db.UpdateCameraPairingStatusByID(cameraID, false)
		pm.db.SetCameraPairedByID(cameraID, false)
		notifier()
		return nil, fmt.Errorf("failed in BLE pairing operation: %v", bleErr)
	}

	pm.db.UpdateCameraPairingStatusByID(cameraID, false)
	notifier()

	managedCamera, _ := pm.db.GetManagedCameraByID(cameraID)
	if managedCamera == nil {
		if cs, ok := pm.db.GetCameraByID(cameraID); ok {
			managedCamera = &database.ManagedCamera{CameraState: cs}
		}
	}

	pm.log.Infof("Camera %s pairing completed. isPaired=%v", name, verifiedPairingState)
	return managedCamera, nil
}
