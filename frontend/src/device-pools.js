// Pool UI management — adding, updating, removing devices from the three panels
const { LOG_LEVELS } = require('./fileLogger');
const state = require('./state');
const deviceCard = require('./device-card');

let _debugLog = null;
function debugLog(msg, level = LOG_LEVELS.INFO) {
  if (_debugLog) return _debugLog(msg, level);
}

// DOM element references (set once from renderer)
let pairQueueContainer, camerasPoolContainer, syncQueueContainer;
let managedCountDisplay, unpairedCountDisplay, syncCountDisplay;

function setContainers(containers) {
  pairQueueContainer = containers.pairQueueContainer;
  camerasPoolContainer = containers.camerasPoolContainer;
  syncQueueContainer = containers.syncQueueContainer;
  managedCountDisplay = containers.managedCountDisplay;
  unpairedCountDisplay = containers.unpairedCountDisplay;
  syncCountDisplay = containers.syncCountDisplay;
}

function getPoolContainer(poolType) {
  switch (poolType) {
    case 'discovered': return pairQueueContainer;
    case 'managed': return camerasPoolContainer;
    case 'sync': return syncQueueContainer;
    default: return null;
  }
}

function addOrUpdateDevice(device) {
  if (!device || !device.macAddress) return null;
  if (!device.name) device.name = device.model || 'Unnamed Camera';

  const existingDevice = state.allDevices[device.macAddress];
  const isNewDevice = !existingDevice;

  let rssiChanged = false;
  let lastSeenChanged = false;
  let significantChange = false;

  if (!isNewDevice) {
    rssiChanged = existingDevice.rssi !== device.rssi;
    lastSeenChanged = existingDevice.lastSeen !== device.lastSeen;
    significantChange =
      existingDevice.isReachable !== device.isReachable ||
      existingDevice.isPaired !== device.isPaired ||
      existingDevice.isManaged !== device.isManaged ||
      existingDevice.isSyncing !== device.isSyncing ||
      existingDevice.name !== device.name ||
      existingDevice.visualStatus !== device.visualStatus;
  }

  if (isNewDevice) {
    debugLog(`New device: ${device.name} (${device.macAddress})`, LOG_LEVELS.INFO);
    significantChange = true;
  } else if (significantChange) {
    debugLog(`Device changed: ${device.name} — status=${device.visualStatus}, paired=${device.isPaired}, managed=${device.isManaged}`, LOG_LEVELS.DEBUG);
  }

  // Preserve newer timestamps
  if (existingDevice?.lastSeen && device.lastSeen && existingDevice.lastSeen > device.lastSeen) {
    device.lastSeen = existingDevice.lastSeen;
  }

  const poolMembership = {
    discovered: deviceCard.shouldShowInDiscoveredPool(device),
    managed: deviceCard.shouldShowInManagedPool(device),
    sync: deviceCard.shouldShowInSyncQueue(device)
  };

  const previousMembership = state.uiState.poolMembership[device.macAddress] || {};
  const membershipChanged = isNewDevice ||
    previousMembership.discovered !== poolMembership.discovered ||
    previousMembership.managed !== poolMembership.managed ||
    previousMembership.sync !== poolMembership.sync;

  state.allDevices[device.macAddress] = device;

  if (isNewDevice || membershipChanged || significantChange) {
    updateDeviceInPools(device, poolMembership, isNewDevice, membershipChanged);
    state.uiState.poolMembership[device.macAddress] = poolMembership;
    updateCounters();
  } else if (rssiChanged) {
    deviceCard.updateSignalStrengthForDevice(device.macAddress, device.rssi);
  } else if (lastSeenChanged) {
    updateDeviceTimestamp(device.macAddress, device.lastSeen);
  } else {
    updateDeviceElementInPlace(device);
    state.uiState.poolMembership[device.macAddress] = poolMembership;
  }

  return device;
}

function updateDeviceInPools(device, poolMembership, isNewDevice, membershipChanged) {
  const macAddress = device.macAddress;

  if (membershipChanged && !isNewDevice) {
    const oldMembership = state.uiState.poolMembership[macAddress] || {};
    if (oldMembership.discovered && !poolMembership.discovered) removeDeviceFromPool(macAddress, 'discovered');
    if (oldMembership.managed && !poolMembership.managed) removeDeviceFromPool(macAddress, 'managed');
    if (oldMembership.sync && !poolMembership.sync) removeDeviceFromPool(macAddress, 'sync');
  }

  if (poolMembership.discovered) updateDeviceInPool(device, 'discovered');
  if (poolMembership.managed) updateDeviceInPool(device, 'managed');
  if (poolMembership.sync) updateDeviceInPool(device, 'sync');

  if (!poolMembership.discovered) removeDeviceFromPool(device.macAddress, 'discovered');

  if (membershipChanged) cleanupDeviceElements(macAddress, poolMembership);
}

