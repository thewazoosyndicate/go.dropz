package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dropz/dropz/pkg/common"
	"github.com/dropz/dropz/pkg/database"
	"github.com/dropz/dropz/pkg/logger"
	"github.com/dropz/dropz/pkg/protocol"
	"google.golang.org/grpc"
)

// Manager is the interface the DropzServer needs from the manager package
type Manager interface {
	// Methods required by the server
	ManageCamera(cameraID string) (*database.ManagedCamera, error)
	UnmanageCamera(cameraID string) error
	PairCamera(cameraID string) (*database.ManagedCamera, error)
	ForceSync(cameraID string) (*database.SyncQueueEntry, error)
	CancelSync(cameraID string) error
	CreateGroup(name string, cameraIDs []string) (*database.Group, error)
	UpdateGroup(groupID, name string, cameraIDs []string) (*database.Group, error)
	DeleteGroup(groupID string) error
	GetAllVideos() []*database.VideoFile
	GetVideosByCamera(cameraID string, startDate, endDate time.Time, limit, offset int) ([]*database.VideoFile, int)
	GetConfig() database.Config
	UpdateConfig(config database.Config) error
	GetSetting(settingName string) (interface{}, error)
	UpdateSetting(settingName string, value interface{}) (database.Config, error)
	ResetSetting(settingName string) (database.Config, error)

	// Notifier interface methods
	SetNotifier(notifier common.UpdateNotifier)
}

// DropzServer implements the DropzService gRPC service
type DropzServer struct {
	protocol.UnimplementedDropzServiceServer
	manager Manager
	log     logger.Logger
	server  *grpc.Server

	// Stream management
	discoveredStreams map[protocol.DropzService_WatchDiscoveredCamerasServer]chan bool
	managedStreams    map[protocol.DropzService_WatchManagedCamerasServer]chan bool
	syncQueueStreams  map[protocol.DropzService_WatchSyncQueueServer]chan bool

	// Internal state management
	lastHeartbeat       time.Time
	mutex               sync.RWMutex
	streamMutex         sync.RWMutex
	heartbeatInterval   time.Duration
	streamUpdateChannel chan struct{}
	ctx                 context.Context    // For managing lifecycle of internal goroutines
	cancel              context.CancelFunc // For managing lifecycle of internal goroutines
	wg                  sync.WaitGroup     // To wait for internal goroutines
}

// NewDropzServer creates a new DropzServer
func NewDropzServer(manager Manager) *DropzServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &DropzServer{
		manager:             manager,
		log:                 logger.GetLogger(),
		discoveredStreams:   make(map[protocol.DropzService_WatchDiscoveredCamerasServer]chan bool),
		managedStreams:      make(map[protocol.DropzService_WatchManagedCamerasServer]chan bool),
		syncQueueStreams:    make(map[protocol.DropzService_WatchSyncQueueServer]chan bool),
		lastHeartbeat:       time.Now(),
		heartbeatInterval:   10 * time.Second,
		streamUpdateChannel: make(chan struct{}, 10), // Buffer for update notifications
		ctx:                 ctx,
		cancel:              cancel,
	}
}

