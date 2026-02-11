package tlv

// BuildCommandPacket builds a TLV command packet with proper header
// Uses Extended 13-bit format as recommended by OpenGoPro spec
func BuildCommandPacket(commandID byte, params []byte) []byte {
	// Calculate total message length (command ID + params)
	messageLen := 1 + len(params)
	
	// Build the message data
	messageData := make([]byte, 0, messageLen)
	messageData = append(messageData, commandID)
	messageData = append(messageData, params...)
	
	return BuildTLVPackets(messageData)
}

// BuildQueryPacket builds a TLV query packet
func BuildQueryPacket(queryID byte, params []byte) []byte {
	// Calculate total message length (query ID + params)
	messageLen := 1 + len(params)
	
	// Build the message data
	messageData := make([]byte, 0, messageLen)
	messageData = append(messageData, queryID)
	messageData = append(messageData, params...)
	
	return BuildTLVPackets(messageData)
}

// BuildTLVPackets builds TLV packets from raw message data
// Automatically handles fragmentation for messages > 20 bytes
// Uses Extended 13-bit header format as recommended by OpenGoPro
func BuildTLVPackets(messageData []byte) []byte {
	messageLen := len(messageData)
	
	// For single packet messages (≤ 20 bytes total including header)
	if messageLen <= 18 { // 20 - 2 bytes for extended 13-bit header
		return buildSinglePacket(messageData)
	}
	
	// For multi-packet messages
	return buildMultiPackets(messageData)
}

// buildSinglePacket builds a single TLV packet with Extended 13-bit header
func buildSinglePacket(data []byte) []byte {
	messageLen := len(data)
	
	// Build Extended 13-bit header (RECOMMENDED format)
	// First byte: 001LLLLL (bits 5-6 = 01 for extended 13-bit, bits 0-4 = upper 5 bits of length)
	// Second byte: LLLLLLLL (lower 8 bits of length)
	header := make([]byte, 2)
	header[0] = HeaderExtended13 | byte((messageLen>>8)&Extended13Mask)
	header[1] = byte(messageLen & 0xFF)
	
	// Combine header and data
	packet := make([]byte, 0, 2+messageLen)
	packet = append(packet, header...)
	packet = append(packet, data...)
	
	return packet
}

// buildMultiPackets builds multiple TLV packets for large messages
func buildMultiPackets(data []byte) []byte {
	messageLen := len(data)
	packets := make([]byte, 0, messageLen+20) // Rough estimate with headers
	
	// First packet with Extended 13-bit header
	// Can hold 18 bytes of data (20 - 2 for header)
	firstPacketDataSize := 18
	if messageLen < firstPacketDataSize {
		firstPacketDataSize = messageLen
	}
	
	// Build first packet header
	header := make([]byte, 2)
	header[0] = HeaderExtended13 | byte((messageLen>>8)&Extended13Mask)
	header[1] = byte(messageLen & 0xFF)
	
	// Add first packet
	firstPacket := make([]byte, 0, 20)
	firstPacket = append(firstPacket, header...)
	firstPacket = append(firstPacket, data[:firstPacketDataSize]...)
	packets = append(packets, firstPacket...)
	
	// Build continuation packets
	remaining := data[firstPacketDataSize:]
	packetCounter := 0
	
	for len(remaining) > 0 {
		// Continuation packet can hold 19 bytes of data (20 - 1 for header)
		chunkSize := 19
		if len(remaining) < chunkSize {
			chunkSize = len(remaining)
		}
		
		// Build continuation header (1 byte)
		// Format: 1000CCCC (bit 7 = 1 for continuation, bits 0-3 = counter)
		contHeader := ContinuationBit | byte(packetCounter&PacketCountMask)
		
		// Build continuation packet
		contPacket := make([]byte, 0, 20)
		contPacket = append(contPacket, contHeader)
		contPacket = append(contPacket, remaining[:chunkSize]...)
		
		packets = append(packets, contPacket...)
		
		remaining = remaining[chunkSize:]
		packetCounter = (packetCounter + 1) & 0x0F // Wrap at 16
	}
	
	return packets
}

// SplitIntoPackets splits already-built TLV packets into BLE-sized chunks
// This is useful when sending multi-packet messages
func SplitIntoPackets(data []byte) [][]byte {
	var packets [][]byte
	
	for len(data) > 0 {
		chunkSize := MaxPacketSize
		if len(data) < chunkSize {
			chunkSize = len(data)
		}
		
		packet := make([]byte, chunkSize)
		copy(packet, data[:chunkSize])
		packets = append(packets, packet)
		
		data = data[chunkSize:]
	}
	
	return packets
}

