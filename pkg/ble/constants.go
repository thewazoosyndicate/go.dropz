package ble

// GoPro BLE constants
const (
	// Service UUIDs
	GoProWifiServiceUUID      = "b5f90001-aa8d-11e3-9046-0002a5d5c51b"
	GoProControlServiceUUID   = "0000fea6-0000-1000-8000-00805f9b34fb"
	GoProCameraManagementUUID = "b5f90090-aa8d-11e3-9046-0002a5d5c51b"

	// Characteristic UUIDs for WiFi
	WifiSSIDCharUUID     = "b5f90002-aa8d-11e3-9046-0002a5d5c51b"
	WifiPasswordCharUUID = "b5f90003-aa8d-11e3-9046-0002a5d5c51b"
	WifiPowerCharUUID    = "b5f90004-aa8d-11e3-9046-0002a5d5c51b"
	WifiStateCharUUID    = "b5f90005-aa8d-11e3-9046-0002a5d5c51b"

	// Network Management Characteristic UUIDs
	NetworkMgmtCommandCharUUID  = "b5f90091-aa8d-11e3-9046-0002a5d5c51b"
	NetworkMgmtResponseCharUUID = "b5f90092-aa8d-11e3-9046-0002a5d5c51b"

	// Characteristic UUIDs for Control & Query
	CommandCharUUID          = "b5f90072-aa8d-11e3-9046-0002a5d5c51b"
	CommandResponseCharUUID  = "b5f90073-aa8d-11e3-9046-0002a5d5c51b"
	SettingsCharUUID         = "b5f90074-aa8d-11e3-9046-0002a5d5c51b"
	SettingsResponseCharUUID = "b5f90075-aa8d-11e3-9046-0002a5d5c51b"
	QueryCharUUID            = "b5f90076-aa8d-11e3-9046-0002a5d5c51b"
	QueryResponseCharUUID    = "b5f90077-aa8d-11e3-9046-0002a5d5c51b"

	// Command IDs
	CommandSetShutter       = 0x01
	CommandSleep            = 0x05
	CommandSetDateTime      = 0x0D
	CommandGetDateTime      = 0x0E
	CommandSetLocalDateTime = 0x0F
	CommandGetLocalDateTime = 0x10
	CommandRebootCamera     = 0x11
	CommandSetAPControl     = 0x17
	CommandHilightMoment    = 0x18
	CommandGetHardwareInfo  = 0x3C
	CommandLoadPresetGroup  = 0x3E
	CommandLoadPreset       = 0x40
	CommandSetAnalytics     = 0x50
	CommandGetOpenGoProVer  = 0x51
	CommandKeepAlive        = 0x5B

	// Query IDs (only official ones from OpenGoPro spec)
	QueryGetSettingValues            = 0x12
	QueryGetStatusValues             = 0x13
	QueryGetSettingCapabilities      = 0x32
	QueryRegisterSettingUpdates      = 0x52
	QueryRegisterStatusUpdates       = 0x53
	QueryRegisterCapabilityUpdates   = 0x62
	QueryUnregisterSettingUpdates    = 0x72
	QueryUnregisterStatusUpdates     = 0x73
	QueryUnregisterCapabilityUpdates = 0x82
	QueryAsyncSettingNotification    = 0x92
	QueryAsyncStatusNotification     = 0x93
	QueryAsyncCapabilityNotification = 0xA2

	// Protobuf Command IDs (for complex commands using protobuf)
	ProtobufCommandSetCameraControl = 0x69 // RequestSetCameraControlStatus
	ProtobufCommandSetTurboActive   = 0x6B // RequestSetTurboActive
	ProtobufCommandGetLastMedia     = 0x6D // RequestGetLastCapturedMedia
	ProtobufCommandGetPresetStatus  = 0x72 // RequestGetPresetStatus

	// Settings IDs
	SettingResolution           = 0x02
	SettingFrameRate            = 0x03
	SettingFieldOfView          = 0x79
	SettingVideoStabilization   = 0x7E
	SettingVideoMode            = 0x01
	SettingPhotoMode            = 0x29
	SettingTimelapseMode        = 0x2A
	SettingBurstMode            = 0x2B
	SettingNightlapseMode       = 0x2C
	SettingVideoPerformanceMode = 0x2D
	SettingHyperSmooth          = 0x87
	SettingLinearHorizon        = 0x96
	SettingMaxLens              = 0x9E
	SettingHiLight              = 0x5A
	SettingBeep                 = 0x56
	SettingLED                  = 0x55
	SettingAutoOff              = 0x59
	SettingScreenSaver          = 0x5B
	SettingBrightness           = 0x57

	// Packet constants
	MaxPacketSize     = 20
	PacketHeaderCont  = 0x80
	PacketHeaderStart = 0x40
	PacketHeaderMask  = 0xC0
	PacketLengthMask  = 0x3F

	// Status IDs (for QueryGetStatusValues)
	StatusBatteryPresent             = 1
	StatusBatteryLevel               = 2
	StatusSystemOverheating          = 6
	StatusCameraBusy                 = 8
	StatusQuickCaptureEnabled        = 9
	StatusEncodingActive             = 10
	StatusLCDLockActive              = 11
	StatusVideoProgressCounter       = 13
	StatusWirelessEnabled            = 17
	StatusPairingState               = 19 // Primary pairing state
	StatusPairingType                = 20
	StatusPairingTime                = 21
	StatusWifiScanState              = 22
	StatusWifiScanTime               = 23
	StatusWifiProvisioningState      = 24
	StatusWirelessRemoteVersion      = 26
	StatusWirelessRemoteConnected    = 27
	StatusWirelessPairingFlags       = 28
	StatusWirelessClientSSID         = 29
	StatusWirelessAPSSID             = 30
	StatusWirelessConnectedDevices   = 31
	StatusPreviewStreamEnabled       = 32
	StatusPhotosRemaining            = 34
	StatusVideoTimeRemaining         = 35
	StatusTotalPhotos                = 38
	StatusTotalVideos                = 39
	StatusOTAStatus                  = 41
	StatusOTACancelRequest           = 42
	StatusLocateCameraActive         = 45
	StatusTimelapseCountdown         = 49
	StatusSDCardSpaceRemaining       = 54
	StatusPreviewStreamSupported     = 55
	StatusWifiSignalStrength         = 56
	StatusHilightsCount              = 58
	StatusHilightsTime               = 59
	StatusMinStatusUpdateTime        = 60
	StatusGPSLock                    = 68
	StatusAPModeEnabled              = 69
	StatusBatteryPercentage          = 70
	StatusDigitalZoomLevel           = 75
	StatusDigitalZoomAvailable       = 77
	StatusMobileFriendlyVideo        = 78
	StatusFirstTimeUseFlow           = 79
	StatusWifi5GHzAvailable          = 81
	StatusSystemReady                = 82
	StatusBatteryOKForOTA            = 83
	StatusSystemTooHot               = 85
	StatusCameraOrientation          = 86
	StatusZoomWhileEncoding          = 88
	StatusCurrentFlatmode            = 89
	StatusCurrentVideoPreset         = 93
	StatusCurrentPhotoPreset         = 94
	StatusCurrentTimelapsePreset     = 95
	StatusCurrentPresetGroup         = 96
	StatusCurrentPreset              = 97
	StatusPresetModified             = 98
	StatusLiveBurstsRemaining        = 99
	StatusTotalLiveBursts            = 100
	StatusCaptureDelayActive         = 101
	StatusLinuxCoreActive            = 104
	StatusCameraLensType             = 105
	StatusVideoHindsightActive       = 106
	StatusScheduledCaptureSet        = 108
	StatusBandwidthTest              = 110
	StatusSDCardWriteSpeedError      = 111
	StatusSDCardWriteSpeedErrorCount = 112
	StatusTurboTransferActive        = 113
	StatusCameraControlStatus        = 114
	StatusUSBConnected               = 115
	StatusCameraControlUSB           = 116
	StatusSDCardCapacity             = 117

	// Pairing State constants (Status ID 19)
	PairingStateNeverStarted = 0
	PairingStateStarted      = 1
	PairingStateAborted      = 2
	PairingStateCancelled    = 3
	PairingStateCompleted    = 4

	// BLE packet handling constants
	GeneralPurposeCommandHeader = 0x10
	ExtendedCommandHeader       = 0x90
	KeepAliveCommandHeader      = 0xA0
	PacketTypeStart             = 0x10
	PacketTypeContinuation      = 0x00
)
