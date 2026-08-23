// UI state store
let theme = $state('light');
let logsExpanded = $state(false);
let settingsOpen = $state(false);
// {type: 'camera'|'group', id, name, referenceCameraId} or null
let cameraSettingsTarget = $state(null);
let searchQuery = $state('');
let sortBy = $state('signal');
let toasts = $state([]);
let logs = $state([]);
let showBackendLogs = $state(true);
let logLevel = $state('info');

export function getTheme() { return theme; }
export function getLogsExpanded() { return logsExpanded; }
export function getSettingsOpen() { return settingsOpen; }
export function getCameraSettingsTarget() { return cameraSettingsTarget; }
export function openCameraSettings(target) { cameraSettingsTarget = target; }
export function closeCameraSettings() { cameraSettingsTarget = null; }
export function getSearchQuery() { return searchQuery; }
export function getSortBy() { return sortBy; }
export function getToasts() { return toasts; }
export function getLogs() { return logs; }
export function getShowBackendLogs() { return showBackendLogs; }
export function getLogLevel() { return logLevel; }

export function setTheme(value) {
  theme = value;
  document.documentElement.setAttribute('data-theme', value === 'dark' ? 'dark' : '');
  if (value !== 'dark') document.documentElement.removeAttribute('data-theme');
  try { localStorage.setItem('darkMode', value === 'dark' ? 'true' : 'false'); } catch (_) {}
}

export function toggleTheme() {
  setTheme(theme === 'dark' ? 'light' : 'dark');
}

export function loadThemePreference() {
  try {
    const saved = localStorage.getItem('darkMode');
    if (saved !== null) setTheme(saved === 'true' ? 'dark' : 'light');
    else if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) setTheme('dark');
  } catch (_) {}
}

export function setLogsExpanded(value) { logsExpanded = value; }
export function setSettingsOpen(value) { settingsOpen = value; }
export function setSearchQuery(value) { searchQuery = value; }
export function setSortBy(value) { sortBy = value; }
export function setShowBackendLogs(value) { showBackendLogs = value; }
export function setLogLevel(value) { logLevel = value; }

let nextToastId = 0;
export function addToast(message, type = 'info') {
  const id = nextToastId++;
  toasts.push({ id, message, type });
  setTimeout(() => removeToast(id), 4000);
}

export function removeToast(id) {
  const idx = toasts.findIndex(t => t.id === id);
  if (idx >= 0) toasts.splice(idx, 1);
}

export function addLog(message, type = 'info', source = 'app') {
  logs.push({ message, type, source, time: new Date() });
  // Keep max 500 entries
  while (logs.length > 500) logs.shift();
}

export function clearLogs() {
  logs.length = 0;
  addLog('Logs cleared', 'info');
}
