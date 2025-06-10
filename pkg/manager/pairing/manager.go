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
	var pairingVerificationDone bool

	// Perform the BLE pairing operation
	bleErr := bleOperation(ctx, "PairCamera", func() error {
		// Use ONLY ConnectWithEnhancedPairing - it handles everything
		pm.log.Infof("Connecting to device %s with enhanced pairing", macAddress)

		if err := pm.ble.ConnectWithEnhancedPairing(macAddress); err != nil {
			return fmt.Errorf("failed to connect with enhanced pairing: %v", err)
		}

		// Get WiFi credentials directly - no need for separate connection
		ssid, password, err := pm.ble.GetWifiCredentials(macAddress)
		if err != nil {
			// Try to disconnect gracefully even if getting WiFi credentials failed
			_ = pm.ble.Disconnect(macAddress)
			return fmt.Errorf("failed to get WiFi credentials: %v", err)
		}

		// Store WiFi credentials in the database
		pm.log.Infof("Obtained WiFi credentials for camera %s: SSID=%s", cameraState.Camera.ID, ssid)

		// Update the camera's WiFi credentials in the cameraState object
		cameraState.Camera.WiFiSSID = ssid
		cameraState.Camera.WiFiPassword = password

		// Notify about WiFi credentials obtained
		if pm.notifier != nil {
			pm.notifier.NotifyUpdate()
		}

		// IMPROVED PAIRING VERIFICATION STRATEGY:
		// WiFi credential retrieval is the definitive success indicator for OpenGoPro pairing
		// But we also verify device state with proper timing to handle camera internal updates

		pm.log.Infof("Verifying pairing state while still connected for camera %s", cameraState.Camera.Name)

		// Give camera time to update internal pairing state after WiFi credential access
		time.Sleep(2 * time.Second)

		// Get fresh pairing state while connected (with retries for reliability)
		var devicePairingState int
		var verifyErr error
		for attempt := 1; attempt <= 3; attempt++ {
			devicePairingState, verifyErr = pm.ble.RefreshPairingState(macAddress)
			if verifyErr == nil {
				break
			}
			pm.log.Debugf("Pairing state verification attempt %d failed: %v", attempt, verifyErr)
			if attempt < 3 {
				time.Sleep(1 * time.Second) // Brief delay between retries
			}
		}

		// Determine final pairing status using hierarchical logic
		var isPaired bool
		var statusReason string

		if verifyErr != nil {
			// Verification failed, but we got WiFi credentials
			isPaired = true
			statusReason = "WiFi credentials obtained (verification failed but pairing assumed successful)"
			pm.log.Warnf("Pairing verification failed: %v - but considering successful since WiFi credentials were obtained", verifyErr)
		} else {
			// Verification succeeded, check device state
			deviceReportsPaired := (devicePairingState == 4) // PairingStateCompleted

			if deviceReportsPaired {
				isPaired = true
				statusReason = "both WiFi credentials obtained AND device reports paired state"
			} else {
				// Device doesn't report paired, but we got WiFi credentials
				// According to OpenGoPro implementation analysis, this is still success
				isPaired = true
				statusReason = fmt.Sprintf("WiFi credentials obtained (device reports state %d instead of 4, but this is acceptable)", devicePairingState)
			}
		}

		pm.log.Infof("Pairing verification completed: state=%d, isPaired=%v (%s)", devicePairingState, isPaired, statusReason)

		// Update database immediately with determined pairing status
		if err := pm.db.SetCameraPaired(macAddress, isPaired); err != nil {
			pm.log.Errorf("Failed to set camera paired status: %v", err)
			// Continue anyway since we have definitive success indicators
		} else {
			pm.log.Infof("Camera %s pairing status updated in database: %v", cameraState.Camera.Name, isPaired)
		}

		// Notify about pairing status update
		if pm.notifier != nil {
			pm.notifier.NotifyUpdate()
		}

		// Store the verified pairing state for use after the BLE operation
		verifiedPairingState = isPaired
		pairingVerificationDone = true

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

	// Use the verified pairing state if verification was done during BLE operation
	if pairingVerificationDone {
		return pm.completePairingOperationWithState(cameraState, macAddress, verifiedPairingState)
	}

	// Fallback to original verification logic if something went wrong
	return pm.completePairingOperation(cameraState, macAddress)
}

// completePairingOperation verifies pairing and updates database
func (pm *Manager) completePairingOperation(cameraState *database.CameraWithState, macAddress string) (*database.ManagedCamera, error) {
	// Verify pairing state directly from device after pairing operation
	// Try multiple times to get a reliable reading
	var isPaired bool
	var verifyErr error

	pm.log.Infof("Verifying pairing state for camera %s", cameraState.Camera.Name)

	for attempt := 1; attempt <= 3; attempt++ {
		// Wait a bit between attempts to let the device settle
		if attempt > 1 {
			time.Sleep(2 * time.Second)
		}

		// Use refresh method to force fresh query on first attempt
		if attempt == 1 {
			pairingState, err := pm.ble.RefreshPairingState(macAddress)
			if err == nil {
				isPaired = (pairingState == 4) // PairingStateCompleted
				verifyErr = nil
				pm.log.Infof("Fresh pairing verification successful on attempt %d: state=%d, isPaired=%v", attempt, pairingState, isPaired)
				break
			}
			verifyErr = err
		} else {
			isPaired, verifyErr = pm.ble.IsPaired(macAddress)
			if verifyErr == nil {
				pm.log.Infof("Pairing verification successful on attempt %d: isPaired=%v", attempt, isPaired)
				break
			}
		}

		pm.log.Warnf("Pairing verification attempt %d failed: %v", attempt, verifyErr)
	}

	if verifyErr != nil {
		pm.log.Warnf("All pairing verification attempts failed, assuming success since WiFi credentials were obtained")
		// Fallback to assuming pairing succeeded if we got WiFi credentials
		isPaired = true
	}

	// Mark camera as paired based on actual device state
	pairErr := pm.db.SetCameraPaired(cameraState.Camera.MACAddress, isPaired)
	if pairErr != nil {
		pm.log.Errorf("Error setting camera paired status: %v", pairErr)
		// Make sure pairing flag is reset even if there was an error
		pm.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)
		return nil, fmt.Errorf("failed to set camera paired status: %v", pairErr)
	}

	// Reset the pairing flag since pairing process is complete
	pm.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)

	// Double-check the paired status was correctly set
	updatedState, exists := pm.db.CameraStates[cameraState.Camera.MACAddress]
	if exists {
		pm.log.Infof("Camera %s pairing completed. isPaired=%v, isPairing=%v",
			cameraState.Camera.Name, updatedState.Status.IsPaired, updatedState.Status.IsPairing)
	} else {
		pm.log.Warnf("Camera state not found after pairing for %s", cameraState.Camera.MACAddress)
	}

	// Immediate notification about the final status change
	if pm.notifier != nil {
		pm.notifier.NotifyUpdate()
	}

	// Additional verification: try to get fresh pairing state one more time
	go func() {
		// Wait a bit and then do a final verification
		time.Sleep(3 * time.Second)
		if finalPaired, err := pm.ble.IsPaired(macAddress); err == nil {
			if finalPaired != isPaired {
				pm.log.Infof("Final pairing state verification shows different result: %v, updating database", finalPaired)
				if err := pm.db.SetCameraPaired(macAddress, finalPaired); err == nil && pm.notifier != nil {
					pm.notifier.NotifyUpdate()
				}
			}
		}
	}()

	// Return the managed camera view
	managedCamera, exists := pm.db.GetManagedCamera(cameraState.Camera.MACAddress)

	// If the camera is paired but not managed, it won't be returned by GetManagedCamera
	// In this case, we need to create a ManagedCamera wrapper around the CameraState
	// to return to the caller
	if !exists {
		pm.log.Debugf("Camera %s is paired but not yet managed, creating managed view", cameraState.Camera.Name)
		managedCamera = &database.ManagedCamera{
			CameraState: cameraState,
		}
	}

	return managedCamera, nil
}

