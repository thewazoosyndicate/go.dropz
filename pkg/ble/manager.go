package ble

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble/events"
	"github.com/dropz/dropz/pkg/ble/models"
	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// Manager coordinates all BLE operations using modular components
type Manager struct {
	adapter     *bluetooth.Adapter
	devices     map[string]*Device           // MAC -> Device
	connections map[string]*bluetooth.Device // MAC -> Connection
	mutex       sync.RWMutex
	log         logger.Logger

	// Context for background tasks
	ctx        context.Context
	cancelFunc context.CancelFunc

	// Component managers
	connManager        *ConnectionManager
	discoveryMgr       *DiscoveryManager
	characteristicsMgr *CharacteristicsManager
	responseMgr        *ResponseHandler
	pairingMgr         *PairingManager

	// Event handling
	eventHandlers map[events.EventType][]events.EventHandler

	// Keep-alive goroutine cancellation map
	keepAliveCancels map[string]context.CancelFunc
}

// ManagerConfig holds configuration for creating a new Manager
type ManagerConfig struct {
	Logger  logger.Logger
	Adapter *bluetooth.Adapter
}

// NewManager creates a new coordinating BLE manager
func NewManager(cfg ManagerConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		adapter:          cfg.Adapter,
		devices:          make(map[string]*Device),
		connections:      make(map[string]*bluetooth.Device),
		log:              cfg.Logger,
		ctx:              ctx,
		cancelFunc:       cancel,
		eventHandlers:    make(map[events.EventType][]events.EventHandler),
		keepAliveCancels: make(map[string]context.CancelFunc),
	}

	// Initialize component managers
	eventEmitter := events.NewEventEmitter(cfg.Logger)
	manager.responseMgr = NewResponseHandler(eventEmitter, cfg.Logger)
	manager.connManager = NewConnectionManager(cfg.Adapter, manager.responseMgr, cfg.Logger)
	manager.discoveryMgr = NewDiscoveryManager(cfg.Adapter, eventEmitter, cfg.Logger)
	manager.characteristicsMgr = NewCharacteristicsManager(cfg.Logger)
	manager.pairingMgr = NewPairingManager(manager.connManager, manager.characteristicsMgr, manager.responseMgr, eventEmitter, cfg.Logger)

	return manager
}

// Helper method to get active connection for a device
func (m *Manager) getConnection(macAddress string) *bluetooth.Device {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.connections[macAddress]
}

// Helper method to add a connection
func (m *Manager) addConnection(macAddress string, device *bluetooth.Device) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connections[macAddress] = device
}

// Helper method to remove a connection
func (m *Manager) removeConnection(macAddress string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.connections, macAddress)
}

// withRetry provides retry logic with exponential backoff
func (m *Manager) withRetry(operation, macAddress string, maxRetries int, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if attempt < maxRetries-1 {
				waitTime := time.Duration(attempt+1) * 500 * time.Millisecond
				// Only log retries at TRACE level to reduce noise
				m.log.Tracef("BLE retry attempt %d/%d for operation '%s' on device %s (backoff: %v): %v",
					attempt+1, maxRetries, operation, macAddress, waitTime, err)
				time.Sleep(waitTime)
			} else {
				// Log final failure at ERROR level for critical operations
				m.log.Errorf("BLE operation failed permanently: operation='%s' device=%s attempts=%d error=%v",
					operation, macAddress, maxRetries, lastErr)
			}
		} else {
			// Log successful retries at INFO level for visibility
			if attempt > 0 {
				m.log.Infof("BLE operation recovered: operation='%s' device=%s attempts=%d",
					operation, macAddress, attempt+1)
			}
			return nil
		}
	}
	return fmt.Errorf("operation %s failed after %d retries: %v", operation, maxRetries, lastErr)
}

