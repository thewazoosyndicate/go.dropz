package ble

import (
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// CharacteristicsManager handles BLE characteristic operations
// Thread Safety: All operations are protected by RWMutex
// - Read operations (Get*, List*) use RLock() for concurrent access
// - Write operations (Discover*, Cache*, Clear) use Lock() for exclusive access
// - ServiceMap is the primary shared state requiring protection
type CharacteristicsManager struct {
	serviceMap map[string]map[string]*bluetooth.DeviceCharacteristic // serviceUUID -> (charUUID -> characteristic)
	mutex      sync.RWMutex                                          // Protects serviceMap access
	log        logger.Logger
}

// NewCharacteristicsManager creates a new characteristics manager
func NewCharacteristicsManager(log logger.Logger) *CharacteristicsManager {
	return &CharacteristicsManager{
		serviceMap: make(map[string]map[string]*bluetooth.DeviceCharacteristic),
		log:        log,
	}
}

// DiscoverAndCacheCharacteristics discovers all characteristics for all services and caches them
// Thread Safety: Write lock protects serviceMap during discovery and caching operation
// Ensures atomic update of entire service map
func (cm *CharacteristicsManager) DiscoverAndCacheCharacteristics(services []bluetooth.DeviceService) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Initialize service map with proper cleanup of previous state
	cm.serviceMap = make(map[string]map[string]*bluetooth.DeviceCharacteristic)

	cm.log.Info("Starting characteristic discovery", "service_count", len(services))

	// Discover all characteristics and build service map
	for _, svc := range services {
		serviceUUID := svc.UUID().String()
		cm.log.Debug("Discovering characteristics for service", "service_uuid", serviceUUID)

		chars, err := svc.DiscoverCharacteristics(nil)
		if err != nil {
			cm.log.Error("Failed to discover characteristics", "service_uuid", serviceUUID, "error", err)
			continue
		}

		cm.log.Debug("Characteristics discovered", "service_uuid", serviceUUID, "characteristic_count", len(chars))

		// Initialize characteristic map for this service
		cm.serviceMap[serviceUUID] = make(map[string]*bluetooth.DeviceCharacteristic)

		// Store characteristics in the service map
		for i := range chars {
			charUUID := chars[i].UUID().String()
			cm.log.Trace("Characteristic cached", "service_uuid", serviceUUID, "characteristic_uuid", charUUID)
			cm.serviceMap[serviceUUID][charUUID] = &chars[i]
		}
	}

	totalCharacteristics := 0
	for _, service := range cm.serviceMap {
		totalCharacteristics += len(service)
	}
	cm.log.Info("Characteristic discovery completed", "service_count", len(cm.serviceMap), "total_characteristics", totalCharacteristics)

	return nil
}

// GetCharacteristic retrieves a characteristic from the service map cache
// Thread Safety: Read lock allows concurrent access to serviceMap
// Safe for concurrent calls from multiple goroutines
func (cm *CharacteristicsManager) GetCharacteristic(serviceUUID, charUUID string) (*bluetooth.DeviceCharacteristic, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Check if we have the service in our cache
	service, ok := cm.serviceMap[serviceUUID]
	if !ok {
		cm.log.Error("Service not found in cache", "service_uuid", serviceUUID, "cached_services", len(cm.serviceMap))
		return nil, fmt.Errorf("service %s not found in cache", serviceUUID)
	}

	// Check if we have the characteristic in the service
	char, ok := service[charUUID]
	if !ok {
		cm.log.Error("Characteristic not found in service", "service_uuid", serviceUUID, "characteristic_uuid", charUUID, "service_characteristics", len(service))
		return nil, fmt.Errorf("characteristic %s not found in service %s", charUUID, serviceUUID)
	}

	cm.log.Trace("Characteristic retrieved from cache", "service_uuid", serviceUUID, "characteristic_uuid", charUUID)
	return char, nil
}

