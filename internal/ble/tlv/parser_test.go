package tlv

import (
	"bytes"
	"testing"
	"time"
)

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expected    *PacketHeader
		expectError bool
	}{
		{
			name:        "Empty packet",
			data:        []byte{},
			expected:    nil,
			expectError: true,
		},
		{
			name: "General header (5-bit)",
			data: []byte{0x05}, // 0000 0101 - General type, 5 byte message
			expected: &PacketHeader{
				IsStart:        true,
				IsContinuation: false,
				PacketType:     HeaderTypeGeneral,
				MessageLength:  5,
				HeaderSize:     1,
			},
			expectError: false,
		},
		{
			name: "Extended 13-bit header",
			data: []byte{0x21, 0x00}, // 0010 0001, 0000 0000 - Extended13, 256 byte message
			expected: &PacketHeader{
				IsStart:        true,
				IsContinuation: false,
				PacketType:     HeaderTypeExtended13,
				MessageLength:  256,
				HeaderSize:     2,
			},
			expectError: false,
		},
		{
			name: "Extended 16-bit header",
			data: []byte{0x40, 0x10, 0x00}, // 0100 0000, 0001 0000, 0000 0000 - Extended16, 4096 byte message
			expected: &PacketHeader{
				IsStart:        true,
				IsContinuation: false,
				PacketType:     HeaderTypeExtended16,
				MessageLength:  4096,
				HeaderSize:     3,
			},
			expectError: false,
		},
		{
			name: "Continuation packet",
			data: []byte{0x85}, // 1000 0101 - Continuation, packet counter 5
			expected: &PacketHeader{
				IsStart:        false,
				IsContinuation: true,
				PacketCounter:  5,
				HeaderSize:     1,
			},
			expectError: false,
		},
		{
			name: "Maximum general header length",
			data: []byte{0x1F}, // 0001 1111 - General type, 31 byte message (max for 5-bit)
			expected: &PacketHeader{
				IsStart:        true,
				IsContinuation: false,
				PacketType:     HeaderTypeGeneral,
				MessageLength:  31,
				HeaderSize:     1,
			},
			expectError: false,
		},
		{
			name: "Maximum continuation counter",
			data: []byte{0x8F}, // 1000 1111 - Continuation, packet counter 15 (max)
			expected: &PacketHeader{
				IsStart:        false,
				IsContinuation: true,
				PacketCounter:  15,
				HeaderSize:     1,
			},
			expectError: false,
		},
		{
			name:        "Extended 13-bit insufficient data",
			data:        []byte{0x20}, // Extended13 but missing second byte
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Extended 16-bit insufficient data",
			data:        []byte{0x40, 0x00}, // Extended16 but missing third byte
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := ParseHeader(tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if header == nil {
				t.Errorf("Expected header but got nil")
				return
			}

			// Compare fields
			if header.IsStart != tt.expected.IsStart {
				t.Errorf("IsStart: got %v, want %v", header.IsStart, tt.expected.IsStart)
			}
			if header.IsContinuation != tt.expected.IsContinuation {
				t.Errorf("IsContinuation: got %v, want %v", header.IsContinuation, tt.expected.IsContinuation)
			}
			if header.IsStart && header.PacketType != tt.expected.PacketType {
				t.Errorf("PacketType: got %v, want %v", header.PacketType, tt.expected.PacketType)
			}
			if header.IsStart && header.MessageLength != tt.expected.MessageLength {
				t.Errorf("MessageLength: got %d, want %d", header.MessageLength, tt.expected.MessageLength)
			}
			if header.IsContinuation && header.PacketCounter != tt.expected.PacketCounter {
				t.Errorf("PacketCounter: got %d, want %d", header.PacketCounter, tt.expected.PacketCounter)
			}
			if header.HeaderSize != tt.expected.HeaderSize {
				t.Errorf("HeaderSize: got %d, want %d", header.HeaderSize, tt.expected.HeaderSize)
			}
		})
	}
}

