package ble

import (
	"context"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// ConnectionState represents device connection state
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReady
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReady:
		return "ready"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

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

// DeviceConnection holds connection data for a device
type DeviceConnection struct {
	mutex        sync.RWMutex
	device       *bluetooth.Device
	state        ConnectionState
	services     map[string]bluetooth.DeviceService
	chars        map[string]bluetooth.DeviceCharacteristic
	lastActivity time.Time
	responses    chan Response
}

// NewDeviceConnection creates a new device connection
func NewDeviceConnection() *DeviceConnection {
	return &DeviceConnection{
		services:  make(map[string]bluetooth.DeviceService),
		chars:     make(map[string]bluetooth.DeviceCharacteristic),
		responses: make(chan Response, 10),
		state:     StateDisconnected,
	}
}

// SetDevice sets the bluetooth device
func (dc *DeviceConnection) SetDevice(device *bluetooth.Device) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	dc.device = device
}

// GetDevice returns the bluetooth device
func (dc *DeviceConnection) GetDevice() *bluetooth.Device {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	return dc.device
}

// SetState updates the connection state
func (dc *DeviceConnection) SetState(state ConnectionState) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	dc.state = state
	dc.lastActivity = time.Now()
}

// GetState returns the current state
func (dc *DeviceConnection) GetState() ConnectionState {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	return dc.state
}

// SetCharacteristic stores a characteristic
func (dc *DeviceConnection) SetCharacteristic(uuid string, char bluetooth.DeviceCharacteristic) {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()
	dc.chars[uuid] = char
}

// GetCharacteristic retrieves a characteristic
func (dc *DeviceConnection) GetCharacteristic(uuid string) (bluetooth.DeviceCharacteristic, bool) {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()
	char, exists := dc.chars[uuid]
	return char, exists
}

// Close closes the connection and cleans up
func (dc *DeviceConnection) Close() {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	if dc.device != nil {
		dc.device.Disconnect()
	}
	dc.state = StateDisconnected
	close(dc.responses)
}

// DeviceDiscoveryCallback is called when a device is discovered during scanning
type DeviceDiscoveryCallback func(Device)

// BLEInterface defines the interface for BLE operations used by manager package
type BLEInterface interface {
	// Scanning and discovery
	StartScanning(ctx context.Context) error
	StartScanningWithCallback(ctx context.Context, callback DeviceDiscoveryCallback) error
	StopScanning() error
	GetDiscoveredDevices() []Device

	// Connection management
	Connect(macAddress string) error
	Disconnect(macAddress string) error
	ConnectWithEnhancedPairing(macAddress string) error

	// OpenGoPro operations
	GetWifiCredentials(macAddress string) (string, string, error)
	EnableWifi(macAddress string) error
	GetBatteryLevel(macAddress string) (int, error)
	SetDateTime(macAddress string, t time.Time) error
	Sleep(macAddress string) error
	KeepAlive(macAddress string) error
	GetMetadata(macAddress string) (map[string]string, error)

	// Pairing operations
	IsPaired(macAddress string) (bool, error)
	RefreshPairingState(macAddress string) (int, error)

	// Lifecycle
	Start() error
	Stop() error
}
