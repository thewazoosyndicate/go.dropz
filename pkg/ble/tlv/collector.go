package tlv

import (
	"fmt"
	"time"
)

// NewFragmentCollector creates a new fragment collector with the specified timeout
func NewFragmentCollector(timeout time.Duration) *FragmentCollector {
	fc := &FragmentCollector{
		fragments: make(map[byte]*MessageFragments),
		timeout:   timeout,
		stopCh:    make(chan struct{}),
	}

	go fc.cleanupStaleFragments()

	return fc
}

// Stop terminates the background cleanup goroutine
func (fc *FragmentCollector) Stop() {
	close(fc.stopCh)
}

// Reset clears all pending fragments
func (fc *FragmentCollector) Reset() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.fragments = make(map[byte]*MessageFragments)
}

// ProcessFragment processes a TLV packet fragment and returns a complete message if ready
func (fc *FragmentCollector) ProcessFragment(data []byte) (*TLVMessage, error) {
	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	payload, err := ExtractPayload(data, header)
	if err != nil {
		return nil, fmt.Errorf("failed to extract payload: %w", err)
	}

	if header.IsStart {
		return fc.handleStartPacket(header, payload)
	}

	if header.IsContinuation {
		return fc.handleContinuationPacket(header, payload)
	}

	return nil, fmt.Errorf("packet is neither start nor continuation")
}

func (fc *FragmentCollector) handleStartPacket(header *PacketHeader, payload []byte) (*TLVMessage, error) {
	if len(payload) < 2 {
		return nil, fmt.Errorf("start packet payload too short: %d bytes", len(payload))
	}

	commandID := payload[0]

	fc.mu.Lock()
	defer fc.mu.Unlock()

	if _, exists := fc.fragments[commandID]; exists {
		delete(fc.fragments, commandID)
	}

	// Single-packet message
	if header.MessageLength <= len(payload) {
		msg, err := ParseTLVResponse(payload[:header.MessageLength])
		if err != nil {
			return nil, err
		}
		msg.Timestamp = time.Now()
		return msg, nil
	}

	// Multi-packet message — create fragment tracker
	fragments := &MessageFragments{
		commandID:      commandID,
		expectedLength: header.MessageLength,
		receivedLength: len(payload),
		packets:        make(map[int][]byte),
		lastPacketNum:  -1,
		startTime:      time.Now(),
	}

	fragments.packets[-1] = make([]byte, len(payload))
	copy(fragments.packets[-1], payload)

	fc.fragments[commandID] = fragments

	return nil, nil
}

func (fc *FragmentCollector) handleContinuationPacket(header *PacketHeader, payload []byte) (*TLVMessage, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	var fragments *MessageFragments
	for _, frag := range fc.fragments {
		expectedCounter := (frag.lastPacketNum + 1) & 0x0F
		if header.PacketCounter == expectedCounter {
			fragments = frag
			break
		}
	}

	if fragments == nil {
		return nil, fmt.Errorf("continuation packet %d with no matching start packet", header.PacketCounter)
	}

	if err := ValidatePacketSequence(fragments.lastPacketNum, header.PacketCounter); err != nil {
		return nil, fmt.Errorf("packet sequence error for command 0x%02X: %w", fragments.commandID, err)
	}

	fragments.packets[header.PacketCounter] = make([]byte, len(payload))
	copy(fragments.packets[header.PacketCounter], payload)
	fragments.receivedLength += len(payload)
	fragments.lastPacketNum = header.PacketCounter

	if fragments.receivedLength >= fragments.expectedLength {
		return fc.assembleMessage(fragments)
	}

	return nil, nil
}

func (fc *FragmentCollector) assembleMessage(fragments *MessageFragments) (*TLVMessage, error) {
	fullData := make([]byte, 0, fragments.expectedLength)

	if startPacket, exists := fragments.packets[-1]; exists {
		fullData = append(fullData, startPacket...)
	} else {
		return nil, fmt.Errorf("missing start packet for command 0x%02X", fragments.commandID)
	}

	if fragments.lastPacketNum >= 0 {
		for i := 0; i <= fragments.lastPacketNum; i++ {
			packetNum := i & 0x0F
			if packet, exists := fragments.packets[packetNum]; exists {
				fullData = append(fullData, packet...)
			} else {
				return nil, fmt.Errorf("missing continuation packet %d for command 0x%02X", packetNum, fragments.commandID)
			}
		}
	}

	if len(fullData) > fragments.expectedLength {
		fullData = fullData[:fragments.expectedLength]
	}

	msg, err := ParseTLVResponse(fullData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assembled message: %w", err)
	}

	msg.Timestamp = time.Now()
	delete(fc.fragments, fragments.commandID)

	return msg, nil
}

func (fc *FragmentCollector) cleanupStaleFragments() {
	ticker := time.NewTicker(time.Duration(CleanupInterval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-fc.stopCh:
			return
		case <-ticker.C:
			fc.mu.Lock()
			now := time.Now()
			for cmdID, fragments := range fc.fragments {
				if now.Sub(fragments.startTime) > fc.timeout {
					delete(fc.fragments, cmdID)
				}
			}
			fc.mu.Unlock()
		}
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

	respChan := make(chan *TLVMessage, 1)

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

// SetPushHandler sets the handler for unsolicited push notifications (0x92/0x93)
func (rt *ResponseTracker) SetPushHandler(handler func(string, *TLVMessage)) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.pushHandler = handler
}

// RouteResponse routes a response to the appropriate waiting channel,
// or dispatches to the push handler for async notifications.
func (rt *ResponseTracker) RouteResponse(macAddress string, msg *TLVMessage) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if ch, exists := rt.pendingCommands[msg.CommandID]; exists {
		select {
		case ch <- msg:
			return true
		default:
			return false
		}
	}

	// Unsolicited push notifications have no pending command
	if (msg.CommandID == 0x92 || msg.CommandID == 0x93) && rt.pushHandler != nil {
		rt.pushHandler(macAddress, msg)
		return true
	}

	return false
}
