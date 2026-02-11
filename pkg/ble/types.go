package ble

import "time"

// Device represents a GoPro BLE device
type Device struct {
	Name            string    `json:"name"`
	MACAddress      string    `json:"mac_address"`
	RSSI            int32     `json:"rssi"`
	ModelID         int       `json:"model_id,omitempty"`
	ModelName       string    `json:"model_name,omitempty"`
	FirmwareVersion string    `json:"firmware_version,omitempty"`
	SerialNumber    string    `json:"serial_number,omitempty"`
	WiFiSSID        string    `json:"wifi_ssid,omitempty"`
	WiFiPassword    string    `json:"wifi_password,omitempty"`
	LastSeen        time.Time `json:"last_seen"`
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
	MACAddress      string
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