function updateDeviceInPool(device, poolType) {
  const container = getPoolContainer(poolType);
  if (!container) return;

  let gridContainer = container.querySelector('.gopro-grid-container');
  if (!gridContainer) {
    container.innerHTML = '';
    gridContainer = document.createElement('div');
    gridContainer.className = 'gopro-grid-container';
    container.appendChild(gridContainer);
  }

  const existingElement = gridContainer.querySelector(`.device-card[data-mac="${device.macAddress}"]`);
  if (existingElement) {
    deviceCard.updateDeviceElement(existingElement, device, poolType === 'sync');
  } else {
    const el = deviceCard.createGoProElement(device, poolType === 'sync');
    insertDeviceInCorrectPosition(gridContainer, el, device, poolType);
    if (!state.uiState.deviceElements[device.macAddress]) state.uiState.deviceElements[device.macAddress] = {};
    state.uiState.deviceElements[device.macAddress][poolType] = el;
    removeEmptyState(gridContainer);
  }
}

function removeDeviceFromPool(macAddress, poolType) {
  const container = getPoolContainer(poolType);
  if (!container) return;
  const element = container.querySelector(`.device-card[data-mac="${macAddress}"]`);
  if (element) {
    element.remove();
    if (state.uiState.deviceElements[macAddress]) {
      delete state.uiState.deviceElements[macAddress][poolType];
      if (Object.keys(state.uiState.deviceElements[macAddress]).length === 0) delete state.uiState.deviceElements[macAddress];
    }
    const gridContainer = container.querySelector('.gopro-grid-container');
    if (gridContainer && gridContainer.children.length === 0) showEmptyState(gridContainer, poolType);
  }
}

function insertDeviceInCorrectPosition(gridContainer, deviceElement, device, poolType) {
  const children = Array.from(gridContainer.children);
  let insertIndex = children.length;

  if (poolType === 'discovered') {
    for (let i = 0; i < children.length; i++) {
      const childDevice = state.allDevices[children[i].getAttribute('data-mac')];
      if (childDevice && (device.rssi || -100) > (childDevice.rssi || -100)) { insertIndex = i; break; }
    }
  } else {
    for (let i = 0; i < children.length; i++) {
      const childDevice = state.allDevices[children[i].getAttribute('data-mac')];
      if (childDevice && (deviceCard.getDeviceDisplayName(device) || '').localeCompare(deviceCard.getDeviceDisplayName(childDevice) || '') < 0) { insertIndex = i; break; }
    }
  }

  if (insertIndex >= children.length) gridContainer.appendChild(deviceElement);
  else gridContainer.insertBefore(deviceElement, children[insertIndex]);
}

function showEmptyState(gridContainer, poolType) {
  if (!gridContainer) return;
  const emptyState = document.createElement('div');
  emptyState.className = 'empty-state';
  const logo = document.createElement('img');
  logo.src = 'imgs/3_dropz.svg';
  logo.alt = 'Dropz Logo';
  logo.className = 'empty-state-logo';
  const message = document.createElement('p');
  const messages = {
    'discovered': 'No GoPro devices detected',
    'managed': 'No managed GoPro devices',
    'sync': 'No GoPro devices in sync queue'
  };
  message.textContent = messages[poolType] || 'No devices';
  emptyState.appendChild(logo);
  emptyState.appendChild(message);
  gridContainer.appendChild(emptyState);
}

function removeEmptyState(gridContainer) {
  if (!gridContainer) return;
  const emptyState = gridContainer.querySelector('.empty-state');
  if (emptyState) emptyState.remove();
}

function updateCounters() {
  const devices = Object.values(state.allDevices);
  const managedCount = devices.filter(d => deviceCard.shouldShowInManagedPool(d)).length;
  const discoveredCount = devices.filter(d => deviceCard.shouldShowInDiscoveredPool(d)).length;
  const syncCount = devices.filter(d => deviceCard.shouldShowInSyncQueue(d)).length;

  if (managedCountDisplay) managedCountDisplay.textContent = managedCount;
  if (unpairedCountDisplay) unpairedCountDisplay.textContent = discoveredCount;
  if (syncCountDisplay) syncCountDisplay.textContent = syncCount;

  state.uiState.currentDeviceCounts = {
    total: Object.keys(state.allDevices).length,
    discovered: discoveredCount,
    managed: managedCount,
    sync: syncCount
  };
}

function cleanupDisconnectedDevices() {
  const now = new Date();
  const timeout = 30000;

  for (const mac in state.allDevices) {
    const device = state.allDevices[mac];
    const lastSeen = new Date(device.lastSeen);
    if (now - lastSeen <= timeout) continue;

    const membership = state.uiState.poolMembership[mac] || {};

    if (device.isManaged && device.isPaired) {
      device.isReachable = false;
      device.rssi = -100;
      device.visualStatus = 'Unreachable';
      if (membership.managed && state.uiState.deviceElements[mac]?.managed) {
        deviceCard.updateDeviceElement(state.uiState.deviceElements[mac].managed, device, false);
      }
      if (membership.discovered) removeDeviceFromPool(mac, 'discovered');
      if (membership.sync) removeDeviceFromPool(mac, 'sync');
      state.uiState.poolMembership[mac] = { discovered: false, managed: true, sync: false };
    } else {
      if (membership.discovered) removeDeviceFromPool(mac, 'discovered');
      if (membership.managed) removeDeviceFromPool(mac, 'managed');
      if (membership.sync) removeDeviceFromPool(mac, 'sync');
      delete state.allDevices[mac];
      delete state.uiState.poolMembership[mac];
      delete state.uiState.deviceElements[mac];
    }

    if (state.syncQueue.some(entry => entry.cameraId === device.id)) {
      state.syncQueue = state.syncQueue.filter(entry => entry.cameraId !== device.id);
    }
    delete state.pairingInProgress[mac];
  }

  updateCounters();
}

