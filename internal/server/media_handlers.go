package server

import (
	"context"

	"github.com/dropz/dropz/internal/protocol"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetCameraMedia returns the cached media catalog for one camera.
func (s *DropzServer) GetCameraMedia(ctx context.Context, req *protocol.GetCameraMediaRequest) (*protocol.GetCameraMediaResponse, error) {
	if req.GetCameraId() == "" {
		return nil, status.Error(codes.InvalidArgument, "camera_id required")
	}
	items, updated, err := s.manager.GetCameraMedia(req.GetCameraId())
	if err != nil {
		return nil, rpcError(err)
	}
	resp := &protocol.GetCameraMediaResponse{}
	for _, item := range items {
		resp.Items = append(resp.Items, toProtoCameraMediaItem(item))
	}
	if !updated.IsZero() {
		resp.UpdatedAt = timestamppb.New(updated)
	}
	return resp, nil
}

// RequestMediaDownload queues a selection download or catalog refresh.
func (s *DropzServer) RequestMediaDownload(ctx context.Context, req *protocol.RequestMediaDownloadRequest) (*protocol.RequestMediaDownloadResponse, error) {
	if req.GetCameraId() == "" {
		return nil, status.Error(codes.InvalidArgument, "camera_id required")
	}
	entry, err := s.manager.RequestMediaDownload(req.GetCameraId(), req.GetFileNames())
	if err != nil {
		return nil, rpcError(err)
	}
	return &protocol.RequestMediaDownloadResponse{Entry: toProtoSyncQueueEntry(entry)}, nil
}

// PreviewMedia fetches one clip's LRV proxy into the preview cache.
func (s *DropzServer) PreviewMedia(ctx context.Context, req *protocol.PreviewMediaRequest) (*protocol.PreviewMediaResponse, error) {
	if req.GetCameraId() == "" || req.GetCameraPath() == "" {
		return nil, status.Error(codes.InvalidArgument, "camera_id and camera_path required")
	}
	if err := s.manager.PreviewMedia(req.GetCameraId(), req.GetCameraPath()); err != nil {
		return nil, rpcError(err)
	}
	return &protocol.PreviewMediaResponse{}, nil
}
