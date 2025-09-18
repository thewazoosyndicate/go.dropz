package ble

import (
	"fmt"
	"strings"
	"time"

	"github.com/dropz/dropz/pkg/logger"
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

// parseHardwareInfo parses the TLV response from GetHardwareInfo command
func parseHardwareInfo(data []byte, log logger.Logger) (*HardwareInfo, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("hardware info response too short: %d bytes", len(data))
	}

	info := &HardwareInfo{}
	offset := 0

	// Parse TLV fields
	for offset < len(data) {
		if offset+2 > len(data) {
			break // Not enough data for type and length
		}

		fieldType := data[offset]
		fieldLen := int(data[offset+1])
		offset += 2

		if offset+fieldLen > len(data) {
			log.Warnf("Hardware info field %d truncated: expected %d bytes, have %d",
				fieldType, fieldLen, len(data)-offset)
			break
		}

		fieldData := data[offset : offset+fieldLen]
		offset += fieldLen

		// Parse based on field type
		switch fieldType {
		case 0x01: // Model Number
			if fieldLen >= 4 {
				info.ModelNumber = int(fieldData[0])<<24 | int(fieldData[1])<<16 |
					int(fieldData[2])<<8 | int(fieldData[3])
			}
		case 0x02: // Model Name
			info.ModelName = strings.TrimRight(string(fieldData), "\x00")
		case 0x03: // Firmware Version
			info.FirmwareVersion = strings.TrimRight(string(fieldData), "\x00")
		case 0x04: // Serial Number
			info.SerialNumber = strings.TrimRight(string(fieldData), "\x00")
		case 0x05: // AP SSID
			info.APSSID = strings.TrimRight(string(fieldData), "\x00")
		case 0x06: // MAC Address
			if fieldLen == 6 {
				info.MACAddress = fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
					fieldData[0], fieldData[1], fieldData[2],
					fieldData[3], fieldData[4], fieldData[5])
			}
		default:
			log.Tracef("Unknown hardware info field type: 0x%02X", fieldType)
		}
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