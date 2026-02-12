// gRPC unary action calls
import { getClient, proto } from './client.js';
import { processCamera, processSyncQueueEntry } from './streams.js';
import { updateDevice, setPairingInProgress, getAllDevices } from '../stores/devices.svelte.js';
import { addOrUpdateSyncEntry, removeSyncEntry } from '../stores/sync.svelte.js';
import { setAppConfig, setAutoPair, setAutoSync, getAppConfig, updateConfigField } from '../stores/config.svelte.js';
import { addToast, addLog } from '../stores/ui.svelte.js';

function getDisplayName(device) {
  if (device.wifiSsid?.trim()) return device.wifiSsid.substring(0, 12);
  return device.name || 'Unknown GoPro';
}

export function pairDevice(macAddress) {
  const devices = getAllDevices();
  const device = devices[macAddress];
  if (!device?.id) return;

  setPairingInProgress(macAddress, true);
  addLog(`Pairing ${getDisplayName(device)}...`, 'info');

  const client = getClient();
  const request = new proto.PairCameraRequest();
  request.setCameraId(device.id);

  const timeout = setTimeout(() => {
    setPairingInProgress(macAddress, false);
    addToast(`Pairing timed out for ${getDisplayName(device)}`, 'error');
  }, 30000);

  client.pairCamera(request, (error, response) => {
    clearTimeout(timeout);
    setPairingInProgress(macAddress, false);

    if (error || !response.getSuccess()) {
      const msg = error ? error.message : response.getMessage();
      addToast(`Failed to pair ${getDisplayName(device)}`, 'error');
      addLog(`Pairing failed: ${msg}`, 'error');
      return;
    }

    addToast(`Paired ${getDisplayName(device)}`, 'success');
    addLog(`Paired ${getDisplayName(device)}`, 'success');

    const resultCamera = response.getCamera();
    if (resultCamera) {
      const updated = processCamera(resultCamera);
      if (updated) updateDevice(updated);
    } else {
      device.isPaired = true;
      device.isPairing = false;
      updateDevice({ ...device });
    }
  });
}

export function addToSyncQueue(macAddress) {
  const devices = getAllDevices();
  const device = devices[macAddress];
  if (!device?.id) return;

  addLog(`Adding ${getDisplayName(device)} to sync queue`, 'info');

  const client = getClient();
  const request = new proto.ForceSyncRequest();
  request.setCameraId(device.id);

  client.forceSync(request, (error, response) => {
    if (error || !response.getSuccess()) {
      addToast(`Failed to sync ${getDisplayName(device)}`, 'error');
      return;
    }

    addToast(`Syncing ${getDisplayName(device)}`, 'success');
    device.isSynced = false;

    const queueEntry = response.getQueueEntry();
    if (queueEntry) {
      const processed = processSyncQueueEntry(queueEntry);
      if (processed) addOrUpdateSyncEntry(processed);
    }
  });
}

export function cancelSync(macAddress) {
  const devices = getAllDevices();
  const device = devices[macAddress];
  if (!device?.id) return;

  addLog(`Cancelling sync for ${getDisplayName(device)}`, 'info');

  const client = getClient();
  const request = new proto.CancelSyncRequest();
  request.setCameraId(device.id);

  client.cancelSync(request, (error, response) => {
    if (error || !response.getSuccess()) {
      addToast(`Failed to cancel sync`, 'error');
      return;
    }

    addToast(`Cancelled sync for ${getDisplayName(device)}`, 'info');
    device.isSynced = true;
    device.isSyncing = false;
    removeSyncEntry(device.id);
    updateDevice({ ...device });
  });
}

export function toggleDeviceManaged(macAddress, isManaged) {
  const devices = getAllDevices();
  const device = devices[macAddress];
  if (!device?.id) return;

  const client = getClient();
  const RequestClass = isManaged ? proto.ManageCameraRequest : proto.UnmanageCameraRequest;
  const method = isManaged ? 'manageCamera' : 'unmanageCamera';
  const request = new RequestClass();
  request.setCameraId(device.id);

  client[method](request, (error, response) => {
    if (error || !response.getSuccess()) return;

    const action = isManaged ? 'Added' : 'Removed';
    addToast(`${action} ${getDisplayName(device)} ${isManaged ? 'to' : 'from'} pool`, 'success');

    device.isManaged = isManaged;
    if (isManaged) device.isPaired = true;

    if (isManaged && response.getCamera) {
      const managedCamera = response.getCamera();
      if (managedCamera) {
        const updated = processCamera(managedCamera);
        if (updated) { updateDevice(updated); return; }
      }
    }
    updateDevice({ ...device });
  });
}

