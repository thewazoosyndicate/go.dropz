package ble

import (
	"errors"
	"fmt"
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
