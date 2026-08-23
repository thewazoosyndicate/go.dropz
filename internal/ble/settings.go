package ble

import (
	"fmt"
)

// Query IDs for settings (OpenGoPro query table).
const (
	QueryGetSettings            = 0x12
	QueryGetSettingCapabilities = 0x32
)

// parseTLVMulti parses [ID][Length][Value...] triplets keeping every
// occurrence: capability responses repeat the setting ID once per
// supported option, which parseTLVPairs would collapse.
func parseTLVMulti(data []byte) map[byte][][]byte {
	result := make(map[byte][][]byte)
	offset := 0
	for offset+2 <= len(data) {
		id := data[offset]
		length := int(data[offset+1])
		offset += 2
		if offset+length > len(data) {
			break
		}
		value := make([]byte, length)
		copy(value, data[offset:offset+length])
		result[id] = append(result[id], value)
		offset += length
	}
	return result
}

// settingBytesToInt64 decodes a big-endian setting value of any length 1..8.
func settingBytesToInt64(v []byte) (int64, bool) {
	if len(v) == 0 || len(v) > 8 {
		return 0, false
	}
	var out int64
	for _, b := range v {
		out = out<<8 | int64(b)
	}
	return out, true
}

// settingInt64ToBytes encodes a value in the wire format of the setting.
// Unknown settings default to 1 byte, the format of 56 of the 59 settings.
func settingInt64ToBytes(id byte, v int64) []byte {
	size := 1
	if def, ok := SettingDefs[id]; ok && def.Format == "int64ub" {
		size = 8
	}
	out := make([]byte, size)
	for i := size - 1; i >= 0; i-- {
		out[i] = byte(v)
		v >>= 8
	}
	return out
}

// GetSettingValues queries current values (query 0x12).
// Empty ids means all settings.
func (m *Manager) GetSettingValues(macAddress string, ids []byte) (map[byte]int64, error) {
	response, err := m.sendQuery(macAddress, QueryGetSettings, ids)
	if err != nil {
		return nil, fmt.Errorf("get setting values: %w", err)
	}
	values := make(map[byte]int64)
	for id, raw := range parseTLVPairs(response.Data) {
		if v, ok := settingBytesToInt64(raw); ok {
			values[id] = v
		}
	}
	return values, nil
}

// GetSettingCapabilities queries the options the camera supports right now
// (query 0x32). This is the authority on per-model and per-state support:
// never gate on a static table. Empty ids means all settings.
func (m *Manager) GetSettingCapabilities(macAddress string, ids []byte) (map[byte][]int64, error) {
	response, err := m.sendQuery(macAddress, QueryGetSettingCapabilities, ids)
	if err != nil {
		return nil, fmt.Errorf("get setting capabilities: %w", err)
	}
	caps := make(map[byte][]int64)
	for id, raws := range parseTLVMulti(response.Data) {
		for _, raw := range raws {
			if v, ok := settingBytesToInt64(raw); ok {
				caps[id] = append(caps[id], v)
			}
		}
	}
	return caps, nil
}

// SetSetting writes one setting value (TLV on GP-0074).
// Status 2 means the camera rejected the option in its current state.
func (m *Manager) SetSetting(macAddress string, id byte, value int64) error {
	raw := settingInt64ToBytes(id, value)
	params := append([]byte{byte(len(raw))}, raw...)
	resp, err := m.sendSetting(macAddress, id, params)
	if err != nil {
		return fmt.Errorf("set setting %d: %w", id, err)
	}
	switch resp.Status {
	case 0:
		return nil
	case 2:
		return fmt.Errorf("set setting %d: option %d not available in the camera's current state", id, value)
	default:
		return fmt.Errorf("set setting %d: camera returned status %d", id, resp.Status)
	}
}
