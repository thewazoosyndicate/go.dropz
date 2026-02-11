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
	discoveryCallback DeviceDiscoveryCallback // callback for live discovery updates
	metadataCallback  MetadataUpdateFunc      // callback for metadata updates during connection
	tlvCollector    *tlv.FragmentCollector
	responseTracker *tlv.ResponseTracker
}

// NewManager creates a new BLE manager
func NewManager(adapter *bluetooth.Adapter, log *logrus.Logger) *Manager {
	return &Manager{
		adapter:           adapter,
		discoveredDevices: make(map[string]*Device),
		connectedDevices:  make(map[string]*bluetooth.Device),
		characteristics:   make(map[string]map[string]bluetooth.DeviceCharacteristic),
		log:               log,
		tlvCollector:      tlv.NewFragmentCollector(10 * time.Second),
		responseTracker:   tlv.NewResponseTracker(),
	}
}

// StartScanning scans for GoPro devices with optional live discovery callback
func (m *Manager) StartScanningWithCallback(ctx context.Context, callback DeviceDiscoveryCallback) error {
	m.mutex.Lock()
	if m.isScanning {
		m.mutex.Unlock()
		return nil
	}
	m.isScanning = true
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
	// clear scanning flag and discovery callback
	m.isScanning = false
	m.discoveryCallback = nil
	m.mutex.Unlock()

	m.log.Info("Stopping BLE scan")
	return m.adapter.StopScan()
}

