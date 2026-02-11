// gRPC action calls — pair, sync, manage, toggle operations
const { LOG_LEVELS } = require('./fileLogger');
const state = require('./state');

// Lazy-loaded to avoid circular deps
let _devicePools = null;
function devicePools() {
  if (!_devicePools) _devicePools = require('./device-pools');
  return _devicePools;
}
let _deviceCard = null;
function dc() {
  if (!_deviceCard) _deviceCard = require('./device-card');
  return _deviceCard;
}
let _streams = null;
function streams() {
  if (!_streams) _streams = require('./streams');
  return _streams;
}

let _debugLog = null;
let _addLogEntry = null;
let _showToast = null;
function debugLog(msg, level = LOG_LEVELS.INFO) { if (_debugLog) _debugLog(msg, level); }
function addLogEntry(msg, type) { if (_addLogEntry) _addLogEntry(msg, type); }
function showToast(msg, type) { if (_showToast) _showToast(msg, type); }

function pairDevice(macAddress) {
  const device = state.allDevices[macAddress];
  if (!device || !device.id) {
    debugLog(`Cannot pair device, no device ID found for ${macAddress}`, LOG_LEVELS.ERROR);
    resetPairButtonState(macAddress);
    return;
  }

  debugLog(`Pairing device: ${device.name} (${device.id})`, LOG_LEVELS.INFO);
  addLogEntry(`Attempting to pair ${dc().getDeviceDisplayName(device)}`, 'info');

  state.pairingInProgress[macAddress] = true;
  updateDeviceStatus(macAddress, 'pairing');

  // 30s timeout — if backend never responds, reset the UI
  clearTimeout(state.pairingTimeouts[macAddress]);
  state.pairingTimeouts[macAddress] = setTimeout(() => {
    if (state.pairingInProgress[macAddress]) {
      state.pairingInProgress[macAddress] = false;
      resetPairButtonState(macAddress);
      updateDeviceStatus(macAddress, 'discovered');
      showToast(`Pairing timed out for ${dc().getDeviceDisplayName(device)}`, 'error');
      addLogEntry(`Pairing timed out for ${dc().getDeviceDisplayName(device)}`, 'error');
    }
  }, 30000);

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    const pairRequest = new grpcUtils.PairCameraRequest();
    pairRequest.setCameraId(device.id);

    client.pairCamera(pairRequest, (pairError, pairResponse) => {
      clearTimeout(state.pairingTimeouts[macAddress]);
      state.pairingInProgress[macAddress] = false;

      if (pairError) {
        debugLog(`Error pairing device: ${pairError.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to pair ${dc().getDeviceDisplayName(device)}: ${pairError.message}`, 'error');
        showToast(`Failed to pair ${dc().getDeviceDisplayName(device)}`, 'error');
        resetPairButtonState(macAddress);
        updateDeviceStatus(macAddress, 'discovered');
        return;
      }

      if (!pairResponse.getSuccess()) {
        debugLog(`Pairing failed: ${pairResponse.getMessage()}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to pair ${dc().getDeviceDisplayName(device)}: ${pairResponse.getMessage()}`, 'error');
        showToast(`Failed to pair ${dc().getDeviceDisplayName(device)}`, 'error');
        resetPairButtonState(macAddress);
        updateDeviceStatus(macAddress, 'discovered');
        return;
      }

      debugLog(`Successfully paired device: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`Successfully paired ${dc().getDeviceDisplayName(device)}`, 'success');
      showToast(`Paired ${dc().getDeviceDisplayName(device)}`, 'success');

      const pairResultCamera = pairResponse.getCamera();
      if (pairResultCamera) {
        const updatedDevice = streams().processCamera(pairResultCamera);
        if (updatedDevice) {
          devicePools().addOrUpdateDevice(updatedDevice);
        } else {
          device.isPaired = true;
          device.isPairing = false;
          devicePools().addOrUpdateDevice(device);
        }
      } else {
        device.isPaired = true;
        device.isPairing = false;
        devicePools().addOrUpdateDevice(device);
      }

      if (state.autoSync && device.isManaged) {
        addToSyncQueue(macAddress);
      }
    });
  } catch (error) {
    clearTimeout(state.pairingTimeouts[macAddress]);
    debugLog(`Exception during pairing: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error pairing ${dc().getDeviceDisplayName(device)}: ${error.message}`, 'error');
    showToast(`Error pairing ${dc().getDeviceDisplayName(device)}`, 'error');
    state.pairingInProgress[macAddress] = false;
    resetPairButtonState(macAddress);
    updateDeviceStatus(macAddress, 'discovered');
  }
}

function addToSyncQueue(macAddress) {
  const device = state.allDevices[macAddress];
  if (!device || !device.id) {
    debugLog(`Cannot add to sync queue, no device ID found for ${macAddress}`, LOG_LEVELS.ERROR);
    resetSyncButtonState(macAddress);
    return;
  }

  debugLog(`Adding device to sync queue: ${device.name} (${device.id})`, LOG_LEVELS.INFO);
  addLogEntry(`Adding ${dc().getDeviceDisplayName(device)} to sync queue`, 'info');

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    const request = new grpcUtils.ForceSyncRequest();
    request.setCameraId(device.id);

    client.forceSync(request, (error, response) => {
      if (error) {
        debugLog(`Error adding to sync queue: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to add ${dc().getDeviceDisplayName(device)} to sync queue: ${error.message}`, 'error');
        showToast(`Failed to sync ${dc().getDeviceDisplayName(device)}`, 'error');
        resetSyncButtonState(macAddress);
        return;
      }

      if (!response.getSuccess()) {
        debugLog(`Adding to sync queue failed: ${response.getMessage()}`, LOG_LEVELS.ERROR);
        resetSyncButtonState(macAddress);
        return;
      }

      debugLog(`Successfully added device to sync queue: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`Added ${dc().getDeviceDisplayName(device)} to sync queue`, 'success');
      showToast(`Syncing ${dc().getDeviceDisplayName(device)}`, 'success');
      device.isSynced = false;

      const queueEntry = response.getQueueEntry();
      if (queueEntry) {
        const processedEntry = streams().processSyncQueueEntry(queueEntry);
        if (processedEntry) {
          const existingIndex = state.syncQueue.findIndex(e => e.cameraId === processedEntry.cameraId);
          if (existingIndex >= 0) state.syncQueue[existingIndex] = processedEntry;
          else state.syncQueue.push(processedEntry);
          devicePools().updateSyncQueueUI();
          devicePools().updateCounters();
        }
      }
    });
  } catch (error) {
    debugLog(`Exception adding to sync queue: ${error.message}`, LOG_LEVELS.ERROR);
    resetSyncButtonState(macAddress);
  }
}

