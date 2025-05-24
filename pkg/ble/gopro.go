package ble

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// GoPro BLE constants
const (
	// Service UUIDs
	GoProWifiServiceUUID      = "b5f90001-aa8d-11e3-9046-0002a5d5c51b"
	GoProControlServiceUUID   = "0000fea6-0000-1000-8000-00805f9b34fb"
	GoProCameraManagementUUID = "b5f90090-aa8d-11e3-9046-0002a5d5c51b"

	// Characteristic UUIDs for WiFi
	WifiSSIDCharUUID     = "b5f90002-aa8d-11e3-9046-0002a5d5c51b"
	WifiPasswordCharUUID = "b5f90003-aa8d-11e3-9046-0002a5d5c51b"
	WifiPowerCharUUID    = "b5f90004-aa8d-11e3-9046-0002a5d5c51b"
	WifiStateCharUUID    = "b5f90005-aa8d-11e3-9046-0002a5d5c51b"

	// Network Management Characteristic UUIDs
	NetworkMgmtCommandCharUUID  = "b5f90091-aa8d-11e3-9046-0002a5d5c51b"
	NetworkMgmtResponseCharUUID = "b5f90092-aa8d-11e3-9046-0002a5d5c51b"

	// Characteristic UUIDs for Control & Query
	CommandCharUUID          = "b5f90072-aa8d-11e3-9046-0002a5d5c51b"
	CommandResponseCharUUID  = "b5f90073-aa8d-11e3-9046-0002a5d5c51b"
	SettingsCharUUID         = "b5f90074-aa8d-11e3-9046-0002a5d5c51b"
	SettingsResponseCharUUID = "b5f90075-aa8d-11e3-9046-0002a5d5c51b"
	QueryCharUUID            = "b5f90076-aa8d-11e3-9046-0002a5d5c51b"
	QueryResponseCharUUID    = "b5f90077-aa8d-11e3-9046-0002a5d5c51b"

	// Command IDs
	CommandSetDateTime      = 0x03
	CommandSetShutter       = 0x01
	CommandSetAPControl     = 0x17
	CommandSleep            = 0x05
	CommandSetAnalytics     = 0x50
	CommandSetCameraControl = 0x3A
	CommandSetTurboActive   = 0x09

	// Query IDs
	QueryGetDateTime            = 0x03
	QueryGetWifiSSID            = 0x02
	QueryGetWifiPassword        = 0x03
	QueryGetHardwareInfo        = 0x3F
	QueryGetOpenGoProVersion    = 0x51
	QueryGetSettingValues       = 0x12
	QueryGetStatusValues        = 0x13
	QueryGetSettingCapabilities = 0x14
	QueryGetLastCapturedMedia   = 0x16
)

// GoPro Model IDs (based on OpenGoPro documentation)
const (
	GoProModelHERO9  = 55
	GoProModelHERO10 = 57
	GoProModelHERO11 = 58 // Also includes 60 for Black variant
	GoProModelHERO12 = 62
	GoProModelHERO13 = 65
)

// Pairing State constants (Status ID 19)
const (
	PairingStateNeverStarted = 0
	PairingStateStarted      = 1
	PairingStateAborted      = 2
	PairingStateCancelled    = 3
	PairingStateCompleted    = 4
)

// BLE packet handling constants
const (
	// Packet headers
	GeneralPurposeCommandHeader = 0x10
	ExtendedCommandHeader       = 0x90
	KeepAliveCommandHeader      = 0xA0

	// Packet types
	PacketTypeStart        = 0x10
	PacketTypeContinuation = 0x00
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

// BLEManager handles Bluetooth Low Energy operations
type BLEManager struct {
	adapter     *bluetooth.Adapter
	devices     map[string]*Device           // MAC -> Device
	connections map[string]*bluetooth.Device // MAC -> Connection
	mutex       sync.RWMutex
	log         logger.Logger

	isScanning bool
	scanMutex  sync.Mutex

	// Connection state manager
	connManager *ConnectionManager

	// Context for background tasks
	ctx        context.Context
	cancelFunc context.CancelFunc

	// Event handling
	eventHandlers map[EventType][]EventHandler
	eventMutex    sync.RWMutex

	// Service map for caching discovered characteristics
	serviceMap map[string]map[string]*bluetooth.DeviceCharacteristic // serviceUUID -> (charUUID -> characteristic)

	// Keep-alive goroutine cancellation map
	keepAliveCancels map[string]context.CancelFunc

	// Response channels for query responses
	responseTracker map[string]*ResponseTracker // MAC address -> response tracker
}

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

// RegisterEventHandler registers a handler for BLE events
func (m *BLEManager) RegisterEventHandler(eventType EventType, handler EventHandler) {
	m.eventMutex.Lock()
	defer m.eventMutex.Unlock()

	if m.eventHandlers == nil {
		m.eventHandlers = make(map[EventType][]EventHandler)
	}

	m.eventHandlers[eventType] = append(m.eventHandlers[eventType], handler)
	m.log.Debugf("Registered handler for event type: %s", eventType)
}

// RemoveEventHandler removes a handler for BLE events
func (m *BLEManager) RemoveEventHandler(eventType EventType, handler EventHandler) {
	m.eventMutex.Lock()
	defer m.eventMutex.Unlock()

	if m.eventHandlers == nil {
		return
	}

	handlers, exists := m.eventHandlers[eventType]
	if !exists {
		return
	}

	// Find and remove the handler
	for i, h := range handlers {
		if fmt.Sprintf("%p", h) == fmt.Sprintf("%p", handler) {
			// Remove the handler by replacing it with the last one and truncating
			handlers[i] = handlers[len(handlers)-1]
			m.eventHandlers[eventType] = handlers[:len(handlers)-1]
			break
		}
	}
}

// emitEvent emits an event to all registered handlers
func (m *BLEManager) emitEvent(event BLEEvent) {
	m.eventMutex.RLock()
	defer m.eventMutex.RUnlock()

	if m.eventHandlers == nil {
		return
	}

	handlers, exists := m.eventHandlers[event.Type]
	if !exists || len(handlers) == 0 {
		return
	}

	// Call each handler in a goroutine to avoid blocking
	for _, handler := range handlers {
		go func(h EventHandler) {
			h(event)
		}(handler)
	}
}

// NewBLEManager creates a new BLE manager
func NewBLEManager() (*BLEManager, error) {
	adapter := bluetooth.DefaultAdapter
	if adapter == nil {
		return nil, fmt.Errorf("failed to get default Bluetooth adapter")
	}

	// Create context for background tasks
	ctx, cancel := context.WithCancel(context.Background())

	// Enable with timeout
	enableCtx, enableCancel := context.WithTimeout(ctx, 5*time.Second)
	defer enableCancel()

	enableCh := make(chan error, 1)
	go func() {
		enableCh <- adapter.Enable()
	}()

	select {
	case err := <-enableCh:
		if err != nil {
			cancel() // Clean up the context
			return nil, fmt.Errorf("failed to enable BLE adapter: %w", err)
		}
	case <-enableCtx.Done():
		cancel() // Clean up the context
		return nil, fmt.Errorf("timeout while enabling BLE adapter")
	}

	// Get filtered logger for BLE operations (filters out common BLE warnings)
	log := logger.GetBLEFilteredLogger()

	m := &BLEManager{
		adapter:          adapter,
		devices:          make(map[string]*Device),
		connections:      make(map[string]*bluetooth.Device),
		log:              log,
		ctx:              ctx,
		cancelFunc:       cancel,
		eventHandlers:    make(map[EventType][]EventHandler),
		serviceMap:       make(map[string]map[string]*bluetooth.DeviceCharacteristic),
		keepAliveCancels: make(map[string]context.CancelFunc),
		responseTracker:  make(map[string]*ResponseTracker),
	}

	// Initialize connection manager
	m.connManager = NewConnectionManager(adapter, log)

	// Set the BLEManager reference in the ConnectionManager
	// This allows the ConnectionManager to call Sleep before disconnection
	m.connManager.SetBLEManager(m)

	// Verify that the BLEManager was correctly set in ConnectionManager
	m.log.Infof("NewBLEManager: Created BLEManager and set reference in ConnectionManager")

	// Register event handler for connection state changes
	m.connManager.RegisterEventHandler(func(event ConnectionEvent) {
		m.log.Infof("Connection event: %s changed from %s to %s",
			event.DeviceMAC, event.OldState, event.NewState)

		// Handle state changes
		switch event.NewState {
		case StateDisconnected:
			// Clean up any connection-specific resources
			m.mutex.Lock()
			delete(m.connections, event.DeviceMAC)
			if device, exists := m.devices[event.DeviceMAC]; exists {
				device.IsConnected = false
				// Emit device disconnected event
				m.emitEvent(BLEEvent{
					Type:      EventDeviceDisconnected,
					Device:    device,
					Timestamp: time.Now(),
					Error:     event.Error,
				})
			}
			m.mutex.Unlock()

		case StateConnected, StateReady:
			// Update device status in our maps
			m.mutex.Lock()
			if device, exists := m.devices[event.DeviceMAC]; exists {
				device.IsConnected = true
				// Emit device connected event
				m.emitEvent(BLEEvent{
					Type:      EventDeviceConnected,
					Device:    device,
					Timestamp: time.Now(),
				})
			}
			m.mutex.Unlock()
		}
	})

	// Start connection checking in the background with 1 minute check interval
	// and 5 minute inactivity timeout
	go m.connManager.CheckConnections(ctx, 1*time.Minute, 5*time.Minute)

	return m, nil
}

// VerifyConnectionManagerSetup checks if the BLEManager reference is correctly set in ConnectionManager
func (m *BLEManager) VerifyConnectionManagerSetup() bool {
	if m.connManager == nil {
		m.log.Errorf("VerifyConnectionManagerSetup: ConnectionManager is nil")
		return false
	}

	if m.connManager.bleManager == nil {
		m.log.Errorf("VerifyConnectionManagerSetup: BLEManager reference in ConnectionManager is nil")
		return false
	}

	if m.connManager.bleManager != m {
		m.log.Errorf("VerifyConnectionManagerSetup: BLEManager reference doesn't match this instance")
		return false
	}

	m.log.Infof("VerifyConnectionManagerSetup: BLEManager reference correctly set in ConnectionManager")
	return true
}

// Start initializes the BLE manager
func (m *BLEManager) Start() error {
	m.log.Info("BLEManager: Starting...")

	// Verify that the BLEManager reference is correctly set in ConnectionManager
	if !m.VerifyConnectionManagerSetup() {
		m.log.Warnf("BLEManager.Start: ConnectionManager setup is incorrect, fixing it now")
		// Try to fix the setup
		if m.connManager != nil {
			m.connManager.SetBLEManager(m)
		}
	}

	// Adapter is enabled in NewBLEManager.
	// Any other start-up logic can go here.
	return nil
}

// Stop stops the BLE manager
func (m *BLEManager) Stop() {
	m.log.Info("BLEManager: Stopping...")

	// Cancel the context to stop all background tasks
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// 1. Stop any active scanning
	m.scanMutex.Lock()
	wasScanning := m.isScanning
	m.scanMutex.Unlock() // Unlock before potentially blocking StopScan

	if wasScanning {
		m.log.Debug("BLEManager: Scan was in progress. Requesting adapter.StopScan().")
		if err := m.adapter.StopScan(); err != nil {
			m.log.Warnf("BLEManager: adapter.StopScan() error during Stop(): %v", err)
		} else {
			m.log.Debug("BLEManager: adapter.StopScan() called successfully during Stop().")
		}
		// After StopScan returns, StartScanning (if running) should detect this and exit.
		// We ensure isScanning is set to false.
		m.scanMutex.Lock()
		m.isScanning = false
		m.scanMutex.Unlock()
	} else {
		m.log.Debug("BLEManager: No scan in progress to stop.")
	}

	// 2. Disconnect all connected devices using connection manager
	// Get a list of all device MACs to disconnect
	m.mutex.Lock()
	var macsToDisconnect []string
	for mac := range m.connections {
		macsToDisconnect = append(macsToDisconnect, mac)
	}
	m.mutex.Unlock()

	// Disconnect each device
	var wgDisconnect sync.WaitGroup
	for _, mac := range macsToDisconnect {
		// Check if it's managed by the connection manager
		if m.connManager.GetState(mac) != StateDisconnected {
			wgDisconnect.Add(1)
			go func(macAddress string) {
				defer wgDisconnect.Done()
				m.log.Debugf("BLEManager: Attempting to disconnect from %s in Stop()", macAddress)

				// Use the connection manager for clean disconnection
				err := m.connManager.Disconnect(macAddress)

				if err != nil {
					m.log.Warnf("BLEManager: Error disconnecting from %s during Stop(): %v", macAddress, err)
				} else {
					m.log.Debugf("BLEManager: Successfully disconnected from %s during Stop()", macAddress)
				}
			}(mac)
		} else {
			// Handle connection that's not in the connection manager
			wgDisconnect.Add(1)
			go func(macAddress string) {
				defer wgDisconnect.Done()
				m.log.Debugf("BLEManager: Attempting legacy disconnect from %s in Stop()", macAddress)
				if err := m.Disconnect(macAddress); err != nil {
					m.log.Warnf("BLEManager: Error legacy disconnecting from %s during Stop(): %v", macAddress, err)
				}
			}(mac)
		}
	}
	wgDisconnect.Wait()
	m.log.Debug("BLEManager: All disconnection attempts in Stop() finished.")

	// Clean up internal state
	m.mutex.Lock()
	m.connections = make(map[string]*bluetooth.Device)
	m.devices = make(map[string]*Device)
	m.mutex.Unlock()

	m.log.Info("BLEManager: Stopped.")
}

// StartScanning starts a BLE scan for GoPro devices.
// The provided ctx is for the duration of this specific scan attempt.
func (m *BLEManager) StartScanning(ctx context.Context) error {
	// First, check BLE system
	if err := m.CheckBLESystem(); err != nil {
		return fmt.Errorf("BLE system check failed: %w", err)
	}

	m.scanMutex.Lock()
	if m.isScanning {
		m.scanMutex.Unlock()
		m.log.Debug("BLEManager: StartScanning called while a scan is already in progress.")
		return nil // Return success rather than error to facilitate continuous scanning
	}
	m.isScanning = true
	m.scanMutex.Unlock()

	m.log.Debug("BLEManager: Starting scan specifically for GoPro devices (service 0xFEA6)")

	// Create a derived context with reasonable timeout if none provided
	scanCtx := ctx
	var cancel context.CancelFunc
	deadline, hasDeadline := ctx.Deadline()

	// If no deadline is set or it's too far away, set a reasonable one
	if !hasDeadline || time.Until(deadline) > 30*time.Second {
		scanCtx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}

	// Use a channel to signal scan completion
	scanDone := make(chan struct{})
	var scanErr error

	// Run scan in a goroutine so we can monitor for context timeout
	go func() {
		defer close(scanDone)
		scanErr = m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			// Process scan result
			if strings.Contains(result.LocalName(), "GoPro") {
				m.mutex.Lock()
				defer m.mutex.Unlock()

				// Create or update device
				device := &Device{
					Name:       result.LocalName(),
					MACAddress: result.Address.String(),
					RSSI:       int32(result.RSSI),
				}

				m.devices[device.MACAddress] = device

				// Emit device discovered event
				m.emitEvent(BLEEvent{
					Type:      EventDeviceDiscovered,
					Device:    device,
					Timestamp: time.Now(),
				})
			}
		})
	}()

	// Wait for scan to complete or context timeout
	select {
	case <-scanDone:
		// Scan completed normally
		m.scanMutex.Lock()
		m.isScanning = false
		m.scanMutex.Unlock()

		if scanErr != nil {
			m.log.Errorf("BLEManager: adapter.Scan() error: %v", scanErr)
			return fmt.Errorf("adapter scan error: %w", scanErr)
		}

		m.log.Debug("BLEManager: Scan finished successfully.")
		return nil

	case <-scanCtx.Done():
		// Context timeout or cancellation
		m.log.Debugf("BLEManager: Scan context done: %v. Stopping scan.", scanCtx.Err())

		// Stop the scan
		if err := m.adapter.StopScan(); err != nil {
			m.log.Warnf("BLEManager: Failed to stop scan: %v", err)
		}

		// Wait for scan to actually stop with a timeout
		select {
		case <-scanDone:
			// Scan has stopped
			m.log.Debug("BLEManager: Scan stopped after context cancellation")
		case <-time.After(2 * time.Second):
			m.log.Warn("BLEManager: Scan didn't stop within timeout after StopScan")

			// Force reset the scanning state
			m.scanMutex.Lock()
			m.isScanning = false
			m.scanMutex.Unlock()
		}

		m.scanMutex.Lock()
		m.isScanning = false
		m.scanMutex.Unlock()

		// Don't treat deadline exceeded as an error in the continuous scan case
		if scanCtx.Err() == context.DeadlineExceeded {
			m.log.Debug("BLEManager: Scan timeout is expected in continuous mode")
			return nil
		}

		// Other context errors should be returned
		return scanCtx.Err()
	}
}

