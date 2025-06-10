package ble

import (
	"errors"
	"fmt"
	"time"
)

// Standard BLE errors
var (
	ErrDeviceNotFound   = errors.New("device not found")
	ErrConnectionFailed = errors.New("connection failed")
	ErrServiceNotFound  = errors.New("service not found")
	ErrCharNotFound     = errors.New("characteristic not found")
	ErrTimeout          = errors.New("operation timed out")
	ErrInvalidResponse  = errors.New("invalid response")
	ErrDeviceNotReady   = errors.New("device not ready")
	ErrInvalidParameter = errors.New("invalid parameter")
)

// BLEError represents a BLE operation error with context
type BLEError struct {
	Operation string
	Device    string
	Err       error
	Timestamp time.Time
}

func (e *BLEError) Error() string {
	if e.Device != "" {
		return fmt.Sprintf("BLE %s failed for %s: %v", e.Operation, e.Device, e.Err)
	}
	return fmt.Sprintf("BLE %s failed: %v", e.Operation, e.Err)
}

func (e *BLEError) Unwrap() error {
	return e.Err
}

// NewBLEError creates a new BLE error
func NewBLEError(operation, device string, err error) *BLEError {
	return &BLEError{
		Operation: operation,
		Device:    device,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// ValidateMAC validates MAC address format
func ValidateMAC(macAddress string) error {
	if macAddress == "" {
		return ErrInvalidParameter
	}
	if len(macAddress) != 17 {
		return fmt.Errorf("invalid MAC address length: %d", len(macAddress))
	}
	return nil
}

// IsRetryableError determines if an error is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check specific error types
	switch {
	case errors.Is(err, ErrTimeout):
		return true
	case errors.Is(err, ErrCharNotFound):
		return true
	case errors.Is(err, ErrConnectionFailed):
		return true
	case errors.Is(err, ErrInvalidResponse):
		return true
	}

	// BLE errors are generally retryable
	var bleErr *BLEError
	if errors.As(err, &bleErr) {
		return true
	}

	return false
}

// BackoffDuration calculates exponential backoff for retries
func BackoffDuration(attempts int) time.Duration {
	baseDuration := 100 * time.Millisecond
	maxDuration := 5 * time.Second

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
