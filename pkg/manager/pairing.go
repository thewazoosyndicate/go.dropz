package manager

import (
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/manager/pairing"
)

// PairCamera pairs with a GoPro camera using BLE
func (m *GoProManager) PairCamera(cameraID string) (*database.ManagedCamera, error) {
	if m.notifier != nil {
		// Recreate the pairing manager with the current notifier
		m.pairingManager = pairing.NewManager(m.db, m.ble, m.log, m.notifier, m.ctx)
	}
	return m.pairingManager.PairCamera(cameraID, m.BLEOperation)
}

// syncDevicePairingStates checks actual device pairing states and updates database accordingly
func (m *GoProManager) syncDevicePairingStates() {
	if m.notifier != nil {
		// Recreate the pairing manager with the current notifier
		m.pairingManager = pairing.NewManager(m.db, m.ble, m.log, m.notifier, m.ctx)
	}
	m.pairingManager.SyncDevicePairingStates()
}

// VerifyAndFixPairingState checks a specific camera's pairing state and fixes database if needed
func (m *GoProManager) VerifyAndFixPairingState(cameraID string) error {
	if m.notifier != nil {
		// Recreate the pairing manager with the current notifier
		m.pairingManager = pairing.NewManager(m.db, m.ble, m.log, m.notifier, m.ctx)
	}
	return m.pairingManager.VerifyAndFixPairingState(cameraID)
}
