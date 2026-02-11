//go:build darwin

package manager

func isTransientBLEError(_ error) bool {
	return false
}
