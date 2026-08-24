package model

import "testing"

func TestNewMediaCount(t *testing.T) {
	cases := []struct {
		name string
		md   CameraMetadata
		want int32
	}{
		{"no baseline", CameraMetadata{NumPhotos: 5, NumVideos: 5}, 0},
		{"clean", CameraMetadata{NumPhotos: 5, NumVideos: 5, SyncedNumPhotos: 5, SyncedNumVideos: 5, SyncedSpaceKB: 1}, 0},
		{"new clips", CameraMetadata{NumPhotos: 6, NumVideos: 7, SyncedNumPhotos: 5, SyncedNumVideos: 5, SyncedSpaceKB: 1}, 3},
		{"deleted on camera", CameraMetadata{NumPhotos: 1, NumVideos: 1, SyncedNumPhotos: 5, SyncedNumVideos: 5, SyncedSpaceKB: 1}, 0},
	}
	for _, tc := range cases {
		cs := &CameraWithState{Metadata: tc.md}
		if got := cs.NewMediaCount(); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestSyncSessionCountFiles(t *testing.T) {
	s := &SyncSession{Files: []SyncFile{
		{State: SyncFileDone, SizeBytes: 10},
		{State: SyncFileDone, SizeBytes: 20},
		{State: SyncFileFailed, SizeBytes: 99},
		{State: SyncFileSkipped, SizeBytes: 5},
		{State: SyncFileQueued, SizeBytes: 7},
	}}
	s.CountFiles()
	if s.FilesDownloaded != 2 || s.FilesFailed != 1 || s.FilesSkipped != 1 || s.BytesDownloaded != 30 {
		t.Errorf("counts = %d/%d/%d %dB", s.FilesDownloaded, s.FilesFailed, s.FilesSkipped, s.BytesDownloaded)
	}
}
