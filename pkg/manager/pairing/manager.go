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
	cameraState, found := pm.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	macAddress := cameraState.Camera.MACAddress
	pm.log.Infof("Starting pairing process for camera %s", cameraState.Camera.Name)

	pm.db.UpdateCameraPairingStatus(macAddress, true)
	notify(notifier)

	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	var verifiedPairingState bool

	bleErr := bleOperation(ctx, "PairCamera", func() error {
		pm.log.Infof("Connecting to device %s for pairing", macAddress)

		if err := pm.ble.Connect(macAddress); err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}

		updatedState, exists := pm.db.CameraStates[macAddress]
		if !exists {
			return fmt.Errorf("camera state not found after connection")
		}

		isPaired := updatedState.Camera.WiFiSSID != "" && updatedState.Camera.WiFiPassword != ""

		if isPaired {
			pm.log.Infof("Pairing successful - WiFi credentials obtained: SSID=%s", updatedState.Camera.WiFiSSID)
		} else {
			pm.log.Warnf("Pairing may be incomplete - no WiFi credentials obtained")
		}

		if err := pm.db.SetCameraPaired(macAddress, isPaired); err != nil {
			pm.log.Errorf("Failed to set camera paired status: %v", err)
		}

		notify(notifier)
		verifiedPairingState = isPaired

		if err := pm.ble.Disconnect(macAddress); err != nil {
			pm.log.Warnf("Failed to disconnect from camera: %v", err)
		}

		return nil
	})

	if bleErr != nil {
		pm.log.Errorf("BLE pairing operation failed: %v", bleErr)
		pm.db.UpdateCameraPairingStatus(macAddress, false)
		return nil, fmt.Errorf("failed in BLE pairing operation: %v", bleErr)
	}

	// Complete pairing
	pm.db.UpdateCameraPairingStatus(macAddress, false)
	notify(notifier)

	managedCamera, exists := pm.db.GetManagedCamera(macAddress)
	if !exists {
		managedCamera = &database.ManagedCamera{CameraState: cameraState}
	}

	pm.log.Infof("Camera %s pairing completed. isPaired=%v", cameraState.Camera.Name, verifiedPairingState)
	return managedCamera, nil
}

func notify(notifier func()) {
	if notifier != nil {
		notifier()
	}
}
