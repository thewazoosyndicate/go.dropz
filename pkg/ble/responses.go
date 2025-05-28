package ble

import (
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble/events"
	"github.com/dropz/dropz/pkg/logger"
)

// Timeout constants for different BLE operations based on OpenGoPro analysis
const (
	// DefaultTimeout for BLE operations
	DefaultTimeout = 8 * time.Second

	// Specialized timeouts for different operation types
	PairingTimeout      = 10 * time.Second // Pairing operations need reasonable time
	HardwareInfoTimeout = 8 * time.Second  // Hardware info queries can be slow
	QuickTimeout        = 5 * time.Second  // For fast operations like keep-alive
	StatusTimeout       = 6 * time.Second  // For status queries
)

// ResponseHandler manages response tracking and notification handling with request correlation
// Thread Safety: All operations are protected by appropriate mutexes
// - responseTracker map: Protected by RWMutex for concurrent device access
// - Individual ResponseTracker instances: Protected by their own RWMutex
// - Cleanup operations: Coordinated with proper locking hierarchy
type ResponseHandler struct {
	responseTracker map[string]*ResponseTracker // MAC address -> response tracker (protected by mutex)
	mutex           sync.RWMutex                // Protects responseTracker map access
	eventEmitter    *events.EventEmitter
	log             logger.Logger
	cleanupTicker   *time.Ticker
	stopCleanup     chan struct{}
}

// NewResponseHandler creates a new response handler with request correlation
// Thread Safety: Initializes thread-safe data structures and starts cleanup goroutine.
// The cleanup goroutine handles concurrent access safely.
func NewResponseHandler(eventEmitter *events.EventEmitter, log logger.Logger) *ResponseHandler {
	rh := &ResponseHandler{
		responseTracker: make(map[string]*ResponseTracker),
		eventEmitter:    eventEmitter,
		log:             log,
		cleanupTicker:   time.NewTicker(1 * time.Second), // Cleanup every second
		stopCleanup:     make(chan struct{}),
	}

	// Start cleanup goroutine for timed-out requests
	go rh.cleanupTimedOutRequests()

	return rh
}

// Stop stops the response handler and cleans up resources
// Thread Safety: Safely stops cleanup goroutine and prevents race conditions.
// Should be called once during shutdown.
func (rh *ResponseHandler) Stop() {
	close(rh.stopCleanup)
	if rh.cleanupTicker != nil {
		rh.cleanupTicker.Stop()
	}
}

// cleanupTimedOutRequests periodically cleans up timed-out requests
// Thread Safety: Goroutine safely handles cleanup with proper locking hierarchy.
// Prevents deadlocks by acquiring tracker locks after main mutex.
func (rh *ResponseHandler) cleanupTimedOutRequests() {
	for {
		select {
		case <-rh.cleanupTicker.C:
			rh.cleanupExpiredRequests()
		case <-rh.stopCleanup:
			return
		}
	}
}

// cleanupExpiredRequests removes requests that have exceeded their timeout
// Thread Safety: Write lock protects responseTracker map, individual tracker locks
// prevent race conditions during request cleanup. Lock hierarchy: main mutex -> tracker mutex.
func (rh *ResponseHandler) cleanupExpiredRequests() {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	now := time.Now()
	for _, tracker := range rh.responseTracker {
		tracker.mutex.Lock()
		for cmdID, req := range tracker.pendingRequests {
			if now.Sub(req.StartTime) > req.Timeout {
				// Close the response channel to signal timeout
				close(req.ResponseChan)
				delete(tracker.pendingRequests, cmdID)
				delete(tracker.fragmentBuffer, cmdID)
				rh.log.Debug("Cleaned up timed-out request", "command_id", cmdID, "timeout", req.Timeout)
			}
		}
		tracker.mutex.Unlock()
	}
}

