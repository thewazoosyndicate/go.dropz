package sync

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
)

// QueueManager handles sync queue operations
type QueueManager struct {
	db  *database.Database
	log *logrus.Logger
}

// NewQueueManager creates a new queue manager
func NewQueueManager(db *database.Database, log *logrus.Logger) *QueueManager {
	return &QueueManager{
		db:  db,
		log: log,
	}
}

// AddToQueue adds a camera to the sync queue
func (qm *QueueManager) AddToQueue(cameraID string) (*database.SyncQueueEntry, error) {
	// Check if camera is already in the queue
	for _, entry := range qm.db.GetSyncQueue() {
		if entry.CameraID == cameraID {
			qm.log.Tracef("Camera %s is already in the sync queue", cameraID)
			return entry, nil
		}
	}

	// Create a new sync queue entry
	syncEntry := &database.SyncQueueEntry{
		CameraID:         cameraID,
		QueuedAt:         time.Now(),
		ProgressPercent:  0,
		CurrentOperation: "Waiting to start",
	}

	// Add to sync queue
	if err := qm.db.AddSyncQueueEntry(syncEntry); err != nil {
		return nil, fmt.Errorf("failed to add camera to sync queue: %v", err)
	}

	// Database already logs this operation, no need to duplicate
	return syncEntry, nil
}

// RemoveFromQueue removes a camera from the sync queue
func (qm *QueueManager) RemoveFromQueue(cameraID string) error {
	qm.db.RemoveSyncQueueEntry(cameraID)
	// Database already logs this operation, no need to duplicate
	return nil
}

// UpdateQueueEntry updates a sync queue entry
func (qm *QueueManager) UpdateQueueEntry(entry *database.SyncQueueEntry) error {
	return qm.db.UpdateSyncQueueEntry(entry)
}

// GetQueuePosition returns the position of a camera in the sync queue
func (qm *QueueManager) GetQueuePosition(cameraID string) (int, bool) {
	queue := qm.db.GetSyncQueue()
	for i, entry := range queue {
		if entry.CameraID == cameraID {
			return i + 1, true // 1-based position
		}
	}
	return 0, false
}

// GetQueueLength returns the total number of items in the sync queue
func (qm *QueueManager) GetQueueLength() int {
	return len(qm.db.GetSyncQueue())
}

// GetNextInQueue returns the next camera to sync from the queue
func (qm *QueueManager) GetNextInQueue() (*database.SyncQueueEntry, bool) {
	queue := qm.db.GetSyncQueue()
	if len(queue) == 0 {
		return nil, false
	}

	// Return the oldest entry (FIFO)
	return queue[0], true
}

// ClearQueue removes all entries from the sync queue
func (qm *QueueManager) ClearQueue() error {
	queue := qm.db.GetSyncQueue()
	for _, entry := range queue {
		qm.db.RemoveSyncQueueEntry(entry.CameraID)
	}
	qm.log.Info("Cleared sync queue")
	return nil
}

// GetQueueStats returns statistics about the sync queue
func (qm *QueueManager) GetQueueStats() map[string]interface{} {
	queue := qm.db.GetSyncQueue()

	stats := map[string]interface{}{
		"total_items": len(queue),
		"waiting":     0,
		"processing":  0,
	}

	for _, entry := range queue {
		if entry.ProgressPercent == 0 {
			stats["waiting"] = stats["waiting"].(int) + 1
		} else if entry.ProgressPercent < 100 {
			stats["processing"] = stats["processing"].(int) + 1
		}
	}

	return stats
}
