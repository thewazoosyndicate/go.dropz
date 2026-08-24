//go:build darwin

package ble

// IsTransient reports transient errors worth retrying; none known on macOS.
func IsTransient(_ error) bool {
	return false
}
