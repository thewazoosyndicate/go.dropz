package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/database"
)

// PairCamera pairs with a GoPro camera using BLE
func (m *GoProManager) PairCamera(cameraID string) (*database.ManagedCamera, error) {
	// First check if we can find the camera
	var cameraState *database.CameraWithState

	// Check all cameras
	for _, state := range m.db.CameraStates {
		if state.Camera.ID == cameraID {
			cameraState = state
			break
		}
	}

	if cameraState == nil {
		return nil, fmt.Errorf("camera with ID %s not found", cameraID)
	}

	// First check if the device is already paired by querying it directly
	macAddress := cameraState.Camera.MACAddress
	m.log.Infof("Checking current pairing state for camera %s", cameraState.Camera.Name)

	// Try to check current pairing state from device
	if isPaired, err := m.ble.IsPaired(macAddress); err == nil && isPaired {
		m.log.Infof("Camera %s is already paired on device, updating database", cameraState.Camera.Name)
		// Update database to reflect actual device state
		if err := m.db.SetCameraPaired(macAddress, true); err != nil {
			m.log.Warnf("Failed to update database pairing status: %v", err)
		}
		// Notify about the status change
		if m.notifier != nil {
			m.notifier.NotifyUpdate()
		}
		// Return the managed camera view
		managedCamera, _ := m.db.GetManagedCamera(macAddress)
		return managedCamera, nil
	}

	// Mark camera as pairing
	m.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, true)
	m.log.Infof("Starting pairing process for camera %s", cameraState.Camera.Name)

	// Notify about the status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	// TODO: Implement actual pairing logic here
	// For now, just simulate a successful pairing
	// Create a context with a reasonable timeout for pairing operation
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	// Perform the BLE pairing operation
	bleErr := m.BLEOperation(ctx, "PairCamera", func() error {
		// Get the MAC address, which is used for BLE operations
		macAddress := cameraState.Camera.MACAddress

		// Connect to the device with enhanced model-specific pairing
		m.log.Infof("Connecting to device %s with enhanced pairing", macAddress)

		// Notify about connection attempt
		if m.notifier != nil {
			m.notifier.NotifyUpdate()
		}

		if err := m.ble.ConnectWithEnhancedPairing(macAddress); err != nil {
			return fmt.Errorf("failed to connect with enhanced pairing: %v", err)
		}

		m.log.Infof("Successfully connected to device %s, retrieving WiFi credentials", macAddress)

		// Notify about successful connection
		if m.notifier != nil {
			m.notifier.NotifyUpdate()
		}

		// Get WiFi credentials
		ssid, password, err := m.ble.GetWifiCredentials(macAddress)
		if err != nil {
			// Try to disconnect gracefully even if getting WiFi credentials failed
			_ = m.ble.Disconnect(macAddress)
			return fmt.Errorf("failed to get WiFi credentials: %v", err)
		}

		// Store WiFi credentials in the database
		m.log.Infof("Obtained WiFi credentials for camera %s: SSID=%s", cameraID, ssid)

		// Update the camera's WiFi credentials in the cameraState object
		cameraState.Camera.WiFiSSID = ssid
		cameraState.Camera.WiFiPassword = password

		// Notify about WiFi credentials obtained
		if m.notifier != nil {
			m.notifier.NotifyUpdate()
		}

		// Disconnect from the camera
		if err := m.ble.Disconnect(macAddress); err != nil {
			m.log.Warnf("Failed to disconnect from camera: %v", err)
			// This is not critical, we can continue
		}

		m.log.Infof("Pairing operation completed for camera %s", cameraState.Camera.Name)

		return nil
	})

	// Handle any BLE operation errors
	if bleErr != nil {
		m.log.Errorf("BLE pairing operation failed: %v", bleErr)
		// Make sure pairing flag is reset if BLE operation failed
		m.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)
		return nil, fmt.Errorf("failed in BLE pairing operation: %v", bleErr)
	}

	// Verify pairing state directly from device after pairing operation
	// Try multiple times to get a reliable reading
	var isPaired bool
	var verifyErr error

	m.log.Infof("Verifying pairing state for camera %s", cameraState.Camera.Name)

	for attempt := 1; attempt <= 3; attempt++ {
		// Wait a bit between attempts to let the device settle
		if attempt > 1 {
			time.Sleep(2 * time.Second)
		}

		// Use refresh method to force fresh query on first attempt
		if attempt == 1 {
			pairingState, err := m.ble.RefreshPairingState(macAddress)
			if err == nil {
				isPaired = (pairingState == 4) // PairingStateCompleted
				verifyErr = nil
				m.log.Infof("Fresh pairing verification successful on attempt %d: state=%d, isPaired=%v", attempt, pairingState, isPaired)
				break
			}
			verifyErr = err
		} else {
			isPaired, verifyErr = m.ble.IsPaired(macAddress)
			if verifyErr == nil {
				m.log.Infof("Pairing verification successful on attempt %d: isPaired=%v", attempt, isPaired)
				break
			}
		}

		m.log.Warnf("Pairing verification attempt %d failed: %v", attempt, verifyErr)
	}

	if verifyErr != nil {
		m.log.Warnf("All pairing verification attempts failed, assuming success since WiFi credentials were obtained")
		// Fallback to assuming pairing succeeded if we got WiFi credentials
		isPaired = true
	}

	// Mark camera as paired based on actual device state
	pairErr := m.db.SetCameraPaired(cameraState.Camera.MACAddress, isPaired)
	if pairErr != nil {
		m.log.Errorf("Error setting camera paired status: %v", pairErr)
		// Make sure pairing flag is reset even if there was an error
		m.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)
		return nil, fmt.Errorf("failed to set camera paired status: %v", pairErr)
	}

	// Reset the pairing flag since pairing process is complete
	m.db.UpdateCameraPairingStatus(cameraState.Camera.MACAddress, false)

	// Double-check the paired status was correctly set
	updatedState, exists := m.db.CameraStates[cameraState.Camera.MACAddress]
	if exists {
		m.log.Infof("Camera %s pairing completed. isPaired=%v, isPairing=%v",
			cameraState.Camera.Name, updatedState.Status.IsPaired, updatedState.Status.IsPairing)
	} else {
		m.log.Warnf("Camera state not found after pairing for %s", cameraState.Camera.MACAddress)
	}

	// Immediate notification about the final status change
	if m.notifier != nil {
		m.notifier.NotifyUpdate()
	}

	// Additional verification: try to get fresh pairing state one more time
	go func() {
		// Wait a bit and then do a final verification
		time.Sleep(3 * time.Second)
		if finalPaired, err := m.ble.IsPaired(macAddress); err == nil {
			if finalPaired != isPaired {
				m.log.Infof("Final pairing state verification shows different result: %v, updating database", finalPaired)
				if err := m.db.SetCameraPaired(macAddress, finalPaired); err == nil && m.notifier != nil {
					m.notifier.NotifyUpdate()
				}
			}
		}
	}()

	// Return the managed camera view
	managedCamera, exists := m.db.GetManagedCamera(cameraState.Camera.MACAddress)

	// If the camera is paired but not managed, it won't be returned by GetManagedCamera
	// In this case, we need to create a ManagedCamera wrapper around the CameraState
	// to return to the caller
	if !exists {
		m.log.Debugf("Camera %s is paired but not yet managed, creating managed view", cameraState.Camera.Name)
		managedCamera = &database.ManagedCamera{
			CameraState: cameraState,
		}
	}

	return managedCamera, nil
}