// withRetryAndReturn provides retry logic with exponential backoff for functions that return values
func (m *Manager) withRetryAndReturn(operation, macAddress string, maxRetries int, fn func() (*QueryResponseData, error)) (*QueryResponseData, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := fn()
		if err != nil {
			lastErr = err
			if attempt < maxRetries-1 {
				waitTime := time.Duration(attempt+1) * 500 * time.Millisecond
				m.log.Tracef("BLE retry attempt %d/%d for operation '%s' on device %s (backoff: %v): %v",
					attempt+1, maxRetries, operation, macAddress, waitTime, err)
				time.Sleep(waitTime)
			} else {
				m.log.Errorf("BLE operation failed permanently: operation='%s' device=%s attempts=%d error=%v",
					operation, macAddress, maxRetries, lastErr)
			}
		} else {
			if attempt > 0 {
				m.log.Infof("BLE operation recovered: operation='%s' device=%s attempts=%d",
					operation, macAddress, attempt+1)
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("operation %s failed after %d retries: %v", operation, maxRetries, lastErr)
}

// createPackets breaks large payloads into BLE-compatible packets
func (m *Manager) createPackets(payload []byte) [][]byte {
	const maxDataSize = 18 // Conservative BLE packet size
	const headerSize = 2

	if len(payload) <= maxDataSize {
		// Single packet
		packet := make([]byte, headerSize+len(payload))
		packet[0] = 0x00 // Start and end packet
		packet[1] = byte(len(payload))
		copy(packet[2:], payload)
		return [][]byte{packet}
	}

	// Multiple packets needed
	var packets [][]byte
	offset := 0
	packetIndex := 0

	for offset < len(payload) {
		remainingData := len(payload) - offset
		dataSize := maxDataSize
		if remainingData < dataSize {
			dataSize = remainingData
		}

		packet := make([]byte, headerSize+dataSize)

		// Set packet header
		if packetIndex == 0 {
			packet[0] = 0x20 // Start packet
		} else if offset+dataSize >= len(payload) {
			packet[0] = 0x80 // End packet
		} else {
			packet[0] = 0x60 // Continuation packet
		}

		packet[1] = byte(dataSize)
		copy(packet[2:], payload[offset:offset+dataSize])

		packets = append(packets, packet)
		offset += dataSize
		packetIndex++
	}

	return packets
}

// BLEInterface implementation methods

func (m *Manager) Start() error {
	return nil // Manager starts components as needed
}

func (m *Manager) Stop() {
	m.cancelFunc()
}

func (m *Manager) StartScanning(ctx context.Context) error {
	return m.discoveryMgr.StartScanning(ctx)
}

func (m *Manager) GetDiscoveredDevices() []Device {
	return m.discoveryMgr.GetDiscoveredDevices()
}

func (m *Manager) Connect(macAddress string) error {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return NewConnectionError(macAddress, "connect", err, "Check MAC address format (XX:XX:XX:XX:XX:XX)", time.Now())
	}

	// Validate manager state
	if m.adapter == nil {
		return NewConnectionError(macAddress, "connect", ErrNotReady, "BLE adapter not initialized", time.Now())
	}

	if m.connManager == nil {
		return NewConnectionError(macAddress, "connect", ErrNotReady, "Connection manager not initialized", time.Now())
	}

	if m.discoveryMgr == nil {
		return NewConnectionError(macAddress, "connect", ErrNotReady, "Discovery manager not initialized", time.Now())
	}

	if m.characteristicsMgr == nil {
		return NewConnectionError(macAddress, "connect", ErrNotReady, "Characteristics manager not initialized", time.Now())
	}

	m.log.Infof("Connecting to device %s using OpenGoPro BLE specification", macAddress)

	// Phase 0: Verify device is discovered before attempting connection
	m.log.Debugf("Verifying device %s is in discovered devices list", macAddress)
	discoveredDevices := m.discoveryMgr.GetDiscoveredDevices()
	var targetDevice *Device
	for i := range discoveredDevices {
		if discoveredDevices[i].MACAddress == macAddress {
			targetDevice = &discoveredDevices[i]
			break
		}
	}

	if targetDevice == nil {
		m.log.Errorf("Device %s not found in discovered devices, cannot connect", macAddress)
		return NewConnectionError(macAddress, "connect", ErrDeviceNotFound,
			"Device not discovered - run scan first to discover devices", time.Now())
	}

	m.log.Debugf("Device %s found in discovered devices: %s (RSSI: %d)", macAddress, targetDevice.Name, targetDevice.RSSI)

	// Phase 1: Establish basic BLE connection first
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m.log.Infof("Phase 1: Establishing BLE connection to device %s", macAddress)

	// Use connection manager to establish the connection with proper state management
	device, err := m.connManager.Connect(ctx, macAddress)
	if err != nil {
		m.log.Errorf("Failed to establish basic connection for device %s: %v", macAddress, err)
		return WrapWithContext(err, "Phase 1 - establish BLE connection", macAddress)
	}

	// Validate device pointer
	if device == nil {
		err := NewConnectionError(macAddress, "connect", ErrConnectionFailed,
			"Connection returned nil device pointer", time.Now())
		return err
	}

	m.addConnection(macAddress, device)
	m.log.Infof("Phase 1 complete: Basic BLE connection established for device %s", macAddress)

	// Phase 2: Discover services AFTER connection is established
	m.log.Infof("Phase 2: Discovering services for device %s", macAddress)

	// Implement retry logic for service discovery
	var discoveredServices []bluetooth.DeviceService
	err = m.withRetry("discover services", macAddress, 3, func() error {
		var discErr error
		discoveredServices, discErr = device.DiscoverServices(nil)
		if discErr != nil {
			return discErr
		}

		if len(discoveredServices) == 0 {
			return fmt.Errorf("no services discovered")
		}

		return nil
	})

	if err != nil {
		m.log.Errorf("Failed to discover services for device %s: %v", macAddress, err)
		// Clean up the connection on service discovery failure
		m.removeConnection(macAddress)
		_ = m.connManager.Disconnect(macAddress)
		return NewConnectionError(macAddress, "discover_services", err,
			"Service discovery failed - ensure device is compatible and in range", time.Now())
	}

	m.log.Infof("Phase 2 complete: Discovered %d services for device %s", len(discoveredServices), macAddress)

	// Phase 3: Cache the discovered services in characteristics manager
	m.log.Infof("Phase 3: Caching services and characteristics for device %s", macAddress)

	err = m.withRetry("cache characteristics", macAddress, 2, func() error {
		return m.characteristicsMgr.DiscoverAndCacheCharacteristics(discoveredServices)
	})

	if err != nil {
		m.log.Errorf("Failed to cache services for device %s: %v", macAddress, err)
		// Clean up the connection on caching failure
		m.removeConnection(macAddress)
		_ = m.connManager.Disconnect(macAddress)
		return NewConnectionError(macAddress, "cache_characteristics", err,
			"Failed to cache characteristics - ensure device supports OpenGoPro", time.Now())
	}

	m.log.Infof("Phase 3 complete: Services and characteristics cached for device %s", macAddress)

	// Phase 4: Update connection state to ready
	m.log.Infof("Phase 4: Setting device %s state to ready", macAddress)
	m.connManager.ChangeState(macAddress, StateReady, nil)

	m.log.Infof("Successfully connected to device %s with %d services discovered and cached", macAddress, len(discoveredServices))
	return nil
}

func (m *Manager) Disconnect(macAddress string) error {
	m.removeConnection(macAddress)

	// Clear the characteristics cache when disconnecting
	// Note: The current design supports one device connection at a time
	m.characteristicsMgr.Clear()
	m.log.Debugf("Cleared characteristics cache after disconnecting device %s", macAddress)

	// Use enhanced disconnect with full resource cleanup
	return m.connManager.DisconnectAndCleanup(macAddress)
}

func (m *Manager) DisconnectGoPro(macAddress string) error {
	// Put the GoPro to sleep before disconnecting for battery conservation
	if err := m.Sleep(macAddress); err != nil {
		m.log.Warnf("Sleep command failed before disconnect: device=%s error=%v", macAddress, err)
	} else {
		m.log.Tracef("Sleep command successful before disconnect: device=%s", macAddress)
	}
	return m.Disconnect(macAddress)
}

func (m *Manager) ConnectWithEnhancedPairing(macAddress string) error {
	return m.pairingMgr.ConnectWithEnhancedPairing(macAddress)
}

func (m *Manager) IsPaired(macAddress string) (bool, error) {
	return m.pairingMgr.IsPaired(macAddress)
}

func (m *Manager) IsPairedWithVerification(macAddress string) (bool, error) {
	return m.pairingMgr.IsPairedWithVerification(macAddress)
}

func (m *Manager) RefreshPairingState(macAddress string) (int, error) {
	return m.pairingMgr.RefreshPairingState(macAddress)
}

func (m *Manager) GetPairingState(macAddress string) (int, error) {
	return m.pairingMgr.GetPairingState(macAddress)
}

// GetWifiCredentials retrieves WiFi SSID and password from the GoPro with comprehensive validation
func (m *Manager) GetWifiCredentials(macAddress string) (string, string, error) {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return "", "", NewConnectionError(macAddress, "get_wifi_credentials", err, "Check MAC address format", time.Now())
	}

	bleDevice := m.getConnection(macAddress)
	if bleDevice == nil {
		return "", "", NewConnectionError(macAddress, "get_wifi_credentials", ErrDeviceNotFound,
			"No active connection - connect to device first", time.Now())
	}

	var ssid, password string
	var wifiErr error

	m.log.Tracef("Attempting to retrieve WiFi credentials for device %s", macAddress)

	wifiErr = m.withRetry("get WiFi credentials", macAddress, 3, func() error {
		// Validate device pointer before use
		if bleDevice == nil {
			return fmt.Errorf("device pointer is nil")
		}

		// Discover WiFi service directly with retry logic
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		if len(svcs) == 0 {
			return fmt.Errorf("no services discovered")
		}

		// Find WiFi service
		var wifiService bluetooth.DeviceService
		found := false
		for _, svc := range svcs {
			svcUUID := svc.UUID().String()
			if strings.EqualFold(svcUUID, GoProWifiServiceUUID) {
				wifiService = svc
				found = true
				m.log.Debugf("Found WiFi service for device %s", macAddress)
				break
			}
		}

		if !found {
			return NewConnectionError(macAddress, "get_wifi_credentials", ErrServiceNotFound,
				"WiFi service not found - ensure device supports WiFi provisioning", time.Now())
		}

		// Discover characteristics with validation
		chars, err := wifiService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover WiFi characteristics: %v", err)
		}

		if len(chars) == 0 {
			return fmt.Errorf("no WiFi characteristics discovered")
		}

		// Find SSID and Password characteristics
		var ssidChar, passwordChar *bluetooth.DeviceCharacteristic
		for i, char := range chars {
			charUUID := char.UUID().String()
			switch {
			case strings.EqualFold(charUUID, WifiSSIDCharUUID):
				ssidChar = &chars[i]
				m.log.Debugf("Found WiFi SSID characteristic for device %s", macAddress)
			case strings.EqualFold(charUUID, WifiPasswordCharUUID):
				passwordChar = &chars[i]
				m.log.Debugf("Found WiFi password characteristic for device %s", macAddress)
			}
		}

		if ssidChar == nil {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiSSIDCharUUID, "read",
				fmt.Errorf("WiFi SSID characteristic not found for device %s", macAddress))
		}

		if passwordChar == nil {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiPasswordCharUUID, "read",
				fmt.Errorf("WiFi password characteristic not found for device %s", macAddress))
		}

		// Read SSID with validation
		ssidData := make([]byte, 32)
		n, err := ssidChar.Read(ssidData)
		if err != nil {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiSSIDCharUUID, "read",
				fmt.Errorf("failed to read SSID: %v", err))
		}

		if n == 0 {
			return fmt.Errorf("empty SSID data received")
		}

		ssid = strings.TrimSpace(string(ssidData[:n]))
		m.log.Debugf("Successfully read SSID for device %s (length: %d)", macAddress, len(ssid))

		// Read Password with validation
		passwordData := make([]byte, 64)
		n, err = passwordChar.Read(passwordData)
		if err != nil {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiPasswordCharUUID, "read",
				fmt.Errorf("failed to read password: %v", err))
		}

		if n == 0 {
			return fmt.Errorf("empty password data received")
		}

		password = strings.TrimSpace(string(passwordData[:n]))
		m.log.Debugf("Successfully read WiFi password for device %s (length: %d)", macAddress, len(password))

		// Validate credential content
		if ssid == "" {
			return fmt.Errorf("empty WiFi SSID received")
		}

		if password == "" {
			return fmt.Errorf("empty WiFi password received")
		}

		// Basic SSID validation (length and printable characters)
		if len(ssid) > 32 {
			return fmt.Errorf("invalid SSID length: %d (max 32)", len(ssid))
		}

		// Basic password validation (minimum length for security)
		if len(password) < 8 {
			m.log.Warnf("WiFi password for device %s is shorter than recommended (length: %d)", macAddress, len(password))
		}

		return nil
	})

	if wifiErr != nil {
		m.log.Errorf("Failed to retrieve WiFi credentials for device %s: %v", macAddress, wifiErr)
		return "", "", wifiErr
	}

	m.log.Infof("Successfully retrieved WiFi credentials for device %s (SSID: %s)", macAddress, ssid)
	return ssid, password, nil
}