// InitializeTracker initializes response tracker for a device with request correlation
// Thread Safety: Write lock ensures atomic creation/access of responseTracker map.
// Safe for concurrent initialization of different devices.
func (rh *ResponseHandler) InitializeTracker(macAddress string) {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if rh.responseTracker[macAddress] == nil {
		rh.responseTracker[macAddress] = &ResponseTracker{
			responseChannel: make(chan *QueryResponseData, 10), // Legacy channel for backward compatibility
			pendingRequests: make(map[byte]*PendingRequest),
			fragmentBuffer:  make(map[byte]*FragmentedResponse),
		}
		rh.log.Trace("Response tracker initialized", "mac_address", macAddress, "buffer_size", 10)
	} else {
		rh.log.Trace("Response tracker already exists", "mac_address", macAddress)
	}
}

// GetTracker gets the response tracker for a device
// Thread Safety: Read lock allows concurrent access to responseTracker map.
// Returns tracker which has its own thread-safe operations.
func (rh *ResponseHandler) GetTracker(macAddress string) *ResponseTracker {
	rh.mutex.RLock()
	defer rh.mutex.RUnlock()
	return rh.responseTracker[macAddress]
}

// CleanupTracker removes the response tracker for a device and cancels pending requests
// Thread Safety: Write lock protects responseTracker map, tracker lock ensures
// atomic cleanup of pending requests. Prevents race conditions during device removal.
func (rh *ResponseHandler) CleanupTracker(macAddress string) {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if tracker, exists := rh.responseTracker[macAddress]; exists {
		tracker.mutex.Lock()
		// Cancel all pending requests
		for cmdID, req := range tracker.pendingRequests {
			close(req.ResponseChan)
			delete(tracker.pendingRequests, cmdID)
		}
		// Clear fragment buffers
		tracker.fragmentBuffer = make(map[byte]*FragmentedResponse)
		tracker.mutex.Unlock()

		close(tracker.responseChannel)
		delete(rh.responseTracker, macAddress)
		rh.log.Info("Response tracker cleaned up", "mac_address", macAddress, "tracker_count", len(rh.responseTracker))
	} else {
		rh.log.Trace("Response tracker not found for cleanup", "mac_address", macAddress)
	}
}

// CreateQueryResponseHandler creates a notification handler for query responses with correlation
// Thread Safety: Returns a handler function that safely processes concurrent notifications.
// Handler uses thread-safe tracker operations and prevents race conditions in response processing.
func (rh *ResponseHandler) CreateQueryResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Trace("Received query response", "mac_address", macAddress, "response_bytes", buf, "length", len(buf))

		// First try to process as fragmented response
		responseData := rh.processFragmentedResponse(macAddress, buf)
		if responseData == nil {
			// If not fragmented or incomplete fragment, try simple parsing
			if len(buf) >= 3 {
				// Basic response structure: [Query ID] [Status] [Data...]
				queryID := buf[0]
				status := buf[1]
				data := buf[2:]

				responseData = &QueryResponseData{
					QueryID:      queryID,
					Status:       status,
					Data:         data,
					ResponseTime: time.Now(),
					IsFragmented: false,
				}
			} else {
				rh.log.Error("Invalid query response format", "mac_address", macAddress, "response_length", len(buf), "expected_min", 3)
				return
			}
		}

		if responseData != nil {
			rh.log.Debug("Parsed query response", "mac_address", macAddress, "query_id", responseData.QueryID, "status", responseData.Status, "data_length", len(responseData.Data), "is_fragmented", responseData.IsFragmented)

			// Get tracker and update cached data
			tracker := rh.GetTracker(macAddress)
			if tracker != nil {
				tracker.mutex.Lock()
				// Store model ID if this is a hardware info response
				if responseData.QueryID == CommandGetHardwareInfo && responseData.Status == 0 && len(responseData.Data) >= 1 {
					tracker.ModelID = int(responseData.Data[0])
					rh.log.Info("Model ID detected from hardware info", "mac_address", macAddress, "model_id", tracker.ModelID)

					// Emit model detected event
					if rh.eventEmitter != nil {
						rh.eventEmitter.EmitEvent(events.BLEEvent{
							Type: events.EventModelDetected,
							Device: map[string]string{
								"mac_address": macAddress,
								"model_id":    string(rune(tracker.ModelID)),
							},
							Timestamp: time.Now(),
						})
					}
				}
				tracker.mutex.Unlock()
			}

			// Route response to waiting request or handle as legacy response
			rh.handleResponse(macAddress, responseData)

			// Handle specific response types
			rh.handleSpecificResponse(macAddress, responseData.QueryID, responseData.Status, responseData.Data)
		}
	}
}

