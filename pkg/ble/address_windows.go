package ble

import (
	"tinygo.org/x/bluetooth"
)

// createAddress creates a bluetooth.Address from a MAC address string
// Windows uses the same MAC-based address structure as Linux
func createAddress(mac bluetooth.MAC) (bluetooth.Address, error) {
	addr := bluetooth.Address{
		MACAddress: bluetooth.MACAddress{MAC: mac},
	}
	return addr, nil
}
