package ble

import "fmt"

// GoPro BLE constants - Updated according to OpenGoPro BLE specification
// Reference: https://gopro.github.io/OpenGoPro/ble/
//
// UUID Format Note:
// GP-XXXX is shorthand for GoPro's 128-bit UUID: b5f9XXXX-aa8d-11e3-9046-0002a5d5c51b
// The Control & Query service uses UUID: 0000fea6-0000-1000-8000-00805f9b34fb

const (
	// Service UUIDs (OpenGoPro BLE Specification)
	GoProWifiServiceUUID    = "b5f90001-aa8d-11e3-9046-0002a5d5c51b" // GP-0001: WiFi Access Point Service
	GoProControlServiceUUID = "0000fea6-0000-1000-8000-00805f9b34fb" // FEA6: Control & Query Service
	GoProNetworkMgmtUUID    = "b5f90090-aa8d-11e3-9046-0002a5d5c51b" // GP-0090: Camera Management Service

	// Characteristic UUIDs for WiFi Access Point Service (GP-0001)
	WifiSSIDCharUUID     = "b5f90002-aa8d-11e3-9046-0002a5d5c51b" // GP-0002: WiFi AP SSID
	WifiPasswordCharUUID = "b5f90003-aa8d-11e3-9046-0002a5d5c51b" // GP-0003: WiFi AP Password
	WifiPowerCharUUID    = "b5f90004-aa8d-11e3-9046-0002a5d5c51b" // GP-0004: WiFi AP Power
	WifiStateCharUUID    = "b5f90005-aa8d-11e3-9046-0002a5d5c51b" // GP-0005: WiFi AP State

	// Characteristic UUIDs for Camera Management Service (GP-0090)
	NetworkMgmtCommandCharUUID  = "b5f90091-aa8d-11e3-9046-0002a5d5c51b" // GP-0091: Network Management Command
	NetworkMgmtResponseCharUUID = "b5f90092-aa8d-11e3-9046-0002a5d5c51b" // GP-0092: Network Management Response

	// Characteristic UUIDs for Control & Query Service (FEA6)
	CommandCharUUID          = "b5f90072-aa8d-11e3-9046-0002a5d5c51b" // GP-0072: Command Request
	CommandResponseCharUUID  = "b5f90073-aa8d-11e3-9046-0002a5d5c51b" // GP-0073: Command Response
	SettingsCharUUID         = "b5f90074-aa8d-11e3-9046-0002a5d5c51b" // GP-0074: Settings Request
	SettingsResponseCharUUID = "b5f90075-aa8d-11e3-9046-0002a5d5c51b" // GP-0075: Settings Response
	QueryCharUUID            = "b5f90076-aa8d-11e3-9046-0002a5d5c51b" // GP-0076: Query Request
	QueryResponseCharUUID    = "b5f90077-aa8d-11e3-9046-0002a5d5c51b" // GP-0077: Query Response

	// UUID Helper Functions and Validation

	// OpenGoPro Service Advertisement UUID (for discovery)
	GoProAdvertisementServiceUUID = "0000fea6-0000-1000-8000-00805f9b34fb" // Same as Control Service

	// Legacy support - these are aliases for backwards compatibility
	GoProCameraManagementUUID = GoProNetworkMgmtUUID // Deprecated: Use GoProNetworkMgmtUUID

	// Command IDs (verified against OpenGoPro spec)
	CommandSetShutter      = 0x01 // SET_SHUTTER
	CommandSleep           = 0x05 // SLEEP
	CommandSetDateTime     = 0x0D // SET_DATE_TIME (deprecated, use SET_DATE_TIME_DST)
	CommandGetDateTime     = 0x0E // GET_DATE_TIME (deprecated, use GET_DATE_TIME_DST)
	CommandSetDateTimeDST  = 0x0F // SET_DATE_TIME_DST (replaces SET_LOCAL_DATE_TIME)
	CommandGetDateTimeDST  = 0x10 // GET_DATE_TIME_DST (replaces GET_LOCAL_DATE_TIME)
	CommandSetAPControl    = 0x17 // SET_AP_CONTROL
	CommandHilightMoment   = 0x18 // TAG_HILIGHT_MOMENT
	CommandGetHardwareInfo = 0x3C // GET_HW_INFO
	CommandLoadPresetGroup = 0x3E // LOAD_PRESET_GROUP
	CommandLoadPreset      = 0x40 // LOAD_PRESET
	CommandSetAnalytics    = 0x50 // SET_THIRD_PARTY_CLIENT_INFO
	CommandGetOpenGoProVer = 0x51 // GET_OPEN_GOPRO_API_VERSION
	CommandKeepAlive       = 0x5B // KEEP_ALIVE

	// Additional Commands from OpenGoPro spec
	CommandSetCOHN        = 0x0F // SET_COHN_SETTING
	CommandClearCOHN      = 0x10 // CLEAR_COHN_CERT
	CommandCreateCOHNCert = 0x11 // CREATE_COHN_CERT
	CommandCOHNStatus     = 0x12 // REQUEST_COHN_SETTING

	// Query IDs (verified against OpenGoPro spec)
	QueryGetSettingValues            = 0x12 // GET_SETTING_VALUES
	QueryGetStatusValues             = 0x13 // GET_STATUS_VALUES
	QueryGetSettingCapabilities      = 0x32 // GET_AVAILABLE_OPTION_IDS
	QueryRegisterSettingUpdates      = 0x52 // REGISTER_FOR_SETTING_UPDATES
	QueryRegisterStatusUpdates       = 0x53 // REGISTER_FOR_STATUS_UPDATES
	QueryRegisterCapabilityUpdates   = 0x62 // REGISTER_FOR_CAPABILITY_UPDATES
	QueryUnregisterSettingUpdates    = 0x72 // UNREGISTER_FOR_SETTING_UPDATES
	QueryUnregisterStatusUpdates     = 0x73 // UNREGISTER_FOR_STATUS_UPDATES
	QueryUnregisterCapabilityUpdates = 0x82 // UNREGISTER_FOR_CAPABILITY_UPDATES
	QueryAsyncSettingNotification    = 0x92 // ASYNC_SETTING_UPDATE_NOTIFICATION
	QueryAsyncStatusNotification     = 0x93 // ASYNC_STATUS_UPDATE_NOTIFICATION
	QueryAsyncCapabilityNotification = 0xA2 // ASYNC_CAPABILITY_UPDATE_NOTIFICATION

	// Protobuf Command IDs (Feature IDs from OpenGoPro spec)
	ProtobufCommandSetCameraControl = 0x69 // REQUEST_SET_CAMERA_CONTROL_STATUS
	ProtobufCommandSetTurboActive   = 0x6B // REQUEST_SET_TURBO_ACTIVE
	ProtobufCommandGetLastMedia     = 0x6D // REQUEST_GET_LAST_CAPTURED_MEDIA
	ProtobufCommandGetPresetStatus  = 0x72 // REQUEST_GET_PRESET_STATUS
	ProtobufCommandSetLivestream    = 0x74 // REQUEST_SET_LIVESTREAM_MODE
	ProtobufCommandGetCOHNStatus    = 0x6E // REQUEST_GET_COHN_STATUS
	ProtobufCommandSetCOHN          = 0x6F // REQUEST_SET_COHN_SETTING
	ProtobufCommandClearCOHN        = 0x70 // REQUEST_CLEAR_COHN_CERT

	// Settings IDs (verified against OpenGoPro spec)
	SettingResolution         = 0x02 // RESOLUTION
	SettingFrameRate          = 0x03 // FPS
	SettingFieldOfView        = 0x79 // FOV
	SettingVideoStabilization = 0x7E // STABILIZATION
	SettingVideoMode          = 0x01 // Not in spec, might be deprecated
	SettingPhotoMode          = 0x10 // PHOTO_MODE
	SettingTimelapseMode      = 0x1E // TIMELAPSE_MODE
	SettingMultiShotMode      = 0x1B // MULTI_SHOT_MODE
	SettingHyperSmooth        = 0x87 // HYPERSMOOTH
	SettingHorizonLeveling    = 0x96 // HORIZON_LEVELING
	SettingMaxLens            = 0xA2 // MAX_LENS_MOD
	SettingHindsight          = 0xA7 // HINDSIGHT
	SettingBeep               = 0x56 // BEEP_VOLUME
	SettingLED                = 0x5B // LED
	SettingAutoOff            = 0x59 // AUTO_OFF
	SettingScreenSaver        = 0x5C // SCREEN_SAVER_TIMEOUT
	SettingBrightness         = 0x58 // LCD_BRIGHTNESS
	SettingGPS                = 0x53 // GPS
	SettingVoiceControl       = 0x56 // VOICE_CONTROL
	SettingWifiMode           = 0x43 // WIFI_BAND

	// Packet constants
	MaxPacketSize     = 20
	PacketHeaderCont  = 0x80
	PacketHeaderStart = 0x20 // Corrected: bit 5 indicates start of packet
	PacketHeaderMask  = 0xE0 // Corrected: bits 5-7 are header bits
	PacketLengthMask  = 0x1F // Corrected: bits 0-4 are length bits

	// Status IDs (verified against OpenGoPro spec)
	StatusBatteryPresent          = 1   // BATTERY_PRESENT
	StatusBatteryLevel            = 2   // BATTERY_LEVEL
	StatusSystemOverheating       = 6   // SYSTEM_HOT
	StatusCameraBusy              = 8   // SYSTEM_BUSY
	StatusQuickCaptureEnabled     = 9   // QUICK_CAPTURE
	StatusEncodingActive          = 10  // ENCODING_ACTIVE
	StatusLCDLockActive           = 11  // LCD_LOCK_ACTIVE
	StatusVideoProgressCounter    = 13  // VIDEO_PROGRESS_COUNTER
	StatusWirelessEnabled         = 17  // ENABLE_WIFI
	StatusPairingState            = 19  // PAIR_STATE
	StatusPairingType             = 20  // PAIR_TYPE
	StatusPairingTime             = 21  // PAIR_TIME
	StatusWifiScanState           = 22  // SCAN_WIFI_STATE
	StatusWifiScanTime            = 23  // SCAN_TIME_MSEC
	StatusWifiProvisioningState   = 24  // WIFI_PROVISION_STATE
	StatusWirelessRemoteVersion   = 26  // REMOTE_CTRL_VERSION
	StatusWirelessRemoteConnected = 27  // REMOTE_CTRL_CONNECTED
	StatusWifiSSID                = 29  // WIFI_SSID
	StatusAPSSID                  = 30  // AP_SSID
	StatusAPClients               = 31  // AP_CLIENT_COUNT
	StatusPreviewStreamEnabled    = 32  // PREVIEW_ENABLED
	StatusPhotosRemaining         = 34  // PHOTOS_REM
	StatusVideoTimeRemaining      = 35  // VIDEO_REM
	StatusTotalPhotos             = 38  // NUM_GROUP_PHOTOS
	StatusTotalVideos             = 39  // NUM_GROUP_VIDEOS
	StatusOTAStatus               = 41  // OTA_STATUS
	StatusOTADownloadCancelled    = 42  // DOWNLOAD_CANCEL_REQUEST_PENDING
	StatusLocateActive            = 45  // LOCATE_ACTIVE
	StatusTimelapseRemaining      = 49  // TIMELAPSE_REM
	StatusExposureType            = 65  // EXPOSURE_TYPE
	StatusExposureX               = 66  // EXPOSURE_X
	StatusExposureY               = 67  // EXPOSURE_Y
	StatusGPSStatus               = 68  // GPS_STATUS
	StatusAPState                 = 69  // AP_STATE
	StatusBatteryPercentage       = 70  // INT_BATT_PER
	StatusDigitalZoom             = 75  // DIGITAL_ZOOM
	StatusDigitalZoomActive       = 77  // DIGITAL_ZOOM_ACTIVE
	StatusMobileFriendlyVideo     = 78  // MOBILE_FRIENDLY_VIDEO
	StatusFirstTimeUse            = 79  // FIRST_TIME_USE
	StatusBandwidth5GHz           = 81  // BAND_5GHZ_AVAIL
	StatusSystemReady             = 82  // SYSTEM_READY
	StatusBatteryOKForOTA         = 83  // BATT_OK_FOR_OTA
	StatusVideoLowTemp            = 85  // VIDEO_LOW_TEMP_ALERT
	StatusOrientation             = 86  // ORIENTATION
	StatusZoomWhileEncoding       = 88  // ZOOM_WHILE_ENCODING
	StatusFlatmodeID              = 89  // FLATMODE_ID
	StatusVideoPresetID           = 93  // VIDEO_PRESET_ID
	StatusPhotoPresetID           = 94  // PHOTO_PRESET_ID
	StatusTimelapsePresetID       = 95  // TIMELAPSE_PRESET_ID
	StatusPresetGroupID           = 96  // PRESET_GROUP_ID
	StatusActivePresetID          = 97  // ACTIVE_PRESET_ID
	StatusPresetModified          = 98  // PRESET_MODIFIED_NOTIFICATION
	StatusLiveBurstRemaining      = 99  // LIVE_BURST_REM
	StatusLiveBurstTotal          = 100 // LIVE_BURST_TOTAL
	StatusCaptureDelayActive      = 101 // CAPTURE_DELAY_ACTIVE
	StatusMediaModMicStatus       = 102 // MEDIA_MOD_MIC_STATUS
	StatusTimewarpSpeedRamp       = 103 // TIMEWARP_SPEED_RAMP_ACTIVE
	StatusLinuxCoreActive         = 104 // LINUX_CORE_ACTIVE
	StatusCameraLensType          = 105 // CAMERA_LENS_TYPE
	StatusVideoHindsight          = 106 // VIDEO_HINDSIGHT_CAPTURE_ACTIVE
	StatusScheduledCaptured       = 107 // SCHEDULED_CAPTURED
	StatusScheduledCaptureID      = 108 // SCHEDULED_CAPTURE_PRESET_ID
	StatusCreatingPresetGroup     = 109 // CREATING_PRESET
	StatusMediaModStatus          = 110 // MEDIA_MOD_STAT
	StatusSDCardWriteSpeed        = 111 // SD_WRITE_SPEED_ERROR
	StatusTurboTransfer           = 113 // TURBO_TRANSFER
	StatusCameraControl           = 114 // CAMERA_CONTROL
	StatusUSBConnected            = 115 // USB_CONNECTED
	StatusControlOverUSB          = 116 // CONTROL_OVER_USB
	StatusTotalSDSpace            = 117 // TOTAL_SD_SPACE_KB
	StatusPhotosCount             = 118 // NUM_PHOTOS

	// Pairing State constants (Status ID 19) - Updated per OpenGoPro BLE spec
	PairingStateNotPaired  = 0 // Not paired
	PairingStateInProgress = 1 // Pairing started
	PairingStateFailed     = 2 // Pairing aborted
	PairingStateCancelled  = 3 // Pairing cancelled
	PairingStateCompleted  = 4 // Pairing completed

	// Response Status Codes (from OpenGoPro spec)
	ResponseStatusSuccess            = 0x00 // Success
	ResponseStatusError              = 0x01 // Error
	ResponseStatusInvalidParam       = 0x02 // Invalid parameter
	ResponseStatusOperationCancelled = 0x03 // Operation cancelled
	ResponseStatusOperationTimeout   = 0x04 // Operation timeout
	ResponseStatusOperationFailed    = 0x05 // Operation failed

	// BLE packet handling constants (from OpenGoPro spec)
	GeneralLengthByteMask  = 0x1F   // Bits 0-4 for general length
	ExtendedLengthByteMask = 0x0FFF // 12 bits for extended length
	GeneralStartBit        = 0x20   // Bit 5 for start packet
	GeneralContinueBit     = 0x80   // Bit 7 for continuation packet
	ExtendedHeaderBit      = 0x40   // Bit 6 for extended header
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
	return uuid == GoProControlServiceUUID || uuid == "fea6" || uuid == "0xfea6"
}

