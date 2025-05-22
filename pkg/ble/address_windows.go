//go:build windows

package ble

import (
	"tinygo.org/x/bluetooth"
)

// createAddress creates a bluetooth.Address from a MAC address string
// Windows uses the same MAC-based address structure as Linux
func createAddress(addrStr string) (bluetooth.Address, error) {
	mac, err := bluetooth.ParseMAC(addrStr)
	if err != nil {
		return bluetooth.Address{}, err
	}
	addr := bluetooth.Address{bluetooth.MACAddress{MAC: mac}}
	return addr, nil
}
