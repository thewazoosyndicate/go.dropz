// source: gopro.proto
/**
 * @fileoverview
 * @enhanceable
 * @suppress {missingRequire} reports error on implicit type usages.
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!
/* eslint-disable */
// @ts-nocheck

var jspb = require('google-protobuf');
var goog = jspb;
var global = (function() {
  if (this) { return this; }
  if (typeof window !== 'undefined') { return window; }
  if (typeof global !== 'undefined') { return global; }
  if (typeof self !== 'undefined') { return self; }
  return Function('return this')();
}.call(null));

var common_pb = require('./common_pb.js');
goog.object.extend(proto, common_pb);
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
goog.object.extend(proto, google_protobuf_timestamp_pb);
goog.exportSymbol('proto.dropz.ApplyCameraSettingsRequest', null, global);
goog.exportSymbol('proto.dropz.ApplyCameraSettingsResponse', null, global);
goog.exportSymbol('proto.dropz.Camera', null, global);
goog.exportSymbol('proto.dropz.CameraMediaItem', null, global);
goog.exportSymbol('proto.dropz.CameraMetadata', null, global);
goog.exportSymbol('proto.dropz.CameraSetting', null, global);
goog.exportSymbol('proto.dropz.CameraSettingsResult', null, global);
goog.exportSymbol('proto.dropz.CameraStatus', null, global);
goog.exportSymbol('proto.dropz.CameraWithState', null, global);
goog.exportSymbol('proto.dropz.CancelSyncRequest', null, global);
goog.exportSymbol('proto.dropz.CancelSyncResponse', null, global);
goog.exportSymbol('proto.dropz.CreateGroupRequest', null, global);
goog.exportSymbol('proto.dropz.DeleteGroupRequest', null, global);
goog.exportSymbol('proto.dropz.DiscoveredCamera', null, global);
goog.exportSymbol('proto.dropz.ForceSyncRequest', null, global);
goog.exportSymbol('proto.dropz.ForceSyncResponse', null, global);
goog.exportSymbol('proto.dropz.ForgetCameraRequest', null, global);
goog.exportSymbol('proto.dropz.ForgetCameraResponse', null, global);
goog.exportSymbol('proto.dropz.GetCameraMediaRequest', null, global);
goog.exportSymbol('proto.dropz.GetCameraMediaResponse', null, global);
goog.exportSymbol('proto.dropz.GetCameraSettingsRequest', null, global);
goog.exportSymbol('proto.dropz.GetCameraSettingsResponse', null, global);
goog.exportSymbol('proto.dropz.GetDiscoveredCamerasRequest', null, global);
goog.exportSymbol('proto.dropz.GetDiscoveredCamerasResponse', null, global);
goog.exportSymbol('proto.dropz.GetGroupsRequest', null, global);
goog.exportSymbol('proto.dropz.GetGroupsResponse', null, global);
goog.exportSymbol('proto.dropz.GetManagedCamerasRequest', null, global);
goog.exportSymbol('proto.dropz.GetManagedCamerasResponse', null, global);
goog.exportSymbol('proto.dropz.GetSyncHistoryRequest', null, global);
goog.exportSymbol('proto.dropz.GetSyncHistoryResponse', null, global);
goog.exportSymbol('proto.dropz.GetSyncQueueRequest', null, global);
goog.exportSymbol('proto.dropz.GetSyncQueueResponse', null, global);
goog.exportSymbol('proto.dropz.Group', null, global);
goog.exportSymbol('proto.dropz.LoadGroupRequest', null, global);
goog.exportSymbol('proto.dropz.LoadGroupResponse', null, global);
goog.exportSymbol('proto.dropz.ManageCameraRequest', null, global);
goog.exportSymbol('proto.dropz.ManageCameraResponse', null, global);
goog.exportSymbol('proto.dropz.ManagedCamera', null, global);
goog.exportSymbol('proto.dropz.MoveCamerasToGroupRequest', null, global);
goog.exportSymbol('proto.dropz.PairCameraRequest', null, global);
goog.exportSymbol('proto.dropz.PairCameraResponse', null, global);
goog.exportSymbol('proto.dropz.PreviewMediaRequest', null, global);
goog.exportSymbol('proto.dropz.PreviewMediaResponse', null, global);
goog.exportSymbol('proto.dropz.PreviewVideoRequest', null, global);
goog.exportSymbol('proto.dropz.PreviewVideoResponse', null, global);
goog.exportSymbol('proto.dropz.RequestMediaDownloadRequest', null, global);
goog.exportSymbol('proto.dropz.RequestMediaDownloadResponse', null, global);
goog.exportSymbol('proto.dropz.SaveManagedAsGroupRequest', null, global);
goog.exportSymbol('proto.dropz.SetCameraAliasRequest', null, global);
goog.exportSymbol('proto.dropz.SetCameraAliasResponse', null, global);
goog.exportSymbol('proto.dropz.SetGroupSyncRequest', null, global);
goog.exportSymbol('proto.dropz.SetPreviewSessionRequest', null, global);
goog.exportSymbol('proto.dropz.SetPreviewSessionResponse', null, global);
goog.exportSymbol('proto.dropz.SettingApplyResult', null, global);
goog.exportSymbol('proto.dropz.SettingChange', null, global);
goog.exportSymbol('proto.dropz.SettingOption', null, global);
goog.exportSymbol('proto.dropz.SyncFile', null, global);
goog.exportSymbol('proto.dropz.SyncFileState', null, global);
goog.exportSymbol('proto.dropz.SyncOutcome', null, global);
goog.exportSymbol('proto.dropz.SyncPhase', null, global);
goog.exportSymbol('proto.dropz.SyncPhaseTiming', null, global);
goog.exportSymbol('proto.dropz.SyncQueueEntry', null, global);
goog.exportSymbol('proto.dropz.SyncSession', null, global);
goog.exportSymbol('proto.dropz.UnmanageCameraRequest', null, global);
goog.exportSymbol('proto.dropz.UnmanageCameraResponse', null, global);
goog.exportSymbol('proto.dropz.UpdateGroupRequest', null, global);
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.Camera = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.Camera, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.Camera.displayName = 'proto.dropz.Camera';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CameraStatus = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.CameraStatus, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CameraStatus.displayName = 'proto.dropz.CameraStatus';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CameraMetadata = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.CameraMetadata, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CameraMetadata.displayName = 'proto.dropz.CameraMetadata';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CameraWithState = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.CameraWithState, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CameraWithState.displayName = 'proto.dropz.CameraWithState';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.DiscoveredCamera = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.DiscoveredCamera, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.DiscoveredCamera.displayName = 'proto.dropz.DiscoveredCamera';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ManagedCamera = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.ManagedCamera, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ManagedCamera.displayName = 'proto.dropz.ManagedCamera';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SyncFile = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SyncFile, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SyncFile.displayName = 'proto.dropz.SyncFile';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SyncPhaseTiming = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SyncPhaseTiming, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SyncPhaseTiming.displayName = 'proto.dropz.SyncPhaseTiming';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SyncQueueEntry = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.SyncQueueEntry.repeatedFields_, null);
};
goog.inherits(proto.dropz.SyncQueueEntry, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SyncQueueEntry.displayName = 'proto.dropz.SyncQueueEntry';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SyncSession = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.SyncSession.repeatedFields_, null);
};
goog.inherits(proto.dropz.SyncSession, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SyncSession.displayName = 'proto.dropz.SyncSession';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetSyncHistoryRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.GetSyncHistoryRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetSyncHistoryRequest.displayName = 'proto.dropz.GetSyncHistoryRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetSyncHistoryResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.GetSyncHistoryResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.GetSyncHistoryResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetSyncHistoryResponse.displayName = 'proto.dropz.GetSyncHistoryResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SetCameraAliasRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SetCameraAliasRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SetCameraAliasRequest.displayName = 'proto.dropz.SetCameraAliasRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SetCameraAliasResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SetCameraAliasResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SetCameraAliasResponse.displayName = 'proto.dropz.SetCameraAliasResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.Group = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.Group.repeatedFields_, null);
};
goog.inherits(proto.dropz.Group, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.Group.displayName = 'proto.dropz.Group';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.MoveCamerasToGroupRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.MoveCamerasToGroupRequest.repeatedFields_, null);
};
goog.inherits(proto.dropz.MoveCamerasToGroupRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.MoveCamerasToGroupRequest.displayName = 'proto.dropz.MoveCamerasToGroupRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SetGroupSyncRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SetGroupSyncRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SetGroupSyncRequest.displayName = 'proto.dropz.SetGroupSyncRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetDiscoveredCamerasRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.GetDiscoveredCamerasRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetDiscoveredCamerasRequest.displayName = 'proto.dropz.GetDiscoveredCamerasRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetDiscoveredCamerasResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.GetDiscoveredCamerasResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.GetDiscoveredCamerasResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetDiscoveredCamerasResponse.displayName = 'proto.dropz.GetDiscoveredCamerasResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetManagedCamerasRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.GetManagedCamerasRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetManagedCamerasRequest.displayName = 'proto.dropz.GetManagedCamerasRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetManagedCamerasResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.GetManagedCamerasResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.GetManagedCamerasResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetManagedCamerasResponse.displayName = 'proto.dropz.GetManagedCamerasResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetSyncQueueRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.GetSyncQueueRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetSyncQueueRequest.displayName = 'proto.dropz.GetSyncQueueRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetSyncQueueResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.GetSyncQueueResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.GetSyncQueueResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetSyncQueueResponse.displayName = 'proto.dropz.GetSyncQueueResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ManageCameraRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.ManageCameraRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ManageCameraRequest.displayName = 'proto.dropz.ManageCameraRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ManageCameraResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.ManageCameraResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ManageCameraResponse.displayName = 'proto.dropz.ManageCameraResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.UnmanageCameraRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.UnmanageCameraRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.UnmanageCameraRequest.displayName = 'proto.dropz.UnmanageCameraRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.UnmanageCameraResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.UnmanageCameraResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.UnmanageCameraResponse.displayName = 'proto.dropz.UnmanageCameraResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.PairCameraRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.PairCameraRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.PairCameraRequest.displayName = 'proto.dropz.PairCameraRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.PairCameraResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.PairCameraResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.PairCameraResponse.displayName = 'proto.dropz.PairCameraResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ForgetCameraRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.ForgetCameraRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ForgetCameraRequest.displayName = 'proto.dropz.ForgetCameraRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ForgetCameraResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.ForgetCameraResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ForgetCameraResponse.displayName = 'proto.dropz.ForgetCameraResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ForceSyncRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.ForceSyncRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ForceSyncRequest.displayName = 'proto.dropz.ForceSyncRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ForceSyncResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.ForceSyncResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ForceSyncResponse.displayName = 'proto.dropz.ForceSyncResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CancelSyncRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.CancelSyncRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CancelSyncRequest.displayName = 'proto.dropz.CancelSyncRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CancelSyncResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.CancelSyncResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CancelSyncResponse.displayName = 'proto.dropz.CancelSyncResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetGroupsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.GetGroupsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetGroupsRequest.displayName = 'proto.dropz.GetGroupsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetGroupsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.GetGroupsResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.GetGroupsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetGroupsResponse.displayName = 'proto.dropz.GetGroupsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CreateGroupRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.CreateGroupRequest.repeatedFields_, null);
};
goog.inherits(proto.dropz.CreateGroupRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CreateGroupRequest.displayName = 'proto.dropz.CreateGroupRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.UpdateGroupRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.UpdateGroupRequest.repeatedFields_, null);
};
goog.inherits(proto.dropz.UpdateGroupRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.UpdateGroupRequest.displayName = 'proto.dropz.UpdateGroupRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.DeleteGroupRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.DeleteGroupRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.DeleteGroupRequest.displayName = 'proto.dropz.DeleteGroupRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.LoadGroupRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.LoadGroupRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.LoadGroupRequest.displayName = 'proto.dropz.LoadGroupRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.LoadGroupResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.LoadGroupResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.LoadGroupResponse.displayName = 'proto.dropz.LoadGroupResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SaveManagedAsGroupRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SaveManagedAsGroupRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SaveManagedAsGroupRequest.displayName = 'proto.dropz.SaveManagedAsGroupRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SettingOption = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SettingOption, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SettingOption.displayName = 'proto.dropz.SettingOption';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CameraSetting = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.CameraSetting.repeatedFields_, null);
};
goog.inherits(proto.dropz.CameraSetting, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CameraSetting.displayName = 'proto.dropz.CameraSetting';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SettingChange = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SettingChange, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SettingChange.displayName = 'proto.dropz.SettingChange';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SettingApplyResult = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SettingApplyResult, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SettingApplyResult.displayName = 'proto.dropz.SettingApplyResult';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CameraSettingsResult = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.CameraSettingsResult.repeatedFields_, null);
};
goog.inherits(proto.dropz.CameraSettingsResult, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CameraSettingsResult.displayName = 'proto.dropz.CameraSettingsResult';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetCameraSettingsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.GetCameraSettingsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetCameraSettingsRequest.displayName = 'proto.dropz.GetCameraSettingsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetCameraSettingsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.GetCameraSettingsResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.GetCameraSettingsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetCameraSettingsResponse.displayName = 'proto.dropz.GetCameraSettingsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ApplyCameraSettingsRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.ApplyCameraSettingsRequest.repeatedFields_, null);
};
goog.inherits(proto.dropz.ApplyCameraSettingsRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ApplyCameraSettingsRequest.displayName = 'proto.dropz.ApplyCameraSettingsRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.ApplyCameraSettingsResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.ApplyCameraSettingsResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.ApplyCameraSettingsResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.ApplyCameraSettingsResponse.displayName = 'proto.dropz.ApplyCameraSettingsResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.CameraMediaItem = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.CameraMediaItem, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.CameraMediaItem.displayName = 'proto.dropz.CameraMediaItem';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetCameraMediaRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.GetCameraMediaRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetCameraMediaRequest.displayName = 'proto.dropz.GetCameraMediaRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.GetCameraMediaResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.GetCameraMediaResponse.repeatedFields_, null);
};
goog.inherits(proto.dropz.GetCameraMediaResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.GetCameraMediaResponse.displayName = 'proto.dropz.GetCameraMediaResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.PreviewMediaRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.PreviewMediaRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.PreviewMediaRequest.displayName = 'proto.dropz.PreviewMediaRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.PreviewMediaResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.PreviewMediaResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.PreviewMediaResponse.displayName = 'proto.dropz.PreviewMediaResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SetPreviewSessionRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SetPreviewSessionRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SetPreviewSessionRequest.displayName = 'proto.dropz.SetPreviewSessionRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.SetPreviewSessionResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.SetPreviewSessionResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.SetPreviewSessionResponse.displayName = 'proto.dropz.SetPreviewSessionResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.PreviewVideoRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.PreviewVideoRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.PreviewVideoRequest.displayName = 'proto.dropz.PreviewVideoRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.PreviewVideoResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.PreviewVideoResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.PreviewVideoResponse.displayName = 'proto.dropz.PreviewVideoResponse';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.RequestMediaDownloadRequest = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, proto.dropz.RequestMediaDownloadRequest.repeatedFields_, null);
};
goog.inherits(proto.dropz.RequestMediaDownloadRequest, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.RequestMediaDownloadRequest.displayName = 'proto.dropz.RequestMediaDownloadRequest';
}
/**
 * Generated by JsPbCodeGenerator.
 * @param {Array=} opt_data Optional initial data array, typically from a
 * server response, or constructed directly in Javascript. The array is used
 * in place and becomes part of the constructed object. It is not cloned.
 * If no data is provided, the constructed object will be empty, but still
 * valid.
 * @extends {jspb.Message}
 * @constructor
 */
proto.dropz.RequestMediaDownloadResponse = function(opt_data) {
  jspb.Message.initialize(this, opt_data, 0, -1, null, null);
};
goog.inherits(proto.dropz.RequestMediaDownloadResponse, jspb.Message);
if (goog.DEBUG && !COMPILED) {
  /**
   * @public
   * @override
   */
  proto.dropz.RequestMediaDownloadResponse.displayName = 'proto.dropz.RequestMediaDownloadResponse';
}



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.Camera.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.Camera.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.Camera} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.Camera.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    name: jspb.Message.getFieldWithDefault(msg, 2, ""),
    alias: jspb.Message.getFieldWithDefault(msg, 3, ""),
    bleAddress: jspb.Message.getFieldWithDefault(msg, 4, ""),
    wifiSsid: jspb.Message.getFieldWithDefault(msg, 5, ""),
    wifiPassword: jspb.Message.getFieldWithDefault(msg, 6, ""),
    rssi: jspb.Message.getFieldWithDefault(msg, 7, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.Camera}
 */
proto.dropz.Camera.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.Camera;
  return proto.dropz.Camera.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.Camera} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.Camera}
 */
proto.dropz.Camera.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setAlias(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setBleAddress(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setWifiSsid(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setWifiPassword(value);
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setRssi(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.Camera.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.Camera.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.Camera} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.Camera.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getAlias();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getBleAddress();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getWifiSsid();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getWifiPassword();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getRssi();
  if (f !== 0) {
    writer.writeInt32(
      7,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.dropz.Camera.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Camera} returns this
 */
proto.dropz.Camera.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.dropz.Camera.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Camera} returns this
 */
proto.dropz.Camera.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string alias = 3;
 * @return {string}
 */
proto.dropz.Camera.prototype.getAlias = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Camera} returns this
 */
proto.dropz.Camera.prototype.setAlias = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string ble_address = 4;
 * @return {string}
 */
proto.dropz.Camera.prototype.getBleAddress = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Camera} returns this
 */
proto.dropz.Camera.prototype.setBleAddress = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional string wifi_ssid = 5;
 * @return {string}
 */
proto.dropz.Camera.prototype.getWifiSsid = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Camera} returns this
 */
