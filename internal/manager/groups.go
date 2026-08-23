package manager

import (
	"fmt"
	"time"

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

	m.log.Info("Created group", "group", name, "cameras", len(cameraIDs))
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

	m.log.Info("Updated group", "group", name, "cameras", len(cameraIDs))
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

	m.log.Info("Deleted group", "group", group.Name)
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
			m.ManageCamera(id)
		} else if !inGroup && cam.Status.IsManaged {
			m.UnmanageCamera(id)
		}
	}

	m.log.Info("Loaded group", "group", group.Name, "cameras", len(group.CameraIDs))
	m.notify()
	return nil
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
