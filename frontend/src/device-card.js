// Device card rendering — creating and updating DOM elements for GoPro devices
const { logToFile, LOG_LEVELS } = require('./fileLogger');
const state = require('./state');

// Lazy-loaded references to avoid circular deps
let _actions = null;
function actions() {
  if (!_actions) _actions = require('./actions');
  return _actions;
}

function debugLog(msg, level = LOG_LEVELS.INFO) {
  // Delegate to renderer's debugLog via the injected reference
  if (debugLog._delegate) return debugLog._delegate(msg, level);
  logToFile(msg, level);
}
debugLog._delegate = null;

function createGoProElement(device, inSyncQueue = false) {
  const template = document.getElementById('gopro-item-template');
  const fragment = template.content.cloneNode(true);
  const element = fragment.querySelector('.device-card');

  const statusCode = getStatusShortcode(device);
  element.classList.add(`status-${statusCode.toLowerCase()}`);
  element.setAttribute('data-mac', device.macAddress);

  const nameElement = element.querySelector('.device-name');
  if (nameElement) {
    const displayName = getDeviceDisplayName(device);
    nameElement.textContent = displayName;
    nameElement.title = displayName;
    const badge = document.createElement('span');
    badge.className = `status-badge status-${statusCode.toLowerCase()}`;
    badge.textContent = formatStatus(statusCode);
    nameElement.appendChild(document.createTextNode(' '));
    nameElement.appendChild(badge);
  }

  const macElement = element.querySelector('.device-mac');
  if (macElement) {
    const formattedMac = formatMacAddress(device.macAddress);
    macElement.textContent = formattedMac;
    macElement.title = formattedMac;
  }

  const modelElement = element.querySelector('.device-model');
  if (modelElement) modelElement.textContent = device.model || 'Unknown Model';

  const firmwareElement = element.querySelector('.device-firmware');
  if (firmwareElement) firmwareElement.textContent = device.firmwareVersion || 'Unknown FW';

  const signalElement = element.querySelector('.device-signal');
  if (signalElement) {
    const rssi = device.rssi || -100;
    signalElement.appendChild(createSignalStrengthIndicator(rssi));
    const rssiValue = document.createElement('span');
    rssiValue.textContent = `${rssi} dBm`;
    rssiValue.classList.add('rssi-value');
    signalElement.appendChild(rssiValue);
  }

  const batteryElement = element.querySelector('.device-battery');
  if (batteryElement && device.batteryLevel !== undefined) {
    batteryElement.appendChild(createBatteryIndicator(device.batteryLevel, device.isCharging));
  }

  const statusElement = element.querySelector('.device-status');
  if (statusElement) {
    statusElement.textContent = device.lastSeen ? `Last seen: ${formatTimeSince(device.lastSeen)}` : '';
    statusElement.classList.add(`status-${statusCode.toLowerCase()}`);
  }

  // Delegate button/toggle setup + sync progress to updateDeviceControls/updateSyncProgress
  updateDeviceControls(element, device, inSyncQueue);
  updateSyncProgress(element, device, inSyncQueue);

  return element;
}

function updateDeviceElement(element, device, inSyncQueue = false) {
  const nameElement = element.querySelector('.device-name');
  if (nameElement) {
    nameElement.innerHTML = '';
    const displayName = getDeviceDisplayName(device);
    nameElement.textContent = displayName;
    nameElement.title = displayName;
    const badge = document.createElement('span');
    badge.className = `status-badge status-${getStatusShortcode(device).toLowerCase()}`;
    badge.textContent = formatStatus(getStatusShortcode(device));
    nameElement.appendChild(document.createTextNode(' '));
    nameElement.appendChild(badge);
  }

  const modelElement = element.querySelector('.device-model');
  if (modelElement) modelElement.textContent = device.model || 'Unknown Model';

  const firmwareElement = element.querySelector('.device-firmware');
  if (firmwareElement) firmwareElement.textContent = device.firmwareVersion || 'Unknown FW';

  updateSignalStrengthInElement(element, device.rssi || -100);

  const batteryElement = element.querySelector('.device-battery');
  if (batteryElement && device.batteryLevel !== undefined) {
    batteryElement.innerHTML = '';
    batteryElement.appendChild(createBatteryIndicator(device.batteryLevel, device.isCharging));
  }

  const statusElement = element.querySelector('.device-status');
  if (statusElement) {
    statusElement.textContent = device.lastSeen ? `Last seen: ${formatTimeSince(device.lastSeen)}` : '';
    statusElement.className = `device-status status-${getStatusShortcode(device).toLowerCase()}`;
  }

  // Update status class on card
  element.className = element.className.replace(/status-\w+/g, '');
  element.classList.add(`status-${getStatusShortcode(device).toLowerCase()}`);

  updateSyncProgress(element, device, inSyncQueue);
  updateDeviceControls(element, device, inSyncQueue);
}

