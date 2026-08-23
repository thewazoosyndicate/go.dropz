package ble

import "testing"

// Real capture from the official Kotlin SDK test vectors
// (kmp_sdk TestBleAdvertisementParsing): schema 2, processor awake only,
// camera id 65, hashed (non-ASCII) id_hash, serial tail "0053".
func TestParseAdvKotlinSdkVector(t *testing.T) {
	manuf := []byte{0x02, 0x01, 0x41, 0x23, 0x00, 0x99, 0x64, 0x26, 0x61, 0x21, 0xE0, 0x0F}
	info, ok := parseAdvManufacturerData(manuf)
	if !ok {
		t.Fatal("expected valid parse")
	}
	if !info.ProcessorOn || info.WiFiAPOn || info.PairingMode || info.NewMedia {
		t.Errorf("status flags wrong: %+v", info)
	}
	if info.ModelID != 65 || info.SchemaVersion != 2 {
		t.Errorf("model/schema wrong: %+v", info)
	}

	tail, ok := parseAdvServiceData([]byte{0x47, 0x3B, 0x28, 0x2D, '0', '0', '5', '3'})
	if !ok || tail != "0053" {
		t.Fatalf("tail = %q, %v", tail, ok)
	}
	// id_hash is not ASCII: the middle is unknowable, so no serial may be
	// fabricated (a partial one would never match the GATT-read serial).
	if got := assembleSerial(info, tail); got != "" {
		t.Errorf("serial = %q, want empty for hashed id_hash", got)
	}
}

func TestParseAdvSchemaV2AsciiHash(t *testing.T) {
	// Some v2 cameras broadcast the ASCII serial middle in the id_hash slot.
	manuf := []byte{0x02, advStatusPairingMode, 62, 0, 0, '1', '3', '2', '4', '5', '0', 0}
	info, _ := parseAdvManufacturerData(manuf)
	if !info.PairingMode || info.ProcessorOn || info.NewMedia {
		t.Errorf("status flags wrong: %+v", info)
	}
	if got := assembleSerial(info, "0711"); got != "C3501324500711" {
		t.Errorf("serial = %q, want C3501324500711", got)
	}
}

func TestParseAdvSchemaV3(t *testing.T) {
	manuf := []byte{0x03, advStatusProcessorOn | advStatusNewMedia, 70, 0xC3, 0x00,
		0xDE, 0xAD, 0xBE, 0xEF, '2', '4', 0x0F}
	info, _ := parseAdvManufacturerData(manuf)
	if !info.NewMedia || !info.ProcessorOn || info.PairingMode {
		t.Errorf("status flags wrong: %+v", info)
	}
	tail, ok := parseAdvServiceData([]byte{1, 2, 3, 4, '5', '3', '0', '0', '7', '1', '1', '2'})
	if !ok || tail != "53007112" {
		t.Fatalf("tail = %q, %v", tail, ok)
	}
	if got := assembleSerial(info, tail); got != "C3592453007112" {
		t.Errorf("serial = %q, want C3592453007112", got)
	}
}

func TestParseAdvRejects(t *testing.T) {
	if _, ok := parseAdvManufacturerData([]byte{0x02, 0x00}); ok {
		t.Error("short payload must not parse")
	}
	if _, ok := parseAdvServiceData([]byte{1, 2, 3, 4}); ok {
		t.Error("mac-only payload must not parse")
	}
	if _, ok := parseAdvServiceData([]byte{1, 2, 3, 4, 0xFF, 0xFE, 0x00, 0x01}); ok {
		t.Error("binary payload must not parse as serial tail")
	}
	// unknown model id: no prefix, no serial
	info, _ := parseAdvManufacturerData([]byte{0x03, 0, 99, 0, 0, 0, 0, 0, 0, '2', '4', 0})
	if got := assembleSerial(info, "53007112"); got != "" {
		t.Errorf("serial = %q, want empty for unknown model", got)
	}
	// cross-schema mismatch: v3 mid with v2-length tail cannot form 14 chars
	info, _ = parseAdvManufacturerData([]byte{0x03, 0, 70, 0, 0, 0, 0, 0, 0, '2', '4', 0})
	if got := assembleSerial(info, "0711"); got != "" {
		t.Errorf("serial = %q, want empty for length mismatch", got)
	}
}