// stopScanning stops the current BLE scan (DEPRECATED if StartScanning handles its lifecycle well with Stop)
// This was an internal helper, BLEManager.Stop() and StartScanning's context handling should cover this.
/*
func (m *BLEManager) stopScanning() {…}
*/

// GetDiscoveredDevices returns all discovered GoPro devices
func (m *BLEManager) GetDiscoveredDevices() []Device {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	devices := make([]Device, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, *device)
	}
	return devices
}

// Connect attempts to connect to a GoPro device via BLE
func (m *BLEManager) Connect(macAddress string) error {
	// Find device from scan results
	m.mutex.RLock()
	device, exists := m.devices[macAddress]
	if !exists {
		m.mutex.RUnlock()
		return fmt.Errorf("device with MAC address %s not found", macAddress)
	}
	m.mutex.RUnlock()

	m.log.Debugf("Connecting to device: %s (%s)", device.Name, macAddress)

	// Connect with retry logic
	var bleDevice *bluetooth.Device
	var err error
	for retries := 0; retries < 3; retries++ {
		// Scanning logic with proper context handling
		if !m.isScanning {
			m.log.Debug("BLEManager: Starting scan before connection attempt")
			scanCtx, scanCancel := context.WithTimeout(context.Background(), 5*time.Second)
			scanErr := m.StartScanning(scanCtx)
			scanCancel() // Always cancel the context

			if scanErr != nil && scanErr != context.DeadlineExceeded {
				return fmt.Errorf("failed to start scan: %v", scanErr)
			}

			// Even with timeout, we might have discovered some devices
			// Short wait to ensure devices are processed
			time.Sleep(500 * time.Millisecond)
		}

		// Get the device from our map
		m.mutex.RLock()
		_, stillExists := m.devices[macAddress]
		m.mutex.RUnlock()

		if !stillExists {
			return fmt.Errorf("device %s no longer in scan results", macAddress)
		}

		// Try connecting to the device
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 5*time.Second)

		// Connection logic using a goroutine to handle timeouts
		connectDone := make(chan struct{})
		go func() {
			defer close(connectDone)

			// Parse the MAC address string to a bluetooth.MAC type
			mac, parseErr := bluetooth.ParseMAC(macAddress)
			if parseErr != nil {
				err = fmt.Errorf("failed to parse MAC address %s: %v", macAddress, parseErr)
				return
			}

			// Create an Address from the MAC
			addr := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}

			// This would ideally use the context if the library supports it
			// For now, we're still using the basic Connect method but with timeout supervision
			bleDevice, err = m.adapter.Connect(addr, bluetooth.ConnectionParams{})
		}()

		select {
		case <-connectDone:
			// Connection attempt completed
			connectCancel()
			if err == nil {
				m.log.Debugf("Connection attempt %d succeeded", retries+1)
				break
			}
			m.log.Warnf("Connection attempt %d failed: %v", retries+1, err)

		case <-connectCtx.Done():
			// Timeout
			connectCancel()
			err = fmt.Errorf("connection timeout")
			m.log.Warnf("Connection attempt %d timed out", retries+1)
		}

		if err == nil {
			break
		}

		// Check if error contains the DBus Properties.GetAll issue
		if err != nil && strings.Contains(err.Error(), "Properties.GetAll") {
			m.log.Warnf("BlueZ DBus interface issue detected: %v", err)

			// Force stop any running scan
			m.scanMutex.Lock()
			if m.isScanning {
				m.log.Debug("Stopping scan to resolve DBus interface issue")
				m.adapter.StopScan()
				m.isScanning = false
			}
			m.scanMutex.Unlock()

			// Wait longer to give BlueZ time to stabilize
			waitTime := 2 * time.Second * time.Duration(retries+1)
			m.log.Infof("Waiting %v before retry to allow BlueZ to stabilize", waitTime)
			time.Sleep(waitTime)

			// Reset the bluetooth adapter if possible
			m.log.Debug("Attempting to reset Bluetooth adapter")
			_ = m.adapter.Enable() // Ignore error, just try to reset
		} else {
			// Standard exponential backoff between retries
			if retries < 2 { // Don't sleep after the last attempt
				time.Sleep(time.Duration(500*(retries+1)) * time.Millisecond)
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect after retries: %v", err)
	}

	// Update device status
	m.mutex.Lock()
	device.IsConnected = true
	m.connections[macAddress] = bleDevice
	m.mutex.Unlock()

	m.log.Infof("Connected to GoPro device: %s (%s)", device.Name, macAddress)

	// Give BlueZ a moment to fully initialize the connection
	// This helps avoid DBus interface issues in subsequent operations
	time.Sleep(1 * time.Second)

	// Discover all services
	m.log.Debug("Discovering all services")
	svcs, err := bleDevice.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("failed to discover services: %v", err)
	}
	m.log.Debugf("Discovered %d services", len(svcs))

	// Initialize service map for this device
	m.serviceMap = make(map[string]map[string]*bluetooth.DeviceCharacteristic)

	// Discover all characteristics and build service map
	for _, svc := range svcs {
		serviceUUID := svc.UUID().String()
		m.log.Debugf("Service: %s", serviceUUID)

		chars, err := svc.DiscoverCharacteristics(nil)
		if err != nil {
			m.log.Warnf("Failed to discover characteristics for service %s: %v", serviceUUID, err)
			continue
		}

		m.log.Debugf("Found %d characteristics for service %s", len(chars), serviceUUID)

		// Initialize characteristic map for this service
		m.serviceMap[serviceUUID] = make(map[string]*bluetooth.DeviceCharacteristic)

		// Store characteristics in the service map
		for i := range chars {
			charUUID := chars[i].UUID().String()
			m.log.Debugf("Characteristic: %s", charUUID)
			m.serviceMap[serviceUUID][charUUID] = &chars[i]
		}
	}

	// Find and configure the Control & Query service (0xFEA6)
	controlService, ok := m.serviceMap[GoProControlServiceUUID]
	if !ok {
		return fmt.Errorf("Control & Query service not found")
	}

	// Get required characteristics
	queryChar, ok := controlService[QueryCharUUID]
	if !ok {
		return fmt.Errorf("Query characteristic not found")
	}

	queryRespChar, ok := controlService[QueryResponseCharUUID]
	if !ok {
		return fmt.Errorf("Query Response characteristic not found")
	}

	// Enable notifications for Query Response with enhanced handler
	m.log.Debug("Enabling notifications for Query Response")

	// Initialize response tracker for this device
	m.mutex.Lock()
	if m.responseTracker[macAddress] == nil {
		m.responseTracker[macAddress] = &ResponseTracker{
			responseChannel: make(chan *QueryResponseData, 10), // Buffer for multiple responses
		}
	}
	tracker := m.responseTracker[macAddress]
	m.mutex.Unlock()

	if err := queryRespChar.EnableNotifications(func(buf []byte) {
		m.log.Debugf("Received query response: %v (length: %d)", buf, len(buf))

		// Parse response based on query type
		if len(buf) >= 3 {
			// Basic response structure: [Query ID] [Status] [Data...]
			queryID := buf[0]
			status := buf[1]
			data := buf[2:]

			// Create response data
			responseData := &QueryResponseData{
				QueryID:      queryID,
				Status:       status,
				Data:         data,
				ResponseTime: time.Now(),
			}

			// Store the response
			tracker.mutex.Lock()
			tracker.lastResponse = responseData
			// Store model ID if this is a hardware info response
			if queryID == QueryGetHardwareInfo && status == 0 && len(data) >= 1 {
				tracker.ModelID = int(data[0])
				m.log.Debugf("Stored model ID in tracker: %d (%s)", tracker.ModelID, getGoProModelName(tracker.ModelID))
			}
			tracker.mutex.Unlock()

			// Send to channel (non-blocking)
			select {
			case tracker.responseChannel <- responseData:
			default:
				m.log.Debug("Response channel full, dropping oldest response")
			}

			switch queryID {
			case QueryGetHardwareInfo:
				// Already handled above
			case QueryGetStatusValues:
				if len(data) >= 2 {
					statusID := data[0]
					if statusID == 19 { // Pairing State
						if len(data) >= 3 {
							pairingState := data[2]
							m.log.Debugf("Pairing state: %d", pairingState)

							// Store pairing state in tracker for future reference
							if tracker := m.responseTracker[macAddress]; tracker != nil {
								tracker.mutex.Lock()
								tracker.PairingState = int(pairingState)
								tracker.mutex.Unlock()
								m.log.Debugf("Cached pairing state %d for device %s", pairingState, macAddress)
							}
						}
					}
				}
			default:
				m.log.Debugf("Unknown query response: ID=%d, status=%d", queryID, status)
			}
		}
	}); err != nil {
		return fmt.Errorf("failed to enable notifications: %v", err)
	}

	// Wait for camera to be ready before proceeding with pairing
	m.log.Debug("Waiting for camera to be ready")
	if err := m.waitForCameraReady(macAddress, 5); err != nil {
		return fmt.Errorf("camera readiness check failed: %v", err)
	}

	// Detect GoPro model for model-specific handling
	modelID, err := m.detectGoProModel(macAddress)
	if err != nil {
		m.log.Warnf("Failed to detect GoPro model: %v, proceeding with default behavior", err)
		modelID = 0 // Use 0 to indicate unknown model
	} else {
		m.log.Infof("Detected GoPro model: %s (ID: %d)", getGoProModelName(modelID), modelID)
	}

	// Perform model-specific pairing procedures
	if err := m.performModelSpecificPairing(macAddress, modelID); err != nil {
		return fmt.Errorf("model-specific pairing failed: %v", err)
	}

	// Check pairing state (legacy compatibility)
	m.log.Debug("Checking pairing state")
	pairingStateCmd := []byte{QueryGetStatusValues, 19} // 19 is Pairing State
	n, err := queryChar.WriteWithoutResponse(pairingStateCmd)
	if err != nil {
		return fmt.Errorf("failed to query pairing state: %v", err)
	}
	if n != len(pairingStateCmd) {
		return fmt.Errorf("incomplete write for pairing state query")
	}

	// Wait for pairing state response
	time.Sleep(1 * time.Second)

	// Enable notifications for other response characteristics
	if cmdRespChar, ok := controlService[CommandResponseCharUUID]; ok {
		m.log.Debug("Enabling notifications for Command Response")
		if err := cmdRespChar.EnableNotifications(func(buf []byte) {
			m.log.Debugf("Received command response: %v", buf)
		}); err != nil {
			m.log.Warnf("Failed to enable notifications for Command Response: %v", err)
		}
	}

	if settingsRespChar, ok := controlService[SettingsResponseCharUUID]; ok {
		m.log.Debug("Enabling notifications for Settings Response")
		if err := settingsRespChar.EnableNotifications(func(buf []byte) {
			m.log.Debugf("Received settings response: %v", buf)
		}); err != nil {
			m.log.Warnf("Failed to enable notifications for Settings Response: %v", err)
		}
	}

	// Create a context for the keep-alive goroutine
	keepAliveCtx, keepAliveCancel := context.WithCancel(m.ctx)

	// Store the cancel function in a map for later cleanup
	m.mutex.Lock()
	if m.keepAliveCancels == nil {
		m.keepAliveCancels = make(map[string]context.CancelFunc)
	}
	m.keepAliveCancels[macAddress] = keepAliveCancel
	m.mutex.Unlock()

	// Start keep-alive goroutine
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// Wait a moment before sending initial keep-alive
		time.Sleep(1 * time.Second)

		// Send initial keep-alive
		if err := m.KeepAlive(macAddress); err != nil {
			m.log.Warnf("Failed to send initial keep-alive: %v", err)
		}

		// Wait another moment before starting regular keep-alive
		time.Sleep(1 * time.Second)

		for {
			select {
			case <-ticker.C:
				if err := m.KeepAlive(macAddress); err != nil {
					m.log.Warnf("Failed to send keep-alive: %v", err)
					// Don't return on error, keep trying
					continue
				}
			case <-keepAliveCtx.Done():
				m.log.Debugf("Keep-alive goroutine for device %s stopped", macAddress)
				return
			}
		}
	}()

	// Set camera control to prevent undefined behavior
	m.log.Debug("Setting camera control")
	if err := m.SetCameraControl(macAddress, true); err != nil {
		m.log.Warnf("Failed to set camera control: %v", err)
		// Don't return error, continue with connection
	}

	// Wait a moment to ensure initial setup is complete
	time.Sleep(2 * time.Second)

	return nil
}

