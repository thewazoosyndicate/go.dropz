package ble

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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
// Thread Safety: All operations are protected by RWMutex with proper read/write locking
// Read operations use RLock() to allow concurrent access
// Write operations use Lock() for exclusive access
type ConnectionManager struct {
	adapter         *bluetooth.Adapter
	connections     map[string]*ConnectionInfo
	responseHandler *ResponseHandler
	mutex           sync.RWMutex // Protects connections map and eventHandlers slice
	log             logger.Logger
	eventHandlers   []func(ConnectionEvent)
}

// ConnectionInfo stores connection details for a device
// Thread Safety: Individual fields are accessed under ConnectionManager's mutex
// Atomic operations are used for goroutineCount to avoid lock contention
// Context and cancel are set once during creation and are read-only after that
type ConnectionInfo struct {
	Device          *bluetooth.Device
	State           ConnectionState
	LastStateChange time.Time
	LastActivity    time.Time
	Metadata        map[string]interface{}
	notifyChars     map[string]bluetooth.DeviceCharacteristic
	serviceCache    map[string]bluetooth.DeviceService
	// OpenGoPro-specific caches
	services        map[string]bluetooth.DeviceService
	characteristics map[string]*bluetooth.DeviceCharacteristic
	// Connection lifecycle management
	ctx            context.Context
	cancel         context.CancelFunc
	cleanupDone    chan struct{}
	goroutineCount int32 // Track active goroutines using atomic operations for thread safety
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(adapter *bluetooth.Adapter, responseHandler *ResponseHandler, log logger.Logger) *ConnectionManager {
	return &ConnectionManager{
		adapter:         adapter,
		connections:     make(map[string]*ConnectionInfo),
		responseHandler: responseHandler,
		log:             log,
	}
}

// RegisterEventHandler registers a handler for connection events
// Thread Safety: Write lock protects eventHandlers slice during modification
func (cm *ConnectionManager) RegisterEventHandler(handler func(ConnectionEvent)) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.eventHandlers = append(cm.eventHandlers, handler)
}

