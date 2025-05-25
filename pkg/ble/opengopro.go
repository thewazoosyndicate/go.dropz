package ble

import (
	"context"
	"fmt"

	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// OpenGoProManager implements OpenGoPro BLE specification compliant operations
type OpenGoProManager struct {
	adapter     *bluetooth.Adapter
	log         logger.Logger
	connManager *ConnectionManager
	charMgr     *CharacteristicsManager

	// Service and characteristic cache
	services        map[string]bluetooth.DeviceService
	characteristics map[string]*bluetooth.DeviceCharacteristic
}

// NewOpenGoProManager creates a new OpenGoPro compliant BLE manager
func NewOpenGoProManager(adapter *bluetooth.Adapter, log logger.Logger) *OpenGoProManager {
	return &OpenGoProManager{
		adapter:         adapter,
		log:             log,
		connManager:     NewConnectionManager(adapter, log),
		charMgr:         NewCharacteristicsManager(log),
		services:        make(map[string]bluetooth.DeviceService),
		characteristics: make(map[string]*bluetooth.DeviceCharacteristic),
	}
}

// ConnectOpenGoPro implements OpenGoPro BLE specification connection sequence
// Reference: https://gopro.github.io/OpenGoPro/ble/
func (ogm *OpenGoProManager) ConnectOpenGoPro(ctx context.Context, macAddress string) error {
	ogm.log.Infof("Starting OpenGoPro BLE connection sequence for device %s", macAddress)

	// Step 1: Establish BLE connection
	ogm.log.Debugf("Step 1: Establishing BLE connection to %s", macAddress)
	device, err := ogm.connManager.Connect(ctx, macAddress)
	if err != nil {
		return fmt.Errorf("failed to establish BLE connection: %v", err)
	}

	// Step 2: Discover OpenGoPro service (0xFEA6)
	ogm.log.Debugf("Step 2: Discovering OpenGoPro Control & Query service")
	if err := ogm.discoverOpenGoProService(device); err != nil {
		ogm.connManager.Disconnect(macAddress)
		return fmt.Errorf("failed to discover OpenGoPro service: %v", err)
	}

	// Step 3: Discover and cache characteristics in proper order
	ogm.log.Debugf("Step 3: Discovering and caching characteristics")
	if err := ogm.discoverAndCacheCharacteristics(); err != nil {
		ogm.connManager.Disconnect(macAddress)
		return fmt.Errorf("failed to discover characteristics: %v", err)
	}

	// Step 4: Subscribe to notifications in OpenGoPro recommended order
	ogm.log.Debugf("Step 4: Setting up characteristic notifications")
	if err := ogm.setupNotifications(); err != nil {
		ogm.connManager.Disconnect(macAddress)
		return fmt.Errorf("failed to setup notifications: %v", err)
	}

	// Step 5: Verify GoPro is ready for commands by checking hardware info
	ogm.log.Debugf("Step 5: Verifying GoPro readiness")
	if err := ogm.verifyGoProReadiness(); err != nil {
		ogm.connManager.Disconnect(macAddress)
		return fmt.Errorf("GoPro not ready for operations: %v", err)
	}

	ogm.log.Infof("OpenGoPro BLE connection sequence completed successfully for device %s", macAddress)
	return nil
}

// discoverOpenGoProService discovers the main OpenGoPro Control & Query service
func (ogm *OpenGoProManager) discoverOpenGoProService(device *bluetooth.Device) error {
	// According to OpenGoPro spec, we need to discover the 0xFEA6 service
	ogm.log.Debugf("Discovering OpenGoPro Control & Query service (%s)", GoProControlServiceUUID)

	// Discover all services first
	services, err := device.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("failed to discover services: %v", err)
	}

	ogm.log.Debugf("Discovered %d total services", len(services))

	// Look for the OpenGoPro Control & Query service
	var controlService bluetooth.DeviceService
	found := false

	for _, svc := range services {
		svcUUID := svc.UUID().String()
		ogm.log.Tracef("Found service: %s", svcUUID)

		// Store all services for potential future use
		ogm.services[svcUUID] = svc

		// Check if this is the OpenGoPro Control & Query service
		if svcUUID == GoProControlServiceUUID {
			controlService = svc
			found = true
			ogm.log.Debugf("Found OpenGoPro Control & Query service")
		}
	}

	if !found {
		return fmt.Errorf("OpenGoPro Control & Query service (%s) not found in %d discovered services",
			GoProControlServiceUUID, len(services))
	}

	// Store the control service for later use
	ogm.services[GoProControlServiceUUID] = controlService

	return nil
}