proto.dropz.Camera.prototype.setWifiSsid = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional string wifi_password = 6;
 * @return {string}
 */
proto.dropz.Camera.prototype.getWifiPassword = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Camera} returns this
 */
proto.dropz.Camera.prototype.setWifiPassword = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional int32 rssi = 7;
 * @return {number}
 */
proto.dropz.Camera.prototype.getRssi = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.Camera} returns this
 */
proto.dropz.Camera.prototype.setRssi = function(value) {
  return jspb.Message.setProto3IntField(this, 7, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CameraStatus.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CameraStatus.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CameraStatus} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraStatus.toObject = function(includeInstance, msg) {
  var f, obj = {
    lastSeen: (f = msg.getLastSeen()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    lastSynced: (f = msg.getLastSynced()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    isPairing: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    isPaired: jspb.Message.getBooleanFieldWithDefault(msg, 4, false),
    isManaged: jspb.Message.getBooleanFieldWithDefault(msg, 5, false),
    isReachable: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    isSynced: jspb.Message.getBooleanFieldWithDefault(msg, 7, false),
    isSyncing: jspb.Message.getBooleanFieldWithDefault(msg, 8, false),
    lastSyncError: jspb.Message.getFieldWithDefault(msg, 9, ""),
    inPairingMode: jspb.Message.getBooleanFieldWithDefault(msg, 10, false),
    previewEnabled: jspb.Message.getBooleanFieldWithDefault(msg, 11, false),
    newMediaCount: jspb.Message.getFieldWithDefault(msg, 12, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CameraStatus}
 */
proto.dropz.CameraStatus.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CameraStatus;
  return proto.dropz.CameraStatus.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CameraStatus} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CameraStatus}
 */
proto.dropz.CameraStatus.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setLastSeen(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setLastSynced(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsPairing(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsPaired(value);
      break;
    case 5:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsManaged(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsReachable(value);
      break;
    case 7:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsSynced(value);
      break;
    case 8:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setIsSyncing(value);
      break;
    case 9:
      var value = /** @type {string} */ (reader.readString());
      msg.setLastSyncError(value);
      break;
    case 10:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setInPairingMode(value);
      break;
    case 11:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setPreviewEnabled(value);
      break;
    case 12:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setNewMediaCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CameraStatus.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CameraStatus.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CameraStatus} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraStatus.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getLastSeen();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getLastSynced();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getIsPairing();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getIsPaired();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
  f = message.getIsManaged();
  if (f) {
    writer.writeBool(
      5,
      f
    );
  }
  f = message.getIsReachable();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getIsSynced();
  if (f) {
    writer.writeBool(
      7,
      f
    );
  }
  f = message.getIsSyncing();
  if (f) {
    writer.writeBool(
      8,
      f
    );
  }
  f = message.getLastSyncError();
  if (f.length > 0) {
    writer.writeString(
      9,
      f
    );
  }
  f = message.getInPairingMode();
  if (f) {
    writer.writeBool(
      10,
      f
    );
  }
  f = message.getPreviewEnabled();
  if (f) {
    writer.writeBool(
      11,
      f
    );
  }
  f = message.getNewMediaCount();
  if (f !== 0) {
    writer.writeInt32(
      12,
      f
    );
  }
};


/**
 * optional google.protobuf.Timestamp last_seen = 1;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.CameraStatus.prototype.getLastSeen = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 1));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.CameraStatus} returns this
*/
proto.dropz.CameraStatus.prototype.setLastSeen = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.clearLastSeen = function() {
  return this.setLastSeen(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.hasLastSeen = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional google.protobuf.Timestamp last_synced = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.CameraStatus.prototype.getLastSynced = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.CameraStatus} returns this
*/
proto.dropz.CameraStatus.prototype.setLastSynced = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.clearLastSynced = function() {
  return this.setLastSynced(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.hasLastSynced = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional bool is_pairing = 3;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getIsPairing = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setIsPairing = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool is_paired = 4;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getIsPaired = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setIsPaired = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};


/**
 * optional bool is_managed = 5;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getIsManaged = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 5, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setIsManaged = function(value) {
  return jspb.Message.setProto3BooleanField(this, 5, value);
};


/**
 * optional bool is_reachable = 6;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getIsReachable = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setIsReachable = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional bool is_synced = 7;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getIsSynced = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 7, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setIsSynced = function(value) {
  return jspb.Message.setProto3BooleanField(this, 7, value);
};


/**
 * optional bool is_syncing = 8;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getIsSyncing = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 8, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setIsSyncing = function(value) {
  return jspb.Message.setProto3BooleanField(this, 8, value);
};


/**
 * optional string last_sync_error = 9;
 * @return {string}
 */
proto.dropz.CameraStatus.prototype.getLastSyncError = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 9, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setLastSyncError = function(value) {
  return jspb.Message.setProto3StringField(this, 9, value);
};


/**
 * optional bool in_pairing_mode = 10;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getInPairingMode = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 10, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setInPairingMode = function(value) {
  return jspb.Message.setProto3BooleanField(this, 10, value);
};


/**
 * optional bool preview_enabled = 11;
 * @return {boolean}
 */
proto.dropz.CameraStatus.prototype.getPreviewEnabled = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 11, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setPreviewEnabled = function(value) {
  return jspb.Message.setProto3BooleanField(this, 11, value);
};


/**
 * optional int32 new_media_count = 12;
 * @return {number}
 */
proto.dropz.CameraStatus.prototype.getNewMediaCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 12, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraStatus} returns this
 */
proto.dropz.CameraStatus.prototype.setNewMediaCount = function(value) {
  return jspb.Message.setProto3IntField(this, 12, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CameraMetadata.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CameraMetadata.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CameraMetadata} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraMetadata.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    firmwareVersion: jspb.Message.getFieldWithDefault(msg, 2, ""),
    model: jspb.Message.getFieldWithDefault(msg, 3, ""),
    serialNumber: jspb.Message.getFieldWithDefault(msg, 4, ""),
    batteryLevel: jspb.Message.getFieldWithDefault(msg, 5, 0),
    hardwareVersion: jspb.Message.getFieldWithDefault(msg, 7, ""),
    numPhotos: jspb.Message.getFieldWithDefault(msg, 8, 0),
    numVideos: jspb.Message.getFieldWithDefault(msg, 9, 0),
    sdCardStatus: jspb.Message.getFieldWithDefault(msg, 10, 0),
    remainingSpaceKb: jspb.Message.getFieldWithDefault(msg, 11, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CameraMetadata}
 */
proto.dropz.CameraMetadata.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CameraMetadata;
  return proto.dropz.CameraMetadata.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CameraMetadata} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CameraMetadata}
 */
proto.dropz.CameraMetadata.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setFirmwareVersion(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setModel(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setSerialNumber(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setBatteryLevel(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setHardwareVersion(value);
      break;
    case 8:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setNumPhotos(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setNumVideos(value);
      break;
    case 10:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setSdCardStatus(value);
      break;
    case 11:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setRemainingSpaceKb(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CameraMetadata.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CameraMetadata.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CameraMetadata} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraMetadata.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getFirmwareVersion();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getModel();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getSerialNumber();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getBatteryLevel();
  if (f !== 0) {
    writer.writeInt32(
      5,
      f
    );
  }
  f = message.getHardwareVersion();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getNumPhotos();
  if (f !== 0) {
    writer.writeInt32(
      8,
      f
    );
  }
  f = message.getNumVideos();
  if (f !== 0) {
    writer.writeInt32(
      9,
      f
    );
  }
  f = message.getSdCardStatus();
  if (f !== 0) {
    writer.writeInt32(
      10,
      f
    );
  }
  f = message.getRemainingSpaceKb();
  if (f !== 0) {
    writer.writeInt64(
      11,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.dropz.CameraMetadata.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string firmware_version = 2;
 * @return {string}
 */
proto.dropz.CameraMetadata.prototype.getFirmwareVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setFirmwareVersion = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional string model = 3;
 * @return {string}
 */
proto.dropz.CameraMetadata.prototype.getModel = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setModel = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional string serial_number = 4;
 * @return {string}
 */
proto.dropz.CameraMetadata.prototype.getSerialNumber = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setSerialNumber = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * optional int32 battery_level = 5;
 * @return {number}
 */
proto.dropz.CameraMetadata.prototype.getBatteryLevel = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setBatteryLevel = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional string hardware_version = 7;
 * @return {string}
 */
proto.dropz.CameraMetadata.prototype.getHardwareVersion = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setHardwareVersion = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional int32 num_photos = 8;
 * @return {number}
 */
proto.dropz.CameraMetadata.prototype.getNumPhotos = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setNumPhotos = function(value) {
  return jspb.Message.setProto3IntField(this, 8, value);
};


/**
 * optional int32 num_videos = 9;
 * @return {number}
 */
proto.dropz.CameraMetadata.prototype.getNumVideos = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setNumVideos = function(value) {
  return jspb.Message.setProto3IntField(this, 9, value);
};


/**
 * optional int32 sd_card_status = 10;
 * @return {number}
 */
proto.dropz.CameraMetadata.prototype.getSdCardStatus = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setSdCardStatus = function(value) {
  return jspb.Message.setProto3IntField(this, 10, value);
};


/**
 * optional int64 remaining_space_kb = 11;
 * @return {number}
 */
proto.dropz.CameraMetadata.prototype.getRemainingSpaceKb = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 11, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraMetadata} returns this
 */
proto.dropz.CameraMetadata.prototype.setRemainingSpaceKb = function(value) {
  return jspb.Message.setProto3IntField(this, 11, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CameraWithState.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CameraWithState.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CameraWithState} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraWithState.toObject = function(includeInstance, msg) {
  var f, obj = {
    camera: (f = msg.getCamera()) && proto.dropz.Camera.toObject(includeInstance, f),
    status: (f = msg.getStatus()) && proto.dropz.CameraStatus.toObject(includeInstance, f),
    groupId: jspb.Message.getFieldWithDefault(msg, 3, ""),
    metadata: (f = msg.getMetadata()) && proto.dropz.CameraMetadata.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CameraWithState}
 */
proto.dropz.CameraWithState.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CameraWithState;
  return proto.dropz.CameraWithState.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CameraWithState} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CameraWithState}
 */
proto.dropz.CameraWithState.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.Camera;
      reader.readMessage(value,proto.dropz.Camera.deserializeBinaryFromReader);
      msg.setCamera(value);
      break;
    case 2:
      var value = new proto.dropz.CameraStatus;
      reader.readMessage(value,proto.dropz.CameraStatus.deserializeBinaryFromReader);
      msg.setStatus(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.setGroupId(value);
      break;
    case 4:
      var value = new proto.dropz.CameraMetadata;
      reader.readMessage(value,proto.dropz.CameraMetadata.deserializeBinaryFromReader);
      msg.setMetadata(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CameraWithState.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CameraWithState.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CameraWithState} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraWithState.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCamera();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.dropz.Camera.serializeBinaryToWriter
    );
  }
  f = message.getStatus();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      proto.dropz.CameraStatus.serializeBinaryToWriter
    );
  }
  f = message.getGroupId();
  if (f.length > 0) {
    writer.writeString(
      3,
      f
    );
  }
  f = message.getMetadata();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      proto.dropz.CameraMetadata.serializeBinaryToWriter
    );
  }
};


/**
 * optional Camera camera = 1;
 * @return {?proto.dropz.Camera}
 */
proto.dropz.CameraWithState.prototype.getCamera = function() {
  return /** @type{?proto.dropz.Camera} */ (
    jspb.Message.getWrapperField(this, proto.dropz.Camera, 1));
};


/**
 * @param {?proto.dropz.Camera|undefined} value
 * @return {!proto.dropz.CameraWithState} returns this
*/
proto.dropz.CameraWithState.prototype.setCamera = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.CameraWithState} returns this
 */
proto.dropz.CameraWithState.prototype.clearCamera = function() {
  return this.setCamera(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.CameraWithState.prototype.hasCamera = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * optional CameraStatus status = 2;
 * @return {?proto.dropz.CameraStatus}
 */
proto.dropz.CameraWithState.prototype.getStatus = function() {
  return /** @type{?proto.dropz.CameraStatus} */ (
    jspb.Message.getWrapperField(this, proto.dropz.CameraStatus, 2));
};


/**
 * @param {?proto.dropz.CameraStatus|undefined} value
 * @return {!proto.dropz.CameraWithState} returns this
*/
proto.dropz.CameraWithState.prototype.setStatus = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.CameraWithState} returns this
 */
proto.dropz.CameraWithState.prototype.clearStatus = function() {
  return this.setStatus(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.CameraWithState.prototype.hasStatus = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional string group_id = 3;
 * @return {string}
 */
proto.dropz.CameraWithState.prototype.getGroupId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 3, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraWithState} returns this
 */
proto.dropz.CameraWithState.prototype.setGroupId = function(value) {
  return jspb.Message.setProto3StringField(this, 3, value);
};


/**
 * optional CameraMetadata metadata = 4;
 * @return {?proto.dropz.CameraMetadata}
 */
proto.dropz.CameraWithState.prototype.getMetadata = function() {
  return /** @type{?proto.dropz.CameraMetadata} */ (
    jspb.Message.getWrapperField(this, proto.dropz.CameraMetadata, 4));
};


/**
 * @param {?proto.dropz.CameraMetadata|undefined} value
 * @return {!proto.dropz.CameraWithState} returns this
*/
proto.dropz.CameraWithState.prototype.setMetadata = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.CameraWithState} returns this
 */
proto.dropz.CameraWithState.prototype.clearMetadata = function() {
  return this.setMetadata(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.CameraWithState.prototype.hasMetadata = function() {
  return jspb.Message.getField(this, 4) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.DiscoveredCamera.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.DiscoveredCamera.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.DiscoveredCamera} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.DiscoveredCamera.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraState: (f = msg.getCameraState()) && proto.dropz.CameraWithState.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.DiscoveredCamera}
 */
proto.dropz.DiscoveredCamera.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.DiscoveredCamera;
  return proto.dropz.DiscoveredCamera.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.DiscoveredCamera} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.DiscoveredCamera}
 */
proto.dropz.DiscoveredCamera.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.CameraWithState;
      reader.readMessage(value,proto.dropz.CameraWithState.deserializeBinaryFromReader);
      msg.setCameraState(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.DiscoveredCamera.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.DiscoveredCamera.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.DiscoveredCamera} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.DiscoveredCamera.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraState();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.dropz.CameraWithState.serializeBinaryToWriter
    );
  }
};


/**
 * optional CameraWithState camera_state = 1;
 * @return {?proto.dropz.CameraWithState}
 */
proto.dropz.DiscoveredCamera.prototype.getCameraState = function() {
  return /** @type{?proto.dropz.CameraWithState} */ (
    jspb.Message.getWrapperField(this, proto.dropz.CameraWithState, 1));
};


/**
 * @param {?proto.dropz.CameraWithState|undefined} value
 * @return {!proto.dropz.DiscoveredCamera} returns this
*/
proto.dropz.DiscoveredCamera.prototype.setCameraState = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.DiscoveredCamera} returns this
 */
proto.dropz.DiscoveredCamera.prototype.clearCameraState = function() {
  return this.setCameraState(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.DiscoveredCamera.prototype.hasCameraState = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ManagedCamera.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ManagedCamera.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ManagedCamera} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ManagedCamera.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraState: (f = msg.getCameraState()) && proto.dropz.CameraWithState.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ManagedCamera}
 */
proto.dropz.ManagedCamera.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ManagedCamera;
  return proto.dropz.ManagedCamera.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ManagedCamera} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ManagedCamera}
 */
proto.dropz.ManagedCamera.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.CameraWithState;
      reader.readMessage(value,proto.dropz.CameraWithState.deserializeBinaryFromReader);
      msg.setCameraState(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ManagedCamera.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ManagedCamera.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ManagedCamera} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ManagedCamera.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraState();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.dropz.CameraWithState.serializeBinaryToWriter
    );
  }
};


/**
 * optional CameraWithState camera_state = 1;
 * @return {?proto.dropz.CameraWithState}
 */
proto.dropz.ManagedCamera.prototype.getCameraState = function() {
  return /** @type{?proto.dropz.CameraWithState} */ (
    jspb.Message.getWrapperField(this, proto.dropz.CameraWithState, 1));
};


/**
 * @param {?proto.dropz.CameraWithState|undefined} value
 * @return {!proto.dropz.ManagedCamera} returns this
*/
proto.dropz.ManagedCamera.prototype.setCameraState = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.ManagedCamera} returns this
 */
proto.dropz.ManagedCamera.prototype.clearCameraState = function() {
  return this.setCameraState(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.ManagedCamera.prototype.hasCameraState = function() {
  return jspb.Message.getField(this, 1) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SyncFile.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SyncFile.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SyncFile} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncFile.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    cameraPath: jspb.Message.getFieldWithDefault(msg, 2, ""),
    sizeBytes: jspb.Message.getFieldWithDefault(msg, 3, 0),
    state: jspb.Message.getFieldWithDefault(msg, 4, 0),
    bytesDone: jspb.Message.getFieldWithDefault(msg, 5, 0),
    error: jspb.Message.getFieldWithDefault(msg, 6, ""),
    localPath: jspb.Message.getFieldWithDefault(msg, 7, ""),
    durationMs: jspb.Message.getFieldWithDefault(msg, 8, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SyncFile}
 */
proto.dropz.SyncFile.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SyncFile;
  return proto.dropz.SyncFile.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SyncFile} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SyncFile}
 */