// notifyEvent notifies all registered handlers of a connection event
// Thread Safety: Read lock protects eventHandlers during iteration
// Handlers are copied to slice to avoid holding lock during handler execution
func (cm *ConnectionManager) notifyEvent(event ConnectionEvent) {
	cm.mutex.RLock()
	handlers := make([]func(ConnectionEvent), len(cm.eventHandlers))
	copy(handlers, cm.eventHandlers)
	cm.mutex.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// notifyEventWithHandlers notifies handlers without acquiring locks
// Used when handlers are already captured while holding a lock
func (cm *ConnectionManager) notifyEventWithHandlers(event ConnectionEvent, handlers []func(ConnectionEvent)) {
	for _, handler := range handlers {
		go handler(event)
	}
}

// ChangeState changes the connection state and notifies handlers
// Thread Safety: Write lock protects state transitions and connection map modifications
// Ensures atomic state changes across the entire operation
func (cm *ConnectionManager) ChangeState(macAddress string, newState ConnectionState, err error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	info, exists := cm.connections[macAddress]
	if !exists {
		if newState == StateConnecting {
			// Create a new connection info with proper initialization
			ctx, cancel := context.WithCancel(context.Background())
			info = &ConnectionInfo{
				State:           StateDisconnected, // Initial state
				LastStateChange: time.Now(),
				Metadata:        make(map[string]interface{}),
				notifyChars:     make(map[string]bluetooth.DeviceCharacteristic),
				serviceCache:    make(map[string]bluetooth.DeviceService),
				services:        make(map[string]bluetooth.DeviceService),
				characteristics: make(map[string]*bluetooth.DeviceCharacteristic),
				ctx:             ctx,
				cancel:          cancel,
				cleanupDone:     make(chan struct{}),
				goroutineCount:  0,
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

		// Create the event
		event := ConnectionEvent{
			DeviceMAC: macAddress,
			OldState:  oldState,
			NewState:  newState,
			Timestamp: now,
			Error:     err,
		}

		cm.log.Debugf("Device %s state changed: %s -> %s", macAddress, oldState, newState)

		// Cleanup on disconnect - clear caches and cancel context
		if newState == StateDisconnected && oldState != StateDisconnected {
			// Clear cached services and characteristics
			info.serviceCache = make(map[string]bluetooth.DeviceService)
			info.notifyChars = make(map[string]bluetooth.DeviceCharacteristic)
			info.services = make(map[string]bluetooth.DeviceService)
			info.characteristics = make(map[string]*bluetooth.DeviceCharacteristic)

			// Cancel context to stop all goroutines
			if info.cancel != nil {
				info.cancel()
			}
		}

		// Capture event handlers while still holding the lock to avoid deadlock
		// This prevents the need for notifyEvent to acquire a lock later
		handlers := make([]func(ConnectionEvent), len(cm.eventHandlers))
		copy(handlers, cm.eventHandlers)

		// Schedule notification to happen after the lock is released
		// This is done in a separate goroutine to prevent deadlock
		go func() {
			cm.notifyEventWithHandlers(event, handlers)
		}()
	}
}

// GetState returns the current connection state for a device
// Thread Safety: Read lock allows concurrent access to connection state
func (cm *ConnectionManager) GetState(macAddress string) ConnectionState {
	cm.log.Infof("GetState called for device %s", macAddress)
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if info, exists := cm.connections[macAddress]; exists {
		cm.log.Debugf("Current state for device %s: %s", macAddress, info.State)
		// Return the current state
		return info.State
	}
	cm.log.Debugf("GetState: Device %s not found in connections", macAddress)
	return StateDisconnected
}

// RecordActivity updates the last activity timestamp for a device
// Thread Safety: Write lock protects LastActivity field modification
func (cm *ConnectionManager) RecordActivity(macAddress string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if info, exists := cm.connections[macAddress]; exists {
		info.LastActivity = time.Now()
	}
}

// GetConnection returns the active connection for a device
// Thread Safety: Read lock allows concurrent access to device connections
// Returns device only if in connected state for safety
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
// Thread Safety: Read lock allows concurrent access to connection info
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
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return nil, NewConnectionError(macAddress, "connect", err, "Invalid MAC address format", time.Now())
	}

	if ctx == nil {
		return nil, NewConnectionError(macAddress, "connect", ErrInvalidParameter, "Context cannot be nil", time.Now())
	}

	if cm.adapter == nil {
		return nil, NewConnectionError(macAddress, "connect", ErrNotReady, "BLE adapter not initialized", time.Now())
	}

	// Check current state
	cm.log.Debugf("ConnectionManager.Connect called for device %s", macAddress)
	currentState := cm.GetState(macAddress)
	cm.log.Tracef("ConnectionManager.Connect: Device %s current state: %s", macAddress, currentState)

	if currentState == StateConnecting || currentState == StateConnected || currentState == StateReady {
		cm.log.Warnf("Device %s is already in state %s", macAddress, currentState)

		// If already connected, return the existing device
		cm.mutex.RLock()
		info := cm.connections[macAddress]
		var device *bluetooth.Device
		if info != nil {
			device = info.Device
		}
		cm.mutex.RUnlock()

		if device != nil && (currentState == StateConnected || currentState == StateReady) {
			return device, nil
		}

		// If connecting but no device yet, wait for connection to complete
		if currentState == StateConnecting {
			return nil, NewConnectionError(macAddress, "connect", ErrConnectionFailed,
				"Connection already in progress - wait for completion or retry later", time.Now())
		}
	}

	// Set state to connecting
	cm.ChangeState(macAddress, StateConnecting, nil)
	cm.log.Debugf("Attempting to connect to device %s (state: %s)", macAddress, cm.GetState(macAddress))

	// Implement retry logic for transient failures
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Calculate timeout based on attempt (progressively longer)
		timeoutDuration := time.Duration(15+attempt*5) * time.Second
		connectCtx, cancel := context.WithTimeout(ctx, timeoutDuration)

		cm.log.Debugf("Connection attempt %d/%d for device %s (timeout: %v)",
			attempt+1, maxRetries, macAddress, timeoutDuration)

		device, err := cm.attemptConnection(connectCtx, macAddress)
		cancel()

		if err == nil {
			// Connection succeeded
			cm.mutex.Lock()
			if info := cm.connections[macAddress]; info != nil {
				info.Device = device
			}
			cm.mutex.Unlock()

			cm.log.Infof("Successfully connected to device %s on attempt %d", macAddress, attempt+1)
			cm.ChangeState(macAddress, StateConnected, nil)

			// Start keep-alive goroutine for the connected device
			cm.startKeepAliveGoroutine(macAddress, 30*time.Second)

			return device, nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryableError(err) {
			cm.log.Errorf("Non-retryable error for device %s: %v", macAddress, err)
			break
		}

		if attempt < maxRetries-1 {
			backoffDuration := BackoffDuration(attempt)
			cm.log.Warnf("Connection attempt %d failed for device %s, retrying in %v: %v",
				attempt+1, macAddress, backoffDuration, err)

			select {
			case <-time.After(backoffDuration):
				// Continue to next attempt
			case <-ctx.Done():
				cm.ChangeState(macAddress, StateDisconnected, ctx.Err())
				return nil, NewConnectionError(macAddress, "connect", ctx.Err(),
					"Connection cancelled during retry backoff", time.Now())
			}
		}
	}

	// All attempts failed
	cm.log.Errorf("All %d connection attempts failed for device %s: %v", maxRetries, macAddress, lastErr)
	cm.ChangeState(macAddress, StateDisconnected, lastErr)

	return nil, NewConnectionError(macAddress, "connect", lastErr,
		fmt.Sprintf("Failed to connect after %d attempts - check device proximity and power state", maxRetries), time.Now())
}

// attemptConnection performs a single connection attempt with proper error handling
func (cm *ConnectionManager) attemptConnection(ctx context.Context, macAddress string) (*bluetooth.Device, error) {
	// Use a channel to handle connection completion
	connectDone := make(chan struct{})
	var device *bluetooth.Device
	var connectErr error

	go func() {
		defer close(connectDone)
		cm.log.Debugf("Starting connection goroutine for device %s", macAddress)

		// Parse the MAC address string using the tinygo bluetooth library
		mac, parseErr := bluetooth.ParseMAC(macAddress)
		if parseErr != nil {
			connectErr = fmt.Errorf("failed to parse MAC address %s: %v", macAddress, parseErr)
			cm.log.Errorf("MAC parsing failed for %s: %v", macAddress, parseErr)
			return
		}

		// Create bluetooth.Address with the parsed MAC
		addr := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}

		cm.log.Debugf("Attempting adapter.Connect for device %s", macAddress)

		// Perform the actual connection
		deviceValue, err := cm.adapter.Connect(addr, bluetooth.ConnectionParams{})
		if err == nil {
			device = &deviceValue
			cm.log.Debugf("adapter.Connect succeeded for device %s", macAddress)
		} else {
			connectErr = err
			cm.log.Errorf("adapter.Connect failed for device %s: %v", macAddress, err)
		}
	}()

	// Wait for connection or timeout
	select {
	case <-connectDone:
		return device, connectErr
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Disconnect properly disconnects from a device with enhanced lifecycle management
func (cm *ConnectionManager) Disconnect(macAddress string) error {
	cm.log.Infof("ConnectionManager.Disconnect called for device %s (state: %s)",
		macAddress, cm.GetState(macAddress))

	// Use the enhanced DisconnectAndCleanup method for proper resource management
	return cm.DisconnectAndCleanup(macAddress)
}

// InactiveDevices returns devices that haven't had activity for the specified duration
// Thread Safety: Read lock protects access to connections map during iteration
// Creates copy of MAC addresses to avoid holding lock during processing
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
				if err := cm.DisconnectAndCleanup(mac); err != nil {
					cm.log.Errorf("Failed to disconnect inactive device %s: %v", mac, err)
				}
			}
		}
	}
}

