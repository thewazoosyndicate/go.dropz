//go:build darwin

package ble

import (
	"fmt"
	"strings"

	"tinygo.org/x/bluetooth"
)

// createAddress creates a bluetooth.Address from a MAC address string or UUID
// On macOS, we need to handle two cases:
// 1. When we have a real UUID (during pairing/connection)
// 2. When we have a MAC address (during discovery)
func createAddress(mac bluetooth.MAC) (bluetooth.Address, error) {
	var addr bluetooth.Address

	// First check if the MAC address string is actually a UUID
	macStr := mac.String()
	if strings.Contains(macStr, "-") && len(macStr) == 36 {
		// This is already a UUID, parse it directly
		uuid, err := bluetooth.ParseUUID(macStr)
		if err != nil {
			return addr, fmt.Errorf("failed to parse UUID: %v", err)
		}
		addr.UUID = uuid
		return addr, nil
	}

	// Otherwise, create a UUID from the MAC address
	uuidStr := "00000000-0000-0000-0000-" + strings.ReplaceAll(macStr, ":", "")
	uuid, err := bluetooth.ParseUUID(uuidStr)
	if err != nil {
		return addr, fmt.Errorf("failed to create UUID from MAC: %v", err)
	}
	addr.UUID = uuid

	return addr, nil
}

// parseAddress handles either a UUID string or a MAC address string on macOS.
// If a UUID is provided, it is parsed directly. If a MAC is provided, it is
// converted into a UUID-based Address.
func parseAddress(addrStr string) (bluetooth.Address, error) {
	// Check if addrStr looks like a UUID
	if strings.Contains(addrStr, "-") && len(addrStr) == 36 {
		uuid, err := bluetooth.ParseUUID(addrStr)
		if err != nil {
			return bluetooth.Address{}, fmt.Errorf("failed to parse UUID: %v", err)
		}
		return bluetooth.Address{UUID: uuid}, nil
	}

	mac, err := bluetooth.ParseMAC(addrStr)
	if err != nil {
		return bluetooth.Address{}, err
	}
	return createAddress(mac)
}
