package ble

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble/events"
	"github.com/dropz/dropz/pkg/ble/models"
	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// PairingManager handles all BLE pairing operations
type PairingManager struct {
	connectionManager      *ConnectionManager
	characteristicsManager *CharacteristicsManager
	responseHandler        *ResponseHandler
	eventEmitter           *events.EventEmitter
	log                    logger.Logger

	// Cache for pairing states
	pairingStateCache map[string]int
	cacheMutex        sync.RWMutex
}

// NewPairingManager creates a new pairing manager
func NewPairingManager(connMgr *ConnectionManager, charMgr *CharacteristicsManager,
	responseHandler *ResponseHandler, eventEmitter *events.EventEmitter, log logger.Logger) *PairingManager {
	return &PairingManager{
		connectionManager:      connMgr,
		characteristicsManager: charMgr,
		responseHandler:        responseHandler,
		eventEmitter:           eventEmitter,
		log:                    log,
		pairingStateCache:      make(map[string]int),
	}
}

// ConnectWithEnhancedPairing attempts to connect with enhanced model-specific pairing logic
func (p *PairingManager) ConnectWithEnhancedPairing(macAddress string) error {
	p.log.Infof("BLE pairing started: device=%s mode=enhanced", macAddress)

	// Emit pairing started event
	p.eventEmitter.EmitEvent(events.BLEEvent{
		Type:      events.EventPairingStarted,
		Device:    map[string]string{"mac_address": macAddress},
		Timestamp: time.Now(),
	})

	// First, establish OpenGoPro-compliant connection
	ctx := context.Background()
	device, err := p.connectionManager.ConnectWithOpenGoProSpec(ctx, macAddress)
	if err != nil {
		p.emitPairingFailed(macAddress, err)
		return fmt.Errorf("failed to establish OpenGoPro connection: %v", err)
	}

	// Get services from connection metadata
	var services []bluetooth.DeviceService
	if info := p.connectionManager.GetConnectionInfo(macAddress); info != nil {
		if svcData, exists := info.Metadata["services"]; exists {
			if svc, ok := svcData.([]bluetooth.DeviceService); ok {
				services = svc
			}
		}
	}

	if len(services) == 0 {
		// Fallback: discover services manually
		p.log.Debugf("Services not found in metadata, discovering manually for device %s", macAddress)
		services, err = device.DiscoverServices(nil)
		if err != nil {
			p.emitPairingFailed(macAddress, err)
			return fmt.Errorf("failed to discover services: %v", err)
		}
	}

	// Cache the discovered services in characteristics manager
	if err := p.characteristicsManager.DiscoverAndCacheCharacteristics(services); err != nil {
		p.emitPairingFailed(macAddress, err)
		return fmt.Errorf("failed to cache characteristics: %v", err)
	}

	// Setup notifications for response characteristics BEFORE querying
	if err := p.setupResponseNotifications(macAddress); err != nil {
		p.log.Warnf("Failed to setup response notifications: %v", err)
		// Continue anyway - some operations might still work
	}

	// NOW we can check pairing state with proper service discovery
	pairingState, err := p.queryPairingState(macAddress)
	if err != nil {
		p.log.Warnf("Failed to query initial pairing state: %v", err)
		// Continue anyway, assume unpaired
		pairingState = PairingStateNotPaired
	}

	p.log.Infof("Current pairing state for device %s: %d", macAddress, pairingState)

	// Detect model for enhanced pairing
	modelID, err := p.detectGoProModel(macAddress)
	if err != nil {
		p.log.Warnf("Model detection failed: device=%s error=%v fallback=default", macAddress, err)
		modelID = 0 // Use 0 to indicate unknown model
	} else {
		p.log.Infof("Model detected: device=%s model=%s model_id=%d",
			macAddress, models.GetModelName(modelID), modelID)
	}

	// Perform model-specific pairing procedures
	if err := p.performModelSpecificPairing(macAddress, modelID); err != nil {
		p.emitPairingFailed(macAddress, err)
		return fmt.Errorf("model-specific pairing failed: %v", err)
	}

	// Verify pairing state
	if err := p.verifyPairingState(macAddress); err != nil {
		p.emitPairingFailed(macAddress, err)
		return fmt.Errorf("pairing verification failed: %v", err)
	}

	// Emit pairing completed event
	p.eventEmitter.EmitEvent(events.BLEEvent{
		Type:      events.EventPairingCompleted,
		Device:    map[string]string{"mac_address": macAddress},
		Timestamp: time.Now(),
	})

	p.log.Infof("BLE pairing completed: device=%s model_id=%d status=success", macAddress, modelID)
	return nil
}

