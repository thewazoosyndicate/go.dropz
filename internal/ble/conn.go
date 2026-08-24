package ble

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/ble/tlv"
	"tinygo.org/x/bluetooth"
)

// beginConnecting raises the connect gate so the scanner waits before
// restarting; the returned release is idempotent and safe to defer.
// Counted rather than a single channel: status checks and syncs connect
// concurrently, and an overwritten channel strands the scanner forever.
func (m *Manager) beginConnecting() (release func()) {
	m.mutex.Lock()
	if m.connectingCount == 0 {
		m.connectingDone = make(chan struct{})
	}
	m.connectingCount++
	m.mutex.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			m.mutex.Lock()
			m.connectingCount--
			if m.connectingCount == 0 {
				close(m.connectingDone)
				m.connectingDone = nil
			}
			m.mutex.Unlock()
		})
	}
}

// ConnectingDone returns a channel that is closed when all in-flight connect
// attempts have finished GATT setup, or nil when none are in flight.
// The scanner should wait on this before restarting to avoid BlueZ conflicts.
func (m *Manager) ConnectingDone() <-chan struct{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.connectingDone == nil {
		return nil
	}
	return m.connectingDone
}

// connectAndDiscover handles the shared BLE connect + service discovery retry loop.
// preDiscoveryHook runs after adapter.Connect() but before DiscoverServices() on each attempt
// (used by pairing to insert D-Bus bonding). Returns the discovered services.
// On success, the connection is stored in conns. On failure, cleanup is done.
// Returns errAlreadyConnected (a no-op for callers) when a connection exists.
// The caller owns the connect gate (beginConnecting) around this call.
func (m *Manager) connectAndDiscover(macAddress string, preDiscoveryHook func(string) error) ([]bluetooth.DeviceService, time.Time, error) {
	if m.adapter == nil {
		return nil, time.Time{}, ErrBluetoothUnavailable
	}

	if err := validateAddress(macAddress); err != nil {
		return nil, time.Time{}, fmt.Errorf("invalid device address: %w", err)
	}

	if m.getConn(macAddress) != nil {
		m.log.Debug("Device already connected", "ble_addr", macAddress)
		return nil, time.Time{}, errAlreadyConnected
	}

	connectStart := time.Now()

	_ = m.StopScanning()
	time.Sleep(ScanToConnectDelay)

	addr, parseErr := parseAddress(macAddress)
	if parseErr != nil {
		return nil, time.Time{}, parseErr
	}

	m.log.Debug("Connecting to GoPro device", "ble_addr", macAddress)

	// Retry loop: full disconnect+reconnect between attempts (BlueZ won't
	// recover a stale GATT handle, so re-calling DiscoverServices is useless).
	var device bluetooth.Device
	var services []bluetooth.DeviceService
	var connected, everConnected bool
	var lastConnErr error

	for retry := 0; retry < ServiceDiscoveryRetries; retry++ {
		if retry > 0 {
			m.log.Debug("Reconnecting for service discovery retry", "ble_addr", macAddress, "attempt", retry+1, "max", ServiceDiscoveryRetries)
			if connected {
				_ = device.Disconnect()
				connected = false
			}
			m.dropConn(macAddress)
			time.Sleep(time.Duration(retry+1) * time.Second)
		}

		var connErr error
		device, connErr = m.adapter.Connect(addr, bluetooth.ConnectionParams{})
		if connErr != nil {
			// Retries at Debug; the terminal failure is returned and the
			// caller owns the failure log
			m.log.Debug("Connection attempt failed", "ble_addr", macAddress, "attempt", retry+1, "err", connErr)
			lastConnErr = connErr
			continue
		}
		connected = true
		everConnected = true
		m.log.Debug("BLE connected", "ble_addr", macAddress, "attempt", retry+1, "elapsed", time.Since(connectStart))

		c := m.newConn(macAddress, &device)
		m.mutex.Lock()
		m.conns[macAddress] = c
		m.mutex.Unlock()

		if preDiscoveryHook != nil {
			if hookErr := preDiscoveryHook(macAddress); hookErr != nil {
				m.log.Warn("Pre-discovery hook failed", "ble_addr", macAddress, "err", hookErr)
			}
		}

		time.Sleep(PostConnectDelay)

		var discoverErr error
		services, discoverErr = device.DiscoverServices(nil)
		m.log.Debug("Service discovery result", "ble_addr", macAddress, "services", len(services), "err", discoverErr, "elapsed", time.Since(connectStart))

		if discoverErr == nil && len(services) > 0 {
			break
		}
		m.log.Debug("Service discovery attempt failed", "ble_addr", macAddress, "attempt", retry+1, "err", discoverErr)
	}

	if len(services) == 0 {
		m.dropConn(macAddress)
		if connected {
			_ = device.Disconnect()
		}
		if !everConnected {
			return nil, time.Time{}, m.connectFailure(macAddress, lastConnErr)
		}
		return nil, time.Time{}, fmt.Errorf("service discovery failed after %d retries", ServiceDiscoveryRetries)
	}

	m.resetConnectAborts(macAddress)
	return services, connectStart, nil
}

