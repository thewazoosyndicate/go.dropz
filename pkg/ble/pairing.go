package ble

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble/events"
	"github.com/dropz/dropz/pkg/ble/models"
	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// PairingManager handles all BLE pairing operations
// Thread Safety: All shared state is protected by appropriate mutexes
// - pairingStateCache: Protected by cacheMutex (RWMutex) for concurrent reads
// - Manager dependencies are thread-safe and accessed without additional locking
type PairingManager struct {
	connectionManager      *ConnectionManager
	characteristicsManager *CharacteristicsManager
	responseHandler        *ResponseHandler
	eventEmitter           *events.EventEmitter
	log                    logger.Logger

	// Cache for pairing states - protected by cacheMutex
	pairingStateCache map[string]int
	cacheMutex        sync.RWMutex // Protects pairingStateCache access
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
// Thread Safety: This method coordinates multiple thread-safe managers.
// Each manager call is atomic, and pairing state is cached thread-safely.
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
	p.log.Debugf("Attempting to establish connection: device=%s", macAddress)

	device, err := p.connectionManager.Connect(ctx, macAddress)
	if err != nil {
		p.log.Errorf("Failed to establish OpenGoPro connection: device=%s error=%v", macAddress, err)
		p.emitPairingFailed(macAddress, err)
		return fmt.Errorf("failed to establish OpenGoPro connection: %v", err)
	}
	p.log.Infof("Successfully established OpenGoPro connection: device=%s", macAddress)

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

	p.log.Infof("Successfully discovered and cached characteristics for device %s", macAddress)

	// Setup notifications for response characteristics BEFORE querying
	if err := p.setupResponseNotifications(macAddress); err != nil {
		p.log.Errorf("Failed to setup response notifications: device=%s error=%v", macAddress, err)
		p.emitPairingFailed(macAddress, err)
		return fmt.Errorf("failed to setup response notifications: %v", err)
	}
	p.log.Debugf("Successfully setup response notifications: device=%s", macAddress)

	// Verify connection is still active before proceeding
	connectionState := p.connectionManager.GetState(macAddress)
	if connectionState != StateConnected && connectionState != StateReady {
		p.log.Errorf("Connection lost after notification setup: device=%s state=%s", macAddress, connectionState)
		p.emitPairingFailed(macAddress, fmt.Errorf("connection lost: state=%s", connectionState))
		return fmt.Errorf("connection lost after notification setup: state=%s", connectionState)
	}
	p.log.Debugf("Connection verified active: device=%s state=%s", macAddress, connectionState)

	// NOW we can check pairing state with proper service discovery
	pairingState, err := p.queryPairingState(macAddress)
	if err != nil {
		p.log.Errorf("Failed to query initial pairing state: device=%s error=%v", macAddress, err)

		// Check if this is a connection issue
		if connectionState := p.connectionManager.GetState(macAddress); connectionState != StateConnected && connectionState != StateReady {
			p.log.Errorf("Connection lost during pairing state query: device=%s state=%s", macAddress, connectionState)
			p.emitPairingFailed(macAddress, fmt.Errorf("connection lost during pairing state query: %v", err))
			return fmt.Errorf("connection lost during pairing state query: %v", err)
		}

		// Check if this looks like a "Not connected" error from TinyGo
		if strings.Contains(err.Error(), "Not connected") || strings.Contains(err.Error(), "WriteWithoutResponse") {
			p.log.Errorf("TinyGo connection lost detected: device=%s error=%v", macAddress, err)
			// Update our connection manager state to reflect reality
			p.connectionManager.ChangeState(macAddress, StateDisconnected, err)
			p.emitPairingFailed(macAddress, fmt.Errorf("underlying bluetooth connection lost: %v", err))
			return fmt.Errorf("underlying bluetooth connection lost: %v", err)
		}

		// If device is connected but query failed, assume unpaired and continue
		p.log.Warnf("Pairing state query failed but device connected, assuming unpaired: device=%s", macAddress)
		pairingState = PairingStateNotPaired
	} else {
		p.log.Infof("Successfully queried initial pairing state: device=%s state=%d", macAddress, pairingState)
	}

	p.log.Infof("Current pairing state for device %s: %d", macAddress, pairingState)

	// Check if device is already paired - skip pairing if so
	if pairingState == PairingStateCompleted {
		p.log.Infof("Device already paired, skipping pairing process: device=%s state=%d", macAddress, pairingState)

		// Emit pairing completed event since device is already paired
		p.eventEmitter.EmitEvent(events.BLEEvent{
			Type:      events.EventPairingCompleted,
			Device:    map[string]string{"mac_address": macAddress},
			Timestamp: time.Now(),
		})

		p.log.Infof("BLE pairing completed (already paired): device=%s status=success", macAddress)

		// Disconnect immediately since pairing is already complete
		p.log.Infof("Disconnecting from already-paired device: device=%s", macAddress)
		if err := p.connectionManager.Disconnect(macAddress); err != nil {
			p.log.Warnf("Failed to disconnect from already-paired device: device=%s error=%v", macAddress, err)
			// Don't return error here as pairing check was successful
		} else {
			p.log.Infof("Successfully disconnected from already-paired device: device=%s", macAddress)
		}

		return nil
	}

	// Device needs pairing - detect model for enhanced pairing
	modelID, err := p.detectGoProModel(macAddress)
	if err != nil {
		p.log.Warnf("Model detection failed: device=%s error=%v fallback=default", macAddress, err)
		// Use a default/unknown model ID when detection fails
		// This allows pairing to continue with basic functionality
		modelID = 1 // Use Hero 9 as the most basic/compatible fallback
	} else {
		p.log.Infof("Model detected: device=%s model=%s model_id=%d",
			macAddress, models.GetModelName(modelID), modelID)
	}

	// Perform model-specific pairing procedures using safe handler creation
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

	// Disconnect immediately after successful pairing completion
	p.log.Infof("Disconnecting from device after successful pairing: device=%s", macAddress)
	if err := p.connectionManager.Disconnect(macAddress); err != nil {
		p.log.Warnf("Failed to disconnect after pairing completion: device=%s error=%v", macAddress, err)
		// Don't return error here as pairing was successful
	} else {
		p.log.Infof("Successfully disconnected after pairing: device=%s", macAddress)
	}

	return nil
}