proto.dropz.SyncFile.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraPath(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setSizeBytes(value);
      break;
    case 4:
      var value = /** @type {!proto.dropz.SyncFileState} */ (reader.readEnum());
      msg.setState(value);
      break;
    case 5:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setBytesDone(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setError(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setLocalPath(value);
      break;
    case 8:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setDurationMs(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SyncFile.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SyncFile.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SyncFile} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncFile.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCameraPath();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getSizeBytes();
  if (f !== 0) {
    writer.writeInt64(
      3,
      f
    );
  }
  f = message.getState();
  if (f !== 0.0) {
    writer.writeEnum(
      4,
      f
    );
  }
  f = message.getBytesDone();
  if (f !== 0) {
    writer.writeInt64(
      5,
      f
    );
  }
  f = message.getError();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getLocalPath();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getDurationMs();
  if (f !== 0) {
    writer.writeInt64(
      8,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.dropz.SyncFile.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string camera_path = 2;
 * @return {string}
 */
proto.dropz.SyncFile.prototype.getCameraPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setCameraPath = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int64 size_bytes = 3;
 * @return {number}
 */
proto.dropz.SyncFile.prototype.getSizeBytes = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setSizeBytes = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional SyncFileState state = 4;
 * @return {!proto.dropz.SyncFileState}
 */
proto.dropz.SyncFile.prototype.getState = function() {
  return /** @type {!proto.dropz.SyncFileState} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {!proto.dropz.SyncFileState} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setState = function(value) {
  return jspb.Message.setProto3EnumField(this, 4, value);
};


/**
 * optional int64 bytes_done = 5;
 * @return {number}
 */
proto.dropz.SyncFile.prototype.getBytesDone = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setBytesDone = function(value) {
  return jspb.Message.setProto3IntField(this, 5, value);
};


/**
 * optional string error = 6;
 * @return {string}
 */
proto.dropz.SyncFile.prototype.getError = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setError = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional string local_path = 7;
 * @return {string}
 */
proto.dropz.SyncFile.prototype.getLocalPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setLocalPath = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional int64 duration_ms = 8;
 * @return {number}
 */
proto.dropz.SyncFile.prototype.getDurationMs = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncFile} returns this
 */
proto.dropz.SyncFile.prototype.setDurationMs = function(value) {
  return jspb.Message.setProto3IntField(this, 8, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SyncPhaseTiming.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SyncPhaseTiming.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SyncPhaseTiming} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncPhaseTiming.toObject = function(includeInstance, msg) {
  var f, obj = {
    phase: jspb.Message.getFieldWithDefault(msg, 1, 0),
    startedAt: (f = msg.getStartedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    finishedAt: (f = msg.getFinishedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SyncPhaseTiming}
 */
proto.dropz.SyncPhaseTiming.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SyncPhaseTiming;
  return proto.dropz.SyncPhaseTiming.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SyncPhaseTiming} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SyncPhaseTiming}
 */
proto.dropz.SyncPhaseTiming.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {!proto.dropz.SyncPhase} */ (reader.readEnum());
      msg.setPhase(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setStartedAt(value);
      break;
    case 3:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setFinishedAt(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SyncPhaseTiming.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SyncPhaseTiming.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SyncPhaseTiming} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncPhaseTiming.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPhase();
  if (f !== 0.0) {
    writer.writeEnum(
      1,
      f
    );
  }
  f = message.getStartedAt();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getFinishedAt();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * optional SyncPhase phase = 1;
 * @return {!proto.dropz.SyncPhase}
 */
proto.dropz.SyncPhaseTiming.prototype.getPhase = function() {
  return /** @type {!proto.dropz.SyncPhase} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {!proto.dropz.SyncPhase} value
 * @return {!proto.dropz.SyncPhaseTiming} returns this
 */
proto.dropz.SyncPhaseTiming.prototype.setPhase = function(value) {
  return jspb.Message.setProto3EnumField(this, 1, value);
};


/**
 * optional google.protobuf.Timestamp started_at = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.SyncPhaseTiming.prototype.getStartedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.SyncPhaseTiming} returns this
*/
proto.dropz.SyncPhaseTiming.prototype.setStartedAt = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.SyncPhaseTiming} returns this
 */
proto.dropz.SyncPhaseTiming.prototype.clearStartedAt = function() {
  return this.setStartedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.SyncPhaseTiming.prototype.hasStartedAt = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional google.protobuf.Timestamp finished_at = 3;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.SyncPhaseTiming.prototype.getFinishedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 3));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.SyncPhaseTiming} returns this
*/
proto.dropz.SyncPhaseTiming.prototype.setFinishedAt = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.SyncPhaseTiming} returns this
 */
proto.dropz.SyncPhaseTiming.prototype.clearFinishedAt = function() {
  return this.setFinishedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.SyncPhaseTiming.prototype.hasFinishedAt = function() {
  return jspb.Message.getField(this, 3) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.SyncQueueEntry.repeatedFields_ = [16,17];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SyncQueueEntry.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SyncQueueEntry.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SyncQueueEntry} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncQueueEntry.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    queuedAt: (f = msg.getQueuedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    priority: jspb.Message.getFieldWithDefault(msg, 3, 0),
    progressPercent: jspb.Message.getFieldWithDefault(msg, 4, 0),
    currentOperation: jspb.Message.getFieldWithDefault(msg, 5, ""),
    fileIndex: jspb.Message.getFieldWithDefault(msg, 6, 0),
    fileCount: jspb.Message.getFieldWithDefault(msg, 7, 0),
    fileName: jspb.Message.getFieldWithDefault(msg, 8, ""),
    fileBytes: jspb.Message.getFieldWithDefault(msg, 9, 0),
    fileTotal: jspb.Message.getFieldWithDefault(msg, 10, 0),
    bytesDone: jspb.Message.getFieldWithDefault(msg, 11, 0),
    bytesTotal: jspb.Message.getFieldWithDefault(msg, 12, 0),
    rateBps: jspb.Message.getFieldWithDefault(msg, 13, 0),
    startedAt: (f = msg.getStartedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    phase: jspb.Message.getFieldWithDefault(msg, 15, 0),
    phasesList: jspb.Message.toObjectList(msg.getPhasesList(),
    proto.dropz.SyncPhaseTiming.toObject, includeInstance),
    filesList: jspb.Message.toObjectList(msg.getFilesList(),
    proto.dropz.SyncFile.toObject, includeInstance),
    stepIndex: jspb.Message.getFieldWithDefault(msg, 18, 0),
    stepCount: jspb.Message.getFieldWithDefault(msg, 19, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SyncQueueEntry}
 */
proto.dropz.SyncQueueEntry.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SyncQueueEntry;
  return proto.dropz.SyncQueueEntry.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SyncQueueEntry} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SyncQueueEntry}
 */
proto.dropz.SyncQueueEntry.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setQueuedAt(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setPriority(value);
      break;
    case 4:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setProgressPercent(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setCurrentOperation(value);
      break;
    case 6:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setFileIndex(value);
      break;
    case 7:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setFileCount(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setFileName(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setFileBytes(value);
      break;
    case 10:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setFileTotal(value);
      break;
    case 11:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setBytesDone(value);
      break;
    case 12:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setBytesTotal(value);
      break;
    case 13:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setRateBps(value);
      break;
    case 14:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setStartedAt(value);
      break;
    case 15:
      var value = /** @type {!proto.dropz.SyncPhase} */ (reader.readEnum());
      msg.setPhase(value);
      break;
    case 16:
      var value = new proto.dropz.SyncPhaseTiming;
      reader.readMessage(value,proto.dropz.SyncPhaseTiming.deserializeBinaryFromReader);
      msg.addPhases(value);
      break;
    case 17:
      var value = new proto.dropz.SyncFile;
      reader.readMessage(value,proto.dropz.SyncFile.deserializeBinaryFromReader);
      msg.addFiles(value);
      break;
    case 18:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setStepIndex(value);
      break;
    case 19:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setStepCount(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SyncQueueEntry.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SyncQueueEntry.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SyncQueueEntry} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncQueueEntry.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getQueuedAt();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getPriority();
  if (f !== 0) {
    writer.writeInt32(
      3,
      f
    );
  }
  f = message.getProgressPercent();
  if (f !== 0) {
    writer.writeInt32(
      4,
      f
    );
  }
  f = message.getCurrentOperation();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getFileIndex();
  if (f !== 0) {
    writer.writeInt32(
      6,
      f
    );
  }
  f = message.getFileCount();
  if (f !== 0) {
    writer.writeInt32(
      7,
      f
    );
  }
  f = message.getFileName();
  if (f.length > 0) {
    writer.writeString(
      8,
      f
    );
  }
  f = message.getFileBytes();
  if (f !== 0) {
    writer.writeInt64(
      9,
      f
    );
  }
  f = message.getFileTotal();
  if (f !== 0) {
    writer.writeInt64(
      10,
      f
    );
  }
  f = message.getBytesDone();
  if (f !== 0) {
    writer.writeInt64(
      11,
      f
    );
  }
  f = message.getBytesTotal();
  if (f !== 0) {
    writer.writeInt64(
      12,
      f
    );
  }
  f = message.getRateBps();
  if (f !== 0) {
    writer.writeInt64(
      13,
      f
    );
  }
  f = message.getStartedAt();
  if (f != null) {
    writer.writeMessage(
      14,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getPhase();
  if (f !== 0.0) {
    writer.writeEnum(
      15,
      f
    );
  }
  f = message.getPhasesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      16,
      f,
      proto.dropz.SyncPhaseTiming.serializeBinaryToWriter
    );
  }
  f = message.getFilesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      17,
      f,
      proto.dropz.SyncFile.serializeBinaryToWriter
    );
  }
  f = message.getStepIndex();
  if (f !== 0) {
    writer.writeInt32(
      18,
      f
    );
  }
  f = message.getStepCount();
  if (f !== 0) {
    writer.writeInt32(
      19,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.SyncQueueEntry.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional google.protobuf.Timestamp queued_at = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.SyncQueueEntry.prototype.getQueuedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
*/
proto.dropz.SyncQueueEntry.prototype.setQueuedAt = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.clearQueuedAt = function() {
  return this.setQueuedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.SyncQueueEntry.prototype.hasQueuedAt = function() {
  return jspb.Message.getField(this, 2) != null;
};


/**
 * optional int32 priority = 3;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getPriority = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setPriority = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional int32 progress_percent = 4;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getProgressPercent = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 4, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setProgressPercent = function(value) {
  return jspb.Message.setProto3IntField(this, 4, value);
};


/**
 * optional string current_operation = 5;
 * @return {string}
 */
proto.dropz.SyncQueueEntry.prototype.getCurrentOperation = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setCurrentOperation = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional int32 file_index = 6;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getFileIndex = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 6, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setFileIndex = function(value) {
  return jspb.Message.setProto3IntField(this, 6, value);
};


/**
 * optional int32 file_count = 7;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getFileCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 7, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setFileCount = function(value) {
  return jspb.Message.setProto3IntField(this, 7, value);
};


/**
 * optional string file_name = 8;
 * @return {string}
 */
proto.dropz.SyncQueueEntry.prototype.getFileName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setFileName = function(value) {
  return jspb.Message.setProto3StringField(this, 8, value);
};


/**
 * optional int64 file_bytes = 9;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getFileBytes = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setFileBytes = function(value) {
  return jspb.Message.setProto3IntField(this, 9, value);
};


/**
 * optional int64 file_total = 10;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getFileTotal = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setFileTotal = function(value) {
  return jspb.Message.setProto3IntField(this, 10, value);
};


/**
 * optional int64 bytes_done = 11;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getBytesDone = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 11, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setBytesDone = function(value) {
  return jspb.Message.setProto3IntField(this, 11, value);
};


/**
 * optional int64 bytes_total = 12;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getBytesTotal = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 12, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setBytesTotal = function(value) {
  return jspb.Message.setProto3IntField(this, 12, value);
};


/**
 * optional int64 rate_bps = 13;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getRateBps = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 13, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setRateBps = function(value) {
  return jspb.Message.setProto3IntField(this, 13, value);
};


/**
 * optional google.protobuf.Timestamp started_at = 14;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.SyncQueueEntry.prototype.getStartedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 14));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
*/
proto.dropz.SyncQueueEntry.prototype.setStartedAt = function(value) {
  return jspb.Message.setWrapperField(this, 14, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.clearStartedAt = function() {
  return this.setStartedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.SyncQueueEntry.prototype.hasStartedAt = function() {
  return jspb.Message.getField(this, 14) != null;
};


/**
 * optional SyncPhase phase = 15;
 * @return {!proto.dropz.SyncPhase}
 */
proto.dropz.SyncQueueEntry.prototype.getPhase = function() {
  return /** @type {!proto.dropz.SyncPhase} */ (jspb.Message.getFieldWithDefault(this, 15, 0));
};


/**
 * @param {!proto.dropz.SyncPhase} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setPhase = function(value) {
  return jspb.Message.setProto3EnumField(this, 15, value);
};


/**
 * repeated SyncPhaseTiming phases = 16;
 * @return {!Array<!proto.dropz.SyncPhaseTiming>}
 */
proto.dropz.SyncQueueEntry.prototype.getPhasesList = function() {
  return /** @type{!Array<!proto.dropz.SyncPhaseTiming>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SyncPhaseTiming, 16));
};


/**
 * @param {!Array<!proto.dropz.SyncPhaseTiming>} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
*/
proto.dropz.SyncQueueEntry.prototype.setPhasesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 16, value);
};


/**
 * @param {!proto.dropz.SyncPhaseTiming=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SyncPhaseTiming}
 */
proto.dropz.SyncQueueEntry.prototype.addPhases = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 16, opt_value, proto.dropz.SyncPhaseTiming, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.clearPhasesList = function() {
  return this.setPhasesList([]);
};


/**
 * repeated SyncFile files = 17;
 * @return {!Array<!proto.dropz.SyncFile>}
 */
proto.dropz.SyncQueueEntry.prototype.getFilesList = function() {
  return /** @type{!Array<!proto.dropz.SyncFile>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SyncFile, 17));
};


/**
 * @param {!Array<!proto.dropz.SyncFile>} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
*/
proto.dropz.SyncQueueEntry.prototype.setFilesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 17, value);
};


/**
 * @param {!proto.dropz.SyncFile=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SyncFile}
 */
proto.dropz.SyncQueueEntry.prototype.addFiles = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 17, opt_value, proto.dropz.SyncFile, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.clearFilesList = function() {
  return this.setFilesList([]);
};


/**
 * optional int32 step_index = 18;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getStepIndex = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 18, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setStepIndex = function(value) {
  return jspb.Message.setProto3IntField(this, 18, value);
};


/**
 * optional int32 step_count = 19;
 * @return {number}
 */
proto.dropz.SyncQueueEntry.prototype.getStepCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 19, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncQueueEntry} returns this
 */
proto.dropz.SyncQueueEntry.prototype.setStepCount = function(value) {
  return jspb.Message.setProto3IntField(this, 19, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.SyncSession.repeatedFields_ = [14,15];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SyncSession.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SyncSession.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SyncSession} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncSession.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    cameraId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    startedAt: (f = msg.getStartedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    finishedAt: (f = msg.getFinishedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    outcome: jspb.Message.getFieldWithDefault(msg, 5, 0),
    error: jspb.Message.getFieldWithDefault(msg, 6, ""),
    failedStep: jspb.Message.getFieldWithDefault(msg, 7, ""),
    stepIndex: jspb.Message.getFieldWithDefault(msg, 8, 0),
    stepCount: jspb.Message.getFieldWithDefault(msg, 9, 0),
    filesDownloaded: jspb.Message.getFieldWithDefault(msg, 10, 0),
    filesFailed: jspb.Message.getFieldWithDefault(msg, 11, 0),
    filesSkipped: jspb.Message.getFieldWithDefault(msg, 12, 0),
    bytesDownloaded: jspb.Message.getFieldWithDefault(msg, 13, 0),
    filesList: jspb.Message.toObjectList(msg.getFilesList(),
    proto.dropz.SyncFile.toObject, includeInstance),
    phasesList: jspb.Message.toObjectList(msg.getPhasesList(),
    proto.dropz.SyncPhaseTiming.toObject, includeInstance),
    selection: jspb.Message.getBooleanFieldWithDefault(msg, 16, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SyncSession}
 */
proto.dropz.SyncSession.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SyncSession;
  return proto.dropz.SyncSession.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SyncSession} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SyncSession}
 */
proto.dropz.SyncSession.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 3:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setStartedAt(value);
      break;
    case 4:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setFinishedAt(value);
      break;
    case 5:
      var value = /** @type {!proto.dropz.SyncOutcome} */ (reader.readEnum());
      msg.setOutcome(value);
      break;
    case 6:
      var value = /** @type {string} */ (reader.readString());
      msg.setError(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setFailedStep(value);
      break;
    case 8:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setStepIndex(value);
      break;
    case 9:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setStepCount(value);
      break;
    case 10:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setFilesDownloaded(value);
      break;
    case 11:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setFilesFailed(value);
      break;
    case 12:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setFilesSkipped(value);
      break;
    case 13:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setBytesDownloaded(value);
      break;
    case 14:
      var value = new proto.dropz.SyncFile;
      reader.readMessage(value,proto.dropz.SyncFile.deserializeBinaryFromReader);
      msg.addFiles(value);
      break;
    case 15:
      var value = new proto.dropz.SyncPhaseTiming;
      reader.readMessage(value,proto.dropz.SyncPhaseTiming.deserializeBinaryFromReader);
      msg.addPhases(value);
      break;
    case 16:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSelection(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SyncSession.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SyncSession.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SyncSession} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SyncSession.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getStartedAt();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getFinishedAt();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getOutcome();
  if (f !== 0.0) {
    writer.writeEnum(
      5,
      f
    );
  }
  f = message.getError();
  if (f.length > 0) {
    writer.writeString(
      6,
      f
    );
  }
  f = message.getFailedStep();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getStepIndex();
  if (f !== 0) {
    writer.writeInt32(
      8,
      f
    );
  }
  f = message.getStepCount();
  if (f !== 0) {
    writer.writeInt32(
      9,
      f
    );
  }
  f = message.getFilesDownloaded();
  if (f !== 0) {
    writer.writeInt32(
      10,
      f
    );
  }
  f = message.getFilesFailed();
  if (f !== 0) {
    writer.writeInt32(
      11,
      f
    );
  }
  f = message.getFilesSkipped();
  if (f !== 0) {
    writer.writeInt32(
      12,
      f
    );
  }
  f = message.getBytesDownloaded();
  if (f !== 0) {
    writer.writeInt64(
      13,
      f
    );
  }
  f = message.getFilesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      14,
      f,
      proto.dropz.SyncFile.serializeBinaryToWriter
    );
  }
  f = message.getPhasesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      15,
      f,
      proto.dropz.SyncPhaseTiming.serializeBinaryToWriter
    );
  }
  f = message.getSelection();
  if (f) {
    writer.writeBool(
      16,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.dropz.SyncSession.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string camera_id = 2;
 * @return {string}
 */
proto.dropz.SyncSession.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional google.protobuf.Timestamp started_at = 3;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.SyncSession.prototype.getStartedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 3));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.SyncSession} returns this
*/
proto.dropz.SyncSession.prototype.setStartedAt = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.clearStartedAt = function() {
  return this.setStartedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.SyncSession.prototype.hasStartedAt = function() {
  return jspb.Message.getField(this, 3) != null;
};


/**
 * optional google.protobuf.Timestamp finished_at = 4;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.SyncSession.prototype.getFinishedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 4));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.SyncSession} returns this
*/
proto.dropz.SyncSession.prototype.setFinishedAt = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.clearFinishedAt = function() {
  return this.setFinishedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.SyncSession.prototype.hasFinishedAt = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional SyncOutcome outcome = 5;
 * @return {!proto.dropz.SyncOutcome}
 */
proto.dropz.SyncSession.prototype.getOutcome = function() {
  return /** @type {!proto.dropz.SyncOutcome} */ (jspb.Message.getFieldWithDefault(this, 5, 0));
};


/**
 * @param {!proto.dropz.SyncOutcome} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setOutcome = function(value) {
  return jspb.Message.setProto3EnumField(this, 5, value);
};


/**
 * optional string error = 6;
 * @return {string}
 */
proto.dropz.SyncSession.prototype.getError = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 6, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setError = function(value) {
  return jspb.Message.setProto3StringField(this, 6, value);
};


/**
 * optional string failed_step = 7;
 * @return {string}
 */
proto.dropz.SyncSession.prototype.getFailedStep = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setFailedStep = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional int32 step_index = 8;
 * @return {number}
 */
proto.dropz.SyncSession.prototype.getStepIndex = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 8, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setStepIndex = function(value) {
  return jspb.Message.setProto3IntField(this, 8, value);
};


/**
 * optional int32 step_count = 9;
 * @return {number}
 */
proto.dropz.SyncSession.prototype.getStepCount = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 9, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setStepCount = function(value) {
  return jspb.Message.setProto3IntField(this, 9, value);
};


/**
 * optional int32 files_downloaded = 10;
 * @return {number}
 */
proto.dropz.SyncSession.prototype.getFilesDownloaded = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 10, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setFilesDownloaded = function(value) {
  return jspb.Message.setProto3IntField(this, 10, value);
};


/**
 * optional int32 files_failed = 11;
 * @return {number}
 */
proto.dropz.SyncSession.prototype.getFilesFailed = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 11, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setFilesFailed = function(value) {
  return jspb.Message.setProto3IntField(this, 11, value);
};


/**
 * optional int32 files_skipped = 12;
 * @return {number}
 */
proto.dropz.SyncSession.prototype.getFilesSkipped = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 12, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setFilesSkipped = function(value) {
  return jspb.Message.setProto3IntField(this, 12, value);
};


/**
 * optional int64 bytes_downloaded = 13;
 * @return {number}
 */
proto.dropz.SyncSession.prototype.getBytesDownloaded = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 13, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setBytesDownloaded = function(value) {
  return jspb.Message.setProto3IntField(this, 13, value);
};


/**
 * repeated SyncFile files = 14;
 * @return {!Array<!proto.dropz.SyncFile>}
 */
proto.dropz.SyncSession.prototype.getFilesList = function() {
  return /** @type{!Array<!proto.dropz.SyncFile>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SyncFile, 14));
};


/**
 * @param {!Array<!proto.dropz.SyncFile>} value
 * @return {!proto.dropz.SyncSession} returns this
*/
proto.dropz.SyncSession.prototype.setFilesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 14, value);
};


/**
 * @param {!proto.dropz.SyncFile=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SyncFile}
 */
proto.dropz.SyncSession.prototype.addFiles = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 14, opt_value, proto.dropz.SyncFile, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.clearFilesList = function() {
  return this.setFilesList([]);
};


/**
 * repeated SyncPhaseTiming phases = 15;
 * @return {!Array<!proto.dropz.SyncPhaseTiming>}
 */
proto.dropz.SyncSession.prototype.getPhasesList = function() {
  return /** @type{!Array<!proto.dropz.SyncPhaseTiming>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SyncPhaseTiming, 15));
};


/**
 * @param {!Array<!proto.dropz.SyncPhaseTiming>} value
 * @return {!proto.dropz.SyncSession} returns this
*/
proto.dropz.SyncSession.prototype.setPhasesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 15, value);
};


/**
 * @param {!proto.dropz.SyncPhaseTiming=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SyncPhaseTiming}
 */
proto.dropz.SyncSession.prototype.addPhases = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 15, opt_value, proto.dropz.SyncPhaseTiming, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.clearPhasesList = function() {
  return this.setPhasesList([]);
};


/**
 * optional bool selection = 16;
 * @return {boolean}
 */
proto.dropz.SyncSession.prototype.getSelection = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 16, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.SyncSession} returns this
 */
proto.dropz.SyncSession.prototype.setSelection = function(value) {
  return jspb.Message.setProto3BooleanField(this, 16, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetSyncHistoryRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetSyncHistoryRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetSyncHistoryRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncHistoryRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    limit: jspb.Message.getFieldWithDefault(msg, 1, 0),
    cameraId: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetSyncHistoryRequest}
 */
proto.dropz.GetSyncHistoryRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetSyncHistoryRequest;
  return proto.dropz.GetSyncHistoryRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetSyncHistoryRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetSyncHistoryRequest}
 */
proto.dropz.GetSyncHistoryRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setLimit(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetSyncHistoryRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetSyncHistoryRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetSyncHistoryRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncHistoryRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getLimit();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional int32 limit = 1;
 * @return {number}
 */
proto.dropz.GetSyncHistoryRequest.prototype.getLimit = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.GetSyncHistoryRequest} returns this
 */
proto.dropz.GetSyncHistoryRequest.prototype.setLimit = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional string camera_id = 2;
 * @return {string}
 */
proto.dropz.GetSyncHistoryRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.GetSyncHistoryRequest} returns this
 */
proto.dropz.GetSyncHistoryRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.GetSyncHistoryResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetSyncHistoryResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetSyncHistoryResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetSyncHistoryResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncHistoryResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    sessionsList: jspb.Message.toObjectList(msg.getSessionsList(),
    proto.dropz.SyncSession.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetSyncHistoryResponse}
 */
proto.dropz.GetSyncHistoryResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetSyncHistoryResponse;
  return proto.dropz.GetSyncHistoryResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetSyncHistoryResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetSyncHistoryResponse}
 */
proto.dropz.GetSyncHistoryResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.SyncSession;
      reader.readMessage(value,proto.dropz.SyncSession.deserializeBinaryFromReader);
      msg.addSessions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetSyncHistoryResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetSyncHistoryResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetSyncHistoryResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncHistoryResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSessionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.SyncSession.serializeBinaryToWriter
    );
  }
};


/**
 * repeated SyncSession sessions = 1;
 * @return {!Array<!proto.dropz.SyncSession>}
 */
proto.dropz.GetSyncHistoryResponse.prototype.getSessionsList = function() {
  return /** @type{!Array<!proto.dropz.SyncSession>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SyncSession, 1));
};


/**
 * @param {!Array<!proto.dropz.SyncSession>} value
 * @return {!proto.dropz.GetSyncHistoryResponse} returns this
*/
proto.dropz.GetSyncHistoryResponse.prototype.setSessionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.SyncSession=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SyncSession}
 */
proto.dropz.GetSyncHistoryResponse.prototype.addSessions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.SyncSession, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.GetSyncHistoryResponse} returns this
 */
proto.dropz.GetSyncHistoryResponse.prototype.clearSessionsList = function() {
  return this.setSessionsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SetCameraAliasRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SetCameraAliasRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SetCameraAliasRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetCameraAliasRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    alias: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SetCameraAliasRequest}
 */
proto.dropz.SetCameraAliasRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SetCameraAliasRequest;
  return proto.dropz.SetCameraAliasRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SetCameraAliasRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SetCameraAliasRequest}
 */
