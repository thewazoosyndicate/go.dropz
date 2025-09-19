package ble

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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
	ModelNumber      int
	ModelName        string
	FirmwareVersion  string
	SerialNumber     string
	APSSID          string
	MACAddress      string
}

// SetAPControl controls the WiFi Access Point using the proper OpenGoPro command
func (m *Manager) SetAPControl(macAddress string, mode WiFiAPMode) error {
	m.log.Infof("Setting WiFi AP control: mode=%d device=%s", mode, macAddress)

	// Validate mode
	if mode > WiFiAPModeBounce {
		return fmt.Errorf("invalid WiFi AP mode: %d", mode)
	}

	// Send the command with mode parameter
	_, err := m.sendCommand(macAddress, CmdSetAPControl, []byte{byte(mode)})
	if err != nil {
		return fmt.Errorf("failed to set AP control: %v", err)
	}

	// For enable mode, wait a bit for the AP to come up
	if mode == WiFiAPModeEnable || mode == WiFiAPModeBounce {
		time.Sleep(2 * time.Second)
		m.log.Debug("WiFi AP enable command sent, waiting for AP to start")
	}

	return nil
}

// EnableWiFiAP enables the WiFi Access Point (convenience method)
func (m *Manager) EnableWiFiAP(macAddress string) error {
	return m.SetAPControl(macAddress, WiFiAPModeEnable)
}

// DisableWiFiAP disables the WiFi Access Point (convenience method)
func (m *Manager) DisableWiFiAP(macAddress string) error {
	return m.SetAPControl(macAddress, WiFiAPModeDisable)
}

// GetHardwareInfo retrieves hardware information from the camera
func (m *Manager) GetHardwareInfo(macAddress string) (*HardwareInfo, error) {
	m.log.Debug("Getting hardware info from device")

	response, err := m.sendCommand(macAddress, CmdGetHardwareInfo, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get hardware info: %v", err)
	}

	// Parse the response
	info, err := parseHardwareInfo(response.Data, m.log)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hardware info: %v", err)
	}

	m.log.Infof("Hardware info: Model=%s, FW=%s, SN=%s, SSID=%s",
		info.ModelName, info.FirmwareVersion, info.SerialNumber, info.APSSID)

	return info, nil
}

// parseHardwareInfo parses the hardware info response using OpenGoPro's sequential length-prefixed format
func parseHardwareInfo(data []byte, log *logrus.Logger) (*HardwareInfo, error) {
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
		log.Warnf("Failed to read model number: %v", err)
	} else if len(modelData) == 4 {
		info.ModelNumber = int(modelData[0])<<24 | int(modelData[1])<<16 |
			int(modelData[2])<<8 | int(modelData[3])
	}

	// 2. Model Name
	if modelName, err := readField("model_name"); err != nil {
		log.Warnf("Failed to read model name: %v", err)
	} else {
		info.ModelName = strings.TrimRight(string(modelName), "\x00")
	}

	// 3. Board Type/Deprecated field - skip it
	if _, err := readField("board_type/deprecated"); err != nil {
		log.Tracef("Failed to skip board type/deprecated field: %v", err)
	}

	// 4. Firmware Version
	if firmware, err := readField("firmware_version"); err != nil {
		log.Warnf("Failed to read firmware version: %v", err)
	} else {
		info.FirmwareVersion = strings.TrimRight(string(firmware), "\x00")
	}

	// 5. Serial Number
	if serial, err := readField("serial_number"); err != nil {
		log.Warnf("Failed to read serial number: %v", err)
	} else {
		info.SerialNumber = strings.TrimRight(string(serial), "\x00")
	}

	// 6. AP SSID
	if ssid, err := readField("ap_ssid"); err != nil {
		log.Warnf("Failed to read AP SSID: %v", err)
	} else {
		info.APSSID = strings.TrimRight(string(ssid), "\x00")
	}

	// 7. AP MAC Address
	if mac, err := readField("ap_mac"); err != nil {
		log.Warnf("Failed to read AP MAC: %v", err)
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
	m.log.Debug("Setting third party client flag")

	_, err := m.sendCommand(macAddress, CmdSetThirdPartyClient, nil)
	if err != nil {
		return fmt.Errorf("failed to set third party client: %v", err)
	}

	return nil
}

// PollUntilReady polls the camera until it's ready for operations
// It checks both hardware info availability and system ready status
func (m *Manager) PollUntilReady(macAddress string, timeout time.Duration) error {
	m.log.Info("Polling camera until ready...")

	deadline := time.Now().Add(timeout)
	attempts := 0

	for time.Now().Before(deadline) {
		attempts++

		// First check if we can get hardware info (basic connectivity test)
		info, hwErr := m.GetHardwareInfo(macAddress)
		if hwErr == nil && info != nil {
			// Now check system ready status
			response, statusErr := m.sendQuery(macAddress, QueryGetStatus, []byte{StatusSystemReady})
			if statusErr == nil && len(response.Data) >= 2 {
				// Check if system is ready (value should be 1)
				if response.Data[0] == StatusSystemReady && response.Data[1] == 1 {
					m.log.Infof("Camera is ready after %d attempts", attempts)
					return nil
				} else {
					m.log.Debugf("System not yet ready, status value: %d", response.Data[1])
				}
			} else if statusErr != nil {
				m.log.Tracef("Failed to query system ready status: %v", statusErr)
			}
		} else if hwErr != nil {
			m.log.Tracef("Hardware info not yet available: %v", hwErr)
		}

		// Log the attempt
		if attempts%5 == 0 {
			m.log.Debugf("Still waiting for camera to be ready (attempt %d)", attempts)
		}

		// Wait before retrying with increasing backoff
		backoff := time.Duration(500+min(attempts*100, 1000)) * time.Millisecond
		time.Sleep(backoff)
	}

	return fmt.Errorf("camera did not become ready within %v", timeout)
}