// EnableWifi enables WiFi on the GoPro with comprehensive error handling and validation
func (m *Manager) EnableWifi(macAddress string) error {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return NewConnectionError(macAddress, "enable_wifi", err, "Check MAC address format", time.Now())
	}

	device := m.getConnection(macAddress)
	if device == nil {
		return NewConnectionError(macAddress, "enable_wifi", ErrDeviceNotFound,
			"No active connection - connect to device first", time.Now())
	}

	// Validate device pointer
	if err := ValidateDevicePointer(device); err != nil {
		return NewConnectionError(macAddress, "enable_wifi", err,
			"Device pointer validation failed", time.Now())
	}

	m.log.Infof("Enabling WiFi for device %s", macAddress)

	return m.withRetry("enable WiFi", macAddress, 3, func() error {
		// Discover WiFi service with validation
		svcs, err := device.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		if len(svcs) == 0 {
			return fmt.Errorf("no services discovered on device")
		}

		// Find WiFi service with proper validation
		var wifiService bluetooth.DeviceService
		found := false
		for _, svc := range svcs {
			svcUUID := svc.UUID().String()
			if strings.EqualFold(svcUUID, GoProWifiServiceUUID) {
				wifiService = svc
				found = true
				m.log.Debugf("Found WiFi service for device %s", macAddress)
				break
			}
		}

		if !found {
			return NewConnectionError(macAddress, "enable_wifi", ErrServiceNotFound,
				"WiFi service not found - ensure device supports WiFi", time.Now())
		}

		// Discover characteristics with validation
		chars, err := wifiService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover WiFi characteristics: %v", err)
		}

		if len(chars) == 0 {
			return fmt.Errorf("no WiFi characteristics discovered")
		}

		// Find WiFi Power characteristic with validation
		var wifiPowerChar *bluetooth.DeviceCharacteristic
		for i, char := range chars {
			charUUID := char.UUID().String()
			if strings.EqualFold(charUUID, WifiPowerCharUUID) {
				wifiPowerChar = &chars[i]
				m.log.Debugf("Found WiFi power characteristic for device %s", macAddress)
				break
			}
		}

		if wifiPowerChar == nil {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiPowerCharUUID, "write",
				fmt.Errorf("WiFi power characteristic not found for device %s", macAddress))
		}

		// Validate characteristic pointer
		if err := ValidateCharacteristicPointer(wifiPowerChar); err != nil {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiPowerCharUUID, "write", err)
		}

		// Write the value to enable WiFi (1 = ON) with validation
		wifiEnableData := []byte{1}
		n, err := wifiPowerChar.WriteWithoutResponse(wifiEnableData)
		if err != nil {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiPowerCharUUID, "write",
				fmt.Errorf("failed to enable WiFi: %v", err))
		}

		if n != len(wifiEnableData) {
			return NewCharacteristicError(GoProWifiServiceUUID, WifiPowerCharUUID, "write",
				fmt.Errorf("incomplete write: expected %d bytes, wrote %d", len(wifiEnableData), n))
		}

		m.log.Infof("WiFi enabled successfully for device %s", macAddress)
		return nil
	})
}

