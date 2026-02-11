// Renderer process — thin entry point that wires modules together
const { ipcRenderer } = require('electron');
const { getClient } = require('./grpc-utils');
const { logToFile, LOG_LEVELS, setLogLevel, currentLogLevel } = require('./fileLogger');
const { shouldFilterLogMessage: shouldFilterLog } = require('./log-filters');

const state = require('./state');
const streams = require('./streams');
const devicePools = require('./device-pools');
const deviceCard = require('./device-card');
const actions = require('./actions');
const settings = require('./settings');

// DOM elements
const statusLight = document.getElementById('status-light');
const statusText = document.getElementById('status-text');
const logContent = document.getElementById('log-content');
const clearLogsButton = document.getElementById('clear-logs');
const logLevelSelect = document.getElementById('log-level-select');
const showGoLogsToggle = document.getElementById('show-go-logs-toggle');
const themeToggle = document.getElementById('theme-toggle');
const pairQueueContainer = document.getElementById('pair-queue-container');
const camerasPoolContainer = document.getElementById('cameras-pool-container');
const syncQueueContainer = document.getElementById('sync-queue-container');
const managedCountDisplay = document.getElementById('managed-count');
const unpairedCountDisplay = document.getElementById('unpaired-count');
const syncCountDisplay = document.getElementById('sync-count');
const pairAllToggle = document.getElementById('pair-all-toggle');
const syncQueueToggle = document.getElementById('sync-queue-toggle');
const viewToggleButtons = document.querySelectorAll('.view-button');

let currentViewMode = 'grid';
let isDarkMode = false;

// gRPC readiness gate
let grpcReadyResolve;
let grpcReady = new Promise(resolve => { grpcReadyResolve = resolve; });

// --- Wire modules ---
devicePools.setContainers({
  pairQueueContainer, camerasPoolContainer, syncQueueContainer,
  managedCountDisplay, unpairedCountDisplay, syncCountDisplay
});
settings.setDomRefs({ pairAllToggle, syncQueueToggle, logLevelSelect });

// Inject debugLog and addLogEntry into all modules
deviceCard.setDebugLog(debugLog);
devicePools.setDebugLog(debugLog);
streams.setDebugLog(debugLog);
actions.setDebugLog(debugLog);
actions.setAddLogEntry(addLogEntry);
actions.setShowToast(showToast);
settings.setDebugLog(debugLog);
settings.setAddLogEntry(addLogEntry);
settings.setShowToast(showToast);

logToFile('Renderer process starting up');

// --- Init ---
function init() {
  debugLog('Initializing frontend application', LOG_LEVELS.INFO);
  updateStatus(false);
  loadThemePreference();
  initializeToggles();

  clearLogsButton.addEventListener('click', clearLogs);
  logLevelSelect.addEventListener('change', changeLogLevel);
  themeToggle.addEventListener('click', toggleTheme);
  setupViewToggle();
  settings.setupSettingsHandlers();

  if (showGoLogsToggle) {
    showGoLogsToggle.addEventListener('change', toggleGoLogs);
    showGoLogsToggle.checked = true;
  }

  setupIpcListeners();
  streams.setOnStreamStateChange(handleStreamStateChange);

  setInterval(devicePools.cleanupDisconnectedDevices, 30000);

  grpcReady.then(() => {
    debugLog('Starting device streaming', LOG_LEVELS.INFO);
    streams.startDeviceStreaming(updateStatus);
    settings.loadConfig().then(() => loadCurrentLogLevel());
    debugLog('Initial data loading complete', LOG_LEVELS.INFO);
  });

  window.addEventListener('beforeunload', () => {
    debugLog('App closing, cancelling all streams', LOG_LEVELS.INFO);
    streams.cancelAllStreams();
  });

  debugLog('Frontend initialization complete', LOG_LEVELS.INFO);
}

function initializeToggles() {
  state.autoPair = false;
  state.autoSync = false;
  if (pairAllToggle) {
    pairAllToggle.checked = false;
    pairAllToggle.addEventListener('change', actions.togglePairAll);
  }
  if (syncQueueToggle) {
    syncQueueToggle.checked = false;
    syncQueueToggle.addEventListener('change', actions.toggleAutoSync);
  }
}

function setupViewToggle() {
  if (!viewToggleButtons) return;
  viewToggleButtons.forEach(button => {
    button.addEventListener('click', () => {
      const viewMode = button.getAttribute('data-view');
      if (viewMode === currentViewMode) return;
      currentViewMode = viewMode;
      viewToggleButtons.forEach(btn => {
        btn.classList.toggle('active', btn.getAttribute('data-view') === viewMode);
      });
      [camerasPoolContainer, pairQueueContainer, syncQueueContainer].forEach(c => {
        if (c) { c.classList.remove('devices-grid', 'devices-list'); c.classList.add(`devices-${viewMode}`); }
      });
      debugLog(`View mode changed to: ${viewMode}`, LOG_LEVELS.INFO);
      devicePools.updateDeviceLists();
    });
  });
}