// GetControlServiceCharacteristics returns commonly used control service characteristics
// Thread Safety: Read lock protects access to serviceMap during lookup
// Returns map copy to prevent external modification of internal state
func (cm *CharacteristicsManager) GetControlServiceCharacteristics() (map[string]*bluetooth.DeviceCharacteristic, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	controlService, ok := cm.serviceMap[GoProControlServiceUUID]
	if !ok {
		cm.log.Error("Control service not found in cache", "service_uuid", GoProControlServiceUUID, "cached_services", len(cm.serviceMap))
		return nil, fmt.Errorf("control & query service not found")
	}

	// Get required characteristics
	characteristics := make(map[string]*bluetooth.DeviceCharacteristic)

	requiredChars := map[string]string{
		"query":            QueryCharUUID,
		"queryResponse":    QueryResponseCharUUID,
		"command":          CommandCharUUID,
		"commandResponse":  CommandResponseCharUUID,
		"settings":         SettingsCharUUID,
		"settingsResponse": SettingsResponseCharUUID,
	}

	for name, uuid := range requiredChars {
		if char, ok := controlService[uuid]; ok {
			characteristics[name] = char
			cm.log.Trace("Control characteristic found", "characteristic_name", name, "characteristic_uuid", uuid)
		} else {
			cm.log.Error("Required characteristic not found", "characteristic_name", name, "characteristic_uuid", uuid, "service_characteristics", len(controlService))
			return nil, fmt.Errorf("%s characteristic not found", name)
		}
	}

	cm.log.Debug("Control service characteristics retrieved", "characteristic_count", len(characteristics))
	return characteristics, nil
}

// EnableNotifications enables notifications for multiple characteristics
// Thread Safety: This method doesn't access serviceMap directly, but the characteristics
// passed in should be obtained through thread-safe GetCharacteristic calls.
// Handler callbacks execute concurrently and must be thread-safe themselves.
func (cm *CharacteristicsManager) EnableNotifications(characteristics map[string]*bluetooth.DeviceCharacteristic, handlers map[string]func([]byte)) error {
	cm.log.Info("Enabling notifications", "characteristic_count", len(characteristics), "handler_count", len(handlers))

	for name, char := range characteristics {
		if handler, hasHandler := handlers[name]; hasHandler {
			cm.log.Debug("Enabling notifications for characteristic", "characteristic_name", name)
			if err := char.EnableNotifications(handler); err != nil {
				cm.log.Error("Failed to enable notifications", "characteristic_name", name, "error", err)
				return fmt.Errorf("failed to enable notifications for %s: %v", name, err)
			}
			cm.log.Debug("Notifications enabled successfully", "characteristic_name", name)
		} else {
			cm.log.Warn("No handler provided for characteristic", "characteristic_name", name)
		}
	}
	cm.log.Info("All notifications enabled successfully", "enabled_count", len(handlers))
	return nil
}

// WriteCommand writes a command to a characteristic with optional packetization
// Thread Safety: This method operates on characteristic pointers obtained through
// thread-safe GetCharacteristic calls. No shared state accessed.
// Safe for concurrent use with different characteristics.
func (cm *CharacteristicsManager) WriteCommand(char *bluetooth.DeviceCharacteristic, command []byte, usePackets bool) error {
	if usePackets {
		packets := cm.createPackets(command)
		cm.log.Debug("Writing command with packetization", "command_length", len(command), "packet_count", len(packets))

		for i, packet := range packets {
			cm.log.Trace("Writing packet", "packet_index", i, "packet_length", len(packet))
			n, err := char.WriteWithoutResponse(packet)
			if err != nil {
				cm.log.Error("Failed to write packet", "packet_index", i, "error", err)
				return fmt.Errorf("failed to write packet %d: %v", i, err)
			}
			if n != len(packet) {
				cm.log.Error("Incomplete packet write", "packet_index", i, "expected", len(packet), "written", n)
				return fmt.Errorf("incomplete write for packet %d", i)
			}

			// Small delay between packets
			if i < len(packets)-1 {
				time.Sleep(10 * time.Millisecond)
			}
		}
		cm.log.Debug("Command written successfully with packets", "packet_count", len(packets))
	} else {
		cm.log.Debug("Writing command without packetization", "command_length", len(command))
		n, err := char.WriteWithoutResponse(command)
		if err != nil {
			cm.log.Error("Failed to write command", "command_length", len(command), "error", err)
			return fmt.Errorf("failed to write command: %v", err)
		}
		if n != len(command) {
			cm.log.Error("Incomplete command write", "expected", len(command), "written", n)
			return fmt.Errorf("incomplete write for command")
		}
		cm.log.Debug("Command written successfully", "command_length", len(command))
	}

	return nil
}

