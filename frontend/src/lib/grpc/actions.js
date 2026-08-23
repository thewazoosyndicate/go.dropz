// gRPC unary action calls
import { getClient, proto } from './client.js';
import { processCamera, processSyncQueueEntry } from './streams.js';
import { updateDevice, setPairingInProgress, getAllDevices } from '../stores/devices.svelte.js';
import { addOrUpdateSyncEntry, removeSyncEntry } from '../stores/sync.svelte.js';
import { setAppConfig, setAutoPair, setAutoSync, getAppConfig, updateConfigField } from '../stores/config.svelte.js';
import { addToast, addLog } from '../stores/ui.svelte.js';
import { setVideos, setLoading } from '../stores/videos.svelte.js';
import { setGroups, addOrUpdateGroup, removeGroup as removeGroupFromStore } from '../stores/groups.svelte.js';

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
          statusCheckIntervalSeconds: config.getStatusCheckIntervalSeconds(),
          checkOnReturn: config.getCheckOnReturn(),
          logLevel: config.getLogLevel()
        });
      } catch (e) {
        setAppConfig({
          pairModeEnabled: false, syncEnabled: false,
          scanIntervalSeconds: 30, connectTimeoutSeconds: 30,
          daysThreshold: 7, destinationFolder: '',
          setTimeEnabled: true, inactivityTimeoutSeconds: 60,
          statusCheckIntervalSeconds: 300, checkOnReturn: true,
          logLevel: 'info'
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
      statusCheckIntervalSeconds: config.getStatusCheckIntervalSeconds(),
      checkOnReturn: config.getCheckOnReturn(),
      logLevel: config.getLogLevel()
    });

    addToast('Settings reset to defaults', 'success');
  });
}

export function loadVideos(cameraId = '', limit = 200, offset = 0) {
  setLoading(true);
  const client = getClient();
  const request = new proto.GetVideosRequest();
  request.setCameraId(cameraId);
  request.setLimit(limit);
  request.setOffset(offset);

  client.getVideos(request, (error, response) => {
    setLoading(false);
    if (error) {
      console.error('Error loading videos:', error.message);
      return;
    }

    const videos = response.getVideosList().map(v => ({
      id: v.getId(),
      name: v.getName(),
      path: v.getPath(),
      sizeBytes: v.getSizeBytes(),
      createdAt: v.getCreatedAt()?.toDate(),
      cameraId: v.getCameraId(),
      mimeType: v.getMimeType(),
      thumbnailPath: v.getThumbnailPath(),
    }));

    setVideos(videos, response.getTotalCount());
  });
}

function processGroup(g) {
  return {
    id: g.getId(),
    name: g.getName(),
    cameraIds: g.getCameraIdsList(),
    createdAt: g.getCreatedAt()?.toDate(),
    updatedAt: g.getUpdatedAt()?.toDate(),
  };
}

export function loadGroups() {
  const client = getClient();
  const request = new proto.GetGroupsRequest();

  client.getGroups(request, (error, response) => {
    if (error) {
      console.error('Error loading groups:', error.message);
      return;
    }
    setGroups(response.getGroupsList().map(processGroup));
  });
}

export function saveManagedAsGroup(name) {
  const client = getClient();
  const request = new proto.SaveManagedAsGroupRequest();
  request.setName(name);

  client.saveManagedAsGroup(request, (error, response) => {
    if (error) {
      addToast('Failed to save group', 'error');
      return;
    }
    addOrUpdateGroup(processGroup(response));
    addToast(`Group "${name}" saved`, 'success');
  });
}

export function loadGroup(groupId) {
  const client = getClient();
  const request = new proto.LoadGroupRequest();
  request.setGroupId(groupId);

  client.loadGroup(request, (error, response) => {
    if (error || !response.getSuccess()) {
      addToast('Failed to load group', 'error');
      return;
    }
    addToast('Group loaded', 'success');
  });
}