// CreateCommandResponseHandler creates a notification handler for command responses with correlation
// Thread Safety: Returns a handler function that safely processes concurrent notifications.
// Handler uses thread-safe response processing and prevents race conditions.
func (rh *ResponseHandler) CreateCommandResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Trace("Received command response", "mac_address", macAddress, "response_bytes", buf, "length", len(buf))

		// Parse command response - similar structure to query response
		responseData := rh.processFragmentedResponse(macAddress, buf)
		if responseData == nil && len(buf) >= 2 {
			// Simple command response: [Command ID] [Status] [Data...]
			commandID := buf[0]
			status := buf[1]
			data := buf[2:]

			responseData = &QueryResponseData{
				QueryID:      commandID,
				Status:       status,
				Data:         data,
				ResponseTime: time.Now(),
				IsFragmented: false,
			}
		}

		if responseData != nil {
			rh.log.Debug("Parsed command response", "mac_address", macAddress, "command_id", responseData.QueryID, "status", responseData.Status, "data_length", len(responseData.Data))

			// Route response to waiting request
			rh.handleResponse(macAddress, responseData)
		}

		// Emit command response event
		if rh.eventEmitter != nil {
			rh.eventEmitter.EmitEvent(events.BLEEvent{
				Type: events.EventCommandResponse,
				Device: map[string]string{
					"mac_address": macAddress,
				},
				Timestamp: time.Now(),
			})
			rh.log.Trace("Command response event emitted", "mac_address", macAddress)
		} else {
			rh.log.Warn("Event emitter not available for command response", "mac_address", macAddress)
		}
	}
}

// CreateSettingsResponseHandler creates a notification handler for settings responses with correlation
// Thread Safety: Returns a handler function that safely processes concurrent notifications.
// Handler uses thread-safe response processing and prevents race conditions.
func (rh *ResponseHandler) CreateSettingsResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Trace("Received settings response", "mac_address", macAddress, "response_bytes", buf, "length", len(buf))

		// Parse settings response - similar structure to other responses
		responseData := rh.processFragmentedResponse(macAddress, buf)
		if responseData == nil && len(buf) >= 2 {
			// Simple settings response: [Setting ID] [Status] [Data...]
			settingID := buf[0]
			status := buf[1]
			data := buf[2:]

			responseData = &QueryResponseData{
				QueryID:      settingID,
				Status:       status,
				Data:         data,
				ResponseTime: time.Now(),
				IsFragmented: false,
			}
		}

		if responseData != nil {
			rh.log.Debug("Parsed settings response", "mac_address", macAddress, "setting_id", responseData.QueryID, "status", responseData.Status, "data_length", len(responseData.Data))

			// Route response to waiting request
			rh.handleResponse(macAddress, responseData)
		}

		// Emit settings response event
		if rh.eventEmitter != nil {
			rh.eventEmitter.EmitEvent(events.BLEEvent{
				Type: events.EventSettingsResponse,
				Device: map[string]string{
					"mac_address": macAddress,
				},
				Timestamp: time.Now(),
			})
			rh.log.Trace("Settings response event emitted", "mac_address", macAddress)
		} else {
			rh.log.Warn("Event emitter not available for settings response", "mac_address", macAddress)
		}
	}
}

