package manager

import (
	"fmt"
	"sort"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/model"
)

// settingLabel resolves display names from the generated spec table.
// Unknown settings stay configurable with numeric labels: newer cameras
// may expose settings the table predates.
func settingLabel(id byte) string {
	if def, ok := ble.SettingDefs[id]; ok {
		return def.Name
	}
	return fmt.Sprintf("Setting %d", id)
}

func optionLabel(id byte, value int64) string {
	if def, ok := ble.SettingDefs[id]; ok {
		if name, ok := def.Options[value]; ok {
			return name
		}
	}
	return fmt.Sprintf("%d", value)
}

// buildSettingsSnapshot merges current values with the camera-reported
// capabilities into the stored representation.
func buildSettingsSnapshot(values map[byte]int64, caps map[byte][]int64) []model.CameraSetting {
	settings := make([]model.CameraSetting, 0, len(values))
	for id, value := range values {
		s := model.CameraSetting{
			ID:        int32(id),
			Name:      settingLabel(id),
			Value:     value,
			ValueName: optionLabel(id, value),
		}
		for _, opt := range caps[id] {
			s.Options = append(s.Options, model.SettingOption{Value: opt, Name: optionLabel(id, opt)})
		}
		sort.Slice(s.Options, func(i, j int) bool { return s.Options[i].Value < s.Options[j].Value })
		settings = append(settings, s)
	}
	sort.Slice(settings, func(i, j int) bool { return settings[i].ID < settings[j].ID })
	return settings
}

// settingsSession connects, runs op, and disconnects without Sleep.
// Requires a managed, paired, idle camera.
func (m *GoProManager) settingsSession(cameraID string, op func(bleAddress string) error) error {
	cs, ok := m.db.GetCameraByID(cameraID)
	if !ok {
		return fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}
	if !cs.InManagedPool() || !cs.Status.IsPaired {
		return fmt.Errorf("%w: %s", model.ErrNotManagedPaired, cs.Camera.Name)
	}
	if cs.Status.IsSyncing {
		return fmt.Errorf("camera %s is syncing", cs.Camera.Name)
	}
	bleAddress := cs.Camera.BLEAddress

	return m.BLEOperation(m.ctx, true, func() error {
		if err := m.ble.ConnectForStatusCheck(bleAddress); err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer m.ble.DisconnectQuietly(bleAddress)
		return op(bleAddress)
	})
}

// refreshSettingsLocked queries values and capabilities on an open
// connection and persists the snapshot.
func (m *GoProManager) refreshSettingsLocked(cameraID, bleAddress string) ([]model.CameraSetting, error) {
	values, err := m.ble.GetSettingValues(bleAddress, nil)
	if err != nil {
		return nil, err
	}
	caps, err := m.ble.GetSettingCapabilities(bleAddress, nil)
	if err != nil {
		return nil, err
	}
	snapshot := buildSettingsSnapshot(values, caps)
	m.db.UpdateCameraByID(cameraID, func(cs *model.CameraWithState) {
		cs.Metadata.Settings = snapshot
		cs.Metadata.SettingsUpdated = time.Now()
	})
	m.notify()
	return snapshot, nil
}

// RefreshCameraSettings reads the camera's settings and capabilities over
// BLE and returns the stored snapshot.
func (m *GoProManager) RefreshCameraSettings(cameraID string) ([]model.CameraSetting, time.Time, error) {
	var snapshot []model.CameraSetting
	err := m.settingsSession(cameraID, func(bleAddress string) error {
		var opErr error
		snapshot, opErr = m.refreshSettingsLocked(cameraID, bleAddress)
		return opErr
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	return snapshot, time.Now(), nil
}

// GetCameraSettings returns the last-known snapshot without touching the
// camera; empty if never refreshed.
func (m *GoProManager) GetCameraSettings(cameraID string) ([]model.CameraSetting, time.Time, error) {
	cs, ok := m.db.GetCameraByID(cameraID)
	if !ok {
		return nil, time.Time{}, fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
	}
	return cs.Metadata.Settings, cs.Metadata.SettingsUpdated, nil
}

// ApplyCameraSettings writes the requested values over BLE.
// Settings interdepend (resolution constrains fps), so failed writes get
// one retry pass after the rest applied. Ascending ID order puts
// resolution (2) before fps (3), the common dependency direction.
// A busy or recording camera is refused entirely.
func (m *GoProManager) ApplyCameraSettings(cameraID string, changes map[int32]int64) ([]model.SettingApplyResult, error) {
	if len(changes) == 0 {
		return nil, fmt.Errorf("no settings to apply")
	}

	ids := make([]int32, 0, len(changes))
	for id := range changes {
		if id < 0 || id > 255 {
			return nil, fmt.Errorf("setting id %d out of range", id)
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var results []model.SettingApplyResult
	err := m.settingsSession(cameraID, func(bleAddress string) error {
		statuses, err := m.ble.QueryStatuses(bleAddress, []byte{ble.StatusSystemBusy, ble.StatusEncoding})
		if err != nil {
			return fmt.Errorf("status query: %w", err)
		}
		if cameraInUse(statuses) {
			return fmt.Errorf("camera is busy or recording")
		}

		failed := make(map[int32]string)
		for _, id := range ids {
			if err := m.ble.SetSetting(bleAddress, byte(id), changes[id]); err != nil {
				failed[id] = err.Error()
			}
		}
		// Retry pass: a value rejected earlier may be valid now that the
		// settings it depends on changed.
		for _, id := range ids {
			if _, wasFailed := failed[id]; !wasFailed {
				continue
			}
			if err := m.ble.SetSetting(bleAddress, byte(id), changes[id]); err == nil {
				delete(failed, id)
			}
		}

		for _, id := range ids {
			results = append(results, model.SettingApplyResult{ID: id, Error: failed[id]})
		}

		if _, err := m.refreshSettingsLocked(cameraID, bleAddress); err != nil {
			m.log.Warn("Settings refresh after apply failed", "camera", cameraID, "err", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ApplyGroupSettings applies the same changes to every camera in a group,
// sequentially (one BLE session at a time). Each camera accepts what its
// model and state allow; per-camera results report the rest.
func (m *GoProManager) ApplyGroupSettings(groupID string, changes map[int32]int64) ([]model.GroupSettingsResult, error) {
	group, ok := m.db.GetGroup(groupID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", model.ErrGroupNotFound, groupID)
	}
	if len(group.CameraIDs) == 0 {
		return nil, fmt.Errorf("group %s has no cameras", group.Name)
	}

	results := make([]model.GroupSettingsResult, 0, len(group.CameraIDs))
	for _, cameraID := range group.CameraIDs {
		r := model.GroupSettingsResult{CameraID: cameraID}
		applied, err := m.ApplyCameraSettings(cameraID, changes)
		if err != nil {
			r.Error = err.Error()
		} else {
			r.Results = applied
		}
		results = append(results, r)
	}
	return results, nil
}
