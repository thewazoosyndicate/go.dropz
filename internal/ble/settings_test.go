package ble

import (
	"reflect"
	"testing"
)

func TestParseTLVMultiKeepsDuplicates(t *testing.T) {
	// Capability response shape: setting 2 offered with options 9, 12; setting 3 with 0.
	data := []byte{2, 1, 9, 2, 1, 12, 3, 1, 0}
	got := parseTLVMulti(data)
	want := map[byte][][]byte{
		2: {{9}, {12}},
		3: {{0}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseTLVMulti = %v, want %v", got, want)
	}
}

func TestSettingValueRoundTrip(t *testing.T) {
	if v, ok := settingBytesToInt64([]byte{0x01, 0x02}); !ok || v != 0x0102 {
		t.Errorf("decode = %d, %v", v, ok)
	}
	if _, ok := settingBytesToInt64(nil); ok {
		t.Error("empty value must not decode")
	}

	// setting 2 (Video Resolution) is int8ub
	if got := settingInt64ToBytes(2, 9); !reflect.DeepEqual(got, []byte{9}) {
		t.Errorf("int8ub encode = %v", got)
	}
	// setting 30 (Photo Timelapse Rate) is int64ub
	got := settingInt64ToBytes(30, 3600)
	if len(got) != 8 {
		t.Fatalf("int64ub length = %d", len(got))
	}
	if v, _ := settingBytesToInt64(got); v != 3600 {
		t.Errorf("int64ub round trip = %d", v)
	}
	// unknown setting defaults to 1 byte
	if got := settingInt64ToBytes(200, 5); !reflect.DeepEqual(got, []byte{5}) {
		t.Errorf("unknown encode = %v", got)
	}
}

func TestSettingDefsTable(t *testing.T) {
	res, ok := SettingDefs[2]
	if !ok || res.Name != "Video Resolution" {
		t.Fatalf("setting 2 = %+v", res)
	}
	if res.Options[9] != "1080" {
		t.Errorf("option 9 = %q, want 1080", res.Options[9])
	}
	for id, def := range SettingDefs {
		if def.Format != "int8ub" && def.Format != "int64ub" {
			t.Errorf("setting %d has unexpected format %q", id, def.Format)
		}
	}
}
