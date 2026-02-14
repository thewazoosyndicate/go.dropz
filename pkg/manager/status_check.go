package manager

import (
	"encoding/binary"
	"time"

	"github.com/dropz/dropz/pkg/ble"
	"github.com/dropz/dropz/pkg/database"
	syncpkg "github.com/dropz/dropz/pkg/manager/sync"
)

// checkCameraStatuses queries BLE statuses for all managed cameras.
func (m *GoProManager) checkCameraStatuses() {
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
		m.checkSingleCameraStatusByID(cs.Camera.ID)
	}
}

// checkSingleCameraStatusByID runs a BLE status check for one camera, looked up fresh by ID.
func (m *GoProManager) checkSingleCameraStatusByID(cameraID string) {
	cs, ok := m.db.GetCameraByID(cameraID)
	if !ok {
		m.log.Warnf("Status check: camera %s not found", cameraID)
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

	if err := m.ble.ConnectForStatusCheck(bleAddress); err != nil {
		m.log.Debugf("Status check connect failed for %s: %v", cs.Camera.Name, err)
		return
	}

	statuses, err := m.ble.QueryStatuses(bleAddress, []byte{
		ble.StatusBatteryPercentage,
		ble.StatusNumTotalPhotos,
		ble.StatusNumTotalVideos,
		ble.StatusSDCardStatus,
		ble.StatusSDCardRemainingKB,
	})
	if err != nil {
		m.log.Debugf("Status check query failed for %s: %v", cs.Camera.Name, err)
		m.ble.Disconnect(bleAddress)
		return
	}

	syncQueued = m.processStatusResults(cs, statuses)

	// HERO13+/MAX2 (Broadcom BCM4381, model ID >= 64) ignore Sleep on the
	// first BLE connection from idle. Reconnect so Sleep works on the second.
	// ModelID 0 = unknown (not yet set by full connect) — assume new gen.
	modelID := cs.Metadata.ModelID
	needsReconnectToSleep := !syncQueued && (modelID == 0 || modelID >= 64)

	if needsReconnectToSleep {
		m.ble.DisconnectQuietly(bleAddress)
		if err := m.ble.ConnectForStatusCheck(bleAddress); err != nil {
			m.log.Debugf("Sleep reconnect failed for %s: %v", cs.Camera.Name, err)
			return
		}
	}

	if syncQueued {
		m.ble.DisconnectQuietly(bleAddress)
	} else {
		m.ble.Disconnect(bleAddress)
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
func (m *GoProManager) processStatusResults(cs *database.CameraWithState, statuses map[byte][]byte) bool {
	oldPhotos := cs.Metadata.NumPhotos
	oldVideos := cs.Metadata.NumVideos
	oldRemainingKB := cs.Metadata.RemainingSpaceKB

	var newBattery int32
	var newPhotos, newVideos, newSDStatus int32
	var newRemainingKB int64

	if v, ok := statuses[ble.StatusBatteryPercentage]; ok && len(v) >= 1 {
		newBattery = int32(v[0])
	}
	if v, ok := statuses[ble.StatusNumTotalPhotos]; ok {
		newPhotos = parseIntStatus(v)
	}
	if v, ok := statuses[ble.StatusNumTotalVideos]; ok {
		newVideos = parseIntStatus(v)
	}
	if v, ok := statuses[ble.StatusSDCardStatus]; ok && len(v) >= 1 {
		newSDStatus = int32(v[0])
	}
	if v, ok := statuses[ble.StatusSDCardRemainingKB]; ok {
		newRemainingKB = parseInt64Status(v)
	}

	m.db.UpdateCameraByID(cs.Camera.ID, func(cs *database.CameraWithState) {
		if newBattery > 0 {
			cs.Metadata.BatteryLevel = newBattery
		}
		if newPhotos >= 0 {
			cs.Metadata.NumPhotos = newPhotos
		}
		if newVideos >= 0 {
			cs.Metadata.NumVideos = newVideos
		}
		if newSDStatus != 255 {
			cs.Metadata.SDCardStatusCode = newSDStatus
		}
		if newRemainingKB > 0 {
			cs.Metadata.RemainingSpaceKB = newRemainingKB
		}
	})

	m.log.Debugf("Status check %s: battery=%d%% photos=%d videos=%d sd=%d remaining=%dKB",
		cs.Camera.Name, newBattery, newPhotos, newVideos, newSDStatus, newRemainingKB)

	// Detect new media via count increase OR SD card space decrease (>10MB).
	// Space decrease catches quick-capture footage where counts aren't updated.
	countIncreased := (oldPhotos > 0 && newPhotos > oldPhotos) || (oldVideos > 0 && newVideos > oldVideos)
	spaceDecreased := oldRemainingKB > 0 && newRemainingKB > 0 && (oldRemainingKB-newRemainingKB) > 10*1024
	hasNewMedia := countIncreased || spaceDecreased
	if !hasNewMedia {
		m.notify()
		return false
	}

	m.log.Infof("New media detected on %s (photos: %d→%d, videos: %d→%d), queuing sync",
		cs.Camera.Name, oldPhotos, newPhotos, oldVideos, newVideos)

	m.db.AddSyncQueueEntry(&database.SyncQueueEntry{
		CameraID:         cs.Camera.ID,
		QueuedAt:         time.Now(),
		Priority:         syncpkg.SyncPriorityAuto,
		CurrentOperation: "Waiting to start",
	})
	m.notify()

	return true
}

func parseIntStatus(v []byte) int32 {
	switch len(v) {
	case 1:
		return int32(v[0])
	case 2:
		return int32(binary.BigEndian.Uint16(v))
	case 4:
		return int32(binary.BigEndian.Uint32(v))
	default:
		return 0
	}
}

func parseInt64Status(v []byte) int64 {
	switch len(v) {
	case 1:
		return int64(v[0])
	case 2:
		return int64(binary.BigEndian.Uint16(v))
	case 4:
		return int64(binary.BigEndian.Uint32(v))
	case 8:
		return int64(binary.BigEndian.Uint64(v))
	default:
		return 0
	}
}
