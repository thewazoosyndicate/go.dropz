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
	CommandSetDateTime      = 0x03
	CommandSetShutter       = 0x01
	CommandSetAPControl     = 0x17
	CommandSleep            = 0x05
	CommandSetAnalytics     = 0x50
	CommandSetCameraControl = 0x3A
	CommandSetTurboActive   = 0x09

	// Query IDs
	QueryGetDateTime            = 0x03
	QueryGetWifiSSID            = 0x02
	QueryGetWifiPassword        = 0x03
	QueryGetHardwareInfo        = 0x3E
	QueryGetOpenGoPro           = 0x51
	QueryGetSoftwareVersion     = 0x50
	QueryGetCameraStatus        = 0x0E
	QueryGetBatteryLevel        = 0x23
	QueryGetStorageCapacity     = 0x36
	QueryGetWifiAP              = 0x17
	QueryGetBLERole             = 0x60
	QueryGetPairingState        = 0x3F
	QueryGetLastHiLight         = 0x14
	QueryGetAvailableStreams    = 0x16
	QueryGetCameraState         = 0x42
	QueryGetEncoding            = 0x3D
	QueryGetPresetStatus        = 0x3F
	QueryGetVersionInfo         = 0x51
	QueryGetSettingValues       = 0x12
	QueryGetStatusValues        = 0x13
	QueryGetSettingCapabilities = 0x14
	QueryGetLastCapturedMedia   = 0x16

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