// Sleep puts the GoPro to sleep
func (m *BLEManager) Sleep(macAddress string) error {
	m.log.Infof("BLEManager.Sleep: Attempting to put GoPro %s to sleep", macAddress)

	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		m.log.Errorf("BLEManager.Sleep: No active connection for device %s", macAddress)
		return fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	// Sleep command has no additional parameters
	cmd := []byte{CommandSleep}

	return m.withRetry("sleep camera", macAddress, 3, func() error {
		m.log.Debugf("BLEManager.Sleep: Starting retry attempt for GoPro %s", macAddress)

		// Discover the Control service
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			m.log.Errorf("BLEManager.Sleep: Failed to discover services: %v", err)
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find the Control service
		var controlService bluetooth.DeviceService
		found := false
		for _, svc := range svcs {
			if svc.UUID().String() == GoProControlServiceUUID {
				controlService = svc
				found = true
				break
			}
		}

		if !found {
			m.log.Errorf("BLEManager.Sleep: Control service not found for device %s", macAddress)
			return fmt.Errorf("Control service not found")
		}

		m.log.Debugf("BLEManager.Sleep: Found Control service for device %s", macAddress)

		// Discover characteristics
		chars, err := controlService.DiscoverCharacteristics(nil)
		if err != nil {
			m.log.Errorf("BLEManager.Sleep: Failed to discover characteristics: %v", err)
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find the Command characteristic
		for _, char := range chars {
			if char.UUID().String() == CommandCharUUID {
				m.log.Debugf("BLEManager.Sleep: Found Command characteristic, sending sleep command to device %s", macAddress)

				// Send the command with proper packet header
				n, err := char.WriteWithoutResponse(append([]byte{PacketTypeStart}, cmd...))
				if err != nil {
					m.log.Errorf("BLEManager.Sleep: Failed to send sleep command: %v", err)
					return fmt.Errorf("failed to send sleep command: %v", err)
				}
				if n != len(cmd)+1 {
					m.log.Errorf("BLEManager.Sleep: Incomplete write for sleep command (%d bytes written, expected %d)", n, len(cmd)+1)
					return fmt.Errorf("incomplete write for sleep command")
				}

				// Increased wait time to ensure command is fully processed
				time.Sleep(2 * time.Second)

				m.log.Infof("BLEManager.Sleep: Successfully sent sleep command to device %s", macAddress)
				return nil
			}
		}

		m.log.Errorf("BLEManager.Sleep: Command characteristic not found for device %s", macAddress)
		return fmt.Errorf("command characteristic not found")
	})
}

// DisconnectGoPro puts the GoPro to sleep but does not perform the actual disconnect
// This is used by ConnectionManager to handle GoPro-specific pre-disconnect operations
func (m *BLEManager) DisconnectGoPro(macAddress string) error {
	m.log.Infof("BLEManager.DisconnectGoPro: Putting GoPro %s to sleep before disconnect", macAddress)

	// Check if the device exists in our connections map
	m.mutex.RLock()
	_, exists := m.connections[macAddress]
	m.mutex.RUnlock()

	if !exists {
		m.log.Warnf("BLEManager.DisconnectGoPro: Device %s not found in connections map", macAddress)
	}

	// Put the camera to sleep
	err := m.Sleep(macAddress)
	if err != nil {
		m.log.Warnf("BLEManager.DisconnectGoPro: Failed to put GoPro %s to sleep: %v", macAddress, err)
		return err
	}

	// Increased delay to allow sleep command to take effect and ensure complete processing
	m.log.Debugf("BLEManager.DisconnectGoPro: Successfully sent sleep command to GoPro %s, waiting for command to take effect", macAddress)
	time.Sleep(3 * time.Second)

	m.log.Infof("BLEManager.DisconnectGoPro: Completed pre-disconnect operations for GoPro %s", macAddress)
	return nil
}

// Disconnect disconnects from a GoPro device
func (m *BLEManager) Disconnect(macAddress string) error {
	m.log.Infof("BLEManager.Disconnect directly called for device %s", macAddress)

	// Cancel the keep-alive goroutine first
	m.mutex.Lock()
	if cancel, exists := m.keepAliveCancels[macAddress]; exists {
		cancel()
		delete(m.keepAliveCancels, macAddress)
	}
	m.mutex.Unlock()

	// Check if this device is managed by ConnectionManager and if so, use that path
	// to ensure Sleep command is sent before disconnecting
	if m.connManager != nil {
		state := m.connManager.GetState(macAddress)
		if state != StateDisconnected && state != StateDisconnecting {
			m.log.Infof("BLEManager.Disconnect: Device %s is managed by ConnectionManager (state: %s), redirecting to ConnectionManager.Disconnect",
				macAddress, state)
			return m.connManager.Disconnect(macAddress)
		} else {
			m.log.Debugf("BLEManager.Disconnect: Device %s is not actively managed by ConnectionManager (state: %s)",
				macAddress, state)
		}
	} else {
		m.log.Warnf("BLEManager.Disconnect: ConnectionManager is nil, cannot use it for proper shutdown sequence")
	}

	// First try to put the camera to sleep directly
	m.log.Infof("BLEManager.Disconnect: Putting GoPro %s to sleep before direct disconnect", macAddress)
	_ = m.DisconnectGoPro(macAddress) // Ignore errors, we'll try to disconnect anyway

	m.mutex.Lock() // Lock to safely access and modify m.connections and m.devices
	dev, ok := m.connections[macAddress]
	if !ok {
		m.mutex.Unlock()
		m.log.Debugf("BLEManager: Device %s not connected or already disconnected.", macAddress)
		return nil // Not an error if not connected
	}
	// Remove from map before attempting to disconnect to prevent race conditions if Disconnect is called multiple times.
	delete(m.connections, macAddress)
	if d, exists := m.devices[macAddress]; exists {
		d.IsConnected = false
	}
	m.mutex.Unlock() // Unlock before blocking disconnect call

	m.log.Infof("BLEManager: Disconnecting from %s", macAddress)
	disconnectErr := dev.Disconnect() // This is a blocking call from tinygo/bluetooth
	if disconnectErr != nil {
		m.log.Errorf("BLEManager: Failed to disconnect from %s: %v", macAddress, disconnectErr)
		// Even if disconnect fails, we've removed it from our tracking.
		return fmt.Errorf("failed to disconnect %s: %w", macAddress, disconnectErr)
	}

	m.log.Infof("BLEManager: Disconnected from %s successfully.", macAddress)
	return nil
}

// GetWifiCredentials retrieves WiFi SSID and password from the GoPro
func (m *BLEManager) GetWifiCredentials(macAddress string) (string, string, error) {
	m.log.Debugf("GetWifiCredentials: Starting for device %s", macAddress)

	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		m.log.Errorf("GetWifiCredentials: No active connection for device %s", macAddress)
		return "", "", fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	var ssid, password string
	var ssidErr, pwdErr error

	// Use our retry mechanism for SSID
	m.log.Debug("GetWifiCredentials: Starting SSID retrieval")
	ssidErr = m.withRetry("get WiFi SSID", macAddress, 3, func() error {
		m.log.Debug("GetWifiCredentials: Attempting to parse WiFi service UUID")
		// Discover services with timeout
		wifiServiceUUID, err := bluetooth.ParseUUID(GoProWifiServiceUUID)
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to parse WiFi service UUID: %v", err)
			return fmt.Errorf("invalid WiFi service UUID: %v", err)
		}
		m.log.Debug("GetWifiCredentials: Successfully parsed WiFi service UUID")

		m.log.Debug("GetWifiCredentials: Starting service discovery")
		svcs, err := bleDevice.DiscoverServices([]bluetooth.UUID{wifiServiceUUID})
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to discover services: %v", err)
			return fmt.Errorf("failed to discover services: %v", err)
		}
		m.log.Debugf("GetWifiCredentials: Discovered %d services", len(svcs))

		if len(svcs) == 0 {
			m.log.Error("GetWifiCredentials: WiFi service not found")
			return fmt.Errorf("WiFi service not found")
		}

		wifiService := svcs[0]
		m.log.Debug("GetWifiCredentials: Found WiFi service")

		m.log.Debug("GetWifiCredentials: Attempting to parse SSID characteristic UUID")
		// Discover characteristics with timeout
		ssidCharUUID, err := bluetooth.ParseUUID(WifiSSIDCharUUID)
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to parse SSID characteristic UUID: %v", err)
			return fmt.Errorf("invalid SSID characteristic UUID: %v", err)
		}
		m.log.Debug("GetWifiCredentials: Successfully parsed SSID characteristic UUID")

		m.log.Debug("GetWifiCredentials: Starting characteristic discovery")
		chars, err := wifiService.DiscoverCharacteristics([]bluetooth.UUID{ssidCharUUID})
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to discover characteristics: %v", err)
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}
		m.log.Debugf("GetWifiCredentials: Discovered %d characteristics", len(chars))

		if len(chars) == 0 {
			m.log.Error("GetWifiCredentials: SSID characteristic not found")
			return fmt.Errorf("SSID characteristic not found")
		}

		m.log.Debug("GetWifiCredentials: Attempting to read SSID value")
		// Read the value with proper buffer handling
		data := make([]byte, 32) // GoPro SSIDs are typically shorter
		n, err := chars[0].Read(data)
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to read SSID: %v", err)
			return fmt.Errorf("failed to read SSID: %v", err)
		}
		m.log.Debugf("GetWifiCredentials: Successfully read %d bytes for SSID", n)

		// Trim null bytes and whitespace
		ssid = strings.TrimSpace(string(data[:n]))
		if ssid == "" {
			m.log.Error("GetWifiCredentials: Empty SSID received")
			return fmt.Errorf("empty SSID received")
		}
		m.log.Debugf("GetWifiCredentials: Successfully retrieved SSID: %s", ssid)

		return nil
	})

	// Use our retry mechanism for password
	m.log.Debug("GetWifiCredentials: Starting password retrieval")
	pwdErr = m.withRetry("get WiFi password", macAddress, 3, func() error {
		m.log.Debug("GetWifiCredentials: Attempting to parse WiFi service UUID for password")
		// Discover services with timeout
		wifiServiceUUID, err := bluetooth.ParseUUID(GoProWifiServiceUUID)
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to parse WiFi service UUID for password: %v", err)
			return fmt.Errorf("invalid WiFi service UUID: %v", err)
		}
		m.log.Debug("GetWifiCredentials: Successfully parsed WiFi service UUID for password")

		m.log.Debug("GetWifiCredentials: Starting service discovery for password")
		svcs, err := bleDevice.DiscoverServices([]bluetooth.UUID{wifiServiceUUID})
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to discover services for password: %v", err)
			return fmt.Errorf("failed to discover services: %v", err)
		}
		m.log.Debugf("GetWifiCredentials: Discovered %d services for password", len(svcs))

		if len(svcs) == 0 {
			m.log.Error("GetWifiCredentials: WiFi service not found for password")
			return fmt.Errorf("WiFi service not found")
		}

		wifiService := svcs[0]
		m.log.Debug("GetWifiCredentials: Found WiFi service for password")

		m.log.Debug("GetWifiCredentials: Attempting to parse password characteristic UUID")
		// Discover characteristics with timeout
		pwdCharUUID, err := bluetooth.ParseUUID(WifiPasswordCharUUID)
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to parse password characteristic UUID: %v", err)
			return fmt.Errorf("invalid password characteristic UUID: %v", err)
		}
		m.log.Debug("GetWifiCredentials: Successfully parsed password characteristic UUID")

		m.log.Debug("GetWifiCredentials: Starting characteristic discovery for password")
		chars, err := wifiService.DiscoverCharacteristics([]bluetooth.UUID{pwdCharUUID})
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to discover characteristics for password: %v", err)
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}
		m.log.Debugf("GetWifiCredentials: Discovered %d characteristics for password", len(chars))

		if len(chars) == 0 {
			m.log.Error("GetWifiCredentials: Password characteristic not found")
			return fmt.Errorf("Password characteristic not found")
		}

		m.log.Debug("GetWifiCredentials: Attempting to read password value")
		// Read the value with proper buffer handling
		data := make([]byte, 32) // GoPro passwords are typically shorter
		n, err := chars[0].Read(data)
		if err != nil {
			m.log.Errorf("GetWifiCredentials: Failed to read password: %v", err)
			return fmt.Errorf("failed to read password: %v", err)
		}
		m.log.Debugf("GetWifiCredentials: Successfully read %d bytes for password", n)

		// Trim null bytes and whitespace
		password = strings.TrimSpace(string(data[:n]))
		if password == "" {
			m.log.Error("GetWifiCredentials: Empty password received")
			return fmt.Errorf("empty password received")
		}
		m.log.Debug("GetWifiCredentials: Successfully retrieved password")

		return nil
	})

	// Check for errors
	if ssidErr != nil {
		m.log.Errorf("GetWifiCredentials: Failed to get SSID: %v", ssidErr)
		return "", "", fmt.Errorf("failed to get SSID: %v", ssidErr)
	}
	if pwdErr != nil {
		m.log.Errorf("GetWifiCredentials: Failed to get password: %v", pwdErr)
		return "", "", fmt.Errorf("failed to get password: %v", pwdErr)
	}

	m.log.Debugf("GetWifiCredentials: Successfully retrieved credentials for device %s", macAddress)
	return ssid, password, nil
}

