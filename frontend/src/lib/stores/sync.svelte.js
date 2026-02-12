// Sync queue store
let syncQueue = $state([]);

export function getSyncQueue() { return syncQueue; }

export function setSyncQueue(entries) {
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
