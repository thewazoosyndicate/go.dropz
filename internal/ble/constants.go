package ble

import (
	"time"
)

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

	// Control & Query Characteristics (FEA6)
	CharCommand          = "b5f90072-aa8d-11e3-9046-0002a5d5c51b" // GP-0072
	CharCommandResponse  = "b5f90073-aa8d-11e3-9046-0002a5d5c51b" // GP-0073
	CharSettings         = "b5f90074-aa8d-11e3-9046-0002a5d5c51b" // GP-0074
	CharSettingsResponse = "b5f90075-aa8d-11e3-9046-0002a5d5c51b" // GP-0075
	CharQuery            = "b5f90076-aa8d-11e3-9046-0002a5d5c51b" // GP-0076
	CharQueryResponse    = "b5f90077-aa8d-11e3-9046-0002a5d5c51b" // GP-0077

	// Network Management Characteristics (GP-0090)
	CharNetworkMgmtCommand  = "b5f90091-aa8d-11e3-9046-0002a5d5c51b" // GP-0091
	CharNetworkMgmtResponse = "b5f90092-aa8d-11e3-9046-0002a5d5c51b" // GP-0092

	// Discovery UUID for scanning
	AdvertisementService = "0000fea6-0000-1000-8000-00805f9b34fb" // Same as ServiceControl

	// Command IDs (OpenGoPro specification)
	CmdSleep            = 0x05
	CmdSetDateTime      = 0x0D // Set date/time (no timezone)
	CmdSetLocalDateTime = 0x0F // Set date/time with timezone + DST
	CmdSetAPControl     = 0x17 // Control WiFi Access Point
	CmdGetHardwareInfo  = 0x3C
	// Keep-alive is a SETTING write (LED=66) to GP-0074, not a command.
	// See OpenGoPro state_management: "Set Setting 91 (LED) to 66".
	SettingKeepAlive     = 0x5B
	KeepAliveValue       = 0x42 // LED special value 66
	CmdSetThirdPartyInfo = 0x50 // SET_THIRD_PARTY_CLIENT_INFO per OpenGoPro id_tables

	// Query IDs
	QueryGetStatus               = 0x13
	QueryRegisterStatusUpdates   = 0x53
	QueryUnregisterStatusUpdates = 0x73

	// Async notification IDs (unsolicited push from camera)
	AsyncStatusNotification = 0x93

	// Status IDs
	StatusSDCardStatus      = 33
	StatusNumTotalPhotos    = 38
	StatusNumTotalVideos    = 39
	StatusSDCardRemainingKB = 54
	StatusBatteryPercentage = 70
)

const (
	// Connection and Discovery Timeouts
	ServiceDiscoveryTimeout = 10 * time.Second
	ServiceDiscoveryRetries = 3
	PostConnectDelay        = 500 * time.Millisecond // Connection stabilization delay
	ScanToConnectDelay      = 200 * time.Millisecond // Delay after stopping scan before connecting
	CharDiscoveryTimeout    = 5 * time.Second
	PairingTimeout          = 15 * time.Second
	CameraReadyTimeout      = 10 * time.Second
)

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
	case CharNetworkMgmtCommand:
		return "Network Mgmt Command (GP-0091)"
	case CharNetworkMgmtResponse:
		return "Network Mgmt Response (GP-0092)"
	default:
		return "Unknown Characteristic"
	}
}
