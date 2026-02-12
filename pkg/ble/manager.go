package ble

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble/tlv"
	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"
)


// Manager provides a clean, simple BLE interface for GoPro devices
// following the OpenGoPro BLE specification exactly
type Manager struct {
	adapter         *bluetooth.Adapter
	discoveredDevices map[string]*Device    // discovered devices during scanning
	connectedDevices  map[string]*bluetooth.Device // connected BLE devices
	characteristics   map[string]map[string]bluetooth.DeviceCharacteristic // cached characteristics per device
	mutex           sync.RWMutex
	log             *logrus.Logger
	isScanning      bool
	scanDone        chan struct{}            // closed when StopScanning() is called
	connectingDone  chan struct{}            // closed when connectBase() finishes, so scanner waits
	discoveryCallback DeviceDiscoveryCallback // callback for live discovery updates
	metadataCallback  MetadataUpdateFunc      // callback for metadata updates during connection
	statusCallback    func(macAddress string, statusID byte, value []byte) // push notification callback
	tlvCollector    *tlv.FragmentCollector
	responseTracker *tlv.ResponseTracker
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
func NewManager(adapter *bluetooth.Adapter, log *logrus.Logger) *Manager {
	m := &Manager{
		adapter:           adapter,
		discoveredDevices: make(map[string]*Device),
		connectedDevices:  make(map[string]*bluetooth.Device),
		characteristics:   make(map[string]map[string]bluetooth.DeviceCharacteristic),
		log:               log,
		tlvCollector:      tlv.NewFragmentCollector(10 * time.Second),
		responseTracker:   tlv.NewResponseTracker(),
	}

	// Route async push notifications (0x93) to per-status callbacks
	m.responseTracker.SetPushHandler(func(macAddress string, msg *tlv.TLVMessage) {
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

	return m
}

// StartScanning scans for GoPro devices with optional live discovery callback
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
			_ = m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
				// Primary: check for FEA6 service UUID per OpenGoPro spec
				// Fallback: name-based check for older cameras that may not advertise the UUID
				if result.HasServiceUUID(goProServiceUUID) ||
					strings.Contains(strings.ToLower(result.LocalName()), "gopro") {
					m.addDiscoveredDevice(result)
				}
			})
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

// ConnectingDone returns a channel that is closed when connectBase() finishes.
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
// On success, the device is stored in connectedDevices. On failure, cleanup is done.
func (m *Manager) connectAndDiscover(macAddress string, preDiscoveryHook func(string) error) ([]bluetooth.DeviceService, time.Time, error) {
	if err := validateAddress(macAddress); err != nil {
		return nil, time.Time{}, fmt.Errorf("invalid device address: %v", err)
	}

	m.mutex.RLock()
	if _, exists := m.connectedDevices[macAddress]; exists {
		m.mutex.RUnlock()
		m.log.Debugf("Device %s already connected", macAddress)
		return nil, time.Time{}, nil
	}
	m.mutex.RUnlock()

	connectStart := time.Now()

	// Signal scanner to wait until GATT discovery is complete before restarting.
	m.mutex.Lock()
	m.connectingDone = make(chan struct{})
	m.mutex.Unlock()
	defer m.signalConnectingDone()

	m.log.Debug("Stopping BLE scan for GATT connection")
	m.StopScanning()
	m.adapter.StopScan()
	time.Sleep(ScanToConnectDelay)

	addr, parseErr := parseAddress(macAddress)
	if parseErr != nil {
		return nil, time.Time{}, parseErr
	}

	m.log.Debugf("Connecting to GoPro device: %s", macAddress)

	// Retry loop: full disconnect+reconnect between attempts (BlueZ won't
	// recover a stale GATT handle, so re-calling DiscoverServices is useless).
	var device bluetooth.Device
	var services []bluetooth.DeviceService
	var connected bool

	for retry := 0; retry < ServiceDiscoveryRetries; retry++ {
		if retry > 0 {
			m.log.Debugf("Retry %d/%d: disconnecting and reconnecting...", retry+1, ServiceDiscoveryRetries)
			if connected {
				device.Disconnect()
				connected = false
			}
			m.mutex.Lock()
			delete(m.connectedDevices, macAddress)
			m.mutex.Unlock()
			time.Sleep(time.Duration(retry+1) * time.Second)
		}

		var connErr error
		device, connErr = m.adapter.Connect(addr, bluetooth.ConnectionParams{})
		if connErr != nil {
			m.log.Warnf("Connection attempt %d failed: %v", retry+1, connErr)
			continue
		}
		connected = true
		m.log.Debugf("BLE connected (attempt %d), elapsed=%v", retry+1, time.Since(connectStart))

		m.mutex.Lock()
		m.connectedDevices[macAddress] = &device
		m.mutex.Unlock()

		if preDiscoveryHook != nil {
			if hookErr := preDiscoveryHook(macAddress); hookErr != nil {
				m.log.Warnf("Pre-discovery hook failed: %v", hookErr)
			}
		}

		time.Sleep(PostConnectDelay)

		var discoverErr error
		services, discoverErr = device.DiscoverServices(nil)
		m.log.Debugf("Service discovery result: %d services, err=%v, elapsed=%v", len(services), discoverErr, time.Since(connectStart))

		if discoverErr == nil && len(services) > 0 {
			break
		}
		m.log.Warnf("Service discovery attempt %d failed: %v", retry+1, discoverErr)
	}

	if len(services) == 0 {
		m.mutex.Lock()
		delete(m.connectedDevices, macAddress)
		m.mutex.Unlock()
		device.Disconnect()
		return nil, time.Time{}, fmt.Errorf("service discovery failed after %d retries", ServiceDiscoveryRetries)
	}

	return services, connectStart, nil
}

// discoverCharacteristics enumerates characteristics on the given services,
// caching those in cacheChars and enabling notifications on notifyChars.
func (m *Manager) discoverCharacteristics(macAddress string, services []bluetooth.DeviceService, serviceFilter map[string]bool, cacheChars map[string]bool, notifyChars map[string]bool, connectStart time.Time) {
	m.mutex.Lock()
	m.characteristics[macAddress] = make(map[string]bluetooth.DeviceCharacteristic)
	m.mutex.Unlock()

	for _, service := range services {
		serviceUUID := service.UUID().String()
		if !serviceFilter[serviceUUID] {
			continue
		}

		chars, charErr := service.DiscoverCharacteristics(nil)
		if charErr != nil {
			m.log.Warnf("Failed to discover characteristics for service %s: %v", serviceUUID, charErr)
			continue
		}

		m.mutex.Lock()
		for _, char := range chars {
			charUUID := char.UUID().String()
			if cacheChars[charUUID] {
				m.characteristics[macAddress][charUUID] = char
			}
			if notifyChars[charUUID] {
				notifErr := char.EnableNotifications(func(data []byte) {
					m.handleNotification(macAddress, data)
				})
				if notifErr != nil {
					m.log.Warnf("Failed to enable notifications for %s: %v", GetCharacteristicName(charUUID), notifErr)
				}
			}
		}
		m.mutex.Unlock()
	}

	m.signalConnectingDone()
	m.log.Debugf("Characteristic discovery complete, elapsed=%v", time.Since(connectStart))
}

// cleanupOnError removes connection state for a device. Used as a deferred cleanup.
func (m *Manager) cleanupOnError(macAddress string) {
	m.mutex.Lock()
	dev := m.connectedDevices[macAddress]
	delete(m.characteristics, macAddress)
	delete(m.connectedDevices, macAddress)
	m.mutex.Unlock()
	if dev != nil {
		dev.Disconnect()
	}
}

// connectBase establishes a BLE connection, discovers services/characteristics,
// and polls until the camera is ready.
func (m *Manager) connectBase(macAddress string) (hwInfo *HardwareInfo, err error) {
	services, connectStart, connErr := m.connectAndDiscover(macAddress, nil)
	if connErr != nil {
		return nil, connErr
	}
	if services == nil {
		return nil, nil // already connected
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
	for attempt := 1; attempt <= 10; attempt++ {
		hwInfo, err = m.GetHardwareInfo(macAddress)
		if err == nil && hwInfo != nil {
			break
		}
		if attempt == 10 {
			err = fmt.Errorf("camera not ready after %d attempts: %v", attempt, err)
			return nil, err
		}
		time.Sleep(time.Second)
	}

	m.log.Debugf("Camera ready, elapsed=%v", time.Since(connectStart))

	if tpErr := m.SetThirdPartyClient(macAddress); tpErr != nil {
		m.log.Warnf("Failed to set third party client flag: %v", tpErr)
	}

	if dtErr := m.SetLocalDateTime(macAddress, time.Now()); dtErr != nil {
		m.log.Warnf("Failed to set camera date/time: %v", dtErr)
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
		m.log.Warnf("Failed to read battery level: %v", err)
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
	services, connectStart, err := m.connectAndDiscover(macAddress, nil)
	if err != nil {
		return err
	}
	if services == nil {
		return nil // already connected
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
	m.mutex.RLock()
	device := m.connectedDevices[macAddress]
	m.mutex.RUnlock()

	if device == nil {
		return fmt.Errorf("device not connected: %s", macAddress)
	}

	device.Disconnect()
	m.tlvCollector.Reset()

	m.mutex.Lock()
	delete(m.connectedDevices, macAddress)
	delete(m.characteristics, macAddress)
	m.mutex.Unlock()

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
		return fmt.Errorf("failed to enable WiFi AP: %v", err)
	}

	if regErr := m.RegisterStatusUpdates(macAddress, []byte{StatusBatteryPercentage}); regErr != nil {
		m.log.Warnf("Failed to register status push notifications: %v", regErr)
	}

	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		err = fmt.Errorf("failed to get WiFi credentials: %v", credErr)
		return err
	}
	m.finishConnection(macAddress, hwInfo, ssid, password)
	return nil
}

// ConnectForPairing establishes a minimal BLE connection for pairing.
// Only discovers WiFi AP and Camera Management services — skips the control
// service (which hangs on BlueZ) since pairing doesn't need commands/queries.
func (m *Manager) ConnectForPairing(macAddress string) (err error) {
	// D-Bus bonding runs between connect and service discovery on each attempt
	services, connectStart, connErr := m.connectAndDiscover(macAddress, func(addr string) error {
		if pairErr := m.pairViaDbus(addr); pairErr != nil {
			m.log.Warnf("D-Bus pairing failed: %v", pairErr)
		}
		return nil
	})
	if connErr != nil {
		return connErr
	}
	if services == nil {
		return nil // already connected
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
			m.log.Debugf("SendPairingFinish (best-effort): %v", err)
		}
	}()

	// Read WiFi credentials (now accessible after D-Bus bonding)
	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		m.log.Warnf("Failed to read WiFi credentials during pairing: %v", credErr)
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

	m.log.Debugf("Pairing connection ready, elapsed=%v", time.Since(connectStart))
	return nil
}

// Disconnect closes the connection to a device
func (m *Manager) Disconnect(macAddress string) error {
	m.mutex.RLock()
	device := m.connectedDevices[macAddress]
	m.mutex.RUnlock()
	
	if device == nil {
		return fmt.Errorf("device not connected: %s", macAddress)
	}

	if err := m.Sleep(macAddress); err != nil {
		m.log.Debugf("Sleep failed for %s: %v", macAddress, err)
	} else {
		m.log.Debugf("Sleep command succeeded for %s", macAddress)
	}

	device.Disconnect()
	m.tlvCollector.Reset()

	m.mutex.Lock()
	delete(m.connectedDevices, macAddress)
	delete(m.characteristics, macAddress)
	m.mutex.Unlock()

	return nil
}

// GetWifiCredentials retrieves WiFi SSID and password from the device
func (m *Manager) GetWifiCredentials(macAddress string) (string, string, error) {
	m.mutex.RLock()
	device := m.connectedDevices[macAddress]
	chars := m.characteristics[macAddress]
	m.mutex.RUnlock()
	
	if device == nil {
		return "", "", fmt.Errorf("device not connected: %s", macAddress)
	}

	// Get WiFi SSID
	ssidChar, exists := chars[CharWifiSSID]
	if !exists {
		return "", "", fmt.Errorf("WiFi SSID characteristic not found")
	}

	ssidData := make([]byte, 32)
	n, err := ssidChar.Read(ssidData)
	if err != nil {
		return "", "", fmt.Errorf("failed to read SSID: %v", err)
	}
	ssid := strings.TrimSpace(string(ssidData[:n]))

	// Get WiFi password
	passChar, exists := chars[CharWifiPassword]
	if !exists {
		return "", "", fmt.Errorf("WiFi password characteristic not found")
	}

	passData := make([]byte, 64)
	n, err = passChar.Read(passData)
	if err != nil {
		return "", "", fmt.Errorf("failed to read password: %v", err)
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

// KeepAlive sends a keep-alive to prevent the camera from auto-sleeping
func (m *Manager) KeepAlive(macAddress string) error {
	_, err := m.sendCommand(macAddress, CmdKeepAlive, []byte{0x01, 0x42})
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

func (m *Manager) addDiscoveredDevice(result bluetooth.ScanResult) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	macAddress := result.Address.String()
	localName := result.LocalName()
	rssi := int32(result.RSSI)
	now := time.Now()

	// Check if device already exists
	if existingDevice, exists := m.discoveredDevices[macAddress]; exists {
		// EMA smoothing to reduce RSSI jitter between advertisements
		const emaAlpha = 0.15
		existingDevice.RSSI = int32(emaAlpha*float64(rssi) + (1-emaAlpha)*float64(existingDevice.RSSI))
		existingDevice.LastSeen = now
		if existingDevice.Name == "" || existingDevice.Name != localName {
			existingDevice.Name = localName
		}

		if m.discoveryCallback != nil {
			deviceCopy := *existingDevice
			go func() {
				m.discoveryCallback(deviceCopy)
			}()
		}
	} else {
		// Create new device
		device := &Device{
			Name:       localName,
			BLEAddress: macAddress,
			RSSI:       rssi,
			LastSeen:   now,
		}
		m.discoveredDevices[macAddress] = device
		m.log.Infof("Discovered new GoPro device: %s (%s) RSSI:%d", device.Name, device.BLEAddress, device.RSSI)

		if m.discoveryCallback != nil {
			deviceCopy := *device
			go func() {
				m.discoveryCallback(deviceCopy)
			}()
		}
	}
}


// sendMessage sends a TLV message on the given characteristic and waits for a response.
func (m *Manager) sendMessage(macAddress string, charUUID string, id byte, data []byte, buildPacket func(byte, []byte) []byte) (Response, error) {
	m.mutex.RLock()
	device := m.connectedDevices[macAddress]
	chars := m.characteristics[macAddress]
	m.mutex.RUnlock()

	if device == nil {
		return Response{}, fmt.Errorf("device not connected")
	}

	char, exists := chars[charUUID]
	if !exists {
		return Response{}, fmt.Errorf("characteristic %s not found", GetCharacteristicName(charUUID))
	}

	responseChan := m.responseTracker.RegisterCommand(id)
	defer m.responseTracker.UnregisterCommand(id)

	packets := tlv.SplitIntoPackets(buildPacket(id, data))

	for i, pkt := range packets {
		if _, err := char.WriteWithoutResponse(pkt); err != nil {
			return Response{}, fmt.Errorf("failed to send packet %d: %v", i, err)
		}
	}

	select {
	case message := <-responseChan:
		return Response{
			Status: message.Status,
			Data:   message.Payload,
		}, nil
	case <-time.After(5 * time.Second):
		m.log.Warnf("0x%02X timed out after 5 seconds", id)
		return Response{}, fmt.Errorf("timeout waiting for response to 0x%02X", id)
	}
}

func (m *Manager) sendCommand(macAddress string, commandID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharCommand, commandID, data, tlv.BuildCommandPacket)
}

func (m *Manager) sendQuery(macAddress string, queryID byte, data []byte) (Response, error) {
	return m.sendMessage(macAddress, CharQuery, queryID, data, tlv.BuildQueryPacket)
}

func (m *Manager) handleNotification(macAddress string, data []byte) {
	m.mutex.RLock()
	device := m.connectedDevices[macAddress]
	m.mutex.RUnlock()
	
	if device == nil {
		return
	}

	if m.tlvCollector != nil {
		message, err := m.tlvCollector.ProcessFragment(data)
		if err != nil {
			m.log.Debugf("Error processing TLV fragment: %v", err)
		} else if message != nil && m.responseTracker != nil {
			m.responseTracker.RouteResponse(macAddress, message)
		}
	}
}

// buildProtobufPacket builds a protobuf-style packet: [FeatureID][data...]
// Used for Network Management commands (GP-0091) which don't use TLV framing.
func buildProtobufPacket(featureID byte, data []byte) []byte {
	return append([]byte{featureID}, data...)
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
	m.mutex.RLock()
	_, exists := m.connectedDevices[macAddress]
	m.mutex.RUnlock()
	return exists
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
		return fmt.Errorf("failed to register status updates: %v", err)
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

	m.mutex.Lock()
	devices := make(map[string]*bluetooth.Device, len(m.connectedDevices))
	for k, v := range m.connectedDevices {
		devices[k] = v
	}
	m.mutex.Unlock()

	for macAddress := range devices {
		if err := m.Disconnect(macAddress); err != nil {
			m.log.Warnf("Failed to disconnect from %s: %v", macAddress, err)
		}
	}

	m.tlvCollector.Stop()

	return nil
}
