package ble

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/ble/tlv"
	"tinygo.org/x/bluetooth"
)

// errAlreadyConnected signals a no-op connect; callers treat it as success.
var errAlreadyConnected = errors.New("device already connected")

const (
	responseTimeout     = 5 * time.Second  // per-command TLV response wait
	fragmentTimeout     = 10 * time.Second // stale multi-packet message cleanup
	cameraReadyAttempts = 10               // GetHardwareInfo polls after connect
	scanRetryBackoff    = 2 * time.Second  // wait after a failed adapter scan
)

// conn holds the per-device connection state. Each device gets its own
// fragment collector and response tracker so concurrent traffic from two
// cameras can never collide on command IDs.
type conn struct {
	device    *bluetooth.Device
	chars     map[string]bluetooth.DeviceCharacteristic
	collector *tlv.FragmentCollector
	tracker   *tlv.ResponseTracker
}

// Manager provides a clean, simple BLE interface for GoPro devices
// following the OpenGoPro BLE specification exactly
type Manager struct {
	adapter           *bluetooth.Adapter
	discoveredDevices map[string]*Device // discovered devices during scanning
	conns             map[string]*conn   // active connections by MAC address
	mutex             sync.RWMutex
	log               *slog.Logger
	isScanning        bool
	scanDone          chan struct{}                                        // closed when StopScanning() is called
	connectingDone    chan struct{}                                        // closed when connect finishes, so scanner waits
	discoveryCallback DeviceDiscoveryCallback                              // callback for live discovery updates
	metadataCallback  MetadataUpdateFunc                                   // callback for metadata updates during connection
	statusCallback    func(macAddress string, statusID byte, value []byte) // push notification callback
}

// parseTLVPairs parses a byte sequence of [ID][Length][Value...] triplets into a map.
func parseTLVPairs(data []byte) map[byte][]byte {
	result := make(map[byte][]byte)
	offset := 0
	for offset+2 <= len(data) {
		id := data[offset]
		length := int(data[offset+1])
		offset += 2
		if offset+length > len(data) {
			break
		}
		value := make([]byte, length)
		copy(value, data[offset:offset+length])
		result[id] = value
		offset += length
	}
	return result
}

// NewManager creates a new BLE manager
func NewManager(adapter *bluetooth.Adapter, log *slog.Logger) *Manager {
	return &Manager{
		adapter:           adapter,
		discoveredDevices: make(map[string]*Device),
		conns:             make(map[string]*conn),
		log:               log.With("component", "ble"),
	}
}

// newConn creates the per-device connection state and wires push
// notifications (0x93) through to the manager-level status callback.
func (m *Manager) newConn(macAddress string, device *bluetooth.Device) *conn {
	c := &conn{
		device:    device,
		chars:     make(map[string]bluetooth.DeviceCharacteristic),
		collector: tlv.NewFragmentCollector(fragmentTimeout),
		tracker:   tlv.NewResponseTracker(),
	}
	c.tracker.SetPushHandler(func(addr string, msg *tlv.TLVMessage) {
		m.mutex.RLock()
		cb := m.statusCallback
		m.mutex.RUnlock()
		if cb == nil {
			return
		}
		for id, value := range parseTLVPairs(msg.Payload) {
			cb(macAddress, id, value)
		}
	})
	return c
}

// getConn returns the connection for a device, if any.
func (m *Manager) getConn(macAddress string) *conn {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.conns[macAddress]
}

