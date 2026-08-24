// Sync queue and history store
import { addToast } from './ui.svelte.js';
import { getAllDevices, displayName } from './devices.svelte.js';
import { formatBytes, plural } from '../format.js';

let syncQueue = $state([]);
// Finished sessions, newest first; loaded on start and after every
// queue shrink (a sync just ended)
let history = $state([]);
let historyLoaded = $state(false);
let announced = new Set();

export function getSyncQueue() { return syncQueue; }
export function getSyncHistory() { return history; }
export function isHistoryLoaded() { return historyLoaded; }

// Returns true when an entry left the queue, so the caller can reload
// history; the outcome toast comes from the session, not the label.
export function setSyncQueue(entries) {
  const shrunk = syncQueue.some(old => !entries.some(e => e.cameraId === old.cameraId));
  syncQueue.length = 0;
  entries.forEach(e => syncQueue.push(e));
  return shrunk;
}

export function addOrUpdateSyncEntry(entry) {
  const idx = syncQueue.findIndex(e => e.cameraId === entry.cameraId);
  if (idx >= 0) syncQueue[idx] = entry;
  else syncQueue.push(entry);
}

export function removeSyncEntry(cameraId) {
  const idx = syncQueue.findIndex(e => e.cameraId === cameraId);
  if (idx >= 0) syncQueue.splice(idx, 1);
}

export function getSyncEntryForCamera(cameraId) {
  return syncQueue.find(e => e.cameraId === cameraId);
}

// The radio is a single slot: at most one entry is actually running.
export function getActiveSyncEntry() {
  const devices = Object.values(getAllDevices());
  return syncQueue.find(e => devices.find(d => d.id === e.cameraId)?.isSyncing) || null;
}

// 1-based position among entries still waiting; 0 when running or absent
export function getQueuePosition(cameraId) {
  const devices = Object.values(getAllDevices());
  const waiting = syncQueue.filter(e => !devices.find(d => d.id === e.cameraId)?.isSyncing);
  return waiting.findIndex(e => e.cameraId === cameraId) + 1;
}

export function setSyncHistory(sessions) {
  const first = !historyLoaded;
  history = sessions;
  historyLoaded = true;
  if (first) {
    // Nothing that happened before this launch deserves a toast
    sessions.forEach(s => announced.add(s.id));
    return;
  }
  for (const s of sessions) {
    if (announced.has(s.id)) continue;
    announced.add(s.id);
    announce(s);
  }
}

function cameraLabel(cameraId) {
  const device = Object.values(getAllDevices()).find(d => d.id === cameraId);
  return device ? displayName(device) : 'Camera';
}

// One line per outcome, shared by the toast and the desktop notification.
export function describeSession(s) {
  switch (s.outcome) {
    case 'complete':
      if (s.filesDownloaded === 0) return { text: 'Nothing new to download', tone: 'info' };
      return { text: `${plural(s.filesDownloaded, 'file')}, ${formatBytes(s.bytesDownloaded)}` +
        (s.filesFailed ? `, ${s.filesFailed} failed` : ''), tone: s.filesFailed ? 'warning' : 'success' };
    case 'up_to_date':
      return { text: 'Already up to date', tone: 'info' };
    case 'catalog_refreshed':
      return { text: 'Catalog refreshed', tone: 'info' };
    case 'failed':
      return { text: s.error || 'Sync failed', tone: 'error' };
    case 'cancelled':
      return { text: 'Sync cancelled', tone: 'info' };
    default:
      return { text: 'Sync ended', tone: 'info' };
  }
}

function announce(s) {
  // The cancel action already confirmed itself
  if (s.outcome === 'cancelled') return;
  const name = cameraLabel(s.cameraId);
  const { text, tone } = describeSession(s);
  addToast(`${name}: ${text}`, tone);
  // A drop zone runs while the user is elsewhere; the desktop is where
  // the outcome must land. Electron grants Notification without asking.
  if (typeof Notification !== 'undefined' && !document.hasFocus()) {
    try {
      const n = new Notification(s.outcome === 'failed' ? `${name} sync failed` : `${name} synced`, { body: text, silent: true });
      n.onclick = () => window.focus();
    } catch (_) {}
  }
}

// Local paths downloaded by sessions that finished after `since` (ms)
export function getNewFilePaths(since) {
  const paths = new Set();
  for (const s of history) {
    if (!s.finishedAt || s.finishedAt.getTime() <= since) continue;
    for (const f of s.files) if (f.state === 'done' && f.localPath) paths.add(f.localPath);
  }
  return paths;
}

export function getSessionById(id) {
  return history.find(s => s.id === id) || null;
}
