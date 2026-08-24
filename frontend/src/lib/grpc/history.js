// Sync history: one unary load, refreshed whenever a sync leaves the queue.
// Separate from actions.js so the stream layer can import it without a
// cycle through the action helpers.
import { getClient, proto } from './client.js';
import { setSyncHistory } from '../stores/sync.svelte.js';

const PHASES = ['waiting', 'connect', 'link', 'catalog', 'transfer'];
const FILE_STATES = ['queued', 'downloading', 'done', 'failed', 'skipped'];
const OUTCOMES = ['unknown', 'complete', 'up_to_date', 'catalog_refreshed', 'failed', 'cancelled'];

export function phaseName(value) { return PHASES[value] || 'waiting'; }

export function processSyncFile(f) {
  return {
    name: f.getName(),
    cameraPath: f.getCameraPath(),
    sizeBytes: f.getSizeBytes(),
    state: FILE_STATES[f.getState()] || 'queued',
    bytesDone: f.getBytesDone(),
    error: f.getError(),
    localPath: f.getLocalPath(),
    durationMs: f.getDurationMs(),
  };
}

export function processPhaseTiming(p) {
  return {
    phase: phaseName(p.getPhase()),
    startedAt: p.getStartedAt()?.toDate() || null,
    finishedAt: p.getFinishedAt()?.toDate() || null,
  };
}

export function processSyncSession(s) {
  return {
    id: s.getId(),
    cameraId: s.getCameraId(),
    startedAt: s.getStartedAt()?.toDate() || null,
    finishedAt: s.getFinishedAt()?.toDate() || null,
    outcome: OUTCOMES[s.getOutcome()] || 'unknown',
    error: s.getError(),
    failedStep: s.getFailedStep(),
    stepIndex: s.getStepIndex(),
    stepCount: s.getStepCount(),
    filesDownloaded: s.getFilesDownloaded(),
    filesFailed: s.getFilesFailed(),
    filesSkipped: s.getFilesSkipped(),
    bytesDownloaded: s.getBytesDownloaded(),
    files: s.getFilesList().map(processSyncFile),
    phases: s.getPhasesList().map(processPhaseTiming),
    selection: s.getSelection(),
  };
}

export function loadSyncHistory(limit = 100) {
  return new Promise((resolve) => {
    const client = getClient();
    const request = new proto.GetSyncHistoryRequest();
    request.setLimit(limit);
    client.getSyncHistory(request, (error, response) => {
      if (error) {
        console.error('Error loading sync history:', error.message);
        resolve([]);
        return;
      }
      const sessions = response.getSessionsList().map(processSyncSession);
      setSyncHistory(sessions);
      resolve(sessions);
    });
  });
}