// SendKeepAlive sends a keep-alive command to maintain the connection with comprehensive error handling
func (cm *ConnectionManager) SendKeepAlive(macAddress string) error {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return NewConnectionError(macAddress, "keep_alive", err, "Check MAC address format", time.Now())
	}

	cm.log.Tracef("Sending keep-alive command to device %s", macAddress)

	// Validate connection state first
	currentState := cm.GetState(macAddress)
	if currentState != StateConnected && currentState != StateReady {
		return NewConnectionError(macAddress, "keep_alive",
			fmt.Errorf("device not in connected state: %s", currentState),
			"Ensure device is connected before sending keep-alive", time.Now())
	}

	cm.mutex.RLock()
	info := cm.connections[macAddress]
	var commandChar *bluetooth.DeviceCharacteristic
	var exists bool

	if info != nil && info.characteristics != nil {
		commandChar, exists = info.characteristics[CommandCharUUID]
	}
	cm.mutex.RUnlock()

	// Validate device info exists
	if info == nil {
		return NewConnectionError(macAddress, "keep_alive", ErrDeviceNotFound,
			"Device not found in connection manager", time.Now())
	}

	// Validate characteristics are available
	if info.characteristics == nil {
		return NewConnectionError(macAddress, "keep_alive", ErrCharacteristicNotFound,
			"Device characteristics not discovered", time.Now())
	}

	// Validate command characteristic exists
	if !exists {
		return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "keep_alive",
			fmt.Errorf("command characteristic not available for device %s", macAddress))
	}

	// Validate characteristic pointer is not nil
	if commandChar == nil {
		return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "keep_alive",
			fmt.Errorf("command characteristic is nil for device %s", macAddress))
	}

	// Command ID 0x5B = KEEP_ALIVE according to OpenGoPro spec
	command := []byte{0x5B}

	cm.log.Tracef("Sending keep-alive command 0x%02X to device %s", command[0], macAddress)

	// Send the command with retry logic for transient failures
	maxRetries := 2
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := commandChar.WriteWithoutResponse(command)
		if err == nil {
			cm.log.Tracef("Keep-alive command sent successfully to device %s", macAddress)
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryableError(err) {
			break
		}

		if attempt < maxRetries-1 {
			backoffDuration := BackoffDuration(attempt)
			cm.log.Tracef("Keep-alive attempt %d failed for device %s, retrying in %v: %v",
				attempt+1, macAddress, backoffDuration, err)
			time.Sleep(backoffDuration)
		}
	}

	return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "keep_alive",
		fmt.Errorf("failed to send keep-alive after %d attempts: %v", maxRetries, lastErr))
}