function cancelSync(macAddress) {
  const device = state.allDevices[macAddress];
  if (!device || !device.id) {
    debugLog(`Cannot cancel sync, no device ID found for ${macAddress}`, LOG_LEVELS.ERROR);
    return;
  }

  debugLog(`Cancelling sync for device: ${device.name} (${device.id})`, LOG_LEVELS.INFO);
  addLogEntry(`Cancelling sync for ${dc().getDeviceDisplayName(device)}`, 'info');

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    const request = new grpcUtils.CancelSyncRequest();
    request.setCameraId(device.id);

    client.cancelSync(request, (error, response) => {
      if (error) {
        debugLog(`Error cancelling sync: ${error.message}`, LOG_LEVELS.ERROR);
        showToast(`Failed to cancel sync: ${error.message}`, 'error');
        devicePools().addOrUpdateDevice(device);
        return;
      }
      if (!response.getSuccess()) {
        debugLog(`Cancelling sync failed: ${response.getMessage()}`, LOG_LEVELS.ERROR);
        showToast(`Failed to cancel sync: ${response.getMessage()}`, 'error');
        devicePools().addOrUpdateDevice(device);
        return;
      }

      debugLog(`Successfully cancelled sync for device: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`Cancelled sync for ${dc().getDeviceDisplayName(device)}`, 'success');
      showToast(`Cancelled sync for ${dc().getDeviceDisplayName(device)}`, 'info');

      device.isSynced = true;
      device.isSyncing = false;
      const idx = state.syncQueue.findIndex(e => e.cameraId === device.id);
      if (idx >= 0) state.syncQueue.splice(idx, 1);

      devicePools().addOrUpdateDevice(device);
      devicePools().updateSyncQueueUI();
      devicePools().updateCounters();
    });
  } catch (error) {
    debugLog(`Exception cancelling sync: ${error.message}`, LOG_LEVELS.ERROR);
    showToast(`Failed to cancel sync: ${error.message}`, 'error');
    devicePools().addOrUpdateDevice(device);
  }
}

function toggleDeviceManaged(macAddress, isManaged) {
  const device = state.allDevices[macAddress];
  if (!device || !device.id) {
    debugLog(`Cannot ${isManaged ? 'manage' : 'unmanage'} device, no ID for ${macAddress}`, LOG_LEVELS.ERROR);
    return;
  }

  debugLog(`${isManaged ? 'Managing' : 'Unmanaging'} device: ${device.name}`, LOG_LEVELS.INFO);
  addLogEntry(`${isManaged ? 'Adding' : 'Removing'} ${dc().getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} camera pool`, 'info');

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();

    const RequestClass = isManaged ? grpcUtils.ManageCameraRequest : grpcUtils.UnmanageCameraRequest;
    const method = isManaged ? 'manageCamera' : 'unmanageCamera';
    const request = new RequestClass();
    request.setCameraId(device.id);

    client[method](request, (error, response) => {
      if (error) {
        debugLog(`Error ${isManaged ? 'managing' : 'unmanaging'} device: ${error.message}`, LOG_LEVELS.ERROR);
        return;
      }
      if (!response.getSuccess()) {
        debugLog(`${isManaged ? 'Managing' : 'Unmanaging'} failed: ${response.getMessage()}`, LOG_LEVELS.ERROR);
        return;
      }

      debugLog(`Successfully ${isManaged ? 'managed' : 'unmanaged'} device: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`${isManaged ? 'Added' : 'Removed'} ${dc().getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} camera pool`, 'success');
      showToast(`${isManaged ? 'Added' : 'Removed'} ${dc().getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} pool`, 'success');

      device.isManaged = isManaged;
      if (isManaged) device.isPaired = true;

      if (isManaged && response.getCamera) {
        const managedCamera = response.getCamera();
        if (managedCamera) {
          const updatedDevice = streams().processCamera(managedCamera);
          if (updatedDevice) { devicePools().addOrUpdateDevice(updatedDevice); return; }
        }
      }
      devicePools().addOrUpdateDevice(device);

      if (isManaged && state.autoPair && !device.isPaired) {
        pairDevice(macAddress);
      }
    });
  } catch (error) {
    debugLog(`Exception ${isManaged ? 'managing' : 'unmanaging'} device: ${error.message}`, LOG_LEVELS.ERROR);
  }
}