// --- IPC listeners ---
function setupIpcListeners() {
  ipcRenderer.on('go-binary-status', (event, data) => {
    updateStatus(data.running, data.exitCode);
    if (data.running) {
      addLogEntry('Service started', 'info');
      grpcReadyResolve();
    } else {
      if (state.activeStream) {
        state.activeStream.intentionalCancel = true;
        if (state.activeStream.cancel) state.activeStream.cancel();
        state.activeStream = null;
      }
      addLogEntry(data.exitCode === 0 ? 'Service stopped gracefully' : `Service stopped with exit code ${data.exitCode}`, data.exitCode === 0 ? 'info' : 'error');
    }
  });

  ipcRenderer.on('go-binary-error', (event, errorMessage) => {
    debugLog(`Go binary error: ${errorMessage}`, LOG_LEVELS.ERROR);
    addLogEntry(`Service error: ${errorMessage}`, 'error');
  });

  ipcRenderer.on('go-binary-log', (event, log) => {
    const { level, style, message } = parseGoLogLevel(log);
    logToFile(message, level);
    addLogEntry(message, style, 'go');
  });

  ipcRenderer.on('pair-device-response', (event, result) => {
    if (result.success) {
      const deviceMac = Object.keys(state.pairingInProgress).find(mac => state.pairingInProgress[mac]);
      if (deviceMac && state.allDevices[deviceMac]) {
        actions.updateDeviceStatus(deviceMac, 'paired');
        addLogEntry(`Device ${state.allDevices[deviceMac].name} paired successfully`, 'info');
      }
    } else {
      addLogEntry(`Pairing failed: ${result.message}`, 'error');
    }
    Object.keys(state.pairingInProgress).forEach(mac => delete state.pairingInProgress[mac]);
  });
}

// --- Status UI ---
function updateStatus(running, exitCode) {
  const statusChanged = state.serviceRunning !== running;
  state.serviceRunning = running;

  if (running) {
    statusLight.className = 'status-light running';
    statusText.textContent = 'Running';
  } else {
    statusLight.className = 'status-light';
    statusText.textContent = exitCode !== undefined && exitCode !== 0
      ? `Stopped (Exit code: ${exitCode})`
      : 'Stopped';
  }
}

function handleStreamStateChange() {
  if (!state.serviceRunning) return;
  const statuses = Object.values(state.streamStatus);
  const anyReconnecting = statuses.some(s => s === 'reconnecting');
  if (anyReconnecting) {
    statusLight.classList.add('reconnecting');
    statusText.textContent = 'Running (Reconnecting...)';
  } else {
    statusLight.classList.remove('reconnecting');
    statusText.textContent = 'Running';
  }
}

// --- Logging UI ---
function debugLog(message, level = LOG_LEVELS.INFO, context = {}) {
  if (level > currentLogLevel()) return;
  const levelName = getLogLevelName(level);

  let formatted = message;
  if (Object.keys(context).length > 0) {
    formatted = `${message} ${Object.entries(context).map(([k, v]) => `${k}=${JSON.stringify(v)}`).join(' ')}`;
  }

  const finalMessage = `[FRONTEND-${levelName}] ${formatted}`;
  const consoleMethod = level === LOG_LEVELS.ERROR ? console.error : level === LOG_LEVELS.WARN ? console.warn : console.log;
  consoleMethod(finalMessage);
  logToFile(formatted, level);

  try { ipcRenderer.send('debug-log', finalMessage); } catch (_) {}

  const logStyle = level === LOG_LEVELS.ERROR ? 'error' : level === LOG_LEVELS.WARN ? 'warn' : 'info';
  addLogEntry(`${levelName}: ${formatted}`, logStyle);
}

function getLogLevelName(level) {
  switch (level) {
    case LOG_LEVELS.ERROR: return 'Error';
    case LOG_LEVELS.WARN: return 'Warning';
    case LOG_LEVELS.INFO: return 'Info';
    case LOG_LEVELS.DEBUG: return 'Debug';
    case LOG_LEVELS.TRACE: return 'Trace';
    default: return 'Unknown';
  }
}