// StartScanningWithCallback scans for GoPro devices with optional live discovery callback
func (m *Manager) StartScanningWithCallback(ctx context.Context, callback DeviceDiscoveryCallback) error {
	m.mutex.Lock()
	if m.isScanning {
		m.mutex.Unlock()
		return nil
	}
	m.isScanning = true
	m.scanDone = make(chan struct{})
	m.discoveryCallback = callback
	m.mutex.Unlock()

	// Parse the GoPro service UUID once for scan filtering
	goProServiceUUID, _ := bluetooth.ParseUUID(AdvertisementService)

	// Start scanning in background and restart if interrupted
	go func() {
		m.log.Info("Starting BLE scan for GoPro devices")
		for {
			m.mutex.RLock()
			if !m.isScanning {
				m.mutex.RUnlock()
				break
			}
			m.mutex.RUnlock()
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
				// Backoff so a dead adapter doesn't turn this into a spin loop
				m.log.Warn("BLE scan failed", "err", scanErr)
				time.Sleep(scanRetryBackoff)
			}
			// loop to restart scanning if still enabled
		}
		m.log.Info("Stopped BLE scanning")
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

	m.log.Info("Stopping BLE scan")
	return m.adapter.StopScan()
}

// ScanDone returns a channel that is closed when StopScanning() is called.
func (m *Manager) ScanDone() <-chan struct{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.scanDone
}

// ConnectingDone returns a channel that is closed when a connect attempt finishes.
// The scanner should wait on this before restarting to avoid BlueZ conflicts.
func (m *Manager) ConnectingDone() <-chan struct{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.connectingDone
}

// signalConnectingDone closes the connectingDone channel so the scanner can
// resume. Idempotent — safe to call multiple times or from defer.
func (m *Manager) signalConnectingDone() {
	m.mutex.Lock()
	if m.connectingDone != nil {
		close(m.connectingDone)
		m.connectingDone = nil
	}
	m.mutex.Unlock()
}

// connectAndDiscover handles the shared BLE connect + service discovery retry loop.
// preDiscoveryHook runs after adapter.Connect() but before DiscoverServices() on each attempt
// (used by pairing to insert D-Bus bonding). Returns the discovered services.
// On success, the connection is stored in conns. On failure, cleanup is done.
// Returns errAlreadyConnected (a no-op for callers) when a connection exists.
func (m *Manager) connectAndDiscover(macAddress string, preDiscoveryHook func(string) error) ([]bluetooth.DeviceService, time.Time, error) {
	if err := validateAddress(macAddress); err != nil {
		return nil, time.Time{}, fmt.Errorf("invalid device address: %w", err)
	}

	if m.getConn(macAddress) != nil {
		m.log.Debug("Device already connected", "device", macAddress)
		return nil, time.Time{}, errAlreadyConnected
	}

	connectStart := time.Now()

	// Signal scanner to wait until connection setup is complete before
	// restarting. Resumed by the public Connect* methods once their full
	// GATT setup is done, and by the error paths below; resuming earlier
	// restarts the scan mid-session and breaks notifications on macOS.
	m.mutex.Lock()
	m.connectingDone = make(chan struct{})
	m.mutex.Unlock()

	m.log.Debug("Stopping BLE scan for GATT connection")
	m.StopScanning()
	time.Sleep(ScanToConnectDelay)

	addr, parseErr := parseAddress(macAddress)
	if parseErr != nil {
		m.signalConnectingDone()
		return nil, time.Time{}, parseErr
	}

	m.log.Debug("Connecting to GoPro device", "device", macAddress)

	// Retry loop: full disconnect+reconnect between attempts (BlueZ won't
	// recover a stale GATT handle, so re-calling DiscoverServices is useless).
	var device bluetooth.Device
	var services []bluetooth.DeviceService
	var connected bool

	for retry := 0; retry < ServiceDiscoveryRetries; retry++ {
		if retry > 0 {
			m.log.Debug("Reconnecting for service discovery retry", "attempt", retry+1, "max", ServiceDiscoveryRetries)
			if connected {
				device.Disconnect()
				connected = false
			}
			m.dropConn(macAddress)
			time.Sleep(time.Duration(retry+1) * time.Second)
		}

		var connErr error
		device, connErr = m.adapter.Connect(addr, bluetooth.ConnectionParams{})
		if connErr != nil {
			m.log.Warn("Connection attempt failed", "attempt", retry+1, "err", connErr)
			continue
		}
		connected = true
		m.log.Debug("BLE connected", "attempt", retry+1, "elapsed", time.Since(connectStart))

		c := m.newConn(macAddress, &device)
		m.mutex.Lock()
		m.conns[macAddress] = c
		m.mutex.Unlock()

		if preDiscoveryHook != nil {
			if hookErr := preDiscoveryHook(macAddress); hookErr != nil {
				m.log.Warn("Pre-discovery hook failed", "err", hookErr)
			}
		}

		time.Sleep(PostConnectDelay)

		var discoverErr error
		services, discoverErr = device.DiscoverServices(nil)
		m.log.Debug("Service discovery result", "services", len(services), "err", discoverErr, "elapsed", time.Since(connectStart))

		if discoverErr == nil && len(services) > 0 {
			break
		}
		m.log.Warn("Service discovery attempt failed", "attempt", retry+1, "err", discoverErr)
	}

	if len(services) == 0 {
		m.dropConn(macAddress)
		if connected {
			device.Disconnect()
		}
		m.signalConnectingDone()
		return nil, time.Time{}, fmt.Errorf("service discovery failed after %d retries", ServiceDiscoveryRetries)
	}

	return services, connectStart, nil
}

