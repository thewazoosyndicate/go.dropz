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
	"tinygo.org/x/bluetooth"
)

// Ensure Manager implements BLEInterface at compile time
var _ BLEInterface = (*Manager)(nil)

// Manager provides a clean, simple BLE interface for GoPro devices
// following the OpenGoPro BLE specification exactly
type Manager struct {
	adapter           *bluetooth.Adapter
	devices           map[string]*Device           // discovered devices
	connections       map[string]*DeviceConnection // active connections
	mutex             sync.RWMutex
	log               logger.Logger
	isScanning        bool
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

	defer func() {
		m.mutex.Lock()
		m.isScanning = false
		m.discoveryCallback = nil
		m.mutex.Unlock()
	}()

	m.log.Info("Starting BLE scan for GoPro devices")

	return m.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if strings.Contains(strings.ToLower(result.LocalName()), "gopro") {
			m.addDiscoveredDevice(result)
		}
	})
}

// StopScanning stops the current scan
func (m *Manager) StopScanning() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isScanning {
		return nil
	}

	return m.adapter.StopScan()
}

// GetDiscoveredDevices returns all discovered GoPro devices
func (m *Manager) GetDiscoveredDevices() []Device {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	m.log.Debugf("GetDiscoveredDevices called - internal map has %d devices", len(m.devices))
	for macAddr, device := range m.devices {
		m.log.Debugf("Internal device: %s (%s) RSSI:%d LastSeen:%s",
			device.Name, macAddr, device.RSSI, device.LastSeen.Format(time.RFC3339))
	}

	devices := make([]Device, 0, len(m.devices))
	for _, device := range m.devices {
		devices = append(devices, *device)
	}

	m.log.Debugf("Returning %d devices to caller", len(devices))
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
	m.log.Infof("Successfully connected to GoPro device: %s", macAddress)
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
	response, err := m.sendQuery(macAddress, QueryGetStatus, []byte{StatusBatteryLevel})
	if err != nil {
		return 0, err
	}

	if len(response.Data) < 2 || response.Data[0] != StatusBatteryLevel {
		return 0, fmt.Errorf("invalid battery response")
	}

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

	// First check if camera is ready to accept commands
	// According to OpenGoPro spec: "camera may not be ready to accept specific commands
	// if System Busy or Encoding Active status flags are set"
	conn := m.getConnection(macAddress)
	if conn == nil || conn.GetState() != StateReady {
		return fmt.Errorf("device not ready: %s", macAddress)
	}

	// Check camera status before sending sleep command
	statusResponse, err := m.sendQuery(macAddress, QueryGetStatus, nil)
	if err != nil {
		if strings.Contains(err.Error(), "no response received") {
			m.log.Infof("No status response before sleep, proceeding anyway")
		} else {
			m.log.Warnf("Could not check camera status before sleep, proceeding anyway: %v", err)
		}
	} else {
		// Check if camera is busy or encoding (this would prevent sleep from working)
		if len(statusResponse.Data) >= 2 {
			// Status byte 1: System Busy flag
			// Status byte 2: Encoding Active flag
			systemBusy := statusResponse.Data[0] != 0
			encodingActive := statusResponse.Data[1] != 0

			if systemBusy || encodingActive {
				m.log.Warnf("Camera is busy (SystemBusy=%v, EncodingActive=%v) - sleep command may not work properly",
					systemBusy, encodingActive)
				return fmt.Errorf("camera is busy: SystemBusy=%v, EncodingActive=%v - cannot sleep", systemBusy, encodingActive)
			}
		}
	}

	// Send sleep command and ignore non-zero status or errors (downgrade warnings)
	response, err := m.sendCommand(macAddress, CmdSleep, nil)
	if err != nil {
		m.log.Infof("Sleep command error (ignored): %v", err)
		return nil
	}
	if response.Status != 0 {
		m.log.Infof("Sleep command returned non-zero status 0x%02X (ignored) for device: %s", response.Status, macAddress)
		return nil
	}
	m.log.Infof("Sleep command completed successfully for device: %s", macAddress)
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
			m.log.Debugf("Rediscovered GoPro device: %s (%s) RSSI:%d->%d, LastSeen updated (was %v ago)",
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

	// Build TLV command according to OpenGoPro BLE specification
	// Format: [Length, CommandID, Parameters...]
	var cmd []byte
	if len(data) == 0 {
		// Command without parameters: [Length=1, CommandID]
		cmd = []byte{0x01, commandID}
	} else {
		// Command with parameters: [Length=1+len(data), CommandID, Parameters...]
		length := byte(1 + len(data))
		cmd = append([]byte{length, commandID}, data...)
	}

	// Flush any stale responses before sending
flushLoop:
	for {
		select {
		case <-conn.responses:
			// drop stale
		default:
			break flushLoop
		}
	}

	m.log.Debugf("Sending TLV command 0x%02X (length=%d) to device: %s", commandID, len(cmd)-1, macAddress)
	// Send command
	if _, err := cmdChar.WriteWithoutResponse(cmd); err != nil {
		return Response{}, fmt.Errorf("failed to send command: %v", err)
	}

	// Collect TLV fragments and reassemble payload
	timeout := time.After(5 * time.Second)
	var fullPayload []byte
	var status byte
	var expectedLen int
	first := true
	for {
		select {
		case resp := <-conn.responses:
			// raw TLV fragment in resp.Data
			raw := resp.Data
			// parse header on first fragment: detect start-of-multi-packet and compute total length
			if first {
				first = false
				var hdrLen int
				var totalLen int
				// Check start flag for multi-packet
				if raw[0]&HeaderStart != 0 {
					// multi-packet TLV start: next byte holds full length bits
					hdrLen = 2
					// combine lower 5 bits of first byte as MSB and second byte as LSB
					totalLen = (int(raw[0]&LengthMask) << 8) | int(raw[1])
				} else {
					// single-packet TLV: length in lower 5 bits
					hdrLen = 1
					totalLen = int(raw[0] & LengthMask)
				}
				// extract status (after command ID)
				status = raw[hdrLen+1]
				// append payload chunk after header, command ID, and status
				fullPayload = append(fullPayload, raw[hdrLen+2:]...)
				// expected payload length = totalLen - 2 (command ID + status)
				expectedLen = totalLen - 2
			} else {
				// subsequent fragments contain payload only
				// strip continuation header on multi-packet fragments
				if raw[0]&HeaderCont != 0 {
					// skip header byte
					fullPayload = append(fullPayload, raw[1:]...)
				} else {
					// no header flag, append all bytes
					fullPayload = append(fullPayload, raw...)
				}
			}
			m.log.Debugf("Reassembly for command 0x%02X: got %d/%d bytes", commandID, len(fullPayload), expectedLen)
			if len(fullPayload) >= expectedLen {
				return Response{CommandID: commandID, Status: status, Data: fullPayload, Timestamp: time.Now()}, nil
			}
		case <-timeout:
			m.log.Warnf("Command 0x%02X timed out after 5 seconds", commandID)
			return Response{}, fmt.Errorf("command timeout")
		}
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

	// Build TLV setting command according to OpenGoPro BLE specification
	// Format: [Length, SettingID, Parameters...]
	var setting []byte
	if len(data) == 0 {
		// Setting without parameters: [Length=1, SettingID]
		setting = []byte{0x01, settingID}
	} else {
		// Setting with parameters: [Length=1+len(data), SettingID, Parameters...]
		length := byte(1 + len(data))
		setting = []byte{length, settingID}
		setting = append(setting, data...)
	}

	// Send setting
	_, err := settingsChar.WriteWithoutResponse(setting)
	if err != nil {
		return Response{}, fmt.Errorf("failed to send setting: %v", err)
	}

	// Wait for response
	select {
	case response := <-conn.responses:
		if response.CommandID == settingID {
			return response, nil
		}
	case <-time.After(5 * time.Second):
		return Response{}, fmt.Errorf("setting timeout")
	}

	return Response{}, fmt.Errorf("no response received")
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

	// Build TLV query according to OpenGoPro BLE specification
	// Format: [Length, QueryID, Parameters...]
	var query []byte
	if len(data) == 0 {
		// Query without parameters: [Length=1, QueryID]
		query = []byte{0x01, queryID}
	} else {
		// Query with parameters: [Length=1+len(data), QueryID, Parameters...]
		length := byte(1 + len(data))
		query = []byte{length, queryID}
		query = append(query, data...)
	}

	// Flush any stale responses before sending query
flushLoopQuery:
	for {
		select {
		case <-conn.responses:
			// drop stale
		default:
			break flushLoopQuery
		}
	}
	m.log.Debugf("Sending TLV query 0x%02X to device: %s", queryID, macAddress)
	// Send query
	_, err := queryChar.WriteWithoutResponse(query)
	if err != nil {
		return Response{}, fmt.Errorf("failed to send query: %v", err)
	}

	// Wait for response and parse TLV
	timeout := time.After(5 * time.Second)
	for {
		select {
		case resp := <-conn.responses:
			raw := resp.Data
			if len(raw) < 3 {
				continue // ignore invalid fragment
			}
			// determine header length
			hdrType := raw[0] >> 6
			var hdrLen int
			switch hdrType {
			case 0:
				hdrLen = 1
			case 2:
				hdrLen = 2
			case 3:
				hdrLen = 3
			default:
				hdrLen = 1
			}
			// parse fields: byte hdrLen is QueryID, byte hdrLen+1 is status
			cmdID := raw[hdrLen]
			status := raw[hdrLen+1]
			if cmdID != queryID {
				continue
			}
			// payload is remaining bytes
			payload := raw[hdrLen+2:]
			return Response{CommandID: cmdID, Status: status, Data: payload, Timestamp: resp.Timestamp}, nil
		case <-timeout:
			return Response{}, fmt.Errorf("query timeout")
		}
	}

	// unreachable
}

func (m *Manager) handleNotification(macAddress string, data []byte) {
	// TLV notification - forward raw bytes for reassembly
	if len(data) < 3 {
		return
	}
	conn := m.getConnection(macAddress)
	if conn == nil {
		return
	}
	// Wrap in Response.Data for parser
	resp := Response{
		Data:      data,
		Timestamp: time.Now(),
	}
	select {
	case conn.responses <- resp:
	default:
		// Channel full, drop oldest
		select {
		case <-conn.responses:
			conn.responses <- resp
		default:
		}
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