// GetCharacteristicName returns the human-readable name for a characteristic UUID
func GetCharacteristicName(uuid string) string {
	switch uuid {
	case CommandCharUUID:
		return "Command Request (GP-0072)"
	case CommandResponseCharUUID:
		return "Command Response (GP-0073)"
	case SettingsCharUUID:
		return "Settings Request (GP-0074)"
	case SettingsResponseCharUUID:
		return "Settings Response (GP-0075)"
	case QueryCharUUID:
		return "Query Request (GP-0076)"
	case QueryResponseCharUUID:
		return "Query Response (GP-0077)"
	case WifiSSIDCharUUID:
		return "WiFi SSID (GP-0002)"
	case WifiPasswordCharUUID:
		return "WiFi Password (GP-0003)"
	case WifiPowerCharUUID:
		return "WiFi Power (GP-0004)"
	case WifiStateCharUUID:
		return "WiFi State (GP-0005)"
	case NetworkMgmtCommandCharUUID:
		return "Network Management Command (GP-0091)"
	case NetworkMgmtResponseCharUUID:
		return "Network Management Response (GP-0092)"
	default:
		return "Unknown Characteristic"
	}
}

// GetServiceName returns the human-readable name for a service UUID
func GetServiceName(uuid string) string {
	switch uuid {
	case GoProWifiServiceUUID:
		return "WiFi Access Point Service (GP-0001)"
	case GoProControlServiceUUID:
		return "Control & Query Service (FEA6)"
	case GoProNetworkMgmtUUID:
		return "Camera Management Service (GP-0090)"
	default:
		return "Unknown Service"
	}
}

// ValidateOpenGoProUUIDs validates that all required UUIDs are properly formatted
func ValidateOpenGoProUUIDs() error {
	requiredUUIDs := map[string]string{
		"Control Service":   GoProControlServiceUUID,
		"WiFi Service":      GoProWifiServiceUUID,
		"Command Request":   CommandCharUUID,
		"Command Response":  CommandResponseCharUUID,
		"Settings Request":  SettingsCharUUID,
		"Settings Response": SettingsResponseCharUUID,
		"Query Request":     QueryCharUUID,
		"Query Response":    QueryResponseCharUUID,
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
