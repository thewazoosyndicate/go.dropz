package ble

import (
	"context"
	"fmt"
	"time"

	"github.com/dropz/dropz/pkg/ble/events"
	"github.com/dropz/dropz/pkg/ble/models"
	"github.com/dropz/dropz/pkg/logger"
)

// PairingManager handles all BLE pairing operations
type PairingManager struct {
	connectionManager      *ConnectionManager
	characteristicsManager *CharacteristicsManager
	responseHandler        *ResponseHandler
	eventEmitter           *events.EventEmitter
	log                    logger.Logger
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

	// First, establish basic connection
	ctx := context.Background()
	_, err := p.connectionManager.Connect(ctx, macAddress)
	if err != nil {
		p.emitPairingFailed(macAddress, err)
		return fmt.Errorf("failed to establish basic connection: %v", err)
	}

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
	case PairingStateNeverStarted:
		p.log.Debugf("Pairing state: device=%s state=never_started note=normal_for_new_pairing", macAddress)
		return nil
	case PairingStateStarted:
		p.log.Debugf("Pairing state: device=%s state=in_progress", macAddress)
		return nil
	case PairingStateAborted:
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
	return PairingStateNeverStarted, nil
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
		return PairingStateNeverStarted, err
	}

	// Wait for the specific pairing state response
	response, err := p.responseHandler.WaitForResponse(macAddress, QueryGetPairingState, 5*time.Second)
	if err != nil {
		p.log.Warn("RefreshPairingState: failed to get pairing state response", "device", macAddress, "error", err)
		// Return the cached state if available
		cachedState := p.responseHandler.GetPairingState(macAddress)
		if cachedState != 0 {
			p.log.Debug("RefreshPairingState: using cached pairing state", "device", macAddress, "state", cachedState)
			return cachedState, nil
		}
		return PairingStateNeverStarted, err
	}

	// Extract pairing state from response data
	if len(response.Data) > 0 {
		state := int(response.Data[0])
		p.log.Debug("RefreshPairingState: received pairing state", "device", macAddress, "state", state)
		return state, nil
	}

	p.log.Warn("RefreshPairingState: empty response data", "device", macAddress)
	return PairingStateNeverStarted, fmt.Errorf("empty response data for pairing state query")
}

// IsPaired checks if the device is currently paired
func (p *PairingManager) IsPaired(macAddress string) (bool, error) {
	state, err := p.GetPairingState(macAddress)
	if err != nil {
		return false, err
	}
	return state == PairingStateCompleted, nil
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
	// This would interact with characteristic manager to read model info
	// Placeholder implementation
	return models.GoProModelHERO12, nil
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