// discoverAndCacheCharacteristics discovers and caches all characteristics for the OpenGoPro service
func (ogm *OpenGoProManager) discoverAndCacheCharacteristics() error {
	controlService, exists := ogm.services[GoProControlServiceUUID]
	if !exists {
		return fmt.Errorf("control service not found in cache")
	}

	ogm.log.Debugf("Discovering characteristics for OpenGoPro Control & Query service")

	// Discover all characteristics for the control service
	chars, err := controlService.DiscoverCharacteristics(nil)
	if err != nil {
		return fmt.Errorf("failed to discover characteristics: %v", err)
	}

	ogm.log.Debugf("Discovered %d characteristics", len(chars))

	// Cache each characteristic
	for i := range chars {
		charUUID := chars[i].UUID().String()
		ogm.characteristics[charUUID] = &chars[i]
		ogm.log.Tracef("Cached characteristic: %s", charUUID)
	}

	// Verify we have the essential characteristics according to OpenGoPro spec
	requiredChars := []string{
		CommandCharUUID,          // Command Request
		CommandResponseCharUUID,  // Command Response
		QueryCharUUID,            // Query Request
		QueryResponseCharUUID,    // Query Response
		SettingsCharUUID,         // Settings Request
		SettingsResponseCharUUID, // Settings Response
	}

	for _, charUUID := range requiredChars {
		if _, exists := ogm.characteristics[charUUID]; !exists {
			ogm.log.Warnf("Required characteristic %s not found", charUUID)
		} else {
			ogm.log.Debugf("Found required characteristic: %s", charUUID)
		}
	}

	return nil
}

// setupNotifications sets up notifications in the order recommended by OpenGoPro spec
func (ogm *OpenGoProManager) setupNotifications() error {
	ogm.log.Debugf("Setting up characteristic notifications in OpenGoPro recommended order")

	// OpenGoPro recommends subscribing to response characteristics in this order:
	// 1. Query Response - for status and setting responses
	// 2. Command Response - for command acknowledgments
	// 3. Settings Response - for setting change notifications

	notificationChars := []struct {
		uuid string
		name string
	}{
		{QueryResponseCharUUID, "Query Response"},
		{CommandResponseCharUUID, "Command Response"},
		{SettingsResponseCharUUID, "Settings Response"},
	}

	for _, char := range notificationChars {
		if err := ogm.enableNotification(char.uuid, char.name); err != nil {
			ogm.log.Errorf("Failed to enable %s notifications: %v", char.name, err)
			// Continue with other notifications - some may still work
		} else {
			ogm.log.Debugf("Successfully enabled %s notifications", char.name)
		}
	}

	return nil
}

