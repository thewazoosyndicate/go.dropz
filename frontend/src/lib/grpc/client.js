// gRPC client singleton — uses window.require to bypass Vite bundling
const path = window.require('path');
const appRoot = window.__appRoot;

function appRequire(mod) {
  if (mod.startsWith('.')) return window.require(path.join(appRoot, mod));
  return window.require(mod);
}

const grpc = appRequire('@grpc/grpc-js');
const serviceGrpc = appRequire('./src/proto/service_grpc_pb');
const goProProto = appRequire('./src/proto/gopro_pb');
const configProto = appRequire('./src/proto/config_pb');
const videoProto = appRequire('./src/proto/video_pb');

let _client = null;

// Mirrors the backend's -server-addr flag so a second instance (bench
// harness, screenshot driver) can run beside the real app.
const address = window.process?.env?.DROPZ_GRPC_ADDR || '127.0.0.1:50051';

export function getClient() {
  if (!_client) {
    _client = new serviceGrpc.DropzServiceClient(address, grpc.credentials.createInsecure());
  }
  return _client;
}

export const proto = {
  GetDiscoveredCamerasRequest: goProProto.GetDiscoveredCamerasRequest,
  GetManagedCamerasRequest: goProProto.GetManagedCamerasRequest,
  GetSyncQueueRequest: goProProto.GetSyncQueueRequest,
  PairCameraRequest: goProProto.PairCameraRequest,
  ForceSyncRequest: goProProto.ForceSyncRequest,
  CancelSyncRequest: goProProto.CancelSyncRequest,
  GetSyncHistoryRequest: goProProto.GetSyncHistoryRequest,
  SetCameraAliasRequest: goProProto.SetCameraAliasRequest,
  ManageCameraRequest: goProProto.ManageCameraRequest,
  UnmanageCameraRequest: goProProto.UnmanageCameraRequest,
  GetConfigRequest: configProto.GetConfigRequest,
  UpdateSettingRequest: configProto.UpdateSettingRequest,
  ResetSettingRequest: configProto.ResetSettingRequest,
  GetVideosRequest: videoProto.GetVideosRequest,
  GetGroupsRequest: goProProto.GetGroupsRequest,
  CreateGroupRequest: goProProto.CreateGroupRequest,
  UpdateGroupRequest: goProProto.UpdateGroupRequest,
  DeleteGroupRequest: goProProto.DeleteGroupRequest,
  LoadGroupRequest: goProProto.LoadGroupRequest,
  SaveManagedAsGroupRequest: goProProto.SaveManagedAsGroupRequest,
  GetCameraSettingsRequest: goProProto.GetCameraSettingsRequest,
  ApplyCameraSettingsRequest: goProProto.ApplyCameraSettingsRequest,
  SettingChange: goProProto.SettingChange,
  GetCameraMediaRequest: goProProto.GetCameraMediaRequest,
  RequestMediaDownloadRequest: goProProto.RequestMediaDownloadRequest,
  PreviewMediaRequest: goProProto.PreviewMediaRequest,
  PreviewVideoRequest: goProProto.PreviewVideoRequest,
  SetPreviewSessionRequest: goProProto.SetPreviewSessionRequest,
};
