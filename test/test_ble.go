package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/logger"
	"tinygo.org/x/bluetooth"
)

func main() {
	fmt.Println("=== GoPro BLE Manager Test ===")
	fmt.Println("This test uses the production BLE Manager to demonstrate proper GoPro connectivity")
	fmt.Println()

	// Setup logging
	log := logger.GetLogger()
	fmt.Println("✓ Logger initialized")

	// Enable BLE interface
	adapter := bluetooth.DefaultAdapter
	must("enable adapter", adapter.Enable())
	fmt.Println("✓ BLE adapter enabled")

	// Create BLE Manager
	bleManager := ble.NewManager(adapter, log)
	
	// Set up metadata callback to verify it gets invoked
	bleManager.SetMetadataCallback(func(metadata ble.CameraMetadata) {
		fmt.Println("\n=== METADATA CALLBACK INVOKED ===")
		fmt.Printf("✓ MAC Address: %s\n", metadata.MACAddress)
		fmt.Printf("✓ Model: %s (ID: %d)\n", metadata.ModelName, metadata.ModelID)
		fmt.Printf("✓ Firmware: %s\n", metadata.FirmwareVersion)
		fmt.Printf("✓ Serial: %s\n", metadata.SerialNumber)
		fmt.Printf("✓ Battery: %d%%\n", metadata.BatteryLevel)
		fmt.Printf("✓ WiFi SSID: %s\n", metadata.WiFiSSID)
		fmt.Printf("✓ WiFi Password: %s\n", metadata.WiFiPassword)
		fmt.Println("=================================")
	})
	
	err := bleManager.Start()
	must("start BLE manager", err)
	fmt.Println("✓ BLE Manager started with metadata callback")
	defer bleManager.Stop()

	// Scan for GoPro devices using BLE Manager
	fmt.Println("\nScanning for GoPro devices using BLE Manager...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var foundDevice *ble.Device
	var mu sync.Mutex

	// Start scanning with live callback
	err = bleManager.StartScanningWithCallback(ctx, func(device ble.Device) {
		mu.Lock()
		defer mu.Unlock()
		if foundDevice == nil {
			foundDevice = &device
			fmt.Printf("✓ Found GoPro: %s (%s) RSSI: %d\n", 
				device.Name, device.MACAddress, device.RSSI)
			cancel() // Stop scanning
		}
	})
	must("start scanning", err)

	// Wait for device discovery or timeout
	<-ctx.Done()
	bleManager.StopScanning()

	mu.Lock()
	if foundDevice == nil {
		mu.Unlock()
		log.Fatal("No GoPro devices found within 10 seconds")
	}
	mu.Unlock()

	fmt.Printf("✓ Selected device: %s (%s)\n", foundDevice.Name, foundDevice.MACAddress)

	// DEBUGGING: Use full Connect() but with detailed timeout tracking
	fmt.Printf("\n1. Connecting to %s with timeout monitoring...\n", foundDevice.Name)
	
	// Create a channel to track Connect() progress
	connectDone := make(chan error, 1)
	connectStart := time.Now()
	
	// Run Connect() in background with progress monitoring
	go func() {
		fmt.Println("\n1a. Starting BLE Manager Connect() operation...")
		err := bleManager.Connect(foundDevice.MACAddress)
		connectDone <- err
	}()
	
	// Monitor with timeout
	select {
	case err := <-connectDone:
		duration := time.Since(connectStart)
		if err != nil {
			fmt.Printf("✗ Connect() failed after %v: %v\n", duration, err)
			return
		} else {
			fmt.Printf("✓ Connect() successful after %v\n", duration)
		}
	case <-time.After(30 * time.Second):
		fmt.Printf("✗ Connect() TIMEOUT after 30 seconds - Connection hung!\n")
		fmt.Printf("   This means the issue is inside Connect() method\n")
		fmt.Printf("   Last log was at: %v\n", time.Since(connectStart))
		return
	}
	
	fmt.Println("\n--- Connect() completed successfully, now testing individual operations ---")

	// Step 2: Test WiFi credentials (should work since Connect() succeeded)
	fmt.Println("\n2. Testing WiFi credentials access...")
	done := make(chan error, 1)
	var ssid, password string
	
	go func() {
		var err error
		ssid, password, err = bleManager.GetWifiCredentials(foundDevice.MACAddress)
		done <- err
	}()
	
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("✗ Failed to get WiFi credentials: %v\n", err)
		} else {
			fmt.Printf("✓ WiFi SSID: %s\n", ssid)
			fmt.Printf("✓ WiFi Password: %s\n", password)
			fmt.Println("✓ Application-level 'pairing' successful!")
		}
	case <-time.After(10 * time.Second):
		fmt.Println("✗ GetWifiCredentials TIMEOUT (10 seconds)")
		return
	}

	// Test additional operations
	fmt.Println("\n3. Testing additional operations...")
	
	// Test battery level
	fmt.Println("\n3a. Testing GetBatteryLevel...")
	go func() {
		battery, err := bleManager.GetBatteryLevel(foundDevice.MACAddress)
		if err != nil {
			done <- err
		} else {
			fmt.Printf("✓ Battery Level: %d%%\n", battery)
			done <- nil
		}
	}()
	
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("✗ GetBatteryLevel failed: %v\n", err)
		}
	case <-time.After(10 * time.Second):
		fmt.Println("✗ GetBatteryLevel TIMEOUT")
	}
	
	// Test hardware info
	fmt.Println("\n3b. Testing GetHardwareInfo...")
	go func() {
		hwInfo, err := bleManager.GetHardwareInfo(foundDevice.MACAddress)
		if err != nil {
			done <- err
		} else {
			fmt.Printf("✓ Model: %s (ID: %d)\n", hwInfo.ModelName, hwInfo.ModelNumber)
			fmt.Printf("✓ Firmware: %s\n", hwInfo.FirmwareVersion)
			done <- nil
		}
	}()
	
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("✗ GetHardwareInfo failed: %v\n", err)
		}
	case <-time.After(10 * time.Second):
		fmt.Println("✗ GetHardwareInfo TIMEOUT")
	}
	
	// Test pairing state
	fmt.Println("\n3c. Testing IsPaired...")
	go func() {
		isPaired, err := bleManager.IsPaired(foundDevice.MACAddress)
		if err != nil {
			done <- err
		} else {
			fmt.Printf("✓ Device is paired: %v\n", isPaired)
			done <- nil
		}
	}()
	
	select {
	case err := <-done:
		if err != nil {
			fmt.Printf("✗ IsPaired failed: %v\n", err)
		}
	case <-time.After(10 * time.Second):
		fmt.Println("✗ IsPaired TIMEOUT")
	}

	// Cleanup using BLE Manager
	fmt.Println("\n4. Disconnecting using BLE Manager...")
	err = bleManager.Disconnect(foundDevice.MACAddress)
	if err != nil {
		fmt.Printf("✗ Disconnect error: %v\n", err)
	} else {
		fmt.Println("✓ Disconnected successfully")
	}
	
	fmt.Println("\n=== Test Complete ===")
}

func must(action string, err error) {
	if err != nil {
		log.Fatalf("Failed to %s: %v\n", action, err)
	}
}