// connectFailure classifies a connect that never reached service discovery.
// Repeated link aborts against a camera that is still advertising mean the
// camera dropped its side of the bond; anything else stays generic.
func (m *Manager) connectFailure(macAddress string, lastErr error) error {
	if lastErr == nil {
		return fmt.Errorf("connect failed after %d retries", ServiceDiscoveryRetries)
	}
	if isConnectAbort(lastErr) {
		m.mutex.Lock()
		m.connectAborts[macAddress]++
		aborts := m.connectAborts[macAddress]
		var lastSeen time.Time
		if dev, ok := m.discoveredDevices[macAddress]; ok {
			lastSeen = dev.LastSeen
		}
		m.mutex.Unlock()
		if aborts >= bondLossAbortThreshold && time.Since(lastSeen) < bondLossSeenWindow {
			return fmt.Errorf("%d consecutive aborted connects while advertising: %w", aborts, ErrBondLost)
		}
	}
	return fmt.Errorf("connect failed after %d retries: %w", ServiceDiscoveryRetries, lastErr)
}

// resetConnectAborts clears the bond-loss counter after a working connect.
func (m *Manager) resetConnectAborts(macAddress string) {
	m.mutex.Lock()
	delete(m.connectAborts, macAddress)
	m.mutex.Unlock()
}

// discoverCharacteristics enumerates characteristics on the given services,
// caching those in cacheChars and enabling notifications on notifyChars.
// All BLE I/O runs outside the manager lock: EnableNotifications is a BlueZ
// round-trip, and holding the write lock across it stalls notification
// routing for every other camera.
func (m *Manager) discoverCharacteristics(macAddress string, services []bluetooth.DeviceService, serviceFilter map[string]bool, cacheChars map[string]bool, notifyChars map[string]bool, connectStart time.Time) {
	c := m.getConn(macAddress)
	if c == nil {
		m.log.Warn("Characteristic discovery skipped, not connected", "ble_addr", macAddress)
		return
	}

	newChars := make(map[string]bluetooth.DeviceCharacteristic)
	newCollectors := make(map[string]*tlv.FragmentCollector)
	var subscribed []bluetooth.DeviceCharacteristic

	for _, service := range services {
		serviceUUID := service.UUID().String()
		if !serviceFilter[serviceUUID] {
			continue
		}

		chars, charErr := service.DiscoverCharacteristics(nil)
		if charErr != nil {
			m.log.Warn("Failed to discover characteristics", "ble_addr", macAddress, "service", serviceUUID, "err", charErr)
			continue
		}

		for _, char := range chars {
			charUUID := char.UUID().String()
			if cacheChars[charUUID] {
				newChars[charUUID] = char
			}
			if notifyChars[charUUID] {
				collector := newCollectors[charUUID]
				if collector == nil {
					collector = tlv.NewFragmentCollector(fragmentTimeout)
					collector.SetLogger(m.log.With("ble_addr", macAddress, "char", GetCharacteristicName(charUUID)))
					newCollectors[charUUID] = collector
				}
				notifErr := char.EnableNotifications(func(data []byte) {
					m.handleNotification(macAddress, collector, data)
				})
				if notifErr != nil {
					m.log.Warn("Failed to enable notifications", "ble_addr", macAddress, "char", GetCharacteristicName(charUUID), "err", notifErr)
				} else {
					subscribed = append(subscribed, char)
				}
			}
		}
	}

	m.mutex.Lock()
	// dropConn may have removed this conn mid-setup (shutdown, camera drop);
	// committing then would leak subscriptions past its cleanup snapshot.
	current := m.conns[macAddress] == c
	if current {
		for k, v := range newChars {
			c.chars[k] = v
		}
		for k, v := range newCollectors {
			c.collectors[k] = v
		}
		c.subscribed = append(c.subscribed, subscribed...)
	}
	m.mutex.Unlock()

	if !current {
		for _, char := range subscribed {
			_ = char.EnableNotifications(nil)
		}
		for _, collector := range newCollectors {
			collector.Stop()
		}
		m.log.Debug("Connection dropped during characteristic setup", "ble_addr", macAddress)
		return
	}

	m.log.Debug("Characteristic discovery complete", "ble_addr", macAddress, "elapsed", time.Since(connectStart))
}

