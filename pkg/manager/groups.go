package manager

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/database"
	"github.com/google/uuid"
)

// CreateGroup creates a new group with the specified cameras
func (m *GoProManager) CreateGroup(name string, cameraIDs []string) (*database.Group, error) {
	if err := m.validateCamerasExist(cameraIDs); err != nil {
		return nil, err
	}

	group := &database.Group{
		ID:        uuid.New().String(),
		Name:      name,
		CameraIDs: cameraIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create group: %v", err)
	}

	for _, cameraID := range cameraIDs {
		m.setCameraGroup(cameraID, group.ID)
	}

	m.log.Infof("Created group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// UpdateGroup updates an existing group
func (m *GoProManager) UpdateGroup(groupID, name string, cameraIDs []string) (*database.Group, error) {
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("group with ID %s not found", groupID)
	}

	if err := m.validateCamerasExist(cameraIDs); err != nil {
		return nil, err
	}

	oldCameraIDs := make(map[string]bool)
	for _, id := range group.CameraIDs {
		oldCameraIDs[id] = true
	}

	group.Name = name
	group.CameraIDs = cameraIDs
	group.UpdatedAt = time.Now()

	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to update group: %v", err)
	}

	// Add new cameras to group
	for _, id := range cameraIDs {
		if !oldCameraIDs[id] {
			m.setCameraGroup(id, group.ID)
		}
	}

	// Remove cameras no longer in group
	newCameraIDs := make(map[string]bool)
	for _, id := range cameraIDs {
		newCameraIDs[id] = true
	}
	for id := range oldCameraIDs {
		if !newCameraIDs[id] {
			m.setCameraGroup(id, "")
		}
	}

	m.log.Infof("Updated group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// DeleteGroup deletes a group
func (m *GoProManager) DeleteGroup(groupID string) error {
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group with ID %s not found", groupID)
	}

	for _, cameraID := range group.CameraIDs {
		m.setCameraGroup(cameraID, "")
	}

	if err := m.db.RemoveGroup(groupID); err != nil {
		return fmt.Errorf("failed to delete group: %v", err)
	}

	m.log.Infof("Deleted group %s", group.Name)
	return nil
}

func (m *GoProManager) validateCamerasExist(cameraIDs []string) error {
	for _, cameraID := range cameraIDs {
		if _, found := m.db.GetCameraByID(cameraID); !found {
			return fmt.Errorf("camera %s not found", cameraID)
		}
	}
	return nil
}

func (m *GoProManager) setCameraGroup(cameraID, groupID string) {
	m.db.UpdateCameraByID(cameraID, func(cs *database.CameraWithState) {
		cs.GroupID = groupID
	})
}
