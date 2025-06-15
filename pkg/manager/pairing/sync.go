package pairing

import (
	"fmt"

	"github.com/dropz/dropz/pkg/database"
)

// SyncDevicePairingStates checks actual device pairing states and updates database accordingly
func (pm *Manager) SyncDevicePairingStates() {
	managedCameras := pm.db.GetAllManagedCameras()

	for _, camera := range managedCameras {
		macAddress := camera.CameraState.Camera.MACAddress
		dbPairingState := camera.CameraState.Status.IsPaired

		// Skip if camera is not reachable
		if !camera.CameraState.Status.IsReachable {
			pm.log.Tracef("Skipping pairing sync for unreachable camera %s", camera.CameraState.Camera.Name)
			continue
		}

		// Skip if camera is currently pairing to avoid interference
		if camera.CameraState.Status.IsPairing {
			pm.log.Tracef("Skipping pairing sync for camera %s currently being paired", camera.CameraState.Camera.Name)
			continue
		}

		// Query actual device pairing state
		devicePaired, err := pm.ble.IsPaired(macAddress)
		if err != nil {
			pm.log.Debugf("Failed to query pairing state for camera %s: %v",
				camera.CameraState.Camera.Name, err)
			continue
		}

		// Update database if states don't match
		if dbPairingState != devicePaired {
			pm.log.Infof("Syncing pairing state for camera %s: database=%v, device=%v",
				camera.CameraState.Camera.Name, dbPairingState, devicePaired)

			// Only update when device reports paired; skip when device reports unpaired to preserve existing pairing
			if devicePaired {
				if err := pm.db.SetCameraPaired(macAddress, true); err != nil {
					pm.log.Errorf("Failed to update pairing state for camera %s: %v",
						camera.CameraState.Camera.Name, err)
				} else {
					pm.log.Infof("Successfully updated pairing state for camera %s to paired",
						camera.CameraState.Camera.Name)
					if pm.notifier != nil {
						pm.notifier.NotifyUpdate()
					}
				}
			} else {
				pm.log.Warnf("Device reports unpaired for camera %s; skipping database update to avoid losing pairing", camera.CameraState.Camera.Name)
			}
		} else {
			pm.log.Tracef("Pairing state already in sync for camera %s: %v",
				camera.CameraState.Camera.Name, devicePaired)
		}
	}
}

// VerifyAndFixPairingState checks a specific camera's pairing state and fixes database if needed
func (pm *Manager) VerifyAndFixPairingState(cameraID string) error {
	// Find the camera
	var cameraState *database.CameraWithState
	for _, state := range pm.db.CameraStates {
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
	devicePaired, err := pm.ble.IsPaired(macAddress)
	if err != nil {
		return fmt.Errorf("failed to query device pairing state: %v", err)
	}

	// Fix database if states don't match
	if dbPairingState != devicePaired {
		pm.log.Infof("Fixing pairing state mismatch for camera %s: database=%v, device=%v",
			cameraState.Camera.Name, dbPairingState, devicePaired)

		if err := pm.db.SetCameraPaired(macAddress, devicePaired); err != nil {
			return fmt.Errorf("failed to update pairing state: %v", err)
		}

		// Notify about the state change
		if pm.notifier != nil {
			pm.notifier.NotifyUpdate()
		}

		return nil
	}

	pm.log.Tracef("Pairing state verified for camera %s: %v", cameraState.Camera.Name, devicePaired)
	return nil
}
