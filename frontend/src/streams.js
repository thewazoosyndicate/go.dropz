// gRPC stream management
const { LOG_LEVELS } = require('./fileLogger');
const state = require('./state');
const devicePools = require('./device-pools');
const deviceCard = require('./device-card');

let _debugLog = null;
function debugLog(msg, level = LOG_LEVELS.INFO) {
  if (_debugLog) return _debugLog(msg, level);
}

let _onStreamStateChange = null;

function processCamera(camera) {
  try {
    const cameraState = camera.getCameraState();
    if (!cameraState) return null;
    const cameraData = cameraState.getCamera();
    if (!cameraData) return null;
    const macAddress = cameraData.getMacAddress();
    if (!macAddress) return null;

    const status = cameraState.getStatus();
    const metadata = cameraState.getMetadata();

    return {
      macAddress,
      id: cameraData.getId(),
      name: cameraData.getName() || 'Unnamed Camera',
      alias: cameraData.getAlias(),
      wifiSsid: cameraData.getWifiSsid(),
      wifiPassword: cameraData.getWifiPassword(),
      rssi: cameraData.getRssi(),
      lastSeen: status ? status.getLastSeen()?.toDate() : null,
      lastSynced: status ? status.getLastSynced()?.toDate() : null,
      isPairing: status ? status.getIsPairing() : false,
      isPaired: status ? status.getIsPaired() : false,
      isManaged: status ? status.getIsManaged() : false,
      isReachable: status ? status.getIsReachable() : false,
      isSynced: status ? status.getIsSynced() : false,
      isSyncing: status ? status.getIsSyncing() : false,
      groupId: cameraState.getGroupId(),
      batteryLevel: metadata ? metadata.getBatteryLevel() : null,
      firmwareVersion: metadata ? metadata.getFirmwareVersion() : null,
      model: metadata ? metadata.getModel() : null,
      serialNumber: metadata ? metadata.getSerialNumber() : null,
      hardwareVersion: metadata ? metadata.getHardwareVersion() : null,
      visualStatus: deviceCard.determineVisualStatus(status)
    };
  } catch (error) {
    debugLog(`Error processing camera: ${error.message}`, LOG_LEVELS.ERROR);
    return null;
  }
}

function processSyncQueueEntry(entry) {
  try {
    return {
      cameraId: entry.getCameraId(),
      queuedAt: entry.getQueuedAt()?.toDate(),
      priority: entry.getPriority(),
      progressPercent: entry.getProgressPercent(),
      currentOperation: entry.getCurrentOperation()
    };
  } catch (error) {
    debugLog(`Error processing sync queue entry: ${error.message}`, LOG_LEVELS.ERROR);
    return null;
  }
}

function calculateBackoffDelay() {
  const delay = state.connectionState.baseDelay * Math.pow(2, state.connectionState.retryCount);
  state.connectionState.retryCount = Math.min(state.connectionState.retryCount + 1, state.connectionState.maxRetries);
  return Math.min(delay, 30000);
}

function startStream(streamName, streamKey, client, RequestClass, watchMethod, onData) {
  try {
    const request = new RequestClass();
    request.setChangeCounter(0);
    debugLog(`Starting ${streamName} stream`, LOG_LEVELS.TRACE);

    const call = client[watchMethod](request);
    if (!state.activeStream) state.activeStream = {};
    state.activeStream[streamKey] = {
      call,
      cancel: () => { call.cancel(); debugLog(`${streamName} stream cancelled`, LOG_LEVELS.INFO); },
      intentionalCancel: false
    };

    call.on('data', (response) => {
      if (state.streamStatus[streamKey] !== 'connected') {
        state.streamStatus[streamKey] = 'connected';
        if (_onStreamStateChange) _onStreamStateChange();
      }
      if (response.getHeartbeat() || response.getNoChanges()) return;
      onData(response);
    });

    call.on('error', (error) => {
      state.streamStatus[streamKey] = 'reconnecting';
      if (_onStreamStateChange) _onStreamStateChange();
      debugLog(`${streamName} stream error: ${error.message}`, LOG_LEVELS.ERROR);
      if (state.activeStream[streamKey]?.intentionalCancel) return;
      setTimeout(() => {
        if (state.serviceRunning) {
          debugLog(`Reconnecting ${streamName} stream`, LOG_LEVELS.INFO);
          startStream(streamName, streamKey, client, RequestClass, watchMethod, onData);
        }
      }, calculateBackoffDelay());
    });

    call.on('end', () => {
      state.streamStatus[streamKey] = 'reconnecting';
      if (_onStreamStateChange) _onStreamStateChange();
      debugLog(`${streamName} stream ended`, LOG_LEVELS.INFO);
      if (!state.activeStream[streamKey]?.intentionalCancel && state.serviceRunning) {
        setTimeout(() => startStream(streamName, streamKey, client, RequestClass, watchMethod, onData), calculateBackoffDelay());
      }
    });
  } catch (error) {
    debugLog(`Failed to start ${streamName} stream: ${error.message}`, LOG_LEVELS.ERROR);
  }
}

