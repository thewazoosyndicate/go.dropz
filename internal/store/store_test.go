package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/model"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dropz.db")
	st, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return st, path
}

func addCamera(t *testing.T, st *Store, id, bleAddr string) {
	t.Helper()
	err := st.AddOrUpdateDiscoveredCamera(&model.DiscoveredCamera{
		CameraState: &model.CameraWithState{
			Camera: model.Camera{ID: id, Name: "GoPro Test", BLEAddress: bleAddr},
			Status: model.CameraStatus{LastSeen: time.Now(), IsReachable: true},
		},
	})
	if err != nil {
		t.Fatalf("AddOrUpdateDiscoveredCamera: %v", err)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	st, path := newTestStore(t)
	addCamera(t, st, "cam1", "AA:BB")

	st.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Camera.WiFiSSID = "GP123"
		cs.Status.IsPaired = true
		cs.Status.IsManaged = true
	})

	reloaded, err := New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	cs, ok := reloaded.GetCameraByID("cam1")
	if !ok {
		t.Fatal("camera lost on reload")
	}
	if cs.Camera.WiFiSSID != "GP123" || !cs.Status.IsPaired {
		t.Errorf("durable fields lost: %+v", cs)
	}
}

func TestEphemeralUpdateSkipsFileWrite(t *testing.T) {
	st, path := newTestStore(t)
	addCamera(t, st, "cam1", "AA:BB")

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	err = st.UpdateCamera("AA:BB", func(cs *model.CameraWithState) bool {
		cs.Camera.RSSI = -42
		cs.Status.LastSeen = time.Now()
		return false // ephemeral
	})
	if err != nil {
		t.Fatalf("UpdateCamera: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("ephemeral update rewrote the file")
	}

	// In-memory state must still reflect the change
	cs, _ := st.GetCameraByID("cam1")
	if cs.Camera.RSSI != -42 {
		t.Errorf("in-memory RSSI = %d, want -42", cs.Camera.RSSI)
	}
}

func TestGroupIDDerivedFromGroups(t *testing.T) {
	st, _ := newTestStore(t)
	addCamera(t, st, "cam1", "AA:BB")
	addCamera(t, st, "cam2", "CC:DD")

	st.AddOrUpdateGroup(&model.Group{ID: "g1", Name: "trip", CameraIDs: []string{"cam1"}})

	cs, _ := st.GetCameraByID("cam1")
	if cs.GroupID != "g1" {
		t.Errorf("cam1 GroupID = %q, want g1", cs.GroupID)
	}
	cs2, _ := st.GetCameraByID("cam2")
	if cs2.GroupID != "" {
		t.Errorf("cam2 GroupID = %q, want empty", cs2.GroupID)
	}

	// Removing the group clears derived membership with no camera writes
	st.RemoveGroup("g1")
	cs, _ = st.GetCameraByID("cam1")
	if cs.GroupID != "" {
		t.Errorf("cam1 GroupID after delete = %q, want empty", cs.GroupID)
	}
}

func TestGetGroupReturnsCopy(t *testing.T) {
	st, _ := newTestStore(t)
	st.AddOrUpdateGroup(&model.Group{ID: "g1", Name: "trip", CameraIDs: []string{"a"}})

	g, _ := st.GetGroup("g1")
	g.Name = "mutated"
	g.CameraIDs[0] = "b"

	g2, _ := st.GetGroup("g1")
	if g2.Name != "trip" || g2.CameraIDs[0] != "a" {
		t.Errorf("stored group mutated through returned pointer: %+v", g2)
	}
}

func TestSyncQueuePriorityOrder(t *testing.T) {
	st, _ := newTestStore(t)
	old := time.Now().Add(-time.Hour)
	st.AddSyncQueueEntry(&model.SyncQueueEntry{CameraID: "auto", Priority: 5, QueuedAt: time.Now()})
	st.AddSyncQueueEntry(&model.SyncQueueEntry{CameraID: "manual", Priority: 10, QueuedAt: time.Now()})
	st.AddSyncQueueEntry(&model.SyncQueueEntry{CameraID: "auto-old", Priority: 5, QueuedAt: old})

	q := st.GetSyncQueue()
	if len(q) != 3 || q[0].CameraID != "manual" || q[1].CameraID != "auto-old" {
		ids := make([]string, len(q))
		for i, e := range q {
			ids[i] = e.CameraID
		}
		t.Errorf("queue order = %v, want [manual auto-old auto]", ids)
	}
}

func TestResetTransientStates(t *testing.T) {
	st, _ := newTestStore(t)
	addCamera(t, st, "cam1", "AA:BB")
	st.UpdateCameraByID("cam1", func(cs *model.CameraWithState) {
		cs.Status.IsSyncing = true
		cs.Status.IsPairing = true
	})

	if err := st.ResetTransientStates(); err != nil {
		t.Fatal(err)
	}
	cs, _ := st.GetCameraByID("cam1")
	if cs.Status.IsSyncing || cs.Status.IsPairing || cs.Status.IsReachable {
		t.Errorf("transient states not reset: %+v", cs.Status)
	}
}