// GetHandler returns the appropriate notification handler for a response type
// Thread Safety: Pure function with no shared state access.
// Safe for concurrent use.
func (rh *ResponseHandler) GetHandler(responseType string) func([]byte) {
	switch responseType {
	case "query":
		// Return a generic handler that can be used for any device
		return func(buf []byte) {
			// We need the MAC address to route properly, but this is a generic handler
			// The connection manager should call device-specific handlers instead
			rh.log.Warn("Generic query handler called without device context", "data_length", len(buf))
		}
	case "command":
		return func(buf []byte) {
			rh.log.Warn("Generic command handler called without device context", "data_length", len(buf))
		}
	case "settings":
		return func(buf []byte) {
			rh.log.Warn("Generic settings handler called without device context", "data_length", len(buf))
		}
	default:
		return nil
	}
}

// GetDeviceHandler returns a device-specific notification handler for a response type
// Thread Safety: Returns thread-safe handler functions for each response type.
// Safe for concurrent execution.
func (rh *ResponseHandler) GetDeviceHandler(macAddress, responseType string) func([]byte) {
	switch responseType {
	case "query":
		return rh.CreateQueryResponseHandler(macAddress)
	case "command":
		return rh.CreateCommandResponseHandler(macAddress)
	case "settings":
		return rh.CreateSettingsResponseHandler(macAddress)
	default:
		return nil
	}
}

// handleSpecificResponse handles specific types of query responses
// Thread Safety: Uses thread-safe tracker access and event emission.
// Handler safely processes concurrent notifications with proper locking.
func (rh *ResponseHandler) handleSpecificResponse(macAddress string, queryID, status byte, data []byte) {
	switch queryID {
	case CommandGetHardwareInfo:
		rh.log.Trace("Hardware info response already processed", "mac_address", macAddress)

	case QueryGetStatusValues:
		if len(data) >= 2 {
			statusID := data[0]
			rh.log.Debug("Processing status response", "mac_address", macAddress, "status_id", statusID, "data_length", len(data))

			if statusID == StatusPairingState { // Pairing State (19)
				if len(data) >= 2 {
					pairingState := data[1] // OpenGoPro format: [Status ID] [Value]
					rh.log.Info("Pairing state received", "mac_address", macAddress, "pairing_state", pairingState)

					// Store pairing state in tracker for future reference
					if tracker := rh.GetTracker(macAddress); tracker != nil {
						tracker.mutex.Lock()
						tracker.PairingState = int(pairingState)
						tracker.mutex.Unlock()
						rh.log.Debug("Pairing state cached in tracker", "mac_address", macAddress, "pairing_state", pairingState)
					}

					// Emit pairing state event
					if rh.eventEmitter != nil {
						rh.eventEmitter.EmitEvent(events.BLEEvent{
							Type: events.EventPairingStateChanged,
							Device: map[string]string{
								"mac_address":   macAddress,
								"pairing_state": string(rune(pairingState)),
							},
							Timestamp: time.Now(),
						})
						rh.log.Trace("Pairing state change event emitted", "mac_address", macAddress, "pairing_state", pairingState)
					}
				} else {
					rh.log.Error("Insufficient data for pairing state", "mac_address", macAddress, "data_length", len(data), "required_min", 2)
				}
			} else if statusID == StatusBatteryLevel { // Battery Level (2)
				if len(data) >= 2 {
					batteryLevel := data[1]
					rh.log.Debug("Battery level received", "mac_address", macAddress, "battery_level", batteryLevel)

					// Emit battery level event
					if rh.eventEmitter != nil {
						rh.eventEmitter.EmitEvent(events.BLEEvent{
							Type: events.EventBatteryLevel,
							Device: map[string]string{
								"mac_address":   macAddress,
								"battery_level": string(rune(batteryLevel)),
							},
							Timestamp: time.Now(),
						})
						rh.log.Trace("Battery level event emitted", "mac_address", macAddress, "battery_level", batteryLevel)
					}
				} else {
					rh.log.Error("Insufficient data for battery level", "mac_address", macAddress, "data_length", len(data), "required_min", 2)
				}
			} else if statusID == StatusSystemReady { // System Ready (82)
				if len(data) >= 2 {
					systemReady := data[1]
					rh.log.Info("System ready status received", "mac_address", macAddress, "system_ready", systemReady)

					// Emit system ready event
					if rh.eventEmitter != nil {
						rh.eventEmitter.EmitEvent(events.BLEEvent{
							Type: events.EventSystemReady,
							Device: map[string]string{
								"mac_address":  macAddress,
								"system_ready": string(rune(systemReady)),
							},
							Timestamp: time.Now(),
						})
					}
				}
			} else {
				rh.log.Trace("Unknown status ID received", "mac_address", macAddress, "status_id", statusID, "data_length", len(data))
			}
		} else {
			rh.log.Error("Insufficient data for status values", "mac_address", macAddress, "data_length", len(data), "required_min", 2)
		}

	default:
		rh.log.Trace("Unhandled query response ID", "mac_address", macAddress, "query_id", queryID, "status", status)
	}
}

