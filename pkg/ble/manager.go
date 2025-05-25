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
	manager.connManager = NewConnectionManager(cfg.Adapter, cfg.Logger)
	manager.discoveryMgr = NewDiscoveryManager(cfg.Adapter, eventEmitter, cfg.Logger)
	manager.characteristicsMgr = NewCharacteristicsManager(cfg.Logger)
	manager.responseMgr = NewResponseHandler(eventEmitter, cfg.Logger)
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
		return fmt.Errorf("device %s not discovered, cannot connect", macAddress)
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
		return fmt.Errorf("connection failed: %v", err)
	}

	m.addConnection(macAddress, device)
	m.log.Infof("Phase 1 complete: Basic BLE connection established for device %s", macAddress)

	// Phase 2: Discover services AFTER connection is established
	m.log.Infof("Phase 2: Discovering services for device %s", macAddress)
	discoveredServices, err := device.DiscoverServices(nil)
	if err != nil {
		m.log.Errorf("Failed to discover services for device %s: %v", macAddress, err)
		// Clean up the connection on service discovery failure
		m.removeConnection(macAddress)
		_ = m.connManager.Disconnect(macAddress)
		return fmt.Errorf("service discovery failed: %v", err)
	}
	m.log.Infof("Phase 2 complete: Discovered %d services for device %s", len(discoveredServices), macAddress)

	// Phase 3: Cache the discovered services in characteristics manager
	m.log.Infof("Phase 3: Caching services and characteristics for device %s", macAddress)
	if err := m.characteristicsMgr.DiscoverAndCacheCharacteristics(discoveredServices); err != nil {
		m.log.Errorf("Failed to cache services for device %s: %v", macAddress, err)
		// Clean up the connection on caching failure
		m.removeConnection(macAddress)
		_ = m.connManager.Disconnect(macAddress)
		return fmt.Errorf("failed to cache services: %v", err)
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

	return m.connManager.Disconnect(macAddress)
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

// GetWifiCredentials retrieves WiFi SSID and password from the GoPro
func (m *Manager) GetWifiCredentials(macAddress string) (string, string, error) {
	bleDevice := m.getConnection(macAddress)
	if bleDevice == nil {
		return "", "", fmt.Errorf("no active connection for device %s", macAddress)
	}

	var ssid, password string
	var wifiErr error

	m.log.Tracef("Attempting to retrieve WiFi credentials for device %s", macAddress)

	wifiErr = m.withRetry("get WiFi credentials", macAddress, 3, func() error {
		// Discover WiFi service directly
		svcs, err := bleDevice.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find WiFi service
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

		// Discover characteristics
		chars, err := wifiService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find SSID and Password characteristics
		var ssidChar, passwordChar *bluetooth.DeviceCharacteristic
		for i, char := range chars {
			switch char.UUID().String() {
			case WifiSSIDCharUUID:
				ssidChar = &chars[i]
			case WifiPasswordCharUUID:
				passwordChar = &chars[i]
			}
		}

		if ssidChar == nil || passwordChar == nil {
			return fmt.Errorf("WiFi characteristics not found")
		}

		// Read SSID
		ssidData := make([]byte, 32)
		n, err := ssidChar.Read(ssidData)
		if err != nil {
			return fmt.Errorf("failed to read SSID: %v", err)
		}
		ssid = strings.TrimSpace(string(ssidData[:n]))

		// Read Password
		passwordData := make([]byte, 64)
		n, err = passwordChar.Read(passwordData)
		if err != nil {
			return fmt.Errorf("failed to read password: %v", err)
		}
		password = strings.TrimSpace(string(passwordData[:n]))

		if ssid == "" || password == "" {
			return fmt.Errorf("empty WiFi credentials received")
		}

		return nil
	})

	if wifiErr != nil {
		return "", "", fmt.Errorf("failed to get WiFi credentials: %v", wifiErr)
	}

	m.log.Infof("WiFi credentials retrieved: device=%s ssid=%s", macAddress, ssid)
	return ssid, password, nil
}

// EnableWifi enables WiFi on the GoPro
func (m *Manager) EnableWifi(macAddress string) error {
	device := m.getConnection(macAddress)
	if device == nil {
		return fmt.Errorf("no active connection for device %s", macAddress)
	}

	return m.withRetry("enable WiFi", macAddress, 3, func() error {
		// Discover WiFi service
		svcs, err := device.DiscoverServices(nil)
		if err != nil {
			return fmt.Errorf("failed to discover services: %v", err)
		}

		// Find WiFi service
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

		// Discover characteristics
		chars, err := wifiService.DiscoverCharacteristics(nil)
		if err != nil {
			return fmt.Errorf("failed to discover characteristics: %v", err)
		}

		// Find WiFi Power characteristic
		for _, char := range chars {
			if char.UUID().String() == WifiPowerCharUUID {
				// Write the value to enable WiFi (1 = ON)
				n, err := char.WriteWithoutResponse([]byte{1})
				if err != nil {
					return fmt.Errorf("failed to enable WiFi: %v", err)
				}
				if n != 1 {
					return fmt.Errorf("failed to write complete data to enable WiFi")
				}

				m.log.Infof("WiFi enabled successfully: device=%s", macAddress)
				return nil
			}
		}

		return fmt.Errorf("WiFi power characteristic not found")
	})
}

// GetMetadata retrieves metadata from the GoPro such as firmware version, model, etc.
func (m *Manager) GetMetadata(macAddress string) (map[string]string, error) {
	device := m.getConnection(macAddress)
	if device == nil {
		return nil, fmt.Errorf("no active connection for device %s", macAddress)
	}

	metadata := make(map[string]string)

	metadataErr := m.withRetry("get metadata", macAddress, 3, func() error {
		// Get control service characteristics
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		queryChar := chars["query"]
		queryRespChar := chars["queryResponse"]

		// Set up notification handler for responses
		responseChan := make(chan []byte, 5)
		if err := queryRespChar.EnableNotifications(func(data []byte) {
			select {
			case responseChan <- data:
			default:
				// Channel full, drop response
			}
		}); err != nil {
			return fmt.Errorf("failed to enable notifications: %v", err)
		}

		// Query hardware info (0x3F)
		hardwareQuery := []byte{CommandGetHardwareInfo}
		if _, err := queryChar.WriteWithoutResponse(hardwareQuery); err != nil {
			return fmt.Errorf("failed to query hardware info: %v", err)
		}

		// Wait for response
		select {
		case response := <-responseChan:
			if len(response) >= 3 && response[0] == CommandGetHardwareInfo && response[1] == 0 {
				data := response[2:]
				if len(data) >= 1 {
					modelID := data[0]
					metadata["model_id"] = fmt.Sprintf("%d", modelID)
					metadata["model_name"] = models.GetModelName(int(modelID))
				}
				if len(data) >= 5 {
					firmware := fmt.Sprintf("%d.%d.%d", data[1], data[2], data[3])
					metadata["firmware_version"] = firmware
				}
				if len(data) >= 9 {
					hardware := fmt.Sprintf("%d.%d.%d.%d", data[5], data[6], data[7], data[8])
					metadata["hardware_version"] = hardware
				}
				if len(data) >= 13 {
					serial := string(data[9:13])
					metadata["serial_number"] = serial
				}
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout waiting for hardware info response")
		}

		return nil
	})

	if metadataErr != nil {
		return nil, fmt.Errorf("failed to get metadata: %v", metadataErr)
	}

	return metadata, nil
}

// GetBatteryLevel retrieves the battery level from the GoPro
func (m *Manager) GetBatteryLevel(macAddress string) (int, error) {
	device := m.getConnection(macAddress)
	if device == nil {
		return 0, fmt.Errorf("no active connection for device %s", macAddress)
	}

	var batteryLevel int

	batteryErr := m.withRetry("get battery level", macAddress, 3, func() error {
		// Get control service characteristics
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		queryChar := chars["query"]
		queryRespChar := chars["queryResponse"]

		// Set up notification handler for responses
		responseChan := make(chan []byte, 5)
		if err := queryRespChar.EnableNotifications(func(data []byte) {
			select {
			case responseChan <- data:
			default:
				// Channel full, drop response
			}
		}); err != nil {
			return fmt.Errorf("failed to enable notifications: %v", err)
		}

		// Query battery level (status ID 70 = Internal Battery Percentage)
		// OpenGoPro TLV format: [Command] [Array Length] [Status ID]
		batteryQuery := []byte{QueryGetStatusValues, 0x01, 70}
		if _, err := queryChar.WriteWithoutResponse(batteryQuery); err != nil {
			return fmt.Errorf("failed to query battery level: %v", err)
		}

		// Wait for response
		select {
		case response := <-responseChan:
			if len(response) >= 4 && response[0] == QueryGetStatusValues && response[1] == 0 {
				statusID := response[2]
				if statusID == 70 && len(response) >= 4 {
					batteryLevel = int(response[3])
				}
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("timeout waiting for battery level response")
		}

		return nil
	})

	if batteryErr != nil {
		return 0, fmt.Errorf("failed to get battery level: %v", batteryErr)
	}

	return batteryLevel, nil
}

// SetDateTime sets the date and time on the GoPro
func (m *Manager) SetDateTime(macAddress string, t time.Time) error {
	device := m.getConnection(macAddress)
	if device == nil {
		return fmt.Errorf("no active connection for device %s", macAddress)
	}

	return m.withRetry("set date time", macAddress, 3, func() error {
		// Get control service characteristics
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		commandChar := chars["command"]

		// Create the date/time command
		// Format: [CommandSetDateTime] [Year (BE uint16)] [Month] [Day] [Hour] [Minute] [Second] [DSTOffset]
		cmd := make([]byte, 8)
		cmd[0] = CommandSetDateTime
		binary.BigEndian.PutUint16(cmd[1:3], uint16(t.Year()))
		cmd[3] = byte(t.Month())
		cmd[4] = byte(t.Day())
		cmd[5] = byte(t.Hour())
		cmd[6] = byte(t.Minute())
		cmd[7] = byte(t.Second())

		// Write the command
		if _, err := commandChar.WriteWithoutResponse(cmd); err != nil {
			return fmt.Errorf("failed to set date/time: %v", err)
		}

		m.log.Infof("Date/time synchronized: device=%s time=%v", macAddress, t)
		return nil
	})
}

// SetCameraControl sets the camera control status
func (m *Manager) SetCameraControl(macAddress string, enabled bool) error {
	device := m.getConnection(macAddress)
	if device == nil {
		return fmt.Errorf("no active connection for device %s", macAddress)
	}

	return m.withRetry("set camera control", macAddress, 3, func() error {
		// Get control service characteristics
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		commandChar := chars["command"]

		// Create the camera control command (using protobuf format)
		cmd := make([]byte, 3)
		cmd[0] = 0xF1 // Protobuf command header
		cmd[1] = ProtobufCommandSetCameraControl
		if enabled {
			cmd[2] = 1
		} else {
			cmd[2] = 0
		}

		// Create packets for the command
		packets := m.createPackets(cmd)

		// Write each packet
		for i, packet := range packets {
			if _, err := commandChar.WriteWithoutResponse(packet); err != nil {
				return fmt.Errorf("failed to write camera control packet %d: %v", i, err)
			}

			// Small delay between packets
			if i < len(packets)-1 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		m.log.Infof("Camera control updated: device=%s enabled=%v", macAddress, enabled)
		return nil
	})
}

// KeepAlive sends a keep-alive signal to maintain the connection
func (m *Manager) KeepAlive(macAddress string) error {
	device := m.getConnection(macAddress)
	if device == nil {
		return fmt.Errorf("no active connection for device %s", macAddress)
	}

	return m.withRetry("keep alive", macAddress, 3, func() error {
		// Get control service characteristics
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		settingsChar := chars["settings"]

		// The keep-alive command is a TLV command with ID 0x5B and value 0x42
		cmd := []byte{0x5B, 0x42}

		// Write the keep-alive command
		if _, err := settingsChar.WriteWithoutResponse(cmd); err != nil {
			return fmt.Errorf("failed to write keep-alive: %v", err)
		}

		m.log.Tracef("Keep-alive signal sent: device=%s", macAddress)
		return nil
	})
}

// Sleep puts the GoPro to sleep
func (m *Manager) Sleep(macAddress string) error {
	device := m.getConnection(macAddress)
	if device == nil {
		return fmt.Errorf("no active connection for device %s", macAddress)
	}

	return m.withRetry("sleep", macAddress, 3, func() error {
		// Get control service characteristics
		chars, err := m.characteristicsMgr.GetControlServiceCharacteristics()
		if err != nil {
			return fmt.Errorf("failed to get control service characteristics: %v", err)
		}

		commandChar := chars["command"]

		// Create the sleep command
		cmd := []byte{CommandSleep}

		// Create packets for the command
		packets := m.createPackets(cmd)

		// Write each packet
		for i, packet := range packets {
			if _, err := commandChar.WriteWithoutResponse(packet); err != nil {
				return fmt.Errorf("failed to write sleep packet %d: %v", i, err)
			}

			// Small delay between packets
			if i < len(packets)-1 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Give the camera extra time to process the sleep command
		time.Sleep(500 * time.Millisecond)

		m.log.Tracef("Sleep command sent successfully: device=%s", macAddress)
		return nil
	})
}