// discoverCharacteristics enumerates characteristics on the given services,
// caching those in cacheChars and enabling notifications on notifyChars.
func (m *Manager) discoverCharacteristics(macAddress string, services []bluetooth.DeviceService, serviceFilter map[string]bool, cacheChars map[string]bool, notifyChars map[string]bool, connectStart time.Time) {
	c := m.getConn(macAddress)
	if c == nil {
		m.log.Warn("Characteristic discovery skipped: not connected", "device", macAddress)
		return
	}

	for _, service := range services {
		serviceUUID := service.UUID().String()
		if !serviceFilter[serviceUUID] {
			continue
		}

		chars, charErr := service.DiscoverCharacteristics(nil)
		if charErr != nil {
			m.log.Warn("Failed to discover characteristics", "service", serviceUUID, "err", charErr)
			continue
		}

		m.mutex.Lock()
		for _, char := range chars {
			charUUID := char.UUID().String()
			if cacheChars[charUUID] {
				c.chars[charUUID] = char
			}
			if notifyChars[charUUID] {
				notifErr := char.EnableNotifications(func(data []byte) {
					m.handleNotification(macAddress, data)
				})
				if notifErr != nil {
					m.log.Warn("Failed to enable notifications", "char", GetCharacteristicName(charUUID), "err", notifErr)
				}
			}
		}
		m.mutex.Unlock()
	}

	m.log.Debug("Characteristic discovery complete", "elapsed", time.Since(connectStart))
}

// dropConn removes connection state for a device and stops its collector.
func (m *Manager) dropConn(macAddress string) *conn {
	m.mutex.Lock()
	c := m.conns[macAddress]
	delete(m.conns, macAddress)
	m.mutex.Unlock()

	if c != nil {
		c.collector.Stop()
	}
	m.signalConnectingDone()
	return c
}

// cleanupOnError removes connection state for a device. Used as a deferred cleanup.
func (m *Manager) cleanupOnError(macAddress string) {
	if c := m.dropConn(macAddress); c != nil {
		c.device.Disconnect()
	}
}

