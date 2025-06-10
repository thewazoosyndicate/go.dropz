package ble

import "fmt"

// OpenGoPro BLE Constants - Verified against official specification
// Reference: https://gopro.github.io/OpenGoPro/ble/protocol/ble_setup.html

const (
	// Service UUIDs (OpenGoPro BLE Specification)
	ServiceWifiAP     = "b5f90001-aa8d-11e3-9046-0002a5d5c51b" // GP-0001: WiFi Access Point
	ServiceControl    = "0000fea6-0000-1000-8000-00805f9b34fb" // FEA6: Control & Query
	ServiceCameraMgmt = "b5f90090-aa8d-11e3-9046-0002a5d5c51b" // GP-0090: Camera Management

	// WiFi Access Point Characteristics (GP-0001)
	CharWifiSSID     = "b5f90002-aa8d-11e3-9046-0002a5d5c51b" // GP-0002
	CharWifiPassword = "b5f90003-aa8d-11e3-9046-0002a5d5c51b" // GP-0003
	CharWifiPower    = "b5f90004-aa8d-11e3-9046-0002a5d5c51b" // GP-0004
	CharWifiState    = "b5f90005-aa8d-11e3-9046-0002a5d5c51b" // GP-0005

	// Control & Query Characteristics (FEA6)
	CharCommand          = "b5f90072-aa8d-11e3-9046-0002a5d5c51b" // GP-0072
	CharCommandResponse  = "b5f90073-aa8d-11e3-9046-0002a5d5c51b" // GP-0073
	CharSettings         = "b5f90074-aa8d-11e3-9046-0002a5d5c51b" // GP-0074
	CharSettingsResponse = "b5f90075-aa8d-11e3-9046-0002a5d5c51b" // GP-0075
	CharQuery            = "b5f90076-aa8d-11e3-9046-0002a5d5c51b" // GP-0076
	CharQueryResponse    = "b5f90077-aa8d-11e3-9046-0002a5d5c51b" // GP-0077

	// Camera Management Characteristics (GP-0090)
	CharNetworkCommand  = "b5f90091-aa8d-11e3-9046-0002a5d5c51b" // GP-0091
	CharNetworkResponse = "b5f90092-aa8d-11e3-9046-0002a5d5c51b" // GP-0092

	// Discovery UUID for scanning
	AdvertisementService = "0000fea6-0000-1000-8000-00805f9b34fb" // Same as ServiceControl

	// Command IDs (OpenGoPro specification)
	CmdSetShutter      = 0x01
	CmdSleep           = 0x05
	CmdSetDateTime     = 0x0D
	CmdGetDateTime     = 0x0E
	CmdGetHardwareInfo = 0x3C
	CmdLoadPreset      = 0x40
	CmdKeepAlive       = 0x5B

	// Query IDs
	QueryGetSettings = 0x12
	QueryGetStatus   = 0x13

	// Status IDs
	StatusBatteryLevel = 2
	StatusPairingState = 19
	StatusSystemReady  = 82

	// Pairing States
	PairingNotPaired  = 0
	PairingInProgress = 1
	PairingFailed     = 2
	PairingCancelled  = 3
	PairingCompleted  = 4

	// BLE Packet Constants
	PacketMaxSize = 20
	HeaderStart   = 0x20 // Bit 5 set for start packet
	HeaderCont    = 0x80 // Bit 7 set for continuation
	LengthMask    = 0x1F // Bits 0-4 for length
)

// UUID helper functions for OpenGoPro BLE specification compliance

// IsGoProUUID checks if a UUID follows the GoPro UUID format
// GoPro UUIDs follow the pattern: b5f9XXXX-aa8d-11e3-9046-0002a5d5c51b
func IsGoProUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}

	// Check if it matches the GoPro pattern
	return uuid[:4] == "b5f9" && uuid[8:] == "-aa8d-11e3-9046-0002a5d5c51b"
}

// IsControlServiceUUID checks if a UUID is the OpenGoPro Control & Query service
func IsControlServiceUUID(uuid string) bool {
	return uuid == ServiceControl || uuid == "fea6" || uuid == "0xfea6"
}

// GetCharacteristicName returns the human-readable name for a characteristic UUID
func GetCharacteristicName(uuid string) string {
	switch uuid {
	case CharCommand:
		return "Command Request (GP-0072)"
	case CharCommandResponse:
		return "Command Response (GP-0073)"
	case CharSettings:
		return "Settings Request (GP-0074)"
	case CharSettingsResponse:
		return "Settings Response (GP-0075)"
	case CharQuery:
		return "Query Request (GP-0076)"
	case CharQueryResponse:
		return "Query Response (GP-0077)"
	case CharWifiSSID:
		return "WiFi SSID (GP-0002)"
	case CharWifiPassword:
		return "WiFi Password (GP-0003)"
	case CharWifiPower:
		return "WiFi Power (GP-0004)"
	case CharWifiState:
		return "WiFi State (GP-0005)"
	case CharNetworkCommand:
		return "Network Management Command (GP-0091)"
	case CharNetworkResponse:
		return "Network Management Response (GP-0092)"
	default:
		return "Unknown Characteristic"
	}
}

// GetServiceName returns the human-readable name for a service UUID
func GetServiceName(uuid string) string {
	switch uuid {
	case ServiceWifiAP:
		return "WiFi Access Point Service (GP-0001)"
	case ServiceControl:
		return "Control & Query Service (FEA6)"
	case ServiceCameraMgmt:
		return "Camera Management Service (GP-0090)"
	default:
		return "Unknown Service"
	}
}

// ValidateOpenGoProUUIDs validates that all required UUIDs are properly formatted
func ValidateOpenGoProUUIDs() error {
	requiredUUIDs := map[string]string{
		"Control Service":   ServiceControl,
		"WiFi Service":      ServiceWifiAP,
		"Command Request":   CharCommand,
		"Command Response":  CharCommandResponse,
		"Settings Request":  CharSettings,
		"Settings Response": CharSettingsResponse,
		"Query Request":     CharQuery,
		"Query Response":    CharQueryResponse,
	}

	for name, uuid := range requiredUUIDs {
		if uuid == "" {
			return fmt.Errorf("UUID for %s is empty", name)
		}
		if len(uuid) != 36 {
			return fmt.Errorf("UUID for %s has invalid length: %d", name, len(uuid))
		}
	}

	return nil
}
