package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/dropz/dropz/internal/model"
	"github.com/dropz/dropz/internal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Manager defines the operations the server needs from the business layer
type Manager interface {
	// Reads
	GetDiscoveredCameras() []*model.DiscoveredCamera
	GetManagedCameras() []*model.ManagedCamera
	GetSyncQueue() []*model.SyncQueueEntry
	GetGroups() []*model.Group
	// Camera
	ManageCamera(cameraID string) (*model.ManagedCamera, error)
	UnmanageCamera(cameraID string) error
	PairCamera(cameraID string) (*model.ManagedCamera, error)
	// Sync
	ForceSync(cameraID string) (*model.SyncQueueEntry, error)
	CancelSync(cameraID string) error
	GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*model.VideoFile, int)
	// Groups
	CreateGroup(name string, cameraIDs []string) (*model.Group, error)
	UpdateGroup(groupID, name string, cameraIDs []string) (*model.Group, error)
	DeleteGroup(groupID string) error
	LoadGroup(groupID string) error
	SaveManagedAsGroup(name string) (*model.Group, error)
	// Config
	GetConfig() model.Config
	UpdateConfig(config model.Config) error
	GetSetting(settingName string) (interface{}, error)
	UpdateSetting(settingName string, value interface{}) (model.Config, error)
	ResetSetting(settingName string) (model.Config, error)
	// Notifications
	SetNotifier(notifier func())
}

// rpcError maps domain errors to gRPC status codes.
func rpcError(err error) error {
	switch {
	case errors.Is(err, model.ErrCameraNotFound), errors.Is(err, model.ErrGroupNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, model.ErrNotManagedPaired):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// streamEntry tracks a single gRPC stream and its done channel
type streamEntry struct {
	stream grpc.ServerStream
	done   chan bool
	send   func(heartbeat bool) error // sends either data or heartbeat
}

// DropzServer implements the DropzService gRPC service
type DropzServer struct {
	protocol.UnimplementedDropzServiceServer
	manager Manager
	log     *slog.Logger
	server  *grpc.Server

	// All streams tracked uniformly
	streams     []*streamEntry
	streamMutex sync.RWMutex

	// Internal state management
	heartbeatInterval   time.Duration
	streamUpdateChannel chan struct{}
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
}

// NewDropzServer creates a new DropzServer
func NewDropzServer(manager Manager, log *slog.Logger) *DropzServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &DropzServer{
		manager:             manager,
		log:                 log.With("component", "server"),
		heartbeatInterval:   10 * time.Second,
		streamUpdateChannel: make(chan struct{}, 10),
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// unaryShutdownInterceptor rejects unary RPCs once shutdown has started.
func (s *DropzServer) unaryShutdownInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if s.ctx.Err() != nil {
		return nil, status.Error(codes.Unavailable, "server is shutting down")
	}
	return handler(ctx, req)
}

// streamShutdownInterceptor rejects new streams once shutdown has started.
func (s *DropzServer) streamShutdownInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if s.ctx.Err() != nil {
		return status.Error(codes.Unavailable, "server is shutting down")
	}
	return handler(srv, ss)
}

// Start starts the gRPC server
func (s *DropzServer) Start(address string) error {
	s.log.Debug("Starting DropzServer", "address", address)

	s.manager.SetNotifier(s.NotifyUpdate)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	s.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(s.unaryShutdownInterceptor),
		grpc.ChainStreamInterceptor(s.streamShutdownInterceptor),
	)
	protocol.RegisterDropzServiceServer(s.server, s)

	s.log.Info("gRPC server started", "address", address)

	s.wg.Add(2)
	go s.streamUpdateHandler()
	go s.heartbeatSender()

	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.log.Error("gRPC server error", "err", err)
		}
	}()

	return nil
}

// Stop stops the gRPC server
func (s *DropzServer) Stop() {
	s.log.Info("Initiating DropzServer shutdown")

	s.cancel()

	s.streamMutex.Lock()
	for _, entry := range s.streams {
		close(entry.done)
	}
	s.streams = nil
	s.streamMutex.Unlock()

	if s.server != nil {
		s.server.GracefulStop()
	}

	s.wg.Wait()
	s.log.Info("DropzServer shutdown complete")
}

