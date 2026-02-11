package pairing

import (
	"fmt"
)

// SyncDevicePairingStates checks actual device pairing states and updates database accordingly
func (pm *Manager) SyncDevicePairingStates(notifier func()) {
	managedCameras := pm.db.GetCamerasForManagedPool()

	for _, camera := range managedCameras {
		macAddress := camera.CameraState.Camera.MACAddress

		if !camera.CameraState.Status.IsReachable || camera.CameraState.Status.IsPairing {
			continue
		}

		devicePaired, err := pm.ble.IsPaired(macAddress)
		if err != nil {
			pm.log.Debugf("Failed to query pairing state for camera %s: %v",
				camera.CameraState.Camera.Name, err)
			continue
		}

		dbPairingState := camera.CameraState.Status.IsPaired
		if dbPairingState != devicePaired {
			if err := pm.db.SetCameraPaired(macAddress, devicePaired); err != nil {
				pm.log.Errorf("Failed to update pairing state for camera %s: %v",
					camera.CameraState.Camera.Name, err)
			} else {
				pm.log.Infof("Synced pairing state for camera %s: %v -> %v",
					camera.CameraState.Camera.Name, dbPairingState, devicePaired)
				notifier()
			}
		}
	}
}

// VerifyAndFixPairingState checks a specific camera's pairing state and fixes database if needed
func (pm *Manager) VerifyAndFixPairingState(cameraID string, notifier func()) error {
	cameraState, found := pm.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("camera with ID %s not found", cameraID)
	}

	macAddress := cameraState.Camera.MACAddress
	devicePaired, err := pm.ble.IsPaired(macAddress)
	if err != nil {
		return fmt.Errorf("failed to query device pairing state: %v", err)
	}

	if cameraState.Status.IsPaired != devicePaired {
		pm.log.Infof("Fixing pairing state mismatch for camera %s: database=%v, device=%v",
			cameraState.Camera.Name, cameraState.Status.IsPaired, devicePaired)
		if err := pm.db.SetCameraPaired(macAddress, devicePaired); err != nil {
			return fmt.Errorf("failed to update pairing state: %v", err)
		}
		notifier()
	}

	return nil
}
