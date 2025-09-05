package pairing

import (
	"context"
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/logger"
)

// Manager handles camera pairing operations
type Manager struct {
	db       *database.Database
	ble      ble.BLEInterface
	log      logger.Logger
	notifier common.UpdateNotifier
	ctx      context.Context
}

// NewManager creates a new pairing manager
func NewManager(db *database.Database, ble ble.BLEInterface, log logger.Logger, notifier common.UpdateNotifier, ctx context.Context) *Manager {
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
		pm.notifier.NotifyUpdate()
	}

	// Create a context with a reasonable timeout for pairing operation
	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	// Variable to store verification result from inside BLE operation
	var verifiedPairingState bool

	// Perform the BLE pairing operation
	bleErr := bleOperation(ctx, "PairCamera", func() error {
		// Use ConnectWithEnhancedPairing which now follows OpenGoPro spec
		pm.log.Infof("Connecting to device %s with enhanced pairing", macAddress)

		if err := pm.ble.ConnectWithEnhancedPairing(macAddress); err != nil {
			return fmt.Errorf("failed to connect with enhanced pairing: %v", err)
		}

		// WiFi credentials are now obtained during enhanced pairing
		// The enhanced pairing should have retrieved them
		ssid, password, err := pm.ble.GetWifiCredentials(macAddress)
		if err == nil && ssid != "" && password != "" {
			cameraState.Camera.WiFiSSID = ssid
			cameraState.Camera.WiFiPassword = password
			pm.log.Infof("Retrieved WiFi credentials: SSID=%s", ssid)
		}

		// Notify about WiFi credentials obtained
		if pm.notifier != nil {
			pm.notifier.NotifyUpdate()
		}

		// Fetch hardware metadata and store in database
		pm.log.Infof("Fetching hardware metadata for camera %s", cameraState.Camera.Name)
		hwMeta, err := pm.ble.GetMetadata(macAddress)
		if err != nil {
			pm.log.Warnf("Failed to fetch hardware metadata: %v", err)
		} else {
			// Also fetch battery level
			var batteryLevel int32
			if bl, err2 := pm.ble.GetBatteryLevel(macAddress); err2 != nil {
				pm.log.Warnf("Failed to fetch battery level for camera %s: %v", cameraState.Camera.Name, err2)
			} else {
				batteryLevel = int32(bl)
			}
			meta := database.CameraMetadata{
				ID:              cameraState.Camera.ID,
				Model:           hwMeta["model_name"],
				FirmwareVersion: hwMeta["firmware_version"],
				SerialNumber:    hwMeta["serial_number"],
				BatteryLevel:    batteryLevel,
			}
			if err := pm.db.SetCameraMetadata(macAddress, meta); err != nil {
				pm.log.Errorf("Failed to save camera metadata: %v", err)
			} else if pm.notifier != nil {
				pm.notifier.NotifyUpdate()
			}
		}

		// Simple pairing verification - successful WiFi credential retrieval means paired
		isPaired := cameraState.Camera.WiFiSSID != "" && cameraState.Camera.WiFiPassword != ""
		
		if isPaired {
			pm.log.Infof("Pairing successful - WiFi credentials obtained: SSID=%s", cameraState.Camera.WiFiSSID)
		} else {
			pm.log.Warnf("Pairing may be incomplete - no WiFi credentials obtained")
		}

		// Update database with pairing status
		if err := pm.db.SetCameraPaired(macAddress, isPaired); err != nil {
			pm.log.Errorf("Failed to set camera paired status: %v", err)
		} else {
			pm.log.Infof("Camera %s pairing status updated in database: %v", cameraState.Camera.Name, isPaired)
		}

		// Notify about pairing status update
		if pm.notifier != nil {
			pm.notifier.NotifyUpdate()
		}

		// Store the verified pairing state for use after the BLE operation
		verifiedPairingState = isPaired

		// Now disconnect from the camera
		if err := pm.ble.Disconnect(macAddress); err != nil {
			pm.log.Warnf("Failed to disconnect from camera: %v", err)
			// This is not critical, we can continue
		}

		pm.log.Infof("Pairing operation completed for camera %s", cameraState.Camera.Name)
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
		pm.notifier.NotifyUpdate()
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

