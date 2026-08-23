package ble

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

// Without an adapter the manager must fail soft: sentinel errors, no panics.
func TestDegradedManagerWithoutAdapter(t *testing.T) {
	m := NewManager(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if m.Available() {
		t.Error("Available() = true with nil adapter")
	}

	err := m.StartScanningWithCallback(context.Background(), nil)
	if !errors.Is(err, ErrBluetoothUnavailable) {
		t.Errorf("StartScanningWithCallback err = %v, want ErrBluetoothUnavailable", err)
	}

	if err := m.StopScanning(); err != nil {
		t.Errorf("StopScanning err = %v, want nil", err)
	}

	_, _, err = m.connectAndDiscover("AA:BB:CC:DD:EE:FF", nil)
	if !errors.Is(err, ErrBluetoothUnavailable) {
		t.Errorf("connectAndDiscover err = %v, want ErrBluetoothUnavailable", err)
	}
}