// connectBase establishes a BLE connection, discovers services/characteristics,
// and polls until the camera is ready.
func (m *Manager) connectBase(macAddress string) (hwInfo *HardwareInfo, err error) {
	defer m.signalConnectingDone()
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

	m.log.Debug("Camera ready", "elapsed", time.Since(connectStart))

	if tpErr := m.SetThirdPartyClient(macAddress); tpErr != nil {
		m.log.Warn("Failed to set third party client flag", "err", tpErr)
	}

	if dtErr := m.SetLocalDateTime(macAddress, time.Now()); dtErr != nil {
		m.log.Warn("Failed to set camera date/time", "err", dtErr)
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
		m.log.Warn("Failed to read battery level", "err", err)
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

// ConnectForStatusCheck establishes a lightweight BLE connection for querying statuses.
// Only discovers the Control service (Command/Query chars) — skips GetHardwareInfo,
// SetThirdPartyClient, and SetLocalDateTime.
func (m *Manager) ConnectForStatusCheck(macAddress string) error {
	defer m.signalConnectingDone()
	services, connectStart, err := m.connectAndDiscover(macAddress, nil)
	if errors.Is(err, errAlreadyConnected) {
		return nil
	}
	if err != nil {
		return err
	}

	m.discoverCharacteristics(macAddress, services, map[string]bool{
		ServiceControl: true,
	}, map[string]bool{
		CharCommand: true, CharCommandResponse: true,
		CharQuery: true, CharQueryResponse: true,
	}, map[string]bool{
		CharCommandResponse: true, CharQueryResponse: true,
	}, connectStart)

	return nil
}

// QueryStatuses queries multiple status IDs in a single BLE request and returns parsed TLV results.
func (m *Manager) QueryStatuses(macAddress string, statusIDs []byte) (map[byte][]byte, error) {
	response, err := m.sendQuery(macAddress, QueryGetStatus, statusIDs)
	if err != nil {
		return nil, err
	}
	return parseTLVPairs(response.Data), nil
}

// DisconnectQuietly closes the BLE connection without sending Sleep — camera stays awake.
func (m *Manager) DisconnectQuietly(macAddress string) error {
	c := m.dropConn(macAddress)
	if c == nil {
		return fmt.Errorf("device not connected: %s", macAddress)
	}
	c.device.Disconnect()
	return nil
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
		m.log.Warn("Failed to register status push notifications", "err", regErr)
	}

	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		err = fmt.Errorf("failed to get WiFi credentials: %w", credErr)
		return err
	}
	m.finishConnection(macAddress, hwInfo, ssid, password)
	return nil
}