// Start starts the gRPC server
func (s *DropzServer) Start(address string) error {
	s.log.Debug("Starting DropzServer", "address", address)

	// Register as the update notifier for the manager
	s.manager.SetNotifier(s)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		s.log.Error("Failed to listen on address", "address", address, "error", err)
		return fmt.Errorf("failed to listen on %s: %v", address, err)
	}

	s.server = grpc.NewServer()
	protocol.RegisterDropzServiceServer(s.server, s)

	s.log.Info("gRPC server started", "address", address)

	s.wg.Add(1) // Increment WaitGroup counter for streamUpdateHandler
	go s.streamUpdateHandler()
	s.log.Trace("streamUpdateHandler goroutine started")

	s.wg.Add(1) // Increment WaitGroup counter for heartbeatSender
	go s.heartbeatSender()
	s.log.Trace("heartbeatSender goroutine started")

	// Goroutine to serve gRPC requests
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

	// 1. Signal internal goroutines to stop by canceling the context
	s.log.Debug("Cancelling server context for internal goroutines", "components", []string{"streamUpdateHandler", "heartbeatSender"})
	s.cancel()

	// 2. Close 'done' channels for Watch... streams to allow them to terminate.
	s.streamMutex.Lock()
	s.log.Debug("Closing stream done channels", "discovered_streams", len(s.discoveredStreams), "managed_streams", len(s.managedStreams), "sync_queue_streams", len(s.syncQueueStreams))
	for stream, ch := range s.discoveredStreams {
		close(ch)
		delete(s.discoveredStreams, stream)
	}
	for stream, ch := range s.managedStreams {
		close(ch)
		delete(s.managedStreams, stream)
	}
	for stream, ch := range s.syncQueueStreams {
		close(ch)
		delete(s.syncQueueStreams, stream)
	}
	s.streamMutex.Unlock()
	s.log.Debug("All Watch stream done channels closed and maps cleared")

	// 3. Stop the gRPC server.
	if s.server != nil {
		s.log.Info("Stopping gRPC server gracefully")
		s.server.GracefulStop()
		s.log.Info("gRPC server stopped")
	} else {
		s.log.Debug("gRPC server was nil, no need to stop")
	}

	// 4. Wait for internal goroutines (streamUpdateHandler, heartbeatSender) to finish.
	s.log.Debug("Waiting for internal server goroutines to stop")
	s.wg.Wait()
	s.log.Info("DropzServer shutdown complete")
}

// streamUpdateHandler sends updates to all active streams when changes occur
func (s *DropzServer) streamUpdateHandler() {
	defer s.wg.Done() // Decrement WaitGroup counter on exit
	defer s.log.Trace("Stream update handler goroutine finished")
	s.log.Trace("Stream update handler started")

	for {
		select {
		case <-s.ctx.Done():
			s.log.Trace("Stream update handler stopping due to context cancellation")
			return
		case _, ok := <-s.streamUpdateChannel:
			if !ok {
				s.log.Debug("Stream update handler stopping because streamUpdateChannel was closed")
				return
			}

			// Check context again before processing to handle race during shutdown
			select {
			case <-s.ctx.Done():
				s.log.Trace("Stream update handler: context done during update processing, aborting")
				return
			default:
			}
			s.log.Trace("Stream update handler processing update")

			// Since NotifyUpdate() now only sends notifications for actual changes,
			// we can directly send updates to all streams without redundant counter checks
			s.log.Trace("Sending updates to all active streams")
			s.sendDiscoveredCamerasUpdates()
			s.sendManagedCamerasUpdates()
			s.sendSyncQueueUpdates()
		}
	}
}

// heartbeatSender periodically sends heartbeats to all active streams
func (s *DropzServer) heartbeatSender() {
	defer s.wg.Done() // Decrement WaitGroup counter on exit
	defer s.log.Trace("Heartbeat sender goroutine finished")
	s.log.Trace("Heartbeat sender started", "interval", s.heartbeatInterval)
	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.log.Trace("Heartbeat sender stopping due to context cancellation")
			return
		case <-ticker.C:
			// Check context again before processing
			select {
			case <-s.ctx.Done():
				s.log.Trace("Heartbeat sender: context done during heartbeat processing, aborting")
				return
			default:
			}
			s.log.Trace("Heartbeat sender: sending heartbeats to all streams")
			s.mutex.Lock()
			s.lastHeartbeat = time.Now()
			s.mutex.Unlock()

			s.sendDiscoveredCamerasHeartbeats()
			s.sendManagedCamerasHeartbeats()
			s.sendSyncQueueHeartbeats()
		}
	}
}

