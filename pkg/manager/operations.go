package manager

import (
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/manager/discovery"
	syncpkg "github.com/dropz/dropz/pkg/manager/sync"
)

// ForceSync adds a camera to the sync queue for immediate synchronization
func (m *GoProManager) ForceSync(cameraID string) (*database.SyncQueueEntry, error) {
	syncEntry, err := m.syncCoordinator.ForceSync(cameraID, m.notifier)
	if err != nil {
		return nil, err
	}

	// Trigger immediate sync processing by sending signal to immediateSyncTrigger channel
	select {
	case m.immediateSyncTrigger <- struct{}{}:
		m.log.Debug("Immediate sync trigger sent for manual sync request")
	default:
		m.log.Debug("Immediate sync trigger channel full, sync will be picked up by next cycle")
	}

	return syncEntry, nil
}

// CancelSync cancels an ongoing sync operation and removes the camera from the sync queue
func (m *GoProManager) CancelSync(cameraID string) error {
	return m.syncCoordinator.CancelSync(cameraID, m.activeSyncTasks, &m.mutex, m.notifier)
}

// performCameraSync handles the actual syncing of a camera
func (m *GoProManager) performCameraSync(task *syncpkg.SyncTask) {
	// Delegate to sync coordinator
	m.syncCoordinator.PerformCameraSync(task, m.activeSyncTasks, &m.mutex, m.notifier, m.BLEOperation, m.ctx)
}

// GetVideosByCamera returns videos for a specific camera
func (m *GoProManager) GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*database.VideoFile, int) {
	return m.syncCoordinator.GetVideosByCamera(cameraID, startDate, endDate, limit, offset)
}

// startBackgroundScanner starts the background scanner
func (m *GoProManager) startBackgroundScanner() {
	defer m.wg.Done()
	// Delegate to discovery scanner with live processing
	config := m.db.GetConfig()
	scanInterval := time.Duration(config.ScanIntervalSeconds) * time.Second
	scanner := discovery.NewScanner(m.ble, m.log, scanInterval)
	scanner.StartBackgroundScanner(m.ctx, m.processDiscoveredDeviceLive)
}

// processDiscoveredDeviceLive delegates to discovery processor for live processing
func (m *GoProManager) processDiscoveredDeviceLive(device ble.Device) {
	m.discoveryProcessor.ProcessDiscoveredDeviceLive(device, m.notifier)
}

