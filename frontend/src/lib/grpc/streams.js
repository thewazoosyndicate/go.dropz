// gRPC stream management — writes directly to Svelte stores
import { getClient, proto } from './client.js';
import { updateDevice } from '../stores/devices.svelte.js';
import { setSyncQueue } from '../stores/sync.svelte.js';
import { getConnectionState, setStreamStatus, getServiceRunning } from '../stores/connection.svelte.js';

let activeStreams = {};

function processCamera(camera) {
  try {
    const cameraState = camera.getCameraState();
    if (!cameraState) return null;
    const cameraData = cameraState.getCamera();
    if (!cameraData) return null;
    const macAddress = cameraData.getBleAddress();
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
      inPairingMode: status ? status.getInPairingMode() : false,
      groupId: cameraState.getGroupId(),
      batteryLevel: metadata ? metadata.getBatteryLevel() : null,
      firmwareVersion: metadata ? metadata.getFirmwareVersion() : null,
      model: metadata ? metadata.getModel() : null,
      serialNumber: metadata ? metadata.getSerialNumber() : null,
      numPhotos: metadata ? metadata.getNumPhotos() : null,
      numVideos: metadata ? metadata.getNumVideos() : null,
      remainingSpaceKb: metadata ? metadata.getRemainingSpaceKb() : null,
      lastSyncError: status ? status.getLastSyncError() : null,
    };
  } catch (error) {
    console.error('Error processing camera:', error);
    return null;
  }
}

export function processSyncQueueEntry(entry) {
  try {
    return {
      cameraId: entry.getCameraId(),
      queuedAt: entry.getQueuedAt()?.toDate(),
      priority: entry.getPriority(),
      progressPercent: entry.getProgressPercent(),
      currentOperation: entry.getCurrentOperation(),
      fileIndex: entry.getFileIndex(),
      fileCount: entry.getFileCount(),
      fileName: entry.getFileName(),
      fileBytes: entry.getFileBytes(),
      fileTotal: entry.getFileTotal(),
      bytesDone: entry.getBytesDone(),
      bytesTotal: entry.getBytesTotal(),
      rateBps: entry.getRateBps()
    };
  } catch (error) {
    console.error('Error processing sync queue entry:', error);
    return null;
  }
}

function calculateBackoffDelay() {
  const conn = getConnectionState();
  const delay = conn.baseDelay * Math.pow(2, conn.retryCount);
  conn.retryCount = Math.min(conn.retryCount + 1, conn.maxRetries);
  return Math.min(delay, 30000);
}

function startStream(streamKey, RequestClass, watchMethod, onData) {
  try {
    const client = getClient();
    const request = new RequestClass();
    request.setChangeCounter(0);

    const call = client[watchMethod](request);
    activeStreams[streamKey] = { call, intentionalCancel: false };

    call.on('data', (response) => {
      setStreamStatus(streamKey, 'connected');
      if (response.getHeartbeat() || response.getNoChanges()) return;
      onData(response);
    });

    call.on('error', (error) => {
      setStreamStatus(streamKey, 'reconnecting');
      console.error(`${streamKey} stream error:`, error.message);
      if (activeStreams[streamKey]?.intentionalCancel) return;
      setTimeout(() => {
        if (getServiceRunning()) startStream(streamKey, RequestClass, watchMethod, onData);
      }, calculateBackoffDelay());
    });

    call.on('end', () => {
      setStreamStatus(streamKey, 'reconnecting');
      if (!activeStreams[streamKey]?.intentionalCancel && getServiceRunning()) {
        setTimeout(() => startStream(streamKey, RequestClass, watchMethod, onData), calculateBackoffDelay());
      }
    });
  } catch (error) {
    console.error(`Failed to start ${streamKey} stream:`, error);
  }
}

export function startDeviceStreaming() {
  const conn = getConnectionState();
  if (conn.isConnecting) return;
  conn.isConnecting = true;

  try {
    startStream('discoveredCameras', proto.GetDiscoveredCamerasRequest, 'watchDiscoveredCameras', (response) => {
      response.getCamerasList().forEach(camera => {
        const obj = processCamera(camera);
        if (obj) updateDevice(obj);
      });
    });

    startStream('managedCameras', proto.GetManagedCamerasRequest, 'watchManagedCameras', (response) => {
      response.getCamerasList().forEach(camera => {
        const obj = processCamera(camera);
        if (obj) updateDevice(obj);
      });
    });

    startStream('syncQueue', proto.GetSyncQueueRequest, 'watchSyncQueue', (response) => {
      const entries = response.getQueueList().map(entry => processSyncQueueEntry(entry)).filter(Boolean);
      setSyncQueue(entries);
    });

    conn.retryCount = 0;
    conn.isConnecting = false;
  } catch (error) {
    console.error('Error starting device streams:', error);
    conn.isConnecting = false;
  }
}

export function cancelAllStreams() {
  for (const key of Object.keys(activeStreams)) {
    if (activeStreams[key]) {
      activeStreams[key].intentionalCancel = true;
      try { activeStreams[key].call.cancel(); } catch (_) {}
    }
    setStreamStatus(key, 'disconnected');
  }
  activeStreams = {};
}

// Re-export for use in actions
export { processCamera };
