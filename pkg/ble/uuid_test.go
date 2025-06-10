package ble

import (
	"testing"
)

// TestOpenGoProUUIDs tests that all OpenGoPro UUIDs are correctly formatted
func TestOpenGoProUUIDs(t *testing.T) {
	tests := []struct {
		name string
		uuid string
		want bool
	}{
		{"Valid GoPro Command UUID", CharCommand, true},
		{"Valid GoPro Query UUID", CharQuery, true},
		{"Valid GoPro Settings UUID", CharSettings, true},
		{"Invalid UUID - wrong prefix", "a5f90072-aa8d-11e3-9046-0002a5d5c51b", false},
		{"Invalid UUID - wrong suffix", "b5f90072-bb8d-11e3-9046-0002a5d5c51b", false},
		{"Invalid UUID - too short", "b5f90072", false},
		{"Invalid UUID - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGoProUUID(tt.uuid); got != tt.want {
				t.Errorf("IsGoProUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestControlServiceUUID tests the control service UUID validation
func TestControlServiceUUID(t *testing.T) {
	tests := []struct {
		name string
		uuid string
		want bool
	}{
		{"Valid Control Service UUID", ServiceControl, true},
		{"Valid short form", "fea6", true},
		{"Valid hex form", "0xfea6", true},
		{"Invalid UUID", "b5f90001-aa8d-11e3-9046-0002a5d5c51b", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsControlServiceUUID(tt.uuid); got != tt.want {
				t.Errorf("IsControlServiceUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUUIDValidation tests the overall UUID validation function
func TestUUIDValidation(t *testing.T) {
	err := ValidateOpenGoProUUIDs()
	if err != nil {
		t.Errorf("ValidateOpenGoProUUIDs() failed: %v", err)
	}
}

// TestCharacteristicNames tests the characteristic name mapping
func TestCharacteristicNames(t *testing.T) {
	tests := []struct {
		uuid         string
		expectedName string
	}{
		{CharCommand, "Command Request (GP-0072)"},
		{CharCommandResponse, "Command Response (GP-0073)"},
		{CharQuery, "Query Request (GP-0076)"},
		{CharQueryResponse, "Query Response (GP-0077)"},
		{CharSettings, "Settings Request (GP-0074)"},
		{CharSettingsResponse, "Settings Response (GP-0075)"},
		{CharWifiSSID, "WiFi SSID (GP-0002)"},
		{"unknown-uuid", "Unknown Characteristic"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			if got := GetCharacteristicName(tt.uuid); got != tt.expectedName {
				t.Errorf("GetCharacteristicName() = %v, want %v", got, tt.expectedName)
			}
		})
	}
}

// TestServiceNames tests the service name mapping
func TestServiceNames(t *testing.T) {
	tests := []struct {
		uuid         string
		expectedName string
	}{
		{ServiceWifiAP, "WiFi Access Point Service (GP-0001)"},
		{ServiceControl, "Control & Query Service (FEA6)"},
		{ServiceCameraMgmt, "Camera Management Service (GP-0090)"},
		{"unknown-uuid", "Unknown Service"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			if got := GetServiceName(tt.uuid); got != tt.expectedName {
				t.Errorf("GetServiceName() = %v, want %v", got, tt.expectedName)
			}
		})
	}
}

// BenchmarkIsGoProUUID benchmarks the UUID validation function
func BenchmarkIsGoProUUID(b *testing.B) {
	uuid := CharCommand
	for i := 0; i < b.N; i++ {
		IsGoProUUID(uuid)
	}
}
