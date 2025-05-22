//go:build windows

package ble

import (
	"tinygo.org/x/bluetooth"
)

// createAddress creates a bluetooth.Address from a MAC address string
// Windows uses the same MAC-based address structure as Linux
func createAddress(mac bluetooth.MAC) (bluetooth.Address, error) {
	addr := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}
	return addr, nil
}

// parseAddress parses a MAC address string into a bluetooth.Address on Windows.
func parseAddress(addrStr string) (bluetooth.Address, error) {
	mac, err := bluetooth.ParseMAC(addrStr)
	if err != nil {
		return bluetooth.Address{}, err
	}

	return createAddress(mac)
}
