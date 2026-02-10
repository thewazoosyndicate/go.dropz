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
				if strings.Contains(strings.ToLower(result.LocalName()), "gopro") {
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

// GetDiscoveredDevices returns all discovered GoPro devices
func (m *Manager) GetDiscoveredDevices() []Device {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	devices := make([]Device, 0, len(m.discoveredDevices))
	for _, device := range m.discoveredDevices {
		devices = append(devices, *device)
	}
	return devices
}

// Connect establishes a connection to a GoPro device and performs complete setup
func (m *Manager) Connect(macAddress string) error {
	if err := m.validateMAC(macAddress); err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Check if already connected
	m.mutex.RLock()
	if _, exists := m.connectedDevices[macAddress]; exists {
		m.mutex.RUnlock()
		m.log.Debugf("Device %s already connected", macAddress)
		return nil
	}
	m.mutex.RUnlock()

	m.log.Infof("Connecting to GoPro device: %s", macAddress)

	// Parse MAC and connect
	mac, err := bluetooth.ParseMAC(macAddress)
	if err != nil {
		return fmt.Errorf("failed to parse MAC: %v", err)
	}

	addr := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}
	device, err := m.adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		return fmt.Errorf("BLE connection failed: %v", err)
	}

	// Store the connected device
	m.mutex.Lock()
	m.connectedDevices[macAddress] = &device
	m.mutex.Unlock()

	// Wait for connection to stabilize
	m.log.Debug("Waiting for connection to stabilize...")
	time.Sleep(PostConnectDelay)

	// === Service and Characteristic Discovery (inline from setupOpenGoPro) ===
	
	// Discover services - direct call, no goroutine (works with DBus/BlueZ)
	m.log.Info("Starting service discovery...")
	var services []bluetooth.DeviceService
	
	for retry := 0; retry < ServiceDiscoveryRetries; retry++ {
		if retry > 0 {
			m.log.Infof("Retrying service discovery (attempt %d/%d)...", retry+1, ServiceDiscoveryRetries)
			time.Sleep(time.Second * time.Duration(retry+1))
		}

		// Direct call - no goroutine wrapper that breaks DBus context
		m.log.Debug("Calling DiscoverServices directly...")
		services, err = device.DiscoverServices(nil)
		
		if err == nil && len(services) > 0 {
			m.log.Infof("Successfully discovered %d services", len(services))
			break
		}
		
		if err != nil {
			m.log.Warnf("Service discovery attempt %d failed: %v", retry+1, err)
		}
	}

	if err != nil || len(services) == 0 {
		if device := m.connectedDevices[macAddress]; device != nil {
			device.Disconnect()
		}
		delete(m.characteristics, macAddress)
		delete(m.connectedDevices, macAddress)
		return fmt.Errorf("service discovery failed after %d retries: %v", ServiceDiscoveryRetries, err)
	}

	// Log discovered services
	m.log.Infof("Discovered %d services, listing all UUIDs:", len(services))
	for _, service := range services {
		m.log.Infof("  - Service UUID: %s", service.UUID().String())
	}

	// Initialize characteristics map for this device
	m.mutex.Lock()
	m.characteristics[macAddress] = make(map[string]bluetooth.DeviceCharacteristic)
	m.mutex.Unlock()

	// Discover and cache characteristics - EXACTLY like the test
	for _, service := range services {
		serviceUUID := service.UUID().String()
		m.log.Debugf("Discovering characteristics for service: %s", serviceUUID)
		
		chars, err := service.DiscoverCharacteristics(nil)
		if err != nil {
			m.log.Warnf("Failed to discover characteristics for service %s: %v", serviceUUID, err)
			continue
		}
		
		m.log.Debugf("Found %d characteristics for service %s", len(chars), serviceUUID)
		
		// Store only the characteristics we need
		m.mutex.Lock()
		for _, char := range chars {
			charUUID := char.UUID().String()
			
			// Only store characteristics we actually use
			if charUUID == CharWifiSSID || charUUID == CharWifiPassword ||
			   charUUID == CharCommand || charUUID == CharCommandResponse ||
			   charUUID == CharSettings || charUUID == CharSettingsResponse ||
			   charUUID == CharQuery || charUUID == CharQueryResponse {
				m.log.Tracef("Storing needed characteristic: %s", GetCharacteristicName(charUUID))
				m.characteristics[macAddress][charUUID] = char
			}

			// Enable notifications for response characteristics
			if charUUID == CharCommandResponse || charUUID == CharQueryResponse || charUUID == CharSettingsResponse {
				err := char.EnableNotifications(func(data []byte) {
					m.handleNotification(macAddress, data)
				})
				if err != nil {
					m.log.Warnf("Failed to enable notifications for %s: %v", GetCharacteristicName(charUUID), err)
				} else {
					m.log.Debugf("Enabled notifications for %s", GetCharacteristicName(charUUID))
				}
			}
		}
		m.mutex.Unlock()
	}

	m.log.Info("Service discovery complete")

	// Poll GetHardwareInfo until camera is ready (OpenGoPro spec requirement).
	// Status 0x02 means "camera not ready" (e.g. resuming from suspend).
	m.log.Info("Waiting for camera to be ready...")
	var hwInfo *HardwareInfo
	for attempt := 1; attempt <= 10; attempt++ {
		hwInfo, err = m.GetHardwareInfo(macAddress)
		if err == nil && hwInfo != nil {
			m.log.Infof("Camera ready: %s (FW: %s)", hwInfo.ModelName, hwInfo.FirmwareVersion)
			break
		}
		if attempt == 10 {
			if device := m.connectedDevices[macAddress]; device != nil {
				device.Disconnect()
			}
			delete(m.characteristics, macAddress)
			delete(m.connectedDevices, macAddress)
			return fmt.Errorf("camera not ready after %d attempts: %v", attempt, err)
		}
		m.log.Debugf("Camera not ready (attempt %d/10): %v", attempt, err)
		time.Sleep(time.Second)
	}

	// Identify as third-party client
	if err := m.SetThirdPartyClient(macAddress); err != nil {
		m.log.Warnf("Failed to set third party client flag: %v", err)
	}

	// Sync camera clock to host time
	if err := m.SetLocalDateTime(macAddress, time.Now()); err != nil {
		m.log.Warnf("Failed to set camera date/time: %v", err)
	}

	// Enable WiFi AP
	m.log.Info("Enabling WiFi Access Point")
	if err := m.SetAPControl(macAddress, WiFiAPModeEnable); err != nil {
		if device := m.connectedDevices[macAddress]; device != nil {
			device.Disconnect()
		}
		delete(m.characteristics, macAddress)
		delete(m.connectedDevices, macAddress)
		return fmt.Errorf("failed to enable WiFi AP: %v", err)
	}

	// Wait for WiFi AP to be ready
	time.Sleep(2 * time.Second)

	// Get WiFi credentials
	ssid, password, err := m.GetWifiCredentials(macAddress)
	if err != nil {
		if device := m.connectedDevices[macAddress]; device != nil {
			device.Disconnect()
		}
		delete(m.characteristics, macAddress)
		delete(m.connectedDevices, macAddress)
		return fmt.Errorf("failed to get WiFi credentials: %v", err)
	}
	m.log.Infof("Successfully obtained WiFi credentials: SSID=%s", ssid)

	// Prepare metadata for callback
	metadata := CameraMetadata{
		MACAddress:   macAddress,
		WiFiSSID:     ssid,
		WiFiPassword: password,
	}

	// Add hardware info from readiness poll to metadata
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

	// Fetch battery status and add to metadata
	battery, err := m.GetBatteryLevel(macAddress)
	if err != nil {
		m.log.Warnf("Failed to read battery level: %v", err)
	} else {
		metadata.BatteryLevel = battery
		m.log.Infof("Connected to GoPro %s: battery %d%%", macAddress, battery)
	}

	// Invoke metadata callback if set
	m.mutex.RLock()
	callback := m.metadataCallback
	m.mutex.RUnlock()
	
	if callback != nil {
		m.log.Debug("Invoking metadata update callback")
		callback(metadata)
	}

	m.log.Info("GoPro connection and setup completed successfully")
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

	m.log.Infof("Disconnecting from device: %s", macAddress)

	// Send sleep command before disconnecting
	if err := m.Sleep(macAddress); err != nil {
		m.log.Warnf("Sleep command failed: %v", err)
	}

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