// dropConn removes connection state for a device and stops its collector.
func (m *Manager) dropConn(macAddress string) *conn {
	m.mutex.Lock()
	c := m.conns[macAddress]
	delete(m.conns, macAddress)
	// Snapshot under the lock: discoverCharacteristics appends to these
	// under the same lock, so iterating after release would race a
	// connect in flight.
	var subscribed []bluetooth.DeviceCharacteristic
	var collectors []*tlv.FragmentCollector
	if c != nil {
		subscribed = append(subscribed, c.subscribed...)
		for _, collector := range c.collectors {
			collectors = append(collectors, collector)
		}
	}
	m.mutex.Unlock()

	// Best effort: on a dead link the call fails fast, and the device
	// object teardown removes the watcher anyway.
	for _, char := range subscribed {
		if err := char.EnableNotifications(nil); err != nil {
			m.log.Debug("Unsubscribe failed", "err", err)
		}
	}
	for _, collector := range collectors {
		collector.Stop()
	}
	return c
}

// cleanupOnError removes connection state for a device. Used as a deferred cleanup.
func (m *Manager) cleanupOnError(macAddress string) {
	if c := m.dropConn(macAddress); c != nil {
		_ = c.device.Disconnect()
		m.log.Debug("Connection state dropped after error", "ble_addr", macAddress)
	}
}

// connectBase establishes a BLE connection, discovers services/characteristics,
// and polls until the camera is ready.
func (m *Manager) connectBase(macAddress string) (hwInfo *HardwareInfo, err error) {
	// Gate held until full GATT setup is done; releasing earlier restarts
	// the scan mid-session and breaks notifications on macOS.
	release := m.beginConnecting()
	defer release()

	services, connectStart, connErr := m.connectAndDiscover(macAddress, nil)
	if errors.Is(connErr, errAlreadyConnected) {
		return nil, nil
	}
	if connErr != nil {
		return nil, connErr
	}

	defer func() {
		if err != nil {
			m.cleanupOnError(macAddress)
		}
	}()

	m.discoverCharacteristics(macAddress, services, map[string]bool{
		ServiceWifiAP: true, ServiceControl: true, ServiceCameraMgmt: true,
	}, map[string]bool{
		CharWifiSSID: true, CharWifiPassword: true,
		CharCommand: true, CharCommandResponse: true,
		CharSettings: true, CharSettingsResponse: true,
		CharQuery: true, CharQueryResponse: true,
		CharNetworkMgmtCommand: true, CharNetworkMgmtResponse: true,
	}, map[string]bool{
		CharCommandResponse: true, CharQueryResponse: true,
		CharSettingsResponse: true, CharNetworkMgmtResponse: true,
	}, connectStart)

	// Poll GetHardwareInfo until camera is ready
	for attempt := 1; attempt <= cameraReadyAttempts; attempt++ {
		hwInfo, err = m.GetHardwareInfo(macAddress)
		if err == nil && hwInfo != nil {
			break
		}
		if attempt == cameraReadyAttempts {
			err = fmt.Errorf("camera not ready after %d attempts: %w", attempt, err)
			return nil, err
		}
		time.Sleep(time.Second)
	}

	m.log.Debug("Camera ready", "ble_addr", macAddress, "elapsed", time.Since(connectStart))

	if tpErr := m.SetThirdPartyClient(macAddress); tpErr != nil {
		m.log.Warn("Failed to set third party client flag", "ble_addr", macAddress, "err", tpErr)
	}

	if dtErr := m.SetLocalDateTime(macAddress, time.Now()); dtErr != nil {
		m.log.Warn("Failed to set camera date/time", "ble_addr", macAddress, "err", dtErr)
	}

	return hwInfo, nil
}

