package ble

import (
	"context"
	"strings"
	"time"

	"tinygo.org/x/bluetooth"
)

// closedChan is returned by ScanDone when no scan is active, so a caller that
// races StopScanning selects immediately instead of parking on a nil channel.
var closedChan = func() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// StartScanningWithCallback scans for GoPro devices with optional live discovery callback
func (m *Manager) StartScanningWithCallback(ctx context.Context, callback DeviceDiscoveryCallback) error {
	if m.adapter == nil {
		return ErrBluetoothUnavailable
	}
	m.mutex.Lock()
	if m.isScanning {
		m.mutex.Unlock()
		return nil
	}
	m.isScanning = true
	m.scanDone = make(chan struct{})
	m.discoveryCallback = callback
	// The goroutine watches its own session's channel, not m.isScanning: a
	// goroutine sleeping in backoff cannot observe the flag, and a restart
	// flips it back to true before the sleeper wakes, so a stale goroutine
	// would otherwise scan concurrently with the new one.
	stop := m.scanDone
	m.mutex.Unlock()

	// Parse the GoPro service UUID once for scan filtering
	goProServiceUUID, _ := bluetooth.ParseUUID(AdvertisementService)

	// Start scanning in background and restart if interrupted
	go func() {
		m.log.Debug("Starting BLE scan for GoPro devices")
		// First failure and recovery log once; retries back off silently so a
		// dead adapter cannot flood the rotating log.
		retryDelay := scanRetryBackoff
		scanFailing := false
		for {
			select {
			case <-stop:
				m.log.Debug("BLE scan goroutine stopped")
				return
			default:
			}
			// run scan session (blocks until StopScan or error)
			scanErr := m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
				// Primary: check for FEA6 service UUID per OpenGoPro spec
				// Fallback: name-based check for older cameras that may not advertise the UUID
				if result.HasServiceUUID(goProServiceUUID) ||
					strings.Contains(strings.ToLower(result.LocalName()), "gopro") {
					m.addDiscoveredDevice(result, parseAdvertisement(result, goProServiceUUID))
				}
			})
			if scanErr != nil {
				if !scanFailing {
					m.log.Warn("BLE scan failed, retrying with backoff", "err", scanErr)
					scanFailing = true
				}
				select {
				case <-stop:
					m.log.Debug("BLE scan goroutine stopped")
					return
				case <-time.After(retryDelay):
				}
				if retryDelay *= 2; retryDelay > time.Minute {
					retryDelay = time.Minute
				}
				continue
			}
			if scanFailing {
				m.log.Info("BLE scan recovered")
				scanFailing = false
			}
			retryDelay = scanRetryBackoff
			// loop to restart scanning if still enabled
		}
	}()
	return nil
}

// StopScanning stops the current scan
func (m *Manager) StopScanning() error {
	m.mutex.Lock()
	if !m.isScanning {
		m.mutex.Unlock()
		return nil
	}
	m.isScanning = false
	if m.scanDone != nil {
		close(m.scanDone)
		m.scanDone = nil
	}
	m.discoveryCallback = nil
	m.mutex.Unlock()

	m.log.Debug("Stopping BLE scan")
	if m.adapter == nil {
		return nil
	}
	return m.adapter.StopScan()
}

// ScanDone returns a channel that is closed when StopScanning() is called.
// Never nil: with no scan active it returns an already-closed channel.
func (m *Manager) ScanDone() <-chan struct{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.scanDone == nil {
		return closedChan
	}
	return m.scanDone
}

func (m *Manager) addDiscoveredDevice(result bluetooth.ScanResult, adv AdvInfo) {
	macAddress := result.Address.String()
	localName := result.LocalName()
	rssi := int32(result.RSSI)
	now := time.Now()

	m.mutex.Lock()
	device, exists := m.discoveredDevices[macAddress]
	if exists {
		// EMA smoothing to reduce RSSI jitter between advertisements
		const emaAlpha = 0.15
		device.RSSI = int32(emaAlpha*float64(rssi) + (1-emaAlpha)*float64(device.RSSI))
		device.LastSeen = now
		if device.Name == "" || device.Name != localName {
			device.Name = localName
		}
	} else {
		device = &Device{
			Name:       localName,
			BLEAddress: macAddress,
			RSSI:       rssi,
			LastSeen:   now,
		}
		m.discoveredDevices[macAddress] = device
		// The user-visible discovery line belongs to the discovery processor.
		// The advertisement flags are logged too: without them a flat camera
		// and a camera refusing its bond are the same line, and telling them
		// apart meant reading hours of log by hand.
		m.log.Debug("New GoPro advertisement seen",
			"camera", device.Name, "ble_addr", device.BLEAddress, "rssi", device.RSSI,
			"processor_on", adv.ProcessorOn, "pairing_mode", adv.PairingMode,
			"wifi_ap_on", adv.WiFiAPOn, "new_media", adv.NewMedia, "adv_valid", adv.Valid)
	}
	if adv.Valid {
		// Processor state flips are the interesting transition: a camera that
		// stops answering connects while still advertising has almost always
		// just powered its processor down.
		if device.AdvParsed && device.ProcessorOn != adv.ProcessorOn {
			m.log.Info("Camera processor state changed",
				"camera", device.Name, "ble_addr", device.BLEAddress,
				"processor_on", adv.ProcessorOn)
		}
		device.ProcessorOn = adv.ProcessorOn
		device.AdvParsed = true
		device.PairingMode = adv.PairingMode
		device.NewMedia = adv.NewMedia
		if device.ModelID == 0 && adv.ModelID > 0 {
			device.ModelID = adv.ModelID
		}
	}
	if device.SerialNumber == "" && adv.SerialNumber != "" {
		device.SerialNumber = adv.SerialNumber
	}
	deviceCopy := *device
	callback := m.discoveryCallback
	m.mutex.Unlock()

	// Synchronous, outside the lock: the processing path is in-memory only,
	// and a goroutine per advertisement previously grew unbounded.
	if callback != nil {
		callback(deviceCopy)
	}
}
