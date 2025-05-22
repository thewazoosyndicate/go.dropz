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
