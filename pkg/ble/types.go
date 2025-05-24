package ble

import (
	"sync"
	"time"
)

// Device represents a discovered BLE GoPro device
type Device struct {
	Name        string
	MACAddress  string
	RSSI        int32
	IsConnected bool
}

// EventType represents the type of BLE event
type EventType string

const (
	// EventDeviceDiscovered is emitted when a new device is discovered
	EventDeviceDiscovered EventType = "device_discovered"
	// EventDeviceUpdated is emitted when a device's properties are updated
	EventDeviceUpdated EventType = "device_updated"
	// EventDeviceConnected is emitted when a device is connected
	EventDeviceConnected EventType = "device_connected"
	// EventDeviceDisconnected is emitted when a device is disconnected
	EventDeviceDisconnected EventType = "device_disconnected"
)

// BLEEvent represents an event in the BLE system
type BLEEvent struct {
	Type      EventType
	Device    *Device
	Timestamp time.Time
	Error     error
}

// EventHandler is a function that handles BLE events
type EventHandler func(event BLEEvent)

// ResponseTracker tracks the response for a specific query
type ResponseTracker struct {
	mutex           sync.RWMutex
	lastResponse    *QueryResponseData
	responseChannel chan *QueryResponseData
	ModelID         int // Stores the last detected model ID (if any)
	PairingState    int // Stores the last detected pairing state (if any)
}

// QueryResponseData holds the data for a query response
type QueryResponseData struct {
	QueryID      byte
	Status       byte
	Data         []byte
	ResponseTime time.Time
}