// GetMetadata retrieves metadata from the GoPro with comprehensive error handling and validation
func (m *Manager) GetMetadata(macAddress string) (map[string]string, error) {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return nil, NewConnectionError(macAddress, "get_metadata", err, "Check MAC address format", time.Now())
	}

	device := m.getConnection(macAddress)
	if device == nil {
		return nil, NewConnectionError(macAddress, "get_metadata", ErrDeviceNotFound,
			"No active connection - connect to device first", time.Now())
	}

	// Validate device pointer
	if err := ValidateDevicePointer(device); err != nil {
		return nil, NewConnectionError(macAddress, "get_metadata", err,
			"Device pointer validation failed", time.Now())
	}

	metadata := make(map[string]string)

	m.log.Debugf("Retrieving metadata for device %s", macAddress)

	metadataErr := m.withRetry("get metadata", macAddress, 3, func() error {
		// Get control service characteristics with validation
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		queryChar := chars["query"]
		queryRespChar := chars["queryResponse"]

		// Validate characteristic pointers
		if err := ValidateCharacteristicPointer(queryChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryCharUUID, "validate", err)
		}

		if err := ValidateCharacteristicPointer(queryRespChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryResponseCharUUID, "validate", err)
		}

		// Set up notification handler for responses with timeout
		responseChan := make(chan []byte, 5)
		notificationErr := queryRespChar.EnableNotifications(func(data []byte) {
			select {
			case responseChan <- data:
				// Successfully queued response
			default:
				// Channel full, log warning but don't block
				m.log.Warnf("Response channel full when receiving metadata response for device %s", macAddress)
			}
		})

		if notificationErr != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryResponseCharUUID, "enable_notifications",
				fmt.Errorf("failed to enable notifications: %v", notificationErr))
		}

		// Query hardware info (0x3F) with validation
		hardwareQuery := []byte{CommandGetHardwareInfo}
		n, err := queryChar.WriteWithoutResponse(hardwareQuery)
		if err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryCharUUID, "write",
				fmt.Errorf("failed to query hardware info: %v", err))
		}

		if n != len(hardwareQuery) {
			return NewCharacteristicError(GoProControlServiceUUID, QueryCharUUID, "write",
				fmt.Errorf("incomplete write: expected %d bytes, wrote %d", len(hardwareQuery), n))
		}

		m.log.Tracef("Hardware info query sent to device %s", macAddress)

		// Wait for response with proper timeout and validation
		select {
		case response := <-responseChan:
			// Validate response format
			if err := ValidateResponseFormat(response, CommandGetHardwareInfo); err != nil {
				return NewResponseError(CommandGetHardwareInfo, len(response), err)
			}

			if len(response) >= 3 && response[0] == CommandGetHardwareInfo && response[1] == 0 {
				data := response[2:]

				// Parse model ID with validation
				if len(data) >= 1 {
					modelID := int(data[0])
					if err := ValidateModelID(modelID); err != nil {
						m.log.Warnf("Invalid model ID %d for device %s: %v", modelID, macAddress, err)
						metadata["model_id"] = fmt.Sprintf("%d", modelID)
						metadata["model_name"] = "Unknown"
					} else {
						metadata["model_id"] = fmt.Sprintf("%d", modelID)
						metadata["model_name"] = models.GetModelName(modelID)
					}
				}

				// Parse firmware version with validation
				if len(data) >= 5 {
					firmware := fmt.Sprintf("%d.%d.%d", data[1], data[2], data[3])
					metadata["firmware_version"] = firmware
				}

				// Parse hardware version with validation
				if len(data) >= 9 {
					hardware := fmt.Sprintf("%d.%d.%d.%d", data[5], data[6], data[7], data[8])
					metadata["hardware_version"] = hardware
				}

				// Parse serial number with validation
				if len(data) >= 13 {
					serial := strings.TrimSpace(string(data[9:13]))
					if serial != "" {
						metadata["serial_number"] = serial
					}
				}

				m.log.Debugf("Successfully parsed metadata for device %s: model=%s firmware=%s",
					macAddress, metadata["model_name"], metadata["firmware_version"])
			} else {
				return NewResponseError(CommandGetHardwareInfo, len(response),
					fmt.Errorf("invalid hardware info response format"))
			}

		case <-time.After(10 * time.Second):
			return NewResponseError(CommandGetHardwareInfo, 0,
				fmt.Errorf("timeout waiting for hardware info response"))
		}

		return nil
	})

	if metadataErr != nil {
		m.log.Errorf("Failed to retrieve metadata for device %s: %v", macAddress, metadataErr)
		return nil, metadataErr
	}

	m.log.Infof("Successfully retrieved metadata for device %s", macAddress)
	return metadata, nil
}