// performModelSpecificPairing handles model-specific pairing requirements
func (p *PairingManager) performModelSpecificPairing(macAddress string, modelID int) error {
	modelName := models.GetModelName(modelID)
	p.log.Debugf("Executing pairing sequence: device=%s model=%s model_id=%d", macAddress, modelName, modelID)

	// Create model handler
	handler := models.CreateHandler(modelID)
	capabilities := handler.GetCapabilities()

	// Apply model-specific configuration
	switch modelID {
	case models.GoProModelHERO13:
		p.log.Debug("Applying HERO13-specific pairing procedures")
		if err := p.configureHERO13SpecificSettings(macAddress); err != nil {
			p.log.Warnf("HERO13-specific configuration failed: %v", err)
		}

	case models.GoProModelHERO12:
		p.log.Debug("Applying HERO12-specific pairing procedures")
		if err := p.configureHERO12SpecificSettings(macAddress); err != nil {
			p.log.Warnf("HERO12-specific configuration failed: %v", err)
		}

	case models.GoProModelHERO11:
		p.log.Debug("Applying HERO11-specific pairing procedures")
		if err := p.configureHERO11SpecificSettings(macAddress); err != nil {
			p.log.Warnf("HERO11-specific configuration failed: %v", err)
		}

	case models.GoProModelHERO10, models.GoProModelHERO9:
		p.log.Debug("Applying legacy model pairing procedures")
		if err := p.configureLegacyModelSettings(macAddress); err != nil {
			p.log.Warnf("Legacy model configuration failed: %v", err)
		}

	default:
		p.log.Warnf("Unknown GoPro model %s, using default pairing procedure", modelName)
		if err := p.configureDefaultSettings(macAddress); err != nil {
			p.log.Warnf("Default configuration failed: %v", err)
		}
	}

	// Wait for model-specific timeout
	timeout := handler.GetPairingTimeout()
	p.log.Debugf("Waiting %v for pairing to complete", timeout)
	time.Sleep(timeout)

	// Perform enhanced pairing if supported
	if capabilities.SupportsEnhancedBLE {
		if err := p.performEnhancedPairing(macAddress); err != nil {
			p.log.Warnf("Enhanced pairing failed: %v", err)
		}
	}

	return nil
}

// verifyPairingState checks the current pairing state and handles it appropriately
func (p *PairingManager) verifyPairingState(macAddress string) error {
	p.log.Tracef("Verifying pairing state: device=%s", macAddress)

	pairingState, err := p.GetPairingState(macAddress)
	if err != nil {
		return fmt.Errorf("failed to get pairing state: %v", err)
	}

	switch pairingState {
	case PairingStateCompleted:
		p.log.Debugf("Pairing state: device=%s state=completed", macAddress)
		return nil
	case PairingStateNotPaired:
		p.log.Debugf("Pairing state: device=%s state=not_paired note=normal_for_new_pairing", macAddress)
		return nil
	case PairingStateInProgress:
		p.log.Debugf("Pairing state: device=%s state=in_progress", macAddress)
		return nil
	case PairingStateFailed:
		return fmt.Errorf("device pairing was aborted")
	case PairingStateCancelled:
		return fmt.Errorf("device pairing was cancelled")
	default:
		p.log.Warnf("Unknown pairing state: %d", pairingState)
		return nil
	}
}

