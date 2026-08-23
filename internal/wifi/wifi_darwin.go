//go:build darwin

package wifi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// detectWiFiInterface finds the macOS Wi-Fi interface name (typically "en0")
func detectWiFiInterface() (string, error) {
	out, err := exec.Command("networksetup", "-listallhardwareports").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to list hardware ports: %v", err)
	}

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if strings.Contains(line, "Wi-Fi") || strings.Contains(line, "AirPort") {
			// The device line follows the "Hardware Port" line
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "Device:") {
					return strings.TrimSpace(strings.TrimPrefix(lines[j], "Device:")), nil
				}
			}
		}
	}
	return "", fmt.Errorf("Wi-Fi interface not found")
}

// Connect connects to a GoPro WiFi network using macOS networksetup
func (m *WiFiManager) Connect(ctx context.Context, ssid, password string) error {
	m.log.Debugf("Connecting to WiFi network: ssid=%s", ssid)

	iface, err := detectWiFiInterface()
	if err != nil {
		return fmt.Errorf("failed to detect Wi-Fi interface: %v", err)
	}

	// Try to connect, retrying while the AP becomes visible
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		cmd := exec.CommandContext(ctx, "networksetup", "-setairportnetwork", iface, ssid, password)
		if output, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(output)))
			m.log.Debugf("WiFi connection attempt %d/10 failed: %v", attempt, lastErr)
		} else {
			out := strings.TrimSpace(string(output))
			// networksetup may print an error message even with exit code 0
			if out != "" && !strings.Contains(strings.ToLower(out), "error") {
				m.log.Debugf("WiFi connection command succeeded: %s", out)
			} else if strings.Contains(strings.ToLower(out), "error") {
				lastErr = fmt.Errorf("networksetup: %s", out)
				m.log.Debugf("WiFi connection attempt %d/10 failed: %v", attempt, lastErr)
				time.Sleep(2 * time.Second)
				continue
			}
			lastErr = nil
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(2 * time.Second)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to connect to WiFi %s after 10 attempts: %v", ssid, lastErr)
	}

	// Poll until connected
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

// Disconnect disconnects from the current GoPro WiFi network
func (m *WiFiManager) Disconnect() error {
	iface, err := detectWiFiInterface()
	if err != nil {
		return fmt.Errorf("failed to detect Wi-Fi interface: %v", err)
	}

	// Get current network to remove it from preferred list
	cmd := exec.Command("networksetup", "-getairportnetwork", iface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get current network: %v", err)
	}

	// Output format: "Current Wi-Fi Network: <SSID>"
	out := strings.TrimSpace(string(output))
	prefix := "Current Wi-Fi Network: "
	if !strings.HasPrefix(out, prefix) {
		// Not connected to any network
		return nil
	}
	ssid := strings.TrimPrefix(out, prefix)

	// Remove from preferred networks to disassociate
	cmd = exec.Command("networksetup", "-removepreferredwirelessnetwork", iface, ssid)
	if output, err := cmd.CombinedOutput(); err != nil {
		m.log.Warnf("Failed to remove preferred network %s: %v (%s)", ssid, err, strings.TrimSpace(string(output)))
	}

	return nil
}

// isConnectedTo checks if currently connected to the specified SSID
func (m *WiFiManager) isConnectedTo(ssid string) bool {
	iface, err := detectWiFiInterface()
	if err != nil {
		m.log.Errorf("Failed to detect Wi-Fi interface: %v", err)
		return false
	}

	cmd := exec.Command("networksetup", "-getairportnetwork", iface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.log.Errorf("Failed to check WiFi connection: error=%v output=%s", err, string(output))
		return false
	}

	// Output format: "Current Wi-Fi Network: <SSID>"
	out := strings.TrimSpace(string(output))
	return strings.HasSuffix(out, ": "+ssid)
}
