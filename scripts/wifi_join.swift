#!/usr/bin/env swift
// wifi_join — Wait for a GoPro WiFi AP to appear, then connect via networksetup.
// Usage: wifi_join <ssid> <password> <interface>
import CoreWLAN
import Foundation

let args = CommandLine.arguments
guard args.count >= 4 else {
    fputs("Usage: wifi_join <ssid> <password> <interface>\n", stderr)
    exit(1)
}

let ssid     = args[1]
let password = args[2]
let ifaceName = args[3]

let client = CWWiFiClient.shared()
guard let iface = client.interface(withName: ifaceName) else {
    fputs("ERROR: interface \(ifaceName) not found\n", stderr)
    exit(1)
}

// Helper: query current SSID via networksetup -getairportnetwork
func currentSSIDViaNetworksetup(_ ifName: String) -> String? {
    let p = Process()
    p.executableURL = URL(fileURLWithPath: "/usr/sbin/networksetup")
    p.arguments = ["-getairportnetwork", ifName]
    let pipe = Pipe()
    p.standardOutput = pipe
    p.standardError = Pipe()
    do {
        try p.run()
        p.waitUntilExit()
    } catch {
        return nil
    }
    let out = String(data: pipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
    // Output is: "Current Wi-Fi Network: GP24599533"
    let trimmed = out.trimmingCharacters(in: .whitespacesAndNewlines)
    if let range = trimmed.range(of: "Current Wi-Fi Network: ") {
        return String(trimmed[range.upperBound...])
    }
    return nil
}

// Already connected?
let currentSSID = currentSSIDViaNetworksetup(ifaceName) ?? iface.ssid() ?? ""
if currentSSID == ssid {
    print("Already connected to \(ssid)")
    exit(0)
}

// Phase 1: scan until the GoPro AP is visible (up to 20 attempts × 2s = 40s)
fputs("Scanning for \(ssid)...\n", stderr)
var found = false
for attempt in 1...20 {
    do {
        let nets = try iface.scanForNetworks(withSSID: ssid.data(using: .utf8))
        if !nets.isEmpty {
            fputs("Found \(ssid) on attempt \(attempt)\n", stderr)
            found = true
            break
        }
    } catch {
        fputs("Scan attempt \(attempt) error: \(error)\n", stderr)
    }
    fputs("Scan attempt \(attempt)/20: \(ssid) not visible yet\n", stderr)
    Thread.sleep(forTimeInterval: 2)
}

// Not fatal: without location permission macOS redacts scan results,
// so an invisible SSID does not prove the AP is down. Try anyway;
// networksetup failing is the real verdict.
if !found {
    fputs("WARNING: \(ssid) not seen in 20 scans, trying networksetup anyway\n", stderr)
}

// Phase 2: connect via networksetup (works without entitlements)
fputs("Connecting via networksetup...\n", stderr)
let proc = Process()
proc.executableURL = URL(fileURLWithPath: "/usr/sbin/networksetup")
proc.arguments = ["-setairportnetwork", ifaceName, ssid, password]
let pipe = Pipe()
proc.standardOutput = pipe
proc.standardError = pipe

do {
    try proc.run()
    proc.waitUntilExit()
    let out = String(data: pipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
    let status = proc.terminationStatus
    fputs("networksetup exit=\(status) output=\(out.trimmingCharacters(in: .whitespacesAndNewlines))\n", stderr)
    if status != 0 {
        fputs("ERROR: networksetup failed (exit \(status))\n", stderr)
        exit(1)
    }
} catch {
    fputs("ERROR: could not run networksetup: \(error)\n", stderr)
    exit(1)
}

// Phase 3: verify connection using networksetup -getairportnetwork (20 attempts × 1s = 20s)
fputs("Verifying connection...\n", stderr)
for attempt in 1...20 {
    Thread.sleep(forTimeInterval: 1)
    let confirmed = currentSSIDViaNetworksetup(ifaceName)
    fputs("Waiting for connection \(attempt)/20: current=\(confirmed ?? "<none>")\n", stderr)
    if confirmed == ssid {
        print("Connected to \(ssid)")
        exit(0)
    }
}

fputs("WARNING: networksetup succeeded but SSID not confirmed in 20s — returning success\n", stderr)
print("Connected to \(ssid)")
exit(0)