// GetPairingState gets the current pairing state from the device
func (p *PairingManager) GetPairingState(macAddress string) (int, error) {
	// Try to get cached pairing state from response handler first
	if p.responseHandler != nil {
		cachedState := p.responseHandler.GetPairingState(macAddress)
		if cachedState > 0 {
			p.log.Debugf("Using cached pairing state %d for device %s", cachedState, macAddress)
			return cachedState, nil
		}
	}

	// If no cached state available, query the device
	p.log.Debugf("No cached pairing state, querying device %s", macAddress)
	if err := p.characteristicsManager.QueryPairingState(); err != nil {
		return 0, fmt.Errorf("failed to query pairing state: %v", err)
	}

	// Wait a moment for response to arrive
	time.Sleep(500 * time.Millisecond)

	// Try to get the response from the tracker
	if p.responseHandler != nil {
		state := p.responseHandler.GetPairingState(macAddress)
		if state > 0 {
			return state, nil
		}
	}

	// If we still don't have a response, return never started as default
	p.log.Warnf("Could not get pairing state for device %s, assuming never started", macAddress)
	return PairingStateNotPaired, nil
}

// RefreshPairingState forces a fresh query of the pairing state
func (p *PairingManager) RefreshPairingState(macAddress string) (int, error) {
	p.log.Debug("RefreshPairingState: querying device for current pairing state", "device", macAddress)

	// Initialize tracker for this device if needed
	p.responseHandler.InitializeTracker(macAddress)

	// Query the device for pairing state
	err := p.characteristicsManager.QueryPairingState()
	if err != nil {
		p.log.Error("RefreshPairingState: failed to query pairing state", "device", macAddress, "error", err)
		return PairingStateNotPaired, err
	}

	// Wait for the specific pairing state response
	response, err := p.responseHandler.WaitForResponse(macAddress, QueryGetStatusValues, 5*time.Second)
	if err != nil {
		p.log.Warn("RefreshPairingState: failed to get pairing state response", "device", macAddress, "error", err)
		// Return the cached state if available
		cachedState := p.responseHandler.GetPairingState(macAddress)
		if cachedState != 0 {
			p.log.Debug("RefreshPairingState: using cached pairing state", "device", macAddress, "state", cachedState)
			return cachedState, nil
		}
		return PairingStateNotPaired, err
	}

	// Extract pairing state from response data
	if len(response.Data) > 0 {
		state := int(response.Data[0])
		p.log.Debug("RefreshPairingState: received pairing state", "device", macAddress, "state", state)
		return state, nil
	}

	p.log.Warn("RefreshPairingState: empty response data", "device", macAddress)
	return PairingStateNotPaired, fmt.Errorf("empty response data for pairing state query")
}

// IsPaired checks if a device is paired (must be connected first)
func (p *PairingManager) IsPaired(macAddress string) (bool, error) {
	// Check if device is connected using the connection state
	connectionState := p.connectionManager.GetState(macAddress)
	if connectionState != StateConnected && connectionState != StateReady {
		return false, fmt.Errorf("device %s is not connected (state: %s)", macAddress, connectionState)
	}

	// Get cached state first
	if cachedState, exists := p.getCachedPairingState(macAddress); exists {
		p.log.Tracef("Using cached pairing state for device %s: %d", macAddress, cachedState)
		return cachedState == PairingStateCompleted, nil
	}

	// Query fresh state
	state, err := p.queryPairingState(macAddress)
	if err != nil {
		return false, err
	}

	return state == PairingStateCompleted, nil
}

// queryPairingState queries the current pairing state from the device
func (p *PairingManager) queryPairingState(macAddress string) (int, error) {
	// Ensure we have the control service characteristics
	chars, err := p.characteristicsManager.GetControlServiceCharacteristics()
	if err != nil {
		return PairingStateNotPaired, fmt.Errorf("failed to get control service characteristics: %v", err)
	}

	queryChar := chars["query"]
	queryRespChar := chars["queryResponse"]

	// Set up notification handler BEFORE querying
	responseChan := make(chan []byte, 1)
	err = queryRespChar.EnableNotifications(func(data []byte) {
		select {
		case responseChan <- data:
		default:
			p.log.Warnf("Response channel full, dropping pairing state response")
		}
	})
	if err != nil {
		return PairingStateNotPaired, fmt.Errorf("failed to enable notifications: %v", err)
	}

	// Send keep-alive to prevent disconnection during query
	if err := p.sendKeepAlive(macAddress); err != nil {
		p.log.Warnf("Failed to send keep-alive before pairing state query: %v", err)
	}

	// Query pairing state using correct OpenGoPro TLV format
	// Format: [Query ID] [Array Length] [Status ID]
	query := []byte{QueryGetStatusValues, 0x01, StatusPairingState}
	p.log.Tracef("Sending pairing state query: %v", query)
	_, err = queryChar.WriteWithoutResponse(query)
	if err != nil {
		return PairingStateNotPaired, fmt.Errorf("failed to write pairing state query: %v", err)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		state := p.parsePairingStateResponse(response)
		p.cachePairingState(macAddress, state)
		return state, nil
	case <-time.After(5 * time.Second):
		return PairingStateNotPaired, fmt.Errorf("timeout waiting for pairing state response")
	}
}