// GetBatteryLevel retrieves the battery level from the GoPro with comprehensive error handling
func (m *Manager) GetBatteryLevel(macAddress string) (int, error) {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return 0, NewConnectionError(macAddress, "get_battery_level", err, "Check MAC address format", time.Now())
	}

	device := m.getConnection(macAddress)
	if device == nil {
		return 0, NewConnectionError(macAddress, "get_battery_level", ErrDeviceNotFound,
			"No active connection - connect to device first", time.Now())
	}

	// Validate device pointer
	if err := ValidateDevicePointer(device); err != nil {
		return 0, NewConnectionError(macAddress, "get_battery_level", err,
			"Device pointer validation failed", time.Now())
	}

	var batteryLevel int

	m.log.Debugf("Retrieving battery level for device %s", macAddress)

	batteryErr := m.withRetry("get battery level", macAddress, 3, func() error {
		// Get control service characteristics with validation
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		queryChar := chars["query"]
		queryRespChar := chars["queryResponse"]

		// Validate characteristic pointers
		if err := ValidateCharacteristicPointer(queryChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryCharUUID, "validate", err)
		}

		if err := ValidateCharacteristicPointer(queryRespChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryResponseCharUUID, "validate", err)
		}

		// Set up notification handler for responses with proper channel management
		responseChan := make(chan []byte, 5)
		notificationErr := queryRespChar.EnableNotifications(func(data []byte) {
			select {
			case responseChan <- data:
				// Successfully queued response
			default:
				// Channel full, log warning but don't block
				m.log.Warnf("Response channel full when receiving battery response for device %s", macAddress)
			}
		})

		if notificationErr != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryResponseCharUUID, "enable_notifications",
				fmt.Errorf("failed to enable notifications: %v", notificationErr))
		}

		// Query battery level (status ID 70 = Internal Battery Percentage)
		// OpenGoPro TLV format: [Command] [Array Length] [Status ID]
		batteryQuery := []byte{QueryGetStatusValues, 0x01, 70}
		n, err := queryChar.WriteWithoutResponse(batteryQuery)
		if err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, QueryCharUUID, "write",
				fmt.Errorf("failed to query battery level: %v", err))
		}

		if n != len(batteryQuery) {
			return NewCharacteristicError(GoProControlServiceUUID, QueryCharUUID, "write",
				fmt.Errorf("incomplete write: expected %d bytes, wrote %d", len(batteryQuery), n))
		}

		m.log.Tracef("Battery level query sent to device %s", macAddress)

		// Wait for response with proper timeout and validation
		select {
		case response := <-responseChan:
			// Validate response format
			if err := ValidateResponseFormat(response, QueryGetStatusValues); err != nil {
				return NewResponseError(QueryGetStatusValues, len(response), err)
			}

			if len(response) >= 4 && response[0] == QueryGetStatusValues && response[1] == 0 {
				statusID := response[2]
				if statusID == 70 && len(response) >= 4 {
					batteryLevel = int(response[3])

					// Validate battery level range (0-100%)
					if batteryLevel < 0 || batteryLevel > 100 {
						return NewResponseError(QueryGetStatusValues, len(response),
							fmt.Errorf("invalid battery level %d%% (expected 0-100%%)", batteryLevel))
					}

					m.log.Debugf("Successfully read battery level for device %s: %d%%", macAddress, batteryLevel)
				} else {
					return NewResponseError(QueryGetStatusValues, len(response),
						fmt.Errorf("invalid battery response: statusID=%d, length=%d", statusID, len(response)))
				}
			} else {
				return NewResponseError(QueryGetStatusValues, len(response),
					fmt.Errorf("invalid battery level response format"))
			}

		case <-time.After(8 * time.Second):
			return NewResponseError(QueryGetStatusValues, 0,
				fmt.Errorf("timeout waiting for battery level response"))
		}

		return nil
	})

	if batteryErr != nil {
		m.log.Errorf("Failed to retrieve battery level for device %s: %v", macAddress, batteryErr)
		return 0, batteryErr
	}

	m.log.Infof("Successfully retrieved battery level for device %s: %d%%", macAddress, batteryLevel)
	return batteryLevel, nil
}