// sendDiscoveredCamerasUpdates sends updates to all discovered cameras streams
func (s *DropzServer) sendDiscoveredCamerasUpdates() {
	db := database.GetDatabase()
	discoveredCameras := db.GetCamerasForDiscoveredPool()

	protoCameras := make([]*protocol.DiscoveredCamera, len(discoveredCameras))
	for i, cam := range discoveredCameras {
		protoCameras[i] = cam.ToProtoDiscoveredCamera()
	}

	response := &protocol.GetDiscoveredCamerasResponse{
		Cameras:   protoCameras,
		NoChanges: false,
		Heartbeat: false,
	}

	s.streamMutex.RLock()
	defer s.streamMutex.RUnlock()

	for stream := range s.discoveredStreams {
		if err := stream.Send(response); err != nil {
			s.log.Error("Failed to send discovered cameras update to stream", "stream_ptr", fmt.Sprintf("%p", stream), "error", err, "camera_count", len(protoCameras))
			// Stream cleanup handled by heartbeat or Watch method exit
		} else {
			s.log.Trace("Sent discovered cameras update to stream", "stream_ptr", fmt.Sprintf("%p", stream), "camera_count", len(protoCameras))
		}
	}
}

// sendManagedCamerasUpdates sends updates to all managed cameras streams
func (s *DropzServer) sendManagedCamerasUpdates() {
	db := database.GetDatabase()
	managedCameras := db.GetCamerasForManagedPool()

	protoCameras := make([]*protocol.ManagedCamera, len(managedCameras))
	for i, cam := range managedCameras {
		protoCameras[i] = cam.ToProtoManagedCamera()
	}

	response := &protocol.GetManagedCamerasResponse{
		Cameras:   protoCameras,
		NoChanges: false,
		Heartbeat: false,
	}

	s.streamMutex.RLock()
	defer s.streamMutex.RUnlock()

	for stream := range s.managedStreams {
		if err := stream.Send(response); err != nil {
			s.log.Errorf("Failed to send managed cameras update to stream %p: %v", stream, err)
		}
	}
}

// sendSyncQueueUpdates sends updates to all sync queue streams
func (s *DropzServer) sendSyncQueueUpdates() {
	db := database.GetDatabase()
	queue := db.GetSyncQueue()

	protoQueue := make([]*protocol.SyncQueueEntry, len(queue))
	for i, entry := range queue {
		protoQueue[i] = entry.ToProtoSyncQueueEntry()
	}

	response := &protocol.GetSyncQueueResponse{
		Queue:     protoQueue,
		NoChanges: false,
		Heartbeat: false,
	}

	s.streamMutex.RLock()
	defer s.streamMutex.RUnlock()

	for stream := range s.syncQueueStreams {
		if err := stream.Send(response); err != nil {
			s.log.Errorf("Failed to send sync queue update to stream %p: %v", stream, err)
		}
	}
}

// sendDiscoveredCamerasHeartbeats sends heartbeats to all discovered cameras streams
func (s *DropzServer) sendDiscoveredCamerasHeartbeats() {
	heartbeat := &protocol.GetDiscoveredCamerasResponse{
		Cameras:   []*protocol.DiscoveredCamera{},
		NoChanges: true,
		Heartbeat: true,
	}

	s.streamMutex.Lock() // Write lock to modify map
	defer s.streamMutex.Unlock()

	for stream, doneCh := range s.discoveredStreams {
		select {
		case <-doneCh: // Stream explicitly closed by Stop()
			delete(s.discoveredStreams, stream)
			s.log.Debugf("Cleaned up discovered stream %p (closed by Stop)", stream)
			continue
		default:
		}
		if err := stream.Send(heartbeat); err != nil {
			s.log.Errorf("Failed to send heartbeat to discovered stream %p, closing: %v", stream, err)
			close(doneCh) // Close its done channel
			delete(s.discoveredStreams, stream)
		}
	}
}

// sendManagedCamerasHeartbeats sends heartbeats to all managed cameras streams
func (s *DropzServer) sendManagedCamerasHeartbeats() {
	heartbeat := &protocol.GetManagedCamerasResponse{
		Cameras:   []*protocol.ManagedCamera{},
		NoChanges: true,
		Heartbeat: true,
	}

	s.streamMutex.Lock() // Write lock to modify map
	defer s.streamMutex.Unlock()

	for stream, doneCh := range s.managedStreams {
		select {
		case <-doneCh:
			delete(s.managedStreams, stream)
			s.log.Debugf("Cleaned up managed stream %p (closed by Stop)", stream)
			continue
		default:
		}
		if err := stream.Send(heartbeat); err != nil {
			s.log.Errorf("Failed to send heartbeat to managed stream %p, closing: %v", stream, err)
			close(doneCh)
			delete(s.managedStreams, stream)
		}
	}
}

