package ble

import (
	"strings"

	"tinygo.org/x/bluetooth"
)

// GoPro's Bluetooth SIG company identifier (0x02F2, little-endian on the wire).
const goProCompanyID = 0x02F2

// AdvInfo holds fields parsed from GoPro advertising data.
// Layout per OpenGoPro ble_setup (manufacturer data, schema v2/v3):
// [schema][camera_status][camera_id][capabilities x2][id_hash x6][media_offload]
type AdvInfo struct {
	Valid         bool
	SchemaVersion byte
	ProcessorOn   bool
	WiFiAPOn      bool
	PairingMode   bool // camera shows the pairing UI and accepts bonding
	NewMedia      bool // camera advertises unsynced media
	ModelID       int  // camera_id byte; same numbering as GetHardwareInfo
	SerialNumber  string
}

// Camera status bit positions (MSB first per spec BitStruct).
const (
	advStatusProcessorOn = 0x80
	advStatusWiFiAPOn    = 0x40
	advStatusPairingMode = 0x20
	advStatusNewMedia    = 0x08
)

// parseAdvManufacturerData parses the GoPro manufacturer data payload.
func parseAdvManufacturerData(data []byte) (AdvInfo, bool) {
	if len(data) < 3 {
		return AdvInfo{}, false
	}
	status := data[1]
	return AdvInfo{
		Valid:         true,
		SchemaVersion: data[0],
		ProcessorOn:   status&advStatusProcessorOn != 0,
		WiFiAPOn:      status&advStatusWiFiAPOn != 0,
		PairingMode:   status&advStatusPairingMode != 0,
		NewMedia:      status&advStatusNewMedia != 0,
		ModelID:       int(data[2]),
	}, true
}

// parseAdvServiceData extracts the serial number from the FEA6 service data:
// [ap_mac x4][serial utf8]. The serial identifies a camera across the random
// BLE addresses macOS assigns.
func parseAdvServiceData(data []byte) (string, bool) {
	if len(data) <= 4 {
		return "", false
	}
	serial := strings.TrimRight(string(data[4:]), "\x00")
	if len(serial) < 8 {
		return "", false
	}
	for _, r := range serial {
		isDigit := r >= '0' && r <= '9'
		isUpper := r >= 'A' && r <= 'Z'
		isLower := r >= 'a' && r <= 'z'
		if !isDigit && !isUpper && !isLower {
			return "", false
		}
	}
	return serial, true
}

// parseAdvertisement extracts GoPro fields from a scan result.
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
			if serial, ok := parseAdvServiceData(sd.Data); ok {
				info.SerialNumber = serial
			}
			break
		}
	}
	return info
}
