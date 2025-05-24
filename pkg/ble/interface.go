package ble

import (
	"context"
	"time"
)

// BLEInterface defines the main interface for BLE operations
type BLEInterface interface {
	// Lifecycle methods
	Start() error
	Stop()

	// Scanning methods
	StartScanning(ctx context.Context) error
	GetDiscoveredDevices() []Device

	// Connection methods
	Connect(macAddress string) error
	Disconnect(macAddress string) error
	DisconnectGoPro(macAddress string) error

	// Pairing methods
	ConnectWithEnhancedPairing(macAddress string) error
	IsPaired(macAddress string) (bool, error)
	IsPairedWithVerification(macAddress string) (bool, error)
	RefreshPairingState(macAddress string) (int, error)

	// Device operations
	GetWifiCredentials(macAddress string) (string, string, error)
	EnableWifi(macAddress string) error
	GetMetadata(macAddress string) (map[string]string, error)
	GetBatteryLevel(macAddress string) (int, error)
	SetDateTime(macAddress string, t time.Time) error
	SetCameraControl(macAddress string, enabled bool) error
	KeepAlive(macAddress string) error
	Sleep(macAddress string) error
}

// Ensure Manager implements the interface
var _ BLEInterface = (*Manager)(nil)
