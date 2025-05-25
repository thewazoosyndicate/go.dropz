package ble

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble/events"
	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

// DiscoveryManager handles BLE device discovery and scanning operations
type DiscoveryManager struct {
	adapter      *bluetooth.Adapter
	devices      map[string]*Device
	isScanning   bool
	scanMutex    *sync.RWMutex
	mutex        *sync.RWMutex
	log          logger.Logger
	eventEmitter *events.EventEmitter
}

// NewDiscoveryManager creates a new discovery manager
func NewDiscoveryManager(adapter *bluetooth.Adapter, eventEmitter *events.EventEmitter, log logger.Logger) *DiscoveryManager {
	return &DiscoveryManager{
		adapter:      adapter,
		devices:      make(map[string]*Device),
		scanMutex:    &sync.RWMutex{},
		mutex:        &sync.RWMutex{},
		log:          log,
		eventEmitter: eventEmitter,
	}
}

// StartScanning starts scanning for GoPro devices
func (d *DiscoveryManager) StartScanning(ctx context.Context) error {
	d.scanMutex.Lock()
	if d.isScanning {
		d.scanMutex.Unlock()
		d.log.Tracef("Scan already in progress, skipping duplicate start request")
		return nil // Return success rather than error to facilitate continuous scanning
	}
	d.isScanning = true
	d.scanMutex.Unlock()

	d.log.Infof("Starting BLE scan for GoPro devices (service 0xFEA6)")

	// Create a derived context with reasonable timeout if none provided
	scanCtx := ctx
	var cancel context.CancelFunc
	deadline, hasDeadline := ctx.Deadline()

	// If no deadline is set or it's too far away, set a reasonable one
	if !hasDeadline || time.Until(deadline) > 30*time.Second {
		scanCtx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}

	// Use a channel to signal scan completion
	scanDone := make(chan struct{})
	var scanErr error

	// Run scan in a goroutine so we can monitor for context timeout
	go func() {
		defer close(scanDone)
		scanErr = d.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			d.processScanResult(result)
		})
	}()

	// Wait for scan to complete or context timeout
	select {
	case <-scanDone:
		// Scan completed normally
		d.scanMutex.Lock()
		d.isScanning = false
		d.scanMutex.Unlock()

		if scanErr != nil {
			d.log.Errorf("BLE scan failed: error=%v", scanErr)
			return fmt.Errorf("adapter scan error: %w", scanErr)
		}

		d.log.Infof("BLE scan completed: devices_found=%d", len(d.devices))
		return nil

	case <-scanCtx.Done():
		// Context timeout or cancellation
		d.log.Tracef("BLE scan terminating: reason=%v", scanCtx.Err())
		return d.handleScanTimeout(scanCtx, scanDone)
	}
}

// processScanResult processes a single scan result
func (d *DiscoveryManager) processScanResult(result bluetooth.ScanResult) {
	// Process scan result
	if strings.Contains(result.LocalName(), "GoPro") {
		d.mutex.Lock()
		defer d.mutex.Unlock()

		// Create or update device
		device := &Device{
			Name:       result.LocalName(),
			MACAddress: result.Address.String(),
			RSSI:       int32(result.RSSI),
		}

		isNewDevice := d.devices[device.MACAddress] == nil
		d.log.Tracef("BLE device found: name=%s mac=%s rssi=%d is_new=%t",
			device.Name, device.MACAddress, device.RSSI, isNewDevice)

		d.devices[device.MACAddress] = device

		// Emit device discovered event
		d.eventEmitter.EmitEvent(events.BLEEvent{
			Type:      events.EventDeviceDiscovered,
			Device:    device,
			Timestamp: time.Now(),
		})
	}
}

// handleScanTimeout handles scan timeout and cleanup
func (d *DiscoveryManager) handleScanTimeout(scanCtx context.Context, scanDone chan struct{}) error {
	// Stop the scan
	if err := d.adapter.StopScan(); err != nil {
		d.log.Warnf("DiscoveryManager: Failed to stop scan: %v", err)
	}

	// Wait for scan to actually stop with a timeout
	select {
	case <-scanDone:
		// Scan has stopped
		d.log.Tracef("BLE scan stopped gracefully: reason=cancelled")
	case <-time.After(2 * time.Second):
		d.log.Warnf("BLE scan timeout: action=forced_termination delay=2s")

		// Force reset the scanning state
		d.scanMutex.Lock()
		d.isScanning = false
		d.scanMutex.Unlock()
	}

	d.scanMutex.Lock()
	d.isScanning = false
	d.scanMutex.Unlock()

	// Don't treat deadline exceeded as an error in the continuous scan case
	if scanCtx.Err() == context.DeadlineExceeded {
		d.log.Tracef("BLE scan timeout: mode=continuous status=expected")
		return nil
	}

	// Other context errors should be returned
	return scanCtx.Err()
}

// StopScanning stops the current scan
func (d *DiscoveryManager) StopScanning() error {
	d.scanMutex.Lock()
	defer d.scanMutex.Unlock()

	if !d.isScanning {
		return nil
	}

	if err := d.adapter.StopScan(); err != nil {
		d.log.Warnf("BLE scan termination failed: error=%v", err)
		return err
	}

	d.isScanning = false
	d.log.Infof("BLE scan stopped: device_count=%d", len(d.devices))
	return nil
}

// GetDiscoveredDevices returns all discovered GoPro devices
func (d *DiscoveryManager) GetDiscoveredDevices() []Device {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	devices := make([]Device, 0, len(d.devices))
	for _, device := range d.devices {
		devices = append(devices, *device)
	}
	return devices
}

// IsScanning returns whether a scan is currently in progress
func (d *DiscoveryManager) IsScanning() bool {
	d.scanMutex.RLock()
	defer d.scanMutex.RUnlock()
	return d.isScanning
}

// GetDevice returns a device by MAC address
func (d *DiscoveryManager) GetDevice(macAddress string) (*Device, bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	device, exists := d.devices[macAddress]
	return device, exists
}

// ClearDevices clears all discovered devices
func (d *DiscoveryManager) ClearDevices() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.devices = make(map[string]*Device)
}