// sendSyncQueueHeartbeats sends heartbeats to all sync queue streams
func (s *DropzServer) sendSyncQueueHeartbeats() {
	heartbeat := &protocol.GetSyncQueueResponse{
		Queue:     []*protocol.SyncQueueEntry{},
		NoChanges: true,
		Heartbeat: true,
	}

	s.streamMutex.Lock() // Write lock to modify map
	defer s.streamMutex.Unlock()

	for stream, doneCh := range s.syncQueueStreams {
		select {
		case <-doneCh:
			delete(s.syncQueueStreams, stream)
			s.log.Debugf("Cleaned up sync queue stream %p (closed by Stop)", stream)
			continue
		default:
		}
		if err := stream.Send(heartbeat); err != nil {
			s.log.Errorf("Failed to send heartbeat to sync queue stream %p, closing: %v", stream, err)
			close(doneCh)
			delete(s.syncQueueStreams, stream)
		}
	}
}

// NotifyUpdate implements the common.UpdateNotifier interface
func (s *DropzServer) NotifyUpdate() {
	s.log.Trace("NotifyUpdate called - sending update to clients")

	// Send notification to stream handler
	select {
	case s.streamUpdateChannel <- struct{}{}:
		// Notification sent
	default:
		// Channel is full, but that's okay since the update will be handled by the next notification
		s.log.Debug("streamUpdateChannel is full, skipped sending notification")
	}
}

// GetDiscoveredCameras implements the GetDiscoveredCameras RPC method
func (s *DropzServer) GetDiscoveredCameras(ctx context.Context, req *protocol.GetDiscoveredCamerasRequest) (*protocol.GetDiscoveredCamerasResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Trace("Handling GetDiscoveredCameras request")

	db := database.GetDatabase()

	// Get all discovered cameras (following the same filters as the stream)
	discoveredCameras := db.GetCamerasForDiscoveredPool()

	protoCameras := make([]*protocol.DiscoveredCamera, len(discoveredCameras))
	for i, cam := range discoveredCameras {
		protoCameras[i] = cam.ToProtoDiscoveredCamera()
	}

	s.log.Trace("Returning discovered cameras", "camera_count", len(protoCameras))
	return &protocol.GetDiscoveredCamerasResponse{
		Cameras:   protoCameras,
		NoChanges: false,
		Heartbeat: false,
	}, nil
}

