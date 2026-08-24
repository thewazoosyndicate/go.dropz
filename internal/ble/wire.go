package ble

import (
	"fmt"
	"time"

	"github.com/dropz/dropz/internal/ble/tlv"
)

// parseTLVPairs parses a byte sequence of [ID][Length][Value...] triplets into a map.
func parseTLVPairs(data []byte) map[byte][]byte {
	result := make(map[byte][]byte)
	offset := 0
	for offset+2 <= len(data) {
		id := data[offset]
		length := int(data[offset+1])
		offset += 2
		if offset+length > len(data) {
			break
		}
		value := make([]byte, length)
		copy(value, data[offset:offset+length])
		result[id] = value
		offset += length
	}
	return result
}

// sendMessage sends a TLV message on the given characteristic and waits for a response.
func (m *Manager) sendMessage(macAddress string, charUUID string, id byte, data []byte, buildPacket func(byte, []byte) []byte) (Response, error) {
	c := m.getConn(macAddress)
	if c == nil {
		return Response{}, fmt.Errorf("device not connected")
	}

	m.mutex.RLock()
	char, exists := c.chars[charUUID]
	m.mutex.RUnlock()
	if !exists {
		return Response{}, fmt.Errorf("characteristic %s not found", GetCharacteristicName(charUUID))
	}

	responseChan := c.tracker.RegisterCommand(id)
	defer c.tracker.UnregisterCommand(id)

	packets := tlv.SplitIntoPackets(buildPacket(id, data))

	for i, pkt := range packets {
		if _, err := writeCharacteristic(char, pkt); err != nil {
			return Response{}, fmt.Errorf("failed to send packet %d: %w", i, err)
		}
	}

	select {
	case message, ok := <-responseChan:
		// The tracker closes the channel when a concurrent sender reuses this
		// command ID or the connection drops; dereferencing would panic.
		if !ok || message == nil {
			return Response{}, fmt.Errorf("command 0x%02X superseded or connection dropped", id)
		}
		return Response{
			Status: message.Status,
			Data:   message.Payload,
		}, nil
	case <-time.After(responseTimeout):
		// Returned only; the caller that decides logs it
		return Response{}, fmt.Errorf("timeout waiting for response to 0x%02X after %v", id, responseTimeout)
	}
}

func (m *Manager) sendCommand(macAddress string, commandID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharCommand, commandID, data, tlv.BuildCommandPacket)
}

func (m *Manager) sendQuery(macAddress string, queryID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharQuery, queryID, data, tlv.BuildQueryPacket)
}

func (m *Manager) sendSetting(macAddress string, settingID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharSettings, settingID, data, tlv.BuildCommandPacket)
}

func (m *Manager) handleNotification(macAddress string, collector *tlv.FragmentCollector, data []byte) {
	c := m.getConn(macAddress)
	if c == nil {
		return
	}

	message, err := collector.ProcessFragment(data)
	if err != nil {
		// A real protocol failure: the response this fragment belonged to
		// will surface later as a command timeout
		m.log.Warn("Failed to process TLV fragment", "ble_addr", macAddress, "raw", fmt.Sprintf("%x", data), "err", err)
	} else if message != nil {
		if !c.tracker.RouteResponse(macAddress, message) {
			m.log.Debug("Unrouted notification", "ble_addr", macAddress, "cmd", fmt.Sprintf("0x%02X", message.CommandID), "status", message.Status, "len", len(message.Payload))
		}
	}
}

// buildProtobufPacket frames a protobuf message: packet header + [FeatureID][ActionID][protobuf].
// Protobuf messages use the same packetization as TLV commands (OpenGoPro
// data_protocol); the header was previously missing, so cameras parsed the
// feature ID byte as a length header and dropped the message.
func buildProtobufPacket(featureID byte, data []byte) []byte {
	return tlv.BuildTLVPackets(append([]byte{featureID}, data...))
}