// Sleep puts the device to sleep
func (m *Manager) Sleep(macAddress string) error {
	// According to OpenGoPro BLE spec, Sleep command (ID 0x05) sends a response
	// on Command Response UUID like all other TLV commands
	m.log.Infof("Sending sleep command to device: %s", macAddress)

	// Send sleep command and ignore non-zero status or errors (downgrade warnings)
	response, err := m.sendCommand(macAddress, CmdSleep, nil)
	if err != nil {
		m.log.Tracef("Sleep command error (ignored): %v", err)
		return nil
	}
	if response.Status != 0 {
		m.log.Tracef("Sleep command returned non-zero status 0x%02X (ignored) for device: %s", response.Status, macAddress)
		return nil
	}
	m.log.Tracef("Sleep command completed for device: %s", macAddress)
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
		// Store previous values for change detection
		previousRSSI := existingDevice.RSSI
		previousLastSeen := existingDevice.LastSeen
		previousName := existingDevice.Name

		// Update existing device with fresh discovery data
		existingDevice.RSSI = rssi
		existingDevice.LastSeen = now

		// Update name if it was empty or changed
		if existingDevice.Name == "" || existingDevice.Name != localName {
			existingDevice.Name = localName
		}

		// Calculate time since last seen for logging
		timeSinceLastSeen := now.Sub(previousLastSeen)

		// Log rediscovery with detailed information
		if previousRSSI != rssi || previousName != existingDevice.Name || timeSinceLastSeen > 5*time.Second {
			m.log.Tracef("Rediscovered GoPro device: %s (%s) RSSI:%d->%d, LastSeen updated (was %v ago)",
				existingDevice.Name, macAddress, previousRSSI, rssi, timeSinceLastSeen.Truncate(time.Millisecond))
		} else {
			m.log.Tracef("Rediscovered GoPro device: %s (%s) RSSI:%d", existingDevice.Name, macAddress, rssi)
		}

		// Call live discovery callback if set
		if m.discoveryCallback != nil {
			// Create a copy to avoid race conditions
			deviceCopy := *existingDevice
			go func() {
				m.log.Tracef("Calling live discovery callback for rediscovered device: %s (%s)", deviceCopy.Name, deviceCopy.MACAddress)
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

		// Call live discovery callback if set
		if m.discoveryCallback != nil {
			// Create a copy to avoid race conditions
			deviceCopy := *device
			go func() {
				m.log.Tracef("Calling live discovery callback for new device: %s (%s)", deviceCopy.Name, deviceCopy.MACAddress)
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
	m.log.Debugf("Sending 0x%02X on %s to %s (%d packets)", id, GetCharacteristicName(charUUID), macAddress, len(packets))

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
		m.log.Tracef("Notification for unknown device %s, ignoring", macAddress)
		return
	}

	// Process TLV fragment through the collector
	if m.tlvCollector != nil {
		message, err := m.tlvCollector.ProcessFragment(data)
		if err != nil {
			m.log.Debugf("Error processing TLV fragment from %s: %v", macAddress, err)
		} else if message != nil && m.responseTracker != nil {
			// Route complete message to waiting response
			m.responseTracker.RouteResponse(message)
		}
	}
}

func (m *Manager) validateMAC(macAddress string) error {
	if len(macAddress) != 17 {
		return fmt.Errorf("invalid MAC address length")
	}
	// Add more validation if needed
	return nil
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

	if len(response.Data) < 2 || response.Data[0] != StatusPairingState {
		return 0, fmt.Errorf("invalid pairing state response")
	}

	return int(response.Data[1]), nil
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

	// Stop any active scanning
	m.StopScanning()

	// Disconnect all active connections
	m.mutex.Lock()
	devices := make(map[string]*bluetooth.Device)
	for k, v := range m.connectedDevices {
		devices[k] = v
	}
	m.mutex.Unlock()

	for macAddress := range devices {
		if err := m.Disconnect(macAddress); err != nil {
			m.log.Warnf("Failed to disconnect from %s: %v", macAddress, err)
		}
	}

	return nil
}