function togglePairAll(event) {
  const enabled = event.target.checked;
  const previousValue = state.autoPair;
  debugLog(`Toggling pair-all mode: ${enabled}`, LOG_LEVELS.INFO);
  state.autoPair = enabled;

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    const request = new grpcUtils.UpdateSettingRequest();
    request.setSettingName('pair_mode_enabled');
    request.setBoolValue(enabled);

    client.updateSetting(request, (error) => {
      if (error) {
        debugLog(`Error updating pair_mode_enabled: ${error.message}`, LOG_LEVELS.ERROR);
        state.autoPair = previousValue;
        if (event.target) event.target.checked = previousValue;
        return;
      }
      if (enabled) {
        addLogEntry('Auto-pairing enabled', 'info');
        Object.values(state.allDevices)
          .filter(d => !d.isPaired && !state.pairingInProgress[d.macAddress])
          .forEach(d => pairDevice(d.macAddress));
      } else {
        addLogEntry('Auto-pairing disabled', 'info');
      }
    });
  } catch (error) {
    debugLog(`Exception updating pair_mode_enabled: ${error.message}`, LOG_LEVELS.ERROR);
    state.autoPair = previousValue;
    if (event.target) event.target.checked = previousValue;
  }
}

function toggleAutoSync(event) {
  const enabled = event.target.checked;
  const previousValue = state.autoSync;
  debugLog(`Toggling auto-sync mode: ${enabled}`, LOG_LEVELS.INFO);
  state.autoSync = enabled;

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    const request = new grpcUtils.UpdateSettingRequest();
    request.setSettingName('sync_enabled');
    request.setBoolValue(enabled);

    client.updateSetting(request, (error) => {
      if (error) {
        debugLog(`Error updating sync_enabled: ${error.message}`, LOG_LEVELS.ERROR);
        state.autoSync = previousValue;
        if (event.target) event.target.checked = previousValue;
        return;
      }
      addLogEntry(enabled ? 'Auto-sync enabled' : 'Auto-sync disabled', 'info');
    });
  } catch (error) {
    debugLog(`Exception updating sync_enabled: ${error.message}`, LOG_LEVELS.ERROR);
    state.autoSync = previousValue;
    if (event.target) event.target.checked = previousValue;
  }
}

function resetPairButtonState(macAddress) {
  const el = document.querySelector(`.device-card[data-mac="${macAddress}"]`);
  if (el) {
    const btn = el.querySelector('.pair-button');
    if (btn) { btn.textContent = "Pair"; btn.disabled = false; btn.classList.remove('in-progress'); }
  }
}

function resetSyncButtonState(macAddress) {
  const el = document.querySelector(`.device-card[data-mac="${macAddress}"]`);
  if (el) {
    const btn = el.querySelector('.sync-button');
    if (btn) { btn.textContent = "Sync"; btn.disabled = false; btn.classList.remove('in-progress'); }
  }
}

function updateDeviceStatus(macAddress, status) {
  const device = state.allDevices[macAddress];
  if (!device) return;
  if (status === 'paired') { device.isPairing = false; device.isPaired = true; }
  device.visualStatus = status;
  devicePools().addOrUpdateDevice(device);
  // Highlight status change
  setTimeout(() => {
    document.querySelectorAll(`.device-card[data-mac="${macAddress}"]`).forEach(card => {
      card.classList.add('status-changed');
      setTimeout(() => card.classList.remove('status-changed'), 1500);
    });
  }, 100);
}

module.exports = {
  pairDevice,
  addToSyncQueue,
  cancelSync,
  toggleDeviceManaged,
  togglePairAll,
  toggleAutoSync,
  resetPairButtonState,
  resetSyncButtonState,
  updateDeviceStatus,
  setDebugLog: (fn) => { _debugLog = fn; },
  setAddLogEntry: (fn) => { _addLogEntry = fn; },
  setShowToast: (fn) => { _showToast = fn; }
};
