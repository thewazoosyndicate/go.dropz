package manager

import (
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/manager/discovery"
	"github.com/dropz/dropz/pkg/manager/pairing"
	syncpkg "github.com/dropz/dropz/pkg/manager/sync"
)

// processSyncQueue checks the sync queue and processes cameras that need syncing
func (m *GoProManager) processSyncQueue() {
	// Delegate to sync coordinator
	m.syncCoordinator.ProcessSyncQueue(m.activeSyncTasks, &m.mutex, m.notifier, m.performCameraSync)
}

// ForceSync adds a camera to the sync queue for immediate synchronization
func (m *GoProManager) ForceSync(cameraID string) (*database.SyncQueueEntry, error) {
	return m.syncCoordinator.ForceSync(cameraID, m.notifier)
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
	// Delegate to discovery scanner
	scanner := discovery.NewScanner(m.ble, m.log, m.scanInterval)
	scanner.StartBackgroundScanner(m.ctx, m.processDiscoveredDevices)
}

// processDiscoveredDevices delegates to discovery processor
func (m *GoProManager) processDiscoveredDevices(devices []ble.Device) {
	m.mutex.RLock()
	notifier := m.notifier
	m.mutex.RUnlock()

	if m.discoveryProcessor != nil {
		m.discoveryProcessor.ProcessDiscoveredDevices(devices, notifier)
	}
}

// ===== Pairing Operations =====

// updatePairingManagerNotifier updates the notifier in the pairing manager
func (m *GoProManager) updatePairingManagerNotifier() {
	if m.notifier != nil {
		// Recreate the pairing manager with the current notifier
		m.pairingManager = pairing.NewManager(m.db, m.ble, m.log, m.notifier, m.ctx)
	}
}

// PairCamera pairs with a GoPro camera using BLE (delegated)
func (m *GoProManager) pairCameraDelegate(cameraID string) (*database.ManagedCamera, error) {
	m.updatePairingManagerNotifier()
	return m.pairingManager.PairCamera(cameraID, m.BLEOperation)
}

// syncDevicePairingStates checks actual device pairing states and updates database accordingly (delegated)
func (m *GoProManager) syncDevicePairingStatesDelegate() {
	m.updatePairingManagerNotifier()
	m.pairingManager.SyncDevicePairingStates()
}

// VerifyAndFixPairingState checks a specific camera's pairing state and fixes database if needed (delegated)
func (m *GoProManager) verifyAndFixPairingStateDelegate(cameraID string) error {
	m.updatePairingManagerNotifier()
	return m.pairingManager.VerifyAndFixPairingState(cameraID)
}
