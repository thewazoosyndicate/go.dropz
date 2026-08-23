//go:build darwin

package wifi

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// detectWiFiInterface finds the macOS Wi-Fi interface name (typically "en0")
func detectWiFiInterface() (string, error) {
	out, err := exec.Command("networksetup", "-listallhardwareports").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to list hardware ports: %w", err)
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

// findWifiJoinHelper locates the compiled CoreWLAN helper (scripts/wifi_join.swift).
// networksetup alone reports success without connecting on macOS 15, so the
// helper scans with CoreWLAN until the AP is visible before connecting.
func findWifiJoinHelper() (string, bool) {
	candidates := []string{"bin/wifi_join", "../bin/wifi_join"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "wifi_join"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// Connect connects to a GoPro WiFi network. Prefers the CoreWLAN helper
// when present; falls back to bare networksetup retries otherwise.
func (m *WiFiManager) Connect(ctx context.Context, ssid, password string) error {
	m.log.Debug("Connecting to WiFi network", "ssid", ssid)

	iface, err := detectWiFiInterface()
	if err != nil {
		return fmt.Errorf("failed to detect Wi-Fi interface: %w", err)
	}

	if helper, ok := findWifiJoinHelper(); ok {
		m.log.Debug("Using CoreWLAN helper", "helper", helper, "interface", iface)
		output, err := exec.CommandContext(ctx, helper, ssid, password, iface).CombinedOutput()
		out := strings.TrimSpace(string(output))
		if err != nil {
			return fmt.Errorf("CoreWLAN helper failed: %w (%s)", err, out)
		}
		m.log.Debug("CoreWLAN helper connected", "output", out)
		return nil
	}
	m.log.Warn("wifi_join helper not found, falling back to networksetup (unreliable on macOS 15)")

	// Try to connect, retrying while the AP becomes visible
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		cmd := exec.CommandContext(ctx, "networksetup", "-setairportnetwork", iface, ssid, password)
		if output, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(output)))
			m.log.Debug("WiFi connection attempt failed", "attempt", attempt, "max", 10, "err", lastErr)
		} else {
			out := strings.TrimSpace(string(output))
			// networksetup may print an error message even with exit code 0
			if out != "" && !strings.Contains(strings.ToLower(out), "error") {
				m.log.Debug("WiFi connection command succeeded", "output", out)
			} else if strings.Contains(strings.ToLower(out), "error") {
				lastErr = fmt.Errorf("networksetup: %s", out)
				m.log.Debug("WiFi connection attempt failed", "attempt", attempt, "max", 10, "err", lastErr)
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
		return fmt.Errorf("failed to connect to WiFi %s after 10 attempts: %w", ssid, lastErr)
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
		return fmt.Errorf("failed to detect Wi-Fi interface: %w", err)
	}

	// Get current network to remove it from preferred list
	cmd := exec.Command("networksetup", "-getairportnetwork", iface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get current network: %w", err)
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
		m.log.Warn("Failed to remove preferred network", "ssid", ssid, "err", err, "output", strings.TrimSpace(string(output)))
	}

	return nil
}

// isConnectedTo checks if currently connected to the specified SSID
func (m *WiFiManager) isConnectedTo(ssid string) bool {
	iface, err := detectWiFiInterface()
	if err != nil {
		m.log.Error("Failed to detect Wi-Fi interface", "err", err)
		return false
	}

	cmd := exec.Command("networksetup", "-getairportnetwork", iface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		m.log.Error("Failed to check WiFi connection", "err", err, "output", string(output))
		return false
	}

	// Output format: "Current Wi-Fi Network: <SSID>"
	out := strings.TrimSpace(string(output))
	return strings.HasSuffix(out, ": "+ssid)
}
