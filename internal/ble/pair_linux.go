//go:build linux

package ble

import (
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const agentPath = "/org/bluez/dropz/agent"

// justWorksAgent implements org.bluez.Agent1 for unauthenticated "Just Works" pairing.
type justWorksAgent struct{}

func (justWorksAgent) Release() *dbus.Error { return nil }
func (justWorksAgent) RequestConfirmation(device dbus.ObjectPath, passkey uint32) *dbus.Error {
	return nil
}
func (justWorksAgent) Cancel() *dbus.Error                                              { return nil }
func (justWorksAgent) AuthorizeService(device dbus.ObjectPath, uuid string) *dbus.Error { return nil }

// pairViaDbus pairs via org.bluez.Device1 with a registered "Just Works" agent.
// Pair() is called asynchronously because BlueZ hangs when the device is already
// connected (tinygo calls Connect() before we pair). We watch for the Paired
// property via PropertiesChanged signals instead of relying on Pair()'s return.
func (m *Manager) pairViaDbus(macAddress string) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return fmt.Errorf("failed to connect to system D-Bus: %w", err)
	}

	devPath := "/org/bluez/hci0/dev_" + strings.ReplaceAll(macAddress, ":", "_")
	device := conn.Object("org.bluez", dbus.ObjectPath(devPath))

	// Check if already paired
	paired, err := device.GetProperty("org.bluez.Device1.Paired")
	if err == nil {
		if v, ok := paired.Value().(bool); ok && v {
			m.log.Info("Device already paired, skipping", "device", macAddress)
			return nil
		}
	}

	// Register Just Works agent
	if err := conn.Export(justWorksAgent{}, dbus.ObjectPath(agentPath), "org.bluez.Agent1"); err != nil {
		m.log.Warn("Failed to export pairing agent", "err", err)
	}

	agentMgr := conn.Object("org.bluez", "/org/bluez")
	call := agentMgr.Call("org.bluez.AgentManager1.RegisterAgent", 0, dbus.ObjectPath(agentPath), "NoInputNoOutput")
	if call.Err != nil {
		m.log.Warn("Failed to register pairing agent", "err", call.Err)
	}
	defer func() {
		call := agentMgr.Call("org.bluez.AgentManager1.UnregisterAgent", 0, dbus.ObjectPath(agentPath))
		if call.Err != nil {
			m.log.Warn("Failed to unregister pairing agent", "err", call.Err)
		}
	}()

	// Watch for Paired property changes via D-Bus signals
	matchRule := fmt.Sprintf(
		"type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='%s',arg0='org.bluez.Device1'",
		devPath,
	)
	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule)
	sigCh := make(chan *dbus.Signal, 10)
	conn.Signal(sigCh)
	defer func() {
		conn.RemoveSignal(sigCh)
		conn.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, matchRule)
	}()

	m.log.Info("Initiating D-Bus pairing", "device", macAddress)

	// Fire Pair() asynchronously — it may never return if device is already connected
	pairDone := make(chan *dbus.Call, 1)
	device.Go("org.bluez.Device1.Pair", 0, pairDone)

	timeout := time.After(15 * time.Second)
	for {
		select {
		case sig := <-sigCh:
			if len(sig.Body) < 2 {
				continue
			}
			changed, ok := sig.Body[1].(map[string]dbus.Variant)
			if !ok {
				continue
			}
			if v, exists := changed["Paired"]; exists {
				if paired, ok := v.Value().(bool); ok && paired {
					m.log.Info("D-Bus pairing successful", "device", macAddress)
					return nil
				}
			}

		case result := <-pairDone:
			if result.Err != nil {
				if strings.Contains(result.Err.Error(), "AlreadyExists") {
					m.log.Info("Device already paired", "device", macAddress)
					return nil
				}
				return fmt.Errorf("D-Bus Pair() failed: %w", result.Err)
			}
			m.log.Info("D-Bus pairing successful", "device", macAddress)
			return nil

		case <-timeout:
			return fmt.Errorf("D-Bus pairing timed out after 15s for %s", macAddress)
		}
	}
}
