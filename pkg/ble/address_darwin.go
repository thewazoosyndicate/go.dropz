//go:build darwin

package ble

import (
	"fmt"
	"strings"

	"tinygo.org/x/bluetooth"
)

// createAddress creates a bluetooth.Address from a MAC address string or UUID
func createAddress(addrStr string) (bluetooth.Address, error) {
	var addr bluetooth.Address

	// If the address already looks like a UUID, parse it directly
	if strings.Contains(addrStr, "-") && len(addrStr) == 36 {
		uuid, err := bluetooth.ParseUUID(addrStr)
		if err != nil {
			return addr, fmt.Errorf("failed to parse UUID: %w", err)
		}
		addr.UUID = uuid
		return addr, nil
	}

	// Otherwise create a pseudo-UUID from the MAC
	uuidStr := "00000000-0000-0000-0000-" + strings.ReplaceAll(addrStr, ":", "")
	uuid, err := bluetooth.ParseUUID(uuidStr)
	if err != nil {
		return addr, fmt.Errorf("failed to create UUID from MAC: %w", err)
	}
	addr.UUID = uuid
	return addr, nil
}

// parseAddress parses either a MAC address or a UUID string into a bluetooth.Address.
func parseAddress(addrStr string) (bluetooth.Address, error) {
	// If the address looks like a UUID, parse directly
	if strings.Contains(addrStr, "-") && len(addrStr) == 36 {
		uuid, err := bluetooth.ParseUUID(addrStr)
		if err != nil {
			return bluetooth.Address{}, fmt.Errorf("failed to parse UUID: %w", err)
		}
		return bluetooth.Address{UUID: uuid}, nil
	}

	mac, err := bluetooth.ParseMAC(addrStr)
	if err != nil {
		return bluetooth.Address{}, err
	}
	return createAddress(mac)
}