proto.dropz.SetCameraAliasRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setAlias(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SetCameraAliasRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SetCameraAliasRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SetCameraAliasRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetCameraAliasRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getAlias();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.SetCameraAliasRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SetCameraAliasRequest} returns this
 */
proto.dropz.SetCameraAliasRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string alias = 2;
 * @return {string}
 */
proto.dropz.SetCameraAliasRequest.prototype.getAlias = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SetCameraAliasRequest} returns this
 */
proto.dropz.SetCameraAliasRequest.prototype.setAlias = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SetCameraAliasResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SetCameraAliasResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SetCameraAliasResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetCameraAliasResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    camera: (f = msg.getCamera()) && proto.dropz.CameraWithState.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SetCameraAliasResponse}
 */
proto.dropz.SetCameraAliasResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SetCameraAliasResponse;
  return proto.dropz.SetCameraAliasResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SetCameraAliasResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SetCameraAliasResponse}
 */
proto.dropz.SetCameraAliasResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.CameraWithState;
      reader.readMessage(value,proto.dropz.CameraWithState.deserializeBinaryFromReader);
      msg.setCamera(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SetCameraAliasResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SetCameraAliasResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SetCameraAliasResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetCameraAliasResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCamera();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.dropz.CameraWithState.serializeBinaryToWriter
    );
  }
};


/**
 * optional CameraWithState camera = 1;
 * @return {?proto.dropz.CameraWithState}
 */
proto.dropz.SetCameraAliasResponse.prototype.getCamera = function() {
  return /** @type{?proto.dropz.CameraWithState} */ (
    jspb.Message.getWrapperField(this, proto.dropz.CameraWithState, 1));
};


/**
 * @param {?proto.dropz.CameraWithState|undefined} value
 * @return {!proto.dropz.SetCameraAliasResponse} returns this
*/
proto.dropz.SetCameraAliasResponse.prototype.setCamera = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.SetCameraAliasResponse} returns this
 */
proto.dropz.SetCameraAliasResponse.prototype.clearCamera = function() {
  return this.setCamera(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.SetCameraAliasResponse.prototype.hasCamera = function() {
  return jspb.Message.getField(this, 1) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.Group.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.Group.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.Group.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.Group} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.Group.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, ""),
    name: jspb.Message.getFieldWithDefault(msg, 2, ""),
    cameraIdsList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f,
    createdAt: (f = msg.getCreatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    updatedAt: (f = msg.getUpdatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    syncPaused: jspb.Message.getBooleanFieldWithDefault(msg, 6, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.Group}
 */
proto.dropz.Group.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.Group;
  return proto.dropz.Group.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.Group} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.Group}
 */
proto.dropz.Group.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.addCameraIds(value);
      break;
    case 4:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setCreatedAt(value);
      break;
    case 5:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setUpdatedAt(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSyncPaused(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.Group.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.Group.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.Group} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.Group.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getCameraIdsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      3,
      f
    );
  }
  f = message.getCreatedAt();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getUpdatedAt();
  if (f != null) {
    writer.writeMessage(
      5,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getSyncPaused();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
};


/**
 * optional string id = 1;
 * @return {string}
 */
proto.dropz.Group.prototype.getId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.setId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.dropz.Group.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated string camera_ids = 3;
 * @return {!Array<string>}
 */
proto.dropz.Group.prototype.getCameraIdsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.setCameraIdsList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.addCameraIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.clearCameraIdsList = function() {
  return this.setCameraIdsList([]);
};


/**
 * optional google.protobuf.Timestamp created_at = 4;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.Group.prototype.getCreatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 4));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.Group} returns this
*/
proto.dropz.Group.prototype.setCreatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.clearCreatedAt = function() {
  return this.setCreatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.Group.prototype.hasCreatedAt = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional google.protobuf.Timestamp updated_at = 5;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.Group.prototype.getUpdatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 5));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.Group} returns this
*/
proto.dropz.Group.prototype.setUpdatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 5, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.clearUpdatedAt = function() {
  return this.setUpdatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.Group.prototype.hasUpdatedAt = function() {
  return jspb.Message.getField(this, 5) != null;
};


/**
 * optional bool sync_paused = 6;
 * @return {boolean}
 */
proto.dropz.Group.prototype.getSyncPaused = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.Group} returns this
 */
proto.dropz.Group.prototype.setSyncPaused = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.MoveCamerasToGroupRequest.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.MoveCamerasToGroupRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.MoveCamerasToGroupRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.MoveCamerasToGroupRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraIdsList: (f = jspb.Message.getRepeatedField(msg, 1)) == null ? undefined : f,
    groupId: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.MoveCamerasToGroupRequest}
 */
proto.dropz.MoveCamerasToGroupRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.MoveCamerasToGroupRequest;
  return proto.dropz.MoveCamerasToGroupRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.MoveCamerasToGroupRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.MoveCamerasToGroupRequest}
 */
proto.dropz.MoveCamerasToGroupRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.addCameraIds(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setGroupId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.MoveCamerasToGroupRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.MoveCamerasToGroupRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.MoveCamerasToGroupRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraIdsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      1,
      f
    );
  }
  f = message.getGroupId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * repeated string camera_ids = 1;
 * @return {!Array<string>}
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.getCameraIdsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 1));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.dropz.MoveCamerasToGroupRequest} returns this
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.setCameraIdsList = function(value) {
  return jspb.Message.setField(this, 1, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.dropz.MoveCamerasToGroupRequest} returns this
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.addCameraIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 1, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.MoveCamerasToGroupRequest} returns this
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.clearCameraIdsList = function() {
  return this.setCameraIdsList([]);
};


/**
 * optional string group_id = 2;
 * @return {string}
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.getGroupId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.MoveCamerasToGroupRequest} returns this
 */
proto.dropz.MoveCamerasToGroupRequest.prototype.setGroupId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SetGroupSyncRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SetGroupSyncRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SetGroupSyncRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetGroupSyncRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    groupId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    paused: jspb.Message.getBooleanFieldWithDefault(msg, 2, false),
    exclusive: jspb.Message.getBooleanFieldWithDefault(msg, 3, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SetGroupSyncRequest}
 */
proto.dropz.SetGroupSyncRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SetGroupSyncRequest;
  return proto.dropz.SetGroupSyncRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SetGroupSyncRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SetGroupSyncRequest}
 */
proto.dropz.SetGroupSyncRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGroupId(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setPaused(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setExclusive(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SetGroupSyncRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SetGroupSyncRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SetGroupSyncRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetGroupSyncRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGroupId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getPaused();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
  f = message.getExclusive();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
};


/**
 * optional string group_id = 1;
 * @return {string}
 */
proto.dropz.SetGroupSyncRequest.prototype.getGroupId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SetGroupSyncRequest} returns this
 */
proto.dropz.SetGroupSyncRequest.prototype.setGroupId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional bool paused = 2;
 * @return {boolean}
 */
proto.dropz.SetGroupSyncRequest.prototype.getPaused = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.SetGroupSyncRequest} returns this
 */
proto.dropz.SetGroupSyncRequest.prototype.setPaused = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};


/**
 * optional bool exclusive = 3;
 * @return {boolean}
 */
proto.dropz.SetGroupSyncRequest.prototype.getExclusive = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.SetGroupSyncRequest} returns this
 */
proto.dropz.SetGroupSyncRequest.prototype.setExclusive = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetDiscoveredCamerasRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetDiscoveredCamerasRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetDiscoveredCamerasRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetDiscoveredCamerasRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    changeCounter: jspb.Message.getFieldWithDefault(msg, 1, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetDiscoveredCamerasRequest}
 */
proto.dropz.GetDiscoveredCamerasRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetDiscoveredCamerasRequest;
  return proto.dropz.GetDiscoveredCamerasRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetDiscoveredCamerasRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetDiscoveredCamerasRequest}
 */
proto.dropz.GetDiscoveredCamerasRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setChangeCounter(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetDiscoveredCamerasRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetDiscoveredCamerasRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetDiscoveredCamerasRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetDiscoveredCamerasRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getChangeCounter();
  if (f !== 0) {
    writer.writeInt64(
      1,
      f
    );
  }
};


/**
 * optional int64 change_counter = 1;
 * @return {number}
 */
proto.dropz.GetDiscoveredCamerasRequest.prototype.getChangeCounter = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.GetDiscoveredCamerasRequest} returns this
 */
proto.dropz.GetDiscoveredCamerasRequest.prototype.setChangeCounter = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.GetDiscoveredCamerasResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetDiscoveredCamerasResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetDiscoveredCamerasResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetDiscoveredCamerasResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    camerasList: jspb.Message.toObjectList(msg.getCamerasList(),
    proto.dropz.DiscoveredCamera.toObject, includeInstance),
    changeCounter: jspb.Message.getFieldWithDefault(msg, 2, 0),
    noChanges: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    heartbeat: jspb.Message.getBooleanFieldWithDefault(msg, 4, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetDiscoveredCamerasResponse}
 */
proto.dropz.GetDiscoveredCamerasResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetDiscoveredCamerasResponse;
  return proto.dropz.GetDiscoveredCamerasResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetDiscoveredCamerasResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetDiscoveredCamerasResponse}
 */
proto.dropz.GetDiscoveredCamerasResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.DiscoveredCamera;
      reader.readMessage(value,proto.dropz.DiscoveredCamera.deserializeBinaryFromReader);
      msg.addCameras(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setChangeCounter(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setNoChanges(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setHeartbeat(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetDiscoveredCamerasResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetDiscoveredCamerasResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetDiscoveredCamerasResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCamerasList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.DiscoveredCamera.serializeBinaryToWriter
    );
  }
  f = message.getChangeCounter();
  if (f !== 0) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = message.getNoChanges();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getHeartbeat();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * repeated DiscoveredCamera cameras = 1;
 * @return {!Array<!proto.dropz.DiscoveredCamera>}
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.getCamerasList = function() {
  return /** @type{!Array<!proto.dropz.DiscoveredCamera>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.DiscoveredCamera, 1));
};


/**
 * @param {!Array<!proto.dropz.DiscoveredCamera>} value
 * @return {!proto.dropz.GetDiscoveredCamerasResponse} returns this
*/
proto.dropz.GetDiscoveredCamerasResponse.prototype.setCamerasList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.DiscoveredCamera=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.DiscoveredCamera}
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.addCameras = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.DiscoveredCamera, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.GetDiscoveredCamerasResponse} returns this
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.clearCamerasList = function() {
  return this.setCamerasList([]);
};


/**
 * optional int64 change_counter = 2;
 * @return {number}
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.getChangeCounter = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.GetDiscoveredCamerasResponse} returns this
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.setChangeCounter = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional bool no_changes = 3;
 * @return {boolean}
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.getNoChanges = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.GetDiscoveredCamerasResponse} returns this
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.setNoChanges = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool heartbeat = 4;
 * @return {boolean}
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.getHeartbeat = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.GetDiscoveredCamerasResponse} returns this
 */
proto.dropz.GetDiscoveredCamerasResponse.prototype.setHeartbeat = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetManagedCamerasRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetManagedCamerasRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetManagedCamerasRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetManagedCamerasRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    changeCounter: jspb.Message.getFieldWithDefault(msg, 1, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetManagedCamerasRequest}
 */
proto.dropz.GetManagedCamerasRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetManagedCamerasRequest;
  return proto.dropz.GetManagedCamerasRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetManagedCamerasRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetManagedCamerasRequest}
 */
proto.dropz.GetManagedCamerasRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setChangeCounter(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetManagedCamerasRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetManagedCamerasRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetManagedCamerasRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetManagedCamerasRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getChangeCounter();
  if (f !== 0) {
    writer.writeInt64(
      1,
      f
    );
  }
};


/**
 * optional int64 change_counter = 1;
 * @return {number}
 */
proto.dropz.GetManagedCamerasRequest.prototype.getChangeCounter = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.GetManagedCamerasRequest} returns this
 */
proto.dropz.GetManagedCamerasRequest.prototype.setChangeCounter = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.GetManagedCamerasResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetManagedCamerasResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetManagedCamerasResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetManagedCamerasResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetManagedCamerasResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    camerasList: jspb.Message.toObjectList(msg.getCamerasList(),
    proto.dropz.ManagedCamera.toObject, includeInstance),
    changeCounter: jspb.Message.getFieldWithDefault(msg, 2, 0),
    noChanges: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    heartbeat: jspb.Message.getBooleanFieldWithDefault(msg, 4, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetManagedCamerasResponse}
 */
proto.dropz.GetManagedCamerasResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetManagedCamerasResponse;
  return proto.dropz.GetManagedCamerasResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetManagedCamerasResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetManagedCamerasResponse}
 */
proto.dropz.GetManagedCamerasResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.ManagedCamera;
      reader.readMessage(value,proto.dropz.ManagedCamera.deserializeBinaryFromReader);
      msg.addCameras(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setChangeCounter(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setNoChanges(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setHeartbeat(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetManagedCamerasResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetManagedCamerasResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetManagedCamerasResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetManagedCamerasResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCamerasList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.ManagedCamera.serializeBinaryToWriter
    );
  }
  f = message.getChangeCounter();
  if (f !== 0) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = message.getNoChanges();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getHeartbeat();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * repeated ManagedCamera cameras = 1;
 * @return {!Array<!proto.dropz.ManagedCamera>}
 */
proto.dropz.GetManagedCamerasResponse.prototype.getCamerasList = function() {
  return /** @type{!Array<!proto.dropz.ManagedCamera>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.ManagedCamera, 1));
};


/**
 * @param {!Array<!proto.dropz.ManagedCamera>} value
 * @return {!proto.dropz.GetManagedCamerasResponse} returns this
*/
proto.dropz.GetManagedCamerasResponse.prototype.setCamerasList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.ManagedCamera=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.ManagedCamera}
 */
proto.dropz.GetManagedCamerasResponse.prototype.addCameras = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.ManagedCamera, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.GetManagedCamerasResponse} returns this
 */
proto.dropz.GetManagedCamerasResponse.prototype.clearCamerasList = function() {
  return this.setCamerasList([]);
};


/**
 * optional int64 change_counter = 2;
 * @return {number}
 */
proto.dropz.GetManagedCamerasResponse.prototype.getChangeCounter = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.GetManagedCamerasResponse} returns this
 */
proto.dropz.GetManagedCamerasResponse.prototype.setChangeCounter = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional bool no_changes = 3;
 * @return {boolean}
 */
proto.dropz.GetManagedCamerasResponse.prototype.getNoChanges = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.GetManagedCamerasResponse} returns this
 */
proto.dropz.GetManagedCamerasResponse.prototype.setNoChanges = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool heartbeat = 4;
 * @return {boolean}
 */
proto.dropz.GetManagedCamerasResponse.prototype.getHeartbeat = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.GetManagedCamerasResponse} returns this
 */
proto.dropz.GetManagedCamerasResponse.prototype.setHeartbeat = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetSyncQueueRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetSyncQueueRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetSyncQueueRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncQueueRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    changeCounter: jspb.Message.getFieldWithDefault(msg, 1, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetSyncQueueRequest}
 */
proto.dropz.GetSyncQueueRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetSyncQueueRequest;
  return proto.dropz.GetSyncQueueRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetSyncQueueRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetSyncQueueRequest}
 */
proto.dropz.GetSyncQueueRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setChangeCounter(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetSyncQueueRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetSyncQueueRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetSyncQueueRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncQueueRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getChangeCounter();
  if (f !== 0) {
    writer.writeInt64(
      1,
      f
    );
  }
};


/**
 * optional int64 change_counter = 1;
 * @return {number}
 */
proto.dropz.GetSyncQueueRequest.prototype.getChangeCounter = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.GetSyncQueueRequest} returns this
 */
proto.dropz.GetSyncQueueRequest.prototype.setChangeCounter = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.GetSyncQueueResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetSyncQueueResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetSyncQueueResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetSyncQueueResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncQueueResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    queueList: jspb.Message.toObjectList(msg.getQueueList(),
    proto.dropz.SyncQueueEntry.toObject, includeInstance),
    changeCounter: jspb.Message.getFieldWithDefault(msg, 2, 0),
    noChanges: jspb.Message.getBooleanFieldWithDefault(msg, 3, false),
    heartbeat: jspb.Message.getBooleanFieldWithDefault(msg, 4, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetSyncQueueResponse}
 */
proto.dropz.GetSyncQueueResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetSyncQueueResponse;
  return proto.dropz.GetSyncQueueResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetSyncQueueResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetSyncQueueResponse}
 */
proto.dropz.GetSyncQueueResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.SyncQueueEntry;
      reader.readMessage(value,proto.dropz.SyncQueueEntry.deserializeBinaryFromReader);
      msg.addQueue(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setChangeCounter(value);
      break;
    case 3:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setNoChanges(value);
      break;
    case 4:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setHeartbeat(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetSyncQueueResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetSyncQueueResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetSyncQueueResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetSyncQueueResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getQueueList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.SyncQueueEntry.serializeBinaryToWriter
    );
  }
  f = message.getChangeCounter();
  if (f !== 0) {
    writer.writeInt64(
      2,
      f
    );
  }
  f = message.getNoChanges();
  if (f) {
    writer.writeBool(
      3,
      f
    );
  }
  f = message.getHeartbeat();
  if (f) {
    writer.writeBool(
      4,
      f
    );
  }
};


/**
 * repeated SyncQueueEntry queue = 1;
 * @return {!Array<!proto.dropz.SyncQueueEntry>}
 */
proto.dropz.GetSyncQueueResponse.prototype.getQueueList = function() {
  return /** @type{!Array<!proto.dropz.SyncQueueEntry>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SyncQueueEntry, 1));
};


/**
 * @param {!Array<!proto.dropz.SyncQueueEntry>} value
 * @return {!proto.dropz.GetSyncQueueResponse} returns this
*/
proto.dropz.GetSyncQueueResponse.prototype.setQueueList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.SyncQueueEntry=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SyncQueueEntry}
 */
proto.dropz.GetSyncQueueResponse.prototype.addQueue = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.SyncQueueEntry, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.GetSyncQueueResponse} returns this
 */
proto.dropz.GetSyncQueueResponse.prototype.clearQueueList = function() {
  return this.setQueueList([]);
};


/**
 * optional int64 change_counter = 2;
 * @return {number}
 */
proto.dropz.GetSyncQueueResponse.prototype.getChangeCounter = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.GetSyncQueueResponse} returns this
 */
proto.dropz.GetSyncQueueResponse.prototype.setChangeCounter = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};


/**
 * optional bool no_changes = 3;
 * @return {boolean}
 */
proto.dropz.GetSyncQueueResponse.prototype.getNoChanges = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 3, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.GetSyncQueueResponse} returns this
 */
proto.dropz.GetSyncQueueResponse.prototype.setNoChanges = function(value) {
  return jspb.Message.setProto3BooleanField(this, 3, value);
};


/**
 * optional bool heartbeat = 4;
 * @return {boolean}
 */
proto.dropz.GetSyncQueueResponse.prototype.getHeartbeat = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 4, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.GetSyncQueueResponse} returns this
 */
proto.dropz.GetSyncQueueResponse.prototype.setHeartbeat = function(value) {
  return jspb.Message.setProto3BooleanField(this, 4, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ManageCameraRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ManageCameraRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ManageCameraRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ManageCameraRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ManageCameraRequest}
 */
proto.dropz.ManageCameraRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ManageCameraRequest;
  return proto.dropz.ManageCameraRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ManageCameraRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ManageCameraRequest}
 */
proto.dropz.ManageCameraRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ManageCameraRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ManageCameraRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ManageCameraRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ManageCameraRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.ManageCameraRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ManageCameraRequest} returns this
 */
proto.dropz.ManageCameraRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ManageCameraResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ManageCameraResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ManageCameraResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ManageCameraResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    message: jspb.Message.getFieldWithDefault(msg, 2, ""),
    camera: (f = msg.getCamera()) && proto.dropz.ManagedCamera.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ManageCameraResponse}
 */
proto.dropz.ManageCameraResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ManageCameraResponse;
  return proto.dropz.ManageCameraResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ManageCameraResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ManageCameraResponse}
 */
proto.dropz.ManageCameraResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    case 3:
      var value = new proto.dropz.ManagedCamera;
      reader.readMessage(value,proto.dropz.ManagedCamera.deserializeBinaryFromReader);
      msg.setCamera(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ManageCameraResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ManageCameraResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ManageCameraResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ManageCameraResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getCamera();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.dropz.ManagedCamera.serializeBinaryToWriter
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.dropz.ManageCameraResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.ManageCameraResponse} returns this
 */
proto.dropz.ManageCameraResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string message = 2;
 * @return {string}
 */
proto.dropz.ManageCameraResponse.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ManageCameraResponse} returns this
 */
proto.dropz.ManageCameraResponse.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional ManagedCamera camera = 3;
 * @return {?proto.dropz.ManagedCamera}
 */
proto.dropz.ManageCameraResponse.prototype.getCamera = function() {
  return /** @type{?proto.dropz.ManagedCamera} */ (
    jspb.Message.getWrapperField(this, proto.dropz.ManagedCamera, 3));
};


/**
 * @param {?proto.dropz.ManagedCamera|undefined} value
 * @return {!proto.dropz.ManageCameraResponse} returns this
*/
proto.dropz.ManageCameraResponse.prototype.setCamera = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.ManageCameraResponse} returns this
 */
proto.dropz.ManageCameraResponse.prototype.clearCamera = function() {
  return this.setCamera(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.ManageCameraResponse.prototype.hasCamera = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.UnmanageCameraRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.UnmanageCameraRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.UnmanageCameraRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.UnmanageCameraRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.UnmanageCameraRequest}
 */
proto.dropz.UnmanageCameraRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.UnmanageCameraRequest;
  return proto.dropz.UnmanageCameraRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.UnmanageCameraRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.UnmanageCameraRequest}
 */
proto.dropz.UnmanageCameraRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.UnmanageCameraRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.UnmanageCameraRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.UnmanageCameraRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.UnmanageCameraRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.UnmanageCameraRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.UnmanageCameraRequest} returns this
 */
proto.dropz.UnmanageCameraRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.UnmanageCameraResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.UnmanageCameraResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.UnmanageCameraResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.UnmanageCameraResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    message: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.UnmanageCameraResponse}
 */
proto.dropz.UnmanageCameraResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.UnmanageCameraResponse;
  return proto.dropz.UnmanageCameraResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.UnmanageCameraResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.UnmanageCameraResponse}
 */
proto.dropz.UnmanageCameraResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.UnmanageCameraResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.UnmanageCameraResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.UnmanageCameraResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.UnmanageCameraResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.dropz.UnmanageCameraResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.UnmanageCameraResponse} returns this
 */
proto.dropz.UnmanageCameraResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string message = 2;
 * @return {string}
 */
proto.dropz.UnmanageCameraResponse.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.UnmanageCameraResponse} returns this
 */
proto.dropz.UnmanageCameraResponse.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.PairCameraRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.PairCameraRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.PairCameraRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PairCameraRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.PairCameraRequest}
 */
proto.dropz.PairCameraRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.PairCameraRequest;
  return proto.dropz.PairCameraRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.PairCameraRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.PairCameraRequest}
 */
proto.dropz.PairCameraRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.PairCameraRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.PairCameraRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.PairCameraRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PairCameraRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.PairCameraRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.PairCameraRequest} returns this
 */
proto.dropz.PairCameraRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.PairCameraResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.PairCameraResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.PairCameraResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PairCameraResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    message: jspb.Message.getFieldWithDefault(msg, 2, ""),
    camera: (f = msg.getCamera()) && proto.dropz.ManagedCamera.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.PairCameraResponse}
 */
proto.dropz.PairCameraResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.PairCameraResponse;
  return proto.dropz.PairCameraResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.PairCameraResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.PairCameraResponse}
 */
proto.dropz.PairCameraResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    case 3:
      var value = new proto.dropz.ManagedCamera;
      reader.readMessage(value,proto.dropz.ManagedCamera.deserializeBinaryFromReader);
      msg.setCamera(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.PairCameraResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.PairCameraResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.PairCameraResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PairCameraResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getCamera();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.dropz.ManagedCamera.serializeBinaryToWriter
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.dropz.PairCameraResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.PairCameraResponse} returns this
 */
proto.dropz.PairCameraResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string message = 2;
 * @return {string}
 */
proto.dropz.PairCameraResponse.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.PairCameraResponse} returns this
 */
proto.dropz.PairCameraResponse.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional ManagedCamera camera = 3;
 * @return {?proto.dropz.ManagedCamera}
 */
proto.dropz.PairCameraResponse.prototype.getCamera = function() {
  return /** @type{?proto.dropz.ManagedCamera} */ (
    jspb.Message.getWrapperField(this, proto.dropz.ManagedCamera, 3));
};


