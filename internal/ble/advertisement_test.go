package ble

import "testing"

func TestParseAdvManufacturerData(t *testing.T) {
	// schema 2, status: processor on + pairing mode, camera_id 64 (HERO13)
	data := []byte{0x02, advStatusProcessorOn | advStatusPairingMode, 64, 0, 0, 1, 2, 3, 4, 5, 6, 0}
	info, ok := parseAdvManufacturerData(data)
	if !ok {
		t.Fatal("expected valid parse")
	}
	if !info.ProcessorOn || !info.PairingMode || info.WiFiAPOn || info.NewMedia {
		t.Errorf("status flags wrong: %+v", info)
	}
	if info.ModelID != 64 {
		t.Errorf("ModelID = %d, want 64", info.ModelID)
	}

	// new media flag alone
	info, _ = parseAdvManufacturerData([]byte{0x03, advStatusNewMedia, 62})
	if !info.NewMedia || info.PairingMode {
		t.Errorf("new media flags wrong: %+v", info)
	}

	// too short
	if _, ok := parseAdvManufacturerData([]byte{0x02, 0x00}); ok {
		t.Error("short payload must not parse")
	}
}

func TestParseAdvServiceData(t *testing.T) {
	// 4 bytes AP MAC + serial
	data := append([]byte{0xAA, 0xBB, 0xCC, 0xDD}, []byte("C3501324500711")...)
	serial, ok := parseAdvServiceData(data)
	if !ok || serial != "C3501324500711" {
		t.Errorf("serial = %q, %v", serial, ok)
	}

	// too short
	if _, ok := parseAdvServiceData([]byte{1, 2, 3, 4}); ok {
		t.Error("mac-only payload must not parse")
	}
	// junk bytes are not a serial
	if _, ok := parseAdvServiceData(append([]byte{1, 2, 3, 4}, 0xFF, 0xFE, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06)); ok {
		t.Error("binary payload must not parse as serial")
	}
}
