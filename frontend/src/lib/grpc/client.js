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

let _client = null;

export function getClient() {
  if (!_client) {
    _client = new serviceGrpc.DropzServiceClient(
      '127.0.0.1:50051',
      grpc.credentials.createInsecure()
    );
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
  ManageCameraRequest: goProProto.ManageCameraRequest,
  UnmanageCameraRequest: goProProto.UnmanageCameraRequest,
  GetConfigRequest: configProto.GetConfigRequest,
  UpdateSettingRequest: configProto.UpdateSettingRequest,
  ResetSettingRequest: configProto.ResetSettingRequest,
};
