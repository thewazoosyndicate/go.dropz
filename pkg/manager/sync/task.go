package sync

import (
	"time"

	"github.com/dropz/dropz/pkg/database"
)

// TaskManager handles individual sync task operations
type TaskManager struct {
	db *database.Database
}

// NewTaskManager creates a new task manager
func NewTaskManager(db *database.Database) *TaskManager {
	return &TaskManager{
		db: db,
	}
}

// CreateSyncTask creates a new sync task
func (tm *TaskManager) CreateSyncTask(cameraID, macAddress, cameraName string) *SyncTask {
	return &SyncTask{
		CameraID:   cameraID,
		MACAddress: macAddress,
		CameraName: cameraName,
		StartedAt:  time.Now(),
	}
}

// CompleteSyncTask marks a sync task as completed
func (tm *TaskManager) CompleteSyncTask(task *SyncTask, success bool, errorMessage string) {
	task.CompletedAt = time.Now()
	task.Success = success
	task.ErrorMessage = errorMessage
}

// ValidateSyncTask checks if a sync task is ready to be executed
func (tm *TaskManager) ValidateSyncTask(cameraID string) (bool, string) {
	// Get the camera details
	camera, found := tm.db.GetManagedCameraByID(cameraID)
	if !found {
		return false, "camera not found"
	}

	// Check if the camera is suitable for syncing
	if !camera.CameraState.Status.IsReachable {
		return false, "camera is not reachable"
	}

	if camera.CameraState.Status.IsSyncing {
		return false, "camera is already syncing"
	}

	if !camera.CameraState.Status.IsPaired {
		return false, "camera is not paired"
	}

	return true, ""
}

// GetTaskStatus returns the current status of a sync task
func (tm *TaskManager) GetTaskStatus(task *SyncTask) string {
	if task.CompletedAt.IsZero() {
		return "Running"
	}

	if task.Success {
		return "Completed"
	}

	return "Failed"
}

// GetTaskDuration returns the duration of a sync task
func (tm *TaskManager) GetTaskDuration(task *SyncTask) time.Duration {
	if task.CompletedAt.IsZero() {
		return time.Since(task.StartedAt)
	}

	return task.CompletedAt.Sub(task.StartedAt)
}