// GetCharacteristic safely retrieves a cached characteristic with comprehensive validation
func (cm *ConnectionManager) GetCharacteristic(macAddress, charUUID string) (*bluetooth.DeviceCharacteristic, error) {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return nil, NewCharacteristicError("unknown", charUUID, "get", err)
	}

	if charUUID == "" {
		return nil, NewCharacteristicError("unknown", charUUID, "get",
			fmt.Errorf("characteristic UUID cannot be empty"))
	}

	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	info := cm.connections[macAddress]
	if info == nil {
		return nil, NewCharacteristicError("unknown", charUUID, "get",
			fmt.Errorf("device %s not found in connections", macAddress))
	}

	// Check if device is in a valid state
	if info.State != StateConnected && info.State != StateReady {
		return nil, NewCharacteristicError("unknown", charUUID, "get",
			fmt.Errorf("device %s not in connected state (current: %s)", macAddress, info.State))
	}

	// Validate that characteristics map exists
	if info.characteristics == nil {
		return nil, NewCharacteristicError("unknown", charUUID, "get",
			fmt.Errorf("characteristics not discovered for device %s", macAddress))
	}

	char, exists := info.characteristics[charUUID]
	if !exists {
		return nil, NewCharacteristicError("unknown", charUUID, "get",
			fmt.Errorf("characteristic %s not found in cache for device %s", charUUID, macAddress))
	}

	// Additional validation: ensure the characteristic pointer is not nil
	if char == nil {
		return nil, NewCharacteristicError("unknown", charUUID, "get",
			fmt.Errorf("characteristic %s is nil for device %s", charUUID, macAddress))
	}

	return char, nil
}

// GetCachedServices returns cached services for a device (for internal use)
func (cm *ConnectionManager) GetCachedServices(macAddress string) map[string]bluetooth.DeviceService {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	info := cm.connections[macAddress]
	if info == nil || info.services == nil {
		return nil
	}

	// Return a copy to prevent external modification
	services := make(map[string]bluetooth.DeviceService)
	for uuid, service := range info.services {
		services[uuid] = service
	}
	return services
}

