package ble

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Common error types
var (
	ErrDeviceNotFound         = errors.New("device not found")
	ErrConnectionFailed       = errors.New("connection failed")
	ErrServiceNotFound        = errors.New("service not found")
	ErrCharacteristicNotFound = errors.New("characteristic not found")
	ErrTimeout                = errors.New("operation timed out")
	ErrScanFailed             = errors.New("scan failed")
	ErrDeviceDisconnected     = errors.New("device disconnected")
	ErrSubscriptionFailed     = errors.New("notification subscription failed")
	ErrWriteFailed            = errors.New("characteristic write failed")
	ErrReadFailed             = errors.New("characteristic read failed")

	// New comprehensive error types
	ErrInvalidTLVFormat              = errors.New("invalid TLV format")
	ErrRequiredCharacteristicMissing = errors.New("required characteristic missing")
	ErrInvalidResponseFormat         = errors.New("invalid response format")
	ErrPairingFailed                 = errors.New("pairing failed")
	ErrUnsupportedModel              = errors.New("unsupported GoPro model")
	ErrInvalidParameter              = errors.New("invalid parameter")
	ErrOperationNotSupported         = errors.New("operation not supported")
	ErrNotReady                      = errors.New("device not ready")
	ErrAuthenticationFailed          = errors.New("authentication failed")
	ErrFirmwareIncompatible          = errors.New("firmware incompatible")
)

// BLEError represents an error in BLE operations with additional context
type BLEError struct {
	Op        string    // Operation that failed (e.g., "connect", "scan", "read")
	Device    string    // Device MAC address or name involved
	Err       error     // Underlying error
	Timestamp time.Time // When the error occurred
	Retryable bool      // Whether the operation can be retried
	Attempts  int       // Number of attempts made so far
}

// Error implements the error interface
func (e *BLEError) Error() string {
	if e.Device != "" {
		return fmt.Sprintf("[%s] %s failed for device %s: %v", e.Timestamp.Format(time.RFC3339), e.Op, e.Device, e.Err)
	}
	return fmt.Sprintf("[%s] %s failed: %v", e.Timestamp.Format(time.RFC3339), e.Op, e.Err)
}

// Unwrap returns the underlying error
func (e *BLEError) Unwrap() error {
	return e.Err
}

// NewBLEError creates a new BLEError
func NewBLEError(op, device string, err error, retryable bool, attempts int) *BLEError {
	return &BLEError{
		Op:        op,
		Device:    device,
		Err:       err,
		Timestamp: time.Now(),
		Retryable: retryable,
		Attempts:  attempts,
	}
}

// ValidationError represents validation-specific errors
type ValidationError struct {
	Field  string      // The field that failed validation
	Value  interface{} // The invalid value
	Reason string      // Why validation failed
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%v': %s", e.Field, e.Value, e.Reason)
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value interface{}, reason string) *ValidationError {
	return &ValidationError{
		Field:  field,
		Value:  value,
		Reason: reason,
	}
}

// TLVError represents TLV parsing errors
type TLVError struct {
	Offset int    // Byte offset where error occurred
	Data   []byte // The invalid TLV data
	Reason string // Description of the error
}

func (e *TLVError) Error() string {
	return fmt.Sprintf("TLV parsing error at offset %d: %s (data: %x)", e.Offset, e.Reason, e.Data)
}

// NewTLVError creates a new TLV parsing error
func NewTLVError(offset int, data []byte, reason string) *TLVError {
	return &TLVError{
		Offset: offset,
		Data:   data,
		Reason: reason,
	}
}

// CharacteristicError represents characteristic-specific errors
type CharacteristicError struct {
	ServiceUUID        string // Service UUID
	CharacteristicUUID string // Characteristic UUID
	Operation          string // Operation that failed
	Err                error  // Underlying error
}

func (e *CharacteristicError) Error() string {
	return fmt.Sprintf("characteristic %s in service %s failed %s: %v",
		e.CharacteristicUUID, e.ServiceUUID, e.Operation, e.Err)
}

// Unwrap returns the underlying error
func (e *CharacteristicError) Unwrap() error {
	return e.Err
}

// NewCharacteristicError creates a new characteristic error
func NewCharacteristicError(serviceUUID, charUUID, operation string, err error) *CharacteristicError {
	return &CharacteristicError{
		ServiceUUID:        serviceUUID,
		CharacteristicUUID: charUUID,
		Operation:          operation,
		Err:                err,
	}
}

