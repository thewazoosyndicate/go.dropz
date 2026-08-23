package tlv

import (
	"errors"
	"fmt"
)

var (
	// ErrEmptyPacket indicates an empty packet was received
	ErrEmptyPacket = errors.New("empty packet")
	// ErrInvalidHeader indicates an invalid header format
	ErrInvalidHeader = errors.New("invalid header format")
	// ErrInsufficientData indicates not enough data for the header type
	ErrInsufficientData = errors.New("insufficient data for header type")
	// ErrInvalidHeaderType indicates an unknown header type
	ErrInvalidHeaderType = errors.New("invalid header type")
)

// ParseHeader parses a TLV packet header according to OpenGoPro specification
func ParseHeader(data []byte) (*PacketHeader, error) {
	if len(data) == 0 {
		return nil, ErrEmptyPacket
	}

	header := &PacketHeader{}
	firstByte := data[0]

	// Check if this is a continuation packet (bit 7 set)
	if firstByte&ContinuationBit != 0 {
		header.IsContinuation = true
		header.PacketCounter = int(firstByte & PacketCountMask)
		header.HeaderSize = 1
		return header, nil
	}

	// This is a start packet
	header.IsStart = true

	// Determine header type from bits 5-6
	headerTypeBits := firstByte & HeaderTypeMask

	switch headerTypeBits {
	case HeaderGeneral: // 00: General (5-bit length)
		header.PacketType = HeaderTypeGeneral
		header.MessageLength = int(firstByte & GeneralLenMask)
		header.HeaderSize = 1

	case HeaderExtended13: // 01: Extended (13-bit) - RECOMMENDED for sending
		if len(data) < 2 {
			return nil, fmt.Errorf("%w: extended 13-bit header requires 2 bytes", ErrInsufficientData)
		}
		header.PacketType = HeaderTypeExtended13
		// Upper 5 bits from first byte, lower 8 bits from second byte
		header.MessageLength = (int(firstByte&Extended13Mask) << 8) | int(data[1])
		header.HeaderSize = 2

	case HeaderExtended16: // 10: Extended (16-bit) - RECEIVE ONLY
		if len(data) < 3 {
			return nil, fmt.Errorf("%w: extended 16-bit header requires 3 bytes", ErrInsufficientData)
		}
		header.PacketType = HeaderTypeExtended16
		// Full 16 bits from bytes 1 and 2
		header.MessageLength = (int(data[1]) << 8) | int(data[2])
		header.HeaderSize = 3

	default:
		return nil, fmt.Errorf("%w: header type bits %02x", ErrInvalidHeaderType, headerTypeBits)
	}

	return header, nil
}

// ExtractPayload extracts the payload from a packet based on its header
func ExtractPayload(data []byte, header *PacketHeader) ([]byte, error) {
	if header.IsContinuation {
		// Continuation packet: everything after the header is payload
		if len(data) <= header.HeaderSize {
			return []byte{}, nil
		}
		return data[header.HeaderSize:], nil
	}

	// Start packet: skip header
	if len(data) <= header.HeaderSize {
		return []byte{}, nil
	}
	return data[header.HeaderSize:], nil
}

// ValidatePacketSequence checks if a continuation packet has the expected counter
func ValidatePacketSequence(expected, received int) error {
	// Packet counter wraps from 0xF to 0x0
	nextExpected := (expected + 1) & 0x0F
	if received != nextExpected {
		return fmt.Errorf("packet sequence error: expected %d, got %d", nextExpected, received)
	}
	return nil
}

// ParseTLVResponse parses a complete TLV message into command ID, status, and payload
func ParseTLVResponse(data []byte) (*TLVMessage, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("TLV response too short: need at least 2 bytes, got %d", len(data))
	}

	msg := &TLVMessage{
		CommandID: data[0],
		Status:    data[1],
	}

	// Remaining bytes are the payload
	if len(data) > 2 {
		msg.Payload = data[2:]
	} else {
		msg.Payload = []byte{}
	}

	return msg, nil
}