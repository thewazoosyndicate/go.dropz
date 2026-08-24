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
	m.log.Debug("Connecting to WiFi network", "ssid", ssid)

	connName := fmt.Sprintf("dropz-%s", ssid)

	// Clean up any stale connection profiles for this SSID
	for _, name := range []string{ssid, connName} {
		_ = exec.CommandContext(ctx, "nmcli", "connection", "delete", "id", name).Run()
	}

	// Create a proper connection profile
	// wifi.powersave 2 = disabled. NetworkManager's default powersave
	// collapses GoPro AP throughput to under 1 MB/s on many NICs.
	addCmd := exec.CommandContext(ctx, "nmcli", "connection", "add",
		"type", "wifi",
		"con-name", connName,
		"ifname", "*",
		"ssid", ssid,
		"wifi-sec.key-mgmt", "wpa-psk",
		"wifi-sec.psk", password,
		"wifi.powersave", "2")

	if addOutput, addErr := addCmd.CombinedOutput(); addErr != nil {
		return fmt.Errorf("failed to create WiFi connection for %s: %w (nmcli: %s)",
			ssid, addErr, strings.TrimSpace(string(addOutput)))
	}

	// Try to activate, retrying while the AP becomes visible
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		// Trigger a WiFi rescan so nmcli can discover the new AP
		_ = exec.CommandContext(ctx, "nmcli", "device", "wifi", "rescan").Run()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}

		upCmd := exec.CommandContext(ctx, "nmcli", "connection", "up", "id", connName)
		if output, err := upCmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%w (nmcli: %s)", err, strings.TrimSpace(string(output)))
			m.log.Debug("WiFi activation attempt failed", "ssid", ssid, "attempt", attempt, "max", 10, "err", lastErr)
		} else {
			m.log.Debug("WiFi connection activated", "connection", connName)
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
		return fmt.Errorf("failed to activate WiFi connection %s after 10 attempts: %w", connName, lastErr)
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

// Disconnect brings down only the dropz-created camera profiles; the user's
// own WiFi connections must survive a sync. Bounded by its own timeout so a
// hung nmcli cannot stall sync teardown forever.
func (m *WiFiManager) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "nmcli", "-t", "-f", "NAME,TYPE", "connection", "show", "--active").Output()
	if err != nil {
		return fmt.Errorf("failed to get active connections: %w", err)
	}

	var firstErr error
	for _, line := range strings.Split(string(output), "\n") {
		// TrimSuffix, not Split: an SSID may itself contain colons
		if !strings.HasSuffix(line, ":802-11-wireless") {
			continue
		}
		name := strings.TrimSuffix(line, ":802-11-wireless")
		if !strings.HasPrefix(name, "dropz-") {
			continue
		}
		if out, downErr := exec.CommandContext(ctx, "nmcli", "connection", "down", "id", name).CombinedOutput(); downErr != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to disconnect from %s: %w (nmcli: %s)",
				name, downErr, strings.TrimSpace(string(out)))
		}
	}
	return firstErr
}

// isConnectedTo checks if currently connected to the specified SSID
func (m *WiFiManager) isConnectedTo(ssid string) bool {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE,STATE", "connection", "show", "--active")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Transient during association; the caller polls this in a loop
		m.log.Debug("Failed to check active WiFi connections", "err", err, "output", strings.TrimSpace(string(output)))
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
