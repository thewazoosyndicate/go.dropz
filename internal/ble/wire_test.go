package ble

import (
	"bytes"
	"testing"

	"github.com/dropz/dropz/internal/ble/tlv"
)

// Golden wire-format tests against the OpenGoPro spec
// (docs/ble/protocol/data_protocol and id_tables).

func TestKeepAliveWireFormat(t *testing.T) {
	// Spec: set setting 91 (0x5B) to 66 (0x42): header + 5B 01 42
	got := tlv.BuildCommandPacket(SettingKeepAlive, []byte{0x01, KeepAliveValue})
	want := []byte{0x20, 0x03, 0x5B, 0x01, 0x42} // ext-13 header, len 3
	if !bytes.Equal(got, want) {
		t.Errorf("keep-alive packet = % X, want % X", got, want)
	}
}

func TestSleepWireFormat(t *testing.T) {
	got := tlv.BuildCommandPacket(CmdSleep, nil)
	want := []byte{0x20, 0x01, 0x05}
	if !bytes.Equal(got, want) {
		t.Errorf("sleep packet = % X, want % X", got, want)
	}
}

func TestPairingFinishWireFormat(t *testing.T) {
	// RequestPairingFinish: field 1 varint SUCCESS(0), field 2 string "dropz"
	phoneName := []byte("dropz")
	payload := append([]byte{0x08, 0x00, 0x12, byte(len(phoneName))}, phoneName...)
	data := append([]byte{0x01}, payload...) // ActionID SET_PAIRING_STATE

	got := buildProtobufPacket(0x03, data) // FeatureID WIRELESS_MANAGEMENT

	// Message = feature + action + proto = 1 + 1 + 9 = 11 bytes,
	// framed with an ext-13 header like every other message.
	want := append([]byte{0x20, 0x0B, 0x03, 0x01}, payload...)
	if !bytes.Equal(got, want) {
		t.Errorf("pairing finish packet = % X, want % X", got, want)
	}
}

func TestQueryStatusesWireFormat(t *testing.T) {
	got := tlv.BuildQueryPacket(QueryGetStatus, []byte{
		StatusBatteryPercentage, StatusNumTotalPhotos,
	})
	want := []byte{0x20, 0x03, 0x13, 70, 38}
	if !bytes.Equal(got, want) {
		t.Errorf("query packet = % X, want % X", got, want)
	}
}

func TestContinuationCounterWireFormat(t *testing.T) {
	// 40-byte message: 18 bytes in start packet, then continuation
	// packets with counters starting at 0x0 (spec: "start at 0x0,
	// reset after 0xF").
	msg := make([]byte, 40)
	stream := tlv.BuildTLVPackets(msg)
	packets := tlv.SplitIntoPackets(stream)

	if len(packets) != 3 {
		t.Fatalf("got %d packets, want 3", len(packets))
	}
	if packets[0][0] != 0x20 || packets[0][1] != 40 {
		t.Errorf("start header = % X, want 20 28", packets[0][:2])
	}
	if packets[1][0] != 0x80 {
		t.Errorf("first continuation header = %02X, want 80", packets[1][0])
	}
	if packets[2][0] != 0x81 {
		t.Errorf("second continuation header = %02X, want 81", packets[2][0])
	}
}
