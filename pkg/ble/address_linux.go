//go:build linux

package ble

import (
	"tinygo.org/x/bluetooth"
)

// createAddress creates a bluetooth.Address from a MAC address string
// On Linux, this directly places the MAC into the Address struct
func createAddress(mac bluetooth.MAC) (bluetooth.Address, error) {
	// On Linux, MACAddress is embedded directly in Address
	addr := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}

	return addr, nil
}

// parseAddress parses a MAC address string into a bluetooth.Address on Linux.
func parseAddress(addrStr string) (bluetooth.Address, error) {
	mac, err := bluetooth.ParseMAC(addrStr)
	if err != nil {
		return bluetooth.Address{}, err
	}
	return createAddress(mac)
}
