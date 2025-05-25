package ble

import "fmt"

// OpenGoPro command helpers following the specification

// createOpenGoProPacket creates a properly formatted packet following OpenGoPro spec
func createOpenGoProPacket(payload []byte) [][]byte {
	const maxPayloadSize = 18 // MTU - header size

	if len(payload) <= maxPayloadSize {
		// Single packet
		packet := make([]byte, 2+len(payload))
		packet[0] = 0x00 // General packet, no continuation
		packet[1] = byte(len(payload))
		copy(packet[2:], payload)
		return [][]byte{packet}
	}

	// Multi-packet
	var packets [][]byte
	remaining := payload
	isFirst := true

	for len(remaining) > 0 {
		chunkSize := maxPayloadSize
		if len(remaining) < chunkSize {
			chunkSize = len(remaining)
		}

		packet := make([]byte, 2+chunkSize)

		if isFirst {
			packet[0] = 0x10 // Start packet
			isFirst = false
		} else if len(remaining) <= maxPayloadSize {
			packet[0] = 0x80 // End packet
		} else {
			packet[0] = 0x60 // Continuation packet
		}

		packet[1] = byte(chunkSize)
		copy(packet[2:], remaining[:chunkSize])

		packets = append(packets, packet)
		remaining = remaining[chunkSize:]
	}

	return packets
}

// parseOpenGoProResponse parses multi-packet responses
func parseOpenGoProResponse(packets [][]byte) ([]byte, error) {
	var response []byte

	for _, packet := range packets {
		if len(packet) < 2 {
			return nil, fmt.Errorf("invalid packet format")
		}

		header := packet[0]
		length := packet[1]

		if len(packet) < int(2+length) {
			return nil, fmt.Errorf("packet length mismatch")
		}

		payload := packet[2 : 2+length]
		response = append(response, payload...)

		// Check if this is the last packet
		if header == 0x00 || header == 0x80 {
			break
		}
	}

	return response, nil
}
