package groups

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/logger"
	"github.com/google/uuid"
)

// Manager handles group operations
type Manager struct {
	db  *database.Database
	log logger.Logger
}

// NewManager creates a new group manager
func NewManager(db *database.Database, log logger.Logger) *Manager {
	return &Manager{
		db:  db,
		log: log,
	}
}

// CreateGroup creates a new group with the specified cameras
func (gm *Manager) CreateGroup(name string, cameraIDs []string) (*database.Group, error) {
	// Validate cameras exist
	if err := gm.validateCameras(cameraIDs); err != nil {
		return nil, fmt.Errorf("camera validation failed: %v", err)
	}

	// Create a new group
	group := &database.Group{
		ID:        uuid.New().String(),
		Name:      name,
		CameraIDs: cameraIDs,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add to database
	if err := gm.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to create group: %v", err)
	}

	// Update camera records to associate with this group
	gm.updateCameraGroupAssociations(cameraIDs, group.ID)

	// Save changes
	if err := gm.db.SaveChanges(); err != nil {
		gm.log.Warnf("Failed to update camera group associations: %v", err)
	}

	gm.log.Infof("Created group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// UpdateGroup updates an existing group
func (gm *Manager) UpdateGroup(groupID, name string, cameraIDs []string) (*database.Group, error) {
	// Get the existing group
	group, exists := gm.db.GetGroup(groupID)
	if !exists {
		return nil, fmt.Errorf("group with ID %s not found", groupID)
	}

	// Validate new cameras exist
	if err := gm.validateCameras(cameraIDs); err != nil {
		return nil, fmt.Errorf("camera validation failed: %v", err)
	}

	// Get the original camera IDs for comparison
	originalCameraIDs := make(map[string]bool)
	for _, cameraID := range group.CameraIDs {
		originalCameraIDs[cameraID] = true
	}

	// Update group details
	group.Name = name
	group.CameraIDs = cameraIDs
	group.UpdatedAt = time.Now()

	// Add to database
	if err := gm.db.AddOrUpdateGroup(group); err != nil {
		return nil, fmt.Errorf("failed to update group: %v", err)
	}

	// Update camera records that are newly added to this group
	for _, cameraID := range cameraIDs {
		if !originalCameraIDs[cameraID] {
			// This is a newly added camera
			gm.updateCameraGroupAssociation(cameraID, group.ID)
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
			gm.removeCameraGroupAssociation(cameraID, groupID)
		}
	}

	// Save changes
	if err := gm.db.SaveChanges(); err != nil {
		gm.log.Warnf("Failed to update camera group associations: %v", err)
	}

	gm.log.Infof("Updated group %s with %d cameras", name, len(cameraIDs))
	return group, nil
}

// DeleteGroup deletes a group
func (gm *Manager) DeleteGroup(groupID string) error {
	// Get the existing group
	group, exists := gm.db.GetGroup(groupID)
	if !exists {
		return fmt.Errorf("group with ID %s not found", groupID)
	}

	// Remove the group association from all cameras in the group
	for _, cameraID := range group.CameraIDs {
		gm.removeCameraGroupAssociation(cameraID, groupID)
	}

	// Remove the group from the database
	if err := gm.db.RemoveGroup(groupID); err != nil {
		return fmt.Errorf("failed to delete group: %v", err)
	}

	// Save changes
	if err := gm.db.SaveChanges(); err != nil {
		gm.log.Warnf("Failed to update camera group associations: %v", err)
	}

	gm.log.Infof("Deleted group %s", group.Name)
	return nil
}
