// Fake DropzService for the demo recording. Plain objects hold the state;
// the driver mutates them and calls notify() to push a fresh snapshot on
// every open watch stream. Only the RPCs the storyboard touches exist.
const path = require('path');

const FRONTEND = path.resolve(__dirname, '..', '..', 'frontend');
const grpc = require(path.join(FRONTEND, 'node_modules', '@grpc', 'grpc-js'));
const svc = require(path.join(FRONTEND, 'src', 'proto', 'service_grpc_pb.js'));
const gp = require(path.join(FRONTEND, 'src', 'proto', 'gopro_pb.js'));
const vp = require(path.join(FRONTEND, 'src', 'proto', 'video_pb.js'));
const cp = require(path.join(FRONTEND, 'src', 'proto', 'config_pb.js'));
const { Timestamp } = require(path.join(FRONTEND, 'node_modules', 'google-protobuf', 'google', 'protobuf', 'timestamp_pb.js'));

const PHASE = { waiting: 0, connect: 1, link: 2, catalog: 3, transfer: 4 };
const FILE = { queued: 0, downloading: 1, done: 2, failed: 3, skipped: 4 };
const OUTCOME = { complete: 1, up_to_date: 2, catalog_refreshed: 3, failed: 4, cancelled: 5 };

// jspb asserts integer values on int fields; the simulation produces floats
const int = (n) => Math.round(n || 0);

function ts(d) { return d ? Timestamp.fromDate(new Date(d)) : undefined; }

function cameraWithState(c) {
  const cam = new gp.Camera();
  cam.setId(c.id); cam.setName(c.name); cam.setAlias(c.alias || '');
  cam.setBleAddress(c.ble); cam.setWifiSsid(c.ssid); cam.setWifiPassword('demo');
  cam.setRssi(int(c.rssi));
  const st = new gp.CameraStatus();
  if (c.lastSeen) st.setLastSeen(ts(c.lastSeen));
  if (c.lastSynced) st.setLastSynced(ts(c.lastSynced));
  st.setIsPairing(!!c.isPairing); st.setIsPaired(!!c.isPaired); st.setIsManaged(!!c.isManaged);
  st.setIsReachable(!!c.isReachable); st.setIsSynced(!!c.isSynced); st.setIsSyncing(!!c.isSyncing);
  st.setLastSyncError(c.lastSyncError || ''); st.setInPairingMode(!!c.inPairingMode);
  st.setNewMediaCount(c.newMediaCount || 0);
  const md = new gp.CameraMetadata();
  md.setId(c.id); md.setFirmwareVersion(c.firmware || ''); md.setModel(c.model || '');
  md.setSerialNumber(c.serial || ''); md.setBatteryLevel(int(c.battery ?? 0));
  md.setNumPhotos(int(c.numPhotos || 0)); md.setNumVideos(int(c.numVideos || 0));
  md.setRemainingSpaceKb(int(c.remainingKb || 0));
  const cws = new gp.CameraWithState();
  cws.setCamera(cam); cws.setStatus(st); cws.setGroupId(c.groupId || ''); cws.setMetadata(md);
  return cws;
}

function syncFile(f) {
  const m = new gp.SyncFile();
  m.setName(f.name); m.setCameraPath(f.cameraPath || `/100GOPRO/${f.name}`);
  m.setSizeBytes(int(f.sizeBytes)); m.setState(FILE[f.state] || 0);
  m.setBytesDone(int(f.bytesDone || 0)); m.setError(f.error || '');
  m.setLocalPath(f.localPath || ''); m.setDurationMs(int(f.durationMs || 0));
  return m;
}

function phaseTiming(p) {
  const m = new gp.SyncPhaseTiming();
  m.setPhase(PHASE[p.phase]);
  if (p.startedAt) m.setStartedAt(ts(p.startedAt));
  if (p.finishedAt) m.setFinishedAt(ts(p.finishedAt));
  return m;
}

