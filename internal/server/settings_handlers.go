package server

import (
	"context"

	"github.com/dropz/dropz/internal/protocol"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetCameraSettings returns the stored snapshot, or reads fresh from the
// camera over BLE when refresh is set.
func (s *DropzServer) GetCameraSettings(ctx context.Context, req *protocol.GetCameraSettingsRequest) (*protocol.GetCameraSettingsResponse, error) {
	if req.GetCameraId() == "" {
		return nil, status.Error(codes.InvalidArgument, "camera_id required")
	}
	get := s.manager.GetCameraSettings
	if req.GetRefresh() {
		get = s.manager.RefreshCameraSettings
	}
	settings, updated, err := get(req.GetCameraId())
	if err != nil {
		return nil, rpcError(err)
	}
	resp := &protocol.GetCameraSettingsResponse{Settings: toProtoCameraSettings(settings)}
	if !updated.IsZero() {
		resp.UpdatedAt = timestamppb.New(updated)
	}
	return resp, nil
}

// ApplyCameraSettings writes settings to one camera or every camera in a
// group. Per-camera and per-setting outcomes are always reported; a camera
// rejecting an option its model lacks is a result, not an RPC error.
func (s *DropzServer) ApplyCameraSettings(ctx context.Context, req *protocol.ApplyCameraSettingsRequest) (*protocol.ApplyCameraSettingsResponse, error) {
	if len(req.GetChanges()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no changes given")
	}
	if (req.GetCameraId() == "") == (req.GetGroupId() == "") {
		return nil, status.Error(codes.InvalidArgument, "exactly one of camera_id or group_id required")
	}
	changes := make(map[int32]int64, len(req.GetChanges()))
	for _, c := range req.GetChanges() {
		changes[c.GetId()] = c.GetValue()
	}

	if cameraID := req.GetCameraId(); cameraID != "" {
		results, err := s.manager.ApplyCameraSettings(cameraID, changes)
		if err != nil {
			return nil, rpcError(err)
		}
		return &protocol.ApplyCameraSettingsResponse{Cameras: []*protocol.CameraSettingsResult{
			toProtoCameraSettingsResult(cameraID, "", results),
		}}, nil
	}

	groupResults, err := s.manager.ApplyGroupSettings(req.GetGroupId(), changes)
	if err != nil {
		return nil, rpcError(err)
	}
	resp := &protocol.ApplyCameraSettingsResponse{}
	for _, gr := range groupResults {
		resp.Cameras = append(resp.Cameras, toProtoCameraSettingsResult(gr.CameraID, gr.Error, gr.Results))
	}
	return resp, nil
}
