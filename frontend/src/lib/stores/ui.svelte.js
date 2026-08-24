// UI state store
let theme = $state('light');
let activeTab = $state('cameras');
let settingsOpen = $state(false);
// {type: 'camera'|'group', id, name, referenceCameraId} or null
let cameraSettingsTarget = $state(null);
// {source: 'local'|cameraId, filter?: 'all'|'new'|{sessionId}} or null;
// consumed once by the library view
let libraryTarget = $state(null);
// {path, title, sourcePath, sizeBytes} or null; the in-app video player
// overlay. sourcePath is the full-res library file the proxy stands for,
// empty when there is none to trim.
let playerTarget = $state(null);
// cameraId or null; consumed once by the activity view
let activityTarget = $state(null);
let searchQuery = $state('');
let sortBy = $state('signal');
let toasts = $state([]);
let logs = $state([]);
let logsExpanded = $state(false);
let showBackendLogs = $state(true);
let logLevel = $state('info');
// Group sections the user folded on the Cameras tab, by group id
let collapsedGroups = $state(loadCollapsed());
// Files that arrived after this moment count as new in the library.
// Starts at first launch so an existing library is not all "new".
let libraryLastVisit = $state(loadLastVisit());

export function getTheme() { return theme; }
export function getActiveTab() { return activeTab; }
export function getSettingsOpen() { return settingsOpen; }
export function getCameraSettingsTarget() { return cameraSettingsTarget; }
export function openCameraSettings(target) { cameraSettingsTarget = target; }
export function closeCameraSettings() { cameraSettingsTarget = null; }
export function getLibraryTarget() { return libraryTarget; }
export function openLibrary(source = 'local', filter = 'all') { libraryTarget = { source, filter }; }
export function clearLibraryTarget() { libraryTarget = null; }
export function getActivityTarget() { return activityTarget; }
export function openActivity(cameraId = null) { activityTarget = { cameraId }; setActiveTab('activity'); }
export function clearActivityTarget() { activityTarget = null; }
export function getPlayerTarget() { return playerTarget; }
export function openPlayer(path, title, sourcePath = '', sizeBytes = 0) {
  playerTarget = { path, title, sourcePath, sizeBytes };
}
export function closePlayer() { playerTarget = null; }
export function getSearchQuery() { return searchQuery; }
export function getSortBy() { return sortBy; }
export function getToasts() { return toasts; }
export function getLogs() { return logs; }
export function getLogsExpanded() { return logsExpanded; }
export function getShowBackendLogs() { return showBackendLogs; }
export function getLogLevel() { return logLevel; }
export function getLibraryLastVisit() { return libraryLastVisit; }

export function isGroupCollapsed(id) { return collapsedGroups.has(id); }
export function toggleGroupCollapsed(id) {
  const next = new Set(collapsedGroups);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  collapsedGroups = next;
  try { localStorage.setItem('collapsedGroups', JSON.stringify([...next])); } catch (_) {}
}

function loadCollapsed() {
  try {
    const saved = JSON.parse(localStorage.getItem('collapsedGroups') || '[]');
    if (Array.isArray(saved)) return new Set(saved);
  } catch (_) {}
  return new Set();
}

export function setActiveTab(tab) {
  // Leaving the library is the moment "new" resets; staying on it keeps
  // the badges so the user can still tell what just arrived.
  if (activeTab === 'library' && tab !== 'library') markLibraryVisited();
  activeTab = tab;
}

export function markLibraryVisited() {
  libraryLastVisit = Date.now();
  try { localStorage.setItem('libraryLastVisit', String(libraryLastVisit)); } catch (_) {}
}

function loadLastVisit() {
  try {
    const saved = Number(localStorage.getItem('libraryLastVisit'));
    if (saved > 0) return saved;
  } catch (_) {}
  const now = Date.now();
  try { localStorage.setItem('libraryLastVisit', String(now)); } catch (_) {}
  return now;
}

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
// Toasts confirm; they are never the only record of an outcome
export function addToast(message, type = 'info') {
  const id = nextToastId++;
  toasts.push({ id, message, type });
  setTimeout(() => removeToast(id), type === 'error' ? 8000 : 4000);
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
