package ble

import (
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// CharacteristicsManager handles BLE characteristic operations
type CharacteristicsManager struct {
	serviceMap map[string]map[string]*bluetooth.DeviceCharacteristic // serviceUUID -> (charUUID -> characteristic)
	mutex      sync.RWMutex
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
func (cm *CharacteristicsManager) DiscoverAndCacheCharacteristics(services []bluetooth.DeviceService) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Initialize service map
	cm.serviceMap = make(map[string]map[string]*bluetooth.DeviceCharacteristic)

	// Discover all characteristics and build service map
	for _, svc := range services {
		serviceUUID := svc.UUID().String()
		cm.log.Debugf("Service: %s", serviceUUID)

		chars, err := svc.DiscoverCharacteristics(nil)
		if err != nil {
			cm.log.Warnf("Failed to discover characteristics for service %s: %v", serviceUUID, err)
			continue
		}

		cm.log.Debugf("Found %d characteristics for service %s", len(chars), serviceUUID)

		// Initialize characteristic map for this service
		cm.serviceMap[serviceUUID] = make(map[string]*bluetooth.DeviceCharacteristic)

		// Store characteristics in the service map
		for i := range chars {
			charUUID := chars[i].UUID().String()
			cm.log.Debugf("Characteristic: %s", charUUID)
			cm.serviceMap[serviceUUID][charUUID] = &chars[i]
		}
	}

	return nil
}

// GetCharacteristic retrieves a characteristic from the service map cache
func (cm *CharacteristicsManager) GetCharacteristic(serviceUUID, charUUID string) (*bluetooth.DeviceCharacteristic, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Check if we have the service in our cache
	service, ok := cm.serviceMap[serviceUUID]
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

// GetControlServiceCharacteristics returns commonly used control service characteristics
func (cm *CharacteristicsManager) GetControlServiceCharacteristics() (map[string]*bluetooth.DeviceCharacteristic, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	controlService, ok := cm.serviceMap[GoProControlServiceUUID]
	if !ok {
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
		} else {
			return nil, fmt.Errorf("%s characteristic not found", name)
		}
	}

	return characteristics, nil
}

// EnableNotifications enables notifications for multiple characteristics
func (cm *CharacteristicsManager) EnableNotifications(characteristics map[string]*bluetooth.DeviceCharacteristic, handlers map[string]func([]byte)) error {
	for name, char := range characteristics {
		if handler, hasHandler := handlers[name]; hasHandler {
			cm.log.Debugf("Enabling notifications for %s", name)
			if err := char.EnableNotifications(handler); err != nil {
				cm.log.Warnf("Failed to enable notifications for %s: %v", name, err)
				return fmt.Errorf("failed to enable notifications for %s: %v", name, err)
			}
		}
	}
	return nil
}

// WriteCommand writes a command to a characteristic with optional packetization
func (cm *CharacteristicsManager) WriteCommand(char *bluetooth.DeviceCharacteristic, command []byte, usePackets bool) error {
	if usePackets {
		packets := cm.createPackets(command)
		for i, packet := range packets {
			n, err := char.WriteWithoutResponse(packet)
			if err != nil {
				return fmt.Errorf("failed to write packet %d: %v", i, err)
			}
			if n != len(packet) {
				return fmt.Errorf("incomplete write for packet %d", i)
			}

			// Small delay between packets
			if i < len(packets)-1 {
				time.Sleep(10 * time.Millisecond)
			}
		}
	} else {
		n, err := char.WriteWithoutResponse(command)
		if err != nil {
			return fmt.Errorf("failed to write command: %v", err)
		}
		if n != len(command) {
			return fmt.Errorf("incomplete write for command")
		}
	}

	return nil
}

// SetEnhancedBLEMode enables or disables enhanced BLE features for newer GoPro models
func (cm *CharacteristicsManager) SetEnhancedBLEMode(enabled bool) error {
	// Get the settings characteristic for writing enhanced BLE mode commands
	settingsChar, err := cm.GetCharacteristic(GoProControlServiceUUID, SettingsCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get settings characteristic: %v", err)
	}

	// Enhanced BLE mode command: Setting ID 0x03 (Enhanced BLE) with value 0x01 (enabled) or 0x00 (disabled)
	var enableValue byte = 0x00
	if enabled {
		enableValue = 0x01
	}

	enhancedBLECmd := []byte{0x03, 0x01, enableValue}

	if err := cm.WriteCommand(settingsChar, enhancedBLECmd, false); err != nil {
		return fmt.Errorf("failed to write enhanced BLE mode command: %v", err)
	}

	// Give the camera time to process the command
	time.Sleep(200 * time.Millisecond)

	cm.log.Debugf("Enhanced BLE mode set to %v", enabled)
	return nil
}

// PerformSecurityHandshake performs additional security verification for HERO13 and newer models
func (cm *CharacteristicsManager) PerformSecurityHandshake() error {
	cm.log.Debug("Performing security handshake")

	// Get the command characteristic for security handshake
	cmdChar, err := cm.GetCharacteristic(GoProControlServiceUUID, CommandCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get command characteristic: %v", err)
	}

	// Security handshake command: Command ID 0x5F (Security Challenge)
	securityCmd := []byte{0x5F, 0x01} // Simple handshake request

	if err := cm.WriteCommand(cmdChar, securityCmd, true); err != nil {
		return fmt.Errorf("failed to write security handshake: %v", err)
	}

	// Wait for the handshake to complete
	time.Sleep(500 * time.Millisecond)

	cm.log.Debug("Security handshake completed")
	return nil
}

// SetConnectionParameters sets optimal BLE connection parameters based on model type
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
func (cm *CharacteristicsManager) SendKeepAlive() error {
	// Get the settings characteristic for keep-alive
	settingsChar, err := cm.GetCharacteristic(GoProControlServiceUUID, SettingsCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get settings characteristic: %v", err)
	}

	// Keep-alive command using Turbo Active setting
	cmd := []byte{CommandSetTurboActive, 0x01, 0x00} // Set turbo active to false (keep-alive)

	if err := cm.WriteCommand(settingsChar, cmd, false); err != nil {
		return fmt.Errorf("failed to write keep-alive: %v", err)
	}

	cm.log.Debug("Successfully sent keep-alive signal")
	return nil
}

// SetCameraControl sets the camera control status
func (cm *CharacteristicsManager) SetCameraControl(enabled bool) error {
	// Get the command characteristic
	cmdChar, err := cm.GetCharacteristic(GoProControlServiceUUID, CommandCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get command characteristic: %v", err)
	}

	// Prepare camera control command
	cmd := make([]byte, 2)
	cmd[0] = CommandSetCameraControl
	if enabled {
		cmd[1] = 1 // Enable camera control
	} else {
		cmd[1] = 0 // Disable camera control
	}

	if err := cm.WriteCommand(cmdChar, cmd, true); err != nil {
		return fmt.Errorf("failed to write camera control command: %v", err)
	}

	cm.log.Debugf("Camera control set to %v", enabled)
	return nil
}

// QueryHardwareInfo queries the device for hardware information
func (cm *CharacteristicsManager) QueryHardwareInfo() error {
	// Get the query characteristic
	queryChar, err := cm.GetCharacteristic(GoProControlServiceUUID, QueryCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get query characteristic: %v", err)
	}

	// Hardware info query command
	hwInfoCmd := []byte{QueryGetHardwareInfo}

	if err := cm.WriteCommand(queryChar, hwInfoCmd, false); err != nil {
		return fmt.Errorf("failed to write hardware info query: %v", err)
	}

	cm.log.Debug("Hardware info query sent")
	return nil
}

// QueryPairingState queries the device for current pairing state
func (cm *CharacteristicsManager) QueryPairingState() error {
	// Get the query characteristic
	queryChar, err := cm.GetCharacteristic(GoProControlServiceUUID, QueryCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get query characteristic: %v", err)
	}

	// Pairing state query command (status ID 19)
	pairingStateCmd := []byte{QueryGetStatusValues, 19}

	if err := cm.WriteCommand(queryChar, pairingStateCmd, false); err != nil {
		return fmt.Errorf("failed to write pairing state query: %v", err)
	}

	cm.log.Debug("Pairing state query sent")
	return nil
}

// QueryBatteryLevel queries the device for battery level
func (cm *CharacteristicsManager) QueryBatteryLevel() error {
	// Get the query characteristic
	queryChar, err := cm.GetCharacteristic(GoProControlServiceUUID, QueryCharUUID)
	if err != nil {
		return fmt.Errorf("failed to get query characteristic: %v", err)
	}

	// Battery level query command (status ID 2)
	batteryCmd := []byte{QueryGetStatusValues, 2}

	if err := cm.WriteCommand(queryChar, batteryCmd, false); err != nil {
		return fmt.Errorf("failed to write battery level query: %v", err)
	}

	cm.log.Debug("Battery level query sent")
	return nil
}

// createPackets splits a large payload into BLE packets
func (cm *CharacteristicsManager) createPackets(payload []byte) [][]byte {
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

// Clear clears the characteristic cache
func (cm *CharacteristicsManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.serviceMap = make(map[string]map[string]*bluetooth.DeviceCharacteristic)
}
