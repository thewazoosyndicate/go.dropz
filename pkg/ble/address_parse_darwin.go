//go:build darwin

package ble

import (
	"fmt"
	"strings"

	"tinygo.org/x/bluetooth"
)

// parseAddress converts a MAC or UUID string into a bluetooth.Address.
func parseAddress(addrStr string) (bluetooth.Address, error) {
	var addr bluetooth.Address

	if strings.Contains(addrStr, "-") && len(addrStr) == 36 {
		uuid, err := bluetooth.ParseUUID(addrStr)
		if err != nil {
			return addr, fmt.Errorf("failed to parse UUID: %v", err)
		}
		addr.UUID = uuid
		return addr, nil
	}

	mac, err := bluetooth.ParseMAC(addrStr)
	if err != nil {
		return addr, fmt.Errorf("failed to parse MAC: %v", err)
	}

	return createAddress(mac)
}