// addStream registers a stream and returns its done channel
func (s *DropzServer) addStream(stream grpc.ServerStream, sendFn func(heartbeat bool) error) chan bool {
	done := make(chan bool)
	s.streamMutex.Lock()
	defer s.streamMutex.Unlock()

	select {
	case <-s.ctx.Done():
		close(done)
		return done
	default:
	}

	s.streams = append(s.streams, &streamEntry{
		stream: stream,
		done:   done,
		send:   sendFn,
	})
	return done
}

// removeStream unregisters a stream
func (s *DropzServer) removeStream(stream grpc.ServerStream) {
	s.streamMutex.Lock()
	defer s.streamMutex.Unlock()

	for i, entry := range s.streams {
		if entry.stream == stream {
			s.streams = append(s.streams[:i], s.streams[i+1:]...)
			return
		}
	}
}

// streamUpdateHandler sends updates to all active streams when changes occur
func (s *DropzServer) streamUpdateHandler() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case _, ok := <-s.streamUpdateChannel:
			if !ok {
				return
			}
			// Coalesce bursts: BLE advertisements can notify several times
			// per second and each broadcast is a full snapshot per stream.
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
			select {
			case <-s.streamUpdateChannel:
			default:
			}
			s.forEachStream(false)
		}
	}
}

// heartbeatSender periodically sends heartbeats to all active streams
func (s *DropzServer) heartbeatSender() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			s.forEachStream(true)
		}
	}
}

// forEachStream calls send on every registered stream, removing failed ones
func (s *DropzServer) forEachStream(heartbeat bool) {
	s.streamMutex.Lock()
	defer s.streamMutex.Unlock()

	alive := s.streams[:0]
	for _, entry := range s.streams {
		select {
		case <-entry.done:
			continue // already closed
		default:
		}

		if err := entry.send(heartbeat); err != nil {
			s.log.Error("Stream send failed, removing", "err", err)
			close(entry.done)
			continue
		}
		alive = append(alive, entry)
	}
	s.streams = alive
}

// NotifyUpdate notifies all connected clients of state changes
func (s *DropzServer) NotifyUpdate() {
	// Drain any pending signal, then send a fresh one.
	// This ensures notifications are never dropped.
	select {
	case <-s.streamUpdateChannel:
	default:
	}
	select {
	case s.streamUpdateChannel <- struct{}{}:
	default:
	}
}

// watchStream is the generic Watch implementation for all stream types
func (s *DropzServer) watchStream(stream grpc.ServerStream, sendFn func(heartbeat bool) error) error {
	// Send initial data
	if err := sendFn(false); err != nil {
		return err
	}

	done := s.addStream(stream, sendFn)
	defer s.removeStream(stream)

	select {
	case <-done:
	case <-stream.Context().Done():
	case <-s.ctx.Done():
	}
	return nil
}

// GetDiscoveredCameras implements the GetDiscoveredCameras RPC method
func (s *DropzServer) GetDiscoveredCameras(ctx context.Context, req *protocol.GetDiscoveredCamerasRequest) (*protocol.GetDiscoveredCamerasResponse, error) {
	cameras := s.manager.GetDiscoveredCameras()
	proto := make([]*protocol.DiscoveredCamera, len(cameras))
	for i, cam := range cameras {
		proto[i] = toProtoDiscoveredCamera(cam)
	}
	return &protocol.GetDiscoveredCamerasResponse{Cameras: proto}, nil
}

// WatchDiscoveredCameras implements the streaming RPC
func (s *DropzServer) WatchDiscoveredCameras(req *protocol.GetDiscoveredCamerasRequest, stream protocol.DropzService_WatchDiscoveredCamerasServer) error {
	return s.watchStream(stream, func(heartbeat bool) error {
		if heartbeat {
			return stream.Send(&protocol.GetDiscoveredCamerasResponse{
				Cameras: []*protocol.DiscoveredCamera{}, NoChanges: true, Heartbeat: true,
			})
		}
		resp, err := s.GetDiscoveredCameras(stream.Context(), req)
		if err != nil {
			return err
		}
		return stream.Send(resp)
	})
}

