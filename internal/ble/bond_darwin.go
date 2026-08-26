//go:build darwin

package ble

import "strings"

// CoreBluetooth surfaces a bond the camera refuses as a plain connect
// timeout (tinygo darwin: "timeout on Connect"). Counted the same as the
// Linux abort: the threshold plus the still-advertising window in
// connectFailure keeps ordinary out-of-range timeouts from qualifying.
func isConnectAbort(err error) bool {
	return strings.Contains(err.Error(), "timeout")
}

// ForgetDevice is a no-op on macOS: CoreBluetooth owns bonds and offers no
// programmatic removal; the user unpairs in system settings.
func (m *Manager) ForgetDevice(macAddress string) error {
	m.resetConnectAborts(macAddress)
	return nil
}