function queueEntry(e) {
  const m = new gp.SyncQueueEntry();
  m.setCameraId(e.cameraId); m.setQueuedAt(ts(e.queuedAt)); m.setPriority(int(e.priority || 5));
  m.setProgressPercent(int(e.progressPercent || 0)); m.setCurrentOperation(e.currentOperation || '');
  m.setFileIndex(int(e.fileIndex || 0)); m.setFileCount(int(e.fileCount || 0)); m.setFileName(e.fileName || '');
  m.setFileBytes(int(e.fileBytes || 0)); m.setFileTotal(int(e.fileTotal || 0));
  m.setBytesDone(int(e.bytesDone || 0)); m.setBytesTotal(int(e.bytesTotal || 0)); m.setRateBps(int(e.rateBps || 0));
  if (e.startedAt) m.setStartedAt(ts(e.startedAt));
  m.setPhase(PHASE[e.phase || 'waiting']);
  m.setPhasesList((e.phases || []).map(phaseTiming));
  m.setFilesList((e.files || []).map(syncFile));
  m.setStepIndex(int(e.stepIndex || 0)); m.setStepCount(int(e.stepCount || 9));
  return m;
}

function session(s) {
  const m = new gp.SyncSession();
  m.setId(s.id); m.setCameraId(s.cameraId);
  m.setStartedAt(ts(s.startedAt)); m.setFinishedAt(ts(s.finishedAt));
  m.setOutcome(OUTCOME[s.outcome]); m.setError(s.error || ''); m.setFailedStep(s.failedStep || '');
  m.setStepIndex(int(s.stepIndex || 0)); m.setStepCount(int(s.stepCount || 9));
  m.setFilesDownloaded(int(s.filesDownloaded || 0)); m.setFilesFailed(int(s.filesFailed || 0));
  m.setFilesSkipped(int(s.filesSkipped || 0)); m.setBytesDownloaded(int(s.bytesDownloaded || 0));
  m.setFilesList((s.files || []).map(syncFile));
  m.setPhasesList((s.phases || []).map(phaseTiming));
  m.setSelection(!!s.selection);
  return m;
}

function videoFile(v) {
  const m = new vp.VideoFile();
  m.setId(v.id); m.setName(v.name); m.setPath(v.path); m.setSizeBytes(int(v.sizeBytes));
  m.setCreatedAt(ts(v.createdAt)); m.setSyncedAt(ts(v.syncedAt || v.createdAt));
  m.setCameraId(v.cameraId); m.setMimeType(v.mimeType || 'video/mp4');
  m.setDurationSeconds(int(v.durationSeconds || 12));
  m.setThumbnailPath(v.thumbnailPath || ''); m.setHasProcessed(true);
  m.setPreviewPath(v.previewPath || '');
  return m;
}

function group(g) {
  const m = new gp.Group();
  m.setId(g.id); m.setName(g.name); m.setCameraIdsList(g.cameraIds);
  m.setCreatedAt(ts(g.createdAt)); m.setUpdatedAt(ts(g.updatedAt || g.createdAt));
  m.setSyncPaused(!!g.syncPaused);
  return m;
}