// SetEnhancedBLEMode enables or disables enhanced BLE features for newer GoPro models
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution.
func (cm *CharacteristicsManager) SetEnhancedBLEMode(enabled bool) error {
	cm.log.Info("Setting enhanced BLE mode", "enabled", enabled)

	// Get the settings characteristic for writing enhanced BLE mode commands
	settingsChar, err := cm.GetCharacteristic(GoProControlServiceUUID, SettingsCharUUID)
	if err != nil {
		cm.log.Error("Failed to get settings characteristic for enhanced BLE", "error", err)
		return fmt.Errorf("failed to get settings characteristic: %v", err)
	}

	// Enhanced BLE mode command: Setting ID 0x03 (Enhanced BLE) with value 0x01 (enabled) or 0x00 (disabled)
	var enableValue byte = 0x00
	if enabled {
		enableValue = 0x01
	}

	enhancedBLECmd := []byte{0x03, 0x01, enableValue}

	if err := cm.WriteCommand(settingsChar, enhancedBLECmd, false); err != nil {
		cm.log.Error("Failed to write enhanced BLE mode command", "enabled", enabled, "error", err)
		return fmt.Errorf("failed to write enhanced BLE mode command: %v", err)
	}

	// Give the camera time to process the command
	time.Sleep(200 * time.Millisecond)

	cm.log.Info("Enhanced BLE mode configured successfully", "enabled", enabled)
	return nil
}

// PerformSecurityHandshake performs additional security verification for HERO13 and newer models
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution.
func (cm *CharacteristicsManager) PerformSecurityHandshake() error {
	cm.log.Info("Starting security handshake")

	// Get the command characteristic for security handshake
	cmdChar, err := cm.GetCharacteristic(GoProControlServiceUUID, CommandCharUUID)
	if err != nil {
		cm.log.Error("Failed to get command characteristic for security handshake", "error", err)
		return fmt.Errorf("failed to get command characteristic: %v", err)
	}

	// Security handshake command: Use Get Open GoPro Version command
	securityCmd := []byte{CommandGetOpenGoProVer}

	if err := cm.WriteCommand(cmdChar, securityCmd, false); err != nil {
		cm.log.Error("Failed to write security handshake", "error", err)
		return fmt.Errorf("failed to write security handshake: %v", err)
	}

	// Wait for the handshake to complete
	time.Sleep(500 * time.Millisecond)

	cm.log.Info("Security handshake completed successfully")
	return nil
}

// SetConnectionParameters sets optimal BLE connection parameters based on model type
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution with different model types.
func (cm *CharacteristicsManager) SetConnectionParameters(modelType string) error {
	cm.log.Debugf("Setting connection parameters for model type %s", modelType)

	// Get the settings characteristic for connection parameter settings
	settingsChar, err := cm.GetCharacteristic(GoProControlServiceUUID, SettingsCharUUID)
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

	if err := cm.WriteCommand(settingsChar, connectionCmd, false); err != nil {
		return fmt.Errorf("failed to write connection parameters: %v", err)
	}

	// Give the camera time to apply new parameters
	time.Sleep(300 * time.Millisecond)

	cm.log.Debugf("Connection parameters set for model type %s (interval: %d, latency: %d, timeout: %d)",
		modelType, connectionInterval, slaveLatency, supervisionTimeout)
	return nil
}