function addLogEntry(message, type, source = 'app') {
  if (shouldFilterLogMessage(message, type, source)) return;

  // Auto-scroll if user is near the bottom
  const atBottom = logContent.scrollHeight - logContent.scrollTop - logContent.clientHeight < 40;

  const entry = document.createElement('div');
  entry.className = `log-entry ${type}`;
  entry.textContent = `[${new Date().toLocaleTimeString()}] ${message}`;
  entry.dataset.logType = type;
  entry.dataset.source = source;

  if (source === 'go' && showGoLogsToggle && !showGoLogsToggle.checked) {
    entry.style.display = 'none';
  }

  logContent.appendChild(entry);
  while (logContent.children.length > 500) {
    logContent.removeChild(logContent.firstChild);
  }

  if (atBottom) logContent.scrollTop = logContent.scrollHeight;
}

function showToast(message, type = 'info') {
  const container = document.getElementById('toast-container');
  if (!container) return;

  const toast = document.createElement('div');
  toast.className = `toast ${type}`;

  const text = document.createElement('span');
  text.textContent = message;

  const close = document.createElement('button');
  close.className = 'toast-close';
  close.innerHTML = '&times;';
  close.addEventListener('click', () => removeToast(toast));

  toast.appendChild(text);
  toast.appendChild(close);
  container.appendChild(toast);

  setTimeout(() => removeToast(toast), 4000);
}

function removeToast(toast) {
  if (!toast.parentNode) return;
  toast.classList.add('removing');
  toast.addEventListener('animationend', () => toast.remove());
}

function shouldFilterLogMessage(message, type, source) {
  if (type !== 'warn' || source !== 'go') return false;
  return shouldFilterLog(message);
}

function clearLogs() {
  logContent.innerHTML = '';
  addLogEntry('Logs cleared', 'info');
}

function changeLogLevel() {
  const selectedLevel = logLevelSelect.value;
  setLogLevel(selectedLevel);
  localStorage.setItem('logLevel', selectedLevel);
  settings.updateSetting('log_level', selectedLevel.toLowerCase());
}

function loadCurrentLogLevel() {
  const level = (state.appConfig && state.appConfig.logLevel) || localStorage.getItem('logLevel') || 'info';
  logLevelSelect.value = level;
  setLogLevel(level);
}

function toggleGoLogs(event) {
  const show = event.target.checked;
  document.querySelectorAll('.log-entry[data-source="go"]').forEach(entry => {
    entry.style.display = show ? 'block' : 'none';
  });
  debugLog(`Go logs ${show ? 'shown' : 'hidden'}`, LOG_LEVELS.INFO);
}

function parseGoLogLevel(logMessage) {
  const levelMap = {
    '[TRACE]': { level: LOG_LEVELS.TRACE, style: 'info' },
    '[DEBUG]': { level: LOG_LEVELS.DEBUG, style: 'info' },
    '[INFO]':  { level: LOG_LEVELS.INFO,  style: 'info' },
    '[WARN]':  { level: LOG_LEVELS.WARN,  style: 'warn' },
    '[ERROR]': { level: LOG_LEVELS.ERROR, style: 'error' },
    '[FATAL]': { level: LOG_LEVELS.ERROR, style: 'error' },
    '[PANIC]': { level: LOG_LEVELS.ERROR, style: 'error' }
  };

  let level = LOG_LEVELS.INFO;
  let style = 'info';

  for (const [tag, info] of Object.entries(levelMap)) {
    if (logMessage.includes(tag)) {
      level = info.level;
      style = info.style;
      break;
    }
  }

  // Strip the backend timestamp prefix (e.g. "2024/01/15 14:30:05 ")
  const msg = logMessage.replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+/, '');

  return { level, style, message: `[Go] ${msg}` };
}

// --- Theme ---
function toggleTheme() {
  isDarkMode = !isDarkMode;
  updateTheme();
  try { localStorage.setItem('darkMode', isDarkMode ? 'true' : 'false'); } catch (_) {}
}

function loadThemePreference() {
  try {
    const saved = localStorage.getItem('darkMode');
    if (saved !== null) isDarkMode = saved === 'true';
    else isDarkMode = window.matchMedia?.('(prefers-color-scheme: dark)').matches || false;
    updateTheme();
  } catch (_) {}
}

function updateTheme() {
  const html = document.documentElement;
  const icon = themeToggle.querySelector('i');
  if (isDarkMode) {
    html.setAttribute('data-theme', 'dark');
    icon.className = 'fas fa-sun';
  } else {
    html.removeAttribute('data-theme');
    icon.className = 'fas fa-moon';
  }
}

// --- Bootstrap ---
document.addEventListener('DOMContentLoaded', () => {
  debugLog('DOM fully loaded, calling init()');
  init();
});

if (typeof module !== 'undefined' && module.exports) {
  module.exports = { startDeviceStreaming: streams.startDeviceStreaming };
}
