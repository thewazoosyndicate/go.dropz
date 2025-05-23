//go:build windows

package ble

import (
	"tinygo.org/x/bluetooth"
)

// parseAddress converts a MAC string into a bluetooth.Address.
func parseAddress(addrStr string) (bluetooth.Address, error) {
	mac, err := bluetooth.ParseMAC(addrStr)
	if err != nil {
		return bluetooth.Address{}, err
	}
	return createAddress(mac)
}