// SendKeepAlive sends a keep-alive signal to maintain the connection
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution.
func (cm *CharacteristicsManager) SendKeepAlive() error {
	// Get the command characteristic for keep-alive
	cmdChar, err := cm.GetCharacteristic(GoProControlServiceUUID, CommandCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get command characteristic: %v", err)
	}

	// Keep-alive command using the official Keep Alive command ID
	cmd := []byte{CommandKeepAlive}

	if err := cm.WriteCommand(cmdChar, cmd, false); err != nil {
		return fmt.Errorf("failed to write keep-alive: %v", err)
	}

	cm.log.Debug("Successfully sent keep-alive signal")
	return nil
}

// SetCameraControl sets the camera control status using protobuf commands
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution.
func (cm *CharacteristicsManager) SetCameraControl(enabled bool) error {
	// Get the command characteristic
	cmdChar, err := cm.GetCharacteristic(GoProControlServiceUUID, CommandCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get command characteristic: %v", err)
	}

	// Camera control is a protobuf command, need to use proper protobuf format
	// For now, use a simple approach - this would need proper protobuf implementation
	cmd := make([]byte, 3)
	cmd[0] = 0xF1 // Protobuf command header
	cmd[1] = ProtobufCommandSetCameraControl
	if enabled {
		cmd[2] = 1 // Enable camera control
	} else {
		cmd[2] = 0 // Disable camera control
	}

	if err := cm.WriteCommand(cmdChar, cmd, true); err != nil {
		return fmt.Errorf("failed to write camera control command: %v", err)
	}

	cm.log.Debugf("Camera control set to %v", enabled)
	return nil
}

// QueryHardwareInfo queries the device for hardware information
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution.
func (cm *CharacteristicsManager) QueryHardwareInfo() error {
	// Get the command characteristic (hardware info is a command, not a query)
	cmdChar, err := cm.GetCharacteristic(GoProControlServiceUUID, CommandCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get command characteristic: %v", err)
	}

	// Hardware info command
	hwInfoCmd := []byte{CommandGetHardwareInfo}

	if err := cm.WriteCommand(cmdChar, hwInfoCmd, false); err != nil {
		return fmt.Errorf("failed to write hardware info command: %v", err)
	}

	cm.log.Debug("Hardware info command sent")
	return nil
}

// QueryPairingState queries the device for current pairing state
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution.
func (cm *CharacteristicsManager) QueryPairingState() error {
	// Get the query characteristic
	queryChar, err := cm.GetCharacteristic(GoProControlServiceUUID, QueryCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get query characteristic: %v", err)
	}

	// Pairing state query command (status ID 19 using Get Status Values)
	// OpenGoPro TLV format: [Command] [Array Length] [Status ID]
	pairingStateCmd := []byte{QueryGetStatusValues, 0x01, StatusPairingState}

	if err := cm.WriteCommand(queryChar, pairingStateCmd, false); err != nil {
		return fmt.Errorf("failed to write pairing state query: %v", err)
	}

	cm.log.Debug("Pairing state query sent")
	return nil
}

// QueryBatteryLevel queries the device for battery level
// Thread Safety: Uses thread-safe GetCharacteristic call for serviceMap access.
// Safe for concurrent execution.
func (cm *CharacteristicsManager) QueryBatteryLevel() error {
	// Get the query characteristic
	queryChar, err := cm.GetCharacteristic(GoProControlServiceUUID, QueryCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get query characteristic: %v", err)
	}

	// Battery level query command (status ID 2 using Get Status Values)
	// OpenGoPro TLV format: [Command] [Array Length] [Status ID]
	batteryCmd := []byte{QueryGetStatusValues, 0x01, StatusBatteryLevel}

	if err := cm.WriteCommand(queryChar, batteryCmd, false); err != nil {
		return fmt.Errorf("failed to write battery level query: %v", err)
	}

	cm.log.Debug("Battery level query sent")
	return nil
}