// GetManagedCameras implements the GetManagedCameras RPC method
func (s *DropzServer) GetManagedCameras(ctx context.Context, req *protocol.GetManagedCamerasRequest) (*protocol.GetManagedCamerasResponse, error) {
	cameras := s.manager.GetManagedCameras()
	proto := make([]*protocol.ManagedCamera, len(cameras))
	for i, cam := range cameras {
		proto[i] = toProtoManagedCamera(cam)
	}
	return &protocol.GetManagedCamerasResponse{Cameras: proto}, nil
}

// WatchManagedCameras implements the streaming RPC
func (s *DropzServer) WatchManagedCameras(req *protocol.GetManagedCamerasRequest, stream protocol.DropzService_WatchManagedCamerasServer) error {
	return s.watchStream(stream, func(heartbeat bool) error {
		if heartbeat {
			return stream.Send(&protocol.GetManagedCamerasResponse{
				Cameras: []*protocol.ManagedCamera{}, NoChanges: true, Heartbeat: true,
			})
		}
		resp, err := s.GetManagedCameras(stream.Context(), req)
		if err != nil {
			return err
		}
		return stream.Send(resp)
	})
}

// GetSyncQueue implements the GetSyncQueue RPC method
func (s *DropzServer) GetSyncQueue(ctx context.Context, req *protocol.GetSyncQueueRequest) (*protocol.GetSyncQueueResponse, error) {
	queue := s.manager.GetSyncQueue()
	proto := make([]*protocol.SyncQueueEntry, len(queue))
	for i, entry := range queue {
		proto[i] = toProtoSyncQueueEntry(entry)
	}
	return &protocol.GetSyncQueueResponse{Queue: proto}, nil
}

// WatchSyncQueue implements the streaming RPC
func (s *DropzServer) WatchSyncQueue(req *protocol.GetSyncQueueRequest, stream protocol.DropzService_WatchSyncQueueServer) error {
	return s.watchStream(stream, func(heartbeat bool) error {
		if heartbeat {
			return stream.Send(&protocol.GetSyncQueueResponse{
				Queue: []*protocol.SyncQueueEntry{}, NoChanges: true, Heartbeat: true,
			})
		}
		resp, err := s.GetSyncQueue(stream.Context(), req)
		if err != nil {
			return err
		}
		return stream.Send(resp)
	})
}

// ManageCamera implements the ManageCamera RPC method
func (s *DropzServer) ManageCamera(ctx context.Context, req *protocol.ManageCameraRequest) (*protocol.ManageCameraResponse, error) {
	s.log.Info("Handling ManageCamera request", "camera", req.CameraId)

	managedCamera, err := s.manager.ManageCamera(req.CameraId)
	if err != nil {
		return nil, rpcError(err)
	}
	if managedCamera == nil {
		return nil, status.Error(codes.NotFound, "camera not found")
	}

	s.NotifyUpdate()

	return &protocol.ManageCameraResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s is now managed", req.CameraId),
		Camera:  toProtoManagedCamera(managedCamera),
	}, nil
}

// UnmanageCamera implements the UnmanageCamera RPC method
func (s *DropzServer) UnmanageCamera(ctx context.Context, req *protocol.UnmanageCameraRequest) (*protocol.UnmanageCameraResponse, error) {
	if err := s.manager.UnmanageCamera(req.CameraId); err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()

	return &protocol.UnmanageCameraResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s is no longer managed", req.CameraId),
	}, nil
}

// PairCamera implements the PairCamera RPC method
func (s *DropzServer) PairCamera(ctx context.Context, req *protocol.PairCameraRequest) (*protocol.PairCameraResponse, error) {
	managedCamera, err := s.manager.PairCamera(req.CameraId)
	if err != nil {
		return nil, rpcError(err)
	}
	if managedCamera == nil {
		return nil, status.Error(codes.NotFound, "camera not found")
	}

	s.NotifyUpdate()

	return &protocol.PairCameraResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s paired successfully", req.CameraId),
		Camera:  toProtoManagedCamera(managedCamera),
	}, nil
}

