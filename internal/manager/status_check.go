package manager

import (
	"time"

	"github.com/dropz/dropz/internal/ble"
	"github.com/dropz/dropz/internal/manager/syncer"
	"github.com/dropz/dropz/internal/model"
)

// enqueueStatusChecks queues BLE status checks for all eligible managed cameras.
// The statusCheckWorker drains the queue; a full channel drops the request,
// which is fine because the next tick re-enqueues.
func (m *GoProManager) enqueueStatusChecks() {
	config := m.db.GetConfig()
	if !config.SyncEnabled {
		return
	}

	managedCameras := m.db.GetCamerasForManagedPool()
	for _, cam := range managedCameras {
		cs := cam.CameraState
		if !cs.Status.IsReachable || cs.Status.IsSyncing || cs.Status.IsPairing {
			continue
		}
		if m.ble.IsConnected(cs.Camera.BLEAddress) {
			continue
		}
		select {
		case m.statusCheckRequest <- cs.Camera.ID:
		default:
			m.log.Debug("Status check request dropped, queue full", cs.LogAttrs()...)
		}
	}
}

// checkSingleCameraStatusByID runs a BLE status check for one camera, looked up fresh by ID.
func (m *GoProManager) checkSingleCameraStatusByID(cameraID string) {
	cs, ok := m.db.GetCameraByID(cameraID)
	if !ok {
		m.log.Warn("Camera not found for status check", "camera_id", cameraID)
		return
	}
	if !cs.Status.IsReachable || cs.Status.IsSyncing || cs.Status.IsPairing {
		return
	}
	if m.ble.IsConnected(cs.Camera.BLEAddress) {
		return
	}

	bleAddress := cs.Camera.BLEAddress
	var syncQueued bool

	// Another session (settings, pairing, sync) owns the camera: skip;
	// the next tick re-checks.
	release, ok := m.ble.TryAcquireSession(bleAddress)
	if !ok {
		return
	}
	defer release()

	if err := m.ble.ConnectForStatusCheck(bleAddress); err != nil {
		m.log.Debug("Status check connect failed", append(cs.LogAttrs(), "err", err)...)
		return
	}

	statuses, err := m.ble.QueryStatuses(bleAddress, []byte{
		ble.StatusBatteryPercentage,
		ble.StatusNumTotalPhotos,
		ble.StatusNumTotalVideos,
		ble.StatusSDCardStatus,
		ble.StatusSDCardRemainingKB,
		ble.StatusSystemBusy,
		ble.StatusEncoding,
	})
	if err != nil {
		m.log.Debug("Status check query failed", append(cs.LogAttrs(), "err", err)...)
		_ = m.ble.Disconnect(bleAddress)
		return
	}

	syncQueued = m.processStatusResults(cs, statuses)

	// A sync already waiting in the queue must find the camera awake:
	// Sleep here and the sync's connect races the camera's shutdown
	// (observed on HERO13: GATT discovery fails against a camera going
	// to sleep, and the camera BLE stack stays wedged afterwards).
	for _, entry := range m.db.GetSyncQueue() {
		if entry.CameraID == cs.Camera.ID {
			syncQueued = true
			break
		}
	}

	// A busy or recording camera must not be put to sleep.
	if ble.InUse(statuses) {
		// Fires on every poll while recording, hence Debug
		m.log.Debug("Camera busy or recording, leaving it awake", cs.LogAttrs()...)
		_ = m.ble.DisconnectQuietly(bleAddress)
		return
	}

	// HERO13+/MAX2 (Broadcom BCM4381, model ID >= 64) ignore Sleep on the
	// first BLE connection from idle. Reconnect so Sleep works on the second.
	// ModelID 0 = unknown (not yet set by full connect) — assume new gen.
	modelID := cs.Metadata.ModelID
	needsReconnectToSleep := !syncQueued && (modelID == 0 || modelID >= 64)

	if needsReconnectToSleep {
		_ = m.ble.DisconnectQuietly(bleAddress)
		if err := m.ble.ConnectForStatusCheck(bleAddress); err != nil {
			// Camera stays awake; next status check will retry the sleep
			m.log.Debug("Sleep reconnect failed, camera left awake", append(cs.LogAttrs(), "err", err)...)
			return
		}
	}

	if syncQueued {
		_ = m.ble.DisconnectQuietly(bleAddress)
	} else {
		_ = m.ble.Disconnect(bleAddress)
	}

	// Trigger sync AFTER BLE disconnect so the sync goroutine
	// sees the device as disconnected and does a full Connect.
	if syncQueued {
		select {
		case m.immediateSyncTrigger <- struct{}{}:
		default:
		}
	}
}

