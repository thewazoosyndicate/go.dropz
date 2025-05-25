package events

import "time"

// EventType represents the type of BLE event
type EventType string

const (
	EventDeviceDiscovered     EventType = "device_discovered"
	EventDeviceUpdated        EventType = "device_updated"
	EventDeviceConnected      EventType = "device_connected"
	EventDeviceDisconnected   EventType = "device_disconnected"
	EventPairingStarted       EventType = "pairing_started"
	EventPairingCompleted     EventType = "pairing_completed"
	EventPairingFailed        EventType = "pairing_failed"
	EventPairingStateChanged  EventType = "pairing_state_changed"
	EventModelDetected        EventType = "model_detected"
	EventBatteryLevel         EventType = "battery_level"
	EventSystemReady          EventType = "system_ready"
	EventWifiSSID             EventType = "wifi_ssid"
	EventWifiPassword         EventType = "wifi_password"
	EventCommandResponse      EventType = "command_response"
	EventSettingsResponse     EventType = "settings_response"
	EventCharacteristicRead   EventType = "characteristic_read"
	EventCharacteristicWrite  EventType = "characteristic_write"
	EventNotificationReceived EventType = "notification_received"
)

// BLEEvent represents a BLE-related event
type BLEEvent struct {
	Type      EventType   `json:"type"`
	Device    interface{} `json:"device,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     error       `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}