// WaitForResponse waits for a specific response with timeout (legacy method for backward compatibility)
// Thread Safety: Uses thread-safe SendCommandWithResponse internally.
// Safe for concurrent execution with different devices.
func (rh *ResponseHandler) WaitForResponse(macAddress string, queryID byte, timeout time.Duration) (*QueryResponseData, error) {
	rh.log.Debug("Using legacy WaitForResponse method", "mac_address", macAddress, "query_id", queryID, "timeout", timeout)

	// Use the new correlation system under the hood
	return rh.SendCommandWithResponse(macAddress, queryID, nil, timeout, true)
}

// GetLastResponse returns the last response received for a device
// Thread Safety: Read lock protects concurrent access to tracker lastResponse.
// Safe for concurrent reads from multiple goroutines.
func (rh *ResponseHandler) GetLastResponse(macAddress string) *QueryResponseData {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		return nil
	}

	tracker.mutex.RLock()
	defer tracker.mutex.RUnlock()
	return tracker.lastResponse
}

// GetModelID returns the last detected model ID for a device
// Thread Safety: Read lock protects concurrent access to tracker ModelID.
// Safe for concurrent reads from multiple goroutines.
func (rh *ResponseHandler) GetModelID(macAddress string) int {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		return 0
	}

	tracker.mutex.RLock()
	defer tracker.mutex.RUnlock()
	return tracker.ModelID
}

// GetPairingState returns the last detected pairing state for a device
// Thread Safety: Read lock protects concurrent access to tracker PairingState.
// Safe for concurrent reads from multiple goroutines.
func (rh *ResponseHandler) GetPairingState(macAddress string) int {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		return 0
	}

	tracker.mutex.RLock()
	defer tracker.mutex.RUnlock()
	return tracker.PairingState
}

// SendCommandWithResponse sends a command and waits for the correlated response
// Thread Safety: Uses atomic operations for request registration and response correlation.
// Prevents race conditions between request setup and response handling.
func (rh *ResponseHandler) SendCommandWithResponse(macAddress string, commandID byte, data []byte, timeout time.Duration, isQuery bool) (*QueryResponseData, error) {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		return nil, fmt.Errorf("no response tracker found for device %s", macAddress)
	}

	// Intelligent timeout selection based on OpenGoPro patterns
	if timeout == 0 {
		timeout = rh.selectOptimalTimeout(commandID, isQuery)
	}

	// Create pending request
	responseChan := make(chan *QueryResponseData, 1)
	pendingReq := &PendingRequest{
		CommandID:    commandID,
		ResponseChan: responseChan,
		Timeout:      timeout,
		StartTime:    time.Now(),
		IsQuery:      isQuery,
	}

	// Register the pending request
	tracker.mutex.Lock()
	if tracker.pendingRequests == nil {
		tracker.pendingRequests = make(map[byte]*PendingRequest)
	}
	tracker.pendingRequests[commandID] = pendingReq
	tracker.mutex.Unlock()

	rh.log.Debug("Registered pending request", "mac_address", macAddress, "command_id", commandID, "timeout", timeout, "is_query", isQuery)

	// Wait for response or timeout
	select {
	case response := <-responseChan:
		if response == nil {
			return nil, fmt.Errorf("request timed out for command %d", commandID)
		}
		rh.log.Debug("Received correlated response", "mac_address", macAddress, "command_id", commandID, "response_time", response.ResponseTime)
		return response, nil
	case <-time.After(timeout):
		// Remove from pending requests on timeout
		tracker.mutex.Lock()
		delete(tracker.pendingRequests, commandID)
		delete(tracker.fragmentBuffer, commandID)
		tracker.mutex.Unlock()

		rh.log.Warn("Request timed out", "mac_address", macAddress, "command_id", commandID, "timeout", timeout)
		return nil, fmt.Errorf("timeout waiting for response to command %d", commandID)
	}
}

