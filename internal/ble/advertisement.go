package ble

import (
	"tinygo.org/x/bluetooth"
)

// GoPro's Bluetooth SIG company identifier (0x02F2, little-endian on the wire).
const goProCompanyID = 0x02F2

// modelSerialPrefix maps the advertised model ID to the first 4 serial chars,
// which are never broadcast. Source: spec references.json (Supported Cameras);
// the official SDK maps match but lack 69.
var modelSerialPrefix = map[int]string{
	55: "C344", // HERO9 Black
	57: "C346", // HERO10 Black
	58: "C347", // HERO11 Black
	60: "C349", // HERO11 Black Mini
	62: "C350", // HERO12 Black
	64: "C352", // MAX 2
	65: "C353", // HERO13 Black
	69: "C357", // Mission 1 Pro
	70: "C359", // LIT HERO
}

// AdvInfo holds fields parsed from GoPro advertising data.
// Manufacturer payload (12 bytes, spec ble_setup):
//
//	v2: [schema][status][model_id][caps x2][id_hash x6][offload]
//	v3: [schema][status][model_id][caps][rsvd][variant x4][serial 4-5][offload]
//
// FEA6 service data: [ap_mac x4] + serial tail (last 4 chars in v2, last 8 in v3).
type AdvInfo struct {
	Valid         bool
	SchemaVersion byte
	ProcessorOn   bool
	WiFiAPOn      bool
	PairingMode   bool // camera shows the pairing UI and accepts bonding
	NewMedia      bool // camera advertises unsynced media
	ModelID       int  // camera_id byte; same numbering as GetHardwareInfo
	SerialNumber  string

	serialMid string // v3: serial chars 4-5; v2: id_hash when it is ASCII
}

// Camera status bits, LSB first: spec says "Bit 0: Processor State" and the
// Kotlin SDK's real-capture test vector confirms bit 0 is the LSB.
// The Python SDK's BitStruct reads this byte MSB first; that parser is unused
// upstream and wrong. Do not copy it.
const (
	advStatusProcessorOn = 0x01
	advStatusWiFiAPOn    = 0x02
	advStatusPairingMode = 0x04
	advStatusNewMedia    = 0x10
)

const advManufPayloadLen = 12

func isSerialChar(r byte) bool {
	return (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func allSerialChars(b []byte) bool {
	for _, r := range b {
		if !isSerialChar(r) {
			return false
		}
	}
	return len(b) > 0
}

// parseAdvManufacturerData parses the GoPro manufacturer data payload.
func parseAdvManufacturerData(data []byte) (AdvInfo, bool) {
	if len(data) < 3 {
		return AdvInfo{}, false
	}
	status := data[1]
	info := AdvInfo{
		Valid:         true,
		SchemaVersion: data[0],
		ProcessorOn:   status&advStatusProcessorOn != 0,
		WiFiAPOn:      status&advStatusWiFiAPOn != 0,
		PairingMode:   status&advStatusPairingMode != 0,
		NewMedia:      status&advStatusNewMedia != 0,
		ModelID:       int(data[2]),
	}
	if len(data) < advManufPayloadLen {
		return info, true
	}
	switch {
	case info.SchemaVersion >= 3:
		if mid := data[9:11]; allSerialChars(mid) {
			info.serialMid = string(mid)
		}
	case info.SchemaVersion == 2:
		// v2 broadcasts a hash here; on some cameras it is the ASCII middle
		// of the serial. Only usable when it looks like one.
		if mid := data[5:11]; allSerialChars(mid) {
			info.serialMid = string(mid)
		}
	}
	return info, true
}

// parseAdvServiceData extracts the serial tail from the FEA6 service data:
// last 4 chars (v2, 8-byte payload) or last 8 chars (v3, 12-byte payload).
func parseAdvServiceData(data []byte) (string, bool) {
	if len(data) != 8 && len(data) != 12 {
		return "", false
	}
	tail := data[4:]
	if !allSerialChars(tail) {
		return "", false
	}
	return string(tail), true
}

// assembleSerial rebuilds the full 14-char serial from its broadcast parts.
// Empty when any part is missing: a partial serial must never be stored, it
// would be matched against the full serial read over GATT.
func assembleSerial(info AdvInfo, tail string) string {
	prefix, known := modelSerialPrefix[info.ModelID]
	if !known || info.serialMid == "" || tail == "" {
		return ""
	}
	if len(prefix)+len(info.serialMid)+len(tail) != 14 {
		return ""
	}
	return prefix + info.serialMid + tail
}

// parseAdvertisement extracts GoPro fields from a scan result.
// Serial requires manufacturer and service data in the same result; BlueZ and
// CoreBluetooth both aggregate scan responses, so this is the common case.
func parseAdvertisement(result bluetooth.ScanResult, goProServiceUUID bluetooth.UUID) AdvInfo {
	var info AdvInfo
	for _, md := range result.ManufacturerData() {
		if md.CompanyID == goProCompanyID {
			if parsed, ok := parseAdvManufacturerData(md.Data); ok {
				info = parsed
			}
			break
		}
	}
	for _, sd := range result.ServiceData() {
		if sd.UUID == goProServiceUUID {
			if tail, ok := parseAdvServiceData(sd.Data); ok {
				info.SerialNumber = assembleSerial(info, tail)
			}
			break
		}
	}
	return info
}
