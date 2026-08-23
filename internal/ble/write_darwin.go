//go:build darwin

package ble

import "tinygo.org/x/bluetooth"

// writeCharacteristic uses write-with-response: CoreBluetooth silently
// drops WriteWithoutResponse packets under load (tmp FIXES.md, Fix 5).
func writeCharacteristic(char bluetooth.DeviceCharacteristic, p []byte) (int, error) {
	return char.Write(p)
}
