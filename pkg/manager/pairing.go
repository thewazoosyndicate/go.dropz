package manager

import (
	"github.com/dropz/dropz/pkg/database"
)

// PairCamera pairs with a GoPro camera using BLE
func (m *GoProManager) PairCamera(cameraID string) (*database.ManagedCamera, error) {
	return m.pairCameraDelegate(cameraID)
}

// syncDevicePairingStates checks actual device pairing states and updates database accordingly
func (m *GoProManager) syncDevicePairingStates() {
	m.syncDevicePairingStatesDelegate()
}

// VerifyAndFixPairingState checks a specific camera's pairing state and fixes database if needed
func (m *GoProManager) VerifyAndFixPairingState(cameraID string) error {
	return m.verifyAndFixPairingStateDelegate(cameraID)
}
