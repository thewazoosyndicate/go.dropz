package ble

import (
	"context"
	"fmt"
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
	_, err := m.sendCommand(macAddress, CmdSleep, nil)
	return err
}

// KeepAlive sends a keep-alive command
func (m *Manager) KeepAlive(macAddress string) error {
	_, err := m.sendCommand(macAddress, CmdKeepAlive, nil)
	return err
}

// GetMetadata retrieves device metadata (model, firmware, etc.)
func (m *Manager) GetMetadata(macAddress string) (map[string]string, error) {
	response, err := m.sendCommand(macAddress, CmdGetHardwareInfo, nil)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	if len(response.Data) >= 1 {
		modelID := int(response.Data[0])
		metadata["model_id"] = fmt.Sprintf("%d", modelID)
		metadata["model_name"] = m.getModelName(modelID)
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
	required := []string{CharCommand, CharCommandResponse, CharQuery, CharQueryResponse}

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

	// Build command
	cmd := []byte{commandID}
	if data != nil {
		cmd = append(cmd, data...)
	}

	// Send command
	_, err := cmdChar.WriteWithoutResponse(cmd)
	if err != nil {
		return Response{}, fmt.Errorf("failed to send command: %v", err)
	}

	// Wait for response
	select {
	case response := <-conn.responses:
		if response.CommandID == commandID {
			return response, nil
		}
	case <-time.After(5 * time.Second):
		return Response{}, fmt.Errorf("command timeout")
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

	// Build query
	query := []byte{queryID}
	if data != nil {
		query = append(query, data...)
	}

	// Send query
	_, err := queryChar.WriteWithoutResponse(query)
	if err != nil {
		return Response{}, fmt.Errorf("failed to send query: %v", err)
	}

	// Wait for response
	select {
	case response := <-conn.responses:
		if response.CommandID == queryID {
			return response, nil
		}
	case <-time.After(5 * time.Second):
		return Response{}, fmt.Errorf("query timeout")
	}

	return Response{}, fmt.Errorf("no response received")
}

func (m *Manager) handleNotification(macAddress string, data []byte) {
	if len(data) < 2 {
		return
	}

	conn := m.getConnection(macAddress)
	if conn == nil {
		return
	}

	response := Response{
		CommandID: data[0],
		Status:    data[1],
		Data:      data[2:],
		Timestamp: time.Now(),
	}

	select {
	case conn.responses <- response:
	default:
		// Channel full, drop oldest
		select {
		case <-conn.responses:
			conn.responses <- response
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

func (m *Manager) getModelName(modelID int) string {
	switch modelID {
	case 50:
		return "HERO9"
	case 55:
		return "HERO10"
	case 62:
		return "HERO11"
	case 63:
		return "HERO12"
	case 64:
		return "HERO13"
	default:
		return fmt.Sprintf("Unknown (%d)", modelID)
	}
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
