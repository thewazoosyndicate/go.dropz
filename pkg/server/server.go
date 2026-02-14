package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/dropz/dropz/pkg/protocol"
	"google.golang.org/grpc"
)

// Manager defines the operations the server needs from the business layer
type Manager interface {
	// Camera
	ManageCamera(cameraID string) (*database.ManagedCamera, error)
	UnmanageCamera(cameraID string) error
	PairCamera(cameraID string) (*database.ManagedCamera, error)
	// Sync
	ForceSync(cameraID string) (*database.SyncQueueEntry, error)
	CancelSync(cameraID string) error
	GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*database.VideoFile, int)
	// Groups
	CreateGroup(name string, cameraIDs []string) (*database.Group, error)
	UpdateGroup(groupID, name string, cameraIDs []string) (*database.Group, error)
	DeleteGroup(groupID string) error
	LoadGroup(groupID string) error
	SaveManagedAsGroup(name string) (*database.Group, error)
	// Camera settings (BLE)
	GetCameraSettings(cameraID string) ([]database.CameraSettingInfo, error)
	SetCameraSetting(cameraID string, settingID int32, value int32) error
	SetGroupSetting(groupID string, settingID int32, value int32) (map[string]string, error)
	// Config
	GetConfig() database.Config
	UpdateConfig(config database.Config) error
	GetSetting(settingName string) (interface{}, error)
	UpdateSetting(settingName string, value interface{}) (database.Config, error)
	ResetSetting(settingName string) (database.Config, error)
	// Notifications
	SetNotifier(notifier func())
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
	log     *logrus.Logger
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
func NewDropzServer(manager Manager, log *logrus.Logger) *DropzServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &DropzServer{
		manager:             manager,
		log:                 log,
		heartbeatInterval:   10 * time.Second,
		streamUpdateChannel: make(chan struct{}, 10),
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// Start starts the gRPC server
func (s *DropzServer) Start(address string) error {
	s.log.Debug("Starting DropzServer", "address", address)

	s.manager.SetNotifier(s.NotifyUpdate)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", address, err)
	}

	s.server = grpc.NewServer()
	protocol.RegisterDropzServiceServer(s.server, s)

	s.log.WithFields(logrus.Fields{"address": address}).Info("gRPC server started")

	s.wg.Add(2)
	go s.streamUpdateHandler()
	go s.heartbeatSender()

	go func() {
		if err := s.server.Serve(listener); err != nil {
			s.log.Error("gRPC server error", "error", err)
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
			select {
			case <-s.ctx.Done():
				return
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
			s.log.Errorf("Stream send failed, removing: %v", err)
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
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	db := database.GetDatabase()
	cameras := db.GetCamerasForDiscoveredPool()
	proto := make([]*protocol.DiscoveredCamera, len(cameras))
	for i, cam := range cameras {
		proto[i] = cam.ToProtoDiscoveredCamera()
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
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	db := database.GetDatabase()
	cameras := db.GetCamerasForManagedPool()
	proto := make([]*protocol.ManagedCamera, len(cameras))
	for i, cam := range cameras {
		proto[i] = cam.ToProtoManagedCamera()
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
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	db := database.GetDatabase()
	queue := db.GetSyncQueue()
	proto := make([]*protocol.SyncQueueEntry, len(queue))
	for i, entry := range queue {
		proto[i] = entry.ToProtoSyncQueueEntry()
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
	if s.ctx.Err() != nil {
		return &protocol.ManageCameraResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}
	s.log.WithFields(logrus.Fields{"camera_id": req.CameraId}).Info("Handling ManageCamera request")

	managedCamera, err := s.manager.ManageCamera(req.CameraId)
	if err != nil {
		return &protocol.ManageCameraResponse{Success: false, Message: err.Error()}, nil
	}

	if managedCamera == nil {
		return &protocol.ManageCameraResponse{
			Success: false,
			Message: "Failed to manage camera: camera not found or not eligible for management",
		}, nil
	}

	s.NotifyUpdate()

	return &protocol.ManageCameraResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s is now managed", req.CameraId),
		Camera:  managedCamera.ToProtoManagedCamera(),
	}, nil
}

// UnmanageCamera implements the UnmanageCamera RPC method
func (s *DropzServer) UnmanageCamera(ctx context.Context, req *protocol.UnmanageCameraRequest) (*protocol.UnmanageCameraResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.UnmanageCameraResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	err := s.manager.UnmanageCamera(req.CameraId)
	if err != nil {
		return &protocol.UnmanageCameraResponse{Success: false, Message: err.Error()}, nil
	}

	s.NotifyUpdate()

	return &protocol.UnmanageCameraResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s is no longer managed", req.CameraId),
	}, nil
}

// PairCamera implements the PairCamera RPC method
func (s *DropzServer) PairCamera(ctx context.Context, req *protocol.PairCameraRequest) (*protocol.PairCameraResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.PairCameraResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	managedCamera, err := s.manager.PairCamera(req.CameraId)
	if err != nil {
		return &protocol.PairCameraResponse{Success: false, Message: err.Error()}, nil
	}

	if managedCamera == nil {
		return &protocol.PairCameraResponse{
			Success: false,
			Message: "Failed to pair camera: camera not found or not eligible for pairing",
		}, nil
	}

	s.NotifyUpdate()

	return &protocol.PairCameraResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s paired successfully", req.CameraId),
		Camera:  managedCamera.ToProtoManagedCamera(),
	}, nil
}

// ForceSync implements the ForceSync RPC method
func (s *DropzServer) ForceSync(ctx context.Context, req *protocol.ForceSyncRequest) (*protocol.ForceSyncResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.ForceSyncResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	queueEntry, err := s.manager.ForceSync(req.CameraId)
	if err != nil {
		return &protocol.ForceSyncResponse{Success: false, Message: err.Error()}, nil
	}

	s.NotifyUpdate()

	return &protocol.ForceSyncResponse{
		Success:    true,
		Message:    fmt.Sprintf("Camera %s added to sync queue", req.CameraId),
		QueueEntry: queueEntry.ToProtoSyncQueueEntry(),
	}, nil
}

// CancelSync implements the CancelSync RPC method
func (s *DropzServer) CancelSync(ctx context.Context, req *protocol.CancelSyncRequest) (*protocol.CancelSyncResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.CancelSyncResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	err := s.manager.CancelSync(req.CameraId)
	if err != nil {
		return &protocol.CancelSyncResponse{Success: false, Message: err.Error()}, nil
	}

	s.NotifyUpdate()

	return &protocol.CancelSyncResponse{
		Success: true,
		Message: fmt.Sprintf("Camera %s sync cancelled and removed from queue", req.CameraId),
	}, nil
}

// GetGroups implements the GetGroups RPC method
func (s *DropzServer) GetGroups(ctx context.Context, req *protocol.GetGroupsRequest) (*protocol.GetGroupsResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	db := database.GetDatabase()
	groups := db.GetAllGroups()
	protoGroups := make([]*protocol.Group, len(groups))
	for i, group := range groups {
		protoGroups[i] = group.ToProtoGroup()
	}
	return &protocol.GetGroupsResponse{Groups: protoGroups}, nil
}

// CreateGroup implements the CreateGroup RPC method
func (s *DropzServer) CreateGroup(ctx context.Context, req *protocol.CreateGroupRequest) (*protocol.Group, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	group, err := s.manager.CreateGroup(req.Name, req.CameraIds)
	if err != nil {
		return nil, err
	}

	s.NotifyUpdate()
	return group.ToProtoGroup(), nil
}

// UpdateGroup implements the UpdateGroup RPC method
func (s *DropzServer) UpdateGroup(ctx context.Context, req *protocol.UpdateGroupRequest) (*protocol.Group, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	group, err := s.manager.UpdateGroup(req.GroupId, req.Name, req.CameraIds)
	if err != nil {
		return nil, err
	}

	s.NotifyUpdate()
	return group.ToProtoGroup(), nil
}

// DeleteGroup implements the DeleteGroup RPC method
func (s *DropzServer) DeleteGroup(ctx context.Context, req *protocol.DeleteGroupRequest) (*protocol.OperationResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.OperationResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	err := s.manager.DeleteGroup(req.GroupId)
	if err != nil {
		return &protocol.OperationResponse{Success: false, Message: err.Error()}, nil
	}

	s.NotifyUpdate()

	return &protocol.OperationResponse{
		Success: true,
		Message: fmt.Sprintf("Group %s deleted successfully", req.GroupId),
	}, nil
}

// LoadGroup implements the LoadGroup RPC method
func (s *DropzServer) LoadGroup(ctx context.Context, req *protocol.LoadGroupRequest) (*protocol.LoadGroupResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.LoadGroupResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	err := s.manager.LoadGroup(req.GroupId)
	if err != nil {
		return &protocol.LoadGroupResponse{Success: false, Message: err.Error()}, nil
	}

	s.NotifyUpdate()

	return &protocol.LoadGroupResponse{
		Success: true,
		Message: fmt.Sprintf("Group %s loaded successfully", req.GroupId),
	}, nil
}

// SaveManagedAsGroup implements the SaveManagedAsGroup RPC method
func (s *DropzServer) SaveManagedAsGroup(ctx context.Context, req *protocol.SaveManagedAsGroupRequest) (*protocol.Group, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	group, err := s.manager.SaveManagedAsGroup(req.Name)
	if err != nil {
		return nil, err
	}

	s.NotifyUpdate()
	return group.ToProtoGroup(), nil
}

// GetVideos implements the GetVideos RPC method
func (s *DropzServer) GetVideos(ctx context.Context, req *protocol.GetVideosRequest) (*protocol.GetVideosResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

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
		protoVideos[i] = video.ToProtoVideoFile()
	}

	return &protocol.GetVideosResponse{
		Videos:     protoVideos,
		TotalCount: int32(totalCount),
	}, nil
}

// GetCameraSettings implements the GetCameraSettings RPC method
func (s *DropzServer) GetCameraSettings(ctx context.Context, req *protocol.GetCameraSettingsRequest) (*protocol.GetCameraSettingsResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	settings, err := s.manager.GetCameraSettings(req.CameraId)
	if err != nil {
		return nil, err
	}

	protoSettings := make([]*protocol.CameraSetting, len(settings))
	for i, s := range settings {
		protoSettings[i] = s.ToProtoCameraSetting()
	}
	return &protocol.GetCameraSettingsResponse{Settings: protoSettings}, nil
}

// SetCameraSetting implements the SetCameraSetting RPC method
func (s *DropzServer) SetCameraSetting(ctx context.Context, req *protocol.SetCameraSettingRequest) (*protocol.OperationResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.OperationResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	err := s.manager.SetCameraSetting(req.CameraId, req.SettingId, req.Value)
	if err != nil {
		return &protocol.OperationResponse{Success: false, Message: err.Error()}, nil
	}

	return &protocol.OperationResponse{
		Success: true,
		Message: fmt.Sprintf("Setting %d updated on camera %s", req.SettingId, req.CameraId),
	}, nil
}

// SetGroupSetting implements the SetGroupSetting RPC method
func (s *DropzServer) SetGroupSetting(ctx context.Context, req *protocol.SetGroupSettingRequest) (*protocol.SetGroupSettingResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	results, err := s.manager.SetGroupSetting(req.GroupId, req.SettingId, req.Value)
	if err != nil {
		return nil, err
	}

	return &protocol.SetGroupSettingResponse{CameraResults: results}, nil
}

// GetConfig implements the GetConfig RPC method
func (s *DropzServer) GetConfig(ctx context.Context, req *protocol.GetConfigRequest) (*protocol.GetConfigResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	config := s.manager.GetConfig()
	return &protocol.GetConfigResponse{Config: config.ToProtoConfig()}, nil
}

// UpdateConfig implements the UpdateConfig RPC method
func (s *DropzServer) UpdateConfig(ctx context.Context, req *protocol.UpdateConfigRequest) (*protocol.UpdateConfigResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.UpdateConfigResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}

	if req.Config == nil {
		return &protocol.UpdateConfigResponse{Success: false, Message: "No config provided"}, nil
	}

	dbConfig := database.Config{
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

	err := s.manager.UpdateConfig(dbConfig)
	if err != nil {
		return &protocol.UpdateConfigResponse{Success: false, Message: err.Error()}, nil
	}

	updatedConfig := s.manager.GetConfig()
	return &protocol.UpdateConfigResponse{
		Success: true,
		Message: "Configuration updated successfully",
		Config:  updatedConfig.ToProtoConfig(),
	}, nil
}

// GetSetting implements the GetSetting RPC method
func (s *DropzServer) GetSetting(ctx context.Context, req *protocol.GetSettingRequest) (*protocol.GetSettingResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	value, err := s.manager.GetSetting(req.SettingName)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("unsupported setting type for %s", req.SettingName)
	}

	return response, nil
}

// UpdateSetting implements the UpdateSetting RPC method
func (s *DropzServer) UpdateSetting(ctx context.Context, req *protocol.UpdateSettingRequest) (*protocol.Config, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	var value interface{}
	switch req.Value.(type) {
	case *protocol.UpdateSettingRequest_BoolValue:
		value = req.GetBoolValue()
	case *protocol.UpdateSettingRequest_IntValue:
		value = req.GetIntValue()
	case *protocol.UpdateSettingRequest_StringValue:
		value = req.GetStringValue()
	default:
		return nil, fmt.Errorf("no value provided for setting %s", req.SettingName)
	}

	config, err := s.manager.UpdateSetting(req.SettingName, value)
	if err != nil {
		return nil, err
	}

	s.NotifyUpdate()
	return config.ToProtoConfig(), nil
}

// ResetSetting implements the ResetSetting RPC method
func (s *DropzServer) ResetSetting(ctx context.Context, req *protocol.ResetSettingRequest) (*protocol.Config, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}

	config, err := s.manager.ResetSetting(req.SettingName)
	if err != nil {
		return nil, err
	}

	s.NotifyUpdate()
	return config.ToProtoConfig(), nil
}