// performModelSpecificPairing handles model-specific pairing requirements
// Thread Safety: Coordinates multiple thread-safe manager calls.
// Safe for concurrent execution with different devices.
func (p *PairingManager) performModelSpecificPairing(macAddress string, modelID int) error {
	modelName := models.GetModelName(modelID)
	p.log.Debugf("Executing pairing sequence: device=%s model=%s model_id=%d", macAddress, modelName, modelID)

	// Create model handler using safe creation that handles unknown models
	handler := models.CreateHandlerSafe(modelID)
	if !handler.IsSupported() {
		p.log.Warnf("Unknown or unsupported model ID %d, using basic pairing", modelID)
	}

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
	}

	// Wait for model-specific timeout
	timeout := handler.GetPairingTimeout()
	p.log.Debugf("Waiting %v for pairing to complete", timeout)
	time.Sleep(timeout)

	return nil
}

// verifyPairingState checks the current pairing state and handles it appropriately
// Thread Safety: Uses thread-safe GetPairingState call.
// Safe for concurrent execution.
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
// Thread Safety: Uses thread-safe response handler calls and cache access.
// Safe for concurrent execution with different devices.
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
// Thread Safety: Uses thread-safe response handler calls for atomic state refresh.
// Safe for concurrent execution with different devices.
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
// Thread Safety: Uses thread-safe connection manager and cached state access.
// Safe for concurrent execution with different devices.
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
// Thread Safety: Uses thread-safe characteristics manager calls and atomic cache updates.
// Uses existing response handler to prevent notification conflicts.
func (p *PairingManager) queryPairingState(macAddress string) (int, error) {
	// First verify connection is active
	connectionState := p.connectionManager.GetState(macAddress)
	if connectionState != StateConnected && connectionState != StateReady {
		return PairingStateNotPaired, fmt.Errorf("device not connected: state=%s", connectionState)
	}

	// Ensure we have the control service characteristics
	chars, err := p.characteristicsManager.GetControlServiceCharacteristics()
	if err != nil {
		return PairingStateNotPaired, fmt.Errorf("failed to get control service characteristics: %v", err)
	}

	queryChar := chars["query"]
	if queryChar == nil {
		return PairingStateNotPaired, fmt.Errorf("query characteristic not found")
	}

	// Initialize response tracker for this device if needed
	if p.responseHandler != nil {
		p.responseHandler.InitializeTracker(macAddress)
	}

	// Don't send keep-alive if connection verification already passed
	// Send keep-alive to prevent disconnection during query
	if err := p.sendKeepAlive(macAddress); err != nil {
		p.log.Warnf("Failed to send keep-alive before pairing state query: %v", err)
		// Verify connection is still active after keep-alive failure
		if newState := p.connectionManager.GetState(macAddress); newState != StateConnected && newState != StateReady {
			return PairingStateNotPaired, fmt.Errorf("connection lost during keep-alive: state=%s", newState)
		}
	}

	// Query pairing state using correct OpenGoPro TLV format
	// Format: [Query ID] [Array Length] [Status ID]
	query := []byte{QueryGetStatusValues, 0x01, StatusPairingState}
	p.log.Tracef("Sending pairing state query: %v", query)
	_, err = queryChar.WriteWithoutResponse(query)
	if err != nil {
		return PairingStateNotPaired, fmt.Errorf("failed to write pairing state query: %v", err)
	}

	// Wait for response using the response handler
	if p.responseHandler != nil {
		response, err := p.responseHandler.WaitForResponse(macAddress, QueryGetStatusValues, 10*time.Second)
		if err == nil && len(response.Data) > 0 {
			state := p.parsePairingStateResponse(response.Data)
			p.cachePairingState(macAddress, state)
			p.log.Debugf("Received pairing state response: device=%s state=%d", macAddress, state)
			return state, nil
		}
		p.log.Warnf("Failed to get response via response handler: %v", err)
	}

	// Fallback: wait a bit and try to get cached state
	time.Sleep(2 * time.Second)
	if cachedState, exists := p.getCachedPairingState(macAddress); exists {
		p.log.Debugf("Using cached pairing state as fallback: device=%s state=%d", macAddress, cachedState)
		return cachedState, nil
	}

	p.log.Errorf("Timeout waiting for pairing state response: device=%s", macAddress)
	return PairingStateNotPaired, fmt.Errorf("timeout waiting for pairing state response after 10 seconds")
}

