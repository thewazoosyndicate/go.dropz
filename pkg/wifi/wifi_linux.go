//go:build linux

package wifi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Connect connects to a GoPro WiFi network using NetworkManager (nmcli)
func (m *WiFiManager) Connect(ctx context.Context, ssid, password string) error {
	m.log.Debugf("Connecting to WiFi network: ssid=%s", ssid)

	connName := fmt.Sprintf("dropz-%s", ssid)

	// Clean up any stale connection profiles for this SSID
	for _, name := range []string{ssid, connName} {
		exec.CommandContext(ctx, "nmcli", "connection", "delete", "id", name).Run()
	}

	// Create a proper connection profile
	addCmd := exec.CommandContext(ctx, "nmcli", "connection", "add",
		"type", "wifi",
		"con-name", connName,
		"ifname", "*",
		"ssid", ssid,
		"wifi-sec.key-mgmt", "wpa-psk",
		"wifi-sec.psk", password)

	if addOutput, addErr := addCmd.CombinedOutput(); addErr != nil {
		m.log.Errorf("Failed to create connection profile: error=%v nmcli_output=%s", addErr, string(addOutput))
		return fmt.Errorf("failed to create WiFi connection for %s: %v", ssid, addErr)
	}

	// Try to activate, retrying while the AP becomes visible
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		// Trigger a WiFi rescan so nmcli can discover the new AP
		exec.CommandContext(ctx, "nmcli", "device", "wifi", "rescan").Run()
		time.Sleep(2 * time.Second)

		upCmd := exec.CommandContext(ctx, "nmcli", "connection", "up", "id", connName)
		if output, err := upCmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%v (nmcli: %s)", err, strings.TrimSpace(string(output)))
			m.log.Debugf("WiFi activation attempt %d/10 failed: %v", attempt, lastErr)
		} else {
			m.log.Debugf("WiFi connection activated: connection=%s", connName)
			lastErr = nil
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to activate WiFi connection %s after 10 attempts: %v", connName, lastErr)
	}

	for i := 0; i < 10; i++ {
		if m.isConnectedTo(ssid) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	return fmt.Errorf("timed out waiting for connection to %s", ssid)
}

// Disconnect disconnects from the current WiFi network
func (m *WiFiManager) Disconnect() error {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE", "connection", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		m.log.Errorf("Failed to get active WiFi connections: error=%v", err)
		return fmt.Errorf("failed to get active connections: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":802-11-wireless") {
			parts := strings.Split(line, ":")
			if len(parts) < 1 {
				continue
			}

			ssid := parts[0]

			cmd = exec.Command("nmcli", "connection", "down", "id", ssid)
			if output, err := cmd.CombinedOutput(); err != nil {
				altName := fmt.Sprintf("dropz-%s", ssid)
				cmd = exec.Command("nmcli", "connection", "down", "id", altName)
				if altOutput, altErr := cmd.CombinedOutput(); altErr != nil {
					m.log.Errorf("Failed to disconnect from WiFi network: ssid=%s error=%v nmcli_output=%s alt_error=%v alt_output=%s",
						ssid, err, string(output), altErr, string(altOutput))
					return fmt.Errorf("failed to disconnect from %s: %v (nmcli: %s)", ssid, err, strings.TrimSpace(string(output)))
				}
			}
		}
	}

	return nil
}

// isConnectedTo checks if currently connected to the specified SSID
func (m *WiFiManager) isConnectedTo(ssid string) bool {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE,STATE", "connection", "show", "--active")
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.log.Errorf("Failed to check active WiFi connections: error=%v output=%s", err, string(output))
		return false
	}

	altName := fmt.Sprintf("dropz-%s", ssid)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && (parts[0] == ssid || parts[0] == altName) && parts[2] == "activated" {
			return true
		}
	}
	return false
}
