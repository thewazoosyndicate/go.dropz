package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/model"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dropz.db")
	st, err := New(path, nil)
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

	reloaded, err := New(path, nil)
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

func TestGroupMembershipIsExclusive(t *testing.T) {
	st, _ := newTestStore(t)
	addCamera(t, st, "cam1", "AA:01")
	addCamera(t, st, "cam2", "AA:02")
	st.AddOrUpdateGroup(&model.Group{ID: "g1", Name: "rig A", CameraIDs: []string{"cam1", "cam2"}})
	st.AddOrUpdateGroup(&model.Group{ID: "g2", Name: "rig B", CameraIDs: []string{"cam2"}})

	g1, _ := st.GetGroup("g1")
	if len(g1.CameraIDs) != 1 || g1.CameraIDs[0] != "cam1" {
		t.Errorf("cam2 still in g1: %v", g1.CameraIDs)
	}

	if err := st.MoveCamerasToGroup([]string{"cam1"}, "g2"); err != nil {
		t.Fatal(err)
	}
	g1, _ = st.GetGroup("g1")
	g2, _ := st.GetGroup("g2")
	if len(g1.CameraIDs) != 0 || len(g2.CameraIDs) != 2 {
		t.Errorf("after move g1=%v g2=%v", g1.CameraIDs, g2.CameraIDs)
	}
	if cs, _ := st.GetCameraByID("cam1"); cs.GroupID != "g2" {
		t.Errorf("cam1 GroupID = %q, want g2", cs.GroupID)
	}

	if err := st.MoveCamerasToGroup([]string{"cam1"}, ""); err != nil {
		t.Fatal(err)
	}
	if cs, _ := st.GetCameraByID("cam1"); cs.GroupID != "" {
		t.Errorf("ungrouped cam1 GroupID = %q", cs.GroupID)
	}
	if err := st.MoveCamerasToGroup([]string{"cam1"}, "ghost"); err == nil {
		t.Error("move to unknown group accepted")
	}
}

func TestSetGroupSyncExclusive(t *testing.T) {
	st, _ := newTestStore(t)
	addCamera(t, st, "cam1", "AA:01")
	addCamera(t, st, "cam2", "AA:02")
	st.AddOrUpdateGroup(&model.Group{ID: "g1", CameraIDs: []string{"cam1"}})
	st.AddOrUpdateGroup(&model.Group{ID: "g2", CameraIDs: []string{"cam2"}})

	if err := st.SetGroupSync("g1", false, true); err != nil {
		t.Fatal(err)
	}
	if st.IsCameraSyncPaused("cam1") || !st.IsCameraSyncPaused("cam2") {
		t.Error("exclusive resume must pause the other group only")
	}
	if err := st.SetGroupSync("g1", true, false); err != nil {
		t.Fatal(err)
	}
	if !st.IsCameraSyncPaused("cam1") || !st.IsCameraSyncPaused("cam2") {
		t.Error("plain pause must leave the other group alone")
	}
	if err := st.SetGroupSync("ghost", true, false); err == nil {
		t.Error("unknown group accepted")
	}
	addCamera(t, st, "loose", "AA:03")
	if st.IsCameraSyncPaused("loose") {
		t.Error("ungrouped camera reads as paused")
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

func TestSyncHistoryNewestFirstAndCapped(t *testing.T) {
	st, path := newTestStore(t)
	for i := 0; i < maxSyncHistory+5; i++ {
		s := &model.SyncSession{
			ID: fmt.Sprintf("s%d", i), CameraID: "cam", Outcome: model.SyncOutcomeComplete,
			Files: []model.SyncFile{{Name: "GX.MP4", State: model.SyncFileDone}},
		}
		if err := st.AddSyncSession(s); err != nil {
			t.Fatal(err)
		}
	}
	got := st.GetSyncHistory(0, "")
	if len(got) != maxSyncHistory {
		t.Fatalf("kept %d sessions, want %d", len(got), maxSyncHistory)
	}
	if got[0].ID != fmt.Sprintf("s%d", maxSyncHistory+4) {
		t.Errorf("first = %s, want the newest", got[0].ID)
	}
	// Copies: mutating a result must not touch the store
	got[0].Files[0].State = model.SyncFileFailed
	if st.GetSyncHistory(1, "")[0].Files[0].State != model.SyncFileDone {
		t.Error("GetSyncHistory shared its file slice")
	}
	if n := len(st.GetSyncHistory(3, "")); n != 3 {
		t.Errorf("limit 3 returned %d", n)
	}
	if n := len(st.GetSyncHistory(0, "other")); n != 0 {
		t.Errorf("camera filter returned %d", n)
	}

	reloaded, err := New(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.GetSyncHistory(0, "")) != maxSyncHistory {
		t.Error("history not persisted")
	}
}

func TestSetCameraAlias(t *testing.T) {
	st, _ := newTestStore(t)
	addCamera(t, st, "cam1", "AA:01")
	if err := st.SetCameraAliasByID("cam1", "Helmet cam"); err != nil {
		t.Fatal(err)
	}
	cs, _ := st.GetCameraByID("cam1")
	if cs.Camera.Alias != "Helmet cam" {
		t.Errorf("alias = %q", cs.Camera.Alias)
	}
	if err := st.SetCameraAliasByID("ghost", "x"); err == nil {
		t.Error("unknown camera accepted")
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