/**
 * @param {?proto.dropz.ManagedCamera|undefined} value
 * @return {!proto.dropz.PairCameraResponse} returns this
*/
proto.dropz.PairCameraResponse.prototype.setCamera = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.PairCameraResponse} returns this
 */
proto.dropz.PairCameraResponse.prototype.clearCamera = function() {
  return this.setCamera(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.PairCameraResponse.prototype.hasCamera = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ForgetCameraRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ForgetCameraRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ForgetCameraRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForgetCameraRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ForgetCameraRequest}
 */
proto.dropz.ForgetCameraRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ForgetCameraRequest;
  return proto.dropz.ForgetCameraRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ForgetCameraRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ForgetCameraRequest}
 */
proto.dropz.ForgetCameraRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ForgetCameraRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ForgetCameraRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ForgetCameraRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForgetCameraRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.ForgetCameraRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ForgetCameraRequest} returns this
 */
proto.dropz.ForgetCameraRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ForgetCameraResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ForgetCameraResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ForgetCameraResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForgetCameraResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    message: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ForgetCameraResponse}
 */
proto.dropz.ForgetCameraResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ForgetCameraResponse;
  return proto.dropz.ForgetCameraResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ForgetCameraResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ForgetCameraResponse}
 */
proto.dropz.ForgetCameraResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ForgetCameraResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ForgetCameraResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ForgetCameraResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForgetCameraResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.dropz.ForgetCameraResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.ForgetCameraResponse} returns this
 */
proto.dropz.ForgetCameraResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string message = 2;
 * @return {string}
 */
proto.dropz.ForgetCameraResponse.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ForgetCameraResponse} returns this
 */
proto.dropz.ForgetCameraResponse.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ForceSyncRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ForceSyncRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ForceSyncRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForceSyncRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ForceSyncRequest}
 */
proto.dropz.ForceSyncRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ForceSyncRequest;
  return proto.dropz.ForceSyncRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ForceSyncRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ForceSyncRequest}
 */
proto.dropz.ForceSyncRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ForceSyncRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ForceSyncRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ForceSyncRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForceSyncRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.ForceSyncRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ForceSyncRequest} returns this
 */
proto.dropz.ForceSyncRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ForceSyncResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ForceSyncResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ForceSyncResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForceSyncResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    message: jspb.Message.getFieldWithDefault(msg, 2, ""),
    queueEntry: (f = msg.getQueueEntry()) && proto.dropz.SyncQueueEntry.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ForceSyncResponse}
 */
proto.dropz.ForceSyncResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ForceSyncResponse;
  return proto.dropz.ForceSyncResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ForceSyncResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ForceSyncResponse}
 */
proto.dropz.ForceSyncResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    case 3:
      var value = new proto.dropz.SyncQueueEntry;
      reader.readMessage(value,proto.dropz.SyncQueueEntry.deserializeBinaryFromReader);
      msg.setQueueEntry(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ForceSyncResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ForceSyncResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ForceSyncResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ForceSyncResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getQueueEntry();
  if (f != null) {
    writer.writeMessage(
      3,
      f,
      proto.dropz.SyncQueueEntry.serializeBinaryToWriter
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.dropz.ForceSyncResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.ForceSyncResponse} returns this
 */
proto.dropz.ForceSyncResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string message = 2;
 * @return {string}
 */
proto.dropz.ForceSyncResponse.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ForceSyncResponse} returns this
 */
proto.dropz.ForceSyncResponse.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional SyncQueueEntry queue_entry = 3;
 * @return {?proto.dropz.SyncQueueEntry}
 */
proto.dropz.ForceSyncResponse.prototype.getQueueEntry = function() {
  return /** @type{?proto.dropz.SyncQueueEntry} */ (
    jspb.Message.getWrapperField(this, proto.dropz.SyncQueueEntry, 3));
};


/**
 * @param {?proto.dropz.SyncQueueEntry|undefined} value
 * @return {!proto.dropz.ForceSyncResponse} returns this
*/
proto.dropz.ForceSyncResponse.prototype.setQueueEntry = function(value) {
  return jspb.Message.setWrapperField(this, 3, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.ForceSyncResponse} returns this
 */
proto.dropz.ForceSyncResponse.prototype.clearQueueEntry = function() {
  return this.setQueueEntry(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.ForceSyncResponse.prototype.hasQueueEntry = function() {
  return jspb.Message.getField(this, 3) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CancelSyncRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CancelSyncRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CancelSyncRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CancelSyncRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CancelSyncRequest}
 */
proto.dropz.CancelSyncRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CancelSyncRequest;
  return proto.dropz.CancelSyncRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CancelSyncRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CancelSyncRequest}
 */
proto.dropz.CancelSyncRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CancelSyncRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CancelSyncRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CancelSyncRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CancelSyncRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.CancelSyncRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CancelSyncRequest} returns this
 */
proto.dropz.CancelSyncRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CancelSyncResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CancelSyncResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CancelSyncResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CancelSyncResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    message: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CancelSyncResponse}
 */
proto.dropz.CancelSyncResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CancelSyncResponse;
  return proto.dropz.CancelSyncResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CancelSyncResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CancelSyncResponse}
 */
proto.dropz.CancelSyncResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CancelSyncResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CancelSyncResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CancelSyncResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CancelSyncResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.dropz.CancelSyncResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CancelSyncResponse} returns this
 */
proto.dropz.CancelSyncResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string message = 2;
 * @return {string}
 */
proto.dropz.CancelSyncResponse.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CancelSyncResponse} returns this
 */
proto.dropz.CancelSyncResponse.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetGroupsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetGroupsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetGroupsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetGroupsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetGroupsRequest}
 */
proto.dropz.GetGroupsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetGroupsRequest;
  return proto.dropz.GetGroupsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetGroupsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetGroupsRequest}
 */
proto.dropz.GetGroupsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetGroupsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetGroupsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetGroupsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetGroupsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.GetGroupsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetGroupsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetGroupsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetGroupsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetGroupsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    groupsList: jspb.Message.toObjectList(msg.getGroupsList(),
    proto.dropz.Group.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetGroupsResponse}
 */
proto.dropz.GetGroupsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetGroupsResponse;
  return proto.dropz.GetGroupsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetGroupsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetGroupsResponse}
 */
proto.dropz.GetGroupsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.Group;
      reader.readMessage(value,proto.dropz.Group.deserializeBinaryFromReader);
      msg.addGroups(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetGroupsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetGroupsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetGroupsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetGroupsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGroupsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.Group.serializeBinaryToWriter
    );
  }
};


/**
 * repeated Group groups = 1;
 * @return {!Array<!proto.dropz.Group>}
 */
proto.dropz.GetGroupsResponse.prototype.getGroupsList = function() {
  return /** @type{!Array<!proto.dropz.Group>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.Group, 1));
};


/**
 * @param {!Array<!proto.dropz.Group>} value
 * @return {!proto.dropz.GetGroupsResponse} returns this
*/
proto.dropz.GetGroupsResponse.prototype.setGroupsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.Group=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.Group}
 */
proto.dropz.GetGroupsResponse.prototype.addGroups = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.Group, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.GetGroupsResponse} returns this
 */
proto.dropz.GetGroupsResponse.prototype.clearGroupsList = function() {
  return this.setGroupsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.CreateGroupRequest.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CreateGroupRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CreateGroupRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CreateGroupRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CreateGroupRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    cameraIdsList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CreateGroupRequest}
 */
proto.dropz.CreateGroupRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CreateGroupRequest;
  return proto.dropz.CreateGroupRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CreateGroupRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CreateGroupRequest}
 */
proto.dropz.CreateGroupRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addCameraIds(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CreateGroupRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CreateGroupRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CreateGroupRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CreateGroupRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCameraIdsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.dropz.CreateGroupRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CreateGroupRequest} returns this
 */
proto.dropz.CreateGroupRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated string camera_ids = 2;
 * @return {!Array<string>}
 */
proto.dropz.CreateGroupRequest.prototype.getCameraIdsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.dropz.CreateGroupRequest} returns this
 */
proto.dropz.CreateGroupRequest.prototype.setCameraIdsList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.dropz.CreateGroupRequest} returns this
 */
proto.dropz.CreateGroupRequest.prototype.addCameraIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.CreateGroupRequest} returns this
 */
proto.dropz.CreateGroupRequest.prototype.clearCameraIdsList = function() {
  return this.setCameraIdsList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.UpdateGroupRequest.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.UpdateGroupRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.UpdateGroupRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.UpdateGroupRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.UpdateGroupRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    groupId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    name: jspb.Message.getFieldWithDefault(msg, 2, ""),
    cameraIdsList: (f = jspb.Message.getRepeatedField(msg, 3)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.UpdateGroupRequest}
 */
proto.dropz.UpdateGroupRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.UpdateGroupRequest;
  return proto.dropz.UpdateGroupRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.UpdateGroupRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.UpdateGroupRequest}
 */
proto.dropz.UpdateGroupRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGroupId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 3:
      var value = /** @type {string} */ (reader.readString());
      msg.addCameraIds(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.UpdateGroupRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.UpdateGroupRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.UpdateGroupRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.UpdateGroupRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGroupId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getCameraIdsList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      3,
      f
    );
  }
};


/**
 * optional string group_id = 1;
 * @return {string}
 */
proto.dropz.UpdateGroupRequest.prototype.getGroupId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.UpdateGroupRequest} returns this
 */
proto.dropz.UpdateGroupRequest.prototype.setGroupId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.dropz.UpdateGroupRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.UpdateGroupRequest} returns this
 */
proto.dropz.UpdateGroupRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated string camera_ids = 3;
 * @return {!Array<string>}
 */
proto.dropz.UpdateGroupRequest.prototype.getCameraIdsList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 3));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.dropz.UpdateGroupRequest} returns this
 */
proto.dropz.UpdateGroupRequest.prototype.setCameraIdsList = function(value) {
  return jspb.Message.setField(this, 3, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.dropz.UpdateGroupRequest} returns this
 */
proto.dropz.UpdateGroupRequest.prototype.addCameraIds = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 3, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.UpdateGroupRequest} returns this
 */
proto.dropz.UpdateGroupRequest.prototype.clearCameraIdsList = function() {
  return this.setCameraIdsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.DeleteGroupRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.DeleteGroupRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.DeleteGroupRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.DeleteGroupRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    groupId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.DeleteGroupRequest}
 */
proto.dropz.DeleteGroupRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.DeleteGroupRequest;
  return proto.dropz.DeleteGroupRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.DeleteGroupRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.DeleteGroupRequest}
 */
proto.dropz.DeleteGroupRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGroupId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.DeleteGroupRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.DeleteGroupRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.DeleteGroupRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.DeleteGroupRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGroupId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string group_id = 1;
 * @return {string}
 */
proto.dropz.DeleteGroupRequest.prototype.getGroupId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.DeleteGroupRequest} returns this
 */
proto.dropz.DeleteGroupRequest.prototype.setGroupId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.LoadGroupRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.LoadGroupRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.LoadGroupRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.LoadGroupRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    groupId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.LoadGroupRequest}
 */
proto.dropz.LoadGroupRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.LoadGroupRequest;
  return proto.dropz.LoadGroupRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.LoadGroupRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.LoadGroupRequest}
 */
proto.dropz.LoadGroupRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setGroupId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.LoadGroupRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.LoadGroupRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.LoadGroupRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.LoadGroupRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getGroupId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string group_id = 1;
 * @return {string}
 */
proto.dropz.LoadGroupRequest.prototype.getGroupId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.LoadGroupRequest} returns this
 */
proto.dropz.LoadGroupRequest.prototype.setGroupId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.LoadGroupResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.LoadGroupResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.LoadGroupResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.LoadGroupResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    success: jspb.Message.getBooleanFieldWithDefault(msg, 1, false),
    message: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.LoadGroupResponse}
 */
proto.dropz.LoadGroupResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.LoadGroupResponse;
  return proto.dropz.LoadGroupResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.LoadGroupResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.LoadGroupResponse}
 */
proto.dropz.LoadGroupResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setSuccess(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setMessage(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.LoadGroupResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.LoadGroupResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.LoadGroupResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.LoadGroupResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSuccess();
  if (f) {
    writer.writeBool(
      1,
      f
    );
  }
  f = message.getMessage();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional bool success = 1;
 * @return {boolean}
 */
proto.dropz.LoadGroupResponse.prototype.getSuccess = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 1, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.LoadGroupResponse} returns this
 */
proto.dropz.LoadGroupResponse.prototype.setSuccess = function(value) {
  return jspb.Message.setProto3BooleanField(this, 1, value);
};


/**
 * optional string message = 2;
 * @return {string}
 */
proto.dropz.LoadGroupResponse.prototype.getMessage = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.LoadGroupResponse} returns this
 */
proto.dropz.LoadGroupResponse.prototype.setMessage = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SaveManagedAsGroupRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SaveManagedAsGroupRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SaveManagedAsGroupRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SaveManagedAsGroupRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SaveManagedAsGroupRequest}
 */
proto.dropz.SaveManagedAsGroupRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SaveManagedAsGroupRequest;
  return proto.dropz.SaveManagedAsGroupRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SaveManagedAsGroupRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SaveManagedAsGroupRequest}
 */
proto.dropz.SaveManagedAsGroupRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SaveManagedAsGroupRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SaveManagedAsGroupRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SaveManagedAsGroupRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SaveManagedAsGroupRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.dropz.SaveManagedAsGroupRequest.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SaveManagedAsGroupRequest} returns this
 */
proto.dropz.SaveManagedAsGroupRequest.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SettingOption.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SettingOption.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SettingOption} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SettingOption.toObject = function(includeInstance, msg) {
  var f, obj = {
    value: jspb.Message.getFieldWithDefault(msg, 1, 0),
    name: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SettingOption}
 */
proto.dropz.SettingOption.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SettingOption;
  return proto.dropz.SettingOption.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SettingOption} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SettingOption}
 */
proto.dropz.SettingOption.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setValue(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SettingOption.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SettingOption.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SettingOption} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SettingOption.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getValue();
  if (f !== 0) {
    writer.writeInt64(
      1,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional int64 value = 1;
 * @return {number}
 */
proto.dropz.SettingOption.prototype.getValue = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SettingOption} returns this
 */
proto.dropz.SettingOption.prototype.setValue = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.dropz.SettingOption.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SettingOption} returns this
 */
proto.dropz.SettingOption.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.CameraSetting.repeatedFields_ = [5];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CameraSetting.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CameraSetting.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CameraSetting} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraSetting.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, 0),
    name: jspb.Message.getFieldWithDefault(msg, 2, ""),
    value: jspb.Message.getFieldWithDefault(msg, 3, 0),
    valueName: jspb.Message.getFieldWithDefault(msg, 4, ""),
    optionsList: jspb.Message.toObjectList(msg.getOptionsList(),
    proto.dropz.SettingOption.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CameraSetting}
 */
proto.dropz.CameraSetting.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CameraSetting;
  return proto.dropz.CameraSetting.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CameraSetting} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CameraSetting}
 */
proto.dropz.CameraSetting.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setValue(value);
      break;
    case 4:
      var value = /** @type {string} */ (reader.readString());
      msg.setValueName(value);
      break;
    case 5:
      var value = new proto.dropz.SettingOption;
      reader.readMessage(value,proto.dropz.SettingOption.deserializeBinaryFromReader);
      msg.addOptions(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CameraSetting.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CameraSetting.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CameraSetting} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraSetting.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getValue();
  if (f !== 0) {
    writer.writeInt64(
      3,
      f
    );
  }
  f = message.getValueName();
  if (f.length > 0) {
    writer.writeString(
      4,
      f
    );
  }
  f = message.getOptionsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      5,
      f,
      proto.dropz.SettingOption.serializeBinaryToWriter
    );
  }
};


/**
 * optional int32 id = 1;
 * @return {number}
 */
proto.dropz.CameraSetting.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraSetting} returns this
 */
proto.dropz.CameraSetting.prototype.setId = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional string name = 2;
 * @return {string}
 */
proto.dropz.CameraSetting.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraSetting} returns this
 */
proto.dropz.CameraSetting.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int64 value = 3;
 * @return {number}
 */
proto.dropz.CameraSetting.prototype.getValue = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraSetting} returns this
 */
proto.dropz.CameraSetting.prototype.setValue = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional string value_name = 4;
 * @return {string}
 */
proto.dropz.CameraSetting.prototype.getValueName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 4, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraSetting} returns this
 */
proto.dropz.CameraSetting.prototype.setValueName = function(value) {
  return jspb.Message.setProto3StringField(this, 4, value);
};


/**
 * repeated SettingOption options = 5;
 * @return {!Array<!proto.dropz.SettingOption>}
 */
proto.dropz.CameraSetting.prototype.getOptionsList = function() {
  return /** @type{!Array<!proto.dropz.SettingOption>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SettingOption, 5));
};


/**
 * @param {!Array<!proto.dropz.SettingOption>} value
 * @return {!proto.dropz.CameraSetting} returns this
*/
proto.dropz.CameraSetting.prototype.setOptionsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 5, value);
};


/**
 * @param {!proto.dropz.SettingOption=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SettingOption}
 */
proto.dropz.CameraSetting.prototype.addOptions = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 5, opt_value, proto.dropz.SettingOption, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.CameraSetting} returns this
 */
proto.dropz.CameraSetting.prototype.clearOptionsList = function() {
  return this.setOptionsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SettingChange.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SettingChange.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SettingChange} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SettingChange.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, 0),
    value: jspb.Message.getFieldWithDefault(msg, 2, 0)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SettingChange}
 */
