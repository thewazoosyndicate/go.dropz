package tlv

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/logger"
)

// NewFragmentCollector creates a new fragment collector with the specified timeout
func NewFragmentCollector(timeout time.Duration) *FragmentCollector {
	fc := &FragmentCollector{
		fragments: make(map[byte]*MessageFragments),
		timeout:   timeout,
	}

	// Start cleanup goroutine
	go fc.cleanupStaleFragments()

	return fc
}

// ProcessFragment processes a TLV packet fragment and returns a complete message if ready
func (fc *FragmentCollector) ProcessFragment(data []byte) (*TLVMessage, error) {

	// Parse the header
	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Extract payload
	payload, err := ExtractPayload(data, header)
	if err != nil {
		return nil, fmt.Errorf("failed to extract payload: %w", err)
	}

	// Handle start packet
	if header.IsStart {
		return fc.handleStartPacket(header, payload)
	}

	// Handle continuation packet
	if header.IsContinuation {
		return fc.handleContinuationPacket(header, payload)
	}

	return nil, fmt.Errorf("packet is neither start nor continuation")
}

// handleStartPacket processes a start packet
func (fc *FragmentCollector) handleStartPacket(header *PacketHeader, payload []byte) (*TLVMessage, error) {
	log := logger.GetLogger()

	if len(payload) < 2 {
		return nil, fmt.Errorf("start packet payload too short: %d bytes", len(payload))
	}

	// First byte is command ID
	commandID := payload[0]

	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Check if we already have fragments for this command (shouldn't happen but handle it)
	if existing, exists := fc.fragments[commandID]; exists {
		log.Warnf("Discarding incomplete fragments for command 0x%02X due to new start packet", commandID)
		close(existing.complete)
		delete(fc.fragments, commandID)
	}

	// If the entire message fits in one packet, return it immediately
	if header.MessageLength <= len(payload) {
		log.Tracef("Single packet message for command 0x%02X, length %d", commandID, header.MessageLength)
		msg, err := ParseTLVResponse(payload[:header.MessageLength])
		if err != nil {
			return nil, err
		}
		msg.Timestamp = time.Now()
		return msg, nil
	}

	// Multi-packet message - create fragment tracker
	log.Debugf("Starting multi-packet collection for command 0x%02X, total length %d", commandID, header.MessageLength)
	
	fragments := &MessageFragments{
		commandID:      commandID,
		expectedLength: header.MessageLength,
		receivedLength: len(payload),
		packets:        make(map[int][]byte),
		lastPacketNum:  -1, // Will be 0 for first continuation
		startTime:      time.Now(),
		complete:       make(chan *TLVMessage, 1),
	}

	// Store the first packet's data (use -1 as key for start packet)
	fragments.packets[-1] = make([]byte, len(payload))
	copy(fragments.packets[-1], payload)

	fc.fragments[commandID] = fragments

	return nil, nil // Message not complete yet
}

// handleContinuationPacket processes a continuation packet
func (fc *FragmentCollector) handleContinuationPacket(header *PacketHeader, payload []byte) (*TLVMessage, error) {
	log := logger.GetLogger()

	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Find which command this continuation belongs to
	// We need to check all active fragments since continuation doesn't include command ID
	var fragments *MessageFragments
	for _, frag := range fc.fragments {
		// Check if this is the expected next packet
		expectedCounter := (frag.lastPacketNum + 1) & 0x0F
		if header.PacketCounter == expectedCounter {
			fragments = frag
			break
		}
	}

	if fragments == nil {
		return nil, fmt.Errorf("continuation packet %d with no matching start packet", header.PacketCounter)
	}

	// Validate packet sequence
	if err := ValidatePacketSequence(fragments.lastPacketNum, header.PacketCounter); err != nil {
		return nil, fmt.Errorf("packet sequence error for command 0x%02X: %w", fragments.commandID, err)
	}

	// Store the continuation packet data
	fragments.packets[header.PacketCounter] = make([]byte, len(payload))
	copy(fragments.packets[header.PacketCounter], payload)
	fragments.receivedLength += len(payload)
	fragments.lastPacketNum = header.PacketCounter

	log.Tracef("Received continuation packet %d for command 0x%02X (%d/%d bytes)",
		header.PacketCounter, fragments.commandID, fragments.receivedLength, fragments.expectedLength)

	// Check if message is complete
	if fragments.receivedLength >= fragments.expectedLength {
		return fc.assembleMessage(fragments)
	}

	return nil, nil // Message not complete yet
}

// assembleMessage assembles a complete message from fragments
func (fc *FragmentCollector) assembleMessage(fragments *MessageFragments) (*TLVMessage, error) {
	log := logger.GetLogger()

	// Assemble all packets in order
	fullData := make([]byte, 0, fragments.expectedLength)
	
	// Start with the initial packet (stored at key -1)
	if startPacket, exists := fragments.packets[-1]; exists {
		fullData = append(fullData, startPacket...)
	} else {
		return nil, fmt.Errorf("missing start packet for command 0x%02X", fragments.commandID)
	}

	// Add continuation packets in order (counter wraps 0-15)
	if fragments.lastPacketNum >= 0 {
		// We have continuation packets
		for i := 0; i <= fragments.lastPacketNum; i++ {
			packetNum := i & 0x0F
			if packet, exists := fragments.packets[packetNum]; exists {
				fullData = append(fullData, packet...)
			} else {
				return nil, fmt.Errorf("missing continuation packet %d for command 0x%02X", packetNum, fragments.commandID)
			}
		}
	}

	// Trim to expected length
	if len(fullData) > fragments.expectedLength {
		fullData = fullData[:fragments.expectedLength]
	}

	// Parse the complete message
	msg, err := ParseTLVResponse(fullData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assembled message: %w", err)
	}

	msg.Timestamp = time.Now()

	log.Debugf("Successfully assembled message for command 0x%02X, %d bytes", fragments.commandID, len(fullData))

	// Clean up fragments
	close(fragments.complete)
	delete(fc.fragments, fragments.commandID)

	return msg, nil
}

// cleanupStaleFragments removes fragments that have timed out
func (fc *FragmentCollector) cleanupStaleFragments() {
	ticker := time.NewTicker(time.Duration(CleanupInterval) * time.Millisecond)
	defer ticker.Stop()

	log := logger.GetLogger()

	for range ticker.C {
		fc.mu.Lock()
		now := time.Now()
		
		for cmdID, fragments := range fc.fragments {
			if now.Sub(fragments.startTime) > fc.timeout {
				log.Warnf("Cleaning up stale fragments for command 0x%02X (timeout after %v)", 
					cmdID, fc.timeout)
				close(fragments.complete)
				delete(fc.fragments, cmdID)
			}
		}
		
		fc.mu.Unlock()
	}
}

// NewResponseTracker creates a new response tracker
func NewResponseTracker() *ResponseTracker {
	return &ResponseTracker{
		pendingCommands: make(map[byte]chan *TLVMessage),
	}
}

// RegisterCommand registers a command expecting a response
func (rt *ResponseTracker) RegisterCommand(commandID byte) chan *TLVMessage {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Create response channel
	respChan := make(chan *TLVMessage, 1)
	
	// Close any existing channel for this command
	if existing, exists := rt.pendingCommands[commandID]; exists {
		close(existing)
	}
	
	rt.pendingCommands[commandID] = respChan
	return respChan
}

// UnregisterCommand removes a command from tracking
func (rt *ResponseTracker) UnregisterCommand(commandID byte) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if ch, exists := rt.pendingCommands[commandID]; exists {
		close(ch)
		delete(rt.pendingCommands, commandID)
	}
}

// RouteResponse routes a response to the appropriate waiting channel
func (rt *ResponseTracker) RouteResponse(msg *TLVMessage) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if ch, exists := rt.pendingCommands[msg.CommandID]; exists {
		select {
		case ch <- msg:
			return true
		default:
			// Channel full, response dropped
			logger.GetLogger().Warnf("Response channel full for command 0x%02X", msg.CommandID)
			return false
		}
	}

	logger.GetLogger().Tracef("No pending command for response 0x%02X", msg.CommandID)
	return false
}