// connectBase establishes a BLE connection, discovers services/characteristics,
// and polls until the camera is ready. Shared by Connect and ConnectForPairing.
func (m *Manager) connectBase(macAddress string) (hwInfo *HardwareInfo, err error) {
	if err := validateAddress(macAddress); err != nil {
		return nil, fmt.Errorf("invalid device address: %v", err)
	}

	// Check if already connected
	m.mutex.RLock()
	if _, exists := m.connectedDevices[macAddress]; exists {
		m.mutex.RUnlock()
		m.log.Debugf("Device %s already connected", macAddress)
		return nil, nil
	}
	m.mutex.RUnlock()

	m.log.Debugf("Connecting to GoPro device: %s", macAddress)

	addr, parseErr := parseAddress(macAddress)
	if parseErr != nil {
		return nil, parseErr
	}

	device, connErr := m.adapter.Connect(addr, bluetooth.ConnectionParams{})
	if connErr != nil {
		return nil, fmt.Errorf("BLE connection failed: %v", connErr)
	}

	// Store the connected device
	m.mutex.Lock()
	m.connectedDevices[macAddress] = &device
	m.mutex.Unlock()

	// Cleanup on failure
	defer func() {
		if err != nil {
			m.mutex.Lock()
			dev := m.connectedDevices[macAddress]
			delete(m.characteristics, macAddress)
			delete(m.connectedDevices, macAddress)
			m.mutex.Unlock()
			if dev != nil {
				dev.Disconnect()
			}
		}
	}()

	// Wait for connection to stabilize
	time.Sleep(PostConnectDelay)

	// Discover services
	var services []bluetooth.DeviceService
	var discoverErr error

	for retry := 0; retry < ServiceDiscoveryRetries; retry++ {
		if retry > 0 {
			m.log.Debugf("Retrying service discovery (attempt %d/%d)...", retry+1, ServiceDiscoveryRetries)
			time.Sleep(time.Second * time.Duration(retry+1))
		}

		services, discoverErr = device.DiscoverServices(nil)

		if discoverErr == nil && len(services) > 0 {
			break
		}

		if discoverErr != nil {
			m.log.Warnf("Service discovery attempt %d failed: %v", retry+1, discoverErr)
		}
	}

	if discoverErr != nil || len(services) == 0 {
		err = fmt.Errorf("service discovery failed after %d retries: %v", ServiceDiscoveryRetries, discoverErr)
		return nil, err
	}

	// Initialize characteristics map for this device
	m.mutex.Lock()
	m.characteristics[macAddress] = make(map[string]bluetooth.DeviceCharacteristic)
	m.mutex.Unlock()

	// Discover and cache characteristics
	for _, service := range services {
		serviceUUID := service.UUID().String()

		chars, charErr := service.DiscoverCharacteristics(nil)
		if charErr != nil {
			m.log.Warnf("Failed to discover characteristics for service %s: %v", serviceUUID, charErr)
			continue
		}

		m.mutex.Lock()
		for _, char := range chars {
			charUUID := char.UUID().String()

			if charUUID == CharWifiSSID || charUUID == CharWifiPassword ||
				charUUID == CharCommand || charUUID == CharCommandResponse ||
				charUUID == CharSettings || charUUID == CharSettingsResponse ||
				charUUID == CharQuery || charUUID == CharQueryResponse {
				m.characteristics[macAddress][charUUID] = char
			}

			if charUUID == CharCommandResponse || charUUID == CharQueryResponse || charUUID == CharSettingsResponse {
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

	// Identify as third-party client
	if tpErr := m.SetThirdPartyClient(macAddress); tpErr != nil {
		m.log.Warnf("Failed to set third party client flag: %v", tpErr)
	}

	// Sync camera clock to host time
	if dtErr := m.SetLocalDateTime(macAddress, time.Now()); dtErr != nil {
		m.log.Warnf("Failed to set camera date/time: %v", dtErr)
	}

	return hwInfo, nil
}

// finishConnection performs post-connect setup: metadata collection and callback.
func (m *Manager) finishConnection(macAddress string, hwInfo *HardwareInfo, ssid, password string) {
	metadata := CameraMetadata{
		MACAddress:   macAddress,
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

// Connect establishes a full connection: BLE + services + WiFi AP + credentials + metadata
func (m *Manager) Connect(macAddress string) (err error) {
	hwInfo, err := m.connectBase(macAddress)
	if err != nil {
		return err
	}
	// connectBase returns nil,nil when already connected
	if hwInfo == nil {
		return nil
	}

	// Cleanup on failure after connectBase succeeded
	defer func() {
		if err != nil {
			m.mutex.Lock()
			dev := m.connectedDevices[macAddress]
			delete(m.characteristics, macAddress)
			delete(m.connectedDevices, macAddress)
			m.mutex.Unlock()
			if dev != nil {
				dev.Disconnect()
			}
		}
	}()

	if err = m.SetAPControl(macAddress, WiFiAPModeEnable); err != nil {
		return fmt.Errorf("failed to enable WiFi AP: %v", err)
	}

	// Wait for WiFi AP to be ready
	time.Sleep(2 * time.Second)

	// Get WiFi credentials
	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		err = fmt.Errorf("failed to get WiFi credentials: %v", credErr)
		return err
	}
	m.finishConnection(macAddress, hwInfo, ssid, password)
	return nil
}

// ConnectForPairing establishes a lightweight BLE connection for pairing:
// connect + services + readiness poll + set time + credentials only (no WiFi AP enable)
func (m *Manager) ConnectForPairing(macAddress string) (err error) {
	hwInfo, err := m.connectBase(macAddress)
	if err != nil {
		return err
	}
	if hwInfo == nil {
		return nil
	}

	ssid, password, credErr := m.GetWifiCredentials(macAddress)
	if credErr != nil {
		m.log.Warnf("Failed to get WiFi credentials during pairing: %v", credErr)
	}

	m.finishConnection(macAddress, hwInfo, ssid, password)
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

	// Send sleep command before disconnecting
	m.Sleep(macAddress)

	device.Disconnect()
	
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
	response, err := m.sendQuery(macAddress, QueryGetStatus, []byte{StatusBatteryPercentage})
	if err != nil {
		return 0, err
	}

	// Query responses contain multiple TLV pairs: [StatusID][Length][Value]...
	// Parse the TLV pairs to find the battery percentage
	data := response.Data
	offset := 0
	
	for offset < len(data) {
		if offset+2 > len(data) {
			break // Not enough data for TLV header
		}
		
		statusID := data[offset]
		length := int(data[offset+1])
		offset += 2
		
		if offset+length > len(data) {
			m.log.Warnf("Status TLV truncated: ID=%d, expected %d bytes, have %d", 
				statusID, length, len(data)-offset)
			break
		}
		
		// Check if this is the battery percentage status
		if statusID == StatusBatteryPercentage {
			if length >= 1 {
				return int(data[offset]), nil
			}
			return 0, fmt.Errorf("battery percentage value too short")
		}
		
		// Skip to next TLV pair
		offset += length
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
	// According to OpenGoPro BLE spec, Sleep command (ID 0x05) sends a response
	// on Command Response UUID like all other TLV commands
	response, err := m.sendCommand(macAddress, CmdSleep, nil)
	if err != nil {
		return nil
	}
	_ = response
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
		existingDevice.RSSI = rssi
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
			MACAddress: macAddress,
			RSSI:       rssi,
			LastSeen:   now,
		}
		m.discoveredDevices[macAddress] = device
		m.log.Infof("Discovered new GoPro device: %s (%s) RSSI:%d", device.Name, device.MACAddress, device.RSSI)

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
			// Route complete message to waiting response
			m.responseTracker.RouteResponse(message)
		}
	}
}

// IsConnected checks if a device has an active BLE connection
func (m *Manager) IsConnected(macAddress string) bool {
	m.mutex.RLock()
	_, exists := m.connectedDevices[macAddress]
	m.mutex.RUnlock()
	return exists
}

// IsPaired checks if the device is currently paired
func (m *Manager) IsPaired(macAddress string) (bool, error) {
	pairingState, err := m.RefreshPairingState(macAddress)
	if err != nil {
		return false, err
	}
	return pairingState == PairingCompleted, nil
}

// RefreshPairingState gets the current pairing state from the device
func (m *Manager) RefreshPairingState(macAddress string) (int, error) {
	response, err := m.sendQuery(macAddress, QueryGetStatus, []byte{StatusPairingState})
	if err != nil {
		return 0, err
	}

	// TLV format: [StatusID][Length][Value...]
	if len(response.Data) < 3 || response.Data[0] != StatusPairingState {
		return 0, fmt.Errorf("invalid pairing state response")
	}

	return int(response.Data[2]), nil
}


// SetMetadataCallback sets the callback for metadata updates
func (m *Manager) SetMetadataCallback(callback MetadataUpdateFunc) {
	m.mutex.Lock()
	m.metadataCallback = callback
	m.mutex.Unlock()
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
