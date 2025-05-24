package ble

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// ConnectionState represents the current state of a BLE connection
type ConnectionState int

const (
	// Connection states
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReady // Fully connected with services discovered and characteristics subscribed
	StateDisconnecting
	StateError
)

// ConnectionStateString converts a connection state to a human-readable string
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateReady:
		return "Ready"
	case StateDisconnecting:
		return "Disconnecting"
	case StateError:
		return "Error"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

// ConnectionEvent represents an event in the connection lifecycle
type ConnectionEvent struct {
	DeviceMAC string
	OldState  ConnectionState
	NewState  ConnectionState
	Timestamp time.Time
	Error     error // Optional error if state transition failed
}

// ConnectionManager manages the lifecycle of BLE connections
type ConnectionManager struct {
	adapter       *bluetooth.Adapter
	connections   map[string]*ConnectionInfo
	mutex         sync.RWMutex
	log           logger.Logger
	eventHandlers []func(ConnectionEvent)
}

// ConnectionInfo stores connection details for a device
type ConnectionInfo struct {
	Device          *bluetooth.Device
	State           ConnectionState
	LastStateChange time.Time
	LastActivity    time.Time
	Metadata        map[string]interface{}
	notifyChars     map[string]bluetooth.DeviceCharacteristic
	serviceCache    map[string]bluetooth.DeviceService
	mutex           sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(adapter *bluetooth.Adapter, log logger.Logger) *ConnectionManager {
	return &ConnectionManager{
		adapter:     adapter,
		connections: make(map[string]*ConnectionInfo),
		log:         log,
	}
}

// RegisterEventHandler registers a handler for connection events
func (cm *ConnectionManager) RegisterEventHandler(handler func(ConnectionEvent)) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.eventHandlers = append(cm.eventHandlers, handler)
}