function updateDeviceControls(element, device, inSyncQueue = false) {
  const macAddress = device.macAddress;
  const pairButton = element.querySelector('.pair-button');
  const syncButton = element.querySelector('.sync-button');
  const manageToggle = element.querySelector('.manage-toggle');

  const isInSyncQueue = state.syncQueue.some(entry => entry.cameraId === device.id);

  if (pairButton) {
    const newPairButton = pairButton.cloneNode(true);
    pairButton.parentNode.replaceChild(newPairButton, pairButton);

    if (isInSyncQueue) {
      newPairButton.textContent = "Cancel";
      newPairButton.disabled = false;
      newPairButton.classList.remove('in-progress');
      newPairButton.classList.add('cancel-button');
      newPairButton.addEventListener('click', () => {
        newPairButton.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Cancelling...';
        newPairButton.disabled = true;
        newPairButton.classList.add('in-progress');
        actions().cancelSync(macAddress);
      });
    } else if (state.pairingInProgress[macAddress]) {
      newPairButton.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Pairing...';
      newPairButton.disabled = true;
      newPairButton.classList.add('in-progress');
      newPairButton.classList.remove('cancel-button');
    } else if (!device.isPaired) {
      newPairButton.textContent = "Pair";
      newPairButton.disabled = false;
      newPairButton.classList.remove('in-progress', 'cancel-button');
      newPairButton.addEventListener('click', () => {
        newPairButton.textContent = "Pairing...";
        newPairButton.disabled = true;
        newPairButton.classList.add('in-progress');
        actions().pairDevice(macAddress);
      });
    } else {
      newPairButton.textContent = "Paired";
      newPairButton.disabled = true;
      newPairButton.classList.remove('in-progress', 'cancel-button');
    }
  }

  if (syncButton) {
    const newSyncButton = syncButton.cloneNode(true);
    syncButton.parentNode.replaceChild(newSyncButton, syncButton);

    if (inSyncQueue) {
      newSyncButton.textContent = "Cancel";
      newSyncButton.disabled = false;
      newSyncButton.classList.remove('queued', 'in-progress');
      newSyncButton.classList.add('cancel-button');
      newSyncButton.addEventListener('click', () => {
        newSyncButton.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Cancelling...';
        newSyncButton.disabled = true;
        newSyncButton.classList.add('in-progress');
        actions().cancelSync(macAddress);
      });
    } else if (device.isPaired && device.isManaged) {
      newSyncButton.textContent = "Sync";
      newSyncButton.disabled = false;
      newSyncButton.classList.remove('in-progress', 'queued');
      newSyncButton.addEventListener('click', () => {
        newSyncButton.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Syncing...';
        newSyncButton.disabled = true;
        newSyncButton.classList.add('in-progress');
        actions().addToSyncQueue(macAddress);
      });
    } else {
      newSyncButton.disabled = true;
      newSyncButton.classList.remove('in-progress', 'queued');
    }
  }

  if (manageToggle) {
    const newManageToggle = manageToggle.cloneNode(true);
    manageToggle.parentNode.replaceChild(newManageToggle, manageToggle);
    newManageToggle.checked = !!device.isManaged;
    newManageToggle.addEventListener('change', (event) => {
      actions().toggleDeviceManaged(macAddress, event.target.checked);
    });
  }
}

function updateSyncProgress(element, device, inSyncQueue = false) {
  const progressContainer = element.querySelector('.sync-progress');
  const progressBar = element.querySelector('.sync-progress-fill');
  const progressOperation = element.querySelector('.sync-progress-operation');
  if (!progressContainer || !progressBar || !progressOperation) return;

  if (device.isSyncing) {
    const syncEntry = state.syncQueue.find(entry => entry.cameraId === device.id);
    if (syncEntry) {
      progressBar.style.width = `${syncEntry.progressPercent || 0}%`;
      progressOperation.textContent = syncEntry.currentOperation || 'Preparing...';
    } else {
      progressBar.style.width = '100%';
      progressOperation.textContent = 'Processing...';
    }
    progressContainer.classList.add('active');
  } else {
    progressContainer.classList.remove('active');
  }
}

function updateAllSyncProgressBars() {
  document.querySelectorAll('.device-card[data-mac]').forEach(card => {
    const mac = card.getAttribute('data-mac');
    const device = state.allDevices[mac];
    if (device && device.isSyncing) {
      updateSyncProgress(card, device, shouldShowInSyncQueue(device));
    }
  });
}

function applySignalStrength(bars, rssi) {
  bars.forEach(bar => bar.classList.remove('filled'));
  const normalizedRssi = Math.min(Math.max(rssi, -100), -30);
  const strength = Math.floor(((normalizedRssi + 100) / 70) * 4);
  for (let i = 0; i < strength; i++) {
    if (bars[i]) bars[i].classList.add('filled');
  }
}

