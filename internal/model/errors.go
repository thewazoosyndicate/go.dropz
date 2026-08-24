package model

import "errors"

// Domain error sentinels. The gRPC layer maps these to status codes,
// so wrap them with %w instead of formatting new strings.
var (
	ErrCameraNotFound   = errors.New("camera not found")
	ErrGroupNotFound    = errors.New("group not found")
	ErrNotManagedPaired = errors.New("camera must be managed and paired")
	// Declared here rather than in ble so the gRPC layer does not have to
	// import the transport driver just to map one sentinel.
	ErrBluetoothUnavailable = errors.New("bluetooth unavailable")
)
