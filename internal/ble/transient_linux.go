//go:build linux

package ble

import "strings"

// IsTransient checks for BlueZ/DBus-specific transient errors worth retrying.
func IsTransient(err error) bool {
	return strings.Contains(err.Error(), "Properties.GetAll")
}