// ConnectForPairing establishes a minimal BLE connection for pairing.
// Only discovers WiFi AP and Camera Management services — skips the control
// service (which hangs on BlueZ) since pairing doesn't need commands/queries.
func (m *Manager) ConnectForPairing(macAddress string) (err error) {
	defer m.signalConnectingDone()
	// D-Bus bonding runs between connect and service discovery on each attempt
	services, connectStart, connErr := m.connectAndDiscover(macAddress, func(addr string) error {
		if pairErr := m.pairViaDbus(addr); pairErr != nil {
			m.log.Warn("D-Bus pairing failed", "err", pairErr)
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

	// Fire-and-forget — camera never responds to 0x03
	go func() {
		if err := m.SendPairingFinish(macAddress); err != nil {
			m.log.Debug("SendPairingFinish (best-effort)", "err", err)
		}
	}()

	// Read WiFi credentials (now accessible after D-Bus bonding)
	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		m.log.Warn("Failed to read WiFi credentials during pairing", "err", credErr)
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

	m.log.Debug("Pairing connection ready", "elapsed", time.Since(connectStart))
	return nil
}

// Disconnect closes the connection to a device, sending Sleep first.
func (m *Manager) Disconnect(macAddress string) error {
	if m.getConn(macAddress) == nil {
		return fmt.Errorf("device not connected: %s", macAddress)
	}

	if err := m.Sleep(macAddress); err != nil {
		m.log.Debug("Sleep failed", "device", macAddress, "err", err)
	} else {
		m.log.Debug("Sleep command succeeded", "device", macAddress)
	}

	if c := m.dropConn(macAddress); c != nil {
		c.device.Disconnect()
	}
	return nil
}

// GetWifiCredentials retrieves WiFi SSID and password from the device
func (m *Manager) GetWifiCredentials(macAddress string) (string, string, error) {
	c := m.getConn(macAddress)
	if c == nil {
		return "", "", fmt.Errorf("device not connected: %s", macAddress)
	}

	m.mutex.RLock()
	ssidChar, ssidOK := c.chars[CharWifiSSID]
	passChar, passOK := c.chars[CharWifiPassword]
	m.mutex.RUnlock()

	if !ssidOK {
		return "", "", fmt.Errorf("WiFi SSID characteristic not found")
	}

	ssidData := make([]byte, 32)
	n, err := ssidChar.Read(ssidData)
	if err != nil {
		return "", "", fmt.Errorf("failed to read SSID: %w", err)
	}
	ssid := strings.TrimSpace(string(ssidData[:n]))

	if !passOK {
		return "", "", fmt.Errorf("WiFi password characteristic not found")
	}

	passData := make([]byte, 64)
	n, err = passChar.Read(passData)
	if err != nil {
		return "", "", fmt.Errorf("failed to read password: %w", err)
	}
	password := strings.TrimSpace(string(passData[:n]))

	return ssid, password, nil
}

// GetBatteryLevel queries the battery level from the device
func (m *Manager) GetBatteryLevel(macAddress string) (int, error) {
	statuses, err := m.QueryStatuses(macAddress, []byte{StatusBatteryPercentage})
	if err != nil {
		return 0, err
	}
	if v, ok := statuses[StatusBatteryPercentage]; ok && len(v) >= 1 {
		return int(v[0]), nil
	}
	return 0, fmt.Errorf("battery percentage not found in response")
}

// KeepAlive sends a keep-alive to prevent the camera from auto-sleeping.
// Per OpenGoPro this is the LED setting (91) set to 66, written to the
// Settings characteristic; it was previously sent to the Command
// characteristic, which cameras reject.
func (m *Manager) KeepAlive(macAddress string) error {
	_, err := m.sendSetting(macAddress, SettingKeepAlive, []byte{0x01, KeepAliveValue})
	return err
}

// Sleep puts the device to sleep
func (m *Manager) Sleep(macAddress string) error {
	resp, err := m.sendCommand(macAddress, CmdSleep, nil)
	if err != nil {
		return err
	}
	if resp.Status != 0 {
		return fmt.Errorf("sleep command returned status %d", resp.Status)
	}
	return nil
}

// Private helper methods

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
		m.log.Info("Discovered new GoPro device", "name", device.Name, "device", device.BLEAddress, "rssi", device.RSSI)
	}
	if adv.Valid {
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

// sendMessage sends a TLV message on the given characteristic and waits for a response.
func (m *Manager) sendMessage(macAddress string, charUUID string, id byte, data []byte, buildPacket func(byte, []byte) []byte) (Response, error) {
	c := m.getConn(macAddress)
	if c == nil {
		return Response{}, fmt.Errorf("device not connected")
	}

	m.mutex.RLock()
	char, exists := c.chars[charUUID]
	m.mutex.RUnlock()
	if !exists {
		return Response{}, fmt.Errorf("characteristic %s not found", GetCharacteristicName(charUUID))
	}

	responseChan := c.tracker.RegisterCommand(id)
	defer c.tracker.UnregisterCommand(id)

	packets := tlv.SplitIntoPackets(buildPacket(id, data))

	for i, pkt := range packets {
		if _, err := writeCharacteristic(char, pkt); err != nil {
			return Response{}, fmt.Errorf("failed to send packet %d: %w", i, err)
		}
	}

	select {
	case message := <-responseChan:
		return Response{
			Status: message.Status,
			Data:   message.Payload,
		}, nil
	case <-time.After(responseTimeout):
		m.log.Warn("Command timed out", "id", fmt.Sprintf("0x%02X", id), "timeout", responseTimeout)
		return Response{}, fmt.Errorf("timeout waiting for response to 0x%02X", id)
	}
}

func (m *Manager) sendCommand(macAddress string, commandID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharCommand, commandID, data, tlv.BuildCommandPacket)
}

func (m *Manager) sendQuery(macAddress string, queryID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharQuery, queryID, data, tlv.BuildQueryPacket)
}

func (m *Manager) sendSetting(macAddress string, settingID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharSettings, settingID, data, tlv.BuildCommandPacket)
}

