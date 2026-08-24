package syncer

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/store"
)

func newTestCoordinator(t *testing.T) (*Coordinator, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "dropz.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := NewCoordinator(context.Background(), st, nil, log, func() {}, nil, nil)
	return c, st
}

func seedCamera(t *testing.T, st *store.Store, id string, status model.CameraStatus) {
	t.Helper()
	err := st.AddOrUpdateDiscoveredCamera(&model.DiscoveredCamera{
		CameraState: &model.CameraWithState{
			Camera: model.Camera{ID: id, Name: "GoPro " + id, BLEAddress: "AA:" + id},
			Status: status,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func entry(id string) *model.SyncQueueEntry {
	return &model.SyncQueueEntry{CameraID: id, QueuedAt: time.Now(), Priority: SyncPriorityAuto}
}

func TestTryClaimCameraSuccess(t *testing.T) {
	c, st := newTestCoordinator(t)
	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: true, IsPaired: true, IsReachable: true})
	st.AddSyncQueueEntry(entry("cam1"))

	task, ok := c.tryClaimCamera(entry("cam1"))
	if !ok || task == nil {
		t.Fatal("expected claim to succeed")
	}
	cs, _ := st.GetCameraByID("cam1")
	if !cs.Status.IsSyncing {
		t.Error("claim must mark the camera as syncing")
	}

	// Second claim of the same camera must fail while the task is active
	if _, ok := c.tryClaimCamera(entry("cam1")); ok {
		t.Error("double claim succeeded")
	}
}

func TestTryClaimCameraDropsAutoSyncOfPausedGroup(t *testing.T) {
	c, st := newTestCoordinator(t)
	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: true, IsPaired: true, IsReachable: true})
	st.AddOrUpdateGroup(&model.Group{ID: "g1", CameraIDs: []string{"cam1"}, SyncPaused: true})

	st.AddSyncQueueEntry(entry("cam1"))
	if _, ok := c.tryClaimCamera(entry("cam1")); ok {
		t.Fatal("auto sync claimed for a paused group")
	}
	if len(st.GetSyncQueue()) != 0 {
		t.Error("paused auto entry should leave the queue")
	}

	manual := entry("cam1")
	manual.Priority = SyncPriorityManual
	st.AddSyncQueueEntry(manual)
	if _, ok := c.tryClaimCamera(manual); !ok {
		t.Error("a manual sync must still run for a paused group")
	}
}

func TestTryClaimCameraRemovesUnknownFromQueue(t *testing.T) {
	c, st := newTestCoordinator(t)
	st.AddSyncQueueEntry(entry("ghost"))

	if _, ok := c.tryClaimCamera(entry("ghost")); ok {
		t.Fatal("claimed a camera that does not exist")
	}
	if len(st.GetSyncQueue()) != 0 {
		t.Error("unknown camera should be removed from the queue")
	}
}

func TestTryClaimCameraRequiresManagedPaired(t *testing.T) {
	c, st := newTestCoordinator(t)
	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: true, IsPaired: false, IsReachable: true})
	st.AddSyncQueueEntry(entry("cam1"))

	if _, ok := c.tryClaimCamera(entry("cam1")); ok {
		t.Fatal("claimed an unpaired camera")
	}
	if len(st.GetSyncQueue()) != 0 {
		t.Error("ineligible camera should be removed from the queue")
	}
}

func TestTryClaimCameraSkipsUnreachable(t *testing.T) {
	c, st := newTestCoordinator(t)
	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: true, IsPaired: true, IsReachable: false})
	st.AddSyncQueueEntry(entry("cam1"))

	if _, ok := c.tryClaimCamera(entry("cam1")); ok {
		t.Fatal("claimed an unreachable camera")
	}
	// Stays queued: it syncs when it comes back
	if len(st.GetSyncQueue()) != 1 {
		t.Error("unreachable camera must stay in the queue")
	}
}

func TestForceSyncRequiresManagedPaired(t *testing.T) {
	c, st := newTestCoordinator(t)
	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: false, IsPaired: false})

	if _, err := c.ForceSync("cam1"); err == nil {
		t.Fatal("ForceSync must refuse an unmanaged camera")
	}
	if _, err := c.ForceSync("nope"); err == nil {
		t.Fatal("ForceSync must refuse an unknown camera")
	}
}

func TestCancelSyncClearsQueueAndState(t *testing.T) {
	c, st := newTestCoordinator(t)
	seedCamera(t, st, "cam1", model.CameraStatus{IsManaged: true, IsPaired: true, IsReachable: true})
	st.AddSyncQueueEntry(entry("cam1"))
	if _, ok := c.tryClaimCamera(entry("cam1")); !ok {
		t.Fatal("setup claim failed")
	}

	if err := c.CancelSync("cam1"); err != nil {
		t.Fatal(err)
	}
	if len(st.GetSyncQueue()) != 0 {
		t.Error("queue entry not removed")
	}
	cs, _ := st.GetCameraByID("cam1")
	if cs.Status.IsSyncing {
		t.Error("IsSyncing not cleared")
	}
}
