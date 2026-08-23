// ble-probe: hardware validation harness for the OpenGoPro conformance work.
// Every check maps to a row in docs/conformance.md that only a real camera
// can prove. Run on a machine with Bluetooth, next to a charged GoPro.
//
//	ble-probe scan [-duration 30s]     dump advertisements raw + parsed; toggle
//	                                   pairing mode on the camera and watch bit 2
//	ble-probe validate <name> [flags]  full checklist against a paired camera
//	ble-probe pair <name>              first-time pairing checklist
//
// <name> is a case-insensitive fragment of the camera name or BLE address.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/logging"
	"github.com/dropz/dropz/internal/wifi"
	"tinygo.org/x/bluetooth"
)

type report struct {
	pass, fail, manual int
}

func (r *report) ok(name, detail string) {
	r.pass++
	fmt.Printf("[PASS]   %-28s %s\n", name, detail)
}

func (r *report) bad(name, detail string) {
	r.fail++
	fmt.Printf("[FAIL]   %-28s %s\n", name, detail)
}

func (r *report) check(name string, err error, detail string) {
	if err != nil {
		r.bad(name, err.Error())
	} else {
		r.ok(name, detail)
	}
}

func (r *report) note(name, detail string) {
	fmt.Printf("[info]   %-28s %s\n", name, detail)
}

func (r *report) human(name, detail string) {
	r.manual++
	fmt.Printf("[MANUAL] %-28s %s\n", name, detail)
}