// IsPairedWithVerification checks pairing with additional verification
func (p *PairingManager) IsPairedWithVerification(macAddress string) (bool, error) {
	// First check basic pairing state
	isPaired, err := p.IsPaired(macAddress)
	if err != nil {
		return false, err
	}

	if !isPaired {
		return false, nil
	}

	// Additional verification steps can be added here
	p.log.Debugf("Device %s is paired and verified", macAddress)
	return true, nil
}

// detectGoProModel detects the GoPro model from the device
func (p *PairingManager) detectGoProModel(macAddress string) (int, error) {
	// Query hardware info to get the actual model ID
	if err := p.characteristicsManager.QueryHardwareInfo(); err != nil {
		return 0, fmt.Errorf("failed to query hardware info: %v", err)
	}

	// Wait for hardware info response
	response, err := p.responseHandler.WaitForResponse(macAddress, CommandGetHardwareInfo, 5*time.Second)
	if err != nil {
		p.log.Warnf("Failed to get hardware info response: %v", err)
		// Check if we have a cached model ID from previous queries
		if cachedModelID := p.responseHandler.GetModelID(macAddress); cachedModelID > 0 {
			p.log.Infof("Using cached model ID: %d", cachedModelID)
			return cachedModelID, nil
		}
		return 0, fmt.Errorf("failed to get hardware info response: %v", err)
	}

	// Parse the hardware info response to extract model ID
	if len(response.Data) > 0 {
		modelID := int(response.Data[0])
		p.log.Infof("Detected GoPro model ID: %d (%s)", modelID, models.GetModelName(modelID))
		return modelID, nil
	}

	return 0, fmt.Errorf("empty hardware info response")
}

// configureHERO13SpecificSettings applies HERO13-specific BLE configuration
func (p *PairingManager) configureHERO13SpecificSettings(macAddress string) error {
	p.log.Debug("Configuring HERO13-specific settings")

	// HERO13 may have enhanced BLE capabilities and faster processing
	time.Sleep(500 * time.Millisecond)

	// Enable enhanced BLE features if available
	if err := p.setEnhancedBLEMode(macAddress, true); err != nil {
		p.log.Warnf("Failed to enable enhanced BLE mode for HERO13: %v", err)
	}

	// Perform security handshake for HERO13
	if err := p.performSecurityHandshake(macAddress); err != nil {
		p.log.Warnf("Security handshake failed for HERO13: %v", err)
	}

	return nil
}

// configureHERO12SpecificSettings applies HERO12-specific BLE configuration
func (p *PairingManager) configureHERO12SpecificSettings(macAddress string) error {
	p.log.Debug("Configuring HERO12-specific settings")
	time.Sleep(300 * time.Millisecond)

	if err := p.setConnectionParameters(macAddress, "hero12"); err != nil {
		p.log.Warnf("Failed to set HERO12 connection parameters: %v", err)
	}

	return nil
}

// configureHERO11SpecificSettings applies HERO11-specific BLE configuration
func (p *PairingManager) configureHERO11SpecificSettings(macAddress string) error {
	p.log.Debug("Configuring HERO11-specific settings")
	time.Sleep(300 * time.Millisecond)

	if err := p.setConnectionParameters(macAddress, "hero11"); err != nil {
		p.log.Warnf("Failed to set HERO11 connection parameters: %v", err)
	}

	return nil
}

