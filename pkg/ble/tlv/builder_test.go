package tlv

import (
	"bytes"
	"testing"
)

func TestBuildCommandPacket(t *testing.T) {
	tests := []struct {
		name      string
		commandID byte
		params    []byte
		expected  []byte // Expected packet(s)
	}{
		{
			name:      "Command without parameters",
			commandID: 0x01,
			params:    []byte{},
			expected:  []byte{0x20, 0x01, 0x01}, // Extended13 header (length=1), command ID
		},
		{
			name:      "Command with small parameters",
			commandID: 0x05,
			params:    []byte{0x10, 0x20},
			expected:  []byte{0x20, 0x03, 0x05, 0x10, 0x20}, // Extended13 header (length=3), command + params
		},
		{
			name:      "Command with 17 bytes of parameters (fits in single packet)",
			commandID: 0x0A,
			params:    bytes.Repeat([]byte{0xFF}, 17),
			expected: append([]byte{0x20, 0x12, 0x0A}, // Extended13 header (length=18), command
				bytes.Repeat([]byte{0xFF}, 17)...), // 17 bytes of params
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCommandPacket(tt.commandID, tt.params)
			
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("BuildCommandPacket() = %X, want %X", result, tt.expected)
			}
		})
	}
}

func TestBuildTLVPackets_MultiPacket(t *testing.T) {
	// Test message that requires multiple packets
	// 30 bytes of data should split into 2 packets
	data := bytes.Repeat([]byte{0xAA}, 30)
	
	result := BuildTLVPackets(data)
	
	// First packet: Extended13 header + 18 bytes of data
	expectedFirst := []byte{0x20, 0x1E} // Extended13, length=30
	expectedFirst = append(expectedFirst, data[:18]...)
	
	// Second packet: Continuation header + remaining 12 bytes
	expectedSecond := []byte{0x80} // Continuation, counter=0
	expectedSecond = append(expectedSecond, data[18:]...)
	
	expected := append(expectedFirst, expectedSecond...)
	
	if !bytes.Equal(result, expected) {
		t.Errorf("BuildTLVPackets() multi-packet mismatch\nGot:  %X\nWant: %X", result, expected)
	}
}

func TestBuildTLVPackets_LargeMessage(t *testing.T) {
	// Test with message requiring many continuation packets
	// 100 bytes should split into: 1 start (18 bytes) + 5 continuation packets
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	
	result := BuildTLVPackets(data)
	
	// Verify the header
	if result[0] != 0x20 || result[1] != 0x64 { // Extended13, length=100
		t.Errorf("Invalid header: got %X %X, want 20 64", result[0], result[1])
	}
	
	// Count packets
	packetCount := 0
	offset := 0
	for offset < len(result) {
		if offset == 0 {
			// First packet: 2-byte header + 18 bytes data
			offset += 20
		} else {
			// Continuation packet: 1-byte header + 19 bytes data
			if result[offset]&0x80 == 0 {
				t.Errorf("Expected continuation bit at offset %d", offset)
			}
			expectedCounter := (packetCount - 1) & 0x0F
			if result[offset]&0x0F != byte(expectedCounter) {
				t.Errorf("Wrong counter at offset %d: got %d, want %d", 
					offset, result[offset]&0x0F, expectedCounter)
			}
			offset += 20
		}
		packetCount++
	}
	
	// Should have 6 packets total (1 start + 5 continuations)
	if packetCount != 6 {
		t.Errorf("Wrong packet count: got %d, want 6", packetCount)
	}
}

func TestSplitIntoPackets(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected [][]byte
	}{
		{
			name: "Single packet",
			data: []byte{0x01, 0x02, 0x03},
			expected: [][]byte{
				{0x01, 0x02, 0x03},
			},
		},
		{
			name: "Exactly 20 bytes",
			data: bytes.Repeat([]byte{0xAB}, 20),
			expected: [][]byte{
				bytes.Repeat([]byte{0xAB}, 20),
			},
		},
		{
			name: "21 bytes splits to 2 packets",
			data: bytes.Repeat([]byte{0xCD}, 21),
			expected: [][]byte{
				bytes.Repeat([]byte{0xCD}, 20),
				bytes.Repeat([]byte{0xCD}, 1),
			},
		},
		{
			name: "45 bytes splits to 3 packets",
			data: bytes.Repeat([]byte{0xEF}, 45),
			expected: [][]byte{
				bytes.Repeat([]byte{0xEF}, 20),
				bytes.Repeat([]byte{0xEF}, 20),
				bytes.Repeat([]byte{0xEF}, 5),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitIntoPackets(tt.data)
			
			if len(result) != len(tt.expected) {
				t.Fatalf("Wrong number of packets: got %d, want %d", len(result), len(tt.expected))
			}
			
			for i, packet := range result {
				if !bytes.Equal(packet, tt.expected[i]) {
					t.Errorf("Packet %d mismatch:\nGot:  %X\nWant: %X", i, packet, tt.expected[i])
				}
			}
		})
	}
}

func TestValidateMessageLength(t *testing.T) {
	tests := []struct {
		name        string
		length      int
		expectError bool
	}{
		{
			name:        "Valid small length",
			length:      10,
			expectError: false,
		},
		{
			name:        "Valid max length",
			length:      MaxMessageLength,
			expectError: false,
		},
		{
			name:        "Zero length",
			length:      0,
			expectError: true,
		},
		{
			name:        "Negative length",
			length:      -1,
			expectError: true,
		},
		{
			name:        "Exceeds max length",
			length:      MaxMessageLength + 1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageLength(tt.length)
			if tt.expectError && err == nil {
				t.Errorf("Expected error for length %d", tt.length)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for length %d: %v", tt.length, err)
			}
		})
	}
}

func TestPacketCounterWrapping(t *testing.T) {
	// Test that packet counter wraps correctly from 15 to 0
	// Create a message that needs 17 continuation packets
	data := bytes.Repeat([]byte{0x42}, 350) // 1 start + 17 continuations
	
	result := BuildTLVPackets(data)
	
	// Find all continuation headers
	offset := 20 // Skip first packet
	expectedCounter := 0
	
	for offset < len(result) {
		if result[offset]&0x80 == 0 {
			t.Fatalf("Expected continuation bit at offset %d", offset)
		}
		
		actualCounter := int(result[offset] & 0x0F)
		if actualCounter != expectedCounter {
			t.Errorf("Counter at offset %d: got %d, want %d", offset, actualCounter, expectedCounter)
		}
		
		expectedCounter = (expectedCounter + 1) & 0x0F // Wrap at 16
		offset += 20 // Move to next packet
	}
}