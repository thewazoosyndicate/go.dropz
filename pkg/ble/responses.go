package ble

import (
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble/events"
	"github.com/dropz/dropz/pkg/logger"
)

// ResponseHandler manages response tracking and notification handling
type ResponseHandler struct {
	responseTracker map[string]*ResponseTracker // MAC address -> response tracker
	mutex           sync.RWMutex
	eventEmitter    *events.EventEmitter
	log             logger.Logger
}

// NewResponseHandler creates a new response handler
func NewResponseHandler(eventEmitter *events.EventEmitter, log logger.Logger) *ResponseHandler {
	return &ResponseHandler{
		responseTracker: make(map[string]*ResponseTracker),
		eventEmitter:    eventEmitter,
		log:             log,
	}
}

// InitializeTracker initializes response tracker for a device
func (rh *ResponseHandler) InitializeTracker(macAddress string) {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if rh.responseTracker[macAddress] == nil {
		rh.responseTracker[macAddress] = &ResponseTracker{
			responseChannel: make(chan *QueryResponseData, 10), // Buffer for multiple responses
		}
		rh.log.Trace("Response tracker initialized", "mac_address", macAddress, "buffer_size", 10)
	} else {
		rh.log.Trace("Response tracker already exists", "mac_address", macAddress)
	}
}

// GetTracker gets the response tracker for a device
func (rh *ResponseHandler) GetTracker(macAddress string) *ResponseTracker {
	rh.mutex.RLock()
	defer rh.mutex.RUnlock()
	return rh.responseTracker[macAddress]
}

// CleanupTracker removes the response tracker for a device
func (rh *ResponseHandler) CleanupTracker(macAddress string) {
	rh.mutex.Lock()
	defer rh.mutex.Unlock()

	if tracker, exists := rh.responseTracker[macAddress]; exists {
		close(tracker.responseChannel)
		delete(rh.responseTracker, macAddress)
		rh.log.Info("Response tracker cleaned up", "mac_address", macAddress, "tracker_count", len(rh.responseTracker))
	} else {
		rh.log.Trace("Response tracker not found for cleanup", "mac_address", macAddress)
	}
}

// CreateQueryResponseHandler creates a notification handler for query responses
func (rh *ResponseHandler) CreateQueryResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Trace("Received query response", "mac_address", macAddress, "response_bytes", buf, "length", len(buf))

		// Parse response based on query type
		if len(buf) >= 3 {
			// Basic response structure: [Query ID] [Status] [Data...]
			queryID := buf[0]
			status := buf[1]
			data := buf[2:]

			rh.log.Debug("Parsed query response", "mac_address", macAddress, "query_id", queryID, "status", status, "data_length", len(data))

			// Create response data
			responseData := &QueryResponseData{
				QueryID:      queryID,
				Status:       status,
				Data:         data,
				ResponseTime: time.Now(),
			}

			// Get tracker
			tracker := rh.GetTracker(macAddress)
			if tracker == nil {
				rh.log.Error("No response tracker found for device", "mac_address", macAddress, "query_id", queryID)
				return
			}

			// Store the response
			tracker.mutex.Lock()
			tracker.lastResponse = responseData
			// Store model ID if this is a hardware info response
			if queryID == QueryGetHardwareInfo && status == 0 && len(data) >= 1 {
				tracker.ModelID = int(data[0])
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

			// Send to channel (non-blocking)
			select {
			case tracker.responseChannel <- responseData:
				rh.log.Trace("Response queued in channel", "mac_address", macAddress, "query_id", queryID)
			default:
				rh.log.Warn("Response channel full, dropping oldest response", "mac_address", macAddress, "query_id", queryID, "channel_capacity", cap(tracker.responseChannel))
			}

			// Handle specific response types
			rh.handleSpecificResponse(macAddress, queryID, status, data)
		} else {
			rh.log.Error("Invalid query response format", "mac_address", macAddress, "response_length", len(buf), "expected_min", 3)
		}
	}
}

// CreateCommandResponseHandler creates a notification handler for command responses
func (rh *ResponseHandler) CreateCommandResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Trace("Received command response", "mac_address", macAddress, "response_bytes", buf, "length", len(buf))

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

// CreateSettingsResponseHandler creates a notification handler for settings responses
func (rh *ResponseHandler) CreateSettingsResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Trace("Received settings response", "mac_address", macAddress, "response_bytes", buf, "length", len(buf))

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