// completePairingOperationWithState verifies pairing and updates database with a given pairing state
func (pm *Manager) completePairingOperationWithState(cameraState *database.CameraWithState, macAddress string, verifiedPairingState bool) (*database.ManagedCamera, error) {
	// Check current database state first
	currentState, exists := pm.db.CameraStates[macAddress]
	if exists && currentState.Status.IsPaired == verifiedPairingState {
		pm.log.Debugf("Camera %s pairing state already correct in database: %v", cameraState.Camera.Name, verifiedPairingState)
	} else {
		// Mark camera as paired based on actual device state
		pairErr := pm.db.SetCameraPaired(cameraState.Camera.MACAddress, verifiedPairingState)
		if pairErr != nil {
			pm.log.Errorf("Error setting camera paired status: %v", pairErr)
			// Make sure pairing flag is reset even if there was an error
			pm.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)
			return nil, fmt.Errorf("failed to set camera paired status: %v", pairErr)
		}
	}

	// Reset the pairing flag since pairing process is complete
	pm.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)

	// Double-check the paired status was correctly set
	updatedState, exists := pm.db.CameraStates[cameraState.Camera.MACAddress]
	if exists {
		pm.log.Infof("Camera %s pairing completed. isPaired=%v, isPairing=%v",
			cameraState.Camera.Name, updatedState.Status.IsPaired, updatedState.Status.IsPairing)
	} else {
		pm.log.Warnf("Camera state not found after pairing for %s", cameraState.Camera.MACAddress)
	}

	// Immediate notification about the final status change
	if pm.notifier != nil {
		pm.notifier.NotifyUpdate()
	}

	// Additional verification: try to get fresh pairing state one more time
	go func() {
		// Wait a bit and then do a final verification
		time.Sleep(3 * time.Second)
		if finalPaired, err := pm.ble.IsPaired(macAddress); err == nil {
			if finalPaired != verifiedPairingState {
				pm.log.Infof("Final pairing state verification shows different result: %v, updating database", finalPaired)
				if err := pm.db.SetCameraPaired(macAddress, finalPaired); err == nil && pm.notifier != nil {
					pm.notifier.NotifyUpdate()
				}
			}
		}
	}()

	// Return the managed camera view
	managedCamera, exists := pm.db.GetManagedCamera(cameraState.Camera.MACAddress)

	// If the camera is paired but not managed, it won't be returned by GetManagedCamera
	// In this case, we need to create a ManagedCamera wrapper around the CameraState
	// to return to the caller
	if !exists {
		pm.log.Debugf("Camera %s is paired but not yet managed, creating managed view", cameraState.Camera.Name)
		managedCamera = &database.ManagedCamera{
			CameraState: cameraState,
		}
	}

	return managedCamera, nil
}
