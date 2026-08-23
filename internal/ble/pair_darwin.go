//go:build darwin

package ble

// pairViaDbus is a no-op on macOS — CoreBluetooth handles pairing transparently.
func (m *Manager) pairViaDbus(macAddress string) error {
	return nil
}