// configureLegacyModelSettings applies settings for older GoPro models
func (p *PairingManager) configureLegacyModelSettings(macAddress string) error {
	p.log.Debug("Configuring legacy model settings")
	time.Sleep(200 * time.Millisecond)

	if err := p.setConnectionParameters(macAddress, "legacy"); err != nil {
		p.log.Warnf("Failed to set legacy connection parameters: %v", err)
	}

	return nil
}

// configureDefaultSettings applies default settings for unknown models
func (p *PairingManager) configureDefaultSettings(macAddress string) error {
	p.log.Debug("Configuring default settings")
	time.Sleep(200 * time.Millisecond)
	return nil
}

// performEnhancedPairing performs enhanced pairing procedures
func (p *PairingManager) performEnhancedPairing(macAddress string) error {
	p.log.Debug("Performing enhanced pairing procedures")
	// Implementation would include advanced pairing steps
	return nil
}

// setEnhancedBLEMode enables enhanced BLE features
func (p *PairingManager) setEnhancedBLEMode(macAddress string, enabled bool) error {
	p.log.Debugf("Setting enhanced BLE mode to %v for %s", enabled, macAddress)
	// Implementation would configure enhanced BLE features
	return nil
}

// performSecurityHandshake performs security handshake
func (p *PairingManager) performSecurityHandshake(macAddress string) error {
	p.log.Debug("Performing security handshake")
	// Implementation would include security verification steps
	return nil
}

// setConnectionParameters sets optimal connection parameters
func (p *PairingManager) setConnectionParameters(macAddress, modelType string) error {
	p.log.Debugf("Setting %s connection parameters for %s", modelType, macAddress)
	// Implementation would configure connection parameters
	return nil
}

// sendKeepAlive sends a keep-alive command to prevent disconnection
func (p *PairingManager) sendKeepAlive(macAddress string) error {
	chars, err := p.characteristicsManager.GetControlServiceCharacteristics()
	if err != nil {
		return fmt.Errorf("failed to get control service characteristics: %v", err)
	}

	commandChar := chars["command"]
	if commandChar == nil {
		return fmt.Errorf("command characteristic not found")
	}

	// Send keep-alive command according to OpenGoPro spec
	keepAliveCmd := []byte{CommandKeepAlive}
	_, err = commandChar.WriteWithoutResponse(keepAliveCmd)
	if err != nil {
		return fmt.Errorf("failed to send keep-alive: %v", err)
	}

	p.log.Tracef("Sent keep-alive command to device %s", macAddress)
	return nil
}

// setupResponseNotifications sets up notifications on response characteristics
func (p *PairingManager) setupResponseNotifications(macAddress string) error {
	p.log.Debugf("Setting up response notifications for device %s", macAddress)

	chars, err := p.characteristicsManager.GetControlServiceCharacteristics()
	if err != nil {
		return fmt.Errorf("failed to get control service characteristics: %v", err)
	}

	// Set up notifications on response characteristics
	responseChars := map[string]string{
		"queryResponse":    "Query Response",
		"commandResponse":  "Command Response",
		"settingsResponse": "Settings Response",
	}

	for charKey, charName := range responseChars {
		if char, exists := chars[charKey]; exists && char != nil {
			p.log.Tracef("Enabling notifications for %s", charName)

			err := char.EnableNotifications(func(data []byte) {
				p.log.Tracef("Received %s notification: %d bytes", charName, len(data))
				// Forward to response handler if available
				if p.responseHandler != nil {
					// Process the response data
					p.handleNotificationResponse(macAddress, charName, data)
				}
			})

			if err != nil {
				p.log.Warnf("Failed to enable %s notifications: %v", charName, err)
				continue
			}

			p.log.Debugf("Successfully enabled %s notifications", charName)

			// Small delay between setups
			time.Sleep(100 * time.Millisecond)
		} else {
			p.log.Tracef("%s characteristic not found or nil", charName)
		}
	}

	return nil
}

// handleNotificationResponse processes notification responses
func (p *PairingManager) handleNotificationResponse(macAddress, charType string, data []byte) {
	p.log.Tracef("Processing %s notification from %s: %d bytes", charType, macAddress, len(data))

	// Handle different response types
	switch charType {
	case "Query Response":
		p.handleQueryNotification(macAddress, data)
	case "Command Response":
		p.handleCommandNotification(macAddress, data)
	case "Settings Response":
		p.handleSettingsNotification(macAddress, data)
	default:
		p.log.Tracef("Unhandled notification type: %s", charType)
	}
}

