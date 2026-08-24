// Sync queue store
import { addToast } from './ui.svelte.js';
import { getAllDevices } from './devices.svelte.js';

let syncQueue = $state([]);

export function getSyncQueue() { return syncQueue; }

export function setSyncQueue(entries) {
  // A BLE-verified no-op sync streams "Already up to date" as its final
  // state, then leaves the queue; without a toast the skip would look
  // like a silent failure.
  for (const old of syncQueue) {
    if (old.currentOperation === 'Already up to date' &&
        !entries.some(e => e.cameraId === old.cameraId)) {
      const device = Object.values(getAllDevices()).find(d => d.id === old.cameraId);
      addToast(`${device?.name || 'Camera'} already up to date`, 'info');
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