function updateDeviceTimestamp(macAddress, lastSeen) {
  const elements = state.uiState.deviceElements[macAddress];
  if (elements) {
    Object.values(elements).forEach(element => {
      const statusElement = element.querySelector('.device-status');
      if (statusElement && lastSeen) {
        statusElement.textContent = `Last seen: ${deviceCard.formatTimeSince(lastSeen)}`;
      }
    });
  }
}

function updateDeviceElementInPlace(device) {
  const elements = state.uiState.deviceElements[device.macAddress];
  if (elements) {
    Object.entries(elements).forEach(([poolType, element]) => {
      deviceCard.updateDeviceElement(element, device, poolType === 'sync');
    });
  }
}

function cleanupDeviceElements(macAddress, currentMembership) {
  if (state.uiState.deviceElements[macAddress]) {
    Object.keys(state.uiState.deviceElements[macAddress]).forEach(poolType => {
      if (!currentMembership[poolType]) delete state.uiState.deviceElements[macAddress][poolType];
    });
    if (Object.keys(state.uiState.deviceElements[macAddress]).length === 0) delete state.uiState.deviceElements[macAddress];
  }
}

// Full rebuild functions (used by view toggle and sync queue stream)
function updatePairQueueUI() {
  if (!pairQueueContainer) return;
  pairQueueContainer.innerHTML = '';
  const gridContainer = document.createElement('div');
  gridContainer.className = 'gopro-grid-container';
  pairQueueContainer.appendChild(gridContainer);

  let devices = Object.values(state.allDevices).filter(d =>
    d.isReachable && (!d.isPaired || !d.isManaged)
  );
  devices.sort((a, b) => (b.rssi || -100) - (a.rssi || -100));

  if (devices.length === 0) {
    showEmptyState(gridContainer, 'discovered');
  } else {
    devices.forEach(device => {
      const el = deviceCard.createGoProElement(device);
      gridContainer.appendChild(el);
      if (!state.uiState.deviceElements[device.macAddress]) state.uiState.deviceElements[device.macAddress] = {};
      state.uiState.deviceElements[device.macAddress].discovered = el;
    });
  }
}

function updateCamerasPoolUI() {
  if (!camerasPoolContainer) return;
  camerasPoolContainer.innerHTML = '';
  const gridContainer = document.createElement('div');
  gridContainer.className = 'gopro-grid-container';
  camerasPoolContainer.appendChild(gridContainer);

  let devices = Object.values(state.allDevices).filter(d =>
    d.isReachable && d.isPaired && d.isManaged
  );
  devices.sort((a, b) => (a.name || '').localeCompare(b.name || ''));

  if (devices.length === 0) {
    showEmptyState(gridContainer, 'managed');
  } else {
    devices.forEach(device => {
      const el = deviceCard.createGoProElement(device);
      gridContainer.appendChild(el);
      if (!state.uiState.deviceElements[device.macAddress]) state.uiState.deviceElements[device.macAddress] = {};
      state.uiState.deviceElements[device.macAddress].managed = el;
    });
  }
}

function updateSyncQueueUI() {
  if (!syncQueueContainer) return;
  syncQueueContainer.innerHTML = '';
  const gridContainer = document.createElement('div');
  gridContainer.className = 'gopro-grid-container';
  syncQueueContainer.appendChild(gridContainer);

  let devices = Object.values(state.allDevices).filter(d =>
    d.isReachable && d.isPaired && d.isManaged && !d.isSynced
  );

  if (devices.length === 0) {
    showEmptyState(gridContainer, 'sync');
  } else {
    devices.forEach(device => {
      const el = deviceCard.createGoProElement(device, true);
      gridContainer.appendChild(el);
      if (!state.uiState.deviceElements[device.macAddress]) state.uiState.deviceElements[device.macAddress] = {};
      state.uiState.deviceElements[device.macAddress].sync = el;
    });
  }
}

function updateDeviceLists() {
  state.uiState.deviceElements = {};
  state.uiState.poolMembership = {};
  updatePairQueueUI();
  updateCamerasPoolUI();
  updateSyncQueueUI();
  updateCounters();
}

module.exports = {
  setContainers,
  addOrUpdateDevice,
  updateCounters,
  cleanupDisconnectedDevices,
  updateDeviceLists,
  updateSyncQueueUI,
  setDebugLog: (fn) => { _debugLog = fn; }
};