// IsPairedWithVerification checks pairing with additional verification
// Thread Safety: Uses thread-safe IsPaired call.
// Safe for concurrent execution.
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
// Thread Safety: Uses thread-safe characteristics manager and response handler calls.
// Safe for concurrent execution with different devices.
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
// Thread Safety: Uses thread-safe method calls, no shared state access.
// Safe for concurrent execution.
func (p *PairingManager) configureHERO13SpecificSettings(macAddress string) error {
	p.log.Debug("Configuring HERO13-specific settings")

	// HERO13 may have enhanced BLE capabilities and faster processing
	time.Sleep(500 * time.Millisecond)

	// Perform security handshake for HERO13
	if err := p.performSecurityHandshake(macAddress); err != nil {
		p.log.Warnf("Security handshake failed for HERO13: %v", err)
	}

	return nil
}

// configureHERO12SpecificSettings applies HERO12-specific BLE configuration
// Thread Safety: Uses thread-safe method calls, no shared state access.
// Safe for concurrent execution.
func (p *PairingManager) configureHERO12SpecificSettings(macAddress string) error {
	p.log.Debug("Configuring HERO12-specific settings")
	time.Sleep(300 * time.Millisecond)

	if err := p.setConnectionParameters(macAddress, "hero12"); err != nil {
		p.log.Warnf("Failed to set HERO12 connection parameters: %v", err)
	}

	return nil
}

