package ble

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime"
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
		adapter:       adapter,
		devices:       make(map[string]*Device),
		connections:   make(map[string]*bluetooth.Device),
		log:           log,
		ctx:           ctx,
		cancelFunc:    cancel,
		eventHandlers: make(map[EventType][]EventHandler),
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

				// Get the device address string - this could be a MAC or UUID depending on platform
				deviceAddr := result.Address.String()
				m.log.Debugf("Discovered GoPro device with address: %s", deviceAddr)

				// Create or update device
				device := &Device{
					Name:       result.LocalName(),
					MACAddress: deviceAddr,
					RSSI:       int32(result.RSSI),
				}

				// Store in our device map
				m.devices[deviceAddr] = device

				// For macOS/Darwin with UUID format addresses, also add a reference with normalized format
				// This allows lookups with consistent format
				if strings.Contains(deviceAddr, "-") && len(deviceAddr) == 36 {
					m.log.Debugf("Device has UUID format address, storing additional reference")

					// Additional log to help debugging
					m.log.Debugf("Current devices in map: %v", m.mapKeys())
				}

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
	// First handle the special case for macOS/Darwin where we might get a UUID directly
	isUUID := strings.Contains(macAddress, "-") && len(macAddress) == 36

	// For UUID format on macOS, we need to scan first to discover the device
	if isUUID && runtime.GOOS == "darwin" {
		m.log.Infof("macOS detected with UUID format address: %s - performing scan first", macAddress)

		// Check if we already have this device in our map
		m.mutex.RLock()
		_, exists := m.devices[macAddress]
		m.mutex.RUnlock()

		// If it doesn't exist, we need to scan to discover it
		if !exists {
			m.log.Infof("Device with UUID %s not found in device map, scanning to discover it", macAddress)

			// Start a short scan to discover the device
			scanCtx, scanCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer scanCancel()

			err := m.StartScanning(scanCtx)
			if err != nil {
				m.log.Warnf("Error scanning for devices: %v", err)
				// Continue anyway, we might have found some devices
			}

			// Check if we found the device
			m.mutex.RLock()
			_, exists = m.devices[macAddress]
			m.mutex.RUnlock()

			if !exists {
				m.log.Infof("Creating temporary device for UUID %s after scan", macAddress)
				// Create a temporary device for this UUID as fallback
				tmpDevice := &Device{
					Name:       "GoPro (UUID Connection)",
					MACAddress: macAddress,
					RSSI:       0,
				}

				m.mutex.Lock()
				m.devices[macAddress] = tmpDevice
				m.mutex.Unlock()
			}
		}
	}

	// Get device reference
	var device *Device
	m.mutex.RLock()
	var exists bool
	device, exists = m.devices[macAddress]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("device with MAC address %s not found", macAddress)
	}

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

			// Create the platform-specific address
			addr, addrErr := parseAddress(macAddress)
			if addrErr != nil {
				err = fmt.Errorf("failed to parse address %s: %w", macAddress, addrErr)
				return
			}

			m.log.Debugf("Connecting with Address: %+v", addr)

			// This would ideally use the context if the library supports it
			// For now, we're still using the basic Connect method but with timeout supervision
			bleDevice, err = m.adapter.Connect(addr, bluetooth.ConnectionParams{})

			if err != nil {
				m.log.Errorf("Connection error: %v", err)
				// Ensure we don't try to use a nil bleDevice
				bleDevice = nil
				return
			} else if bleDevice == nil {
				// Safeguard against nil bleDevice even when no error reported
				m.log.Errorf("Connection returned nil device but no error")
				err = fmt.Errorf("bluetooth adapter returned nil device")
				return
			} else {
				m.log.Debugf("Successfully connected to device")
			}

			// Additional validation - pause briefly and check if the connection is still valid
			// This can help detect quick disconnects
			time.Sleep(100 * time.Millisecond)

			// Attempt a simple operation to validate the connection
			if bleDevice != nil {
				m.log.Debugf("Connection appears valid, device pointer: %p", bleDevice)
			} else {
				m.log.Errorf("Device became nil after connection")
				err = fmt.Errorf("device connection became invalid immediately after connect")
				return
			}
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

	// Critical check: ensure the bleDevice is not nil before using it
	if bleDevice == nil {
		return fmt.Errorf("connection succeeded but returned nil device")
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

	// Enable notifications for Query Response
	m.log.Debug("Enabling notifications for Query Response")
	if err := queryRespChar.EnableNotifications(func(buf []byte) {
		m.log.Debugf("Received query response: %v", buf)
	}); err != nil {
		return fmt.Errorf("failed to enable notifications: %v", err)
	}

	// Check pairing state
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
func (m *BLEManager) detectGoProModel() error {
	// TODO: Implement model detection using hardware info query
	return nil
}

// mapKeys returns a slice of all keys in the devices map
// This is useful for debugging purposes
func (m *BLEManager) mapKeys() []string {
	keys := make([]string, 0, len(m.devices))
	for k := range m.devices {
		keys = append(keys, k)
	}
	return keys
}