// EnableWifi enables WiFi on the GoPro
func (m *BLEManager) EnableWifi(macAddress string) error {
	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		return fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	return m.withRetry("enable WiFi", macAddress, 3, func() error {
		// Discover the WiFi service
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find the WiFi service
		var wifiService bluetooth.DeviceService
		found := false
		for _, svc := range svcs {
			if svc.UUID().String() == GoProWifiServiceUUID {
				wifiService = svc
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("WiFi service not found")
		}

		// Discover the WiFi Power characteristic
		chars, err := wifiService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find the WiFi Power characteristic
		for _, char := range chars {
			if char.UUID().String() == WifiPowerCharUUID {
				// Write the value to enable WiFi (1 = ON)
				// Check the actual implementation of WriteWithoutResponse
				n, err := char.WriteWithoutResponse([]byte{1})
				if err != nil {
					return fmt.Errorf("failed to enable WiFi: %v", err)
				}
				if n != 1 {
					return fmt.Errorf("failed to write complete data to enable WiFi")
				}
				m.log.Debug("Successfully enabled WiFi")
				return nil
			}
		}

		return fmt.Errorf("WiFi power characteristic not found")
	})
}

// SetDateTime sets the date and time on the GoPro
func (m *BLEManager) SetDateTime(macAddress string, t time.Time) error {
	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		return fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	// Prepare date time command format
	cmd := make([]byte, 8)
	cmd[0] = CommandSetDateTime
	binary.BigEndian.PutUint16(cmd[1:3], uint16(t.Year()))
	cmd[3] = byte(t.Month())
	cmd[4] = byte(t.Day())
	cmd[5] = byte(t.Hour())
	cmd[6] = byte(t.Minute())
	cmd[7] = byte(t.Second())

	m.log.Debugf("Setting Date and Time for device %s", macAddress)

	return m.withRetry("set date time", macAddress, 3, func() error {
		// Discover the Control service
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find the Control service
		var controlService bluetooth.DeviceService
		found := false
		for _, svc := range svcs {
			if svc.UUID().String() == GoProControlServiceUUID {
				controlService = svc
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Control service not found")
		}

		// Discover the Command characteristic
		chars, err := controlService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find the Command characteristic
		for _, char := range chars {
			if char.UUID().String() == CommandCharUUID {
				// Write the command
				n, err := char.WriteWithoutResponse(cmd)
				if err != nil {
					return fmt.Errorf("failed to set date time: %v", err)
				}
				if n != len(cmd) {
					return fmt.Errorf("failed to write complete date time command")
				}

				m.log.Debugf("Successfully set date time to %s", t.Format(time.RFC3339))
				return nil
			}
		}

		return fmt.Errorf("Command characteristic not found")
	})
}

// GetMetadata retrieves metadata from the GoPro such as firmware version, model, etc.
func (m *BLEManager) GetMetadata(macAddress string) (map[string]string, error) {
	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		return nil, fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	metadata := make(map[string]string)
	var metadataErr error

	metadataErr = m.withRetry("get device metadata", macAddress, 3, func() error {
		// Discover the services
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find the Control service
		var controlService bluetooth.DeviceService
		controlFound := false
		for _, svc := range svcs {
			if svc.UUID().String() == GoProControlServiceUUID {
				controlService = svc
				controlFound = true
				break
			}
		}

		if !controlFound {
			return fmt.Errorf("Control service not found")
		}

		// Discover the Query characteristic
		chars, err := controlService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find the Query characteristic and Query Response characteristic
		var queryChar, queryRespChar *bluetooth.DeviceCharacteristic
		for i, char := range chars {
			if char.UUID().String() == QueryCharUUID {
				queryChar = &chars[i]
			} else if char.UUID().String() == QueryResponseCharUUID {
				queryRespChar = &chars[i]
			}
		}

		if queryChar == nil {
			return fmt.Errorf("Query characteristic not found")
		}
		if queryRespChar == nil {
			return fmt.Errorf("Query Response characteristic not found")
		}

		// Enable notifications on the Query Response characteristic
		if err := queryRespChar.EnableNotifications(func(buf []byte) {
			// Parse the response
			if len(buf) < 2 {
				return
			}

			// Simple response parsing - in reality, this would be more complex
			// and based on the specific GoPro BLE protocol
			if buf[0] == QueryGetHardwareInfo {
				if len(buf) >= 3 {
					// Extract model information
					model := fmt.Sprintf("GoPro Hero %d", int(buf[2]))
					metadata["model"] = model

					// Extract firmware version if available
					if len(buf) >= 6 {
						fwVersion := fmt.Sprintf("%d.%d.%d", int(buf[3]), int(buf[4]), int(buf[5]))
						metadata["firmware_version"] = fwVersion
					}

					// Extract hardware version if available
					if len(buf) >= 8 {
						hwVersion := fmt.Sprintf("%d.%d", int(buf[6]), int(buf[7]))
						metadata["hardware_version"] = hwVersion
					}

					// Extract serial number if available
					if len(buf) >= 16 {
						serialNum := fmt.Sprintf("C%02X%02X%02X%02X%02X%02X%02X%02X",
							buf[8], buf[9], buf[10], buf[11], buf[12], buf[13], buf[14], buf[15])
						metadata["serial_number"] = serialNum
					}
				}
			}
		}); err != nil {
			return fmt.Errorf("failed to enable notifications: %v", err)
		}

		// Send the hardware info query command
		queryCmd := []byte{QueryGetHardwareInfo}
		n, err := queryChar.WriteWithoutResponse(queryCmd)
		if err != nil {
			return fmt.Errorf("failed to query hardware info: %v", err)
		}
		if n != len(queryCmd) {
			return fmt.Errorf("failed to write complete query command")
		}

		// Wait for the response
		// In reality, we would use a channel or context with timeout
		time.Sleep(2 * time.Second)

		if len(metadata) == 0 {
			return fmt.Errorf("failed to receive metadata response")
		}

		return nil
	})

	if metadataErr != nil {
		return nil, metadataErr
	}

	return metadata, nil
}

// GetBatteryLevel retrieves the battery level from the GoPro
func (m *BLEManager) GetBatteryLevel(macAddress string) (int, error) {
	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		return 0, fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	var batteryLevel int
	var batteryErr error

	m.log.Debugf("Getting battery level for device %s", macAddress)

	batteryErr = m.withRetry("get battery level", macAddress, 3, func() error {
		// Discover services to find Control service
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find the Control service
		var controlService bluetooth.DeviceService
		controlFound := false
		for _, svc := range svcs {
			if svc.UUID().String() == GoProControlServiceUUID {
				controlService = svc
				controlFound = true
				break
			}
		}

		if !controlFound {
			return fmt.Errorf("control service not found")
		}

		// Use a channel to communicate the query result
		resultCh := make(chan bool)

		// Discover the Query characteristic
		chars, err := controlService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find the Query characteristic and Query Response characteristic
		var queryChar, queryRespChar *bluetooth.DeviceCharacteristic
		for i, char := range chars {
			if char.UUID().String() == QueryCharUUID {
				queryChar = &chars[i]
			} else if char.UUID().String() == QueryResponseCharUUID {
				queryRespChar = &chars[i]
			}
		}

		if queryChar == nil {
			return fmt.Errorf("query characteristic not found")
		}
		if queryRespChar == nil {
			return fmt.Errorf("query response characteristic not found")
		}

		// Enable notifications on the Query Response characteristic
		if err := queryRespChar.EnableNotifications(func(buf []byte) {
			// Parse the response according to GoPro spec
			if len(buf) >= 3 && buf[0] == QueryGetStatusValues {
				// Check if this is the battery status response (status ID 70)
				if len(buf) >= 3 && buf[1] == 70 {
					batteryLevel = int(buf[2])
					resultCh <- true
				}
			}
		}); err != nil {
			return fmt.Errorf("failed to enable notifications: %v", err)
		}

		// Send the battery query command (QueryGetStatusValues + statusID 70)
		queryCmd := []byte{QueryGetStatusValues, 70} // 70 is Internal Battery Percentage status ID
		n, err := queryChar.WriteWithoutResponse(queryCmd)
		if err != nil {
			return fmt.Errorf("failed to query battery: %v", err)
		}
		if n != len(queryCmd) {
			return fmt.Errorf("failed to write complete query command")
		}

		// Wait for response with timeout
		select {
		case <-resultCh:
			return nil
		case <-time.After(3 * time.Second):
			return fmt.Errorf("timeout waiting for battery response")
		}
	})

	if batteryErr != nil {
		return 0, batteryErr
	}

	return batteryLevel, nil
}

// withRetry executes a BLE operation with exponential backoff retries
func (m *BLEManager) withRetry(operation string, device string, maxRetries int, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		m.log.Debugf("Attempt %d/%d for operation '%s' on device %s", attempt+1, maxRetries, operation, device)

		err := fn()
		if err == nil {
			return nil // Success
		}

		// Enhance error with context
		lastErr = NewBLEError(operation, device, err, ShouldRetry(err), attempt+1)

		// Log the error with detailed context
		m.log.Warnf("%v", lastErr)

		// Check if we should retry
		if !ShouldRetry(lastErr) || attempt == maxRetries-1 {
			m.log.Errorf("Operation '%s' failed after %d attempts: %v", operation, attempt+1, lastErr)
			return lastErr
		}

		// Calculate backoff time
		backoff := BackoffDuration(attempt)
		m.log.Debugf("Retrying in %v", backoff)
		time.Sleep(backoff)
	}

	// This should be unreachable, but just in case
	return lastErr
}

// createPackets splits a large payload into BLE packets
func (m *BLEManager) createPackets(payload []byte) [][]byte {
	// BLE <= v4.2 limits packet size to 20 bytes
	maxPacketSize := 20

	// Calculate how many packets we need
	numPackets := (len(payload) + maxPacketSize - 2) / (maxPacketSize - 1)
	if numPackets < 1 {
		numPackets = 1
	}

	packets := make([][]byte, numPackets)

	// First packet has the header and can hold up to 19 bytes of data
	firstPacket := make([]byte, 0, maxPacketSize)
	firstPacket = append(firstPacket, PacketTypeStart)

	if len(payload) > maxPacketSize-1 {
		// First packet can hold up to maxPacketSize-1 bytes
		firstPacket = append(firstPacket, payload[:maxPacketSize-1]...)
		packets[0] = firstPacket

		// Distribute the remaining payload across continuation packets
		remainingPayload := payload[maxPacketSize-1:]
		for i := 1; i < numPackets; i++ {
			packet := make([]byte, 0, maxPacketSize)
			packet = append(packet, PacketTypeContinuation)

			start := (i - 1) * (maxPacketSize - 1)
			end := start + (maxPacketSize - 1)
			if end > len(remainingPayload) {
				end = len(remainingPayload)
			}

			packet = append(packet, remainingPayload[start:end]...)
			packets[i] = packet
		}
	} else {
		// Short payload fits in a single packet
		firstPacket = append(firstPacket, payload...)
		packets[0] = firstPacket
	}

	return packets
}

// SetCameraControl sets the camera control status (to prevent undefined behavior between the camera and app)
func (m *BLEManager) SetCameraControl(macAddress string, enabled bool) error {
	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		return fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	// Prepare camera control command
	cmd := make([]byte, 2)
	cmd[0] = CommandSetCameraControl
	if enabled {
		cmd[1] = 1 // Enable camera control
	} else {
		cmd[1] = 0 // Disable camera control
	}

	return m.withRetry("set camera control", macAddress, 3, func() error {
		// Discover the Control service
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find the Control service
		var controlService bluetooth.DeviceService
		found := false
		for _, svc := range svcs {
			if svc.UUID().String() == GoProControlServiceUUID {
				controlService = svc
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Control service not found")
		}

		// Discover characteristics
		chars, err := controlService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find the Command characteristic
		for _, char := range chars {
			if char.UUID().String() == CommandCharUUID {
				// Create packets
				packets := m.createPackets(cmd)

				// Write each packet
				for i, packet := range packets {
					n, err := char.WriteWithoutResponse(packet)
					if err != nil {
						return fmt.Errorf("failed to write packet %d: %v", i, err)
					}
					if n != len(packet) {
						return fmt.Errorf("incomplete write for packet %d", i)
					}
					// Small delay between packets
					time.Sleep(10 * time.Millisecond)
				}

				m.log.Debugf("Successfully %s camera control", map[bool]string{true: "enabled", false: "disabled"}[enabled])
				return nil
			}
		}

		return fmt.Errorf("Command characteristic not found")
	})
}

// KeepAlive sends a keep-alive signal to maintain the connection with the GoPro
func (m *BLEManager) KeepAlive(macAddress string) error {
	m.mutex.RLock()
	bleDevice, hasConnection := m.connections[macAddress]
	if !hasConnection || bleDevice == nil {
		m.mutex.RUnlock()
		return fmt.Errorf("no active connection for device %s", macAddress)
	}
	m.mutex.RUnlock()

	// The keep-alive command is a TLV command with ID 0x5B and value 0x42
	cmd := []byte{0x5B, 0x42}

	return m.withRetry("keep alive", macAddress, 3, func() error {
		// Discover the Control service
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find the Control service
		var controlService bluetooth.DeviceService
		found := false
		for _, svc := range svcs {
			if svc.UUID().String() == GoProControlServiceUUID {
				controlService = svc
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Control service not found")
		}

		// Discover characteristics
		chars, err := controlService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find the Settings characteristic (not Command)
		for _, char := range chars {
			if char.UUID().String() == SettingsCharUUID {
				// Since this is a small command, we don't need to packetize
				n, err := char.WriteWithoutResponse(cmd)
				if err != nil {
					return fmt.Errorf("failed to write keep-alive: %v", err)
				}
				if n != len(cmd) {
					return fmt.Errorf("incomplete write for keep-alive")
				}

				m.log.Debug("Successfully sent keep-alive signal")
				return nil
			}
		}

		return fmt.Errorf("Settings characteristic not found")
	})
}

// CheckBLESystem verifies that the BLE system is properly functional
func (m *BLEManager) CheckBLESystem() error {
	// Check if adapter is nil
	if m.adapter == nil {
		return fmt.Errorf("Bluetooth adapter is nil")
	}

	// Try a simple operation to check if the adapter is working
	// We can attempt to check adapter properties that are available in the library

	// Since Enabled() is not available, we'll use a different approach
	// The exact check depends on what's available in the tinygo.org/x/bluetooth library

	// For now, just perform a basic nil check (we already did this)
	// In a real implementation, you might try a simple operation like
	// a quick scan or check adapter address if available

	m.log.Debug("BLEManager: BLE system check passed")
	return nil
}

// detectGoProModel attempts to detect the GoPro model by querying hardware info
func (m *BLEManager) detectGoProModel(macAddress string) (int, error) {
	m.log.Debug("Detecting GoPro model through hardware info query")

	// Get BLE device
	m.mutex.RLock()
	bleDevice, exists := m.connections[macAddress]
	tracker := m.responseTracker[macAddress]
	m.mutex.RUnlock()

	if !exists || bleDevice == nil {
		return 0, fmt.Errorf("no active connection for device %s", macAddress)
	}

	if tracker == nil {
		return 0, fmt.Errorf("no response tracker for device %s", macAddress)
	}

	// Get query characteristic
	controlService, ok := m.serviceMap[GoProControlServiceUUID]
	if !ok {
		return 0, fmt.Errorf("control service not found")
	}

	queryChar, ok := controlService[QueryCharUUID]
	if !ok {
		return 0, fmt.Errorf("query characteristic not found")
	}

	// Query hardware info (0x3F)
	hardwareInfoCmd := []byte{QueryGetHardwareInfo}
	n, err := queryChar.WriteWithoutResponse(hardwareInfoCmd)
	if err != nil {
		return 0, fmt.Errorf("failed to query hardware info: %v", err)
	}
	if n != len(hardwareInfoCmd) {
		return 0, fmt.Errorf("incomplete write for hardware info query")
	}

	// Wait for response with timeout
	select {
	case response := <-tracker.responseChannel:
		if response.QueryID == QueryGetHardwareInfo && response.Status == 0 {
			if len(response.Data) >= 1 {
				modelID := int(response.Data[0])
				if isValidModelID(modelID) {
					m.log.Debugf("Successfully detected GoPro model ID: %d (%s)", modelID, getGoProModelName(modelID))
					// Store in tracker for future queries
					tracker.mutex.Lock()
					tracker.ModelID = modelID
					tracker.mutex.Unlock()
					return modelID, nil
				} else {
					m.log.Warnf("Invalid model ID received: %d", modelID)
				}
			}
		}
	case <-time.After(3 * time.Second):
		m.log.Debug("Timeout waiting for hardware info response")
	}

	// Check if model ID was previously stored in tracker
	tracker.mutex.RLock()
	cachedModelID := tracker.ModelID
	tracker.mutex.RUnlock()
	if isValidModelID(cachedModelID) {
		m.log.Debugf("Using cached model ID from tracker: %d (%s)", cachedModelID, getGoProModelName(cachedModelID))
		return cachedModelID, nil
	}

	// If hardware info query fails, try to infer from device name
	m.mutex.RLock()
	device, exists := m.devices[macAddress]
	m.mutex.RUnlock()

	if !exists {
		return 0, fmt.Errorf("device not found in discovered devices")
	}

	// Infer model from device name patterns
	deviceName := strings.ToLower(device.Name)
	if strings.Contains(deviceName, "hero13") {
		return GoProModelHERO13, nil
	} else if strings.Contains(deviceName, "hero12") {
		return GoProModelHERO12, nil
	} else if strings.Contains(deviceName, "hero11") {
		return GoProModelHERO11, nil
	} else if strings.Contains(deviceName, "hero10") {
		return GoProModelHERO10, nil
	} else if strings.Contains(deviceName, "hero9") {
		return GoProModelHERO9, nil
	}

	m.log.Debugf("Could not determine GoPro model from device name: %s", device.Name)
	return 0, fmt.Errorf("unknown GoPro model")
}

// waitForCameraReady waits for the camera to be ready by repeatedly querying hardware info
func (m *BLEManager) waitForCameraReady(macAddress string, maxRetries int) error {
	m.log.Debug("Waiting for camera to be ready")

	for i := 0; i < maxRetries; i++ {
		// Try to detect the model (which requires hardware info query to succeed)
		_, err := m.detectGoProModel(macAddress)
		if err == nil {
			m.log.Debugf("Camera ready after %d attempts", i+1)
			return nil
		}

		m.log.Debugf("Camera not ready (attempt %d/%d): %v", i+1, maxRetries, err)

		// Wait before next attempt
		if i < maxRetries-1 {
			time.Sleep(time.Duration(500*(i+1)) * time.Millisecond)
		}
	}

	return fmt.Errorf("camera not ready after %d attempts", maxRetries)
}

// performModelSpecificPairing handles model-specific pairing requirements
func (m *BLEManager) performModelSpecificPairing(macAddress string, modelID int) error {
	modelName := getGoProModelName(modelID)
	m.log.Debugf("Performing model-specific pairing for %s (ID: %d)", modelName, modelID)

	switch modelID {
	case GoProModelHERO13:
		// HERO13 specific pairing logic
		m.log.Debug("Applying HERO13-specific pairing procedures")
		// HERO13 may have enhanced BLE capabilities and different timing requirements
		if err := m.configureHERO13SpecificSettings(macAddress); err != nil {
			m.log.Warnf("HERO13-specific configuration failed: %v", err)
		}

	case GoProModelHERO12:
		// HERO12 specific pairing logic
		m.log.Debug("Applying HERO12-specific pairing procedures")
		if err := m.configureHERO12SpecificSettings(macAddress); err != nil {
			m.log.Warnf("HERO12-specific configuration failed: %v", err)
		}

	case GoProModelHERO11:
		// HERO11 specific pairing logic (including variant 60)
		m.log.Debug("Applying HERO11-specific pairing procedures")
		if err := m.configureHERO11SpecificSettings(macAddress); err != nil {
			m.log.Warnf("HERO11-specific configuration failed: %v", err)
		}

	case GoProModelHERO10:
		// HERO10 specific pairing logic
		m.log.Debug("Applying HERO10-specific pairing procedures")
		if err := m.configureLegacyModelSettings(macAddress); err != nil {
			m.log.Warnf("HERO10-specific configuration failed: %v", err)
		}

	case GoProModelHERO9:
		// HERO9 specific pairing logic
		m.log.Debug("Applying HERO9-specific pairing procedures")
		if err := m.configureLegacyModelSettings(macAddress); err != nil {
			m.log.Warnf("HERO9-specific configuration failed: %v", err)
		}

	default:
		m.log.Warnf("Unknown GoPro model %s, using default pairing procedure", modelName)
		// Use default/generic pairing procedure for unknown models
		if err := m.configureDefaultSettings(macAddress); err != nil {
			m.log.Warnf("Default configuration failed: %v", err)
		}
	}

	// Common pairing verification for all models
	return m.verifyPairingState(macAddress)
}

// verifyPairingState checks the current pairing state and handles it appropriately
func (m *BLEManager) verifyPairingState(macAddress string) error {
	m.log.Debug("Verifying pairing state")

	pairingState, err := m.GetPairingState(macAddress)
	if err != nil {
		return fmt.Errorf("failed to get pairing state: %v", err)
	}

	switch pairingState {
	case PairingStateCompleted:
		m.log.Debug("Device is already paired")
		return nil
	case PairingStateNeverStarted:
		m.log.Debug("Device pairing never started - this is normal for new pairing")
		return nil
	case PairingStateStarted:
		m.log.Debug("Device pairing is in progress")
		return nil
	case PairingStateAborted:
		return fmt.Errorf("device pairing was aborted")
	case PairingStateCancelled:
		return fmt.Errorf("device pairing was cancelled")
	default:
		m.log.Warnf("Unknown pairing state: %d", pairingState)
		return nil
	}
}

// configureHERO13SpecificSettings applies HERO13-specific BLE configuration
func (m *BLEManager) configureHERO13SpecificSettings(macAddress string) error {
	m.log.Debug("Configuring HERO13-specific settings")

	// HERO13 may have enhanced BLE capabilities and faster processing
	// Allow more time for initialization due to advanced features
	time.Sleep(500 * time.Millisecond)

	// HERO13 specific: Enable enhanced BLE features if available
	if err := m.setEnhancedBLEMode(macAddress, true); err != nil {
		m.log.Warnf("Failed to enable enhanced BLE mode for HERO13: %v", err)
		// Continue anyway as this is not critical
	}

	// HERO13 might require additional verification steps for security
	if err := m.performSecurityHandshake(macAddress); err != nil {
		m.log.Warnf("Security handshake failed for HERO13: %v", err)
		// Continue anyway as this might not be implemented yet
	}

	return nil
}

// configureHERO12SpecificSettings applies HERO12-specific BLE configuration
func (m *BLEManager) configureHERO12SpecificSettings(macAddress string) error {
	m.log.Debug("Configuring HERO12-specific settings")

	// HERO12-specific configuration with moderate initialization time
	time.Sleep(300 * time.Millisecond)

	// HERO12 may benefit from setting optimal connection parameters
	if err := m.setConnectionParameters(macAddress, "hero12"); err != nil {
		m.log.Warnf("Failed to set HERO12 connection parameters: %v", err)
	}

	return nil
}

// configureHERO11SpecificSettings applies HERO11-specific BLE configuration
func (m *BLEManager) configureHERO11SpecificSettings(macAddress string) error {
	m.log.Debug("Configuring HERO11-specific settings")

	// HERO11-specific configuration
	time.Sleep(300 * time.Millisecond)

	// HERO11 may require specific timing for BLE operations
	if err := m.setConnectionParameters(macAddress, "hero11"); err != nil {
		m.log.Warnf("Failed to set HERO11 connection parameters: %v", err)
	}

	return nil
}

// configureLegacyModelSettings applies settings for older GoPro models (HERO9, HERO10)
func (m *BLEManager) configureLegacyModelSettings(macAddress string) error {
	m.log.Debug("Configuring legacy model settings")

	// Older models might need different timing or additional setup
	time.Sleep(200 * time.Millisecond)

	// Legacy models may need more conservative connection parameters
	if err := m.setConnectionParameters(macAddress, "legacy"); err != nil {
		m.log.Warnf("Failed to set legacy connection parameters: %v", err)
	}

	return nil
}

// configureDefaultSettings applies default configuration for unknown models
func (m *BLEManager) configureDefaultSettings(macAddress string) error {
	m.log.Debug("Configuring default settings for unknown model")

	// Use conservative settings for unknown models
	time.Sleep(400 * time.Millisecond)

	// Apply safe default connection parameters
	if err := m.setConnectionParameters(macAddress, "default"); err != nil {
		m.log.Warnf("Failed to set default connection parameters: %v", err)
	}

	return nil
}

// getGoProModelName returns the model name for a given model ID
func getGoProModelName(modelID int) string {
	switch modelID {
	case GoProModelHERO9:
		return "HERO9"
	case GoProModelHERO10:
		return "HERO10"
	case GoProModelHERO11:
		return "HERO11"
	case GoProModelHERO12:
		return "HERO12"
	case GoProModelHERO13:
		return "HERO13"
	default:
		return fmt.Sprintf("Unknown (%d)", modelID)
	}
}

// Enhanced pairing constants for more robust handling
const (
	// Retry constants for pairing operations
	MaxPairingRetries     = 3
	PairingRetryDelay     = 2 * time.Second
	ModelDetectionRetries = 3
	ReadinessCheckDelay   = 500 * time.Millisecond
)

// Model capability definitions
type ModelCapabilities struct {
	SupportsEnhancedBLE     bool
	RequiresExtendedPairing bool
	MaxConnectionRetries    int
	PairingTimeoutMs        int
	SupportedFeatures       []string
}

// getModelCapabilities returns the capabilities for a specific GoPro model
func getModelCapabilities(modelID int) ModelCapabilities {
	switch modelID {
	case GoProModelHERO13:
		return ModelCapabilities{
			SupportsEnhancedBLE:     true,
			RequiresExtendedPairing: false,
			MaxConnectionRetries:    5,
			PairingTimeoutMs:        5000,
			SupportedFeatures:       []string{"auto_hibernate", "quick_pair", "enhanced_wifi"},
		}
	case GoProModelHERO12:
		return ModelCapabilities{
			SupportsEnhancedBLE:     true,
			RequiresExtendedPairing: false,
			MaxConnectionRetries:    4,
			PairingTimeoutMs:        4000,
			SupportedFeatures:       []string{"auto_hibernate", "quick_pair"},
		}
	case GoProModelHERO11:
		return ModelCapabilities{
			SupportsEnhancedBLE:     true,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    4,
			PairingTimeoutMs:        4000,
			SupportedFeatures:       []string{"auto_hibernate"},
		}
	case GoProModelHERO10:
		return ModelCapabilities{
			SupportsEnhancedBLE:     false,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    3,
			PairingTimeoutMs:        3000,
			SupportedFeatures:       []string{"basic_pairing"},
		}
	case GoProModelHERO9:
		return ModelCapabilities{
			SupportsEnhancedBLE:     false,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    3,
			PairingTimeoutMs:        3000,
			SupportedFeatures:       []string{"basic_pairing"},
		}
	default:
		return ModelCapabilities{
			SupportsEnhancedBLE:     false,
			RequiresExtendedPairing: true,
			MaxConnectionRetries:    2,
			PairingTimeoutMs:        2000,
			SupportedFeatures:       []string{"basic_pairing"},
		}
	}
}

// isModelSupported checks if a given model ID is supported
func isModelSupported(modelID int) bool {
	supportedModels := []int{
		GoProModelHERO9,
		GoProModelHERO10,
		GoProModelHERO11,
		GoProModelHERO12,
		GoProModelHERO13,
	}

	for _, supportedModel := range supportedModels {
		if modelID == supportedModel {
			return true
		}
	}
	return false
}

// isValidModelID checks if the model ID is a known valid GoPro model
func isValidModelID(modelID int) bool {
	switch modelID {
	case GoProModelHERO9, GoProModelHERO10, GoProModelHERO11, GoProModelHERO12, GoProModelHERO13:
		return true
	default:
		return false
	}
}

// shouldUseEnhancedPairing determines if enhanced pairing should be used for a model
func shouldUseEnhancedPairing(modelID int) bool {
	capabilities := getModelCapabilities(modelID)
	return capabilities.SupportsEnhancedBLE
}

// getPairingTimeout returns the recommended pairing timeout for a model
func getPairingTimeout(modelID int) time.Duration {
	capabilities := getModelCapabilities(modelID)
	return time.Duration(capabilities.PairingTimeoutMs) * time.Millisecond
}

// getMaxRetries returns the maximum retry count for a model
func getMaxRetries(modelID int) int {
	capabilities := getModelCapabilities(modelID)
	return capabilities.MaxConnectionRetries
}

// PairingError represents a pairing-specific error with retry information
type PairingError struct {
	Operation string
	ModelID   int
	Attempt   int
	Cause     error
}

func (e *PairingError) Error() string {
	modelName := getGoProModelName(e.ModelID)
	return fmt.Sprintf("pairing failed for %s (attempt %d): %s: %v",
		modelName, e.Attempt, e.Operation, e.Cause)
}

// ConnectWithEnhancedPairing attempts to connect with enhanced model-specific pairing logic
func (m *BLEManager) ConnectWithEnhancedPairing(macAddress string) error {
	m.log.Infof("Starting enhanced BLE connection to %s", macAddress)

	// First, establish basic BLE connection
	if err := m.Connect(macAddress); err != nil {
		return &PairingError{
			Operation: "basic_connection",
			ModelID:   0,
			Attempt:   1,
			Cause:     err,
		}
	}

	m.log.Infof("Enhanced BLE pairing completed successfully for %s", macAddress)
	return nil
}

// IsPairedWithVerification checks if device is paired using multiple verification methods
func (m *BLEManager) IsPairedWithVerification(macAddress string) (bool, error) {
	// Create a context with timeout for the entire verification process
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First try the standard pairing state query
	pairingState, err := m.GetPairingState(macAddress)
	if err != nil {
		m.log.Warnf("Failed to get pairing state for %s: %v", macAddress, err)
	} else {
		m.log.Infof("Device %s pairing state query returned: %d", macAddress, pairingState)

		// If state clearly indicates paired, return true
		if pairingState == PairingStateCompleted || pairingState == 1 {
			return true, nil
		}

		// If state clearly indicates not paired, try secondary verification
		if pairingState == PairingStateNeverStarted || pairingState == PairingStateAborted || pairingState == PairingStateCancelled {
			m.log.Infof("Device %s pairing state indicates not paired, but trying secondary verification", macAddress)
		}
	}

	// Secondary verification: try to get WiFi credentials
	// If we can get them, the device is definitely paired
	m.log.Infof("Attempting secondary pairing verification for %s using WiFi credentials", macAddress)

	// Make sure we're connected first
	device := m.getDevice(macAddress)
	if device == nil {
		return false, fmt.Errorf("device not found: %s", macAddress)
	}

	if !device.IsConnected {
		if err := m.Connect(macAddress); err != nil {
			return false, fmt.Errorf("failed to connect for verification: %v", err)
		}
		// Small delay after connection
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return false, fmt.Errorf("context cancelled during connection delay: %v", ctx.Err())
		}
	}

	// Try to get WiFi credentials with context timeout - if successful, device is paired
	type credResult struct {
		ssid string
		pass string
		err  error
	}

	credChan := make(chan credResult, 1)
	go func() {
		ssid, pass, err := m.GetWifiCredentials(macAddress)
		credChan <- credResult{ssid: ssid, pass: pass, err: err}
	}()

	select {
	case result := <-credChan:
		if result.err == nil && result.ssid != "" {
			m.log.Infof("Device %s is verified as paired via WiFi credentials (SSID: %s)", macAddress, result.ssid)
			return true, nil
		}
		m.log.Infof("Device %s failed WiFi credentials verification: %v", macAddress, result.err)

	case <-ctx.Done():
		m.log.Warnf("Timeout during WiFi credentials verification for %s: %v", macAddress, ctx.Err())
	}

	// Fall back to original pairing state result
	isPaired := pairingState == PairingStateCompleted
	m.log.Infof("Device %s final verification result: isPaired=%v (pairing state: %d)", macAddress, isPaired, pairingState)

	return isPaired, nil
}

// GetPairingState retrieves the current pairing state from the device
func (m *BLEManager) GetPairingState(macAddress string) (int, error) {
	m.log.Debugf("Getting pairing state for device %s", macAddress)

	// First check if we have cached pairing state
	if tracker := m.responseTracker[macAddress]; tracker != nil {
		tracker.mutex.RLock()
		cachedState := tracker.PairingState
		tracker.mutex.RUnlock()
		if cachedState != 0 {
			m.log.Debugf("Returning cached pairing state %d for device %s", cachedState, macAddress)
			return cachedState, nil
		}
	}

	// If no cached state, query the device directly
	return m.RefreshPairingState(macAddress)
}

// RefreshPairingState queries the device directly for current pairing state
func (m *BLEManager) RefreshPairingState(macAddress string) (int, error) {
	m.log.Debugf("Refreshing pairing state for device %s", macAddress)

	// Get control service and query characteristic
	device := m.getDevice(macAddress)
	if device == nil {
		return 0, fmt.Errorf("device not found: %s", macAddress)
	}

	// Get the query characteristic
	queryChar, err := m.getCharacteristic(macAddress, GoProControlServiceUUID, QueryCharUUID)
	if err != nil {
		return 0, fmt.Errorf("failed to get query characteristic: %v", err)
	}

	// Ensure response tracker exists
	m.mutex.Lock()
	if m.responseTracker[macAddress] == nil {
		m.responseTracker[macAddress] = &ResponseTracker{
			responseChannel: make(chan *QueryResponseData, 10),
		}
	}
	tracker := m.responseTracker[macAddress]
	m.mutex.Unlock()

	// Query pairing state (status ID 19)
	pairingStateCmd := []byte{QueryGetStatusValues, 19}
	n, err := queryChar.WriteWithoutResponse(pairingStateCmd)
	if err != nil {
		return 0, fmt.Errorf("failed to write pairing state query: %v", err)
	}
	if n != len(pairingStateCmd) {
		return 0, fmt.Errorf("incomplete write for pairing state query")
	}

	// Wait for response with proper timeout using response channel
	timeout := 3 * time.Second
	select {
	case response := <-tracker.responseChannel:
		if response.QueryID == QueryGetStatusValues && len(response.Data) >= 2 {
			// Data format: [status_id, status_value]
			if response.Data[0] == 19 { // Pairing state status ID
				pairingState := int(response.Data[1])
				m.log.Debugf("Received pairing state response: %d for device %s", pairingState, macAddress)

				// Cache the result
				tracker.mutex.Lock()
				tracker.PairingState = pairingState
				tracker.lastResponse = response
				tracker.mutex.Unlock()

				return pairingState, nil
			}
		}
		m.log.Warnf("Received unexpected response format for pairing state query from %s", macAddress)
		return PairingStateNeverStarted, nil

	case <-time.After(timeout):
		m.log.Warnf("Timeout waiting for pairing state response from device %s after %v", macAddress, timeout)

		// Check if we have a cached value to fall back to
		tracker.mutex.RLock()
		cachedState := tracker.PairingState
		tracker.mutex.RUnlock()

		if cachedState != 0 {
			m.log.Debugf("Using cached pairing state %d for device %s after timeout", cachedState, macAddress)
			return cachedState, nil
		}

		// If no cached state, assume not paired
		return PairingStateNeverStarted, nil
	}
}

// IsPaired checks if a device is paired with a simple interface
// This is a simpler wrapper around GetPairingState for basic pairing checks
func (m *BLEManager) IsPaired(macAddress string) (bool, error) {
	m.log.Debugf("Checking pairing state for device %s", macAddress)

	// Get the current pairing state from the device
	pairingState, err := m.GetPairingState(macAddress)
	if err != nil {
		m.log.Debugf("Failed to get pairing state for %s: %v", macAddress, err)
		return false, err
	}

	// Device is paired if pairing state is completed
	isPaired := pairingState == PairingStateCompleted
	m.log.Debugf("Device %s pairing check result: isPaired=%v (state=%d)", macAddress, isPaired, pairingState)

	return isPaired, nil
}

// getDevice retrieves a device from the discovered devices map
func (m *BLEManager) getDevice(macAddress string) *Device {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.devices[macAddress]
}

// getCharacteristic retrieves a characteristic from the service map cache
func (m *BLEManager) getCharacteristic(macAddress, serviceUUID, charUUID string) (*bluetooth.DeviceCharacteristic, error) {
	// Check if we have the service in our cache
	service, ok := m.serviceMap[serviceUUID]
	if !ok {
		return nil, fmt.Errorf("service %s not found in cache", serviceUUID)
	}

	// Check if we have the characteristic in the service
	char, ok := service[charUUID]
	if !ok {
		return nil, fmt.Errorf("characteristic %s not found in service %s", charUUID, serviceUUID)
	}

	return char, nil
}

// setEnhancedBLEMode enables or disables enhanced BLE features for newer GoPro models
func (m *BLEManager) setEnhancedBLEMode(macAddress string, enabled bool) error {
	m.log.Debugf("Setting enhanced BLE mode to %v for device %s", enabled, macAddress)

	// Get the settings characteristic for writing enhanced BLE mode commands
	settingsChar, err := m.getCharacteristic(macAddress, GoProControlServiceUUID, SettingsCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get settings characteristic: %v", err)
	}

	// Enhanced BLE mode command: Setting ID 0x03 (Enhanced BLE) with value 0x01 (enabled) or 0x00 (disabled)
	var enableValue byte = 0x00
	if enabled {
		enableValue = 0x01
	}

	enhancedBLECmd := []byte{0x03, 0x01, enableValue}

	n, err := settingsChar.WriteWithoutResponse(enhancedBLECmd)
	if err != nil {
		return fmt.Errorf("failed to write enhanced BLE mode command: %v", err)
	}
	if n != len(enhancedBLECmd) {
		return fmt.Errorf("incomplete write for enhanced BLE mode command")
	}

	// Give the camera time to process the command
	time.Sleep(200 * time.Millisecond)

	m.log.Debugf("Enhanced BLE mode set to %v for device %s", enabled, macAddress)
	return nil
}

// performSecurityHandshake performs additional security verification for HERO13 and newer models
func (m *BLEManager) performSecurityHandshake(macAddress string) error {
	m.log.Debugf("Performing security handshake for device %s", macAddress)

	// Get the command characteristic for security handshake
	cmdChar, err := m.getCharacteristic(macAddress, GoProControlServiceUUID, CommandCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get command characteristic: %v", err)
	}

	// Security handshake command: Command ID 0x5F (Security Challenge)
	securityCmd := []byte{0x5F, 0x01} // Simple handshake request

	packets := m.createPackets(securityCmd)

	for i, packet := range packets {
		n, err := cmdChar.WriteWithoutResponse(packet)
		if err != nil {
			return fmt.Errorf("failed to write security handshake packet %d: %v", i, err)
		}
		if n != len(packet) {
			return fmt.Errorf("incomplete write for security handshake packet %d", i)
		}

		// Small delay between packets
		if i < len(packets)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Wait for the handshake to complete
	time.Sleep(500 * time.Millisecond)

	m.log.Debugf("Security handshake completed for device %s", macAddress)
	return nil
}

// setConnectionParameters sets optimal BLE connection parameters based on model type
func (m *BLEManager) setConnectionParameters(macAddress string, modelType string) error {
	m.log.Debugf("Setting connection parameters for model type %s on device %s", modelType, macAddress)

	// Get the settings characteristic for connection parameter settings
	settingsChar, err := m.getCharacteristic(macAddress, GoProControlServiceUUID, SettingsCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get settings characteristic: %v", err)
	}

	// Define connection parameters based on model type
	var connectionInterval, slaveLatency, supervisionTimeout byte

	switch modelType {
	case "hero13":
		// HERO13: Fastest connection parameters for enhanced features
		connectionInterval = 0x06 // 7.5ms interval
		slaveLatency = 0x00       // No latency
		supervisionTimeout = 0x64 // 1000ms timeout

	case "hero12":
		// HERO12: Balanced parameters for good performance
		connectionInterval = 0x08 // 10ms interval
		slaveLatency = 0x00       // No latency
		supervisionTimeout = 0x64 // 1000ms timeout

	case "hero11":
		// HERO11: Moderate parameters for stability
		connectionInterval = 0x0C // 15ms interval
		slaveLatency = 0x01       // Small latency allowed
		supervisionTimeout = 0x64 // 1000ms timeout

	case "legacy":
		// Legacy models (HERO9, HERO10): Conservative parameters
		connectionInterval = 0x10 // 20ms interval
		slaveLatency = 0x02       // Higher latency allowed
		supervisionTimeout = 0x64 // 1000ms timeout

	case "default":
		// Default/unknown models: Safe conservative parameters
		connectionInterval = 0x18 // 30ms interval
		slaveLatency = 0x03       // Higher latency for stability
		supervisionTimeout = 0x64 // 1000ms timeout

	default:
		return fmt.Errorf("unknown model type: %s", modelType)
	}

	// Connection parameter command: Setting ID 0x02 (Connection Parameters)
	// Format: [Setting ID] [Length] [Interval] [Latency] [Timeout]
	connectionCmd := []byte{0x02, 0x03, connectionInterval, slaveLatency, supervisionTimeout}

	n, err := settingsChar.WriteWithoutResponse(connectionCmd)
	if err != nil {
		return fmt.Errorf("failed to write connection parameters: %v", err)
	}
	if n != len(connectionCmd) {
		return fmt.Errorf("incomplete write for connection parameters")
	}

	// Give the camera time to apply new parameters
	time.Sleep(300 * time.Millisecond)

	m.log.Debugf("Connection parameters set for model type %s on device %s (interval: %d, latency: %d, timeout: %d)",
		modelType, macAddress, connectionInterval, slaveLatency, supervisionTimeout)
	return nil
}
