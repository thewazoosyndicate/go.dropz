// Sync queue store
import { addToast } from './ui.svelte.js';
import { getAllDevices } from './devices.svelte.js';

let syncQueue = $state([]);

export function getSyncQueue() { return syncQueue; }

export function setSyncQueue(entries) {
  // Finished syncs stream a final label ("Already up to date", "Sync
  // complete"), then leave the queue; without a toast the exit would look
  // like a silent failure.
  for (const old of syncQueue) {
    if (entries.some(e => e.cameraId === old.cameraId)) continue;
    const device = Object.values(getAllDevices()).find(d => d.id === old.cameraId);
    const name = device?.name || 'Camera';
    if (old.currentOperation === 'Already up to date') {
      addToast(`${name} already up to date`, 'info');
    } else if (old.currentOperation === 'Sync complete') {
      addToast(`${name} synced`, 'success');
    }
  }
  syncQueue.length = 0;
  entries.forEach(e => syncQueue.push(e));
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