function createSignalStrengthIndicator(rssi) {
  const container = document.createElement('div');
  container.classList.add('signal-bars');
  for (let i = 0; i < 4; i++) {
    const bar = document.createElement('div');
    bar.classList.add('signal-bar');
    container.appendChild(bar);
  }
  applySignalStrength(container.querySelectorAll('.signal-bar'), rssi);
  return container;
}

function createBatteryIndicator(level, isCharging = false) {
  const container = document.createElement('div');
  container.classList.add('battery-indicator');
  const icon = document.createElement('div');
  icon.classList.add('battery-icon');
  if (isCharging) icon.classList.add('charging');
  const fill = document.createElement('div');
  fill.classList.add('battery-fill');
  const fillLevel = Math.max(0, Math.min(100, level));
  fill.style.width = `${fillLevel}%`;
  if (fillLevel < 20) fill.classList.add('battery-low');
  else if (fillLevel < 50) fill.classList.add('battery-medium');
  else fill.classList.add('battery-good');
  icon.appendChild(fill);
  container.appendChild(icon);
  const text = document.createElement('span');
  text.textContent = `${fillLevel}%`;
  container.appendChild(text);
  return container;
}

function updateSignalStrengthInElement(element, rssi) {
  const signalElement = element.querySelector('.device-signal');
  if (!signalElement) return;
  const barsContainer = signalElement.querySelector('.signal-bars');
  if (barsContainer) applySignalStrength(barsContainer.querySelectorAll('.signal-bar'), rssi);
  const rssiValueEl = signalElement.querySelector('.rssi-value');
  if (rssiValueEl) rssiValueEl.textContent = `${rssi} dBm`;
}

function updateSignalStrengthForDevice(macAddress, rssi) {
  document.querySelectorAll(`.device-card[data-mac="${macAddress}"]`).forEach(card => {
    updateSignalStrengthInElement(card, rssi);
  });
}

function formatMacAddress(mac) {
  if (!mac) return 'Unknown';
  if (mac.includes(':')) return mac;
  return mac.match(/.{1,2}/g).join(':');
}

function getDeviceDisplayName(device) {
  if (device.wifiSsid && device.wifiSsid.trim() !== '') {
    return device.wifiSsid.substring(0, 12);
  }
  return device.name || 'Unknown GoPro';
}

function getStatusShortcode(device) {
  if (!device.isReachable) return 'unreachable';
  if (device.isPairing) return 'pairing';
  if (device.isSyncing) return 'syncing';
  if (!device.isPaired && device.isManaged) return 'unavailable';
  if (!device.isPaired) return 'discovered';
  if (device.isPaired && !device.isManaged) return 'available';
  if (device.isPaired && device.isManaged) return 'managed';
  return 'unknown';
}

function formatStatus(status) {
  if (typeof status !== 'string') return 'Unknown';
  const s = status.toLowerCase();
  const map = {
    'discovered': 'Discovered',
    'managed': 'Managed',
    'pairing': 'Pairing', 'paired': 'Paired',
    'unavailable': 'Unavailable', 'available': 'Available',
    'syncing': 'Syncing', 'unreachable': 'Unreachable',
    'unknown': 'Unknown'
  };
  if (map[s]) return map[s];
  if (status.includes('_')) {
    return status.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()).join(' ');
  }
  return status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
}

function formatTimeSince(date) {
  if (!date) return 'unknown';
  if (typeof date === 'string') date = new Date(date);
  const seconds = Math.floor((new Date() - date) / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} ${minutes === 1 ? 'minute' : 'minutes'} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} ${hours === 1 ? 'hour' : 'hours'} ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days} ${days === 1 ? 'day' : 'days'} ago`;
  return date.toLocaleDateString();
}

function determineVisualStatus(status) {
  if (!status) return 'Unknown';
  if (!status.getIsReachable()) return 'Unreachable';
  if (status.getIsPairing()) return 'Pairing';
  if (status.getIsSyncing()) return 'Syncing';
  const isPaired = status.getIsPaired();
  const isManaged = status.getIsManaged();
  if (!isPaired) return 'Discovered';
  if (!isManaged) return 'Available';
  if (isManaged) return 'Managed';
  return 'Unknown';
}

function shouldShowInDiscoveredPool(device) {
  return !device.isPaired || !device.isManaged;
}

function shouldShowInManagedPool(device) {
  return device.isReachable && device.isPaired && device.isManaged;
}

function shouldShowInSyncQueue(device) {
  return device.isReachable && device.isPaired && device.isManaged && !device.isSynced;
}

module.exports = {
  createGoProElement,
  updateDeviceElement,
  updateAllSyncProgressBars,
  updateSignalStrengthForDevice,
  getDeviceDisplayName,
  formatTimeSince,
  shouldShowInDiscoveredPool,
  shouldShowInManagedPool,
  shouldShowInSyncQueue,
  determineVisualStatus,
  setDebugLog: (fn) => { debugLog._delegate = fn; }
};
