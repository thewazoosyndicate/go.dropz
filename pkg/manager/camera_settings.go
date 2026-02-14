package manager

import (
	"context"
	"fmt"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
)

// GetCameraSettings queries a camera's current settings and valid values via BLE.
func (m *GoProManager) GetCameraSettings(cameraID string) ([]database.CameraSettingInfo, error) {
	cs, found := m.db.GetCameraByID(cameraID)
	if !found {
		return nil, fmt.Errorf("camera %s not found", cameraID)
	}
	if !cs.Status.IsPaired {
		return nil, fmt.Errorf("camera %s is not paired", cameraID)
	}
	if !cs.Status.IsReachable {
		return nil, fmt.Errorf("camera %s is not reachable", cameraID)
	}

	bleAddress := cs.Camera.BLEAddress
	alreadyConnected := m.ble.IsConnected(bleAddress)

	var settings []database.CameraSettingInfo
	err := m.BLEOperation(context.Background(), false, func() error {
		if !alreadyConnected {
			if connErr := m.ble.ConnectForSettings(bleAddress); connErr != nil {
				return fmt.Errorf("connect failed: %v", connErr)
			}
		}

		knownSettingIDs := []byte{
			ble.SettingVideoResolution, ble.SettingFPS,
			ble.SettingAutoPowerDown, ble.SettingGPS,
			ble.SettingVideoAspectRatio, ble.SettingVideoDigitalLens,
			ble.SettingPhotoDigitalLens, ble.SettingPhotoOutput,
			ble.SettingMediaFormat, ble.SettingAntiFlicker,
			ble.SettingHypersmooth, ble.SettingVideoBitRate,
			ble.SettingBitDepth, ble.SettingVideoProfile,
		}

		values, err := m.ble.QuerySettingValues(bleAddress, knownSettingIDs)
		if err != nil {
			if !alreadyConnected {
				m.ble.Disconnect(bleAddress)
			}
			return fmt.Errorf("query setting values failed: %v", err)
		}

		// Query capabilities for all reported settings in one request
		settingIDs := make([]byte, 0, len(values))
		for id := range values {
			settingIDs = append(settingIDs, id)
		}

		caps, capErr := m.ble.QuerySettingCapabilities(bleAddress, settingIDs)
		if capErr != nil {
			m.log.Warnf("QuerySettingCapabilities failed: %v", capErr)
			caps = make(map[byte][]byte)
		}

		if !alreadyConnected {
			m.ble.Disconnect(bleAddress)
		}

		for id, val := range values {
			var currentValue int32
			if len(val) >= 1 {
				currentValue = int32(val[0])
			}

			var validValues []int32
			if capBytes, ok := caps[id]; ok {
				validValues = make([]int32, len(capBytes))
				for i, b := range capBytes {
					validValues[i] = int32(b)
				}
			}

			settings = append(settings, database.CameraSettingInfo{
				ID:           int32(id),
				CurrentValue: currentValue,
				ValidValues:  validValues,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// SetCameraSetting writes a setting value to a camera via BLE.
func (m *GoProManager) SetCameraSetting(cameraID string, settingID, value int32) error {
	cs, found := m.db.GetCameraByID(cameraID)
	if !found {
		return fmt.Errorf("camera %s not found", cameraID)
	}
	if !cs.Status.IsPaired {
		return fmt.Errorf("camera %s is not paired", cameraID)
	}
	if !cs.Status.IsReachable {
		return fmt.Errorf("camera %s is not reachable", cameraID)
	}

	bleAddress := cs.Camera.BLEAddress
	alreadyConnected := m.ble.IsConnected(bleAddress)

	return m.BLEOperation(context.Background(), false, func() error {
		if !alreadyConnected {
			if connErr := m.ble.ConnectForSettings(bleAddress); connErr != nil {
				return fmt.Errorf("connect failed: %v", connErr)
			}
		}

		err := m.ble.WriteSetting(bleAddress, byte(settingID), byte(value))

		if !alreadyConnected {
			m.ble.Disconnect(bleAddress)
		}

		return err
	})
}

// SetGroupSetting writes a setting to all cameras in a group, returning per-camera results.
func (m *GoProManager) SetGroupSetting(groupID string, settingID, value int32) (map[string]string, error) {
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("group %s not found", groupID)
	}

	results := make(map[string]string, len(group.CameraIDs))
	for _, camID := range group.CameraIDs {
		if err := m.SetCameraSetting(camID, settingID, value); err != nil {
			results[camID] = err.Error()
		} else {
			results[camID] = ""
		}
	}

	return results, nil
}
