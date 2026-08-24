//go:build darwin

package ble

// No known link-abort signature on CoreBluetooth; bond-loss detection is
// Linux-only until observed on macOS.
func isConnectAbort(_ error) bool {
	return false
}

// ForgetDevice is a no-op on macOS: CoreBluetooth owns bonds and offers no
// programmatic removal; the user unpairs in system settings.
func (m *Manager) ForgetDevice(macAddress string) error {
	m.resetConnectAborts(macAddress)
	return nil
}