// ConnectionError represents connection-specific errors with enhanced context
type ConnectionError struct {
	DeviceMAC  string    // MAC address of the device
	Operation  string    // Operation that failed (connect, disconnect, etc.)
	Err        error     // Underlying error
	Suggestion string    // Actionable suggestion for the user
	Timestamp  time.Time // When the error occurred
	Retryable  bool      // Whether the operation can be retried
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection error for device %s during %s: %v (suggestion: %s)",
		e.DeviceMAC, e.Operation, e.Err, e.Suggestion)
}

// Unwrap returns the underlying error
func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// NewConnectionError creates a new connection error
func NewConnectionError(deviceMAC, operation string, err error, suggestion string, timestamp time.Time) *ConnectionError {
	return &ConnectionError{
		DeviceMAC:  deviceMAC,
		Operation:  operation,
		Err:        err,
		Suggestion: suggestion,
		Timestamp:  timestamp,
		Retryable:  IsRetryableError(err),
	}
}

// PairingError represents pairing-specific errors
type PairingError struct {
	DeviceMAC string    // MAC address of the device
	Stage     string    // Pairing stage that failed
	Err       error     // Underlying error
	Timestamp time.Time // When the error occurred
}

func (e *PairingError) Error() string {
	return fmt.Sprintf("pairing failed for device %s at stage '%s': %v", e.DeviceMAC, e.Stage, e.Err)
}

// Unwrap returns the underlying error
func (e *PairingError) Unwrap() error {
	return e.Err
}