export function togglePairAll(enabled) {
  const previousValue = !enabled;
  setAutoPair(enabled);

  const client = getClient();
  const request = new proto.UpdateSettingRequest();
  request.setSettingName('pair_mode_enabled');
  request.setBoolValue(enabled);

  client.updateSetting(request, (error) => {
    if (error) {
      setAutoPair(previousValue);
      return;
    }
    addLog(enabled ? 'Auto-pairing enabled' : 'Auto-pairing disabled', 'info');
    if (enabled) {
      const devices = getAllDevices();
      Object.values(devices)
        .filter(d => !d.isPaired)
        .forEach(d => pairDevice(d.macAddress));
    }
  });
}

export function toggleAutoSync(enabled) {
  const previousValue = !enabled;
  setAutoSync(enabled);

  const client = getClient();
  const request = new proto.UpdateSettingRequest();
  request.setSettingName('sync_enabled');
  request.setBoolValue(enabled);

  client.updateSetting(request, (error) => {
    if (error) {
      setAutoSync(previousValue);
      return;
    }
    addLog(enabled ? 'Auto-sync enabled' : 'Auto-sync disabled', 'info');
  });
}

export function loadConfig() {
  return new Promise((resolve) => {
    const client = getClient();
    const request = new proto.GetConfigRequest();

    client.getConfig(request, (error, response) => {
      if (error) {
        console.error('Error loading config:', error.message);
        resolve();
        return;
      }

      const config = response.getConfig();
      try {
        setAppConfig({
          pairModeEnabled: config.getPairModeEnabled(),
          syncEnabled: config.getSyncEnabled(),
          scanIntervalSeconds: config.getScanIntervalSeconds(),
          connectTimeoutSeconds: config.getConnectTimeoutSeconds(),
          daysThreshold: config.getDaysThreshold(),
          destinationFolder: config.getDestinationFolder(),
          setTimeEnabled: config.getSetTimeEnabled(),
          inactivityTimeoutSeconds: config.getInactivityTimeoutSeconds(),
          inactivitySyncIntervalSeconds: config.getInactivitySyncIntervalSeconds(),
          logLevel: config.getLogLevel()
        });
      } catch (e) {
        setAppConfig({
          pairModeEnabled: false, syncEnabled: false,
          scanIntervalSeconds: 30, connectTimeoutSeconds: 30,
          daysThreshold: 7, destinationFolder: '',
          setTimeEnabled: true, inactivityTimeoutSeconds: 60,
          inactivitySyncIntervalSeconds: 600, logLevel: 'info'
        });
      }
      resolve();
    });
  });
}

export function updateSetting(key, value) {
  const client = getClient();
  const request = new proto.UpdateSettingRequest();
  request.setSettingName(key);

  if (typeof value === 'boolean') request.setBoolValue(value);
  else if (typeof value === 'number') request.setIntValue(value);
  else request.setStringValue(String(value));

  client.updateSetting(request, (error) => {
    if (error) {
      addToast('Failed to update setting', 'error');
      return;
    }
    updateConfigField(key, value);
  });
}

export function resetAllSettings() {
  const client = getClient();
  const request = new proto.ResetSettingRequest();
  request.setSettingName('all');

  client.resetSetting(request, (error, config) => {
    if (error) {
      addToast('Failed to reset settings', 'error');
      return;
    }

    setAppConfig({
      pairModeEnabled: config.getPairModeEnabled(),
      syncEnabled: config.getSyncEnabled(),
      scanIntervalSeconds: config.getScanIntervalSeconds(),
      connectTimeoutSeconds: config.getConnectTimeoutSeconds(),
      daysThreshold: config.getDaysThreshold(),
      destinationFolder: config.getDestinationFolder(),
      setTimeEnabled: config.getSetTimeEnabled(),
      inactivityTimeoutSeconds: config.getInactivityTimeoutSeconds(),
      inactivitySyncIntervalSeconds: config.getInactivitySyncIntervalSeconds(),
      logLevel: config.getLogLevel()
    });

    addToast('Settings reset to defaults', 'success');
  });
}
