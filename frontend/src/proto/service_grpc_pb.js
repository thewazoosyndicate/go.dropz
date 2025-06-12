// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var gopro_pb = require('./gopro_pb.js');
var video_pb = require('./video_pb.js');
var logs_pb = require('./logs_pb.js');
var config_pb = require('./config_pb.js');
var common_pb = require('./common_pb.js');

function serialize_dropz_CancelSyncRequest(arg) {
  if (!(arg instanceof gopro_pb.CancelSyncRequest)) {
    throw new Error('Expected argument of type dropz.CancelSyncRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_CancelSyncRequest(buffer_arg) {
  return gopro_pb.CancelSyncRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_CancelSyncResponse(arg) {
  if (!(arg instanceof gopro_pb.CancelSyncResponse)) {
    throw new Error('Expected argument of type dropz.CancelSyncResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_CancelSyncResponse(buffer_arg) {
  return gopro_pb.CancelSyncResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_Config(arg) {
  if (!(arg instanceof config_pb.Config)) {
    throw new Error('Expected argument of type dropz.Config');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_Config(buffer_arg) {
  return config_pb.Config.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_CreateGroupRequest(arg) {
  if (!(arg instanceof gopro_pb.CreateGroupRequest)) {
    throw new Error('Expected argument of type dropz.CreateGroupRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_CreateGroupRequest(buffer_arg) {
  return gopro_pb.CreateGroupRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_DeleteGroupRequest(arg) {
  if (!(arg instanceof gopro_pb.DeleteGroupRequest)) {
    throw new Error('Expected argument of type dropz.DeleteGroupRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_DeleteGroupRequest(buffer_arg) {
  return gopro_pb.DeleteGroupRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_ForceSyncRequest(arg) {
  if (!(arg instanceof gopro_pb.ForceSyncRequest)) {
    throw new Error('Expected argument of type dropz.ForceSyncRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_ForceSyncRequest(buffer_arg) {
  return gopro_pb.ForceSyncRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_ForceSyncResponse(arg) {
  if (!(arg instanceof gopro_pb.ForceSyncResponse)) {
    throw new Error('Expected argument of type dropz.ForceSyncResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_ForceSyncResponse(buffer_arg) {
  return gopro_pb.ForceSyncResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetConfigRequest(arg) {
  if (!(arg instanceof config_pb.GetConfigRequest)) {
    throw new Error('Expected argument of type dropz.GetConfigRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetConfigRequest(buffer_arg) {
  return config_pb.GetConfigRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetConfigResponse(arg) {
  if (!(arg instanceof config_pb.GetConfigResponse)) {
    throw new Error('Expected argument of type dropz.GetConfigResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetConfigResponse(buffer_arg) {
  return config_pb.GetConfigResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetDiscoveredCamerasRequest(arg) {
  if (!(arg instanceof gopro_pb.GetDiscoveredCamerasRequest)) {
    throw new Error('Expected argument of type dropz.GetDiscoveredCamerasRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetDiscoveredCamerasRequest(buffer_arg) {
  return gopro_pb.GetDiscoveredCamerasRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetDiscoveredCamerasResponse(arg) {
  if (!(arg instanceof gopro_pb.GetDiscoveredCamerasResponse)) {
    throw new Error('Expected argument of type dropz.GetDiscoveredCamerasResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetDiscoveredCamerasResponse(buffer_arg) {
  return gopro_pb.GetDiscoveredCamerasResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetGroupsRequest(arg) {
  if (!(arg instanceof gopro_pb.GetGroupsRequest)) {
    throw new Error('Expected argument of type dropz.GetGroupsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetGroupsRequest(buffer_arg) {
  return gopro_pb.GetGroupsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetGroupsResponse(arg) {
  if (!(arg instanceof gopro_pb.GetGroupsResponse)) {
    throw new Error('Expected argument of type dropz.GetGroupsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetGroupsResponse(buffer_arg) {
  return gopro_pb.GetGroupsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetLogsRequest(arg) {
  if (!(arg instanceof logs_pb.GetLogsRequest)) {
    throw new Error('Expected argument of type dropz.GetLogsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetLogsRequest(buffer_arg) {
  return logs_pb.GetLogsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetLogsResponse(arg) {
  if (!(arg instanceof logs_pb.GetLogsResponse)) {
    throw new Error('Expected argument of type dropz.GetLogsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetLogsResponse(buffer_arg) {
  return logs_pb.GetLogsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetManagedCamerasRequest(arg) {
  if (!(arg instanceof gopro_pb.GetManagedCamerasRequest)) {
    throw new Error('Expected argument of type dropz.GetManagedCamerasRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetManagedCamerasRequest(buffer_arg) {
  return gopro_pb.GetManagedCamerasRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetManagedCamerasResponse(arg) {
  if (!(arg instanceof gopro_pb.GetManagedCamerasResponse)) {
    throw new Error('Expected argument of type dropz.GetManagedCamerasResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetManagedCamerasResponse(buffer_arg) {
  return gopro_pb.GetManagedCamerasResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetSettingRequest(arg) {
  if (!(arg instanceof config_pb.GetSettingRequest)) {
    throw new Error('Expected argument of type dropz.GetSettingRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetSettingRequest(buffer_arg) {
  return config_pb.GetSettingRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetSettingResponse(arg) {
  if (!(arg instanceof config_pb.GetSettingResponse)) {
    throw new Error('Expected argument of type dropz.GetSettingResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetSettingResponse(buffer_arg) {
  return config_pb.GetSettingResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetSyncQueueRequest(arg) {
  if (!(arg instanceof gopro_pb.GetSyncQueueRequest)) {
    throw new Error('Expected argument of type dropz.GetSyncQueueRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetSyncQueueRequest(buffer_arg) {
  return gopro_pb.GetSyncQueueRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetSyncQueueResponse(arg) {
  if (!(arg instanceof gopro_pb.GetSyncQueueResponse)) {
    throw new Error('Expected argument of type dropz.GetSyncQueueResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetSyncQueueResponse(buffer_arg) {
  return gopro_pb.GetSyncQueueResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetVideosRequest(arg) {
  if (!(arg instanceof video_pb.GetVideosRequest)) {
    throw new Error('Expected argument of type dropz.GetVideosRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetVideosRequest(buffer_arg) {
  return video_pb.GetVideosRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_GetVideosResponse(arg) {
  if (!(arg instanceof video_pb.GetVideosResponse)) {
    throw new Error('Expected argument of type dropz.GetVideosResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_GetVideosResponse(buffer_arg) {
  return video_pb.GetVideosResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_Group(arg) {
  if (!(arg instanceof gopro_pb.Group)) {
    throw new Error('Expected argument of type dropz.Group');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_Group(buffer_arg) {
  return gopro_pb.Group.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_ManageCameraRequest(arg) {
  if (!(arg instanceof gopro_pb.ManageCameraRequest)) {
    throw new Error('Expected argument of type dropz.ManageCameraRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_ManageCameraRequest(buffer_arg) {
  return gopro_pb.ManageCameraRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_ManageCameraResponse(arg) {
  if (!(arg instanceof gopro_pb.ManageCameraResponse)) {
    throw new Error('Expected argument of type dropz.ManageCameraResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_ManageCameraResponse(buffer_arg) {
  return gopro_pb.ManageCameraResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_OperationResponse(arg) {
  if (!(arg instanceof common_pb.OperationResponse)) {
    throw new Error('Expected argument of type dropz.OperationResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_OperationResponse(buffer_arg) {
  return common_pb.OperationResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_PairCameraRequest(arg) {
  if (!(arg instanceof gopro_pb.PairCameraRequest)) {
    throw new Error('Expected argument of type dropz.PairCameraRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_PairCameraRequest(buffer_arg) {
  return gopro_pb.PairCameraRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_PairCameraResponse(arg) {
  if (!(arg instanceof gopro_pb.PairCameraResponse)) {
    throw new Error('Expected argument of type dropz.PairCameraResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_PairCameraResponse(buffer_arg) {
  return gopro_pb.PairCameraResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_ResetSettingRequest(arg) {
  if (!(arg instanceof config_pb.ResetSettingRequest)) {
    throw new Error('Expected argument of type dropz.ResetSettingRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_ResetSettingRequest(buffer_arg) {
  return config_pb.ResetSettingRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_UnmanageCameraRequest(arg) {
  if (!(arg instanceof gopro_pb.UnmanageCameraRequest)) {
    throw new Error('Expected argument of type dropz.UnmanageCameraRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_UnmanageCameraRequest(buffer_arg) {
  return gopro_pb.UnmanageCameraRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_UnmanageCameraResponse(arg) {
  if (!(arg instanceof gopro_pb.UnmanageCameraResponse)) {
    throw new Error('Expected argument of type dropz.UnmanageCameraResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_UnmanageCameraResponse(buffer_arg) {
  return gopro_pb.UnmanageCameraResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_UpdateConfigRequest(arg) {
  if (!(arg instanceof config_pb.UpdateConfigRequest)) {
    throw new Error('Expected argument of type dropz.UpdateConfigRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_UpdateConfigRequest(buffer_arg) {
  return config_pb.UpdateConfigRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_UpdateConfigResponse(arg) {
  if (!(arg instanceof config_pb.UpdateConfigResponse)) {
    throw new Error('Expected argument of type dropz.UpdateConfigResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_UpdateConfigResponse(buffer_arg) {
  return config_pb.UpdateConfigResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_UpdateGroupRequest(arg) {
  if (!(arg instanceof gopro_pb.UpdateGroupRequest)) {
    throw new Error('Expected argument of type dropz.UpdateGroupRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_UpdateGroupRequest(buffer_arg) {
  return gopro_pb.UpdateGroupRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_dropz_UpdateSettingRequest(arg) {
  if (!(arg instanceof config_pb.UpdateSettingRequest)) {
    throw new Error('Expected argument of type dropz.UpdateSettingRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_dropz_UpdateSettingRequest(buffer_arg) {
  return config_pb.UpdateSettingRequest.deserializeBinary(new Uint8Array(buffer_arg));
}


// Service definition for communication between frontend and backend
var DropzServiceService = exports.DropzServiceService = {
  // Camera discovery and management
getDiscoveredCameras: {
    path: '/dropz.DropzService/GetDiscoveredCameras',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.GetDiscoveredCamerasRequest,
    responseType: gopro_pb.GetDiscoveredCamerasResponse,
    requestSerialize: serialize_dropz_GetDiscoveredCamerasRequest,
    requestDeserialize: deserialize_dropz_GetDiscoveredCamerasRequest,
    responseSerialize: serialize_dropz_GetDiscoveredCamerasResponse,
    responseDeserialize: deserialize_dropz_GetDiscoveredCamerasResponse,
  },
  watchDiscoveredCameras: {
    path: '/dropz.DropzService/WatchDiscoveredCameras',
    requestStream: false,
    responseStream: true,
    requestType: gopro_pb.GetDiscoveredCamerasRequest,
    responseType: gopro_pb.GetDiscoveredCamerasResponse,
    requestSerialize: serialize_dropz_GetDiscoveredCamerasRequest,
    requestDeserialize: deserialize_dropz_GetDiscoveredCamerasRequest,
    responseSerialize: serialize_dropz_GetDiscoveredCamerasResponse,
    responseDeserialize: deserialize_dropz_GetDiscoveredCamerasResponse,
  },
  // Camera Pool management (managed cameras)
getManagedCameras: {
    path: '/dropz.DropzService/GetManagedCameras',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.GetManagedCamerasRequest,
    responseType: gopro_pb.GetManagedCamerasResponse,
    requestSerialize: serialize_dropz_GetManagedCamerasRequest,
    requestDeserialize: deserialize_dropz_GetManagedCamerasRequest,
    responseSerialize: serialize_dropz_GetManagedCamerasResponse,
    responseDeserialize: deserialize_dropz_GetManagedCamerasResponse,
  },
  watchManagedCameras: {
    path: '/dropz.DropzService/WatchManagedCameras',
    requestStream: false,
    responseStream: true,
    requestType: gopro_pb.GetManagedCamerasRequest,
    responseType: gopro_pb.GetManagedCamerasResponse,
    requestSerialize: serialize_dropz_GetManagedCamerasRequest,
    requestDeserialize: deserialize_dropz_GetManagedCamerasRequest,
    responseSerialize: serialize_dropz_GetManagedCamerasResponse,
    responseDeserialize: deserialize_dropz_GetManagedCamerasResponse,
  },
  manageCamera: {
    path: '/dropz.DropzService/ManageCamera',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.ManageCameraRequest,
    responseType: gopro_pb.ManageCameraResponse,
    requestSerialize: serialize_dropz_ManageCameraRequest,
    requestDeserialize: deserialize_dropz_ManageCameraRequest,
    responseSerialize: serialize_dropz_ManageCameraResponse,
    responseDeserialize: deserialize_dropz_ManageCameraResponse,
  },
  unmanageCamera: {
    path: '/dropz.DropzService/UnmanageCamera',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.UnmanageCameraRequest,
    responseType: gopro_pb.UnmanageCameraResponse,
    requestSerialize: serialize_dropz_UnmanageCameraRequest,
    requestDeserialize: deserialize_dropz_UnmanageCameraRequest,
    responseSerialize: serialize_dropz_UnmanageCameraResponse,
    responseDeserialize: deserialize_dropz_UnmanageCameraResponse,
  },
  // Pairing operations
pairCamera: {
    path: '/dropz.DropzService/PairCamera',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.PairCameraRequest,
    responseType: gopro_pb.PairCameraResponse,
    requestSerialize: serialize_dropz_PairCameraRequest,
    requestDeserialize: deserialize_dropz_PairCameraRequest,
    responseSerialize: serialize_dropz_PairCameraResponse,
    responseDeserialize: deserialize_dropz_PairCameraResponse,
  },
  // Sync Queue operations
getSyncQueue: {
    path: '/dropz.DropzService/GetSyncQueue',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.GetSyncQueueRequest,
    responseType: gopro_pb.GetSyncQueueResponse,
    requestSerialize: serialize_dropz_GetSyncQueueRequest,
    requestDeserialize: deserialize_dropz_GetSyncQueueRequest,
    responseSerialize: serialize_dropz_GetSyncQueueResponse,
    responseDeserialize: deserialize_dropz_GetSyncQueueResponse,
  },
  watchSyncQueue: {
    path: '/dropz.DropzService/WatchSyncQueue',
    requestStream: false,
    responseStream: true,
    requestType: gopro_pb.GetSyncQueueRequest,
    responseType: gopro_pb.GetSyncQueueResponse,
    requestSerialize: serialize_dropz_GetSyncQueueRequest,
    requestDeserialize: deserialize_dropz_GetSyncQueueRequest,
    responseSerialize: serialize_dropz_GetSyncQueueResponse,
    responseDeserialize: deserialize_dropz_GetSyncQueueResponse,
  },
  forceSync: {
    path: '/dropz.DropzService/ForceSync',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.ForceSyncRequest,
    responseType: gopro_pb.ForceSyncResponse,
    requestSerialize: serialize_dropz_ForceSyncRequest,
    requestDeserialize: deserialize_dropz_ForceSyncRequest,
    responseSerialize: serialize_dropz_ForceSyncResponse,
    responseDeserialize: deserialize_dropz_ForceSyncResponse,
  },
  cancelSync: {
    path: '/dropz.DropzService/CancelSync',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.CancelSyncRequest,
    responseType: gopro_pb.CancelSyncResponse,
    requestSerialize: serialize_dropz_CancelSyncRequest,
    requestDeserialize: deserialize_dropz_CancelSyncRequest,
    responseSerialize: serialize_dropz_CancelSyncResponse,
    responseDeserialize: deserialize_dropz_CancelSyncResponse,
  },
  // Group management
getGroups: {
    path: '/dropz.DropzService/GetGroups',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.GetGroupsRequest,
    responseType: gopro_pb.GetGroupsResponse,
    requestSerialize: serialize_dropz_GetGroupsRequest,
    requestDeserialize: deserialize_dropz_GetGroupsRequest,
    responseSerialize: serialize_dropz_GetGroupsResponse,
    responseDeserialize: deserialize_dropz_GetGroupsResponse,
  },
  createGroup: {
    path: '/dropz.DropzService/CreateGroup',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.CreateGroupRequest,
    responseType: gopro_pb.Group,
    requestSerialize: serialize_dropz_CreateGroupRequest,
    requestDeserialize: deserialize_dropz_CreateGroupRequest,
    responseSerialize: serialize_dropz_Group,
    responseDeserialize: deserialize_dropz_Group,
  },
  updateGroup: {
    path: '/dropz.DropzService/UpdateGroup',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.UpdateGroupRequest,
    responseType: gopro_pb.Group,
    requestSerialize: serialize_dropz_UpdateGroupRequest,
    requestDeserialize: deserialize_dropz_UpdateGroupRequest,
    responseSerialize: serialize_dropz_Group,
    responseDeserialize: deserialize_dropz_Group,
  },
  deleteGroup: {
    path: '/dropz.DropzService/DeleteGroup',
    requestStream: false,
    responseStream: false,
    requestType: gopro_pb.DeleteGroupRequest,
    responseType: common_pb.OperationResponse,
    requestSerialize: serialize_dropz_DeleteGroupRequest,
    requestDeserialize: deserialize_dropz_DeleteGroupRequest,
    responseSerialize: serialize_dropz_OperationResponse,
    responseDeserialize: deserialize_dropz_OperationResponse,
  },
  // Video management
getVideos: {
    path: '/dropz.DropzService/GetVideos',
    requestStream: false,
    responseStream: false,
    requestType: video_pb.GetVideosRequest,
    responseType: video_pb.GetVideosResponse,
    requestSerialize: serialize_dropz_GetVideosRequest,
    requestDeserialize: deserialize_dropz_GetVideosRequest,
    responseSerialize: serialize_dropz_GetVideosResponse,
    responseDeserialize: deserialize_dropz_GetVideosResponse,
  },
  // Configuration & Settings
getConfig: {
    path: '/dropz.DropzService/GetConfig',
    requestStream: false,
    responseStream: false,
    requestType: config_pb.GetConfigRequest,
    responseType: config_pb.GetConfigResponse,
    requestSerialize: serialize_dropz_GetConfigRequest,
    requestDeserialize: deserialize_dropz_GetConfigRequest,
    responseSerialize: serialize_dropz_GetConfigResponse,
    responseDeserialize: deserialize_dropz_GetConfigResponse,
  },
  updateConfig: {
    path: '/dropz.DropzService/UpdateConfig',
    requestStream: false,
    responseStream: false,
    requestType: config_pb.UpdateConfigRequest,
    responseType: config_pb.UpdateConfigResponse,
    requestSerialize: serialize_dropz_UpdateConfigRequest,
    requestDeserialize: deserialize_dropz_UpdateConfigRequest,
    responseSerialize: serialize_dropz_UpdateConfigResponse,
    responseDeserialize: deserialize_dropz_UpdateConfigResponse,
  },
  getSetting: {
    path: '/dropz.DropzService/GetSetting',
    requestStream: false,
    responseStream: false,
    requestType: config_pb.GetSettingRequest,
    responseType: config_pb.GetSettingResponse,
    requestSerialize: serialize_dropz_GetSettingRequest,
    requestDeserialize: deserialize_dropz_GetSettingRequest,
    responseSerialize: serialize_dropz_GetSettingResponse,
    responseDeserialize: deserialize_dropz_GetSettingResponse,
  },
  updateSetting: {
    path: '/dropz.DropzService/UpdateSetting',
    requestStream: false,
    responseStream: false,
    requestType: config_pb.UpdateSettingRequest,
    responseType: config_pb.Config,
    requestSerialize: serialize_dropz_UpdateSettingRequest,
    requestDeserialize: deserialize_dropz_UpdateSettingRequest,
    responseSerialize: serialize_dropz_Config,
    responseDeserialize: deserialize_dropz_Config,
  },
  resetSetting: {
    path: '/dropz.DropzService/ResetSetting',
    requestStream: false,
    responseStream: false,
    requestType: config_pb.ResetSettingRequest,
    responseType: config_pb.Config,
    requestSerialize: serialize_dropz_ResetSettingRequest,
    requestDeserialize: deserialize_dropz_ResetSettingRequest,
    responseSerialize: serialize_dropz_Config,
    responseDeserialize: deserialize_dropz_Config,
  },
  // Logs
getLogs: {
    path: '/dropz.DropzService/GetLogs',
    requestStream: false,
    responseStream: false,
    requestType: logs_pb.GetLogsRequest,
    responseType: logs_pb.GetLogsResponse,
    requestSerialize: serialize_dropz_GetLogsRequest,
    requestDeserialize: deserialize_dropz_GetLogsRequest,
    responseSerialize: serialize_dropz_GetLogsResponse,
    responseDeserialize: deserialize_dropz_GetLogsResponse,
  },
};

exports.DropzServiceClient = grpc.makeGenericClientConstructor(DropzServiceService, 'DropzService');
