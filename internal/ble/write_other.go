//go:build !darwin

package ble

import "tinygo.org/x/bluetooth"

// writeCharacteristic sends a packet without response: reliable on BlueZ,
// and the only write primitive tinygo exposes on Linux.
func writeCharacteristic(char bluetooth.DeviceCharacteristic, p []byte) (int, error) {
	return char.WriteWithoutResponse(p)
}
