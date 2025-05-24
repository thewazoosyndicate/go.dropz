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
	}
}

// CreateQueryResponseHandler creates a notification handler for query responses
func (rh *ResponseHandler) CreateQueryResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Debugf("Received query response: %v (length: %d)", buf, len(buf))

		// Parse response based on query type
		if len(buf) >= 3 {
			// Basic response structure: [Query ID] [Status] [Data...]
			queryID := buf[0]
			status := buf[1]
			data := buf[2:]

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
				rh.log.Warnf("No response tracker found for device %s", macAddress)
				return
			}

			// Store the response
			tracker.mutex.Lock()
			tracker.lastResponse = responseData
			// Store model ID if this is a hardware info response
			if queryID == QueryGetHardwareInfo && status == 0 && len(data) >= 1 {
				tracker.ModelID = int(data[0])
				rh.log.Debugf("Stored model ID in tracker: %d", tracker.ModelID)

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
			default:
				rh.log.Debug("Response channel full, dropping oldest response")
			}

			// Handle specific response types
			rh.handleSpecificResponse(macAddress, queryID, status, data)
		}
	}
}

// CreateCommandResponseHandler creates a notification handler for command responses
func (rh *ResponseHandler) CreateCommandResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Debugf("Received command response: %v", buf)

		// Emit command response event
		if rh.eventEmitter != nil {
			rh.eventEmitter.EmitEvent(events.BLEEvent{
				Type: events.EventCommandResponse,
				Device: map[string]string{
					"mac_address": macAddress,
				},
				Timestamp: time.Now(),
			})
		}
	}
}

// CreateSettingsResponseHandler creates a notification handler for settings responses
func (rh *ResponseHandler) CreateSettingsResponseHandler(macAddress string) func([]byte) {
	return func(buf []byte) {
		rh.log.Debugf("Received settings response: %v", buf)

		// Emit settings response event
		if rh.eventEmitter != nil {
			rh.eventEmitter.EmitEvent(events.BLEEvent{
				Type: events.EventSettingsResponse,
				Device: map[string]string{
					"mac_address": macAddress,
				},
				Timestamp: time.Now(),
			})
		}
	}
}

// handleSpecificResponse handles specific types of query responses
func (rh *ResponseHandler) handleSpecificResponse(macAddress string, queryID, status byte, data []byte) {
	switch queryID {
	case QueryGetHardwareInfo:
		// Already handled above

	case QueryGetStatusValues:
		if len(data) >= 2 {
			statusID := data[0]
			if statusID == 19 { // Pairing State
				if len(data) >= 3 {
					pairingState := data[2]
					rh.log.Debugf("Pairing state: %d", pairingState)

					// Store pairing state in tracker for future reference
					if tracker := rh.GetTracker(macAddress); tracker != nil {
						tracker.mutex.Lock()
						tracker.PairingState = int(pairingState)
						tracker.mutex.Unlock()
						rh.log.Debugf("Cached pairing state %d for device %s", pairingState, macAddress)
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
					}
				}
			} else if statusID == 2 { // Battery Level
				if len(data) >= 2 {
					batteryLevel := data[1]
					rh.log.Debugf("Battery level: %d%%", batteryLevel)

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
					}
				}
			}
		}

	case QueryGetWifiSSID:
		if status == 0 && len(data) > 0 {
			ssid := string(data)
			rh.log.Debugf("WiFi SSID: %s", ssid)

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
			}
		}

	case QueryGetWifiPassword:
		if status == 0 && len(data) > 0 {
			password := string(data)
			rh.log.Debugf("WiFi Password: %s", password)

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
			}
		}

	default:
		rh.log.Debugf("Unhandled query response ID: %d", queryID)
	}
}

// WaitForResponse waits for a specific response with timeout
func (rh *ResponseHandler) WaitForResponse(macAddress string, queryID byte, timeout time.Duration) (*QueryResponseData, error) {
	tracker := rh.GetTracker(macAddress)
	if tracker == nil {
		return nil, fmt.Errorf("no response tracker found for device %s", macAddress)
	}

	select {
	case response := <-tracker.responseChannel:
		if response.QueryID == queryID {
			return response, nil
		}
		// If it's not the response we're looking for, put it back
		select {
		case tracker.responseChannel <- response:
		default:
			// Channel full, drop the response
		}
		return nil, fmt.Errorf("received unexpected response (expected ID %d, got %d)", queryID, response.QueryID)

	case <-time.After(timeout):
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