// selectOptimalTimeout selects appropriate timeout based on command type and OpenGoPro patterns
// Thread Safety: Pure function with no shared state access. Safe for concurrent use.
func (rh *ResponseHandler) selectOptimalTimeout(commandID byte, isQuery bool) time.Duration {
	// Based on OpenGoPro implementation analysis, different operations need different timeouts

	if isQuery {
		switch commandID {
		case QueryGetStatusValues:
			// Status queries including pairing state
			return StatusTimeout
		case QueryGetSettingValues:
			// Settings queries are typically faster
			return DefaultTimeout
		case QueryGetSettingCapabilities:
			// Capability queries can be slow due to large responses
			return HardwareInfoTimeout
		default:
			return DefaultTimeout
		}
	} else {
		// Command timeouts
		switch commandID {
		case CommandGetHardwareInfo:
			// Hardware info is notoriously slow
			return HardwareInfoTimeout
		case ProtobufCommandSetCameraControl, CommandKeepAlive:
			// Control commands are typically fast
			return QuickTimeout
		case CommandGetOpenGoProVer:
			// Version queries can be slow on some models
			return DefaultTimeout
		default:
			return DefaultTimeout
		}
	}
}

// handleResponse processes incoming responses and routes them to waiting requests
// Thread Safety: Uses tracker lock to atomically route responses to pending requests.
// Prevents race conditions between response arrival and request cleanup.
func (rh *ResponseHandler) handleResponse(macAddress string, responseData *QueryResponseData) {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		rh.log.Error("No response tracker found for response", "mac_address", macAddress, "query_id", responseData.QueryID)
		return
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	// Check for pending request
	if pendingReq, exists := tracker.pendingRequests[responseData.QueryID]; exists {
		// Send response to waiting request
		select {
		case pendingReq.ResponseChan <- responseData:
			rh.log.Debug("Routed response to pending request", "mac_address", macAddress, "command_id", responseData.QueryID)
		default:
			rh.log.Warn("Failed to route response - channel full or closed", "mac_address", macAddress, "command_id", responseData.QueryID)
		}

		// Clean up the pending request
		delete(tracker.pendingRequests, responseData.QueryID)
		delete(tracker.fragmentBuffer, responseData.QueryID)
	} else {
		// No pending request - store as last response for backward compatibility
		tracker.lastResponse = responseData

		// Try to send to legacy channel (non-blocking)
		select {
		case tracker.responseChannel <- responseData:
			rh.log.Trace("Response queued in legacy channel", "mac_address", macAddress, "query_id", responseData.QueryID)
		default:
			rh.log.Warn("Legacy response channel full, dropping response", "mac_address", macAddress, "query_id", responseData.QueryID)
		}
	}
}