function startDiscoveredCamerasStream(client) {
  const grpcUtils = require('./grpc-utils');
  startStream('DiscoveredCameras', 'discoveredCamerasStream', client,
    grpcUtils.GetDiscoveredCamerasRequest, 'watchDiscoveredCameras',
    (response) => {
      response.getCamerasList().forEach(camera => {
        const obj = processCamera(camera);
        if (obj) devicePools.addOrUpdateDevice(obj);
      });
    });
}

function startManagedCamerasStream(client) {
  const grpcUtils = require('./grpc-utils');
  startStream('ManagedCameras', 'managedCamerasStream', client,
    grpcUtils.GetManagedCamerasRequest, 'watchManagedCameras',
    (response) => {
      response.getCamerasList().forEach(camera => {
        const obj = processCamera(camera);
        if (obj) devicePools.addOrUpdateDevice(obj);
      });
    });
}

function startSyncQueueStream(client) {
  const grpcUtils = require('./grpc-utils');
  startStream('SyncQueue', 'syncQueueStream', client,
    grpcUtils.GetSyncQueueRequest, 'watchSyncQueue',
    (response) => {
      state.syncQueue = response.getQueueList().map(entry => processSyncQueueEntry(entry));
      deviceCard.updateAllSyncProgressBars();
      devicePools.updateSyncQueueUI();
      devicePools.updateCounters();
    });
}

function startDeviceStreaming(updateStatusFn) {
  if (state.connectionState.isConnecting) return;
  state.connectionState.isConnecting = true;
  debugLog('Starting device streaming with gRPC', LOG_LEVELS.INFO);

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();

    startDiscoveredCamerasStream(client);
    startManagedCamerasStream(client);
    startSyncQueueStream(client);

    state.connectionState.retryCount = 0;
    state.connectionState.isConnecting = false;

    if (updateStatusFn) updateStatusFn(true);
  } catch (error) {
    debugLog(`Error starting device streams: ${error.message}`, LOG_LEVELS.ERROR);
    state.connectionState.isConnecting = false;
    handleConnectionFailure(error, updateStatusFn);
  }
}

function handleConnectionFailure(error, updateStatusFn) {
  debugLog(`Connection failure: ${error.message}`, LOG_LEVELS.ERROR);
  if (state.connectionState.retryCount < state.connectionState.maxRetries && state.serviceRunning) {
    const delay = calculateBackoffDelay();
    debugLog(`Reconnecting in ${delay}ms (attempt ${state.connectionState.retryCount})`, LOG_LEVELS.INFO);
    setTimeout(() => {
      if (state.serviceRunning) startDeviceStreaming(updateStatusFn);
    }, delay);
  }
}

function cancelAllStreams() {
  debugLog('Cancelling all active streams', LOG_LEVELS.INFO);
  if (state.activeStream) {
    ['discoveredCamerasStream', 'managedCamerasStream', 'syncQueueStream'].forEach(key => {
      if (state.activeStream[key]) {
        state.activeStream[key].intentionalCancel = true;
        state.activeStream[key].cancel();
      }
    });
    state.activeStream = null;
  }
  ['discoveredCamerasStream', 'managedCamerasStream', 'syncQueueStream'].forEach(key => {
    state.streamStatus[key] = 'disconnected';
  });
  if (_onStreamStateChange) _onStreamStateChange();
}

module.exports = {
  processCamera,
  processSyncQueueEntry,
  startDeviceStreaming,
  cancelAllStreams,
  setDebugLog: (fn) => { _debugLog = fn; },
  setOnStreamStateChange: (fn) => { _onStreamStateChange = fn; }
};
