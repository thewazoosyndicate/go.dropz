package manager

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/store"
)

func newTestManager(t *testing.T) *GoProManager {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "dropz.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &GoProManager{db: st, log: log}
}

func seedCamera(t *testing.T, m *GoProManager, photos, videos int32, remainingKB int64) *model.CameraWithState {
	t.Helper()
	err := m.db.AddOrUpdateDiscoveredCamera(&model.DiscoveredCamera{
		CameraState: &model.CameraWithState{
			Camera: model.Camera{ID: "cam1", Name: "GoPro Test", BLEAddress: "AA:BB"},
			Status: model.CameraStatus{LastSeen: time.Now(), IsReachable: true, IsPaired: true, IsManaged: true},
			Metadata: model.CameraMetadata{
				NumPhotos: photos, NumVideos: videos, RemainingSpaceKB: remainingKB,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	cs, _ := m.db.GetCameraByID("cam1")
	return cs
}

func be32(v uint32) []byte { return []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)} }

func TestParseIntStatus(t *testing.T) {
	cases := []struct {
		in   []byte
		want int32
	}{
		{[]byte{7}, 7},
		{[]byte{0x01, 0x02}, 258},
		{be32(70000), 70000},
		{be32(0xFFFFFFFF), -1}, // HERO13+ sentinel
		// Invalid lengths must parse as invalid (-1), not a fake zero
		// that would overwrite stored counts and disable detection.
		{nil, -1},
		{[]byte{1, 2, 3}, -1},
	}
	for _, c := range cases {
		if got := ble.ParseIntStatus(c.in); got != c.want {
			t.Errorf("ble.ParseIntStatus(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestProcessStatusResultsQueuesSyncOnNewMedia(t *testing.T) {
	m := newTestManager(t)
	cs := seedCamera(t, m, 10, 5, 1000*1024)

	queued := m.processStatusResults(cs, map[byte][]byte{
		ble.StatusNumTotalPhotos: be32(10), // unchanged
		ble.StatusNumTotalVideos: be32(6),  // one new video
	})
	if !queued {
		t.Fatal("expected sync to be queued on video count increase")
	}
	if len(m.db.GetSyncQueue()) != 1 {
		t.Fatal("sync queue entry missing")
	}
}

func TestProcessStatusResultsIgnoresSentinelCounts(t *testing.T) {
	m := newTestManager(t)
	cs := seedCamera(t, m, 10, 5, 1000*1024)

	// HERO13+ post-wake sentinels: counts -1, SD status 255
	queued := m.processStatusResults(cs, map[byte][]byte{
		ble.StatusNumTotalPhotos: be32(0xFFFFFFFF),
		ble.StatusNumTotalVideos: be32(0xFFFFFFFF),
		ble.StatusSDCardStatus:   {255},
	})
	if queued {
		t.Fatal("sentinel values must not queue a sync")
	}

	got, _ := m.db.GetCameraByID("cam1")
	if got.Metadata.NumPhotos != 10 || got.Metadata.NumVideos != 5 {
		t.Errorf("sentinel overwrote stored counts: %+v", got.Metadata)
	}
	if got.Metadata.SDCardStatusCode == 255 {
		t.Error("sentinel SD status stored")
	}
}

func TestProcessStatusResultsDetectsSpaceDecrease(t *testing.T) {
	m := newTestManager(t)
	cs := seedCamera(t, m, 0, 0, 1000*1024)

	// Counts stay zero but 20MB of space vanished (quick capture footage)
	queued := m.processStatusResults(cs, map[byte][]byte{
		ble.StatusSDCardRemainingKB: be32(1000*1024 - 20*1024),
	})
	if !queued {
		t.Fatal("expected sync on >10MB space decrease")
	}
}

func TestProcessStatusResultsNoChange(t *testing.T) {
	m := newTestManager(t)
	cs := seedCamera(t, m, 10, 5, 1000*1024)

	queued := m.processStatusResults(cs, map[byte][]byte{
		ble.StatusNumTotalPhotos:    be32(10),
		ble.StatusNumTotalVideos:    be32(5),
		ble.StatusSDCardRemainingKB: be32(1000 * 1024),
	})
	if queued {
		t.Fatal("no new media, nothing should be queued")
	}
	if len(m.db.GetSyncQueue()) != 0 {
		t.Fatal("sync queue should be empty")
	}
}