// startKeepAliveGoroutine starts a context-aware keep-alive goroutine with proper cleanup
func (cm *ConnectionManager) startKeepAliveGoroutine(macAddress string, interval time.Duration) {
	cm.mutex.RLock()
	info, exists := cm.connections[macAddress]
	cm.mutex.RUnlock()

	if !exists {
		cm.log.Errorf("Cannot start keep-alive for non-existent device %s", macAddress)
		return
	}

	// Track this goroutine
	atomic.AddInt32(&info.goroutineCount, 1)

	go func() {
		defer func() {
			atomic.AddInt32(&info.goroutineCount, -1)
			cm.log.Debugf("Keep-alive goroutine exiting for device %s", macAddress)
		}()

		cm.log.Debugf("Starting keep-alive goroutine for device %s (interval: %v)", macAddress, interval)

		// Create ticker with proper cleanup
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Initial delay before first keep-alive
		initialDelay := time.NewTimer(2 * time.Second)
		defer initialDelay.Stop()

		select {
		case <-info.ctx.Done():
			return
		case <-initialDelay.C:
			// Continue to main loop
		}

		for {
			select {
			case <-info.ctx.Done():
				cm.log.Tracef("Keep-alive goroutine context cancelled for device %s", macAddress)
				return

			case <-ticker.C:
				// Check if we're still connected before sending keep-alive
				currentState := cm.GetState(macAddress)
				if currentState != StateReady && currentState != StateConnected {
					cm.log.Debugf("Keep-alive stopping: device %s not in connected state (%s)", macAddress, currentState)
					return
				}

				// Send keep-alive with error handling
				if err := cm.SendKeepAlive(macAddress); err != nil {
					cm.log.Warnf("Keep-alive failed for device %s: %v", macAddress, err)
					// Don't exit on keep-alive failure, but log it
				} else {
					cm.log.Tracef("Keep-alive sent successfully for device %s", macAddress)
					cm.RecordActivity(macAddress)
				}
			}
		}
	}()
}

