package manager

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/database"
	"github.com/google/uuid"
)

// CreateGroup creates a new group with the specified cameras
func (m *GoProManager) CreateGroup(name string, cameraIDs []string) (*database.Group, error) {
	// Create a new group
	group := &database.Group{
		ID:        uuid.New().String(),
		Name:      name,
		CameraIDs: cameraIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add to database
	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create group: %v", err)
	}

	// Update camera records to associate with this group
	for _, cameraID := range cameraIDs {
		// Find the camera in database
		found := false
		for _, state := range m.db.CameraStates {
			if state.Camera.ID == cameraID {
				state.GroupID = group.ID
				found = true
				break
			}
		}

		if !found {
			m.log.Warnf("Camera %s not found when adding to group", cameraID)
		}
	}

	// Save changes
	if err := m.db.SaveChanges(); err != nil {
		m.log.Warnf("Failed to update camera group associations: %v", err)
	}

	m.log.Infof("Created group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// UpdateGroup updates an existing group
func (m *GoProManager) UpdateGroup(groupID, name string, cameraIDs []string) (*database.Group, error) {
	// Get the existing group
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("group with ID %s not found", groupID)
	}

	// Update group details
	group.Name = name

	// Get the original camera IDs for comparison
	originalCameraIDs := make(map[string]bool)
	for _, cameraID := range group.CameraIDs {
		originalCameraIDs[cameraID] = true
	}

	// Update camera list
	group.CameraIDs = cameraIDs
	group.UpdatedAt = time.Now()

	// Add to database
	if err := m.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to update group: %v", err)
	}

	// Update camera records that are newly added to this group
	for _, cameraID := range cameraIDs {
		if !originalCameraIDs[cameraID] {
			// This is a newly added camera
			found := false
			for _, state := range m.db.CameraStates {
				if state.Camera.ID == cameraID {
					state.GroupID = group.ID
					found = true
					break
				}
			}

			if !found {
				m.log.Warnf("Camera %s not found when adding to group", cameraID)
			}
		}
	}

	// Remove the group association from cameras that were removed from the group
	for cameraID := range originalCameraIDs {
		stillInGroup := false
		for _, id := range cameraIDs {
			if id == cameraID {
				stillInGroup = true
				break
			}
		}

		if !stillInGroup {
			// This camera was removed from the group
			found := false
			for _, state := range m.db.CameraStates {
				if state.Camera.ID == cameraID && state.GroupID == groupID {
					state.GroupID = ""
					found = true
					break
				}
			}

			if !found {
				m.log.Warnf("Camera %s not found when removing from group", cameraID)
			}
		}
	}

	// Save changes
	if err := m.db.SaveChanges(); err != nil {
		m.log.Warnf("Failed to update camera group associations: %v", err)
	}

	m.log.Infof("Updated group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// DeleteGroup deletes a group
func (m *GoProManager) DeleteGroup(groupID string) error {
	// Get the existing group
	group, exists := m.db.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group with ID %s not found", groupID)
	}

	// Remove the group association from all cameras in the group
	for _, cameraID := range group.CameraIDs {
		found := false
		for _, state := range m.db.CameraStates {
			if state.Camera.ID == cameraID && state.GroupID == groupID {
				state.GroupID = ""
				found = true
				break
			}
		}

		if !found {
			m.log.Warnf("Camera %s not found when removing from group", cameraID)
		}
	}

	// Remove the group from the database
	if err := m.db.RemoveGroup(groupID); err != nil {
		return fmt.Errorf("failed to delete group: %v", err)
	}

	// Save changes
	if err := m.db.SaveChanges(); err != nil {
		m.log.Warnf("Failed to update camera group associations: %v", err)
	}

	m.log.Infof("Deleted group %s", group.Name)
	return nil
}

// GetAllVideos returns all videos
func (m *GoProManager) GetAllVideos() []*database.VideoFile {
	return m.db.GetAllVideos()
}
