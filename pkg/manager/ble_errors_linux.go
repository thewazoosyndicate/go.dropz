//go:build linux

package manager

import "strings"

// isTransientBLEError checks for BlueZ/DBus-specific transient errors worth retrying
func isTransientBLEError(err error) bool {
	return strings.Contains(err.Error(), "Properties.GetAll")
}