proto.dropz.SettingChange.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SettingChange;
  return proto.dropz.SettingChange.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SettingChange} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SettingChange}
 */
proto.dropz.SettingChange.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setValue(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SettingChange.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SettingChange.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SettingChange} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SettingChange.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getValue();
  if (f !== 0) {
    writer.writeInt64(
      2,
      f
    );
  }
};


/**
 * optional int32 id = 1;
 * @return {number}
 */
proto.dropz.SettingChange.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SettingChange} returns this
 */
proto.dropz.SettingChange.prototype.setId = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional int64 value = 2;
 * @return {number}
 */
proto.dropz.SettingChange.prototype.getValue = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 2, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SettingChange} returns this
 */
proto.dropz.SettingChange.prototype.setValue = function(value) {
  return jspb.Message.setProto3IntField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SettingApplyResult.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SettingApplyResult.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SettingApplyResult} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SettingApplyResult.toObject = function(includeInstance, msg) {
  var f, obj = {
    id: jspb.Message.getFieldWithDefault(msg, 1, 0),
    error: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SettingApplyResult}
 */
proto.dropz.SettingApplyResult.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SettingApplyResult;
  return proto.dropz.SettingApplyResult.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SettingApplyResult} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SettingApplyResult}
 */
proto.dropz.SettingApplyResult.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {number} */ (reader.readInt32());
      msg.setId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setError(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SettingApplyResult.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SettingApplyResult.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SettingApplyResult} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SettingApplyResult.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getId();
  if (f !== 0) {
    writer.writeInt32(
      1,
      f
    );
  }
  f = message.getError();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional int32 id = 1;
 * @return {number}
 */
proto.dropz.SettingApplyResult.prototype.getId = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 1, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.SettingApplyResult} returns this
 */
proto.dropz.SettingApplyResult.prototype.setId = function(value) {
  return jspb.Message.setProto3IntField(this, 1, value);
};


/**
 * optional string error = 2;
 * @return {string}
 */
proto.dropz.SettingApplyResult.prototype.getError = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SettingApplyResult} returns this
 */
proto.dropz.SettingApplyResult.prototype.setError = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.CameraSettingsResult.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CameraSettingsResult.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CameraSettingsResult.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CameraSettingsResult} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraSettingsResult.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    error: jspb.Message.getFieldWithDefault(msg, 2, ""),
    resultsList: jspb.Message.toObjectList(msg.getResultsList(),
    proto.dropz.SettingApplyResult.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CameraSettingsResult}
 */
proto.dropz.CameraSettingsResult.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CameraSettingsResult;
  return proto.dropz.CameraSettingsResult.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CameraSettingsResult} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CameraSettingsResult}
 */
proto.dropz.CameraSettingsResult.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setError(value);
      break;
    case 3:
      var value = new proto.dropz.SettingApplyResult;
      reader.readMessage(value,proto.dropz.SettingApplyResult.deserializeBinaryFromReader);
      msg.addResults(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CameraSettingsResult.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CameraSettingsResult.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CameraSettingsResult} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraSettingsResult.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getError();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getResultsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.dropz.SettingApplyResult.serializeBinaryToWriter
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.CameraSettingsResult.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraSettingsResult} returns this
 */
proto.dropz.CameraSettingsResult.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string error = 2;
 * @return {string}
 */
proto.dropz.CameraSettingsResult.prototype.getError = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraSettingsResult} returns this
 */
proto.dropz.CameraSettingsResult.prototype.setError = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated SettingApplyResult results = 3;
 * @return {!Array<!proto.dropz.SettingApplyResult>}
 */
proto.dropz.CameraSettingsResult.prototype.getResultsList = function() {
  return /** @type{!Array<!proto.dropz.SettingApplyResult>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SettingApplyResult, 3));
};


/**
 * @param {!Array<!proto.dropz.SettingApplyResult>} value
 * @return {!proto.dropz.CameraSettingsResult} returns this
*/
proto.dropz.CameraSettingsResult.prototype.setResultsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.dropz.SettingApplyResult=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SettingApplyResult}
 */
proto.dropz.CameraSettingsResult.prototype.addResults = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.dropz.SettingApplyResult, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.CameraSettingsResult} returns this
 */
proto.dropz.CameraSettingsResult.prototype.clearResultsList = function() {
  return this.setResultsList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetCameraSettingsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetCameraSettingsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetCameraSettingsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraSettingsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    refresh: jspb.Message.getBooleanFieldWithDefault(msg, 2, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetCameraSettingsRequest}
 */
proto.dropz.GetCameraSettingsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetCameraSettingsRequest;
  return proto.dropz.GetCameraSettingsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetCameraSettingsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetCameraSettingsRequest}
 */
proto.dropz.GetCameraSettingsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setRefresh(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetCameraSettingsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetCameraSettingsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetCameraSettingsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraSettingsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getRefresh();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.GetCameraSettingsRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.GetCameraSettingsRequest} returns this
 */
proto.dropz.GetCameraSettingsRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional bool refresh = 2;
 * @return {boolean}
 */
proto.dropz.GetCameraSettingsRequest.prototype.getRefresh = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.GetCameraSettingsRequest} returns this
 */
proto.dropz.GetCameraSettingsRequest.prototype.setRefresh = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.GetCameraSettingsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetCameraSettingsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetCameraSettingsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetCameraSettingsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraSettingsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    settingsList: jspb.Message.toObjectList(msg.getSettingsList(),
    proto.dropz.CameraSetting.toObject, includeInstance),
    updatedAt: (f = msg.getUpdatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetCameraSettingsResponse}
 */
proto.dropz.GetCameraSettingsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetCameraSettingsResponse;
  return proto.dropz.GetCameraSettingsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetCameraSettingsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetCameraSettingsResponse}
 */
proto.dropz.GetCameraSettingsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.CameraSetting;
      reader.readMessage(value,proto.dropz.CameraSetting.deserializeBinaryFromReader);
      msg.addSettings(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setUpdatedAt(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetCameraSettingsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetCameraSettingsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetCameraSettingsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraSettingsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getSettingsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.CameraSetting.serializeBinaryToWriter
    );
  }
  f = message.getUpdatedAt();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * repeated CameraSetting settings = 1;
 * @return {!Array<!proto.dropz.CameraSetting>}
 */
proto.dropz.GetCameraSettingsResponse.prototype.getSettingsList = function() {
  return /** @type{!Array<!proto.dropz.CameraSetting>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.CameraSetting, 1));
};


/**
 * @param {!Array<!proto.dropz.CameraSetting>} value
 * @return {!proto.dropz.GetCameraSettingsResponse} returns this
*/
proto.dropz.GetCameraSettingsResponse.prototype.setSettingsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.CameraSetting=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.CameraSetting}
 */
proto.dropz.GetCameraSettingsResponse.prototype.addSettings = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.CameraSetting, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.GetCameraSettingsResponse} returns this
 */
proto.dropz.GetCameraSettingsResponse.prototype.clearSettingsList = function() {
  return this.setSettingsList([]);
};


/**
 * optional google.protobuf.Timestamp updated_at = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.GetCameraSettingsResponse.prototype.getUpdatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.GetCameraSettingsResponse} returns this
*/
proto.dropz.GetCameraSettingsResponse.prototype.setUpdatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.GetCameraSettingsResponse} returns this
 */
proto.dropz.GetCameraSettingsResponse.prototype.clearUpdatedAt = function() {
  return this.setUpdatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.GetCameraSettingsResponse.prototype.hasUpdatedAt = function() {
  return jspb.Message.getField(this, 2) != null;
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.ApplyCameraSettingsRequest.repeatedFields_ = [3];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ApplyCameraSettingsRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ApplyCameraSettingsRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ApplyCameraSettingsRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    groupId: jspb.Message.getFieldWithDefault(msg, 2, ""),
    changesList: jspb.Message.toObjectList(msg.getChangesList(),
    proto.dropz.SettingChange.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ApplyCameraSettingsRequest}
 */
proto.dropz.ApplyCameraSettingsRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ApplyCameraSettingsRequest;
  return proto.dropz.ApplyCameraSettingsRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ApplyCameraSettingsRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ApplyCameraSettingsRequest}
 */
proto.dropz.ApplyCameraSettingsRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setGroupId(value);
      break;
    case 3:
      var value = new proto.dropz.SettingChange;
      reader.readMessage(value,proto.dropz.SettingChange.deserializeBinaryFromReader);
      msg.addChanges(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ApplyCameraSettingsRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ApplyCameraSettingsRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ApplyCameraSettingsRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getGroupId();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getChangesList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      3,
      f,
      proto.dropz.SettingChange.serializeBinaryToWriter
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ApplyCameraSettingsRequest} returns this
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string group_id = 2;
 * @return {string}
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.getGroupId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.ApplyCameraSettingsRequest} returns this
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.setGroupId = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * repeated SettingChange changes = 3;
 * @return {!Array<!proto.dropz.SettingChange>}
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.getChangesList = function() {
  return /** @type{!Array<!proto.dropz.SettingChange>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.SettingChange, 3));
};


/**
 * @param {!Array<!proto.dropz.SettingChange>} value
 * @return {!proto.dropz.ApplyCameraSettingsRequest} returns this
*/
proto.dropz.ApplyCameraSettingsRequest.prototype.setChangesList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 3, value);
};


/**
 * @param {!proto.dropz.SettingChange=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.SettingChange}
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.addChanges = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 3, opt_value, proto.dropz.SettingChange, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.ApplyCameraSettingsRequest} returns this
 */
proto.dropz.ApplyCameraSettingsRequest.prototype.clearChangesList = function() {
  return this.setChangesList([]);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.ApplyCameraSettingsResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.ApplyCameraSettingsResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.ApplyCameraSettingsResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.ApplyCameraSettingsResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ApplyCameraSettingsResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    camerasList: jspb.Message.toObjectList(msg.getCamerasList(),
    proto.dropz.CameraSettingsResult.toObject, includeInstance)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.ApplyCameraSettingsResponse}
 */
proto.dropz.ApplyCameraSettingsResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.ApplyCameraSettingsResponse;
  return proto.dropz.ApplyCameraSettingsResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.ApplyCameraSettingsResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.ApplyCameraSettingsResponse}
 */
proto.dropz.ApplyCameraSettingsResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.CameraSettingsResult;
      reader.readMessage(value,proto.dropz.CameraSettingsResult.deserializeBinaryFromReader);
      msg.addCameras(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.ApplyCameraSettingsResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.ApplyCameraSettingsResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.ApplyCameraSettingsResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.ApplyCameraSettingsResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCamerasList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.CameraSettingsResult.serializeBinaryToWriter
    );
  }
};


/**
 * repeated CameraSettingsResult cameras = 1;
 * @return {!Array<!proto.dropz.CameraSettingsResult>}
 */
proto.dropz.ApplyCameraSettingsResponse.prototype.getCamerasList = function() {
  return /** @type{!Array<!proto.dropz.CameraSettingsResult>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.CameraSettingsResult, 1));
};


/**
 * @param {!Array<!proto.dropz.CameraSettingsResult>} value
 * @return {!proto.dropz.ApplyCameraSettingsResponse} returns this
*/
proto.dropz.ApplyCameraSettingsResponse.prototype.setCamerasList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.CameraSettingsResult=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.CameraSettingsResult}
 */
proto.dropz.ApplyCameraSettingsResponse.prototype.addCameras = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.CameraSettingsResult, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.ApplyCameraSettingsResponse} returns this
 */
proto.dropz.ApplyCameraSettingsResponse.prototype.clearCamerasList = function() {
  return this.setCamerasList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.CameraMediaItem.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.CameraMediaItem.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.CameraMediaItem} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraMediaItem.toObject = function(includeInstance, msg) {
  var f, obj = {
    name: jspb.Message.getFieldWithDefault(msg, 1, ""),
    cameraPath: jspb.Message.getFieldWithDefault(msg, 2, ""),
    sizeBytes: jspb.Message.getFieldWithDefault(msg, 3, 0),
    createdAt: (f = msg.getCreatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f),
    thumbnailPath: jspb.Message.getFieldWithDefault(msg, 5, ""),
    downloaded: jspb.Message.getBooleanFieldWithDefault(msg, 6, false),
    localPath: jspb.Message.getFieldWithDefault(msg, 7, ""),
    previewPath: jspb.Message.getFieldWithDefault(msg, 8, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.CameraMediaItem}
 */
proto.dropz.CameraMediaItem.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.CameraMediaItem;
  return proto.dropz.CameraMediaItem.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.CameraMediaItem} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.CameraMediaItem}
 */
proto.dropz.CameraMediaItem.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setName(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraPath(value);
      break;
    case 3:
      var value = /** @type {number} */ (reader.readInt64());
      msg.setSizeBytes(value);
      break;
    case 4:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setCreatedAt(value);
      break;
    case 5:
      var value = /** @type {string} */ (reader.readString());
      msg.setThumbnailPath(value);
      break;
    case 6:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setDownloaded(value);
      break;
    case 7:
      var value = /** @type {string} */ (reader.readString());
      msg.setLocalPath(value);
      break;
    case 8:
      var value = /** @type {string} */ (reader.readString());
      msg.setPreviewPath(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.CameraMediaItem.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.CameraMediaItem.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.CameraMediaItem} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.CameraMediaItem.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getName();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCameraPath();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
  f = message.getSizeBytes();
  if (f !== 0) {
    writer.writeInt64(
      3,
      f
    );
  }
  f = message.getCreatedAt();
  if (f != null) {
    writer.writeMessage(
      4,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
  f = message.getThumbnailPath();
  if (f.length > 0) {
    writer.writeString(
      5,
      f
    );
  }
  f = message.getDownloaded();
  if (f) {
    writer.writeBool(
      6,
      f
    );
  }
  f = message.getLocalPath();
  if (f.length > 0) {
    writer.writeString(
      7,
      f
    );
  }
  f = message.getPreviewPath();
  if (f.length > 0) {
    writer.writeString(
      8,
      f
    );
  }
};


/**
 * optional string name = 1;
 * @return {string}
 */
proto.dropz.CameraMediaItem.prototype.getName = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.setName = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string camera_path = 2;
 * @return {string}
 */
proto.dropz.CameraMediaItem.prototype.getCameraPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.setCameraPath = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};


/**
 * optional int64 size_bytes = 3;
 * @return {number}
 */
proto.dropz.CameraMediaItem.prototype.getSizeBytes = function() {
  return /** @type {number} */ (jspb.Message.getFieldWithDefault(this, 3, 0));
};


/**
 * @param {number} value
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.setSizeBytes = function(value) {
  return jspb.Message.setProto3IntField(this, 3, value);
};


/**
 * optional google.protobuf.Timestamp created_at = 4;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.CameraMediaItem.prototype.getCreatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 4));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.CameraMediaItem} returns this
*/
proto.dropz.CameraMediaItem.prototype.setCreatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 4, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.clearCreatedAt = function() {
  return this.setCreatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.CameraMediaItem.prototype.hasCreatedAt = function() {
  return jspb.Message.getField(this, 4) != null;
};


/**
 * optional string thumbnail_path = 5;
 * @return {string}
 */
proto.dropz.CameraMediaItem.prototype.getThumbnailPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 5, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.setThumbnailPath = function(value) {
  return jspb.Message.setProto3StringField(this, 5, value);
};


/**
 * optional bool downloaded = 6;
 * @return {boolean}
 */
proto.dropz.CameraMediaItem.prototype.getDownloaded = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 6, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.setDownloaded = function(value) {
  return jspb.Message.setProto3BooleanField(this, 6, value);
};


/**
 * optional string local_path = 7;
 * @return {string}
 */
proto.dropz.CameraMediaItem.prototype.getLocalPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 7, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.setLocalPath = function(value) {
  return jspb.Message.setProto3StringField(this, 7, value);
};


/**
 * optional string preview_path = 8;
 * @return {string}
 */
proto.dropz.CameraMediaItem.prototype.getPreviewPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 8, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.CameraMediaItem} returns this
 */
proto.dropz.CameraMediaItem.prototype.setPreviewPath = function(value) {
  return jspb.Message.setProto3StringField(this, 8, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetCameraMediaRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetCameraMediaRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetCameraMediaRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraMediaRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetCameraMediaRequest}
 */
proto.dropz.GetCameraMediaRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetCameraMediaRequest;
  return proto.dropz.GetCameraMediaRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetCameraMediaRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetCameraMediaRequest}
 */
proto.dropz.GetCameraMediaRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetCameraMediaRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetCameraMediaRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetCameraMediaRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraMediaRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.GetCameraMediaRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.GetCameraMediaRequest} returns this
 */
proto.dropz.GetCameraMediaRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.GetCameraMediaResponse.repeatedFields_ = [1];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.GetCameraMediaResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.GetCameraMediaResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.GetCameraMediaResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraMediaResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    itemsList: jspb.Message.toObjectList(msg.getItemsList(),
    proto.dropz.CameraMediaItem.toObject, includeInstance),
    updatedAt: (f = msg.getUpdatedAt()) && google_protobuf_timestamp_pb.Timestamp.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.GetCameraMediaResponse}
 */
proto.dropz.GetCameraMediaResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.GetCameraMediaResponse;
  return proto.dropz.GetCameraMediaResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.GetCameraMediaResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.GetCameraMediaResponse}
 */