// SetDateTime sets the date and time on the GoPro with comprehensive error handling and validation
func (m *Manager) SetDateTime(macAddress string, t time.Time) error {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return NewConnectionError(macAddress, "set_datetime", err, "Check MAC address format", time.Now())
	}

	// Validate time parameter
	if t.IsZero() {
		return NewValidationError("datetime", t, "datetime cannot be zero value")
	}

	// Check if time is reasonable (not too far in past or future)
	now := time.Now()
	if t.Before(now.AddDate(-10, 0, 0)) || t.After(now.AddDate(10, 0, 0)) {
		return NewValidationError("datetime", t, "datetime must be within reasonable range (±10 years)")
	}

	device := m.getConnection(macAddress)
	if device == nil {
		return NewConnectionError(macAddress, "set_datetime", ErrDeviceNotFound,
			"No active connection - connect to device first", time.Now())
	}

	// Validate device pointer
	if err := ValidateDevicePointer(device); err != nil {
		return NewConnectionError(macAddress, "set_datetime", err,
			"Device pointer validation failed", time.Now())
	}

	m.log.Infof("Setting date/time for device %s to %v", macAddress, t)

	return m.withRetry("set date time", macAddress, 3, func() error {
		// Get control service characteristics with validation
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		commandChar := chars["command"]

		// Validate characteristic pointer
		if err := ValidateCharacteristicPointer(commandChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "validate", err)
		}

		// Create the date/time command with validation
		// Format: [CommandSetDateTime] [Year (BE uint16)] [Month] [Day] [Hour] [Minute] [Second] [DSTOffset]
		cmd := make([]byte, 8)
		cmd[0] = CommandSetDateTime

		// Validate year range
		year := t.Year()
		if year < 1900 || year > 2100 {
			return NewValidationError("year", year, "year must be between 1900 and 2100")
		}

		binary.BigEndian.PutUint16(cmd[1:3], uint16(year))
		cmd[3] = byte(t.Month())
		cmd[4] = byte(t.Day())
		cmd[5] = byte(t.Hour())
		cmd[6] = byte(t.Minute())
		cmd[7] = byte(t.Second())

		m.log.Tracef("DateTime command payload for device %s: %02X", macAddress, cmd)

		// Write the command with validation
		n, err := commandChar.WriteWithoutResponse(cmd)
		if err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "write",
				fmt.Errorf("failed to set date/time: %v", err))
		}

		if n != len(cmd) {
			return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "write",
				fmt.Errorf("incomplete write: expected %d bytes, wrote %d", len(cmd), n))
		}

		m.log.Infof("Date/time synchronized successfully for device %s: %v", macAddress, t)
		return nil
	})
}

// SetCameraControl sets the camera control status with comprehensive error handling and validation
func (m *Manager) SetCameraControl(macAddress string, enabled bool) error {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return NewConnectionError(macAddress, "set_camera_control", err, "Check MAC address format", time.Now())
	}

	device := m.getConnection(macAddress)
	if device == nil {
		return NewConnectionError(macAddress, "set_camera_control", ErrDeviceNotFound,
			"No active connection - connect to device first", time.Now())
	}

	// Validate device pointer
	if err := ValidateDevicePointer(device); err != nil {
		return NewConnectionError(macAddress, "set_camera_control", err,
			"Device pointer validation failed", time.Now())
	}

	m.log.Infof("Setting camera control for device %s to enabled=%v", macAddress, enabled)

	return m.withRetry("set camera control", macAddress, 3, func() error {
		// Get control service characteristics with validation
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		commandChar := chars["command"]

		// Validate characteristic pointer
		if err := ValidateCharacteristicPointer(commandChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "validate", err)
		}

		// Create the camera control command (using protobuf format)
		cmd := make([]byte, 3)
		cmd[0] = 0xF1 // Protobuf command header
		cmd[1] = ProtobufCommandSetCameraControl
		if enabled {
			cmd[2] = 1
		} else {
			cmd[2] = 0
		}

		m.log.Tracef("Camera control command payload for device %s: %02X", macAddress, cmd)

		// Create packets for the command with validation
		packets := m.createPackets(cmd)
		if len(packets) == 0 {
			return fmt.Errorf("failed to create command packets")
		}

		// Write each packet with validation and retry logic
		for i, packet := range packets {
			if len(packet) == 0 {
				return fmt.Errorf("empty packet %d generated", i)
			}

			n, err := commandChar.WriteWithoutResponse(packet)
			if err != nil {
				return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "write",
					fmt.Errorf("failed to write camera control packet %d: %v", i, err))
			}

			if n != len(packet) {
				return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "write",
					fmt.Errorf("incomplete write for packet %d: expected %d bytes, wrote %d", i, len(packet), n))
			}

			m.log.Tracef("Camera control packet %d/%d written to device %s (%d bytes)",
				i+1, len(packets), macAddress, n)

			// Small delay between packets to avoid overwhelming the device
			if i < len(packets)-1 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		m.log.Infof("Camera control updated successfully for device %s: enabled=%v", macAddress, enabled)
		return nil
	})
}