// finishConnection performs post-connect setup: metadata collection and callback.
func (m *Manager) finishConnection(macAddress string, hwInfo *HardwareInfo, ssid, password string) {
	metadata := CameraMetadata{
		BLEAddress:   macAddress,
		WiFiSSID:     ssid,
		WiFiPassword: password,
	}

	if hwInfo != nil {
		metadata.ModelID = hwInfo.ModelNumber
		metadata.ModelName = hwInfo.ModelName
		metadata.FirmwareVersion = hwInfo.FirmwareVersion
		metadata.SerialNumber = hwInfo.SerialNumber

		m.mutex.Lock()
		if dev, exists := m.discoveredDevices[macAddress]; exists {
			dev.ModelID = hwInfo.ModelNumber
			dev.ModelName = hwInfo.ModelName
			dev.FirmwareVersion = hwInfo.FirmwareVersion
			dev.SerialNumber = hwInfo.SerialNumber
			dev.WiFiSSID = ssid
			dev.WiFiPassword = password
		}
		m.mutex.Unlock()
	}

	battery, err := m.GetBatteryLevel(macAddress)
	if err != nil {
		m.log.Warn("Failed to read battery level", "ble_addr", macAddress, "err", err)
	} else {
		metadata.BatteryLevel = battery
	}

	m.mutex.RLock()
	callback := m.metadataCallback
	m.mutex.RUnlock()

	if callback != nil {
		callback(metadata)
	}
}

// Connect establishes a full connection: BLE + services + WiFi AP + credentials + metadata
func (m *Manager) Connect(macAddress string) (err error) {
	hwInfo, err := m.connectBase(macAddress)
	if err != nil {
		return err
	}
	if hwInfo == nil {
		return nil // already connected
	}

	defer func() {
		if err != nil {
			m.cleanupOnError(macAddress)
		}
	}()

	if err = m.SetAPControl(macAddress, WiFiAPModeEnable); err != nil {
		return fmt.Errorf("failed to enable WiFi AP: %w", err)
	}

	if regErr := m.RegisterStatusUpdates(macAddress, []byte{StatusBatteryPercentage}); regErr != nil {
		m.log.Warn("Failed to register status push notifications", "ble_addr", macAddress, "err", regErr)
	}

	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		err = fmt.Errorf("failed to get WiFi credentials: %w", credErr)
		return err
	}
	m.finishConnection(macAddress, hwInfo, ssid, password)
	return nil
}

// ConnectForStatusCheck establishes a lightweight BLE connection for querying statuses.
// Only discovers the Control service (Command/Query chars) - skips GetHardwareInfo,
// SetThirdPartyClient, and SetLocalDateTime.
func (m *Manager) ConnectForStatusCheck(macAddress string) error {
	release := m.beginConnecting()
	defer release()

	services, connectStart, err := m.connectAndDiscover(macAddress, nil)
	if errors.Is(err, errAlreadyConnected) {
		return nil
	}
	if err != nil {
		return err
	}

	// Settings chars included: keep-alive and setting reads/writes go to
	// GP-0074/0075 and must work on this lightweight connection too.
	m.discoverCharacteristics(macAddress, services, map[string]bool{
		ServiceControl: true,
	}, map[string]bool{
		CharCommand: true, CharCommandResponse: true,
		CharQuery: true, CharQueryResponse: true,
		CharSettings: true, CharSettingsResponse: true,
	}, map[string]bool{
		CharCommandResponse: true, CharQueryResponse: true,
		CharSettingsResponse: true,
	}, connectStart)

	return nil
}