// ForceSync implements the ForceSync RPC method
func (s *DropzServer) ForceSync(ctx context.Context, req *protocol.ForceSyncRequest) (*protocol.ForceSyncResponse, error) {
	queueEntry, err := s.manager.ForceSync(req.CameraId)
	if err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()

	return &protocol.ForceSyncResponse{
		Success:    true,
		Message:    fmt.Sprintf("Camera %s added to sync queue", req.CameraId),
		QueueEntry: toProtoSyncQueueEntry(queueEntry),
	}, nil
}

// CancelSync implements the CancelSync RPC method
func (s *DropzServer) CancelSync(ctx context.Context, req *protocol.CancelSyncRequest) (*protocol.CancelSyncResponse, error) {
	if err := s.manager.CancelSync(req.CameraId); err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()

	return &protocol.CancelSyncResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s sync cancelled and removed from queue", req.CameraId),
	}, nil
}

// GetGroups implements the GetGroups RPC method
func (s *DropzServer) GetGroups(ctx context.Context, req *protocol.GetGroupsRequest) (*protocol.GetGroupsResponse, error) {
	groups := s.manager.GetGroups()
	protoGroups := make([]*protocol.Group, len(groups))
	for i, group := range groups {
		protoGroups[i] = toProtoGroup(group)
	}
	return &protocol.GetGroupsResponse{Groups: protoGroups}, nil
}

// CreateGroup implements the CreateGroup RPC method
func (s *DropzServer) CreateGroup(ctx context.Context, req *protocol.CreateGroupRequest) (*protocol.Group, error) {
	group, err := s.manager.CreateGroup(req.Name, req.CameraIds)
	if err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()
	return toProtoGroup(group), nil
}

// UpdateGroup implements the UpdateGroup RPC method
func (s *DropzServer) UpdateGroup(ctx context.Context, req *protocol.UpdateGroupRequest) (*protocol.Group, error) {
	group, err := s.manager.UpdateGroup(req.GroupId, req.Name, req.CameraIds)
	if err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()
	return toProtoGroup(group), nil
}

// DeleteGroup implements the DeleteGroup RPC method
func (s *DropzServer) DeleteGroup(ctx context.Context, req *protocol.DeleteGroupRequest) (*protocol.OperationResponse, error) {
	if err := s.manager.DeleteGroup(req.GroupId); err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()

	return &protocol.OperationResponse{
		Success: true,
		Message: fmt.Sprintf("Group %s deleted successfully", req.GroupId),
	}, nil
}

// LoadGroup implements the LoadGroup RPC method
func (s *DropzServer) LoadGroup(ctx context.Context, req *protocol.LoadGroupRequest) (*protocol.LoadGroupResponse, error) {
	if err := s.manager.LoadGroup(req.GroupId); err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()

	return &protocol.LoadGroupResponse{
		Success: true,
		Message: fmt.Sprintf("Group %s loaded successfully", req.GroupId),
	}, nil
}

// SaveManagedAsGroup implements the SaveManagedAsGroup RPC method
func (s *DropzServer) SaveManagedAsGroup(ctx context.Context, req *protocol.SaveManagedAsGroupRequest) (*protocol.Group, error) {
	group, err := s.manager.SaveManagedAsGroup(req.Name)
	if err != nil {
		return nil, rpcError(err)
	}

	s.NotifyUpdate()
	return toProtoGroup(group), nil
}