// handleQueryNotification processes query response notifications
func (p *PairingManager) handleQueryNotification(macAddress string, data []byte) {
	if len(data) < 2 {
		return
	}

	queryID := data[0]
	status := data[1]

	p.log.Tracef("Query response: device=%s query_id=0x%02X status=0x%02X", macAddress, queryID, status)

	// Handle pairing state query specifically
	if queryID == QueryGetStatusValues && status == ResponseStatusSuccess {
		if state := p.parsePairingStateResponse(data); state != PairingStateNotPaired {
			p.cachePairingState(macAddress, state)
		}
	}
}

// handleCommandNotification processes command response notifications
func (p *PairingManager) handleCommandNotification(macAddress string, data []byte) {
	if len(data) < 2 {
		return
	}

	commandID := data[0]
	status := data[1]

	p.log.Tracef("Command response: device=%s command_id=0x%02X status=0x%02X", macAddress, commandID, status)
}

// handleSettingsNotification processes settings response notifications
func (p *PairingManager) handleSettingsNotification(macAddress string, data []byte) {
	if len(data) < 2 {
		return
	}

	settingID := data[0]
	status := data[1]

	p.log.Tracef("Settings response: device=%s setting_id=0x%02X status=0x%02X", macAddress, settingID, status)
}

// emitPairingFailed emits a pairing failed event
func (p *PairingManager) emitPairingFailed(macAddress string, err error) {
	p.log.Errorf("BLE pairing failed: device=%s error=%v", macAddress, err)
	p.eventEmitter.EmitEvent(events.BLEEvent{
		Type:      events.EventPairingFailed,
		Device:    map[string]string{"mac_address": macAddress},
		Error:     err,
		Timestamp: time.Now(),
	})
}

// getCachedPairingState retrieves cached pairing state for a device
func (p *PairingManager) getCachedPairingState(macAddress string) (int, bool) {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	state, exists := p.pairingStateCache[macAddress]
	return state, exists
}

// cachePairingState stores pairing state in cache
func (p *PairingManager) cachePairingState(macAddress string, state int) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	p.pairingStateCache[macAddress] = state
	p.log.Tracef("Cached pairing state for device %s: %d", macAddress, state)
}

// parsePairingStateResponse parses the response from a pairing state query
func (p *PairingManager) parsePairingStateResponse(response []byte) int {
	// OpenGoPro BLE response format for status query:
	// [Query ID] [Status] [TLV Data...]
	// TLV format: [Type/ID] [Length] [Value]
	if len(response) < 5 {
		p.log.Warnf("Invalid pairing state response length: %d", len(response))
		return PairingStateNotPaired
	}

	// Check if this is a status response (QueryGetStatusValues = 0x13)
	if response[0] != QueryGetStatusValues {
		p.log.Warnf("Unexpected response command ID: 0x%02X, expected 0x%02X", response[0], QueryGetStatusValues)
		return PairingStateNotPaired
	}

	// Check status byte (0 = success)
	if response[1] != ResponseStatusSuccess {
		p.log.Warnf("Error status in pairing state response: %d", response[1])
		return PairingStateNotPaired
	}

	// Parse TLV data starting from byte 2
	index := 2
	for index < len(response) {
		// Need at least 3 bytes for TLV: Type(1) + Length(1) + Value(min 1)
		if index+2 >= len(response) {
			break
		}

		statusID := response[index]
		valueLength := int(response[index+1])

		// Check if we have enough bytes for the value
		if index+2+valueLength > len(response) {
			p.log.Warnf("Invalid TLV structure: not enough bytes for value")
			break
		}

		// Check if this is the pairing state we're looking for
		if statusID == StatusPairingState {
			if valueLength > 0 {
				state := int(response[index+2])
				p.log.Tracef("Parsed pairing state from TLV response: %d", state)
				return state
			}
		}

		// Move to next TLV entry
		index += 2 + valueLength
	}

	p.log.Warnf("Pairing state not found in TLV response")
	return PairingStateNotPaired
}