// KeepAlive sends a keep-alive signal to maintain the connection with comprehensive error handling
func (m *Manager) KeepAlive(macAddress string) error {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return NewValidationError("keep_alive", err, "Provide a valid MAC address in format XX:XX:XX:XX:XX:XX")
	}

	m.log.Debug("Sending keep-alive signal", "mac_address", macAddress)

	// Validate device pointer
	device := m.getConnection(macAddress)
	if err := ValidateDevicePointer(device); err != nil {
		return NewConnectionError(macAddress, "keep_alive", err, "Ensure device is connected before sending keep-alive", time.Now())
	}

	return m.withRetry("keep alive", macAddress, 3, func() error {
		// Get control service characteristics with validation
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, "settings", "keep_alive",
				fmt.Errorf("failed to get control service characteristics: %v", err))
		}

		// Validate settings characteristic exists
		settingsChar, exists := chars["settings"]
		if !exists {
			return NewCharacteristicError(GoProControlServiceUUID, SettingsCharUUID, "keep_alive",
				fmt.Errorf("settings characteristic not found for device %s", macAddress))
		}

		// Validate characteristic pointer
		if err := ValidateCharacteristicPointer(settingsChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, SettingsCharUUID, "keep_alive", err)
		}

		// The keep-alive command is a TLV command with ID 0x5B and value 0x42 (according to OpenGoPro spec)
		cmd := []byte{0x5B, 0x42}

		m.log.Trace("Writing keep-alive command", "mac_address", macAddress, "command", cmd)

		// Write the keep-alive command with validation
		bytesWritten, err := settingsChar.WriteWithoutResponse(cmd)
		if err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, SettingsCharUUID, "keep_alive",
				fmt.Errorf("failed to write keep-alive command: %v", err))
		}

		// Validate bytes written
		if bytesWritten != len(cmd) {
			return NewCharacteristicError(GoProControlServiceUUID, SettingsCharUUID, "keep_alive",
				fmt.Errorf("incomplete write: expected %d bytes, wrote %d bytes", len(cmd), bytesWritten))
		}

		m.log.Info("Keep-alive signal sent successfully", "mac_address", macAddress, "bytes_written", bytesWritten)
		return nil
	})
}

// Sleep puts the GoPro to sleep with comprehensive error handling and validation
func (m *Manager) Sleep(macAddress string) error {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return NewValidationError("sleep", err, "Provide a valid MAC address in format XX:XX:XX:XX:XX:XX")
	}

	m.log.Debug("Sending sleep command", "mac_address", macAddress)

	// Validate device pointer
	device := m.getConnection(macAddress)
	if err := ValidateDevicePointer(device); err != nil {
		return NewConnectionError(macAddress, "sleep", err, "Ensure device is connected before sending sleep command", time.Now())
	}

	return m.withRetry("sleep", macAddress, 3, func() error {
		// Get control service characteristics with validation
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, "command", "sleep",
				fmt.Errorf("failed to get control service characteristics: %v", err))
		}

		// Validate command characteristic exists
		commandChar, exists := chars["command"]
		if !exists {
			return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "sleep",
				fmt.Errorf("command characteristic not found for device %s", macAddress))
		}

		// Validate characteristic pointer
		if err := ValidateCharacteristicPointer(commandChar); err != nil {
			return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "sleep", err)
		}

		// Create the sleep command (according to OpenGoPro spec)
		cmd := []byte{CommandSleep}

		m.log.Trace("Creating sleep command packets", "mac_address", macAddress, "command", cmd)

		// Create packets for the command
		packets := m.createPackets(cmd)
		if len(packets) == 0 {
			return NewValidationError("sleep", fmt.Errorf("no packets created for sleep command"),
				"Check command data format")
		}

		m.log.Trace("Sending sleep command packets", "mac_address", macAddress, "packet_count", len(packets))

		// Write each packet with validation
		for i, packet := range packets {
			if len(packet) == 0 {
				return NewValidationError("sleep", fmt.Errorf("empty packet %d", i),
					"Check packet creation logic")
			}

			bytesWritten, err := commandChar.WriteWithoutResponse(packet)
			if err != nil {
				return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "sleep",
					fmt.Errorf("failed to write sleep packet %d: %v", i, err))
			}

			// Validate bytes written
			if bytesWritten != len(packet) {
				return NewCharacteristicError(GoProControlServiceUUID, CommandCharUUID, "sleep",
					fmt.Errorf("incomplete write for packet %d: expected %d bytes, wrote %d bytes",
						i, len(packet), bytesWritten))
			}

			m.log.Trace("Sleep packet written successfully", "mac_address", macAddress,
				"packet_index", i, "bytes_written", bytesWritten)

			// Small delay between packets to ensure proper processing
			if i < len(packets)-1 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Give the camera extra time to process the sleep command
		time.Sleep(500 * time.Millisecond)

		m.log.Info("Sleep command sent successfully", "mac_address", macAddress,
			"packets_sent", len(packets))
		return nil
	})
}

