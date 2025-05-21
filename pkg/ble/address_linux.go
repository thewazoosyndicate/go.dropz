package ble

import (
	"tinygo.org/x/bluetooth"
)

// createAddress creates a bluetooth.Address from a MAC address string
// On Linux, this directly places the MAC into the Address struct
func createAddress(mac bluetooth.MAC) (bluetooth.Address, error) {
	// On Linux, MACAddress is embedded directly in Address
	addr := bluetooth.Address{bluetooth.MACAddress{MAC: mac}}
	return addr, nil
}