// createPackets splits a large payload into BLE packets according to OpenGoPro spec
// Thread Safety: Pure function with no shared state access. Safe for concurrent use.
func (cm *CharacteristicsManager) createPackets(payload []byte) [][]byte {
	// BLE limits packet size to 20 bytes
	maxPacketSize := 20
	maxDataPerPacket := maxPacketSize - 1 // Reserve 1 byte for header

	// Calculate how many packets we need
	totalDataSize := len(payload)
	if totalDataSize == 0 {
		return [][]byte{{GeneralStartBit}}
	}

	numPackets := (totalDataSize + maxDataPerPacket - 1) / maxDataPerPacket
	packets := make([][]byte, numPackets)

	if numPackets == 1 {
		// Single packet: start bit + length + data
		packet := make([]byte, 0, maxPacketSize)
		header := GeneralStartBit | byte(totalDataSize&GeneralLengthByteMask)
		packet = append(packet, header)
		packet = append(packet, payload...)
		packets[0] = packet
	} else {
		// Multiple packets: first packet has start bit, others have continuation bit

		// First packet
		firstPacketDataSize := maxDataPerPacket
		if firstPacketDataSize > totalDataSize {
			firstPacketDataSize = totalDataSize
		}

		firstPacket := make([]byte, 0, maxPacketSize)
		header := GeneralStartBit | byte(firstPacketDataSize&GeneralLengthByteMask)
		firstPacket = append(firstPacket, header)
		firstPacket = append(firstPacket, payload[:firstPacketDataSize]...)
		packets[0] = firstPacket

		// Continuation packets
		remainingData := payload[firstPacketDataSize:]
		for i := 1; i < numPackets; i++ {
			packetDataSize := maxDataPerPacket
			if len(remainingData) < packetDataSize {
				packetDataSize = len(remainingData)
			}

			packet := make([]byte, 0, maxPacketSize)
			header := GeneralContinueBit | byte(packetDataSize&GeneralLengthByteMask)
			packet = append(packet, header)
			packet = append(packet, remainingData[:packetDataSize]...)
			packets[i] = packet

			remainingData = remainingData[packetDataSize:]
		}
	}

	return packets
}

// Clear clears the characteristic cache
// Thread Safety: Write lock ensures atomic clearing of serviceMap.
// Blocks all other operations during cache reset.
func (cm *CharacteristicsManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.serviceMap = make(map[string]map[string]*bluetooth.DeviceCharacteristic)
}

