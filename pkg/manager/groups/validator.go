package groups

import (
	"fmt"

	"github.com/dropz/dropz/pkg/database"
)

// validateCameras checks if all cameras in the list exist
func (gm *Manager) validateCameras(cameraIDs []string) error {
	for _, cameraID := range cameraIDs {
		found := false
		for _, state := range gm.db.CameraStates {
			if state.Camera.ID == cameraID {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("camera %s not found", cameraID)
		}
	}

	return nil
}

// updateCameraGroupAssociations updates multiple cameras to associate with a group
func (gm *Manager) updateCameraGroupAssociations(cameraIDs []string, groupID string) {
	for _, cameraID := range cameraIDs {
		gm.updateCameraGroupAssociation(cameraID, groupID)
	}
}

// updateCameraGroupAssociation updates a single camera to associate with a group
func (gm *Manager) updateCameraGroupAssociation(cameraID, groupID string) {
	found := false
	for _, state := range gm.db.CameraStates {
		if state.Camera.ID == cameraID {
			state.GroupID = groupID
			found = true
			break
		}
	}

	if !found {
		gm.log.Warnf("Camera %s not found when adding to group", cameraID)
	}
}

// removeCameraGroupAssociation removes a camera's association with a group
func (gm *Manager) removeCameraGroupAssociation(cameraID, groupID string) {
	found := false
	for _, state := range gm.db.CameraStates {
		if state.Camera.ID == cameraID && state.GroupID == groupID {
			state.GroupID = ""
			found = true
			break
		}
	}

	if !found {
		gm.log.Warnf("Camera %s not found when removing from group", cameraID)
	}
}

// ValidateGroupName checks if a group name is valid
func (gm *Manager) ValidateGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("group name cannot exceed 100 characters")
	}

	return nil
}

// GetGroupCameras returns all cameras in a group
func (gm *Manager) GetGroupCameras(groupID string) ([]string, error) {
	group, exists := gm.db.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("group with ID %s not found", groupID)
	}

	return group.CameraIDs, nil
}

// GetCameraGroups returns all groups that contain a specific camera
func (gm *Manager) GetCameraGroups(cameraID string) ([]*database.Group, error) {
	var groups []*database.Group

	allGroups := gm.db.GetAllGroups()
	for _, group := range allGroups {
		for _, id := range group.CameraIDs {
			if id == cameraID {
				groups = append(groups, group)
				break
			}
		}
	}

	return groups, nil
}

// IsValidGroupOperation checks if a group operation is valid
func (gm *Manager) IsValidGroupOperation(groupID string, operation string) error {
	group, exists := gm.db.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group with ID %s not found", groupID)
	}

	switch operation {
	case "sync":
		// Check if all cameras in the group are paired and reachable
		for _, cameraID := range group.CameraIDs {
			for _, state := range gm.db.CameraStates {
				if state.Camera.ID == cameraID {
					if !state.Status.IsPaired {
						return fmt.Errorf("camera %s in group is not paired", state.Camera.Name)
					}
					if !state.Status.IsReachable {
						return fmt.Errorf("camera %s in group is not reachable", state.Camera.Name)
					}
					break
				}
			}
		}
	case "delete":
		// Any group can be deleted
		return nil
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return nil
}
