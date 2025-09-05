package ble

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/logger"
	"github.com/dropz/dropz/pkg/ble/tlv"
	"tinygo.org/x/bluetooth"
)

// Ensure Manager implements BLEInterface at compile time
var _ BLEInterface = (*Manager)(nil)

// Manager provides a clean, simple BLE interface for GoPro devices
// following the OpenGoPro BLE specification exactly
type Manager struct {
	adapter     *bluetooth.Adapter
	devices     map[string]*Device           // discovered devices
	connections map[string]*DeviceConnection // active connections
	mutex       sync.RWMutex
	log         logger.Logger
	isScanning  bool
	// callback for live discovery updates
	discoveryCallback DeviceDiscoveryCallback // callback for live discovery updates
}

// NewManager creates a new BLE manager
func NewManager(adapter *bluetooth.Adapter, log logger.Logger) *Manager {
	return &Manager{
		adapter:     adapter,
		devices:     make(map[string]*Device),
		connections: make(map[string]*DeviceConnection),
		log:         log,
	}
}

// StartScanning scans for GoPro devices advertising the Control service (0xFEA6)
func (m *Manager) StartScanning(ctx context.Context) error {
	return m.StartScanningWithCallback(ctx, nil)
}

// StartScanningWithCallback scans for GoPro devices with live discovery callback
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

	m.log.Tracef("GetDiscoveredDevices called - internal map has %d devices", len(m.devices))
	for macAddr, device := range m.devices {
		m.log.Tracef("Internal device: %s (%s) RSSI:%d LastSeen:%s",
			device.Name, macAddr, device.RSSI, device.LastSeen.Format(time.RFC3339))
	}

	devices := make([]Device, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, *device)
	}

	m.log.Tracef("Returning %d devices to caller", len(devices))
	return devices
}

// Connect establishes a connection to a GoPro device and performs OpenGoPro BLE setup
func (m *Manager) Connect(macAddress string) error {
	if err := m.validateMAC(macAddress); err != nil {
		return fmt.Errorf("invalid MAC address: %v", err)
	}

	// Check if already connected
	if conn := m.getConnection(macAddress); conn != nil && conn.GetState() == StateConnected {
		return nil
	}

	m.log.Infof("Connecting to GoPro device: %s", macAddress)

	// Create connection object
	conn := NewDeviceConnection()
	m.setConnection(macAddress, conn)
	conn.SetState(StateConnecting)

	// Parse MAC and connect
	mac, err := bluetooth.ParseMAC(macAddress)
	if err != nil {
		conn.SetState(StateError)
		return fmt.Errorf("failed to parse MAC: %v", err)
	}

	addr := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: mac}}
	device, err := m.adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		conn.SetState(StateError)
		return fmt.Errorf("BLE connection failed: %v", err)
	}

	conn.SetDevice(&device)
	conn.SetState(StateConnected)

	// Perform OpenGoPro BLE setup
	if err := m.setupOpenGoPro(macAddress, conn); err != nil {
		conn.Close()
		m.removeConnection(macAddress)
		return fmt.Errorf("OpenGoPro setup failed: %v", err)
	}

	conn.SetState(StateReady)
	// Fetch and log battery status on connect
	if battery, err := m.GetBatteryLevel(macAddress); err != nil {
		m.log.Warnf("Failed to read battery level for device %s: %v", macAddress, err)
	} else {
		m.log.Infof("Connected to GoPro %s: battery %d%%", macAddress, battery)
	}
	return nil
}

// Disconnect closes the connection to a device
func (m *Manager) Disconnect(macAddress string) error {
	conn := m.getConnection(macAddress)
	if conn == nil {
		return fmt.Errorf("device not connected: %s", macAddress)
	}

	m.log.Infof("Disconnecting from device: %s", macAddress)

	// Send sleep command before disconnecting
	if err := m.Sleep(macAddress); err != nil {
		m.log.Warnf("Sleep command failed: %v", err)
	}

	conn.Close()
	m.removeConnection(macAddress)
	return nil
}