// ConnectForPairing establishes a minimal BLE connection for pairing.
// Only discovers WiFi AP and Camera Management services - skips the control
// service (which hangs on BlueZ) since pairing doesn't need commands/queries.
func (m *Manager) ConnectForPairing(macAddress string) (err error) {
	release := m.beginConnecting()
	defer release()

	// D-Bus bonding runs between connect and service discovery on each attempt
	services, connectStart, connErr := m.connectAndDiscover(macAddress, func(addr string) error {
		if pairErr := m.pairViaDbus(addr); pairErr != nil {
			m.log.Warn("D-Bus pairing failed", "ble_addr", addr, "err", pairErr)
		}
		return nil
	})
	if errors.Is(connErr, errAlreadyConnected) {
		return nil
	}
	if connErr != nil {
		return connErr
	}

	defer func() {
		if err != nil {
			m.cleanupOnError(macAddress)
		}
	}()

	m.discoverCharacteristics(macAddress, services, map[string]bool{
		ServiceWifiAP: true, ServiceCameraMgmt: true,
	}, map[string]bool{
		CharWifiSSID: true, CharWifiPassword: true,
		CharNetworkMgmtCommand: true, CharNetworkMgmtResponse: true,
	}, map[string]bool{
		CharNetworkMgmtResponse: true,
	}, connectStart)

	// Synchronous: HERO11 never answers (times out, harmless), but HERO13
	// completes the exchange; firing it in a goroutine raced the
	// post-pairing disconnect, the camera never saw pairing finish, and
	// it dropped the bond at power-off (connects then abort forever).
	if err := m.SendPairingFinish(macAddress); err != nil {
		m.log.Debug("Pairing finish not accepted", "ble_addr", macAddress, "err", err)
	} else {
		m.log.Debug("Pairing finish acknowledged", "ble_addr", macAddress)
	}

	// Read WiFi credentials (now accessible after D-Bus bonding)
	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		m.log.Warn("Failed to read WiFi credentials during pairing", "ble_addr", macAddress, "err", credErr)
	}

	if ssid != "" && password != "" {
		m.mutex.Lock()
		if dev, exists := m.discoveredDevices[macAddress]; exists {
			dev.WiFiSSID = ssid
			dev.WiFiPassword = password
		}
		m.mutex.Unlock()

		m.mutex.RLock()
		callback := m.metadataCallback
		m.mutex.RUnlock()
		if callback != nil {
			callback(CameraMetadata{
				BLEAddress:   macAddress,
				WiFiSSID:     ssid,
				WiFiPassword: password,
			})
		}
	}

	m.log.Debug("Pairing connection ready", "ble_addr", macAddress, "elapsed", time.Since(connectStart))
	return nil
}

// Disconnect closes the connection to a device, sending Sleep first.
func (m *Manager) Disconnect(macAddress string) error {
	if m.getConn(macAddress) == nil {
		return fmt.Errorf("device not connected: %s", macAddress)
	}

	// A failed Sleep leaves the camera awake and draining battery
	if err := m.Sleep(macAddress); err != nil {
		m.log.Warn("Sleep failed, camera left awake", "ble_addr", macAddress, "err", err)
	}

	if c := m.dropConn(macAddress); c != nil {
		_ = c.device.Disconnect()
	}
	m.log.Debug("Disconnected", "ble_addr", macAddress)
	return nil
}

// DisconnectQuietly closes the BLE connection without sending Sleep - camera stays awake.
func (m *Manager) DisconnectQuietly(macAddress string) error {
	c := m.dropConn(macAddress)
	if c == nil {
		return fmt.Errorf("device not connected: %s", macAddress)
	}
	_ = c.device.Disconnect()
	m.log.Debug("Disconnected, camera left awake", "ble_addr", macAddress)
	return nil
}
