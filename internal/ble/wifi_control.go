package ble

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dropz/dropz/internal/logging"
)

// WiFiAPMode represents the WiFi Access Point control modes
type WiFiAPMode byte

const (
	WiFiAPModeDisable WiFiAPMode = 0 // Disable WiFi Access Point
	WiFiAPModeEnable  WiFiAPMode = 1 // Enable WiFi Access Point
	WiFiAPModeBounce  WiFiAPMode = 2 // Bounce (disable then enable) WiFi Access Point
)

// HardwareInfo contains information from GetHardwareInfo command
type HardwareInfo struct {
	ModelNumber     int
	ModelName       string
	FirmwareVersion string
	SerialNumber    string
	APSSID          string
	MACAddress      string
}

// SetAPControl controls the WiFi Access Point using the proper OpenGoPro command
func (m *Manager) SetAPControl(macAddress string, mode WiFiAPMode) error {
	m.log.Debug("Setting WiFi AP control", "mode", mode, "device", macAddress)

	// Validate mode
	if mode > WiFiAPModeBounce {
		return fmt.Errorf("invalid WiFi AP mode: %d", mode)
	}

	// Send the command with length-prefixed mode parameter
	response, err := m.sendCommand(macAddress, CmdSetAPControl, []byte{0x01, byte(mode)})
	if err != nil {
		return fmt.Errorf("failed to send AP control command: %w", err)
	}

	// Check response status (0x00 = success)
	if response.Status != 0x00 {
		m.log.Error("WiFi AP control command failed", "status", fmt.Sprintf("0x%02X", response.Status))
		return fmt.Errorf("WiFi AP control failed: status=0x%02X", response.Status)
	}

	if mode == WiFiAPModeEnable || mode == WiFiAPModeBounce {
		time.Sleep(2 * time.Second)
	}

	return nil
}

// GetHardwareInfo retrieves hardware information from the camera
func (m *Manager) GetHardwareInfo(macAddress string) (*HardwareInfo, error) {
	response, err := m.sendCommand(macAddress, CmdGetHardwareInfo, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get hardware info: %w", err)
	}

	// Parse the response
	info, err := parseHardwareInfo(response.Data, m.log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hardware info: %w", err)
	}

	return info, nil
}

// parseHardwareInfo parses the hardware info response using OpenGoPro's sequential length-prefixed format
func parseHardwareInfo(data []byte, log *slog.Logger) (*HardwareInfo, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("hardware info response too short: %d bytes", len(data))
	}

	info := &HardwareInfo{}
	buf := data

	// Helper function to read length-prefixed field
	readField := func(fieldName string) ([]byte, error) {
		if len(buf) < 1 {
			return nil, fmt.Errorf("missing %s length byte", fieldName)
		}
		length := int(buf[0])
		buf = buf[1:]

		if len(buf) < length {
			return nil, fmt.Errorf("%s truncated: expected %d bytes, have %d",
				fieldName, length, len(buf))
		}

		fieldData := buf[:length]
		buf = buf[length:]
		return fieldData, nil
	}

	// 1. Model Number (4 bytes big-endian)
	if modelData, err := readField("model_number"); err != nil {
		log.Warn("Failed to read model number", "err", err)
	} else if len(modelData) == 4 {
		info.ModelNumber = int(modelData[0])<<24 | int(modelData[1])<<16 |
			int(modelData[2])<<8 | int(modelData[3])
	}

	// 2. Model Name
	if modelName, err := readField("model_name"); err != nil {
		log.Warn("Failed to read model name", "err", err)
	} else {
		info.ModelName = strings.TrimRight(string(modelName), "\x00")
	}

	// 3. Board Type/Deprecated field - skip it
	if _, err := readField("board_type/deprecated"); err != nil {
		logging.Trace(log, "Failed to skip board type/deprecated field", "err", err)
	}

	// 4. Firmware Version
	if firmware, err := readField("firmware_version"); err != nil {
		log.Warn("Failed to read firmware version", "err", err)
	} else {
		info.FirmwareVersion = strings.TrimRight(string(firmware), "\x00")
	}

	// 5. Serial Number
	if serial, err := readField("serial_number"); err != nil {
		log.Warn("Failed to read serial number", "err", err)
	} else {
		info.SerialNumber = strings.TrimRight(string(serial), "\x00")
	}

	// 6. AP SSID
	if ssid, err := readField("ap_ssid"); err != nil {
		log.Warn("Failed to read AP SSID", "err", err)
	} else {
		info.APSSID = strings.TrimRight(string(ssid), "\x00")
	}

	// 7. AP MAC Address
	if mac, err := readField("ap_mac"); err != nil {
		log.Warn("Failed to read AP MAC", "err", err)
	} else if len(mac) == 6 {
		info.MACAddress = fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
			mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
	} else {
		// MAC address may be provided as a string
		info.MACAddress = strings.TrimRight(string(mac), "\x00")
	}

	// Validate we got at least the essential fields
	if info.ModelName == "" && info.FirmwareVersion == "" {
		return nil, fmt.Errorf("hardware info missing essential fields")
	}

	return info, nil
}

// SetThirdPartyClient identifies this client as a third-party app to the camera
func (m *Manager) SetThirdPartyClient(macAddress string) error {
	_, err := m.sendCommand(macAddress, CmdRegisterClient, nil)
	if err != nil {
		return fmt.Errorf("failed to set third party client: %w", err)
	}

	return nil
}

// SetLocalDateTime sets the camera's date/time with timezone and DST info.
// Uses command 0x0F (Set Local Date Time) supported on HERO11+.
// Falls back to 0x0D (Set Date Time) without timezone on older cameras.
func (m *Manager) SetLocalDateTime(macAddress string, t time.Time) error {
	_, offset := t.Zone()
	offsetMinutes := int16(offset / 60)
	isDST := byte(0)
	if t.IsDST() {
		isDST = 1
	}

	year := uint16(t.Year())
	// Length-prefixed payload: 0x0A (10 bytes) + date/time fields
	params := []byte{
		0x0A, // param length
		byte(year >> 8), byte(year & 0xFF),
		byte(t.Month()),
		byte(t.Day()),
		byte(t.Hour()),
		byte(t.Minute()),
		byte(t.Second()),
		byte(offsetMinutes >> 8), byte(offsetMinutes & 0xFF),
		isDST,
	}

	response, err := m.sendCommand(macAddress, CmdSetLocalDateTime, params)
	if err != nil {
		return fmt.Errorf("failed to set local date/time: %w", err)
	}

	if response.Status == 0x02 {
		// Invalid parameter — camera may not support timezone variant, fall back to 0x0D
		m.log.Debug("Camera doesn't support Set Local Date Time, falling back to Set Date Time")
		params = []byte{
			0x07, // param length
			byte(year >> 8), byte(year & 0xFF),
			byte(t.Month()),
			byte(t.Day()),
			byte(t.Hour()),
			byte(t.Minute()),
			byte(t.Second()),
		}
		response, err = m.sendCommand(macAddress, CmdSetDateTime, params)
		if err != nil {
			return fmt.Errorf("failed to set date/time: %w", err)
		}
	}

	if response.Status != 0x00 {
		return fmt.Errorf("set date/time failed: status=0x%02X", response.Status)
	}

	m.log.Info("Camera date/time set", "time", t.Format(time.RFC3339))
	return nil
}
