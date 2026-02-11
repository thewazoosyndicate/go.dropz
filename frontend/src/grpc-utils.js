// gRPC client initialization utility
const grpc = require('@grpc/grpc-js');

// Import generated service definitions
const serviceGrpc = require('./proto/service_grpc_pb');

// Import generated proto message definitions
const goProProto = require('./proto/gopro_pb');
const videoProto = require('./proto/video_pb');
const configProto = require('./proto/config_pb');
const commonProto = require('./proto/common_pb');

// Singleton instance
let _client = null;

// Get gRPC client with lazy initialization
function getClient() {
  if (!_client) {
    // Create client with insecure credentials (since this is local communication)
    // The port (50051) must match what's used in your Go backend
    _client = new serviceGrpc.DropzServiceClient(
      '127.0.0.1:50051',
      grpc.credentials.createInsecure()
    );
  }
  return _client;
}

// Export the client getter and all the proto message constructors for convenience
module.exports = {
  getClient,
  
  // Proto messages for Camera operations
  Camera: goProProto.Camera,
  CameraMetadata: goProProto.CameraMetadata,
  DiscoveredCamera: goProProto.DiscoveredCamera,
  ManagedCamera: goProProto.ManagedCamera,
  SyncQueueEntry: goProProto.SyncQueueEntry,
  Group: goProProto.Group,
  
  // Request/Response types for Camera operations
  GetDiscoveredCamerasRequest: goProProto.GetDiscoveredCamerasRequest,
  GetDiscoveredCamerasResponse: goProProto.GetDiscoveredCamerasResponse,
  GetManagedCamerasRequest: goProProto.GetManagedCamerasRequest,
  GetManagedCamerasResponse: goProProto.GetManagedCamerasResponse,
  ManageCameraRequest: goProProto.ManageCameraRequest,
  ManageCameraResponse: goProProto.ManageCameraResponse,
  UnmanageCameraRequest: goProProto.UnmanageCameraRequest,
  UnmanageCameraResponse: goProProto.UnmanageCameraResponse,
  PairCameraRequest: goProProto.PairCameraRequest,
  PairCameraResponse: goProProto.PairCameraResponse,
  
  // Request/Response types for Sync operations
  GetSyncQueueRequest: goProProto.GetSyncQueueRequest,
  GetSyncQueueResponse: goProProto.GetSyncQueueResponse,
  ForceSyncRequest: goProProto.ForceSyncRequest,
  ForceSyncResponse: goProProto.ForceSyncResponse,
  CancelSyncRequest: goProProto.CancelSyncRequest,
  CancelSyncResponse: goProProto.CancelSyncResponse,
  
  // Request/Response types for Group operations
  GetGroupsRequest: goProProto.GetGroupsRequest,
  GetGroupsResponse: goProProto.GetGroupsResponse,
  CreateGroupRequest: goProProto.CreateGroupRequest,
  UpdateGroupRequest: goProProto.UpdateGroupRequest,
  DeleteGroupRequest: goProProto.DeleteGroupRequest,
  
  // Request/Response types for Video operations
  GetVideosRequest: videoProto.GetVideosRequest,
  GetVideosResponse: videoProto.GetVideosResponse,
  
  // Request/Response types for Config operations
  GetConfigRequest: configProto.GetConfigRequest,
  GetConfigResponse: configProto.GetConfigResponse,
  UpdateConfigRequest: configProto.UpdateConfigRequest,
  UpdateConfigResponse: configProto.UpdateConfigResponse,
  GetSettingRequest: configProto.GetSettingRequest,
  GetSettingResponse: configProto.GetSettingResponse,
  UpdateSettingRequest: configProto.UpdateSettingRequest,
  ResetSettingRequest: configProto.ResetSettingRequest,
  
  // Common types
  CameraStatus: commonProto.CameraStatus,
  OperationResponse: commonProto.OperationResponse
};
