package server

import (
	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/protocol"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Proto conversion lives here, next to the transport that needs it,
// so the domain model stays free of protocol concerns.

func toProtoCameraWithState(c *model.CameraWithState) *protocol.CameraWithState {
	return &protocol.CameraWithState{
		Camera: &protocol.Camera{
			Id:           c.Camera.ID,
			Name:         c.Camera.Name,
			Alias:        c.Camera.Alias,
			BleAddress:   c.Camera.BLEAddress,
			WifiSsid:     c.Camera.WiFiSSID,
			WifiPassword: c.Camera.WiFiPassword,
			Rssi:         c.Camera.RSSI,
		},
		Status: &protocol.CameraStatus{
			LastSeen:       timestamppb.New(c.Status.LastSeen),
			LastSynced:     timestamppb.New(c.Status.LastSynced),
			IsPairing:      c.Status.IsPairing,
			IsPaired:       c.Status.IsPaired,
			IsManaged:      c.Status.IsManaged,
			IsReachable:    c.Status.IsReachable,
			IsSynced:       c.Status.IsSynced,
			IsSyncing:      c.Status.IsSyncing,
			LastSyncError:  c.Status.LastSyncError,
			InPairingMode:  c.Status.InPairingMode,
			PreviewEnabled: c.Status.PreviewEnabled,
			NewMediaCount:  c.NewMediaCount(),
		},
		GroupId: c.GroupID,
		Metadata: &protocol.CameraMetadata{
			Id:               c.Metadata.ID,
			FirmwareVersion:  c.Metadata.FirmwareVersion,
			Model:            c.Metadata.Model,
			SerialNumber:     c.Metadata.SerialNumber,
			BatteryLevel:     c.Metadata.BatteryLevel,
			HardwareVersion:  c.Metadata.HardwareVersion,
			NumPhotos:        c.Metadata.NumPhotos,
			NumVideos:        c.Metadata.NumVideos,
			SdCardStatus:     c.Metadata.SDCardStatusCode,
			RemainingSpaceKb: c.Metadata.RemainingSpaceKB,
		},
	}
}

func toProtoDiscoveredCamera(c *model.DiscoveredCamera) *protocol.DiscoveredCamera {
	return &protocol.DiscoveredCamera{
		CameraState: toProtoCameraWithState(c.CameraState),
	}
}

func toProtoManagedCamera(c *model.ManagedCamera) *protocol.ManagedCamera {
	return &protocol.ManagedCamera{
		CameraState: toProtoCameraWithState(c.CameraState),
	}
}

func toProtoSyncQueueEntry(e *model.SyncQueueEntry) *protocol.SyncQueueEntry {
	out := &protocol.SyncQueueEntry{
		CameraId:         e.CameraID,
		QueuedAt:         timestamppb.New(e.QueuedAt),
		Priority:         e.Priority,
		ProgressPercent:  e.ProgressPercent,
		CurrentOperation: e.CurrentOperation,
		FileIndex:        e.FileIndex,
		FileCount:        e.FileCount,
		FileName:         e.FileName,
		FileBytes:        e.FileBytes,
		FileTotal:        e.FileTotal,
		BytesDone:        e.BytesDone,
		BytesTotal:       e.BytesTotal,
		RateBps:          e.RateBps,
		Phase:            toProtoSyncPhase(e.Phase),
		Phases:           toProtoPhaseTimings(e.Phases),
		Files:            toProtoSyncFiles(e.Files),
		StepIndex:        e.StepIndex,
		StepCount:        e.StepCount,
	}
	if !e.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(e.StartedAt)
	}
	return out
}

