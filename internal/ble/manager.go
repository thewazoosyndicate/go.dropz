package ble

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/ble/tlv"
	"github.com/dropz/dropz/internal/model"
	"tinygo.org/x/bluetooth"
)

// errAlreadyConnected signals a no-op connect; callers treat it as success.
var errAlreadyConnected = errors.New("device already connected")

// ErrBluetoothUnavailable is returned by scan and connect operations when no
// working adapter exists; the app runs degraded instead of refusing to start.
// Aliased from model so the gRPC layer can map it without importing ble.
var ErrBluetoothUnavailable = model.ErrBluetoothUnavailable

// ErrBondLost marks repeated link aborts against a camera that is still
// advertising: the camera dropped its side of the bond (observed on HERO13
// when pairing finish never landed) and only re-pairing recovers.
var ErrBondLost = errors.New("BLE bond lost, camera must be paired again")

const (
	bondLossAbortThreshold = 3               // whole connect calls, not attempts
	bondLossSeenWindow     = 2 * time.Minute // camera must still be advertising
)

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
	device *bluetooth.Device
	chars  map[string]bluetooth.DeviceCharacteristic
	// One collector per notify characteristic: the spec accumulates fragments
	// per source UUID, and a push on one char must not corrupt a fragmented
	// response arriving on another.
	collectors map[string]*tlv.FragmentCollector
	tracker    *tlv.ResponseTracker
	// Characteristics we subscribed on. Unsubscribed at teardown: BlueZ
	// keeps the watcher across reconnects, and a leaked one delivers every
	// notification twice, which breaks multi-packet reassembly.
	subscribed []bluetooth.DeviceCharacteristic
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
	scanDone          chan struct{} // closed when StopScanning() is called; nil while not scanning
	// Connect gate. Counted, not a single channel: status checks and syncs
	// connect concurrently, and an overwritten channel strands the scanner
	// forever on a channel nobody will close.
	connectingCount   int
	connectingDone    chan struct{}                                        // non-nil while connectingCount > 0
	discoveryCallback DeviceDiscoveryCallback                              // callback for live discovery updates
	metadataCallback  MetadataUpdateFunc                                   // callback for metadata updates during connection
	statusCallback    func(macAddress string, statusID byte, value []byte) // push notification callback
	sessions          map[string]bool                                      // addresses with an active logical session
	connectAborts     map[string]int                                       // consecutive fully-aborted connect calls per address
}

// TryAcquireSession claims exclusive use of one camera for a logical BLE
// session (connect, operate, disconnect). Two concurrent sessions to the
// same device collide in BlueZ ("In Progress") and one side's teardown
// sleeps the camera under the other, so every session type must hold this.
// Returns the release func and true, or nil and false when busy.
func (m *Manager) TryAcquireSession(macAddress string) (func(), bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.sessions[macAddress] {
		return nil, false
	}
	m.sessions[macAddress] = true
	return func() {
		m.mutex.Lock()
		delete(m.sessions, macAddress)
		m.mutex.Unlock()
	}, true
}

// AcquireSession polls TryAcquireSession until it succeeds or the timeout
// passes. For sessions that should wait out a short-lived holder.
func (m *Manager) AcquireSession(macAddress string, timeout time.Duration) (func(), error) {
	deadline := time.Now().Add(timeout)
	for {
		if release, ok := m.TryAcquireSession(macAddress); ok {
			return release, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("camera %s is busy with another BLE session", macAddress)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// NewManager creates a new BLE manager.
// A nil adapter yields a degraded manager: scanning and connecting return
// ErrBluetoothUnavailable so the rest of the app can run without Bluetooth.
func NewManager(adapter *bluetooth.Adapter, log *slog.Logger) *Manager {
	m := &Manager{
		adapter:           adapter,
		discoveredDevices: make(map[string]*Device),
		conns:             make(map[string]*conn),
		sessions:          make(map[string]bool),
		connectAborts:     make(map[string]int),
		log:               log.With("component", "ble"),
	}
	if adapter != nil {
		// Our own teardown drops conn state before device.Disconnect, so a
		// disconnect that still has conn state is the camera dropping the link.
		adapter.SetConnectHandler(func(device bluetooth.Device, connected bool) {
			if connected {
				return
			}
			addr := device.Address.String()
			if m.getConn(addr) != nil {
				m.log.Info("Camera dropped BLE link", "ble_addr", addr)
			}
		})
	}
	return m
}

// Available reports whether a working Bluetooth adapter is present.
func (m *Manager) Available() bool {
	return m.adapter != nil
}

// newConn creates the per-device connection state and wires push
// notifications (0x93) through to the manager-level status callback.
func (m *Manager) newConn(macAddress string, device *bluetooth.Device) *conn {
	c := &conn{
		device:     device,
		chars:      make(map[string]bluetooth.DeviceCharacteristic),
		collectors: make(map[string]*tlv.FragmentCollector),
		tracker:    tlv.NewResponseTracker(),
	}
	c.tracker.SetLogger(m.log.With("ble_addr", macAddress))
	c.tracker.SetPushHandler(func(addr string, msg *tlv.TLVMessage) {
		m.mutex.RLock()
		cb := m.statusCallback
		m.mutex.RUnlock()
		if cb == nil {
			return
		}
		// Async: the handler runs under the tracker's lock on the
		// notification goroutine, and the callback persists to the store;
		// doing that inline blocks command routing for the whole write.
		// Pushes are low-rate (battery), so a goroutine per message is fine.
		go func() {
			for id, value := range parseTLVPairs(msg.Payload) {
				cb(macAddress, id, value)
			}
		}()
	})
	return c
}

// getConn returns the connection for a device, if any.
func (m *Manager) getConn(macAddress string) *conn {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.conns[macAddress]
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

// Stop stops the BLE manager
func (m *Manager) Stop() error {
	m.log.Debug("Stopping BLE manager")

	_ = m.StopScanning()

	m.mutex.RLock()
	addresses := make([]string, 0, len(m.conns))
	for macAddress := range m.conns {
		addresses = append(addresses, macAddress)
	}
	m.mutex.RUnlock()

	for _, macAddress := range addresses {
		if err := m.Disconnect(macAddress); err != nil {
			m.log.Warn("Failed to disconnect", "ble_addr", macAddress, "err", err)
		}
	}

	m.log.Debug("BLE manager stopped")
	return nil
}