export function deleteGroup(groupId) {
  const client = getClient();
  const request = new proto.DeleteGroupRequest();
  request.setGroupId(groupId);

  client.deleteGroup(request, (error, response) => {
    if (error || !response.getSuccess()) {
      addToast('Failed to delete group', 'error');
      return;
    }
    removeGroupFromStore(groupId);
    addToast('Group deleted', 'info');
  });
}

export function renameGroup(groupId, name, cameraIds) {
  const client = getClient();
  const request = new proto.UpdateGroupRequest();
  request.setGroupId(groupId);
  request.setName(name);
  request.setCameraIdsList(cameraIds);

  client.updateGroup(request, (error, response) => {
    if (error) {
      addToast('Failed to rename group', 'error');
      return;
    }
    addOrUpdateGroup(processGroup(response));
  });
}

// Camera settings over BLE. Promise-based: the modal drives its own state.
export function fetchCameraSettings(cameraId, refresh) {
  return new Promise((resolve, reject) => {
    const client = getClient();
    const request = new proto.GetCameraSettingsRequest();
    request.setCameraId(cameraId);
    request.setRefresh(!!refresh);
    // BLE refresh connects to the camera; allow up to 45s
    const deadline = new Date(Date.now() + (refresh ? 45000 : 5000));
    client.getCameraSettings(request, { deadline }, (error, response) => {
      if (error) return reject(error);
      resolve({
        updatedAt: response.getUpdatedAt()?.toDate() || null,
        settings: response.getSettingsList().map(s => ({
          id: s.getId(),
          name: s.getName(),
          value: s.getValue(),
          valueName: s.getValueName(),
          options: s.getOptionsList().map(o => ({ value: o.getValue(), name: o.getName() })),
        })),
      });
    });
  });
}

export function applyCameraSettings({ cameraId, groupId, changes }) {
  return new Promise((resolve, reject) => {
    const client = getClient();
    const request = new proto.ApplyCameraSettingsRequest();
    if (cameraId) request.setCameraId(cameraId);
    if (groupId) request.setGroupId(groupId);
    request.setChangesList(changes.map(c => {
      const change = new proto.SettingChange();
      change.setId(c.id);
      change.setValue(c.value);
      return change;
    }));
    // Group applies run one BLE session per camera
    const deadline = new Date(Date.now() + 120000);
    client.applyCameraSettings(request, { deadline }, (error, response) => {
      if (error) return reject(error);
      resolve(response.getCamerasList().map(c => ({
        cameraId: c.getCameraId(),
        error: c.getError(),
        results: c.getResultsList().map(r => ({ id: r.getId(), error: r.getError() })),
      })));
    });
  });
}

// Media browser: cached catalog of what is on a camera.
export function fetchCameraMedia(cameraId) {
  return new Promise((resolve, reject) => {
    const client = getClient();
    const request = new proto.GetCameraMediaRequest();
    request.setCameraId(cameraId);
    client.getCameraMedia(request, (error, response) => {
      if (error) return reject(error);
      resolve({
        updatedAt: response.getUpdatedAt()?.toDate() || null,
        items: response.getItemsList().map(i => ({
          name: i.getName(),
          cameraPath: i.getCameraPath(),
          sizeBytes: i.getSizeBytes(),
          createdAt: i.getCreatedAt()?.toDate(),
          thumbnailPath: i.getThumbnailPath(),
          downloaded: i.getDownloaded(),
        })),
      });
    });
  });
}

// Empty fileNames queues a catalog-only refresh.
export function requestMediaDownload(cameraId, fileNames) {
  return new Promise((resolve, reject) => {
    const client = getClient();
    const request = new proto.RequestMediaDownloadRequest();
    request.setCameraId(cameraId);
    request.setFileNamesList(fileNames || []);
    client.requestMediaDownload(request, (error, response) => {
      if (error) return reject(error);
      resolve(processSyncQueueEntry(response.getEntry()));
    });
  });
}
