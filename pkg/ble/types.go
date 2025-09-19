package ble

import (
	"time"
)


// Device represents a GoPro BLE device
type Device struct {
	Name            string    `json:"name"`
	MACAddress      string    `json:"mac_address"`
	RSSI            int32     `json:"rssi"`
	IsConnected     bool      `json:"is_connected"`
	ModelID         int       `json:"model_id,omitempty"`
	ModelName       string    `json:"model_name,omitempty"`
	FirmwareVersion string    `json:"firmware_version,omitempty"`
	SerialNumber    string    `json:"serial_number,omitempty"`
	WiFiSSID        string    `json:"wifi_ssid,omitempty"`
	WiFiPassword    string    `json:"wifi_password,omitempty"`
	PairingState    int       `json:"pairing_state"`
	LastSeen        time.Time `json:"last_seen"`
}

// Response represents a BLE response from the device
type Response struct {
	CommandID byte      `json:"command_id"`
	Status    byte      `json:"status"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}


// DeviceDiscoveryCallback is called when a device is discovered during scanning
type DeviceDiscoveryCallback func(Device)