function createFakeServer({ address, state, hooks = {} }) {
  const watchers = { discovered: new Set(), managed: new Set(), queue: new Set() };

  function watch(kind, snapshot) {
    return (call) => {
      watchers[kind].add(call);
      call.write(snapshot());
      const drop = (why) => (err) => { watchers[kind].delete(call); console.log(`[fake] ${kind} stream ${why}`, err?.message || ''); };
      call.on('cancelled', drop('cancelled')); call.on('error', drop('error')); call.on('close', drop('closed'));
    };
  }

  const snapshots = {
    discovered: () => {
      const r = new gp.GetDiscoveredCamerasResponse();
      r.setCamerasList(state.cameras.filter(c => c.visible && !(c.isPaired && c.isManaged))
        .map(c => { const d = new gp.DiscoveredCamera(); d.setCameraState(cameraWithState(c)); return d; }));
      return r;
    },
    managed: () => {
      const r = new gp.GetManagedCamerasResponse();
      r.setCamerasList(state.cameras.filter(c => c.visible && c.isPaired && c.isManaged)
        .map(c => { const d = new gp.ManagedCamera(); d.setCameraState(cameraWithState(c)); return d; }));
      return r;
    },
    queue: () => {
      const r = new gp.GetSyncQueueResponse();
      r.setQueueList(state.queue.map(queueEntry));
      return r;
    },
  };

  function notify(...kinds) {
    if (kinds.length === 0) kinds = Object.keys(watchers);
    // Fresh lastSeen on every push: the renderer drops cameras silent for 30 s
    const now = Date.now();
    for (const c of state.cameras) if (c.isReachable) c.lastSeen = now;
    for (const k of kinds) for (const call of watchers[k]) call.write(snapshots[k]());
  }

  const ok = (Resp, fill) => (call, cb) => { const r = new Resp(); if (fill) fill(r, call.request); cb(null, r); };

  const handlers = {
    watchDiscoveredCameras: watch('discovered', snapshots.discovered),
    watchManagedCameras: watch('managed', snapshots.managed),
    watchSyncQueue: watch('queue', snapshots.queue),
    getDiscoveredCameras: ok(gp.GetDiscoveredCamerasResponse, r => r.setCamerasList(snapshots.discovered().getCamerasList())),
    getManagedCameras: ok(gp.GetManagedCamerasResponse, r => r.setCamerasList(snapshots.managed().getCamerasList())),
    getSyncQueue: ok(gp.GetSyncQueueResponse, r => r.setQueueList(snapshots.queue().getQueueList())),
    getSyncHistory: ok(gp.GetSyncHistoryResponse, r => r.setSessionsList(state.history.map(session))),
    getGroups: ok(gp.GetGroupsResponse, r => r.setGroupsList(state.groups.map(group))),
    getConfig: ok(cp.GetConfigResponse, r => {
      const c = new cp.Config();
      c.setPairModeEnabled(false); c.setSyncEnabled(true); c.setScanIntervalSeconds(30);
      c.setConnectTimeoutSeconds(30); c.setDaysThreshold(7); c.setDestinationFolder(state.libraryDir);
      c.setInactivityTimeoutSeconds(60); c.setStatusCheckIntervalSeconds(300); c.setCheckOnReturn(true);
      c.setTurboEnabled(true); c.setSetTimeEnabled(true); c.setLogLevel('info');
      r.setConfig(c);
    }),
    getVideos: ok(vp.GetVideosResponse, r => {
      r.setVideosList(state.videos.map(videoFile)); r.setTotalCount(state.videos.length);
    }),
    getVideoKeyframes: ok(vp.GetVideoKeyframesResponse, r => {
      // GoPro writes a keyframe about every second
      const ms = []; for (let t = 0; t < 12000; t += 1000) ms.push(t);
      r.setKeyframeMsList(ms); r.setDurationMs(int(12000));
    }),
    previewVideo: ok(vp.PreviewVideoResponse, (r, req) => {
      const v = state.videos.find(x => x.path === req.getVideoPath());
      r.setPreviewPath(v?.previewPath || '');
    }),
    pairCamera: (call, cb) => {
      const cam = state.cameras.find(c => c.id === call.request.getCameraId());
      cam.isPairing = true; notify('discovered');
      setTimeout(() => {
        Object.assign(cam, { isPairing: false, isPaired: true, isManaged: true, isSynced: false });
        if (hooks.onPaired) hooks.onPaired(cam);
        const r = new gp.PairCameraResponse();
        r.setSuccess(true); r.setMessage('Paired');
        const m = new gp.ManagedCamera(); m.setCameraState(cameraWithState(cam)); r.setCamera(m);
        cb(null, r);
        notify('managed', 'discovered');
      }, hooks.pairDelayMs || 1200);
    },
    trimVideo: (call, cb) => {
      const req = call.request;
      setTimeout(() => {
        const out = hooks.onTrim ? hooks.onTrim(req.getVideoPath(), req.getStartMs(), req.getEndMs()) : null;
        if (!out) return cb({ code: grpc.status.NOT_FOUND, message: 'No such clip' });
        const r = new vp.TrimVideoResponse();
        r.setOutputPath(out.path); r.setName(out.name); r.setThumbnailPath(out.thumbnailPath || '');
        r.setPreviewPath(out.previewPath || ''); r.setSizeBytes(int(out.sizeBytes));
        r.setStartMs(out.startMs); r.setEndMs(out.endMs);
        cb(null, r);
      }, hooks.trimDelayMs || 1500);
    },
    forceSync: ok(gp.ForceSyncResponse, r => { r.setSuccess(true); }),
    cancelSync: ok(gp.CancelSyncResponse, r => { r.setSuccess(true); }),
    updateSetting: ok(cp.Config),
  };

  const server = new grpc.Server();
  server.addService(svc.DropzServiceService, handlers);

  return {
    notify,
    start: () => new Promise((resolve, reject) => {
      server.bindAsync(address, grpc.ServerCredentials.createInsecure(), (err) => err ? reject(err) : resolve());
    }),
    stop: () => server.forceShutdown(),
  };
}

module.exports = { createFakeServer };