// GetWifiCredentials retrieves WiFi SSID and password from the device
func (m *Manager) GetWifiCredentials(macAddress string) (string, string, error) {
	conn := m.getConnection(macAddress)
	if conn == nil || conn.GetState() != StateReady {
		return "", "", fmt.Errorf("device not ready: %s", macAddress)
	}

	// Get WiFi SSID
	ssidChar, exists := conn.GetCharacteristic(CharWifiSSID)
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
	passChar, exists := conn.GetCharacteristic(CharWifiPassword)
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

// EnableWifi enables WiFi on the device
func (m *Manager) EnableWifi(macAddress string) error {
	conn := m.getConnection(macAddress)
	if conn == nil || conn.GetState() != StateReady {
		return fmt.Errorf("device not ready: %s", macAddress)
	}

	powerChar, exists := conn.GetCharacteristic(CharWifiPower)
	if !exists {
		return fmt.Errorf("WiFi power characteristic not found")
	}

	_, err := powerChar.WriteWithoutResponse([]byte{1}) // 1 = enable
	return err
}

// GetBatteryLevel queries the battery level from the device
func (m *Manager) GetBatteryLevel(macAddress string) (int, error) {
	response, err := m.sendQuery(macAddress, QueryGetStatus, []byte{StatusBatteryPercentage})
	if err != nil {
		return 0, err
	}

	if len(response.Data) < 2 || response.Data[0] != StatusBatteryPercentage {
		return 0, fmt.Errorf("invalid battery response")
	}
	// Return the percentage value directly
	return int(response.Data[1]), nil
}

// SetDateTime sets the date and time on the device
func (m *Manager) SetDateTime(macAddress string, t time.Time) error {
	data := []byte{
		byte(t.Year() >> 8), byte(t.Year() & 0xFF),
		byte(t.Month()), byte(t.Day()),
		byte(t.Hour()), byte(t.Minute()), byte(t.Second()),
	}

	_, err := m.sendCommand(macAddress, CmdSetDateTime, data)
	return err
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

// KeepAlive sends a keep-alive command using Settings characteristic with parameter 0x42
func (m *Manager) KeepAlive(macAddress string) error {
	// According to OpenGoPro BLE spec, Keep Alive (ID 0x5B) uses Settings characteristic
	// and requires parameter 0x42
	_, err := m.sendSetting(macAddress, CmdKeepAlive, []byte{0x42})
	return err
}

// GetMetadata retrieves device metadata (model, firmware, etc.)
func (m *Manager) GetMetadata(macAddress string) (map[string]string, error) {
	response, err := m.sendCommand(macAddress, CmdGetHardwareInfo, nil)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	d := response.Data
	// Payload format:
	// [0] padding, [1:5] model_number (uint32), [5] name_len, [6:6+name_len] name,
	// next padding, next 4 bytes board_type, [..] firmware_version_len, firmware_version bytes,
	// serial_number_len and serial_number bytes
	if len(d) < 6 {
		return metadata, nil
	}
	// model_number
	modelNum := int(binary.BigEndian.Uint32(d[1:5]))
	metadata["model_id"] = fmt.Sprintf("%d", modelNum)
	// model_name
	nameLen := int(d[5])
	if len(d) >= 6+nameLen {
		metadata["model_name"] = string(d[6 : 6+nameLen])
	}
	// skip past name and padding and board_type
	idx := 6 + nameLen
	if len(d) < idx+1+4+1 {
		return metadata, nil
	}
	// skip padding
	idx++
	// skip board_type uint32
	idx += 4
	// firmware_version
	fwLen := int(d[idx])
	idx++
	if len(d) >= idx+fwLen {
		metadata["firmware_version"] = string(d[idx : idx+fwLen])
		idx += fwLen
	}
	// serial_number
	if len(d) > idx {
		snLen := int(d[idx])
		idx++
		if len(d) >= idx+snLen {
			metadata["serial_number"] = string(d[idx : idx+snLen])
		}
	}

	return metadata, nil
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
	if existingDevice, exists := m.devices[macAddress]; exists {
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
		m.devices[macAddress] = device
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

func (m *Manager) setupOpenGoPro(macAddress string, conn *DeviceConnection) error {
	device := conn.GetDevice()
	if device == nil {
		return fmt.Errorf("no device available")
	}

	// Discover services
	services, err := device.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("service discovery failed: %v", err)
	}

	// Find and cache required characteristics
	for _, service := range services {
		chars, err := service.DiscoverCharacteristics(nil)
		if err != nil {
			continue
		}

		for _, char := range chars {
			charUUID := char.UUID().String()
			conn.SetCharacteristic(charUUID, char)

			// Enable notifications for response characteristics
			if m.isResponseCharacteristic(charUUID) {
				char.EnableNotifications(func(data []byte) {
					m.handleNotification(macAddress, data)
				})
			}
		}
	}

	// Verify required characteristics are present
	return m.verifyRequiredCharacteristics(conn)
}

func (m *Manager) isResponseCharacteristic(uuid string) bool {
	return uuid == CharCommandResponse || uuid == CharQueryResponse || uuid == CharSettingsResponse
}

func (m *Manager) verifyRequiredCharacteristics(conn *DeviceConnection) error {
	required := []string{CharCommand, CharCommandResponse, CharQuery, CharQueryResponse, CharSettings, CharSettingsResponse}

	for _, uuid := range required {
		if _, exists := conn.GetCharacteristic(uuid); !exists {
			return fmt.Errorf("required characteristic not found: %s", uuid)
		}
	}

	return nil
}

func (m *Manager) sendCommand(macAddress string, commandID byte, data []byte) (Response, error) {
	conn := m.getConnection(macAddress)
	if conn == nil || conn.GetState() != StateReady {
		return Response{}, fmt.Errorf("device not ready")
	}

	cmdChar, exists := conn.GetCharacteristic(CharCommand)
	if !exists {
		return Response{}, fmt.Errorf("command characteristic not found")
	}

	// Register for response
	responseChan := conn.RegisterPendingCommand(commandID)
	defer conn.UnregisterPendingCommand(commandID)

	// Build command packet using Extended 13-bit format (recommended by OpenGoPro)
	packet := tlv.BuildCommandPacket(commandID, data)
	packets := tlv.SplitIntoPackets(packet)

	m.log.Debugf("Sending command 0x%02X to device %s (%d packets)", commandID, macAddress, len(packets))
	
	// Send all packets
	for i, pkt := range packets {
		if _, err := cmdChar.WriteWithoutResponse(pkt); err != nil {
			return Response{}, fmt.Errorf("failed to send command packet %d: %v", i, err)
		}
	}

	// Wait for complete TLV message with timeout
	select {
	case message := <-responseChan:
		return Response{
			CommandID: message.CommandID,
			Status:    message.Status,
			Data:      message.Payload,
			Timestamp: message.Timestamp,
		}, nil
	case <-time.After(5 * time.Second):
		m.log.Warnf("Command 0x%02X timed out after 5 seconds", commandID)
		return Response{}, fmt.Errorf("command timeout")
	}
}

func (m *Manager) sendSetting(macAddress string, settingID byte, data []byte) (Response, error) {
	conn := m.getConnection(macAddress)
	if conn == nil || conn.GetState() != StateReady {
		return Response{}, fmt.Errorf("device not ready")
	}

	settingsChar, exists := conn.GetCharacteristic(CharSettings)
	if !exists {
		return Response{}, fmt.Errorf("settings characteristic not found")
	}

	// Register for response
	responseChan := conn.RegisterPendingCommand(settingID)
	defer conn.UnregisterPendingCommand(settingID)

	// Build setting packet using Extended 13-bit format (recommended by OpenGoPro)
	packet := tlv.BuildSettingPacket(settingID, data)
	packets := tlv.SplitIntoPackets(packet)

	m.log.Debugf("Sending setting 0x%02X to device %s (%d packets)", settingID, macAddress, len(packets))
	
	// Send all packets
	for i, pkt := range packets {
		if _, err := settingsChar.WriteWithoutResponse(pkt); err != nil {
			return Response{}, fmt.Errorf("failed to send setting packet %d: %v", i, err)
		}
	}

	// Wait for complete TLV message with timeout
	select {
	case message := <-responseChan:
		return Response{
			CommandID: message.CommandID,
			Status:    message.Status,
			Data:      message.Payload,
			Timestamp: message.Timestamp,
		}, nil
	case <-time.After(5 * time.Second):
		m.log.Warnf("Setting 0x%02X timed out after 5 seconds", settingID)
		return Response{}, fmt.Errorf("setting timeout")
	}
}

func (m *Manager) sendQuery(macAddress string, queryID byte, data []byte) (Response, error) {
	conn := m.getConnection(macAddress)
	if conn == nil || conn.GetState() != StateReady {
		return Response{}, fmt.Errorf("device not ready")
	}

	queryChar, exists := conn.GetCharacteristic(CharQuery)
	if !exists {
		return Response{}, fmt.Errorf("query characteristic not found")
	}

	// Register for response
	responseChan := conn.RegisterPendingCommand(queryID)
	defer conn.UnregisterPendingCommand(queryID)

	// Build query packet using Extended 13-bit format (recommended by OpenGoPro)
	packet := tlv.BuildQueryPacket(queryID, data)
	packets := tlv.SplitIntoPackets(packet)

	m.log.Debugf("Sending query 0x%02X to device %s (%d packets)", queryID, macAddress, len(packets))
	
	// Send all packets
	for i, pkt := range packets {
		if _, err := queryChar.WriteWithoutResponse(pkt); err != nil {
			return Response{}, fmt.Errorf("failed to send query packet %d: %v", i, err)
		}
	}

	// Wait for complete TLV message with timeout
	select {
	case message := <-responseChan:
		return Response{
			CommandID: message.CommandID,
			Status:    message.Status,
			Data:      message.Payload,
			Timestamp: message.Timestamp,
		}, nil
	case <-time.After(5 * time.Second):
		m.log.Warnf("Query 0x%02X timed out after 5 seconds", queryID)
		return Response{}, fmt.Errorf("query timeout")
	}
}

func (m *Manager) handleNotification(macAddress string, data []byte) {
	conn := m.getConnection(macAddress)
	if conn == nil {
		m.log.Tracef("Notification for unknown device %s, ignoring", macAddress)
		return
	}

	// Process TLV fragment through the collector
	if err := conn.ProcessTLVFragment(data); err != nil {
		m.log.Debugf("Error processing TLV fragment from %s: %v", macAddress, err)
	}
}

func (m *Manager) getConnection(macAddress string) *DeviceConnection {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.connections[macAddress]
}

func (m *Manager) setConnection(macAddress string, conn *DeviceConnection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connections[macAddress] = conn
}

func (m *Manager) removeConnection(macAddress string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.connections, macAddress)
}

func (m *Manager) validateMAC(macAddress string) error {
	if len(macAddress) != 17 {
		return fmt.Errorf("invalid MAC address length")
	}
	// Add more validation if needed
	return nil
}

// ConnectWithEnhancedPairing connects and performs enhanced pairing with the device
func (m *Manager) ConnectWithEnhancedPairing(macAddress string) error {
	// First connect normally
	if err := m.Connect(macAddress); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	// Get WiFi credentials as part of pairing verification
	_, _, err := m.GetWifiCredentials(macAddress)
	if err != nil {
		// Disconnect on failure
		m.Disconnect(macAddress)
		return fmt.Errorf("failed to get WiFi credentials during pairing: %v", err)
	}

	// Retrieve hardware metadata (model, firmware, serial)
	metadata, err := m.GetMetadata(macAddress)
	if err != nil {
		m.log.Warnf("failed to get hardware metadata: %v", err)
	} else {
		// Update cached device info
		m.mutex.Lock()
		if dev, exists := m.devices[macAddress]; exists {
			if id, ok := metadata["model_id"]; ok {
				// update existing modelID
				dev.ModelID, _ = strconv.Atoi(id)
			}
			if name, ok := metadata["model_name"]; ok {
				dev.ModelName = name
			}
			if fw, ok := metadata["firmware_version"]; ok {
				dev.FirmwareVersion = fw
			}
			if sn, ok := metadata["serial_number"]; ok {
				dev.SerialNumber = sn
			}
		}
		m.mutex.Unlock()
	}

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

// Start starts the BLE manager (lifecycle method)
func (m *Manager) Start() error {
	m.log.Info("BLE Manager started")
	return nil
}

// Stop stops the BLE manager (lifecycle method)
func (m *Manager) Stop() error {
	m.log.Info("Stopping BLE Manager")

	// Stop any active scanning
	m.StopScanning()

	// Disconnect all active connections
	m.mutex.Lock()
	connections := make(map[string]*DeviceConnection)
	for k, v := range m.connections {
		connections[k] = v
	}
	m.mutex.Unlock()

	for macAddress := range connections {
		if err := m.Disconnect(macAddress); err != nil {
			m.log.Warnf("Failed to disconnect from %s: %v", macAddress, err)
		}
	}

	return nil
}