// WatchDiscoveredCameras implements the streaming RPC to watch for discovered camera changes
func (s *DropzServer) WatchDiscoveredCameras(req *protocol.GetDiscoveredCamerasRequest, stream protocol.DropzService_WatchDiscoveredCamerasServer) error {
	s.log.Debugf("WatchDiscoveredCameras: New stream request from %p", stream)
	done := make(chan bool)

	s.streamMutex.Lock()
	select {
	case <-s.ctx.Done():
		s.streamMutex.Unlock()
		s.log.Debugf("WatchDiscoveredCameras: Server shutting down, stream %p not started.", stream)
		return s.ctx.Err()
	default:
		s.discoveredStreams[stream] = done
		s.log.Debugf("WatchDiscoveredCameras: Stream %p added to discoveredStreams.", stream)
	}
	s.streamMutex.Unlock()

	defer func() {
		s.streamMutex.Lock()
		if currentDone, ok := s.discoveredStreams[stream]; ok {
			if currentDone == done { // Ensure we are deleting the correct stream instance's channel
				delete(s.discoveredStreams, stream)
				s.log.Debugf("WatchDiscoveredCameras: Stream %p removed from discoveredStreams.", stream)
			}
		}
		// Note: 'done' channel is closed by Stop() or by heartbeat sender on error.
		// Avoid closing 'done' here directly unless logic guarantees it's safe and not already closed.
		s.streamMutex.Unlock()
		s.log.Debugf("WatchDiscoveredCameras: Stream %p defer cleanup finished.", stream)
	}()

	s.log.Debugf("WatchDiscoveredCameras: Sending initial data to stream %p.", stream)
	initialResponse, err := s.GetDiscoveredCameras(stream.Context(), req)
	if err != nil {
		s.log.Errorf("WatchDiscoveredCameras: Error getting initial discovered cameras for stream %p: %v", stream, err)
		return err
	}
	if err := stream.Send(initialResponse); err != nil {
		s.log.Errorf("WatchDiscoveredCameras: Error sending initial discovered cameras for stream %p: %v", stream, err)
		return err
	}
	s.log.Tracef("WatchDiscoveredCameras: Initial data sent to stream %p.", stream)

	select {
	case <-done:
		s.log.Debugf("WatchDiscoveredCameras: Stream %p stopping via internal 'done' channel.", stream)
	case <-stream.Context().Done():
		s.log.Debugf("WatchDiscoveredCameras: Stream %p stopping via RPC context done: %v", stream, stream.Context().Err())
	case <-s.ctx.Done(): // Also listen for server shutdown context
		s.log.Debugf("WatchDiscoveredCameras: Stream %p stopping via server context done.", stream)
	}
	s.log.Debugf("WatchDiscoveredCameras: Stream %p finished.", stream)
	return nil
}

// GetManagedCameras implements the GetManagedCameras RPC method
func (s *DropzServer) GetManagedCameras(ctx context.Context, req *protocol.GetManagedCamerasRequest) (*protocol.GetManagedCamerasResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Trace("Handling GetManagedCameras request")

	db := database.GetDatabase()

	// Get all managed cameras
	managedCameras := db.GetCamerasForManagedPool()

	protoCameras := make([]*protocol.ManagedCamera, len(managedCameras))
	for i, camera := range managedCameras {
		protoCameras[i] = camera.ToProtoManagedCamera()
	}

	return &protocol.GetManagedCamerasResponse{
		Cameras:   protoCameras,
		NoChanges: false,
		Heartbeat: false,
	}, nil
}

// WatchManagedCameras implements the streaming RPC to watch for managed camera changes
func (s *DropzServer) WatchManagedCameras(req *protocol.GetManagedCamerasRequest, stream protocol.DropzService_WatchManagedCamerasServer) error {
	s.log.Debugf("WatchManagedCameras: New stream request from %p", stream)
	done := make(chan bool)
	s.streamMutex.Lock()
	select {
	case <-s.ctx.Done():
		s.streamMutex.Unlock()
		s.log.Debugf("WatchManagedCameras: Server shutting down, stream %p not started.", stream)
		return s.ctx.Err()
	default:
		s.managedStreams[stream] = done
		s.log.Debugf("WatchManagedCameras: Stream %p added to managedStreams.", stream)
	}
	s.streamMutex.Unlock()

	defer func() {
		s.streamMutex.Lock()
		if currentDone, ok := s.managedStreams[stream]; ok {
			if currentDone == done {
				delete(s.managedStreams, stream)
				s.log.Debugf("WatchManagedCameras: Stream %p removed from managedStreams.", stream)
			}
		}
		s.streamMutex.Unlock()
		s.log.Debugf("WatchManagedCameras: Stream %p defer cleanup finished.", stream)
	}()

	s.log.Debugf("WatchManagedCameras: Sending initial data to stream %p.", stream)
	initialResponse, err := s.GetManagedCameras(stream.Context(), req)
	if err != nil {
		s.log.Errorf("WatchManagedCameras: Error getting initial managed cameras for stream %p: %v", stream, err)
		return err
	}
	if err := stream.Send(initialResponse); err != nil {
		s.log.Errorf("WatchManagedCameras: Error sending initial managed cameras for stream %p: %v", stream, err)
		return err
	}
	s.log.Tracef("WatchManagedCameras: Initial data sent to stream %p.", stream)

	select {
	case <-done:
		s.log.Debugf("WatchManagedCameras: Stream %p stopping via internal 'done' channel.", stream)
	case <-stream.Context().Done():
		s.log.Debugf("WatchManagedCameras: Stream %p stopping via RPC context done: %v", stream, stream.Context().Err())
	case <-s.ctx.Done():
		s.log.Debugf("WatchManagedCameras: Stream %p stopping via server context done.", stream)
	}
	s.log.Debugf("WatchManagedCameras: Stream %p finished.", stream)
	return nil
}