func (r *report) summary() int {
	fmt.Printf("\n%d passed, %d failed, %d manual checks\n", r.pass, r.fail, r.manual)
	if r.fail > 0 {
		return 1
	}
	return 0
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]

	adapter := bluetooth.DefaultAdapter
	if adapter == nil {
		fatal("no default Bluetooth adapter")
	}
	if err := adapter.Enable(); err != nil {
		fatal("enable Bluetooth adapter: %v", err)
	}

	switch cmd {
	case "scan":
		fs := flag.NewFlagSet("scan", flag.ExitOnError)
		duration := fs.Duration("duration", 30*time.Second, "how long to scan")
		level := logLevelFlag(fs)
		fs.Parse(args)
		initLogger(*level)
		runScan(adapter, *duration)
	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		doSleep := fs.Bool("sleep", false, "send Sleep on disconnect")
		doWifi := fs.Bool("wifi", false, "also join the camera AP and run HTTP checks")
		doSpeed := fs.Bool("speed", false, "with -wifi: measure throughput turbo off vs on")
		level := logLevelFlag(fs)
		fs.Parse(args)
		initLogger(*level)
		if fs.NArg() != 1 {
			usage()
		}
		os.Exit(runValidate(adapter, fs.Arg(0), *doSleep, *doWifi, *doSpeed))
	case "pair":
		fs := flag.NewFlagSet("pair", flag.ExitOnError)
		level := logLevelFlag(fs)
		fs.Parse(args)
		initLogger(*level)
		if fs.NArg() != 1 {
			usage()
		}
		os.Exit(runPair(adapter, fs.Arg(0)))
	case "settings":
		fs := flag.NewFlagSet("settings", flag.ExitOnError)
		level := logLevelFlag(fs)
		fs.Parse(args)
		initLogger(*level)
		if fs.NArg() != 1 {
			usage()
		}
		os.Exit(runSettings(adapter, fs.Arg(0)))
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ble-probe scan [-duration 30s]")
	fmt.Fprintln(os.Stderr, "       ble-probe validate <name-fragment> [-sleep] [-wifi]")
	fmt.Fprintln(os.Stderr, "       ble-probe pair <name-fragment>")
	fmt.Fprintln(os.Stderr, "       ble-probe settings <name-fragment>")
	os.Exit(2)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

// probeLog goes to stderr so `1>report 2>trace` keeps report and logs apart.
// Default warn: the checklist on stdout is the product, logs are diagnostics.
var probeLog *slog.Logger

func logLevelFlag(fs *flag.FlagSet) *string {
	return fs.String("log-level", "warn", "log level (trace, debug, info, warn, error)")
}

func initLogger(level string) {
	log, _, err := logging.New(logging.Options{Level: level})
	if err != nil {
		fatal("%v", err)
	}
	probeLog = log
}

// rawScan runs a direct adapter scan and calls sighting for each GoPro
// advertisement. Return true from sighting to stop the scan early.
func rawScan(adapter *bluetooth.Adapter, timeout time.Duration, sighting func(bluetooth.ScanResult, ble.AdvInfo) bool) error {
	goProUUID, _ := bluetooth.ParseUUID(ble.AdvertisementService)
	var once sync.Once
	stop := func() { once.Do(func() { adapter.StopScan() }) }

	timer := time.AfterFunc(timeout, stop)
	defer timer.Stop()

	return adapter.Scan(func(a *bluetooth.Adapter, result bluetooth.ScanResult) {
		isGoPro := result.HasServiceUUID(goProUUID) ||
			strings.Contains(strings.ToLower(result.LocalName()), "gopro")
		if !isGoPro {
			return
		}
		if sighting(result, ble.ParseAdvertisement(result)) {
			stop()
		}
	})
}

func matches(result bluetooth.ScanResult, fragment string) bool {
	f := strings.ToLower(fragment)
	return strings.Contains(strings.ToLower(result.LocalName()), f) ||
		strings.Contains(strings.ToLower(result.Address.String()), f)
}

func flagString(info ble.AdvInfo) string {
	onOff := func(b bool) string {
		if b {
			return "ON"
		}
		return "off"
	}
	return fmt.Sprintf("processor=%s wifiAP=%s pairing=%s newMedia=%s model=%d schema=%d serial=%q",
		onOff(info.ProcessorOn), onOff(info.WiFiAPOn), onOff(info.PairingMode),
		onOff(info.NewMedia), info.ModelID, info.SchemaVersion, info.SerialNumber)
}

func dumpRaw(result bluetooth.ScanResult) {
	for _, md := range result.ManufacturerData() {
		fmt.Printf("         manufacturer (company 0x%04X): % X\n", md.CompanyID, md.Data)
	}
	for _, sd := range result.ServiceData() {
		fmt.Printf("         service data (%s): % X\n", sd.UUID.String(), sd.Data)
	}
}

// runScan watches advertisements and reprints a camera whenever its flags
// change. Toggling pairing mode on the camera must flip the pairing flag;
// that is the live proof of the LSB-first bit fix.
func runScan(adapter *bluetooth.Adapter, duration time.Duration) {
	fmt.Printf("Scanning for %v. Toggle pairing mode on a camera and watch the pairing flag.\n\n", duration)
	last := map[string]string{}
	err := rawScan(adapter, duration, func(result bluetooth.ScanResult, info ble.AdvInfo) bool {
		addr := result.Address.String()
		line := flagString(info)
		if last[addr] == line {
			return false
		}
		last[addr] = line
		fmt.Printf("%s  %s  rssi=%d\n", time.Now().Format("15:04:05"), result.LocalName(), result.RSSI)
		fmt.Printf("         addr: %s\n", addr)
		dumpRaw(result)
		fmt.Printf("         %s\n\n", line)
		return false
	})
	if err != nil {
		fatal("scan failed: %v", err)
	}
	if len(last) == 0 {
		fmt.Println("No GoPro advertisements seen. Is a camera awake nearby?")
		os.Exit(1)
	}
}

// findCamera scans until a camera matching fragment is seen.
func findCamera(adapter *bluetooth.Adapter, fragment string, timeout time.Duration) (bluetooth.ScanResult, ble.AdvInfo, error) {
	var found bluetooth.ScanResult
	var adv ble.AdvInfo
	var seen bool
	err := rawScan(adapter, timeout, func(result bluetooth.ScanResult, info ble.AdvInfo) bool {
		if !matches(result, fragment) {
			return false
		}
		// Keep scanning briefly if the serial has not assembled yet:
		// manufacturer and service data can arrive in separate results.
		if seen && info.SerialNumber == "" {
			return false
		}
		found, adv, seen = result, info, true
		return info.SerialNumber != ""
	})
	if err != nil {
		return found, adv, err
	}
	if !seen {
		return found, adv, fmt.Errorf("no camera matching %q seen within %v", fragment, timeout)
	}
	return found, adv, nil
}

func runValidate(adapter *bluetooth.Adapter, fragment string, doSleep, doWifi, doSpeed bool) int {
	r := &report{}
	fmt.Printf("=== dropz hardware validation ===\n\n")

	result, adv, err := findCamera(adapter, fragment, 30*time.Second)
	if err != nil {
		r.bad("discovery", err.Error())
		return r.summary()
	}
	addr := result.Address.String()
	r.ok("discovery", fmt.Sprintf("%s (%s) rssi=%d", result.LocalName(), addr, result.RSSI))
	dumpRaw(result)
	r.note("advertisement", flagString(adv))
	if !adv.Valid {
		r.bad("adv manufacturer data", "no GoPro manufacturer data parsed (company 0x02F2 missing?)")
	}
	if adv.SerialNumber == "" {
		r.note("adv serial", "not assembled (schema 2 with hashed id_hash is normal; schema 3 should assemble)")
	}

	manager := ble.NewManager(adapter, probeLog)
	defer manager.Stop()

	// Connect exercises the spec's readiness gate (hardware info poll),
	// notification subscriptions, 0x50, date/time, AP enable, credentials.
	start := time.Now()
	err = manager.Connect(addr)
	r.check("connect + readiness", err, fmt.Sprintf("full setup in %v", time.Since(start).Round(time.Millisecond)))
	if err != nil {
		return r.summary()
	}

	hw, err := manager.GetHardwareInfo(addr)
	if err != nil {
		r.bad("hardware info", err.Error())
	} else {
		r.ok("hardware info", fmt.Sprintf("%s model=%d fw=%s serial=%s", hw.ModelName, hw.ModelNumber, hw.FirmwareVersion, hw.SerialNumber))
		switch {
		case adv.SerialNumber == "":
			r.note("serial cross-check", "skipped: advertisement serial not assembled")
		case adv.SerialNumber == hw.SerialNumber:
			r.ok("serial cross-check", "advertisement serial matches GATT serial: identity fix proven")
		default:
			r.bad("serial cross-check", fmt.Sprintf("adv %q != gatt %q", adv.SerialNumber, hw.SerialNumber))
		}
		if adv.Valid && adv.ModelID != 0 && hw.ModelNumber != 0 && adv.ModelID != hw.ModelNumber {
			r.bad("model cross-check", fmt.Sprintf("adv model %d != gatt model %d", adv.ModelID, hw.ModelNumber))
		}
	}

	start = time.Now()
	r.check("keep-alive", manager.KeepAlive(addr),
		fmt.Sprintf("LED=66 on GP-0074 answered success on GP-0075 in %v", time.Since(start).Round(time.Millisecond)))

	statuses, err := manager.QueryStatuses(addr, []byte{
		ble.StatusSystemBusy, ble.StatusEncoding, ble.StatusSDCardStatus,
		ble.StatusSDCardRemainingKB, ble.StatusBatteryPercentage,
	})
	if err != nil {
		r.bad("status query", err.Error())
	} else {
		var missing []string
		for _, id := range []byte{ble.StatusSystemBusy, ble.StatusEncoding, ble.StatusSDCardStatus, ble.StatusSDCardRemainingKB, ble.StatusBatteryPercentage} {
			if _, ok := statuses[id]; !ok {
				missing = append(missing, fmt.Sprintf("%d", id))
			}
		}
		detail := fmt.Sprintf("busy=%v encoding=%v sd=%v battery=%v", statuses[ble.StatusSystemBusy], statuses[ble.StatusEncoding], statuses[ble.StatusSDCardStatus], statuses[ble.StatusBatteryPercentage])
		if len(missing) > 0 {
			r.bad("status query", "missing status IDs: "+strings.Join(missing, ","))
		} else {
			r.ok("status query", detail)
		}
	}

	start = time.Now()
	r.check("wifi AP ready (status 69)", manager.WaitForWiFiAPReady(addr, 15*time.Second),
		fmt.Sprintf("AP up after %v", time.Since(start).Round(time.Millisecond)))

	if doWifi {
		validateWifi(r, manager, addr, doSpeed)
	} else {
		r.note("wifi checks", "skipped; rerun with -wifi to validate join, media list, turbo")
	}

	if doSleep {
		r.check("sleep on disconnect", manager.Disconnect(addr), "Sleep 0x05 accepted (camera screen should turn off)")
	} else {
		manager.DisconnectQuietly(addr)
		r.note("disconnect", "quiet (no Sleep); rerun with -sleep to validate Sleep")
	}
	return r.summary()
}

func validateWifi(r *report, manager *ble.Manager, addr string, doSpeed bool) {
	ssid, password, err := manager.GetWifiCredentials(addr)
	if err != nil || ssid == "" || password == "" {
		r.bad("wifi credentials", fmt.Sprintf("ssid=%q err=%v", ssid, err))
		return
	}
	r.ok("wifi credentials", "ssid "+ssid)

	wm := wifi.NewWiFiManager(probeLog)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	start := time.Now()
	if err := wm.Connect(ctx, ssid, password); err != nil {
		r.bad("wifi join", err.Error())
		return
	}
	r.ok("wifi join", fmt.Sprintf("joined %s in %v", ssid, time.Since(start).Round(time.Second)))
	defer func() {
		if err := wm.Disconnect(); err != nil {
			r.note("wifi leave", err.Error())
		}
	}()

	if _, err := wm.GetCameraStatus(ctx); err != nil {
		r.bad("http state", err.Error())
	} else {
		r.ok("http state", "GET /gopro/camera/state answered")
	}

	files, err := wm.ListMedia(ctx)
	if err != nil {
		r.bad("media list", err.Error())
	} else {
		var expanded int
		for _, f := range files {
			if f.Size == 0 {
				expanded++
			}
		}
		r.ok("media list", fmt.Sprintf("%d files (%d expanded group members)", len(files), expanded))
	}

	if err := wm.SetTurboTransfer(ctx, true); err != nil {
		r.note("turbo transfer", "not supported or refused: "+err.Error())
	} else {
		err := wm.SetTurboTransfer(ctx, false)
		r.check("turbo transfer", err, "enabled and disabled (camera briefly showed transfer UI)")
	}

	if doSpeed {
		speedTest(r, ctx, wm, files)
	}
}

// speedTest measures single-stream download throughput with turbo off and
// on, using the largest file on the card (capped at 64MB per run).
// The number that settles whether turbo helps this host and camera.
func speedTest(r *report, ctx context.Context, wm *wifi.WiFiManager, files []wifi.MediaFile) {
	var largest wifi.MediaFile
	for _, f := range files {
		if f.Size > largest.Size {
			largest = f
		}
	}
	if largest.Size < 8<<20 {
		r.note("speed test", "no file of at least 8MB on the card; record a clip first")
		return
	}

	const capBytes = 64 << 20
	measure := func() (float64, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", largest.URL, nil)
		if err != nil {
			return 0, err
		}
		limit := largest.Size
		if limit > capBytes {
			limit = capBytes
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", limit-1))
		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		n, err := io.Copy(io.Discard, resp.Body)
		if err != nil {
			return 0, err
		}
		return float64(n) / 1e6 / time.Since(start).Seconds(), nil
	}

	run := func(label string, turbo bool) {
		if err := wm.SetTurboTransfer(ctx, turbo); err != nil {
			r.note("speed "+label, "turbo toggle failed: "+err.Error())
			return
		}
		time.Sleep(2 * time.Second) // let the camera settle into the mode
		rate, err := measure()
		if err != nil {
			r.bad("speed "+label, err.Error())
			return
		}
		r.ok("speed "+label, fmt.Sprintf("%.1f MB/s (%s, first %dMB)", rate, largest.Name, min(largest.Size, capBytes)>>20))
	}

	run("turbo off", false)
	run("turbo ON", true)
	wm.SetTurboTransfer(ctx, false)
	r.note("speed verdict", "set turbo_enabled in the app config to whichever won")
}

func runPair(adapter *bluetooth.Adapter, fragment string) int {
	r := &report{}
	fmt.Printf("=== dropz pairing validation ===\n")
	fmt.Printf("Put the camera in pairing mode first: Connections > Connect Device > Quick App.\n\n")

	result, adv, err := findCamera(adapter, fragment, 60*time.Second)
	if err != nil {
		r.bad("discovery", err.Error())
		return r.summary()
	}
	addr := result.Address.String()
	r.ok("discovery", fmt.Sprintf("%s (%s)", result.LocalName(), addr))
	dumpRaw(result)
	r.note("advertisement", flagString(adv))

	if adv.PairingMode {
		r.ok("pairing flag (bit 2)", "camera advertises pairing mode: bit-order fix proven live")
	} else {
		r.bad("pairing flag (bit 2)", "camera is on the pairing screen but bit 2 is unset; bit parsing is wrong (or camera is not in pairing mode)")
	}

	manager := ble.NewManager(adapter, probeLog)
	defer manager.Stop()

	start := time.Now()
	err = manager.ConnectForPairing(addr)
	r.check("pairing connect", err, fmt.Sprintf("bond + setup in %v", time.Since(start).Round(time.Millisecond)))
	if err != nil {
		return r.summary()
	}
	defer manager.DisconnectQuietly(addr)

	ssid, password, err := manager.GetWifiCredentials(addr)
	if err != nil || ssid == "" || password == "" {
		r.bad("credentials after bond", fmt.Sprintf("ssid=%q err=%v", ssid, err))
	} else {
		r.ok("credentials after bond", "ssid "+ssid)
	}

	// RequestPairingFinish is fire-and-forget; only the camera UI shows it.
	time.Sleep(3 * time.Second)
	r.human("pairing screen", "the camera's pairing screen must have closed by itself: that proves the RequestPairingFinish framing fix")

	return r.summary()
}

// runSettings validates the settings read path exactly as the app runs it:
// bare 0x12 for all values, then one 0x32 per known setting (the official
// SDK's shape). A bare all-settings 0x32 is attempted last as a data point;
// no reference implementation sends it and a HERO11 never finished answering.
func runSettings(adapter *bluetooth.Adapter, fragment string) int {
	r := &report{}
	fmt.Printf("=== dropz settings validation ===\n\n")

	result, _, err := findCamera(adapter, fragment, 30*time.Second)
	if err != nil {
		r.bad("discovery", err.Error())
		return r.summary()
	}
	addr := result.Address.String()
	r.ok("discovery", fmt.Sprintf("%s (%s)", result.LocalName(), addr))

	manager := ble.NewManager(adapter, probeLog)
	defer manager.Stop()

	if err := manager.ConnectForStatusCheck(addr); err != nil {
		r.bad("connect", err.Error())
		return r.summary()
	}
	defer manager.DisconnectQuietly(addr)
	r.ok("connect", "lightweight status-check connection")

	start := time.Now()
	values, err := manager.GetSettingValues(addr, nil)
	if err != nil {
		r.bad("values (bare 0x12)", err.Error())
		return r.summary()
	}
	r.ok("values (bare 0x12)", fmt.Sprintf("%d settings in %v", len(values), time.Since(start).Round(time.Millisecond)))

	var ids []byte
	for id := range values {
		if _, known := ble.SettingDefs[id]; known {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	start = time.Now()
	totalOptions := 0
	var failures []string
	for _, id := range ids {
		caps, capErr := manager.GetSettingCapabilities(addr, []byte{id})
		if capErr != nil {
			failures = append(failures, fmt.Sprintf("%d: %v", id, capErr))
			continue
		}
		totalOptions += len(caps[id])
	}
	detail := fmt.Sprintf("%d settings, %d options total in %v", len(ids)-len(failures), totalOptions, time.Since(start).Round(time.Millisecond))
	if len(failures) > 0 {
		r.bad("capabilities (per setting)", detail+"; failed: "+strings.Join(failures, "; "))
	} else {
		r.ok("capabilities (per setting)", detail)
	}

	// Data point for the conformance record, not part of the app flow.
	start = time.Now()
	if bare, err := manager.GetSettingCapabilities(addr, nil); err != nil {
		r.note("bare 0x32 (unused by app)", fmt.Sprintf("no usable answer: %v", err))
	} else {
		r.note("bare 0x32 (unused by app)", fmt.Sprintf("answered: %d settings in %v", len(bare), time.Since(start).Round(time.Millisecond)))
	}

	return r.summary()
}