// syncDevicePairingStates checks actual device pairing states and updates database accordingly
func (m *GoProManager) syncDevicePairingStates() {
	managedCameras := m.db.GetAllManagedCameras()

	for _, camera := range managedCameras {
		macAddress := camera.CameraState.Camera.MACAddress
		dbPairingState := camera.CameraState.Status.IsPaired

		// Skip if camera is not reachable
		if !camera.CameraState.Status.IsReachable {
			m.log.Debugf("Skipping pairing sync for unreachable camera %s", camera.CameraState.Camera.Name)
			continue
		}

		// Skip if camera is currently pairing to avoid interference
		if camera.CameraState.Status.IsPairing {
			m.log.Debugf("Skipping pairing sync for camera %s currently being paired", camera.CameraState.Camera.Name)
			continue
		}

		// Query actual device pairing state
		devicePaired, err := m.ble.IsPaired(macAddress)
		if err != nil {
			m.log.Debugf("Failed to query pairing state for camera %s: %v",
				camera.CameraState.Camera.Name, err)
			continue
		}

		// Update database if states don't match
		if dbPairingState != devicePaired {
			m.log.Infof("Syncing pairing state for camera %s: database=%v, device=%v",
				camera.CameraState.Camera.Name, dbPairingState, devicePaired)

			if err := m.db.SetCameraPaired(macAddress, devicePaired); err != nil {
				m.log.Errorf("Failed to update pairing state for camera %s: %v",
					camera.CameraState.Camera.Name, err)
			} else {
				m.log.Infof("Successfully updated pairing state for camera %s to %v",
					camera.CameraState.Camera.Name, devicePaired)
				// Notify about the state change
				if m.notifier != nil {
					m.notifier.NotifyUpdate()
				}
			}
		} else {
			m.log.Debugf("Pairing state already in sync for camera %s: %v",
				camera.CameraState.Camera.Name, devicePaired)
		}
	}
}

// VerifyAndFixPairingState checks a specific camera's pairing state and fixes database if needed
func (m *GoProManager) VerifyAndFixPairingState(cameraID string) error {
	// Find the camera
	var cameraState *database.CameraWithState
	for _, state := range m.db.CameraStates {
		if state.Camera.ID == cameraID {
			cameraState = state
			break
		}
	}

	if cameraState == nil {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	macAddress := cameraState.Camera.MACAddress
	dbPairingState := cameraState.Status.IsPaired

	// Query actual device pairing state
	devicePaired, err := m.ble.IsPaired(macAddress)
	if err != nil {
		return fmt.Errorf("failed to query device pairing state: %v", err)
	}

	// Fix database if states don't match
	if dbPairingState != devicePaired {
		m.log.Infof("Fixing pairing state mismatch for camera %s: database=%v, device=%v",
			cameraState.Camera.Name, dbPairingState, devicePaired)

		if err := m.db.SetCameraPaired(macAddress, devicePaired); err != nil {
			return fmt.Errorf("failed to update pairing state: %v", err)
		}

		// Notify about the state change
		if m.notifier != nil {
			m.notifier.NotifyUpdate()
		}

		return nil
	}

	m.log.Debugf("Pairing state verified for camera %s: %v", cameraState.Camera.Name, devicePaired)
	return nil
}