// processStatusResults compares old vs new counts and queues sync if new media detected.
func (m *GoProManager) processStatusResults(cs *model.CameraWithState, statuses map[byte][]byte) bool {
	oldPhotos := cs.Metadata.NumPhotos
	oldVideos := cs.Metadata.NumVideos
	oldRemainingKB := cs.Metadata.RemainingSpaceKB

	// A response can omit any status ID; a missing one must not overwrite
	// the stored value with zero, or new-media detection (which requires a
	// nonzero old count) is silently disabled from then on.
	var newBattery int32
	newPhotos, newVideos := int32(-1), int32(-1)
	var newSDStatus int32 = 255
	var newRemainingKB int64 = -1

	if v, ok := statuses[ble.StatusBatteryPercentage]; ok && len(v) >= 1 {
		newBattery = int32(v[0])
	}
	if v, ok := statuses[ble.StatusNumTotalPhotos]; ok {
		newPhotos = ble.ParseIntStatus(v)
	}
	if v, ok := statuses[ble.StatusNumTotalVideos]; ok {
		newVideos = ble.ParseIntStatus(v)
	}
	if v, ok := statuses[ble.StatusSDCardStatus]; ok && len(v) >= 1 {
		newSDStatus = int32(v[0])
	}
	if v, ok := statuses[ble.StatusSDCardRemainingKB]; ok {
		newRemainingKB = ble.ParseInt64Status(v)
	}

	_ = m.db.UpdateCameraByID(cs.Camera.ID, func(cs *model.CameraWithState) {
		if newBattery > 0 {
			cs.Metadata.BatteryLevel = newBattery
		}
		// HERO13+ returns 0xFFFFFFFF (-1 as int32) and 0xFF (255) as sentinel
		// values after sleep/wake when the camera can't read the SD card yet.
		// Only overwrite stored values when the camera returns valid data.
		if newPhotos >= 0 {
			cs.Metadata.NumPhotos = newPhotos
		}
		if newVideos >= 0 {
			cs.Metadata.NumVideos = newVideos
		}
		if newSDStatus != 255 {
			cs.Metadata.SDCardStatusCode = newSDStatus
		}
		if newRemainingKB > 0 || (newRemainingKB == 0 && newSDStatus == 0) {
			cs.Metadata.RemainingSpaceKB = newRemainingKB
		}
	})

	m.log.Debug("Status check result", append(cs.LogAttrs(),
		"battery", newBattery, "photos", newPhotos, "videos", newVideos, "sd", newSDStatus, "remaining_kb", newRemainingKB)...)

	// Detect new media via count increase OR SD card space decrease (>10MB).
	// Space decrease catches quick-capture footage where counts aren't updated.
	countIncreased := (oldPhotos > 0 && newPhotos > oldPhotos) || (oldVideos > 0 && newVideos > oldVideos)
	spaceDecreased := oldRemainingKB > 0 && newRemainingKB > 0 && (oldRemainingKB-newRemainingKB) > 10*1024
	hasNewMedia := countIncreased || spaceDecreased
	if !hasNewMedia {
		m.notify()
		return false
	}

	m.log.Info("New media detected, queuing sync", append(cs.LogAttrs(),
		"photos_old", oldPhotos, "photos_new", newPhotos, "videos_old", oldVideos, "videos_new", newVideos)...)

	_ = m.db.AddSyncQueueEntry(&model.SyncQueueEntry{
		CameraID:         cs.Camera.ID,
		QueuedAt:         time.Now(),
		Priority:         syncer.SyncPriorityAuto,
		CurrentOperation: "Waiting to start",
	})
	m.notify()

	return true
}