// NewPairingError creates a new pairing error
func NewPairingError(deviceMAC, stage string, err error) *PairingError {
	return &PairingError{
		DeviceMAC: deviceMAC,
		Stage:     stage,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// ResponseError represents response parsing/validation errors
type ResponseError struct {
	CommandID   byte      // Command ID that generated the response
	ResponseLen int       // Length of the response
	Err         error     // Underlying error
	Timestamp   time.Time // When the error occurred
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("response error for command 0x%02X (length: %d): %v",
		e.CommandID, e.ResponseLen, e.Err)
}

// Unwrap returns the underlying error
func (e *ResponseError) Unwrap() error {
	return e.Err
}

// NewResponseError creates a new response error
func NewResponseError(commandID byte, responseLen int, err error) *ResponseError {
	return &ResponseError{
		CommandID:   commandID,
		ResponseLen: responseLen,
		Err:         err,
		Timestamp:   time.Now(),
	}
}

// GoProError represents GoPro-specific errors
type GoProError struct {
	ModelID   int    // GoPro model ID (if known)
	Operation string // Operation that failed
	Err       error  // Underlying error
}

func (e *GoProError) Error() string {
	if e.ModelID > 0 {
		return fmt.Sprintf("GoPro model %d operation '%s' failed: %v", e.ModelID, e.Operation, e.Err)
	}
	return fmt.Sprintf("GoPro operation '%s' failed: %v", e.Operation, e.Err)
}

func (e *GoProError) Unwrap() error {
	return e.Err
}

// NewGoProError creates a new GoPro-specific error
func NewGoProError(modelID int, operation string, err error) *GoProError {
	return &GoProError{
		ModelID:   modelID,
		Operation: operation,
		Err:       err,
	}
}

// ShouldRetry determines if an error should be retried
func ShouldRetry(err error) bool {
	var bleErr *BLEError
	if errors.As(err, &bleErr) {
		return bleErr.Retryable
	}

	// Generic error handling
	if errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrScanFailed) ||
		errors.Is(err, ErrConnectionFailed) {
		return true
	}

	return false
}

// BackoffDuration calculates the backoff duration for retries
func BackoffDuration(attempts int) time.Duration {
	// Exponential backoff with jitter
	baseDuration := 100 * time.Millisecond
	maxDuration := 5 * time.Second

	// Calculate exponential duration (2^attempts * baseDuration)
	duration := baseDuration
	for i := 0; i < attempts; i++ {
		duration *= 2
		if duration > maxDuration {
			duration = maxDuration
			break
		}
	}

	// Add jitter (±20%)
	jitterFactor := 0.8 + 0.4*float64(time.Now().Nanosecond())/float64(1e9)
	return time.Duration(float64(duration) * jitterFactor)
}

// ValidateTLVFormat validates TLV (Type-Length-Value) format
func ValidateTLVFormat(data []byte) error {
	if len(data) == 0 {
		return NewTLVError(0, data, "empty TLV data")
	}

	offset := 0
	for offset < len(data) {
		// Check if we have at least Type and Length bytes
		if offset+2 > len(data) {
			return NewTLVError(offset, data[offset:], "incomplete TLV header")
		}

		// Type is the first byte
		tlvType := data[offset]
		// Length is the second byte
		tlvLength := int(data[offset+1])

		// Check if we have enough data for the value
		if offset+2+tlvLength > len(data) {
			return NewTLVError(offset, data[offset:],
				fmt.Sprintf("TLV length %d exceeds remaining data", tlvLength))
		}

		// Validate type values (based on OpenGoPro spec)
		if tlvType == 0 {
			return NewTLVError(offset, data[offset:offset+2], "invalid TLV type 0")
		}

		// Move to next TLV
		offset += 2 + tlvLength
	}

	return nil
}

// ValidateResponseFormat validates response format according to OpenGoPro spec
func ValidateResponseFormat(data []byte, expectedCommandID byte) error {
	if len(data) == 0 {
		return NewValidationError("response_data", data, "empty response")
	}

	// First byte should be the command ID
	if data[0] != expectedCommandID {
		return NewValidationError("command_id", data[0],
			fmt.Sprintf("expected command ID 0x%02X, got 0x%02X", expectedCommandID, data[0]))
	}

	// Second byte should be status (if present)
	if len(data) > 1 {
		status := data[1]
		// Validate status codes
		if status > 0x05 { // Based on OpenGoPro response status codes
			return NewValidationError("status_code", status,
				fmt.Sprintf("invalid status code 0x%02X", status))
		}
	}

	return nil
}

// ValidateMAC validates MAC address format
func ValidateMAC(macAddress string) error {
	if macAddress == "" {
		return NewValidationError("mac_address", macAddress, "empty MAC address")
	}

	// Basic MAC address format validation (XX:XX:XX:XX:XX:XX)
	if len(macAddress) != 17 {
		return NewValidationError("mac_address", macAddress, "invalid MAC address length")
	}

	// Check for proper format
	for i, char := range macAddress {
		if i%3 == 2 { // Colon positions
			if char != ':' {
				return NewValidationError("mac_address", macAddress, "invalid MAC address format")
			}
		} else { // Hex digit positions
			if !((char >= '0' && char <= '9') ||
				(char >= 'A' && char <= 'F') ||
				(char >= 'a' && char <= 'f')) {
				return NewValidationError("mac_address", macAddress, "invalid MAC address characters")
			}
		}
	}

	return nil
}

// ValidateModelID validates GoPro model ID
func ValidateModelID(modelID int) error {
	if modelID <= 0 {
		return NewValidationError("model_id", modelID, "model ID must be positive")
	}

	// Check against known model IDs
	validModels := []int{9, 10, 11, 12, 13} // HERO9, HERO10, HERO11, HERO12, HERO13
	for _, validModel := range validModels {
		if modelID == validModel {
			return nil
		}
	}

	return NewValidationError("model_id", modelID, "unsupported GoPro model")
}

// ValidateDevicePointer validates that a device pointer is not nil
func ValidateDevicePointer(device interface{}) error {
	if device == nil {
		return NewValidationError("device_pointer", nil, "device pointer is nil")
	}
	return nil
}

// ValidateCharacteristicPointer validates that a characteristic pointer is not nil
func ValidateCharacteristicPointer(char interface{}) error {
	if char == nil {
		return NewValidationError("characteristic_pointer", nil, "characteristic pointer is nil")
	}
	return nil
}

// IsRetryableError determines if an error is retryable based on its type and context
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check specific error types
	switch {
	case errors.Is(err, ErrTimeout):
		return true
	case errors.Is(err, ErrScanFailed):
		return true
	case errors.Is(err, ErrConnectionFailed):
		return true
	case errors.Is(err, ErrSubscriptionFailed):
		return true
	case errors.Is(err, ErrWriteFailed):
		return true
	case errors.Is(err, ErrReadFailed):
		return true
	}

	// Check BLE error
	var bleErr *BLEError
	if errors.As(err, &bleErr) {
		return bleErr.Retryable
	}

	// Check for specific error messages that indicate retryable conditions
	errMsg := err.Error()
	retryableMessages := []string{
		"Properties.GetAll",
		"connection timeout",
		"device busy",
		"resource temporarily unavailable",
		"bluetooth not ready",
	}

	for _, msg := range retryableMessages {
		if strings.Contains(errMsg, msg) {
			return true
		}
	}

	return false
}

// WrapWithContext wraps an error with additional context
func WrapWithContext(err error, operation, device string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s operation for device %s: %w", operation, device, err)
}
