//go:build darwin

package ble

import (
	"fmt"

	"tinygo.org/x/bluetooth"
)

func parseAddress(addr string) (bluetooth.Address, error) {
	if _, err := bluetooth.ParseUUID(addr); err != nil {
		return bluetooth.Address{}, fmt.Errorf("invalid BLE UUID: %v", err)
	}
	var a bluetooth.Address
	a.Set(addr)
	return a, nil
}

func validateAddress(addr string) error {
	if len(addr) != 36 {
		return fmt.Errorf("invalid UUID address length")
	}
	return nil
}
