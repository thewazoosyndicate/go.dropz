//go:build linux

package ble

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

// isConnectAbort reports BlueZ's link-level connect abort, the signature of
// encrypting with a bond the camera no longer holds.
func isConnectAbort(err error) bool {
	return strings.Contains(err.Error(), "le-connection-abort-by-local")
}

// ForgetDevice removes the BlueZ device entry, discarding the stored bond.
// Recovery for a camera that dropped its side: reconnecting with the stale
// LTK aborts forever until the host forgets it too.
func (m *Manager) ForgetDevice(macAddress string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system D-Bus: %w", err)
	}
	devPath := "/org/bluez/hci0/dev_" + strings.ReplaceAll(macAddress, ":", "_")
	adapter := conn.Object("org.bluez", "/org/bluez/hci0")
	if call := adapter.Call("org.bluez.Adapter1.RemoveDevice", 0, dbus.ObjectPath(devPath)); call.Err != nil {
		return fmt.Errorf("failed to remove device: %w", call.Err)
	}
	m.resetConnectAborts(macAddress)
	return nil
}