func toProtoSyncPhase(p model.SyncPhase) protocol.SyncPhase {
	switch p {
	case model.SyncPhaseConnect:
		return protocol.SyncPhase_SYNC_PHASE_CONNECT
	case model.SyncPhaseLink:
		return protocol.SyncPhase_SYNC_PHASE_LINK
	case model.SyncPhaseCatalog:
		return protocol.SyncPhase_SYNC_PHASE_CATALOG
	case model.SyncPhaseTransfer:
		return protocol.SyncPhase_SYNC_PHASE_TRANSFER
	default:
		return protocol.SyncPhase_SYNC_PHASE_WAITING
	}
}

func toProtoPhaseTimings(timings []model.SyncPhaseTiming) []*protocol.SyncPhaseTiming {
	out := make([]*protocol.SyncPhaseTiming, 0, len(timings))
	for _, t := range timings {
		pt := &protocol.SyncPhaseTiming{
			Phase:     toProtoSyncPhase(t.Phase),
			StartedAt: timestamppb.New(t.StartedAt),
		}
		if !t.FinishedAt.IsZero() {
			pt.FinishedAt = timestamppb.New(t.FinishedAt)
		}
		out = append(out, pt)
	}
	return out
}

func toProtoSyncFileState(s model.SyncFileState) protocol.SyncFileState {
	switch s {
	case model.SyncFileDownloading:
		return protocol.SyncFileState_SYNC_FILE_DOWNLOADING
	case model.SyncFileDone:
		return protocol.SyncFileState_SYNC_FILE_DONE
	case model.SyncFileFailed:
		return protocol.SyncFileState_SYNC_FILE_FAILED
	case model.SyncFileSkipped:
		return protocol.SyncFileState_SYNC_FILE_SKIPPED
	default:
		return protocol.SyncFileState_SYNC_FILE_QUEUED
	}
}

func toProtoSyncFiles(files []model.SyncFile) []*protocol.SyncFile {
	out := make([]*protocol.SyncFile, 0, len(files))
	for _, f := range files {
		out = append(out, &protocol.SyncFile{
			Name:       f.Name,
			CameraPath: f.CameraPath,
			SizeBytes:  f.SizeBytes,
			State:      toProtoSyncFileState(f.State),
			BytesDone:  f.BytesDone,
			Error:      f.Error,
			LocalPath:  f.LocalPath,
			DurationMs: f.DurationMs,
		})
	}
	return out
}

func toProtoSyncOutcome(o model.SyncOutcome) protocol.SyncOutcome {
	switch o {
	case model.SyncOutcomeComplete:
		return protocol.SyncOutcome_SYNC_OUTCOME_COMPLETE
	case model.SyncOutcomeUpToDate:
		return protocol.SyncOutcome_SYNC_OUTCOME_UP_TO_DATE
	case model.SyncOutcomeCatalogRefreshed:
		return protocol.SyncOutcome_SYNC_OUTCOME_CATALOG_REFRESHED
	case model.SyncOutcomeFailed:
		return protocol.SyncOutcome_SYNC_OUTCOME_FAILED
	case model.SyncOutcomeCancelled:
		return protocol.SyncOutcome_SYNC_OUTCOME_CANCELLED
	default:
		return protocol.SyncOutcome_SYNC_OUTCOME_UNSPECIFIED
	}
}

func toProtoSyncSession(s *model.SyncSession) *protocol.SyncSession {
	return &protocol.SyncSession{
		Id:              s.ID,
		CameraId:        s.CameraID,
		StartedAt:       timestamppb.New(s.StartedAt),
		FinishedAt:      timestamppb.New(s.FinishedAt),
		Outcome:         toProtoSyncOutcome(s.Outcome),
		Error:           s.Error,
		FailedStep:      s.FailedStep,
		StepIndex:       s.StepIndex,
		StepCount:       s.StepCount,
		FilesDownloaded: s.FilesDownloaded,
		FilesFailed:     s.FilesFailed,
		FilesSkipped:    s.FilesSkipped,
		BytesDownloaded: s.BytesDownloaded,
		Files:           toProtoSyncFiles(s.Files),
		Phases:          toProtoPhaseTimings(s.Phases),
		Selection:       s.Selection,
	}
}