func (m *Manager) handleNotification(macAddress string, data []byte) {
	c := m.getConn(macAddress)
	if c == nil {
		return
	}

	message, err := c.collector.ProcessFragment(data)
	if err != nil {
		m.log.Debug("Error processing TLV fragment", "raw", fmt.Sprintf("%x", data), "err", err)
	} else if message != nil {
		if !c.tracker.RouteResponse(macAddress, message) {
			m.log.Debug("Unrouted notification", "cmd", fmt.Sprintf("0x%02x", message.CommandID), "status", message.Status, "len", len(message.Payload))
		}
	}
}

// buildProtobufPacket frames a protobuf message: packet header + [FeatureID][ActionID][protobuf].
// Protobuf messages use the same packetization as TLV commands (OpenGoPro
// data_protocol); the header was previously missing, so cameras parsed the
// feature ID byte as a length header and dropped the message.
func buildProtobufPacket(featureID byte, data []byte) []byte {
	return tlv.BuildTLVPackets(append([]byte{featureID}, data...))
}

// SendPairingFinish sends the RequestPairingFinish protobuf command (Feature 0x03, Action 0x01)
// to transition the camera to paired state.
func (m *Manager) SendPairingFinish(macAddress string) error {
	// Hand-encoded protobuf: field 1 (varint, tag=0x08, value=0=SUCCESS), field 2 (string, tag=0x12, "dropz")
	phoneName := []byte("dropz")
	payload := append([]byte{0x08, 0x00, 0x12, byte(len(phoneName))}, phoneName...)
	// ActionID 0x01 prepended to payload
	data := append([]byte{0x01}, payload...)
	_, err := m.sendMessage(macAddress, CharNetworkMgmtCommand, 0x03, data, buildProtobufPacket)
	return err
}

// IsConnected checks if a device has an active BLE connection
func (m *Manager) IsConnected(macAddress string) bool {
	return m.getConn(macAddress) != nil
}

// SetMetadataCallback sets the callback for metadata updates
func (m *Manager) SetMetadataCallback(callback MetadataUpdateFunc) {
	m.mutex.Lock()
	m.metadataCallback = callback
	m.mutex.Unlock()
}

// SetStatusCallback sets the callback for push notification status updates
func (m *Manager) SetStatusCallback(cb func(macAddress string, statusID byte, value []byte)) {
	m.mutex.Lock()
	m.statusCallback = cb
	m.mutex.Unlock()
}

// RegisterStatusUpdates subscribes to push notifications for the given status IDs.
// The camera will send async 0x93 notifications whenever these statuses change.
func (m *Manager) RegisterStatusUpdates(macAddress string, statusIDs []byte) error {
	response, err := m.sendQuery(macAddress, QueryRegisterStatusUpdates, statusIDs)
	if err != nil {
		return fmt.Errorf("failed to register status updates: %w", err)
	}
	if response.Status != 0x00 {
		return fmt.Errorf("register status updates failed: status=0x%02X", response.Status)
	}

	// Dispatch current values to the callback
	m.mutex.RLock()
	cb := m.statusCallback
	m.mutex.RUnlock()
	if cb != nil {
		for id, value := range parseTLVPairs(response.Data) {
			cb(macAddress, id, value)
		}
	}

	return nil
}

// UnregisterStatusUpdates unsubscribes from all push notifications
func (m *Manager) UnregisterStatusUpdates(macAddress string) error {
	_, err := m.sendQuery(macAddress, QueryUnregisterStatusUpdates, nil)
	return err
}

// Stop stops the BLE manager
func (m *Manager) Stop() error {
	m.log.Info("Stopping BLE Manager")

	m.StopScanning()

	m.mutex.RLock()
	addresses := make([]string, 0, len(m.conns))
	for macAddress := range m.conns {
		addresses = append(addresses, macAddress)
	}
	m.mutex.RUnlock()

	for _, macAddress := range addresses {
		if err := m.Disconnect(macAddress); err != nil {
			m.log.Warn("Failed to disconnect", "device", macAddress, "err", err)
		}
	}

	return nil
}
