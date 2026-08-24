package manager

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/internal/manager/syncer"
	"github.com/dropz/dropz/internal/model"
	"github.com/google/uuid"
)

// CreateGroup creates a new group with the specified cameras
func (m *GoProManager) CreateGroup(name string, cameraIDs []string) (*model.Group, error) {
	if err := m.validateCamerasExist(cameraIDs); err != nil {
		return nil, err
	}

	group := &model.Group{
		ID:        uuid.New().String(),
		Name:      name,
		CameraIDs: cameraIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	m.log.Info("Group created", "group_id", group.ID, "group", name, "cameras", len(cameraIDs))
	return group, nil
}

// UpdateGroup updates an existing group
func (m *GoProManager) UpdateGroup(groupID, name string, cameraIDs []string) (*model.Group, error) {
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("%w: %s", model.ErrGroupNotFound, groupID)
	}

	if err := m.validateCamerasExist(cameraIDs); err != nil {
		return nil, err
	}

	group.Name = name
	group.CameraIDs = cameraIDs
	group.UpdatedAt = time.Now()

	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	m.log.Info("Group updated", "group_id", groupID, "group", name, "cameras", len(cameraIDs))
	return group, nil
}

// DeleteGroup deletes a group
func (m *GoProManager) DeleteGroup(groupID string) error {
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("%w: %s", model.ErrGroupNotFound, groupID)
	}

	if err := m.db.RemoveGroup(groupID); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	m.log.Info("Group deleted", "group_id", groupID, "group", group.Name)
	return nil
}

// LoadGroup replaces the current managed set with the group's cameras
func (m *GoProManager) LoadGroup(groupID string) error {
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("%w: %s", model.ErrGroupNotFound, groupID)
	}

	groupCameras := make(map[string]bool, len(group.CameraIDs))
	for _, id := range group.CameraIDs {
		groupCameras[id] = true
	}

	for _, cam := range m.db.GetAllCameras() {
		id := cam.Camera.ID
		inGroup := groupCameras[id]

		if inGroup && !cam.Status.IsManaged {
			if _, err := m.ManageCamera(id); err != nil {
				m.log.Warn("Failed to manage camera during group load", "group_id", groupID, "camera_id", id, "err", err)
			}
		} else if !inGroup && cam.Status.IsManaged {
			if err := m.UnmanageCamera(id); err != nil {
				m.log.Warn("Failed to unmanage camera during group load", "group_id", groupID, "camera_id", id, "err", err)
			}
		}
	}

	m.log.Info("Group loaded", "group_id", groupID, "group", group.Name, "cameras", len(group.CameraIDs))
	m.notify()
	return nil
}

// MoveCamerasToGroup puts the cameras in a group (empty: none) and
// returns every group, since membership left another one.
func (m *GoProManager) MoveCamerasToGroup(cameraIDs []string, groupID string) ([]*model.Group, error) {
	if err := m.validateCamerasExist(cameraIDs); err != nil {
		return nil, err
	}
	if err := m.db.MoveCamerasToGroup(cameraIDs, groupID); err != nil {
		return nil, err
	}
	m.log.Info("Cameras moved to group", "group_id", groupID, "cameras", len(cameraIDs))
	m.notify()
	return m.db.GetAllGroups(), nil
}

// SetGroupSync pauses or resumes a group's auto-sync. Exclusive resumes
// it and pauses the rest: the rig switch. Pausing also drops the group's
// queued auto-syncs so nothing starts behind the user's back.
func (m *GoProManager) SetGroupSync(groupID string, paused, exclusive bool) ([]*model.Group, error) {
	if err := m.db.SetGroupSync(groupID, paused, exclusive); err != nil {
		return nil, err
	}
	for _, entry := range m.db.GetSyncQueue() {
		if entry.Priority < syncer.SyncPriorityManual && m.db.IsCameraSyncPaused(entry.CameraID) {
			_ = m.db.RemoveSyncQueueEntry(entry.CameraID)
		}
	}
	m.log.Info("Group sync changed", "group_id", groupID, "paused", paused, "exclusive", exclusive)
	m.notify()
	return m.db.GetAllGroups(), nil
}

// SaveManagedAsGroup snapshots the current managed cameras as a new group
func (m *GoProManager) SaveManagedAsGroup(name string) (*model.Group, error) {
	managed := m.db.GetCamerasForManagedPool()
	ids := make([]string, len(managed))
	for i, cam := range managed {
		ids[i] = cam.CameraState.Camera.ID
	}
	return m.CreateGroup(name, ids)
}

func (m *GoProManager) validateCamerasExist(cameraIDs []string) error {
	for _, cameraID := range cameraIDs {
		if _, found := m.db.GetCameraByID(cameraID); !found {
			return fmt.Errorf("%w: %s", model.ErrCameraNotFound, cameraID)
		}
	}
	return nil
}
