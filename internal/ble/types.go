package ble

import "time"

// Device represents a GoPro BLE device
type Device struct {
	Name            string    `json:"name"`
	BLEAddress      string    `json:"ble_address"`
	RSSI            int32     `json:"rssi"`
	ModelID         int       `json:"model_id,omitempty"`
	ModelName       string    `json:"model_name,omitempty"`
	FirmwareVersion string    `json:"firmware_version,omitempty"`
	SerialNumber    string    `json:"serial_number,omitempty"`
	WiFiSSID        string    `json:"wifi_ssid,omitempty"`
	WiFiPassword    string    `json:"wifi_password,omitempty"`
	LastSeen        time.Time `json:"last_seen"`

	// Parsed from advertising data (OpenGoPro manufacturer/service data)
	PairingMode bool `json:"pairing_mode,omitempty"`
	NewMedia    bool `json:"new_media,omitempty"`
	// ProcessorOn is the camera's Processor State bit. A camera whose
	// processor is down still advertises — the BLE chip runs on almost
	// nothing — but cannot answer a connect. That is a flat or sleeping
	// camera, and it is indistinguishable from a refused bond without this
	// bit: both look like "advertising fine, connect times out".
	ProcessorOn bool `json:"processor_on,omitempty"`
	// AdvParsed records that a manufacturer payload was parsed at least once,
	// so ProcessorOn == false means "processor down" rather than "unknown".
	AdvParsed bool `json:"-"`
}

// Response represents a BLE response from the device
type Response struct {
	Status byte   `json:"status"`
	Data   []byte `json:"data"`
}

// DeviceDiscoveryCallback is called when a device is discovered during scanning
type DeviceDiscoveryCallback func(Device)

// CameraMetadata represents metadata about a camera
type CameraMetadata struct {
	BLEAddress      string
	ModelID         int
	ModelName       string
	FirmwareVersion string
	SerialNumber    string
	BatteryLevel    int
	WiFiSSID        string
	WiFiPassword    string
}

// MetadataUpdateFunc is called when camera metadata is updated during connection
type MetadataUpdateFunc func(metadata CameraMetadata)