// handleSpecificResponse handles specific types of query responses
func (rh *ResponseHandler) handleSpecificResponse(macAddress string, queryID, status byte, data []byte) {
	switch queryID {
	case QueryGetHardwareInfo:
		rh.log.Trace("Hardware info response already processed", "mac_address", macAddress)

	case QueryGetStatusValues:
		if len(data) >= 2 {
			statusID := data[0]
			if statusID == 19 { // Pairing State
				if len(data) >= 3 {
					pairingState := data[2]
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
					rh.log.Error("Insufficient data for pairing state", "mac_address", macAddress, "data_length", len(data), "required_min", 3)
				}
			} else if statusID == 2 { // Battery Level
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
			} else {
				rh.log.Trace("Unknown status ID received", "mac_address", macAddress, "status_id", statusID, "data_length", len(data))
			}
		} else {
			rh.log.Error("Insufficient data for status values", "mac_address", macAddress, "data_length", len(data), "required_min", 2)
		}

	case QueryGetWifiSSID:
		if status == 0 && len(data) > 0 {
			ssid := string(data)
			rh.log.Info("WiFi SSID received", "mac_address", macAddress, "ssid", ssid, "ssid_length", len(ssid))

			// Emit WiFi SSID event
			if rh.eventEmitter != nil {
				rh.eventEmitter.EmitEvent(events.BLEEvent{
					Type: events.EventWifiSSID,
					Device: map[string]string{
						"mac_address": macAddress,
						"wifi_ssid":   ssid,
					},
					Timestamp: time.Now(),
				})
				rh.log.Trace("WiFi SSID event emitted", "mac_address", macAddress)
			}
		} else {
			rh.log.Warn("Failed to get WiFi SSID", "mac_address", macAddress, "status", status, "data_length", len(data))
		}

	case QueryGetWifiPassword:
		if status == 0 && len(data) > 0 {
			password := string(data)
			rh.log.Info("WiFi password received", "mac_address", macAddress, "password_length", len(password))

			// Emit WiFi password event
			if rh.eventEmitter != nil {
				rh.eventEmitter.EmitEvent(events.BLEEvent{
					Type: events.EventWifiPassword,
					Device: map[string]string{
						"mac_address":   macAddress,
						"wifi_password": password,
					},
					Timestamp: time.Now(),
				})
				rh.log.Trace("WiFi password event emitted", "mac_address", macAddress)
			}
		} else {
			rh.log.Warn("Failed to get WiFi password", "mac_address", macAddress, "status", status, "data_length", len(data))
		}

	default:
		rh.log.Trace("Unhandled query response ID", "mac_address", macAddress, "query_id", queryID, "status", status)
	}
}

// WaitForResponse waits for a specific response with timeout
func (rh *ResponseHandler) WaitForResponse(macAddress string, queryID byte, timeout time.Duration) (*QueryResponseData, error) {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		rh.log.Error("No response tracker found for wait operation", "mac_address", macAddress, "query_id", queryID)
		return nil, fmt.Errorf("no response tracker found for device %s", macAddress)
	}

	rh.log.Debug("Waiting for response", "mac_address", macAddress, "query_id", queryID, "timeout", timeout)

	select {
	case response := <-tracker.responseChannel:
		if response.QueryID == queryID {
			rh.log.Debug("Received expected response", "mac_address", macAddress, "query_id", queryID, "response_time", response.ResponseTime)
			return response, nil
		}
		// If it's not the response we're looking for, put it back
		select {
		case tracker.responseChannel <- response:
			rh.log.Trace("Unexpected response returned to channel", "mac_address", macAddress, "expected_id", queryID, "received_id", response.QueryID)
		default:
			rh.log.Warn("Channel full, dropping unexpected response", "mac_address", macAddress, "expected_id", queryID, "received_id", response.QueryID)
		}
		return nil, fmt.Errorf("received unexpected response (expected ID %d, got %d)", queryID, response.QueryID)

	case <-time.After(timeout):
		rh.log.Warn("Timeout waiting for response", "mac_address", macAddress, "query_id", queryID, "timeout", timeout)
		return nil, fmt.Errorf("timeout waiting for response to query %d", queryID)
	}
}

// GetLastResponse returns the last response received for a device
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
func (rh *ResponseHandler) GetPairingState(macAddress string) int {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		return 0
	}

	tracker.mutex.RLock()
	defer tracker.mutex.RUnlock()
	return tracker.PairingState
}