// GetVideos implements the GetVideos RPC method
func (s *DropzServer) GetVideos(ctx context.Context, req *protocol.GetVideosRequest) (*protocol.GetVideosResponse, error) {
	var startDate, endDate time.Time
	if req.StartDate != nil {
		startDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		endDate = req.EndDate.AsTime()
	}

	videos, totalCount := s.manager.GetVideosByCamera(req.CameraId, startDate, endDate, int(req.Limit), int(req.Offset))
	protoVideos := make([]*protocol.VideoFile, len(videos))
	for i, video := range videos {
		protoVideos[i] = toProtoVideoFile(video)
	}

	return &protocol.GetVideosResponse{
		Videos:     protoVideos,
		TotalCount: int32(totalCount),
	}, nil
}

// GetConfig implements the GetConfig RPC method
func (s *DropzServer) GetConfig(ctx context.Context, req *protocol.GetConfigRequest) (*protocol.GetConfigResponse, error) {
	config := s.manager.GetConfig()
	return &protocol.GetConfigResponse{Config: toProtoConfig(config)}, nil
}

// UpdateConfig implements the UpdateConfig RPC method
func (s *DropzServer) UpdateConfig(ctx context.Context, req *protocol.UpdateConfigRequest) (*protocol.UpdateConfigResponse, error) {
	if req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "no config provided")
	}

	dbConfig := model.Config{
		PairModeEnabled:            req.Config.PairModeEnabled,
		SyncEnabled:                req.Config.SyncEnabled,
		ScanIntervalSeconds:        req.Config.ScanIntervalSeconds,
		ConnectTimeoutSeconds:      req.Config.ConnectTimeoutSeconds,
		DaysThreshold:              req.Config.DaysThreshold,
		DestinationFolder:          req.Config.DestinationFolder,
		InactivityTimeoutSeconds:   req.Config.InactivityTimeoutSeconds,
		StatusCheckIntervalSeconds: req.Config.StatusCheckIntervalSeconds,
		CheckOnReturn:              req.Config.CheckOnReturn,
		SetTimeEnabled:             req.Config.SetTimeEnabled,
		LogLevel:                   req.Config.LogLevel,
		LastUpdated:                req.Config.LastUpdated.AsTime(),
	}

	if err := s.manager.UpdateConfig(dbConfig); err != nil {
		return nil, rpcError(err)
	}

	updatedConfig := s.manager.GetConfig()
	return &protocol.UpdateConfigResponse{
		Success: true,
		Message: "Configuration updated successfully",
		Config:  toProtoConfig(updatedConfig),
	}, nil
}

// GetSetting implements the GetSetting RPC method
func (s *DropzServer) GetSetting(ctx context.Context, req *protocol.GetSettingRequest) (*protocol.GetSettingResponse, error) {
	value, err := s.manager.GetSetting(req.SettingName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	response := &protocol.GetSettingResponse{}
	switch v := value.(type) {
	case bool:
		response.Value = &protocol.GetSettingResponse_BoolValue{BoolValue: v}
	case int32:
		response.Value = &protocol.GetSettingResponse_IntValue{IntValue: v}
	case string:
		response.Value = &protocol.GetSettingResponse_StringValue{StringValue: v}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported setting type for %s", req.SettingName)
	}

	return response, nil
}

// UpdateSetting implements the UpdateSetting RPC method
func (s *DropzServer) UpdateSetting(ctx context.Context, req *protocol.UpdateSettingRequest) (*protocol.Config, error) {
	var value interface{}
	switch req.Value.(type) {
	case *protocol.UpdateSettingRequest_BoolValue:
		value = req.GetBoolValue()
	case *protocol.UpdateSettingRequest_IntValue:
		value = req.GetIntValue()
	case *protocol.UpdateSettingRequest_StringValue:
		value = req.GetStringValue()
	default:
		return nil, status.Errorf(codes.InvalidArgument, "no value provided for setting %s", req.SettingName)
	}

	config, err := s.manager.UpdateSetting(req.SettingName, value)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	s.NotifyUpdate()
	return toProtoConfig(config), nil
}

// ResetSetting implements the ResetSetting RPC method
func (s *DropzServer) ResetSetting(ctx context.Context, req *protocol.ResetSettingRequest) (*protocol.Config, error) {
	config, err := s.manager.ResetSetting(req.SettingName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	s.NotifyUpdate()
	return toProtoConfig(config), nil
}
