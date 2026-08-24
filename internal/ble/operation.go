package ble

import "context"

// Operation runs a BLE operation under the owning manager's retry policy.
// Shared by the sync and pairing packages so both retry the same way.
type Operation func(ctx context.Context, op func() error) error

// InUse reports whether busy (8) or encoding (10) statuses are nonzero
// (OpenGoPro state_management: gate work on statuses 8 and 10).
func InUse(statuses map[byte][]byte) bool {
	for _, id := range []byte{StatusSystemBusy, StatusEncoding} {
		if v, ok := statuses[id]; ok && len(v) >= 1 && v[0] != 0 {
			return true
		}
	}
	return false
}