func TestValidatePacketSequence(t *testing.T) {
	tests := []struct {
		name        string
		expected    int
		received    int
		expectError bool
	}{
		{
			name:        "Valid sequence",
			expected:    0,
			received:    1,
			expectError: false,
		},
		{
			name:        "Valid sequence mid-range",
			expected:    5,
			received:    6,
			expectError: false,
		},
		{
			name:        "Valid wrap from 15 to 0",
			expected:    15,
			received:    0,
			expectError: false,
		},
		{
			name:        "Invalid sequence - skip",
			expected:    0,
			received:    2,
			expectError: true,
		},
		{
			name:        "Invalid sequence - duplicate",
			expected:    5,
			received:    5,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePacketSequence(tt.expected, tt.received)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestParseTLVResponse(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expected    *TLVMessage
		expectError bool
	}{
		{
			name: "Valid response with payload",
			data: []byte{0x01, 0x00, 0x48, 0x65, 0x6C, 0x6C, 0x6F}, // Command 1, Status 0, "Hello"
			expected: &TLVMessage{
				CommandID: 0x01,
				Status:    0x00,
				Payload:   []byte("Hello"),
			},
			expectError: false,
		},
		{
			name: "Valid response without payload",
			data: []byte{0x05, 0x00}, // Command 5, Status 0, no payload
			expected: &TLVMessage{
				CommandID: 0x05,
				Status:    0x00,
				Payload:   []byte{},
			},
			expectError: false,
		},
		{
			name:        "Too short - only command",
			data:        []byte{0x01},
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Empty data",
			data:        []byte{},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseTLVResponse(tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if msg.CommandID != tt.expected.CommandID {
				t.Errorf("CommandID: got %02X, want %02X", msg.CommandID, tt.expected.CommandID)
			}
			if msg.Status != tt.expected.Status {
				t.Errorf("Status: got %02X, want %02X", msg.Status, tt.expected.Status)
			}
			if !bytes.Equal(msg.Payload, tt.expected.Payload) {
				t.Errorf("Payload: got %v, want %v", msg.Payload, tt.expected.Payload)
			}
		})
	}
}

func TestFragmentCollector(t *testing.T) {
	fc := NewFragmentCollector(1 * time.Second)

	t.Run("Single packet message", func(t *testing.T) {
		// Extended 13-bit header: length=4, then Command 0x01, Status 0x00, Payload "Hi"
		packet := []byte{0x20, 0x04, 0x01, 0x00, 0x48, 0x69} // Header + 4 bytes of message

		msg, err := fc.ProcessFragment(packet)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if msg == nil {
			t.Fatalf("Expected message but got nil")
		}
		if msg.CommandID != 0x01 {
			t.Errorf("CommandID: got %02X, want 0x01", msg.CommandID)
		}
		if msg.Status != 0x00 {
			t.Errorf("Status: got %02X, want 0x00", msg.Status)
		}
		if string(msg.Payload) != "Hi" {
			t.Errorf("Payload: got %s, want Hi", string(msg.Payload))
		}
	})

	t.Run("Multi-packet message", func(t *testing.T) {
		// Message: Command 0x02, Status 0x00, Payload: 20 bytes of 'A'
		// Total message length: 22 bytes (1 cmd + 1 status + 20 payload)

		// First packet: Extended 13-bit header for 22 bytes, then first 18 bytes of message
		packet1 := []byte{0x20, 0x16}                               // Header: Extended13, length = 22
		packet1 = append(packet1, 0x02, 0x00)                       // Command and Status
		packet1 = append(packet1, bytes.Repeat([]byte{'A'}, 16)...) // 16 bytes of payload (18 bytes total after header)

		// Second packet: Continuation with counter 0, remaining 4 bytes
		packet2 := []byte{0x80}                                    // Continuation header, counter = 0
		packet2 = append(packet2, bytes.Repeat([]byte{'A'}, 4)...) // Remaining 4 bytes of payload

		// Process first packet
		msg, err := fc.ProcessFragment(packet1)
		if err != nil {
			t.Fatalf("Error processing packet 1: %v", err)
		}
		if msg != nil {
			t.Fatalf("Expected nil message after first packet, got %v", msg)
		}

		// Process second packet
		msg, err = fc.ProcessFragment(packet2)
		if err != nil {
			t.Fatalf("Error processing packet 2: %v", err)
		}
		if msg == nil {
			t.Fatalf("Expected message after second packet")
		}
		if msg.CommandID != 0x02 {
			t.Errorf("CommandID: got %02X, want 0x02", msg.CommandID)
		}
		if msg.Status != 0x00 {
			t.Errorf("Status: got %02X, want 0x00", msg.Status)
		}
		if len(msg.Payload) != 20 {
			t.Errorf("Payload length: got %d, want 20", len(msg.Payload))
		}
	})

	t.Run("Out of order packets", func(t *testing.T) {
		// Try to send continuation packet without start
		contPacket := []byte{0x81, 0x00, 0x00} // Continuation with counter 1

		msg, err := fc.ProcessFragment(contPacket)
		if err == nil {
			t.Errorf("Expected error for out of order packet")
		}
		if msg != nil {
			t.Errorf("Expected nil message for out of order packet")
		}
	})
}