// SendCommandWithResponse sends a command and waits for the correlated response using the new correlation system with comprehensive error handling
func (m *Manager) SendCommandWithResponse(macAddress string, commandID byte, data []byte, timeout time.Duration, isQuery bool) (*QueryResponseData, error) {
	// Validate input parameters
	if err := ValidateMAC(macAddress); err != nil {
		return nil, NewValidationError("send_command_with_response", err, "Provide a valid MAC address in format XX:XX:XX:XX:XX:XX")
	}

	// Validate timeout
	if timeout < 0 {
		return nil, NewValidationError("send_command_with_response",
			fmt.Errorf("invalid timeout: %v", timeout), "Provide a non-negative timeout duration")
	}

	// Validate device pointer
	device := m.getConnection(macAddress)
	if err := ValidateDevicePointer(device); err != nil {
		return nil, NewConnectionError(macAddress, "send_command_with_response", err,
			"Ensure device is connected before sending command", time.Now())
	}

	// Ensure response tracker is initialized
	m.responseMgr.InitializeTracker(macAddress)

	// Set default timeout if not provided
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	// Validate timeout range (reasonable limits)
	if timeout > 60*time.Second {
		return nil, NewValidationError("send_command_with_response",
			fmt.Errorf("timeout too large: %v", timeout), "Use a timeout less than 60 seconds")
	}

	// Build command data with validation
	var commandData []byte
	if isQuery {
		// For queries, build the query command
		if data == nil {
			commandData = []byte{commandID}
		} else {
			// Validate data length for queries
			if len(data) > 255 {
				return nil, NewValidationError("send_command_with_response",
					fmt.Errorf("query data too large: %d bytes", len(data)),
					"Query data should be less than 256 bytes")
			}
			commandData = append([]byte{commandID}, data...)
		}
	} else {
		// For commands, build the command
		if data == nil {
			commandData = []byte{commandID}
		} else {
			// Validate data length for commands
			if len(data) > 1024 {
				return nil, NewValidationError("send_command_with_response",
					fmt.Errorf("command data too large: %d bytes", len(data)),
					"Command data should be less than 1024 bytes")
			}
			commandData = append([]byte{commandID}, data...)
		}
	}

	m.log.Debug("Sending command with response correlation", "mac_address", macAddress,
		"command_id", commandID, "is_query", isQuery, "timeout", timeout,
		"data_length", len(commandData))

	return m.withRetryAndReturn("send command with response", macAddress, 3, func() (*QueryResponseData, error) {
		// Get appropriate characteristic with validation
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return nil, NewCharacteristicError(GoProControlServiceUUID, "control", "send_command_with_response",
				fmt.Errorf("failed to get control service characteristics: %v", err))
		}

		// Select appropriate characteristic based on command type
		var targetChar *bluetooth.DeviceCharacteristic
		var charName, charUUID string
		if isQuery {
			targetChar = chars["query"]
			charName = "query"
			charUUID = QueryCharUUID
		} else {
			targetChar = chars["command"]
			charName = "command"
			charUUID = CommandCharUUID
		}

		// Validate characteristic exists
		if targetChar == nil {
			return nil, NewCharacteristicError(GoProControlServiceUUID, charUUID, "send_command_with_response",
				fmt.Errorf("%s characteristic not found for device %s", charName, macAddress))
		}

		// Validate characteristic pointer
		if err := ValidateCharacteristicPointer(targetChar); err != nil {
			return nil, NewCharacteristicError(GoProControlServiceUUID, charUUID, "send_command_with_response", err)
		}

		// Start waiting for response before sending command to avoid race condition
		responseChan := make(chan *QueryResponseData, 1)
		errorChan := make(chan error, 1)

		// Start response handler goroutine
		go func() {
			defer func() {
				if r := recover(); r != nil {
					errorChan <- fmt.Errorf("response handler panic: %v", r)
				}
			}()

			response, err := m.responseMgr.SendCommandWithResponse(macAddress, commandID, data, timeout, isQuery)
			if err != nil {
				errorChan <- err
			} else {
				responseChan <- response
			}
		}()

		// Give a small delay to ensure the response handler is ready
		time.Sleep(10 * time.Millisecond)

		// Send the command with validation
		m.log.Trace("Writing command data", "mac_address", macAddress, "command_id", commandID,
			"data_length", len(commandData))

		bytesWritten, err := targetChar.WriteWithoutResponse(commandData)
		if err != nil {
			return nil, NewCharacteristicError(GoProControlServiceUUID, charUUID, "send_command_with_response",
				fmt.Errorf("failed to write command: %v", err))
		}

		// Validate bytes written
		if bytesWritten != len(commandData) {
			return nil, NewCharacteristicError(GoProControlServiceUUID, charUUID, "send_command_with_response",
				fmt.Errorf("incomplete write: expected %d bytes, wrote %d bytes", len(commandData), bytesWritten))
		}

		m.log.Debug("Command sent, waiting for response", "mac_address", macAddress,
			"command_id", commandID, "bytes_written", bytesWritten)

		// Wait for response with proper timeout handling
		select {
		case response := <-responseChan:
			if response == nil {
				return nil, NewResponseError(commandID, 0,
					fmt.Errorf("received nil response"))
			}
			m.log.Debug("Command response received successfully", "mac_address", macAddress,
				"command_id", commandID, "status", response.Status)
			return response, nil

		case err := <-errorChan:
			return nil, NewResponseError(commandID, 0, err)

		case <-time.After(timeout + 1*time.Second): // Add buffer to timeout
			return nil, NewResponseError(commandID, 0,
				fmt.Errorf("timeout waiting for command response after %v", timeout))
		}
	})
}

// SendQuery sends a query and waits for response using the new correlation system with comprehensive error handling
func (m *Manager) SendQuery(macAddress string, queryID byte, data []byte, timeout time.Duration) (*QueryResponseData, error) {
	// Enhanced logging for query operations
	m.log.Debug("Sending query with correlation", "mac_address", macAddress, "query_id", queryID,
		"data_length", len(data), "timeout", timeout)

	// Use the enhanced SendCommandWithResponse method
	response, err := m.SendCommandWithResponse(macAddress, queryID, data, timeout, true)
	if err != nil {
		m.log.Error("Query failed", "mac_address", macAddress, "query_id", queryID, "error", err)
		return nil, err
	}

	m.log.Debug("Query completed successfully", "mac_address", macAddress, "query_id", queryID,
		"status", response.Status)
	return response, nil
}

// SendCommand sends a command and waits for response using the new correlation system with comprehensive error handling
func (m *Manager) SendCommand(macAddress string, commandID byte, data []byte, timeout time.Duration) (*QueryResponseData, error) {
	// Enhanced logging for command operations
	m.log.Debug("Sending command with correlation", "mac_address", macAddress, "command_id", commandID,
		"data_length", len(data), "timeout", timeout)

	// Use the enhanced SendCommandWithResponse method
	response, err := m.SendCommandWithResponse(macAddress, commandID, data, timeout, false)
	if err != nil {
		m.log.Error("Command failed", "mac_address", macAddress, "command_id", commandID, "error", err)
		return nil, err
	}

	m.log.Debug("Command completed successfully", "mac_address", macAddress, "command_id", commandID,
		"status", response.Status)
	return response, nil
}