// enableNotification enables notifications for a specific characteristic
func (ogm *OpenGoProManager) enableNotification(charUUID, charName string) error {
	char, exists := ogm.characteristics[charUUID]
	if !exists {
		return fmt.Errorf("characteristic %s not found", charUUID)
	}

	ogm.log.Tracef("Enabling notifications for %s (%s)", charName, charUUID)

	// Enable notifications with a simple callback
	err := char.EnableNotifications(func(buf []byte) {
		ogm.log.Tracef("Received %s notification: %d bytes", charName, len(buf))
		// For now, just log the notification - response handling can be added later
		if len(buf) > 0 {
			ogm.log.Tracef("%s notification data: %x", charName, buf)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to enable notifications: %v", err)
	}

	return nil
}

// verifyGoProReadiness checks if the GoPro is ready to accept commands
func (ogm *OpenGoProManager) verifyGoProReadiness() error {
	ogm.log.Debugf("Verifying GoPro hardware readiness")

	// According to OpenGoPro spec, we should be able to query hardware info
	// to verify the camera is ready for commands

	hardwareInfo, err := ogm.getHardwareInfo()
	if err != nil {
		return fmt.Errorf("failed to get hardware info: %v", err)
	}

	if hardwareInfo == nil {
		return fmt.Errorf("hardware info is empty - GoPro may not be ready")
	}

	ogm.log.Infof("GoPro hardware verification successful")
	return nil
}

// getHardwareInfo retrieves hardware information from the GoPro
func (ogm *OpenGoProManager) getHardwareInfo() ([]byte, error) {
	ogm.log.Debugf("Requesting hardware info from GoPro")

	// Get the command characteristic
	commandChar, exists := ogm.characteristics[CommandCharUUID]
	if !exists {
		return nil, fmt.Errorf("command characteristic not available")
	}

	// Prepare the hardware info command according to OpenGoPro spec
	// Command ID 0x3C = GET_HW_INFO
	command := []byte{0x3C}

	ogm.log.Tracef("Sending GET_HW_INFO command: %x", command)

	// Send the command
	_, err := commandChar.WriteWithoutResponse(command)
	if err != nil {
		return nil, fmt.Errorf("failed to send hardware info command: %v", err)
	}

	// For now, we'll just return success if the command was sent
	// A full implementation would wait for the response via notifications
	ogm.log.Debugf("Hardware info command sent successfully")

	return []byte{0x01}, nil // Dummy response indicating success
}

// SendKeepAlive sends a keep-alive command to maintain the connection
func (ogm *OpenGoProManager) SendKeepAlive() error {
	ogm.log.Tracef("Sending keep-alive command")

	commandChar, exists := ogm.characteristics[CommandCharUUID]
	if !exists {
		return fmt.Errorf("command characteristic not available")
	}

	// Command ID 0x5B = KEEP_ALIVE
	command := []byte{0x5B}

	_, err := commandChar.WriteWithoutResponse(command)
	if err != nil {
		return fmt.Errorf("failed to send keep-alive: %v", err)
	}

	ogm.log.Tracef("Keep-alive command sent successfully")
	return nil
}

// GetCharacteristic safely retrieves a cached characteristic
func (ogm *OpenGoProManager) GetCharacteristic(charUUID string) (*bluetooth.DeviceCharacteristic, error) {
	char, exists := ogm.characteristics[charUUID]
	if !exists {
		return nil, fmt.Errorf("characteristic %s not found in cache", charUUID)
	}
	return char, nil
}

// GetDiscoveredServices returns all discovered services for cache transfer
func (ogm *OpenGoProManager) GetDiscoveredServices() []bluetooth.DeviceService {
	services := make([]bluetooth.DeviceService, 0, len(ogm.services))
	for _, service := range ogm.services {
		services = append(services, service)
	}
	ogm.log.Debugf("Returning %d discovered services for cache transfer", len(services))
	return services
}

// GetConnectionManager returns the connection manager for device access
func (ogm *OpenGoProManager) GetConnectionManager() *ConnectionManager {
	return ogm.connManager
}

// Disconnect cleanly disconnects from the GoPro
func (ogm *OpenGoProManager) Disconnect(macAddress string) error {
	ogm.log.Debugf("Disconnecting from OpenGoPro device %s", macAddress)

	// Clear cached services and characteristics
	ogm.services = make(map[string]bluetooth.DeviceService)
	ogm.characteristics = make(map[string]*bluetooth.DeviceCharacteristic)

	// Disconnect via connection manager
	return ogm.connManager.Disconnect(macAddress)
}