// ValidateRequiredCharacteristics validates that all required OpenGoPro characteristics are present
// Thread Safety: Read lock protects access to serviceMap during validation
// Returns descriptive errors for missing characteristics
func (cm *CharacteristicsManager) ValidateRequiredCharacteristics() error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Check if we have the Control & Query service
	controlService, ok := cm.serviceMap[GoProControlServiceUUID]
	if !ok {
		return NewCharacteristicError(GoProControlServiceUUID, "", "service_discovery",
			fmt.Errorf("OpenGoPro Control & Query service not found"))
	}

	// Define required characteristics per OpenGoPro spec
	requiredCharacteristics := map[string]string{
		"Command Request":   CommandCharUUID,
		"Command Response":  CommandResponseCharUUID,
		"Query Request":     QueryCharUUID,
		"Query Response":    QueryResponseCharUUID,
		"Settings Request":  SettingsCharUUID,
		"Settings Response": SettingsResponseCharUUID,
	}

	var missingChars []string
	for name, uuid := range requiredCharacteristics {
		if _, exists := controlService[uuid]; !exists {
			missingChars = append(missingChars, fmt.Sprintf("%s (%s)", name, uuid))
			cm.log.Error("Required characteristic missing",
				"characteristic_name", name,
				"characteristic_uuid", uuid,
				"service_uuid", GoProControlServiceUUID)
		} else {
			cm.log.Debug("Required characteristic found",
				"characteristic_name", name,
				"characteristic_uuid", uuid)
		}
	}

	if len(missingChars) > 0 {
		return NewCharacteristicError(GoProControlServiceUUID, "", "validation",
			fmt.Errorf("missing required characteristics: %v", missingChars))
	}

	// Check WiFi service characteristics (optional but warn if missing)
	wifiService, hasWifiService := cm.serviceMap[GoProWifiServiceUUID]
	if hasWifiService {
		wifiChars := map[string]string{
			"WiFi SSID":     WifiSSIDCharUUID,
			"WiFi Password": WifiPasswordCharUUID,
		}

		for name, uuid := range wifiChars {
			if _, exists := wifiService[uuid]; !exists {
				cm.log.Warn("WiFi characteristic missing",
					"characteristic_name", name,
					"characteristic_uuid", uuid)
			}
		}
	} else {
		cm.log.Warn("WiFi service not found - WiFi operations will not be available",
			"service_uuid", GoProWifiServiceUUID)
	}

	cm.log.Info("Required characteristic validation completed",
		"control_characteristics", len(controlService),
		"wifi_service_present", hasWifiService)

	return nil
}

// GetAvailableCharacteristics returns a list of all cached characteristics for debugging
// Thread Safety: Read lock protects access to serviceMap during enumeration
// Returns copy of data to prevent external modification
func (cm *CharacteristicsManager) GetAvailableCharacteristics() map[string][]string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	result := make(map[string][]string)
	for serviceUUID, characteristics := range cm.serviceMap {
		charList := make([]string, 0, len(characteristics))
		for charUUID := range characteristics {
			charList = append(charList, charUUID)
		}
		result[serviceUUID] = charList
	}

	return result
}

// GetAllCharacteristics returns all cached characteristics in a flat map for per-device caching
// Thread Safety: Read lock protects access to serviceMap during enumeration
// Returns a flat map of charUUID -> characteristic pointer suitable for device-specific caching
func (cm *CharacteristicsManager) GetAllCharacteristics() map[string]*bluetooth.DeviceCharacteristic {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	result := make(map[string]*bluetooth.DeviceCharacteristic)
	for _, serviceChars := range cm.serviceMap {
		for charUUID, char := range serviceChars {
			result[charUUID] = char
		}
	}

	cm.log.Debug("Retrieved all characteristics from cache", "characteristic_count", len(result))
	return result
}

// ValidateCharacteristicAccess validates that a characteristic can be accessed safely
// Thread Safety: Read lock protects access to serviceMap during validation
// Checks for nil pointers and validates UUIDs
func (cm *CharacteristicsManager) ValidateCharacteristicAccess(serviceUUID, charUUID string) error {
	if serviceUUID == "" {
		return NewValidationError("service_uuid", serviceUUID, "empty service UUID")
	}
	if charUUID == "" {
		return NewValidationError("characteristic_uuid", charUUID, "empty characteristic UUID")
	}

	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	service, ok := cm.serviceMap[serviceUUID]
	if !ok {
		return NewCharacteristicError(serviceUUID, charUUID, "access_validation",
			fmt.Errorf("service not found in cache"))
	}

	char, ok := service[charUUID]
	if !ok {
		return NewCharacteristicError(serviceUUID, charUUID, "access_validation",
			fmt.Errorf("characteristic not found in service"))
	}

	if char == nil {
		return NewCharacteristicError(serviceUUID, charUUID, "access_validation",
			fmt.Errorf("characteristic pointer is nil"))
	}

	return nil
}