// notifyEvent notifies all registered handlers of a connection event
func (cm *ConnectionManager) notifyEvent(event ConnectionEvent) {
	cm.mutex.RLock()
	handlers := make([]func(ConnectionEvent), len(cm.eventHandlers))
	copy(handlers, cm.eventHandlers)
	cm.mutex.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// ChangeState changes the connection state and notifies handlers
func (cm *ConnectionManager) ChangeState(macAddress string, newState ConnectionState, err error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	info, exists := cm.connections[macAddress]
	if !exists {
		if newState == StateConnecting {
			// Create a new connection info
			info = &ConnectionInfo{
				State:           StateDisconnected, // Initial state
				LastStateChange: time.Now(),
				Metadata:        make(map[string]interface{}),
				notifyChars:     make(map[string]bluetooth.DeviceCharacteristic),
				serviceCache:    make(map[string]bluetooth.DeviceService),
			}
			cm.connections[macAddress] = info
		} else {
			cm.log.Warnf("Cannot change state for non-existent connection: %s", macAddress)
			return
		}
	}

	oldState := info.State

	// Only update if state actually changes
	if oldState != newState {
		info.State = newState
		now := time.Now()
		info.LastStateChange = now

		// If connecting to "ready" state, update activity timestamp
		if newState == StateReady {
			info.LastActivity = now
		}

		// Notify handlers about the state change
		cm.notifyEvent(ConnectionEvent{
			DeviceMAC: macAddress,
			OldState:  oldState,
			NewState:  newState,
			Timestamp: now,
			Error:     err,
		})

		cm.log.Debugf("Device %s state changed: %s -> %s", macAddress, oldState, newState)

		// Cleanup on disconnect
		if newState == StateDisconnected && oldState != StateDisconnected {
			// Clear cached services and characteristics
			info.serviceCache = make(map[string]bluetooth.DeviceService)
			info.notifyChars = make(map[string]bluetooth.DeviceCharacteristic)
		}
	}
}

// GetState returns the current connection state for a device
func (cm *ConnectionManager) GetState(macAddress string) ConnectionState {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if info, exists := cm.connections[macAddress]; exists {
		return info.State
	}
	return StateDisconnected
}

// RecordActivity updates the last activity timestamp for a device
func (cm *ConnectionManager) RecordActivity(macAddress string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if info, exists := cm.connections[macAddress]; exists {
		info.LastActivity = time.Now()
	}
}

// Connect initiates a connection to a device with proper state management
func (cm *ConnectionManager) Connect(ctx context.Context, macAddress string) (*bluetooth.Device, error) {
	// Check current state
	currentState := cm.GetState(macAddress)
	if currentState == StateConnecting || currentState == StateConnected || currentState == StateReady {
		cm.log.Warnf("Device %s is already in state %s", macAddress, currentState)

		// If already connected, return the existing device
		cm.mutex.RLock()
		info := cm.connections[macAddress]
		device := info.Device
		cm.mutex.RUnlock()

		if device != nil {
			return device, nil
		}

		// If connecting but no device yet, wait for connection to complete
		if currentState == StateConnecting {
			return nil, NewBLEError("connect", macAddress, fmt.Errorf("connection in progress"), true, 1)
		}
	}

	// Set state to connecting
	cm.ChangeState(macAddress, StateConnecting, nil)

	// Attempt connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Use a channel to handle connection completion
	connectDone := make(chan struct{})
	var device *bluetooth.Device
	var connectErr error

	go func() {
		defer close(connectDone)
		// Actual connection logic depends on the bluetooth library
		// This is a placeholder for the actual connection code
		addr := bluetooth.Address{}
		// Code to parse macAddress into bluetooth.Address format goes here

		device, connectErr = cm.adapter.Connect(addr, bluetooth.ConnectionParams{})
	}()

	// Wait for connection or timeout
	select {
	case <-connectDone:
		// Connection completed (success or error)
		if connectErr != nil {
			cm.ChangeState(macAddress, StateDisconnected, connectErr)
			return nil, NewBLEError("connect", macAddress, connectErr, true, 1)
		}

		// Connection succeeded
		cm.mutex.Lock()
		info := cm.connections[macAddress]
		info.Device = device
		cm.mutex.Unlock()

		cm.ChangeState(macAddress, StateConnected, nil)
		return device, nil

	case <-connectCtx.Done():
		// Timeout or cancellation
		cm.ChangeState(macAddress, StateDisconnected, connectCtx.Err())
		return nil, NewBLEError("connect", macAddress, ErrTimeout, true, 1)
	}
}

// Disconnect properly disconnects from a device
func (cm *ConnectionManager) Disconnect(macAddress string) error {
	// ADD DETAILED DIAGNOSTIC LOGGING
	cm.log.Infof("ConnectionManager.Disconnect called for device %s (state: %s)",
		macAddress, cm.GetState(macAddress))

	currentState := cm.GetState(macAddress)
	if currentState == StateDisconnected || currentState == StateDisconnecting {
		// Already disconnected or disconnecting
		cm.log.Debugf("ConnectionManager.Disconnect: Device %s already in state %s, skipping",
			macAddress, currentState)
		return nil
	}

	cm.ChangeState(macAddress, StateDisconnecting, nil)

	// Get the device
	cm.mutex.RLock()
	info, exists := cm.connections[macAddress]
	if !exists || info.Device == nil {
		cm.mutex.RUnlock()
		cm.log.Debugf("ConnectionManager.Disconnect: Device %s not found in connections map", macAddress)
		cm.ChangeState(macAddress, StateDisconnected, nil)
		return nil // Already disconnected
	}
	device := info.Device
	cm.mutex.RUnlock()

	// Perform the actual disconnection
	var disconnectErr error
	if device != nil {
		cm.log.Infof("ConnectionManager.Disconnect: Performing actual disconnection for device %s", macAddress)
		disconnectErr = device.Disconnect()
	} else {
		cm.log.Warnf("ConnectionManager.Disconnect: Device object is nil for %s", macAddress)
	}

	// Update state based on result
	if disconnectErr != nil {
		cm.log.Errorf("ConnectionManager.Disconnect: Error disconnecting device %s: %v", macAddress, disconnectErr)
		cm.ChangeState(macAddress, StateError, disconnectErr)
		return NewBLEError("disconnect", macAddress, disconnectErr, false, 1)
	}

	cm.log.Infof("ConnectionManager.Disconnect: Successfully completed disconnection of %s", macAddress)
	cm.ChangeState(macAddress, StateDisconnected, nil)
	return nil
}

// InactiveDevices returns devices that haven't had activity for the specified duration
func (cm *ConnectionManager) InactiveDevices(inactivityThreshold time.Duration) []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	now := time.Now()
	var inactive []string

	for mac, info := range cm.connections {
		if info.State == StateReady && now.Sub(info.LastActivity) > inactivityThreshold {
			inactive = append(inactive, mac)
		}
	}

	return inactive
}

// CheckConnections periodically checks all connections and handles issues
func (cm *ConnectionManager) CheckConnections(ctx context.Context, checkInterval, inactivityTimeout time.Duration) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check for inactive devices
			inactiveDevices := cm.InactiveDevices(inactivityTimeout)
			for _, mac := range inactiveDevices {
				cm.log.Warnf("Device %s inactive for %v, disconnecting", mac, inactivityTimeout)
				if err := cm.Disconnect(mac); err != nil {
					cm.log.Errorf("Failed to disconnect inactive device %s: %v", mac, err)
				}
			}
		}
	}
}
