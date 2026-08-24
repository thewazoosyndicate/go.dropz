package ble

import (
	"context"
	"encoding/binary"
)

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

// ParseIntStatus decodes a big-endian TLV status value.
// Unexpected lengths return -1 (invalid) so callers skip the value instead
// of acting on a fake zero.
func ParseIntStatus(v []byte) int32 {
	switch len(v) {
	case 1:
		return int32(v[0])
	case 2:
		return int32(binary.BigEndian.Uint16(v))
	case 4:
		return int32(binary.BigEndian.Uint32(v))
	default:
		return -1
	}
}

// ParseInt64Status decodes a big-endian TLV status value up to 8 bytes.
func ParseInt64Status(v []byte) int64 {
	switch len(v) {
	case 1:
		return int64(v[0])
	case 2:
		return int64(binary.BigEndian.Uint16(v))
	case 4:
		return int64(binary.BigEndian.Uint32(v))
	case 8:
		return int64(binary.BigEndian.Uint64(v))
	default:
		return -1
	}
}