// GetSyncQueue implements the GetSyncQueue RPC method
func (s *DropzServer) GetSyncQueue(ctx context.Context, req *protocol.GetSyncQueueRequest) (*protocol.GetSyncQueueResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Debug("Handling GetSyncQueue request")

	db := database.GetDatabase()

	// Get all sync queue entries
	queue := db.GetSyncQueue()

	protoQueue := make([]*protocol.SyncQueueEntry, len(queue))
	for i, entry := range queue {
		protoQueue[i] = entry.ToProtoSyncQueueEntry()
	}

	return &protocol.GetSyncQueueResponse{
		Queue:     protoQueue,
		NoChanges: false,
		Heartbeat: false,
	}, nil
}

// WatchSyncQueue implements the streaming RPC to watch for sync queue changes
func (s *DropzServer) WatchSyncQueue(req *protocol.GetSyncQueueRequest, stream protocol.DropzService_WatchSyncQueueServer) error {
	s.log.Debugf("WatchSyncQueue: New stream request from %p", stream)
	done := make(chan bool)
	s.streamMutex.Lock()
	select {
	case <-s.ctx.Done():
		s.streamMutex.Unlock()
		s.log.Debugf("WatchSyncQueue: Server shutting down, stream %p not started.", stream)
		return s.ctx.Err()
	default:
		s.syncQueueStreams[stream] = done
		s.log.Debugf("WatchSyncQueue: Stream %p added to syncQueueStreams.", stream)
	}
	s.streamMutex.Unlock()

	defer func() {
		s.streamMutex.Lock()
		if currentDone, ok := s.syncQueueStreams[stream]; ok {
			if currentDone == done {
				delete(s.syncQueueStreams, stream)
				s.log.Debugf("WatchSyncQueue: Stream %p removed from syncQueueStreams.", stream)
			}
		}
		s.streamMutex.Unlock()
		s.log.Debugf("WatchSyncQueue: Stream %p defer cleanup finished.", stream)
	}()

	s.log.Debugf("WatchSyncQueue: Sending initial data to stream %p.", stream)
	initialResponse, err := s.GetSyncQueue(stream.Context(), req)
	if err != nil {
		s.log.Errorf("WatchSyncQueue: Error getting initial sync queue for stream %p: %v", stream, err)
		return err
	}
	if err := stream.Send(initialResponse); err != nil {
		s.log.Errorf("WatchSyncQueue: Error sending initial sync queue for stream %p: %v", stream, err)
		return err
	}
	s.log.Debugf("WatchSyncQueue: Initial data sent to stream %p.", stream)

	select {
	case <-done:
		s.log.Debugf("WatchSyncQueue: Stream %p stopping via internal 'done' channel.", stream)
	case <-stream.Context().Done():
		s.log.Debugf("WatchSyncQueue: Stream %p stopping via RPC context done: %v", stream, stream.Context().Err())
	case <-s.ctx.Done():
		s.log.Debugf("WatchSyncQueue: Stream %p stopping via server context done.", stream)
	}
	s.log.Debugf("WatchSyncQueue: Stream %p finished.", stream)
	return nil
}