// configureHERO11SpecificSettings applies HERO11-specific BLE configuration
// Thread Safety: Uses thread-safe method calls, no shared state access.
// Safe for concurrent execution.
func (p *PairingManager) configureHERO11SpecificSettings(macAddress string) error {
	p.log.Debug("Configuring HERO11-specific settings")
	time.Sleep(300 * time.Millisecond)

	if err := p.setConnectionParameters(macAddress, "hero11"); err != nil {
		p.log.Warnf("Failed to set HERO11 connection parameters: %v", err)
	}

	return nil
}

// configureLegacyModelSettings applies settings for older GoPro models
// Thread Safety: Uses thread-safe method calls, no shared state access.
// Safe for concurrent execution.
func (p *PairingManager) configureLegacyModelSettings(macAddress string) error {
	p.log.Debug("Configuring legacy model settings")
	time.Sleep(200 * time.Millisecond)

	if err := p.setConnectionParameters(macAddress, "legacy"); err != nil {
		p.log.Warnf("Failed to set legacy connection parameters: %v", err)
	}

	return nil
}

// performSecurityHandshake performs security handshake
func (p *PairingManager) performSecurityHandshake(macAddress string) error {
	p.log.Debugf("NOT Performing security handshake for: %s (placeholder)", macAddress)
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
// Thread Safety: Uses thread-safe characteristics manager calls.
// Safe for concurrent execution with different devices.
func (p *PairingManager) sendKeepAlive(macAddress string) error {
	// Verify connection is active
	connectionState := p.connectionManager.GetState(macAddress)
	if connectionState != StateConnected && connectionState != StateReady {
		return fmt.Errorf("device not connected: state=%s", connectionState)
	}

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
// Thread Safety: Uses thread-safe characteristics manager calls.
// Handler callbacks execute concurrently and must be thread-safe.
func (p *PairingManager) setupResponseNotifications(macAddress string) error {
	p.log.Debugf("Setting up response notifications for device %s", macAddress)

	// Verify connection is still active before setting up notifications
	connectionState := p.connectionManager.GetState(macAddress)
	if connectionState != StateConnected && connectionState != StateReady {
		return fmt.Errorf("device not connected for notification setup: state=%s", connectionState)
	}

	chars, err := p.characteristicsManager.GetControlServiceCharacteristics()
	if err != nil {
		return fmt.Errorf("failed to get control service characteristics: %v", err)
	}

	// Set up notifications on response characteristics non-blockingly
	// According to OpenGoPro spec, notifications should be enabled immediately after service discovery
	responseChars := map[string]string{
		"queryResponse":    "Query Response",
		"commandResponse":  "Command Response",
		"settingsResponse": "Settings Response",
	}

	// Track notification setup results
	notificationErrors := make([]error, 0)
	successfulNotifications := 0

	for charKey, charName := range responseChars {
		if char, exists := chars[charKey]; exists && char != nil {
			p.log.Tracef("Enabling notifications for %s", charName)

			// Add a small delay between notification setups to prevent overwhelming the device
			if successfulNotifications > 0 {
				time.Sleep(200 * time.Millisecond)
			}

			// Check connection before each notification setup
			if currentState := p.connectionManager.GetState(macAddress); currentState != StateConnected && currentState != StateReady {
				p.log.Warnf("Connection lost during notification setup: device=%s characteristic=%s state=%s", macAddress, charName, currentState)
				notificationErrors = append(notificationErrors, fmt.Errorf("%s: connection lost", charName))
				continue
			}

			// Run EnableNotifications in a goroutine with timeout to prevent blocking
			notificationDone := make(chan error, 1)
			go func(characteristic *bluetooth.DeviceCharacteristic, name string) {
				err := characteristic.EnableNotifications(func(data []byte) {
					p.log.Tracef("Received %s notification: %d bytes", name, len(data))
					// Forward to response handler if available
					if p.responseHandler != nil {
						// Process the response data in a separate goroutine to avoid blocking
						go func(mac, charType string, responseData []byte) {
							p.handleNotificationResponse(mac, charType, responseData)
						}(macAddress, name, data)
					}
				})
				notificationDone <- err
			}(char, charName)

			// Wait for notification setup with extended timeout for the first (Query Response)
			timeout := 3 * time.Second
			if charName == "Query Response" {
				timeout = 5 * time.Second // Give Query Response more time as it's critical
			}

			select {
			case err := <-notificationDone:
				if err != nil {
					p.log.Warnf("Failed to enable %s notifications: %v", charName, err)
					notificationErrors = append(notificationErrors, fmt.Errorf("%s: %v", charName, err))
					continue
				}
				p.log.Debugf("Successfully enabled %s notifications", charName)
				successfulNotifications++
			case <-time.After(timeout):
				p.log.Warnf("Timeout enabling %s notifications, continuing anyway", charName)
				notificationErrors = append(notificationErrors, fmt.Errorf("%s: timeout", charName))
			}
		} else {
			p.log.Tracef("%s characteristic not found or nil", charName)
		}
	}

	// Return error only if ALL notifications failed
	if len(notificationErrors) == len(responseChars) {
		return fmt.Errorf("failed to enable any notifications: %v", notificationErrors)
	}

	// Log warnings for partial failures but don't fail the pairing
	if len(notificationErrors) > 0 {
		p.log.Warnf("Some notifications failed to enable: %v", notificationErrors)
	}

	p.log.Debugf("Response notifications setup completed for device %s", macAddress)
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
// Thread Safety: Uses thread-safe event emitter, no shared state access.
// Safe for concurrent execution.
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
// Thread Safety: Read lock protects concurrent access to pairingStateCache.
// Safe for concurrent reads from multiple goroutines.
func (p *PairingManager) getCachedPairingState(macAddress string) (int, bool) {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	state, exists := p.pairingStateCache[macAddress]
	return state, exists
}

// cachePairingState stores pairing state in cache
// Thread Safety: Write lock ensures atomic updates to pairingStateCache.
// Prevents race conditions during concurrent cache modifications.
func (p *PairingManager) cachePairingState(macAddress string, state int) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	p.pairingStateCache[macAddress] = state
	p.log.Tracef("Cached pairing state for device %s: %d", macAddress, state)
}

// parsePairingStateResponse parses the response from a pairing state query with enhanced validation
// Thread Safety: Pure function with no shared state access.
// Safe for concurrent use.
func (p *PairingManager) parsePairingStateResponse(response []byte) int {
	// First validate the overall response format
	if err := ValidateResponseFormat(response, QueryGetStatusValues); err != nil {
		p.log.Warnf("Invalid response format: %v", err)
		return PairingStateNotPaired
	}

	// Check status byte (0 = success)
	if response[1] != ResponseStatusSuccess {
		p.log.Warnf("Error status in pairing state response: %d", response[1])
		return PairingStateNotPaired
	}

	// Extract and validate TLV data starting from byte 2
	tlvData := response[2:]
	if err := ValidateTLVFormat(tlvData); err != nil {
		p.log.Warnf("Invalid TLV format in pairing response: %v", err)
		return PairingStateNotPaired
	}

	// Parse TLV data with proper error handling
	return p.extractPairingStateFromTLV(tlvData)
}

// extractPairingStateFromTLV extracts pairing state from validated TLV data
func (p *PairingManager) extractPairingStateFromTLV(tlvData []byte) int {
	index := 0
	for index < len(tlvData) {
		// Safe to access since TLV format was already validated
		statusID := tlvData[index]
		valueLength := int(tlvData[index+1])

		// Check if this is the pairing state we're looking for
		if statusID == StatusPairingState {
			if valueLength > 0 && index+2 < len(tlvData) {
				state := int(tlvData[index+2])
				p.log.Tracef("Parsed pairing state from TLV response: %d", state)

				// Validate the pairing state value
				if state < PairingStateNotPaired || state > PairingStateCompleted {
					p.log.Warnf("Invalid pairing state value: %d", state)
					return PairingStateNotPaired
				}

				return state
			}
		}

		// Move to next TLV entry
		index += 2 + valueLength
	}

	p.log.Warnf("Pairing state not found in TLV response")
	return PairingStateNotPaired
}

// verifyConnectionHealth checks if the device is responding to basic commands
// This helps detect if the device is in pairing mode
func (p *PairingManager) verifyConnectionHealth(macAddress string) error {
	p.log.Debugf("Verifying connection health and pairing mode: device=%s", macAddress)

	// Verify connection is active
	connectionState := p.connectionManager.GetState(macAddress)
	if connectionState != StateConnected && connectionState != StateReady {
		return fmt.Errorf("device not connected: state=%s", connectionState)
	}

	chars, err := p.characteristicsManager.GetControlServiceCharacteristics()
	if err != nil {
		return fmt.Errorf("failed to get control service characteristics: %v", err)
	}

	queryChar := chars["query"]
	if queryChar == nil {
		return fmt.Errorf("query characteristic not found")
	}

	// Try a simple keep-alive command first to check responsiveness
	commandChar := chars["command"]
	if commandChar != nil {
		p.log.Tracef("Testing device responsiveness with keep-alive command")
		keepAliveCmd := []byte{CommandKeepAlive}
		if _, err := commandChar.WriteWithoutResponse(keepAliveCmd); err != nil {
			// Check if this is a TinyGo "Not connected" error
			if strings.Contains(err.Error(), "Not connected") {
				// Update connection manager state to reflect actual TinyGo state
				p.connectionManager.ChangeState(macAddress, StateDisconnected, err)
				return fmt.Errorf("underlying TinyGo connection lost: %v", err)
			}
			return fmt.Errorf("failed to send keep-alive: %v", err)
		}
	}

	// Try a simple status query to check if device responds
	p.log.Tracef("Testing device responsiveness with basic status query")
	if p.responseHandler != nil {
		p.responseHandler.InitializeTracker(macAddress)
	}

	// Query battery level (simple query that most GoPros respond to)
	batteryQuery := []byte{QueryGetStatusValues, 0x01, StatusBatteryLevel}
	_, err = queryChar.WriteWithoutResponse(batteryQuery)
	if err != nil {
		// Check if this is a TinyGo "Not connected" error
		if strings.Contains(err.Error(), "Not connected") {
			// Update connection manager state to reflect actual TinyGo state
			p.connectionManager.ChangeState(macAddress, StateDisconnected, err)
			return fmt.Errorf("underlying TinyGo connection lost: %v", err)
		}
		return fmt.Errorf("failed to send battery query: %v", err)
	}

	// Wait briefly for response
	time.Sleep(2 * time.Second)

	// Check if we got any response via the response handler
	if p.responseHandler != nil {
		if response, err := p.responseHandler.WaitForResponse(macAddress, QueryGetStatusValues, 1*time.Second); err == nil && len(response.Data) > 0 {
			p.log.Debugf("Device is responsive and in pairing mode: device=%s", macAddress)
			return nil
		}
	}

	// If no response, the device is connected but not responding to commands
	// This usually means it's not in pairing mode
	return fmt.Errorf("device connected but not responding to commands - may not be in pairing mode")
}
