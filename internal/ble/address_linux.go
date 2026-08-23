//go:build linux

package ble

import (
	"fmt"

	"tinygo.org/x/bluetooth"
)

func parseAddress(addr string) (bluetooth.Address, error) {
	mac, err := bluetooth.ParseMAC(addr)
	if err != nil {
		return bluetooth.Address{}, fmt.Errorf("failed to parse MAC: %w", err)
	}
	return bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}, nil
}

func validateAddress(addr string) error {
	if len(addr) != 17 {
		return fmt.Errorf("invalid MAC address length")
	}
	return nil
}
