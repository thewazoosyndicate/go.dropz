package pairing

import (
	"context"
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
)

// Manager handles camera pairing operations
type Manager struct {
	db       *database.Database
	ble      *ble.Manager
	log      *logrus.Logger
	notifier common.UpdateNotifier
	ctx      context.Context
}

// NewManager creates a new pairing manager
func NewManager(db *database.Database, ble *ble.Manager, log *logrus.Logger, notifier common.UpdateNotifier, ctx context.Context) *Manager {
	return &Manager{
		db:       db,
		ble:      ble,
		log:      log,
		notifier: notifier,
		ctx:      ctx,
	}
}

// PairCamera pairs with a GoPro camera using BLE
func (pm *Manager) PairCamera(cameraID string, bleOperation func(context.Context, string, func() error) error) (*database.ManagedCamera, error) {
	// First check if we can find the camera
	var cameraState *database.CameraWithState

	// Check all cameras
	for _, state := range pm.db.CameraStates {
		if state.Camera.ID == cameraID {
			cameraState = state
			break
		}
	}

	if cameraState == nil {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	macAddress := cameraState.Camera.MACAddress
	pm.log.Infof("Starting pairing process for camera %s", cameraState.Camera.Name)

	// Mark camera as pairing
	pm.db.UpdateCameraPairingStatus(macAddress, true)

	// Notify about the status change
	if pm.notifier != nil {
		pm.notifier()
	}

	// Create a context with a reasonable timeout for pairing operation
	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	// Variable to store verification result from inside BLE operation
	var verifiedPairingState bool

	// Perform the BLE pairing operation
	bleErr := bleOperation(ctx, "PairCamera", func() error {
		// Use Connect which now does everything including metadata updates
		pm.log.Infof("Connecting to device %s for pairing", macAddress)

		if err := pm.ble.Connect(macAddress); err != nil {
			return fmt.Errorf("failed to connect: %v", err)
		}

		// Connect() has already:
		// - Enabled WiFi AP
		// - Retrieved WiFi credentials
		// - Fetched hardware info and battery
		// - Invoked metadata callback to update database
		
		// Now we just need to verify pairing was successful by checking WiFi credentials
		// Refresh from database since callback updated it
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

		// Update database with pairing status
		if err := pm.db.SetCameraPaired(macAddress, isPaired); err != nil {
			pm.log.Errorf("Failed to set camera paired status: %v", err)
		} else {
			pm.log.Infof("Camera %s pairing status updated in database: %v", updatedState.Camera.Name, isPaired)
		}

		// Notify about pairing status update
		if pm.notifier != nil {
			pm.notifier()
		}

		// Store the verified pairing state for use after the BLE operation
		verifiedPairingState = isPaired

		// Now disconnect from the camera
		if err := pm.ble.Disconnect(macAddress); err != nil {
			pm.log.Warnf("Failed to disconnect from camera: %v", err)
			// This is not critical, we can continue
		}

		pm.log.Infof("Pairing operation completed for camera %s", updatedState.Camera.Name)
		return nil
	})

	// Handle any BLE operation errors
	if bleErr != nil {
		pm.log.Errorf("BLE pairing operation failed: %v", bleErr)
		// Make sure pairing flag is reset if BLE operation failed
		pm.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)
		return nil, fmt.Errorf("failed in BLE pairing operation: %v", bleErr)
	}

	// Complete the pairing operation with the verified state
	return pm.completePairingOperation(cameraState, macAddress, verifiedPairingState)
}

// completePairingOperation completes the pairing operation and updates database
func (pm *Manager) completePairingOperation(cameraState *database.CameraWithState, macAddress string, isPaired bool) (*database.ManagedCamera, error) {
	// Reset the pairing flag since pairing process is complete
	pm.db.UpdateCameraPairingStatus(macAddress, false)

	// Log the pairing completion status
	pm.log.Infof("Camera %s pairing completed. isPaired=%v, WiFi SSID=%s",
		cameraState.Camera.Name, isPaired, cameraState.Camera.WiFiSSID)

	// Final notification about the status change
	if pm.notifier != nil {
		pm.notifier()
	}

	// Return the managed camera view
	managedCamera, exists := pm.db.GetManagedCamera(macAddress)
	if !exists {
		pm.log.Debugf("Camera %s is paired but not yet managed, creating managed view", cameraState.Camera.Name)
		managedCamera = &database.ManagedCamera{
			CameraState: cameraState,
		}
	}

	return managedCamera, nil
}

