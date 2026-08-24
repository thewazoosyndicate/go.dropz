package ble

import (
	"fmt"
	"strings"
)

// QueryStatuses queries multiple status IDs in a single BLE request and returns parsed TLV results.
func (m *Manager) QueryStatuses(macAddress string, statusIDs []byte) (map[byte][]byte, error) {
	response, err := m.sendQuery(macAddress, QueryGetStatus, statusIDs)
	if err != nil {
		return nil, err
	}
	return parseTLVPairs(response.Data), nil
}

// GetWifiCredentials retrieves WiFi SSID and password from the device
func (m *Manager) GetWifiCredentials(macAddress string) (string, string, error) {
	c := m.getConn(macAddress)
	if c == nil {
		return "", "", fmt.Errorf("device not connected: %s", macAddress)
	}

	m.mutex.RLock()
	ssidChar, ssidOK := c.chars[CharWifiSSID]
	passChar, passOK := c.chars[CharWifiPassword]
	m.mutex.RUnlock()

	if !ssidOK {
		return "", "", fmt.Errorf("WiFi SSID characteristic not found")
	}

	ssidData := make([]byte, 32)
	n, err := ssidChar.Read(ssidData)
	if err != nil {
		return "", "", fmt.Errorf("failed to read SSID: %w", err)
	}
	ssid := strings.TrimSpace(string(ssidData[:n]))

	if !passOK {
		return "", "", fmt.Errorf("WiFi password characteristic not found")
	}

	passData := make([]byte, 64)
	n, err = passChar.Read(passData)
	if err != nil {
		return "", "", fmt.Errorf("failed to read password: %w", err)
	}
	password := strings.TrimSpace(string(passData[:n]))

	return ssid, password, nil
}

// GetBatteryLevel queries the battery level from the device
func (m *Manager) GetBatteryLevel(macAddress string) (int, error) {
	statuses, err := m.QueryStatuses(macAddress, []byte{StatusBatteryPercentage})
	if err != nil {
		return 0, err
	}
	if v, ok := statuses[StatusBatteryPercentage]; ok && len(v) >= 1 {
		return int(v[0]), nil
	}
	return 0, fmt.Errorf("battery percentage not found in response")
}

// KeepAlive sends a keep-alive to prevent the camera from auto-sleeping.
// Per OpenGoPro this is the LED setting (91) set to 66, written to the
// Settings characteristic; it was previously sent to the Command
// characteristic, which cameras reject.
func (m *Manager) KeepAlive(macAddress string) error {
	resp, err := m.sendSetting(macAddress, SettingKeepAlive, []byte{0x01, KeepAliveValue})
	if err != nil {
		return err
	}
	if resp.Status != 0 {
		return fmt.Errorf("keep-alive returned status %d", resp.Status)
	}
	return nil
}

// Sleep puts the device to sleep
func (m *Manager) Sleep(macAddress string) error {
	resp, err := m.sendCommand(macAddress, CmdSleep, nil)
	if err != nil {
		return err
	}
	if resp.Status != 0 {
		return fmt.Errorf("sleep command returned status %d", resp.Status)
	}
	return nil
}

// SendPairingFinish sends the RequestPairingFinish protobuf command (Feature 0x03, Action 0x01)
// to transition the camera to paired state.
func (m *Manager) SendPairingFinish(macAddress string) error {
	// Hand-encoded protobuf: field 1 (varint, tag=0x08, value=0=SUCCESS), field 2 (string, tag=0x12, "dropz")
	phoneName := []byte("dropz")
	payload := append([]byte{0x08, 0x00, 0x12, byte(len(phoneName))}, phoneName...)
	// ActionID 0x01 prepended to payload
	data := append([]byte{0x01}, payload...)
	_, err := m.sendMessage(macAddress, CharNetworkMgmtCommand, 0x03, data, buildProtobufPacket)
	return err
}

// RegisterStatusUpdates subscribes to push notifications for the given status IDs.
// The camera will send async 0x93 notifications whenever these statuses change.
func (m *Manager) RegisterStatusUpdates(macAddress string, statusIDs []byte) error {
	response, err := m.sendQuery(macAddress, QueryRegisterStatusUpdates, statusIDs)
	if err != nil {
		return fmt.Errorf("failed to register status updates: %w", err)
	}
	if response.Status != 0x00 {
		return fmt.Errorf("register status updates failed: status=0x%02X", response.Status)
	}

	// Dispatch current values to the callback
	m.mutex.RLock()
	cb := m.statusCallback
	m.mutex.RUnlock()
	if cb != nil {
		for id, value := range parseTLVPairs(response.Data) {
			cb(macAddress, id, value)
		}
	}

	return nil
}
