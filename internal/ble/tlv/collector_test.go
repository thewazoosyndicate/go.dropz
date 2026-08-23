package tlv

import (
	"bytes"
	"testing"
	"time"
)

func packetize(stream []byte) [][]byte {
	var packets [][]byte
	for off := 0; off < len(stream); off += 20 {
		end := off + 20
		if end > len(stream) {
			end = len(stream)
		}
		packets = append(packets, stream[off:end])
	}
	return packets
}

func longMessage(t *testing.T) []byte {
	t.Helper()
	// 500 bytes: 27 packets, the 4-bit continuation counter wraps past 0xF.
	// Sized like a real all-settings capability response (query 0x32).
	msg := make([]byte, 500)
	msg[0] = 0x32
	msg[1] = 0x00
	for i := 2; i < len(msg); i++ {
		msg[i] = byte(i * 7)
	}
	return msg
}

// Counter-wrap regression: packets were stored keyed by the 4-bit counter,
// so packet 16 overwrote packet 0 and long responses never assembled.
func TestCollectorReassemblesWrappingMessage(t *testing.T) {
	msg := longMessage(t)
	fc := NewFragmentCollector(time.Second)
	defer fc.Stop()

	var got *TLVMessage
	for i, pkt := range packetize(BuildTLVPackets(msg)) {
		m, err := fc.ProcessFragment(pkt)
		if err != nil {
			t.Fatalf("packet %d: %v", i, err)
		}
		if m != nil {
			got = m
		}
	}
	if got == nil {
		t.Fatal("message never assembled")
	}
	if got.CommandID != 0x32 || got.Status != 0 {
		t.Errorf("id=0x%02X status=%d", got.CommandID, got.Status)
	}
	if !bytes.Equal(got.Payload, msg[2:]) {
		t.Error("payload corrupted after counter wrap")
	}
}

// Duplicate delivery (stale notification watcher): each duplicate must be
// rejected without damaging the original stream.
func TestCollectorSurvivesDuplicatedPackets(t *testing.T) {
	msg := longMessage(t)
	fc := NewFragmentCollector(time.Second)
	defer fc.Stop()

	var got *TLVMessage
	for _, pkt := range packetize(BuildTLVPackets(msg)) {
		for pass := 0; pass < 2; pass++ {
			m, err := fc.ProcessFragment(pkt)
			if pass == 0 && err != nil {
				t.Fatalf("original packet rejected: %v", err)
			}
			if m != nil {
				got = m
			}
		}
	}
	if got == nil {
		t.Fatal("message never assembled with duplicated stream")
	}
	if !bytes.Equal(got.Payload, msg[2:]) {
		t.Error("payload corrupted by duplicates")
	}
}
