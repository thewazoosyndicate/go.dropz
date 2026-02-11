package pairing

// SyncDevicePairingStates checks actual device pairing states and updates database accordingly
func (pm *Manager) SyncDevicePairingStates(notifier func()) {
	managedCameras := pm.db.GetCamerasForManagedPool()

	for _, camera := range managedCameras {
		macAddress := camera.CameraState.Camera.MACAddress

		if !camera.CameraState.Status.IsReachable || camera.CameraState.Status.IsPairing {
			continue
		}

		// Skip cameras without an active BLE connection — querying would always fail
		if !pm.ble.IsConnected(macAddress) {
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