proto.dropz.GetCameraMediaResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.CameraMediaItem;
      reader.readMessage(value,proto.dropz.CameraMediaItem.deserializeBinaryFromReader);
      msg.addItems(value);
      break;
    case 2:
      var value = new google_protobuf_timestamp_pb.Timestamp;
      reader.readMessage(value,google_protobuf_timestamp_pb.Timestamp.deserializeBinaryFromReader);
      msg.setUpdatedAt(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.GetCameraMediaResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.GetCameraMediaResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.GetCameraMediaResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.GetCameraMediaResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getItemsList();
  if (f.length > 0) {
    writer.writeRepeatedMessage(
      1,
      f,
      proto.dropz.CameraMediaItem.serializeBinaryToWriter
    );
  }
  f = message.getUpdatedAt();
  if (f != null) {
    writer.writeMessage(
      2,
      f,
      google_protobuf_timestamp_pb.Timestamp.serializeBinaryToWriter
    );
  }
};


/**
 * repeated CameraMediaItem items = 1;
 * @return {!Array<!proto.dropz.CameraMediaItem>}
 */
proto.dropz.GetCameraMediaResponse.prototype.getItemsList = function() {
  return /** @type{!Array<!proto.dropz.CameraMediaItem>} */ (
    jspb.Message.getRepeatedWrapperField(this, proto.dropz.CameraMediaItem, 1));
};


/**
 * @param {!Array<!proto.dropz.CameraMediaItem>} value
 * @return {!proto.dropz.GetCameraMediaResponse} returns this
*/
proto.dropz.GetCameraMediaResponse.prototype.setItemsList = function(value) {
  return jspb.Message.setRepeatedWrapperField(this, 1, value);
};


/**
 * @param {!proto.dropz.CameraMediaItem=} opt_value
 * @param {number=} opt_index
 * @return {!proto.dropz.CameraMediaItem}
 */
proto.dropz.GetCameraMediaResponse.prototype.addItems = function(opt_value, opt_index) {
  return jspb.Message.addToRepeatedWrapperField(this, 1, opt_value, proto.dropz.CameraMediaItem, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.GetCameraMediaResponse} returns this
 */
proto.dropz.GetCameraMediaResponse.prototype.clearItemsList = function() {
  return this.setItemsList([]);
};


/**
 * optional google.protobuf.Timestamp updated_at = 2;
 * @return {?proto.google.protobuf.Timestamp}
 */
proto.dropz.GetCameraMediaResponse.prototype.getUpdatedAt = function() {
  return /** @type{?proto.google.protobuf.Timestamp} */ (
    jspb.Message.getWrapperField(this, google_protobuf_timestamp_pb.Timestamp, 2));
};


/**
 * @param {?proto.google.protobuf.Timestamp|undefined} value
 * @return {!proto.dropz.GetCameraMediaResponse} returns this
*/
proto.dropz.GetCameraMediaResponse.prototype.setUpdatedAt = function(value) {
  return jspb.Message.setWrapperField(this, 2, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.GetCameraMediaResponse} returns this
 */
proto.dropz.GetCameraMediaResponse.prototype.clearUpdatedAt = function() {
  return this.setUpdatedAt(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.GetCameraMediaResponse.prototype.hasUpdatedAt = function() {
  return jspb.Message.getField(this, 2) != null;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.PreviewMediaRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.PreviewMediaRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.PreviewMediaRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewMediaRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    cameraPath: jspb.Message.getFieldWithDefault(msg, 2, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.PreviewMediaRequest}
 */
proto.dropz.PreviewMediaRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.PreviewMediaRequest;
  return proto.dropz.PreviewMediaRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.PreviewMediaRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.PreviewMediaRequest}
 */
proto.dropz.PreviewMediaRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraPath(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.PreviewMediaRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.PreviewMediaRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.PreviewMediaRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewMediaRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getCameraPath();
  if (f.length > 0) {
    writer.writeString(
      2,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.PreviewMediaRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.PreviewMediaRequest} returns this
 */
proto.dropz.PreviewMediaRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional string camera_path = 2;
 * @return {string}
 */
proto.dropz.PreviewMediaRequest.prototype.getCameraPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 2, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.PreviewMediaRequest} returns this
 */
proto.dropz.PreviewMediaRequest.prototype.setCameraPath = function(value) {
  return jspb.Message.setProto3StringField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.PreviewMediaResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.PreviewMediaResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.PreviewMediaResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewMediaResponse.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.PreviewMediaResponse}
 */
proto.dropz.PreviewMediaResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.PreviewMediaResponse;
  return proto.dropz.PreviewMediaResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.PreviewMediaResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.PreviewMediaResponse}
 */
proto.dropz.PreviewMediaResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.PreviewMediaResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.PreviewMediaResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.PreviewMediaResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewMediaResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SetPreviewSessionRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SetPreviewSessionRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SetPreviewSessionRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetPreviewSessionRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    enabled: jspb.Message.getBooleanFieldWithDefault(msg, 2, false)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SetPreviewSessionRequest}
 */
proto.dropz.SetPreviewSessionRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SetPreviewSessionRequest;
  return proto.dropz.SetPreviewSessionRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SetPreviewSessionRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SetPreviewSessionRequest}
 */
proto.dropz.SetPreviewSessionRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = /** @type {boolean} */ (reader.readBool());
      msg.setEnabled(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SetPreviewSessionRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SetPreviewSessionRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SetPreviewSessionRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetPreviewSessionRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getEnabled();
  if (f) {
    writer.writeBool(
      2,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.SetPreviewSessionRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.SetPreviewSessionRequest} returns this
 */
proto.dropz.SetPreviewSessionRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * optional bool enabled = 2;
 * @return {boolean}
 */
proto.dropz.SetPreviewSessionRequest.prototype.getEnabled = function() {
  return /** @type {boolean} */ (jspb.Message.getBooleanFieldWithDefault(this, 2, false));
};


/**
 * @param {boolean} value
 * @return {!proto.dropz.SetPreviewSessionRequest} returns this
 */
proto.dropz.SetPreviewSessionRequest.prototype.setEnabled = function(value) {
  return jspb.Message.setProto3BooleanField(this, 2, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.SetPreviewSessionResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.SetPreviewSessionResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.SetPreviewSessionResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetPreviewSessionResponse.toObject = function(includeInstance, msg) {
  var f, obj = {

  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.SetPreviewSessionResponse}
 */
proto.dropz.SetPreviewSessionResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.SetPreviewSessionResponse;
  return proto.dropz.SetPreviewSessionResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.SetPreviewSessionResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.SetPreviewSessionResponse}
 */
proto.dropz.SetPreviewSessionResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.SetPreviewSessionResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.SetPreviewSessionResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.SetPreviewSessionResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.SetPreviewSessionResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.PreviewVideoRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.PreviewVideoRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.PreviewVideoRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewVideoRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    videoPath: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.PreviewVideoRequest}
 */
proto.dropz.PreviewVideoRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.PreviewVideoRequest;
  return proto.dropz.PreviewVideoRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.PreviewVideoRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.PreviewVideoRequest}
 */
proto.dropz.PreviewVideoRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setVideoPath(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.PreviewVideoRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.PreviewVideoRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.PreviewVideoRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewVideoRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getVideoPath();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string video_path = 1;
 * @return {string}
 */
proto.dropz.PreviewVideoRequest.prototype.getVideoPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.PreviewVideoRequest} returns this
 */
proto.dropz.PreviewVideoRequest.prototype.setVideoPath = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.PreviewVideoResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.PreviewVideoResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.PreviewVideoResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewVideoResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    previewPath: jspb.Message.getFieldWithDefault(msg, 1, "")
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.PreviewVideoResponse}
 */
proto.dropz.PreviewVideoResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.PreviewVideoResponse;
  return proto.dropz.PreviewVideoResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.PreviewVideoResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.PreviewVideoResponse}
 */
proto.dropz.PreviewVideoResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setPreviewPath(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.PreviewVideoResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.PreviewVideoResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.PreviewVideoResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.PreviewVideoResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getPreviewPath();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
};


/**
 * optional string preview_path = 1;
 * @return {string}
 */
proto.dropz.PreviewVideoResponse.prototype.getPreviewPath = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.PreviewVideoResponse} returns this
 */
proto.dropz.PreviewVideoResponse.prototype.setPreviewPath = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};



/**
 * List of repeated fields within this message type.
 * @private {!Array<number>}
 * @const
 */
proto.dropz.RequestMediaDownloadRequest.repeatedFields_ = [2];



if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.RequestMediaDownloadRequest.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.RequestMediaDownloadRequest.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.RequestMediaDownloadRequest} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.RequestMediaDownloadRequest.toObject = function(includeInstance, msg) {
  var f, obj = {
    cameraId: jspb.Message.getFieldWithDefault(msg, 1, ""),
    fileNamesList: (f = jspb.Message.getRepeatedField(msg, 2)) == null ? undefined : f
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.RequestMediaDownloadRequest}
 */
proto.dropz.RequestMediaDownloadRequest.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.RequestMediaDownloadRequest;
  return proto.dropz.RequestMediaDownloadRequest.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.RequestMediaDownloadRequest} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.RequestMediaDownloadRequest}
 */
proto.dropz.RequestMediaDownloadRequest.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = /** @type {string} */ (reader.readString());
      msg.setCameraId(value);
      break;
    case 2:
      var value = /** @type {string} */ (reader.readString());
      msg.addFileNames(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.RequestMediaDownloadRequest.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.RequestMediaDownloadRequest.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.RequestMediaDownloadRequest} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.RequestMediaDownloadRequest.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getCameraId();
  if (f.length > 0) {
    writer.writeString(
      1,
      f
    );
  }
  f = message.getFileNamesList();
  if (f.length > 0) {
    writer.writeRepeatedString(
      2,
      f
    );
  }
};


/**
 * optional string camera_id = 1;
 * @return {string}
 */
proto.dropz.RequestMediaDownloadRequest.prototype.getCameraId = function() {
  return /** @type {string} */ (jspb.Message.getFieldWithDefault(this, 1, ""));
};


/**
 * @param {string} value
 * @return {!proto.dropz.RequestMediaDownloadRequest} returns this
 */
proto.dropz.RequestMediaDownloadRequest.prototype.setCameraId = function(value) {
  return jspb.Message.setProto3StringField(this, 1, value);
};


/**
 * repeated string file_names = 2;
 * @return {!Array<string>}
 */
proto.dropz.RequestMediaDownloadRequest.prototype.getFileNamesList = function() {
  return /** @type {!Array<string>} */ (jspb.Message.getRepeatedField(this, 2));
};


/**
 * @param {!Array<string>} value
 * @return {!proto.dropz.RequestMediaDownloadRequest} returns this
 */
proto.dropz.RequestMediaDownloadRequest.prototype.setFileNamesList = function(value) {
  return jspb.Message.setField(this, 2, value || []);
};


/**
 * @param {string} value
 * @param {number=} opt_index
 * @return {!proto.dropz.RequestMediaDownloadRequest} returns this
 */
proto.dropz.RequestMediaDownloadRequest.prototype.addFileNames = function(value, opt_index) {
  return jspb.Message.addToRepeatedField(this, 2, value, opt_index);
};


/**
 * Clears the list making it empty but non-null.
 * @return {!proto.dropz.RequestMediaDownloadRequest} returns this
 */
proto.dropz.RequestMediaDownloadRequest.prototype.clearFileNamesList = function() {
  return this.setFileNamesList([]);
};





if (jspb.Message.GENERATE_TO_OBJECT) {
/**
 * Creates an object representation of this proto.
 * Field names that are reserved in JavaScript and will be renamed to pb_name.
 * Optional fields that are not set will be set to undefined.
 * To access a reserved field use, foo.pb_<name>, eg, foo.pb_default.
 * For the list of reserved names please see:
 *     net/proto2/compiler/js/internal/generator.cc#kKeyword.
 * @param {boolean=} opt_includeInstance Deprecated. whether to include the
 *     JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @return {!Object}
 */
proto.dropz.RequestMediaDownloadResponse.prototype.toObject = function(opt_includeInstance) {
  return proto.dropz.RequestMediaDownloadResponse.toObject(opt_includeInstance, this);
};


/**
 * Static version of the {@see toObject} method.
 * @param {boolean|undefined} includeInstance Deprecated. Whether to include
 *     the JSPB instance for transitional soy proto support:
 *     http://goto/soy-param-migration
 * @param {!proto.dropz.RequestMediaDownloadResponse} msg The msg instance to transform.
 * @return {!Object}
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.RequestMediaDownloadResponse.toObject = function(includeInstance, msg) {
  var f, obj = {
    entry: (f = msg.getEntry()) && proto.dropz.SyncQueueEntry.toObject(includeInstance, f)
  };

  if (includeInstance) {
    obj.$jspbMessageInstance = msg;
  }
  return obj;
};
}


/**
 * Deserializes binary data (in protobuf wire format).
 * @param {jspb.ByteSource} bytes The bytes to deserialize.
 * @return {!proto.dropz.RequestMediaDownloadResponse}
 */
proto.dropz.RequestMediaDownloadResponse.deserializeBinary = function(bytes) {
  var reader = new jspb.BinaryReader(bytes);
  var msg = new proto.dropz.RequestMediaDownloadResponse;
  return proto.dropz.RequestMediaDownloadResponse.deserializeBinaryFromReader(msg, reader);
};


/**
 * Deserializes binary data (in protobuf wire format) from the
 * given reader into the given message object.
 * @param {!proto.dropz.RequestMediaDownloadResponse} msg The message object to deserialize into.
 * @param {!jspb.BinaryReader} reader The BinaryReader to use.
 * @return {!proto.dropz.RequestMediaDownloadResponse}
 */
proto.dropz.RequestMediaDownloadResponse.deserializeBinaryFromReader = function(msg, reader) {
  while (reader.nextField()) {
    if (reader.isEndGroup()) {
      break;
    }
    var field = reader.getFieldNumber();
    switch (field) {
    case 1:
      var value = new proto.dropz.SyncQueueEntry;
      reader.readMessage(value,proto.dropz.SyncQueueEntry.deserializeBinaryFromReader);
      msg.setEntry(value);
      break;
    default:
      reader.skipField();
      break;
    }
  }
  return msg;
};


/**
 * Serializes the message to binary data (in protobuf wire format).
 * @return {!Uint8Array}
 */
proto.dropz.RequestMediaDownloadResponse.prototype.serializeBinary = function() {
  var writer = new jspb.BinaryWriter();
  proto.dropz.RequestMediaDownloadResponse.serializeBinaryToWriter(this, writer);
  return writer.getResultBuffer();
};


/**
 * Serializes the given message to binary data (in protobuf wire
 * format), writing to the given BinaryWriter.
 * @param {!proto.dropz.RequestMediaDownloadResponse} message
 * @param {!jspb.BinaryWriter} writer
 * @suppress {unusedLocalVariables} f is only used for nested messages
 */
proto.dropz.RequestMediaDownloadResponse.serializeBinaryToWriter = function(message, writer) {
  var f = undefined;
  f = message.getEntry();
  if (f != null) {
    writer.writeMessage(
      1,
      f,
      proto.dropz.SyncQueueEntry.serializeBinaryToWriter
    );
  }
};


/**
 * optional SyncQueueEntry entry = 1;
 * @return {?proto.dropz.SyncQueueEntry}
 */
proto.dropz.RequestMediaDownloadResponse.prototype.getEntry = function() {
  return /** @type{?proto.dropz.SyncQueueEntry} */ (
    jspb.Message.getWrapperField(this, proto.dropz.SyncQueueEntry, 1));
};


/**
 * @param {?proto.dropz.SyncQueueEntry|undefined} value
 * @return {!proto.dropz.RequestMediaDownloadResponse} returns this
*/
proto.dropz.RequestMediaDownloadResponse.prototype.setEntry = function(value) {
  return jspb.Message.setWrapperField(this, 1, value);
};


/**
 * Clears the message field making it undefined.
 * @return {!proto.dropz.RequestMediaDownloadResponse} returns this
 */
proto.dropz.RequestMediaDownloadResponse.prototype.clearEntry = function() {
  return this.setEntry(undefined);
};


/**
 * Returns whether this field is set.
 * @return {boolean}
 */
proto.dropz.RequestMediaDownloadResponse.prototype.hasEntry = function() {
  return jspb.Message.getField(this, 1) != null;
};


/**
 * @enum {number}
 */
proto.dropz.SyncFileState = {
  SYNC_FILE_QUEUED: 0,
  SYNC_FILE_DOWNLOADING: 1,
  SYNC_FILE_DONE: 2,
  SYNC_FILE_FAILED: 3,
  SYNC_FILE_SKIPPED: 4
};

/**
 * @enum {number}
 */
proto.dropz.SyncPhase = {
  SYNC_PHASE_WAITING: 0,
  SYNC_PHASE_CONNECT: 1,
  SYNC_PHASE_LINK: 2,
  SYNC_PHASE_CATALOG: 3,
  SYNC_PHASE_TRANSFER: 4
};

/**
 * @enum {number}
 */
proto.dropz.SyncOutcome = {
  SYNC_OUTCOME_UNSPECIFIED: 0,
  SYNC_OUTCOME_COMPLETE: 1,
  SYNC_OUTCOME_UP_TO_DATE: 2,
  SYNC_OUTCOME_CATALOG_REFRESHED: 3,
  SYNC_OUTCOME_FAILED: 4,
  SYNC_OUTCOME_CANCELLED: 5
};

goog.object.extend(exports, proto.dropz);