// processFragmentedResponse handles fragmented responses according to OpenGoPro specification
// Based on analysis of successful OpenGoPro implementations
// Thread Safety: Uses tracker lock to atomically manage fragment buffers.
// Prevents race conditions during fragment assembly from concurrent notifications.
func (rh *ResponseHandler) processFragmentedResponse(macAddress string, buf []byte) *QueryResponseData {
	if len(buf) < 1 {
		rh.log.Error("Invalid packet - empty buffer", "mac_address", macAddress)
		return nil
	}

	// Periodically clean up stale fragments to prevent memory leaks
	go rh.cleanupStaleFragments(macAddress)

	// Parse OpenGoPro packet format according to specification
	header := buf[0]

	// OpenGoPro packet header format:
	// Bit 7 (0x80): Continuation bit - 1 if more fragments follow
	// Bit 6 (0x40): Extended header bit - 1 if 13-bit length field is used
	// Bit 5 (0x20): Start bit - 1 if this is the start of a new message
	// Bits 4-0: Length (5-bit) or part of extended length

	isContinuation := (header & 0x80) != 0
	hasExtendedHeader := (header & 0x40) != 0
	isStart := (header & 0x20) != 0

	var length int
	var dataStart int

	if hasExtendedHeader {
		// 13-bit length field across header and next byte
		if len(buf) < 2 {
			rh.log.Error("Invalid extended header packet - too short", "mac_address", macAddress, "length", len(buf))
			return nil
		}
		// Combine bits 4-0 of header with bits 7-0 of next byte
		length = int((int(header)&0x1F)<<8 | int(buf[1]))
		dataStart = 2
	} else {
		// 5-bit length field in header
		length = int(header & 0x1F)
		dataStart = 1
	}

	if len(buf) < dataStart+length {
		rh.log.Error("Packet shorter than declared length",
			"mac_address", macAddress,
			"declared", length,
			"actual", len(buf)-dataStart,
			"header", fmt.Sprintf("0x%02X", header))
		return nil
	}

	packetData := buf[dataStart : dataStart+length]

	rh.log.Trace("Processing packet",
		"mac_address", macAddress,
		"header", fmt.Sprintf("0x%02X", header),
		"is_start", isStart,
		"is_continuation", isContinuation,
		"has_extended", hasExtendedHeader,
		"length", length,
		"data_start", dataStart)

	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		rh.log.Error("No tracker available", "mac_address", macAddress)
		return nil
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	if tracker.fragmentBuffer == nil {
		tracker.fragmentBuffer = make(map[byte]*FragmentedResponse)
	}

	if isStart {
		// Start of new response (may be single packet or first of multi-packet)
		if len(packetData) < 2 {
			rh.log.Error("Invalid start packet - missing query ID and status", "mac_address", macAddress, "length", len(packetData))
			return nil
		}

		queryID := packetData[0]
		status := packetData[1]
		responseData := packetData[2:]

		if !isContinuation {
			// Complete single-packet response
			rh.log.Debug("Complete single-packet response",
				"mac_address", macAddress,
				"query_id", fmt.Sprintf("0x%02X", queryID),
				"status", fmt.Sprintf("0x%02X", status),
				"data_length", len(responseData))

			return &QueryResponseData{
				QueryID:      queryID,
				Status:       status,
				Data:         responseData,
				ResponseTime: time.Now(),
				IsFragmented: false,
			}
		} else {
			// Start of multi-packet response
			rh.log.Debug("Starting fragmented response",
				"mac_address", macAddress,
				"query_id", fmt.Sprintf("0x%02X", queryID),
				"status", fmt.Sprintf("0x%02X", status),
				"first_fragment_size", len(responseData))

			tracker.fragmentBuffer[queryID] = &FragmentedResponse{
				QueryID:      queryID,
				Status:       status,
				Fragments:    [][]byte{responseData},
				TotalLength:  len(responseData),
				LastReceived: time.Now(),
				IsComplete:   false,
			}

			return nil // Wait for continuation packets
		}
	} else if isContinuation {
		// Continuation packet - find matching fragmented response
		var activeResponse *FragmentedResponse
		var matchingQueryID byte

		// Find the most recent incomplete fragmented response
		// In practice, there should typically be only one active at a time
		for queryID, response := range tracker.fragmentBuffer {
			if !response.IsComplete && time.Since(response.LastReceived) < 30*time.Second {
				activeResponse = response
				matchingQueryID = queryID
				break
			}
		}

		if activeResponse == nil {
			rh.log.Error("No active fragmented response for continuation packet",
				"mac_address", macAddress,
				"active_fragments", len(tracker.fragmentBuffer))
			return nil
		}

		// Add fragment data
		activeResponse.Fragments = append(activeResponse.Fragments, packetData)
		activeResponse.TotalLength += len(packetData)
		activeResponse.LastReceived = time.Now()

		rh.log.Debug("Added continuation fragment",
			"mac_address", macAddress,
			"query_id", fmt.Sprintf("0x%02X", matchingQueryID),
			"fragment_num", len(activeResponse.Fragments),
			"fragment_size", len(packetData),
			"total_size", activeResponse.TotalLength,
			"has_more", isContinuation)

		// If this is the last fragment (no continuation bit), assemble response
		if !isContinuation {
			rh.log.Error("Logic error: continuation packet without continuation bit", "mac_address", macAddress)
			return nil
		}

		return nil // Wait for more fragments
	} else {
		// This is a final fragment (no continuation bit set)
		var activeResponse *FragmentedResponse
		var matchingQueryID byte

		// Find the active fragmented response
		for queryID, response := range tracker.fragmentBuffer {
			if !response.IsComplete && time.Since(response.LastReceived) < 30*time.Second {
				activeResponse = response
				matchingQueryID = queryID
				break
			}
		}

		if activeResponse == nil {
			// No active fragmented response - this might be a single packet response
			// that doesn't follow the expected fragmentation pattern
			rh.log.Debug("No active fragmented response, treating as single packet",
				"mac_address", macAddress,
				"header", fmt.Sprintf("0x%02X", header),
				"data_length", len(packetData))

			// Try to parse as a single packet response
			if len(packetData) >= 2 {
				queryID := packetData[0]
				status := packetData[1]
				responseData := packetData[2:]

				return &QueryResponseData{
					QueryID:      queryID,
					Status:       status,
					Data:         responseData,
					ResponseTime: time.Now(),
					IsFragmented: false,
				}
			} else {
				rh.log.Warn("Packet too short to parse as response",
					"mac_address", macAddress,
					"data_length", len(packetData))
				return nil
			}
		}

		// Add final fragment
		activeResponse.Fragments = append(activeResponse.Fragments, packetData)
		activeResponse.TotalLength += len(packetData)
		activeResponse.IsComplete = true

		// Assemble complete response
		var completeData []byte
		for _, fragment := range activeResponse.Fragments {
			completeData = append(completeData, fragment...)
		}

		// Clean up
		delete(tracker.fragmentBuffer, matchingQueryID)

		rh.log.Debug("Assembled complete fragmented response",
			"mac_address", macAddress,
			"query_id", fmt.Sprintf("0x%02X", activeResponse.QueryID),
			"status", fmt.Sprintf("0x%02X", activeResponse.Status),
			"total_fragments", len(activeResponse.Fragments),
			"total_size", len(completeData))

		return &QueryResponseData{
			QueryID:      activeResponse.QueryID,
			Status:       activeResponse.Status,
			Data:         completeData,
			ResponseTime: time.Now(),
			IsFragmented: true,
			TotalSize:    len(completeData),
		}
	}
}

// cleanupStaleFragments removes old incomplete fragments that have timed out
// Thread Safety: Uses tracker lock to atomically clean up fragment buffers.
// Prevents memory leaks from incomplete fragmented responses.
func (rh *ResponseHandler) cleanupStaleFragments(macAddress string) {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		return
	}

	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()

	if tracker.fragmentBuffer == nil {
		return
	}

	now := time.Now()
	staleFragments := make([]byte, 0)

	// Find fragments that have timed out
	for queryID, response := range tracker.fragmentBuffer {
		if !response.IsComplete && now.Sub(response.LastReceived) > QuickTimeout {
			staleFragments = append(staleFragments, queryID)
		}
	}

	// Remove stale fragments
	for _, queryID := range staleFragments {
		rh.log.Warn("Cleaning up stale fragment",
			"mac_address", macAddress,
			"query_id", fmt.Sprintf("0x%02X", queryID),
			"age", now.Sub(tracker.fragmentBuffer[queryID].LastReceived))
		delete(tracker.fragmentBuffer, queryID)
	}

	if len(staleFragments) > 0 {
		rh.log.Debug("Fragment cleanup completed",
			"mac_address", macAddress,
			"removed_fragments", len(staleFragments))
	}
}