// ManageCamera implements the ManageCamera RPC method
func (s *DropzServer) ManageCamera(ctx context.Context, req *protocol.ManageCameraRequest) (*protocol.ManageCameraResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.ManageCameraResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}
	s.log.Info("Handling ManageCamera request", "camera_id", req.CameraId)

	managedCamera, err := s.manager.ManageCamera(req.CameraId)
	if err != nil {
		s.log.Error("Failed to manage camera", "camera_id", req.CameraId, "error", err)
		return &protocol.ManageCameraResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if managedCamera == nil {
		s.log.Error("ManageCamera returned nil managed camera", "camera_id", req.CameraId)
		return &protocol.ManageCameraResponse{
			Success: false,
			Message: "Failed to manage camera: camera not found or not eligible for management",
		}, nil
	}

	// Notify about updates
	s.NotifyUpdate()

	s.log.Info("Camera managed successfully", "camera_id", req.CameraId, "camera_name", managedCamera.CameraState.Camera.Name)
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
	s.log.Debugf("Handling UnmanageCamera request for camera %s", req.CameraId)

	err := s.manager.UnmanageCamera(req.CameraId)
	if err != nil {
		return &protocol.UnmanageCameraResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Notify about updates
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
	s.log.Debugf("Handling PairCamera request for camera %s", req.CameraId)

	managedCamera, err := s.manager.PairCamera(req.CameraId)
	if err != nil {
		return &protocol.PairCameraResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if managedCamera == nil {
		s.log.Error("PairCamera returned nil managed camera", "camera_id", req.CameraId)
		return &protocol.PairCameraResponse{
			Success: false,
			Message: "Failed to pair camera: camera not found or not eligible for pairing",
		}, nil
	}

	// Notify about updates
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
	s.log.Debugf("Handling ForceSync request for camera %s", req.CameraId)

	queueEntry, err := s.manager.ForceSync(req.CameraId)
	if err != nil {
		return &protocol.ForceSyncResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Notify about updates
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
	s.log.Debugf("Handling CancelSync request for camera %s", req.CameraId)

	err := s.manager.CancelSync(req.CameraId)
	if err != nil {
		return &protocol.CancelSyncResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Notify about updates
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
	s.log.Debug("Handling GetGroups request")

	db := database.GetDatabase()
	groups := db.GetAllGroups()

	protoGroups := make([]*protocol.Group, len(groups))
	for i, group := range groups {
		protoGroups[i] = group.ToProtoGroup()
	}

	return &protocol.GetGroupsResponse{
		Groups: protoGroups,
	}, nil
}

// CreateGroup implements the CreateGroup RPC method
func (s *DropzServer) CreateGroup(ctx context.Context, req *protocol.CreateGroupRequest) (*protocol.Group, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Debugf("Handling CreateGroup request for group %s", req.Name)

	group, err := s.manager.CreateGroup(req.Name, req.CameraIds)
	if err != nil {
		return nil, err
	}

	// Notify about updates
	s.NotifyUpdate()

	return group.ToProtoGroup(), nil
}

// UpdateGroup implements the UpdateGroup RPC method
func (s *DropzServer) UpdateGroup(ctx context.Context, req *protocol.UpdateGroupRequest) (*protocol.Group, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Debugf("Handling UpdateGroup request for group %s", req.GroupId)

	group, err := s.manager.UpdateGroup(req.GroupId, req.Name, req.CameraIds)
	if err != nil {
		return nil, err
	}

	// Notify about updates
	s.NotifyUpdate()

	return group.ToProtoGroup(), nil
}

// DeleteGroup implements the DeleteGroup RPC method
func (s *DropzServer) DeleteGroup(ctx context.Context, req *protocol.DeleteGroupRequest) (*protocol.OperationResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.OperationResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}
	s.log.Debugf("Handling DeleteGroup request for group %s", req.GroupId)

	err := s.manager.DeleteGroup(req.GroupId)
	if err != nil {
		return &protocol.OperationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Notify about updates
	s.NotifyUpdate()

	return &protocol.OperationResponse{
		Success: true,
		Message: fmt.Sprintf("Group %s deleted successfully", req.GroupId),
	}, nil
}

// GetVideos implements the GetVideos RPC method
func (s *DropzServer) GetVideos(ctx context.Context, req *protocol.GetVideosRequest) (*protocol.GetVideosResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Debug("Handling GetVideos request")

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

// GetConfig implements the GetConfig RPC method
func (s *DropzServer) GetConfig(ctx context.Context, req *protocol.GetConfigRequest) (*protocol.GetConfigResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Debug("Handling GetConfig request")

	config := s.manager.GetConfig()

	return &protocol.GetConfigResponse{
		Config: config.ToProtoConfig(),
	}, nil
}

// UpdateConfig implements the UpdateConfig RPC method
func (s *DropzServer) UpdateConfig(ctx context.Context, req *protocol.UpdateConfigRequest) (*protocol.UpdateConfigResponse, error) {
	if s.ctx.Err() != nil {
		return &protocol.UpdateConfigResponse{Success: false, Message: fmt.Sprintf("server is shutting down: %s", s.ctx.Err().Error())}, nil
	}
	s.log.Debug("Handling UpdateConfig request")

	if req.Config == nil {
		return &protocol.UpdateConfigResponse{
			Success: false,
			Message: "No config provided",
		}, nil
	}

	// Convert proto config to database config
	dbConfig := database.Config{
		SyncEnabled:                   req.Config.SyncEnabled,
		ScanIntervalSeconds:           req.Config.ScanIntervalSeconds,
		ConnectTimeoutSeconds:         req.Config.ConnectTimeoutSeconds,
		DaysThreshold:                 req.Config.DaysThreshold,
		DestinationFolder:             req.Config.DestinationFolder,
		InactivityTimeoutSeconds:      req.Config.InactivityTimeoutSeconds,
		InactivitySyncIntervalSeconds: req.Config.InactivitySyncIntervalSeconds,
		SetTimeEnabled:                req.Config.SetTimeEnabled,
		LogLevel:                      req.Config.LogLevel,
		LastUpdated:                   req.Config.LastUpdated.AsTime(),
	}

	err := s.manager.UpdateConfig(dbConfig)
	if err != nil {
		return &protocol.UpdateConfigResponse{
			Success: false,
			Message: err.Error(),
		}, nil
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
	s.log.Debugf("Handling GetSetting request for setting %s", req.SettingName)

	value, err := s.manager.GetSetting(req.SettingName)
	if err != nil {
		return nil, err
	}

	response := &protocol.GetSettingResponse{}

	// Set the appropriate value based on the type
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
	s.log.Debugf("Handling UpdateSetting request for setting %s", req.SettingName)

	var value interface{}

	// Extract the value based on the oneof field
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

	// Notify about updates if needed
	s.NotifyUpdate()

	return config.ToProtoConfig(), nil
}

// ResetSetting implements the ResetSetting RPC method
func (s *DropzServer) ResetSetting(ctx context.Context, req *protocol.ResetSettingRequest) (*protocol.Config, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Debugf("Handling ResetSetting request for setting %s", req.SettingName)

	config, err := s.manager.ResetSetting(req.SettingName)
	if err != nil {
		return nil, err
	}

	// Notify about updates if needed
	s.NotifyUpdate()

	return config.ToProtoConfig(), nil
}

// GetLogs implements the GetLogs RPC method
func (s *DropzServer) GetLogs(ctx context.Context, req *protocol.GetLogsRequest) (*protocol.GetLogsResponse, error) {
	if s.ctx.Err() != nil {
		return nil, fmt.Errorf("server is shutting down: %w", s.ctx.Err())
	}
	s.log.Debug("Handling GetLogs request")

	db := database.GetDatabase()

	var startTime, endTime time.Time
	if req.StartTime != nil {
		startTime = req.StartTime.AsTime()
	}
	if req.EndTime != nil {
		endTime = req.EndTime.AsTime()
	}

	logs, totalCount := db.GetLogs(req.Level, req.CameraId, req.Component, startTime, endTime, int(req.Limit), int(req.Offset))

	protoLogs := make([]*protocol.LogEntry, len(logs))
	for i, logEntry := range logs {
		protoLogs[i] = logEntry.ToProtoLogEntry()
	}

	return &protocol.GetLogsResponse{
		Logs:       protoLogs,
		TotalCount: int32(totalCount),
	}, nil
}
