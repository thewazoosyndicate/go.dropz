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

// GetConnection returns the active connection for a device
func (cm *ConnectionManager) GetConnection(macAddress string) *bluetooth.Device {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if info, exists := cm.connections[macAddress]; exists && info.Device != nil {
		// Only return device if we're in a connected state
		if info.State == StateConnected || info.State == StateReady {
			return info.Device
		}
	}
	return nil
}

// GetConnectionInfo returns connection information for a device
func (cm *ConnectionManager) GetConnectionInfo(macAddress string) *ConnectionInfo {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if info, exists := cm.connections[macAddress]; exists {
		return info
	}
	return nil
}

// Connect initiates a connection to a device with proper state management
func (cm *ConnectionManager) Connect(ctx context.Context, macAddress string) (*bluetooth.Device, error) {
	// Check current state
	cm.log.Debugf("ConnectionManager.Connect called for device %s", macAddress)
	currentState := cm.GetState(macAddress)
	cm.log.Tracef("ConnectionManager.Connect: Device %s current state: %s", macAddress, currentState)
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

	// Attempt connection with a shorter, more aggressive timeout
	connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Use a channel to handle connection completion
	connectDone := make(chan struct{})
	var device *bluetooth.Device
	var connectErr error

	cm.log.Infof("ConnectionManager.Connect: Starting connection attempt for device %s with 15s timeout", macAddress)

	go func() {
		defer close(connectDone)
		cm.log.Debugf("ConnectionManager.Connect: Starting connection goroutine for device %s", macAddress)

		// Parse the MAC address string using the tinygo bluetooth library
		mac, parseErr := bluetooth.ParseMAC(macAddress)
		if parseErr != nil {
			connectErr = fmt.Errorf("failed to parse MAC address %s: %v", macAddress, parseErr)
			cm.log.Errorf("ConnectionManager.Connect: MAC parsing failed for %s: %v", macAddress, parseErr)
			return
		}

		// Create bluetooth.Address with the parsed MAC
		addr := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}

		cm.log.Debugf("ConnectionManager.Connect: Attempting adapter.Connect for device %s", macAddress)

		// Create a timer for the actual connection attempt
		connectionTimer := time.NewTimer(12 * time.Second)
		connectionResult := make(chan struct{})

		go func() {
			defer close(connectionResult)
			deviceValue, err := cm.adapter.Connect(addr, bluetooth.ConnectionParams{})
			if err == nil {
				device = &deviceValue
				cm.log.Debugf("ConnectionManager.Connect: adapter.Connect succeeded for device %s", macAddress)
			} else {
				connectErr = err
				cm.log.Errorf("ConnectionManager.Connect: adapter.Connect failed for device %s: %v", macAddress, err)
			}
		}()

		// Wait for either connection completion or timeout
		select {
		case <-connectionResult:
			connectionTimer.Stop()
			cm.log.Debugf("ConnectionManager.Connect: Connection attempt completed for device %s", macAddress)
		case <-connectionTimer.C:
			connectErr = fmt.Errorf("connection attempt timed out after 12 seconds")
			cm.log.Errorf("ConnectionManager.Connect: Connection attempt timed out for device %s", macAddress)
		}
	}()

	// Wait for connection or timeout
	select {
	case <-connectDone:
		// Connection completed (success or error)
		if connectErr != nil {
			cm.log.Errorf("ConnectionManager.Connect: Connection failed for device %s: %v", macAddress, connectErr)
			cm.ChangeState(macAddress, StateDisconnected, connectErr)
			return nil, NewBLEError("connect", macAddress, connectErr, true, 1)
		}

		// Connection succeeded
		cm.mutex.Lock()
		info := cm.connections[macAddress]
		info.Device = device
		cm.mutex.Unlock()

		cm.log.Infof("ConnectionManager.Connect: Successfully connected to device %s", macAddress)
		cm.ChangeState(macAddress, StateConnected, nil)
		return device, nil

	case <-connectCtx.Done():
		// Timeout or cancellation
		cm.log.Errorf("ConnectionManager.Connect: Context timeout/cancellation for device %s: %v", macAddress, connectCtx.Err())
		cm.ChangeState(macAddress, StateDisconnected, connectCtx.Err())
		return nil, NewBLEError("connect", macAddress, ErrTimeout, true, 1)
	}
}

// ConnectWithOpenGoProSpec connects to a device following OpenGoPro BLE specification
func (cm *ConnectionManager) ConnectWithOpenGoProSpec(ctx context.Context, macAddress string) (*bluetooth.Device, error) {
	cm.log.Infof("Starting OpenGoPro-compliant connection sequence for device %s", macAddress)

	// Step 1: Establish basic BLE connection
	device, err := cm.Connect(ctx, macAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to establish BLE connection: %v", err)
	}

	// Step 2: Wait for connection to stabilize (OpenGoPro recommendation)
	cm.log.Debugf("Waiting for connection to stabilize...")
	time.Sleep(500 * time.Millisecond)

	// Step 3: Discover services with timeout
	cm.log.Debugf("Discovering services...")
	serviceCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	serviceDiscoveryDone := make(chan struct{})
	var services []bluetooth.DeviceService
	var serviceErr error

	go func() {
		defer close(serviceDiscoveryDone)
		services, serviceErr = device.DiscoverServices(nil)
	}()

	select {
	case <-serviceDiscoveryDone:
		if serviceErr != nil {
			cm.ChangeState(macAddress, StateError, serviceErr)
			return nil, fmt.Errorf("service discovery failed: %v", serviceErr)
		}
	case <-serviceCtx.Done():
		cm.ChangeState(macAddress, StateError, serviceCtx.Err())
		return nil, fmt.Errorf("service discovery timeout: %v", serviceCtx.Err())
	}

	cm.log.Debugf("Discovered %d services for device %s", len(services), macAddress)

	// Step 4: Verify OpenGoPro Control & Query service is present
	var hasOpenGoProService bool
	for _, svc := range services {
		if svc.UUID().String() == GoProControlServiceUUID {
			hasOpenGoProService = true
			break
		}
	}

	if !hasOpenGoProService {
		err := fmt.Errorf("OpenGoPro Control & Query service not found")
		cm.ChangeState(macAddress, StateError, err)
		return nil, err
	}

	// Step 5: Mark as ready for OpenGoPro operations
	cm.ChangeState(macAddress, StateReady, nil)
	cm.log.Infof("OpenGoPro BLE connection established successfully for device %s", macAddress)

	// Store services in connection info
	cm.mutex.Lock()
	if info, exists := cm.connections[macAddress]; exists {
		if info.Metadata == nil {
			info.Metadata = make(map[string]interface{})
		}
		info.Metadata["services"] = services
		info.Metadata["opengopro_ready"] = true
	}
	cm.mutex.Unlock()

	return device, nil
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