// DisconnectAndCleanup performs a complete disconnection with full resource cleanup
func (cm *ConnectionManager) DisconnectAndCleanup(macAddress string) error {
	cm.log.Infof("DisconnectAndCleanup called for device %s", macAddress)

	// First, set state to disconnecting to prevent new operations
	cm.ChangeState(macAddress, StateDisconnecting, nil)

	cm.mutex.RLock()
	info, exists := cm.connections[macAddress]
	cm.mutex.RUnlock()

	if !exists {
		cm.log.Debugf("Device %s not found in connections during cleanup", macAddress)
		return nil
	}

	// Cancel context to stop all goroutines
	if info.cancel != nil {
		cm.log.Debugf("Cancelling context for device %s to stop goroutines", macAddress)
		info.cancel()
	}

	// Wait for goroutines to finish with timeout
	cleanup := make(chan struct{})
	go func() {
		defer close(cleanup)

		maxWait := 5 * time.Second
		startTime := time.Now()

		for {
			goroutineCount := atomic.LoadInt32(&info.goroutineCount)
			if goroutineCount == 0 {
				cm.log.Debugf("All goroutines stopped for device %s", macAddress)
				break
			}

			if time.Since(startTime) > maxWait {
				cm.log.Warnf("Timeout waiting for %d goroutines to stop for device %s", goroutineCount, macAddress)
				break
			}

			cm.log.Tracef("Waiting for %d goroutines to stop for device %s", goroutineCount, macAddress)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Wait for cleanup or timeout
	select {
	case <-cleanup:
		cm.log.Debugf("Goroutine cleanup completed for device %s", macAddress)
	case <-time.After(6 * time.Second):
		cm.log.Warnf("Timeout during goroutine cleanup for device %s", macAddress)
	}

	// Perform the actual BLE disconnection
	var disconnectErr error
	if info.Device != nil {
		cm.log.Debugf("Disconnecting BLE device %s", macAddress)
		disconnectErr = info.Device.Disconnect()
		if disconnectErr != nil {
			cm.log.Errorf("BLE disconnect failed for device %s: %v", macAddress, disconnectErr)
		}
	}

	// Clean up connection info
	cm.mutex.Lock()
	if info, exists := cm.connections[macAddress]; exists {
		// Clear all cached data
		info.Device = nil
		info.services = make(map[string]bluetooth.DeviceService)
		info.characteristics = make(map[string]*bluetooth.DeviceCharacteristic)
		info.serviceCache = make(map[string]bluetooth.DeviceService)
		info.notifyChars = make(map[string]bluetooth.DeviceCharacteristic)

		// Signal cleanup completion
		select {
		case <-info.cleanupDone:
			// Already closed
		default:
			close(info.cleanupDone)
		}
	}
	cm.mutex.Unlock()

	// Update final state
	if disconnectErr != nil {
		cm.ChangeState(macAddress, StateError, disconnectErr)
		return NewBLEError("disconnect", macAddress, disconnectErr, false, 1)
	}

	cm.ChangeState(macAddress, StateDisconnected, nil)
	cm.log.Infof("DisconnectAndCleanup completed successfully for device %s", macAddress)
	return nil
}

// CleanupAllConnections performs cleanup of all connections (used during shutdown)
func (cm *ConnectionManager) CleanupAllConnections() error {
	cm.log.Infof("Cleaning up all connections")

	cm.mutex.RLock()
	macAddresses := make([]string, 0, len(cm.connections))
	for mac := range cm.connections {
		macAddresses = append(macAddresses, mac)
	}
	cm.mutex.RUnlock()

	var errors []string
	for _, mac := range macAddresses {
		if err := cm.DisconnectAndCleanup(mac); err != nil {
			cm.log.Errorf("Failed to cleanup connection %s: %v", mac, err)
			errors = append(errors, fmt.Sprintf("%s: %v", mac, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errors, "; "))
	}

	cm.log.Infof("All connections cleaned up successfully")
	return nil
}

// GetActiveGoroutineCount returns the number of active goroutines for a device
func (cm *ConnectionManager) GetActiveGoroutineCount(macAddress string) int32 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if info, exists := cm.connections[macAddress]; exists {
		return atomic.LoadInt32(&info.goroutineCount)
	}
	return 0
}

// IsContextCancelled checks if the connection context for a device has been cancelled
func (cm *ConnectionManager) IsContextCancelled(macAddress string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if info, exists := cm.connections[macAddress]; exists && info.ctx != nil {
		select {
		case <-info.ctx.Done():
			return true
		default:
			return false
		}
	}
	return true // If no context exists, consider it cancelled
}

// WaitForCleanup waits for cleanup completion of a specific device
func (cm *ConnectionManager) WaitForCleanup(macAddress string, timeout time.Duration) error {
	cm.mutex.RLock()
	info, exists := cm.connections[macAddress]
	cm.mutex.RUnlock()

	if !exists {
		return nil // Already cleaned up
	}

	select {
	case <-info.cleanupDone:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for cleanup of device %s", macAddress)
	}
}

// GetConnectionContext returns the context for a specific connection
func (cm *ConnectionManager) GetConnectionContext(macAddress string) context.Context {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if info, exists := cm.connections[macAddress]; exists && info.ctx != nil {
		return info.ctx
	}
	return context.Background()
}

// ValidateConnectionState performs comprehensive validation of connection state
func (cm *ConnectionManager) ValidateConnectionState(macAddress string) error {
	cm.mutex.RLock()
	info, exists := cm.connections[macAddress]
	cm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("device %s not found in connections", macAddress)
	}

	// Check context state
	if info.ctx == nil {
		return fmt.Errorf("device %s has nil context", macAddress)
	}

	select {
	case <-info.ctx.Done():
		return fmt.Errorf("device %s context is cancelled: %v", macAddress, info.ctx.Err())
	default:
		// Context is still active
	}

	// Check state consistency
	state := info.State
	switch state {
	case StateConnected, StateReady:
		if info.Device == nil {
			return fmt.Errorf("device %s in state %s but Device is nil", macAddress, state)
		}
	case StateDisconnected:
		goroutineCount := atomic.LoadInt32(&info.goroutineCount)
		if goroutineCount > 0 {
			return fmt.Errorf("device %s disconnected but has %d active goroutines", macAddress, goroutineCount)
		}
	}

	return nil
}
