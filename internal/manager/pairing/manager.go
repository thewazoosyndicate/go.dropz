package pairing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/store"
)

const pairingTimeout = 30 * time.Second

// BLEOperation runs a BLE operation with the manager's retry policy.
type BLEOperation func(ctx context.Context, critical bool, op func() error) error

type Manager struct {
	db           *store.Store
	ble          *ble.Manager
	log          *slog.Logger
	ctx          context.Context
	bleOperation BLEOperation
	notifier     func()
}

func NewManager(ctx context.Context, db *store.Store, ble *ble.Manager, log *slog.Logger, bleOperation BLEOperation, notifier func()) *Manager {
	return &Manager{
		db:           db,
		ble:          ble,
		log:          log.With("component", "pairing"),
		ctx:          ctx,
		bleOperation: bleOperation,
		notifier:     notifier,
	}
}

// PairCamera pairs with a GoPro camera using BLE
func (pm *Manager) PairCamera(cameraID string) (*model.ManagedCamera, error) {
	cs, found := pm.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}
	bleAddress := cs.Camera.BLEAddress
	name := cs.Camera.Name

	pm.log.Info("Starting pairing process", "camera", name)

	pm.db.UpdateCameraPairingStatusByID(cameraID, true)
	pm.notifier()

	ctx, cancel := context.WithTimeout(pm.ctx, pairingTimeout)
	defer cancel()

	var verifiedPairingState bool

	bleErr := pm.bleOperation(ctx, true, func() error {
		pm.log.Info("Connecting for pairing", "device", bleAddress)

		if err := pm.ble.ConnectForPairing(bleAddress); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		// Verify pairing succeeded by checking if WiFi credentials were read.
		// ConnectForPairing reads them after D-Bus bonding and the metadata
		// callback persists them to the database.
		cam, exists := pm.db.GetCameraByID(cameraID)
		isPaired := exists && cam.Camera.WiFiSSID != "" && cam.Camera.WiFiPassword != ""

		if isPaired {
			pm.log.Info("Pairing verified: WiFi credentials received")
		} else {
			pm.log.Warn("Pairing completed but WiFi credentials not available yet")
		}

		// Only record paired when credentials were actually read; a bond
		// without credentials can't sync and should be retried by the user.
		pm.db.SetCameraPairedByID(cameraID, isPaired)
		pm.notifier()
		verifiedPairingState = isPaired

		if err := pm.ble.DisconnectQuietly(bleAddress); err != nil {
			pm.log.Warn("Failed to disconnect from camera", "err", err)
		}

		return nil
	})

	if bleErr != nil {
		pm.log.Error("BLE pairing operation failed", "camera", name, "err", bleErr)
		pm.db.UpdateCameraPairingStatusByID(cameraID, false)
		pm.db.SetCameraPairedByID(cameraID, false)
		pm.notifier()
		return nil, fmt.Errorf("failed in BLE pairing operation: %w", bleErr)
	}

	pm.db.UpdateCameraPairingStatusByID(cameraID, false)
	pm.notifier()

	managedCamera, _ := pm.db.GetManagedCameraByID(cameraID)
	if managedCamera == nil {
		if cs, ok := pm.db.GetCameraByID(cameraID); ok {
			managedCamera = &model.ManagedCamera{CameraState: cs}
		}
	}

	pm.log.Info("Pairing completed", "camera", name, "paired", verifiedPairingState)
	return managedCamera, nil
}
