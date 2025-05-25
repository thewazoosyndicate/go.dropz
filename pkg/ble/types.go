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

// PendingRequest represents a request waiting for a response
type PendingRequest struct {
	CommandID    byte
	ResponseChan chan *QueryResponseData
	Timeout      time.Duration
	StartTime    time.Time
	IsQuery      bool // true for queries, false for commands
}

// ResponseFragment represents a fragment of a fragmented response
type ResponseFragment struct {
	SequenceNumber int
	Data           []byte
	IsLast         bool
}

// ResponseTracker tracks the response for a specific query with request correlation
// Thread Safety: All fields are protected by RWMutex for concurrent access
// - lastResponse, ModelID, PairingState: Protected for atomic reads/writes
// - pendingRequests, fragmentBuffer: Protected for concurrent request management
type ResponseTracker struct {
	mutex           sync.RWMutex                 // Protects all fields in this struct
	lastResponse    *QueryResponseData           // Last response received (legacy compatibility)
	responseChannel chan *QueryResponseData      // Legacy channel for backward compatibility
	ModelID         int                          // Stores the last detected model ID (if any)
	PairingState    int                          // Stores the last detected pairing state (if any)
	pendingRequests map[byte]*PendingRequest     // CommandID -> PendingRequest (protected by mutex)
	fragmentBuffer  map[byte][]*ResponseFragment // CommandID -> ordered fragments (protected by mutex)
}

// QueryResponseData holds the data for a query response
type QueryResponseData struct {
	QueryID      byte
	Status       byte
	Data         []byte
	ResponseTime time.Time
	IsFragmented bool
	TotalSize    int // For fragmented responses
}
