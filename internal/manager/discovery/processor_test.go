package discovery

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/store"
)

func newTestProcessor(t *testing.T) (*Processor, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "dropz.db"))
	if err != nil {
		t.Fatal(err)
	}
	return NewProcessor(st, slog.New(slog.NewTextHandler(io.Discard, nil))), st
}

func TestSerialIdentitySurvivesAddressChange(t *testing.T) {
	p, st := newTestProcessor(t)

	// First sighting: macOS-style random UUID
	p.processDevice(ble.Device{
		Name: "GoPro 0711", BLEAddress: "uuid-one", RSSI: -50,
		SerialNumber: "C3501324500711",
	})
	// Bluetooth restarted: same camera, new random UUID
	p.processDevice(ble.Device{
		Name: "GoPro 0711", BLEAddress: "uuid-two", RSSI: -48,
		SerialNumber: "C3501324500711",
	})

	cams := st.GetAllCameras()
	if len(cams) != 1 {
		t.Fatalf("got %d cameras, want 1 (ghost entry created)", len(cams))
	}
	if cams[0].Camera.BLEAddress != "uuid-two" {
		t.Errorf("address not updated: %s", cams[0].Camera.BLEAddress)
	}
	if cams[0].Metadata.SerialNumber != "C3501324500711" {
		t.Errorf("serial lost: %s", cams[0].Metadata.SerialNumber)
	}
}

func TestNameFallbackWithoutSerial(t *testing.T) {
	p, st := newTestProcessor(t)

	// Older camera: no serial in advertisement
	p.processDevice(ble.Device{Name: "GoPro 1234", BLEAddress: "uuid-one", RSSI: -50})
	p.processDevice(ble.Device{Name: "GoPro 1234", BLEAddress: "uuid-two", RSSI: -48})

	if got := len(st.GetAllCameras()); got != 1 {
		t.Fatalf("got %d cameras, want 1", got)
	}
}

func TestPairingModeTransitionFiresOnce(t *testing.T) {
	p, st := newTestProcessor(t)

	var fired int
	p.SetOnPairingModeDetected(func(cameraID string) { fired++ })

	dev := ble.Device{Name: "GoPro 0711", BLEAddress: "aa", SerialNumber: "C3501324500711"}
	p.processDevice(dev) // create, not pairing

	dev.PairingMode = true
	p.processDevice(dev) // rising edge
	p.processDevice(dev) // still pairing, no re-fire
	dev.PairingMode = false
	p.processDevice(dev) // dropped
	dev.PairingMode = true
	p.processDevice(dev) // second rising edge

	if fired != 2 {
		t.Errorf("pairing callback fired %d times, want 2", fired)
	}

	cs, _ := st.GetCameraByID(st.GetAllCameras()[0].Camera.ID)
	if !cs.Status.InPairingMode {
		t.Error("InPairingMode not tracked")
	}
}

func TestNewMediaRisingEdge(t *testing.T) {
	p, _ := newTestProcessor(t)

	var fired int
	p.SetOnNewMediaAdvertised(func(cameraID string) { fired++ })

	dev := ble.Device{Name: "GoPro 0711", BLEAddress: "aa", SerialNumber: "C3501324500711"}
	p.processDevice(dev)
	dev.NewMedia = true
	p.processDevice(dev)
	p.processDevice(dev)
	dev.NewMedia = false
	p.processDevice(dev)
	dev.NewMedia = true
	p.processDevice(dev)

	if fired != 2 {
		t.Errorf("new media callback fired %d times, want 2", fired)
	}
}

func TestModelIDLearnedFromAdvertisement(t *testing.T) {
	p, st := newTestProcessor(t)

	p.processDevice(ble.Device{Name: "GoPro 0711", BLEAddress: "aa"})
	p.processDevice(ble.Device{Name: "GoPro 0711", BLEAddress: "aa", ModelID: 64})

	cams := st.GetAllCameras()
	if cams[0].Metadata.ModelID != 64 {
		t.Errorf("ModelID = %d, want 64", cams[0].Metadata.ModelID)
	}
}