func toProtoGroup(g *model.Group) *protocol.Group {
	protoGroup := &protocol.Group{
		Id:        g.ID,
		Name:      g.Name,
		CameraIds: make([]string, len(g.CameraIDs)),
		CreatedAt: timestamppb.New(g.CreatedAt),
		UpdatedAt: timestamppb.New(g.UpdatedAt),
	}
	copy(protoGroup.CameraIds, g.CameraIDs)
	return protoGroup
}

func toProtoVideoFile(v *model.VideoFile) *protocol.VideoFile {
	return &protocol.VideoFile{
		Id:              v.ID,
		Name:            v.Name,
		Path:            v.Path,
		SizeBytes:       v.SizeBytes,
		CreatedAt:       timestamppb.New(v.CreatedAt),
		SyncedAt:        timestamppb.New(v.SyncedAt),
		CameraId:        v.CameraID,
		MimeType:        v.MimeType,
		DurationSeconds: v.DurationSeconds,
		ThumbnailPath:   v.ThumbnailPath,
		HasProcessed:    v.HasProcessed,
		PreviewPath:     v.PreviewPath,
	}
}

func toProtoCameraSettings(settings []model.CameraSetting) []*protocol.CameraSetting {
	out := make([]*protocol.CameraSetting, 0, len(settings))
	for _, s := range settings {
		ps := &protocol.CameraSetting{
			Id:        s.ID,
			Name:      s.Name,
			Value:     s.Value,
			ValueName: s.ValueName,
		}
		for _, o := range s.Options {
			ps.Options = append(ps.Options, &protocol.SettingOption{Value: o.Value, Name: o.Name})
		}
		out = append(out, ps)
	}
	return out
}

func toProtoCameraMediaItem(item model.CameraMediaItem) *protocol.CameraMediaItem {
	return &protocol.CameraMediaItem{
		Name:          item.Name,
		CameraPath:    item.CameraPath,
		SizeBytes:     item.SizeBytes,
		CreatedAt:     timestamppb.New(item.CreatedAt),
		ThumbnailPath: item.ThumbnailPath,
		Downloaded:    item.Downloaded,
		LocalPath:     item.LocalPath,
		PreviewPath:   item.PreviewPath,
	}
}

func toProtoCameraSettingsResult(cameraID, errMsg string, results []model.SettingApplyResult) *protocol.CameraSettingsResult {
	return &protocol.CameraSettingsResult{
		CameraId: cameraID,
		Error:    errMsg,
		Results:  toProtoSettingApplyResults(results),
	}
}

func toProtoSettingApplyResults(results []model.SettingApplyResult) []*protocol.SettingApplyResult {
	out := make([]*protocol.SettingApplyResult, 0, len(results))
	for _, r := range results {
		out = append(out, &protocol.SettingApplyResult{Id: r.ID, Error: r.Error})
	}
	return out
}

func toProtoConfig(c model.Config) *protocol.Config {
	return &protocol.Config{
		PairModeEnabled:            c.PairModeEnabled,
		SyncEnabled:                c.SyncEnabled,
		ScanIntervalSeconds:        c.ScanIntervalSeconds,
		ConnectTimeoutSeconds:      c.ConnectTimeoutSeconds,
		DaysThreshold:              c.DaysThreshold,
		DestinationFolder:          c.DestinationFolder,
		InactivityTimeoutSeconds:   c.InactivityTimeoutSeconds,
		StatusCheckIntervalSeconds: c.StatusCheckIntervalSeconds,
		CheckOnReturn:              c.CheckOnReturn,
		SetTimeEnabled:             c.SetTimeEnabled,
		TurboEnabled:               c.TurboEnabled,
		LogLevel:                   c.LogLevel,
		LastUpdated:                timestamppb.New(c.LastUpdated),
	}
}
