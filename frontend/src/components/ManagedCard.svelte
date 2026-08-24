<script>
  import { addToSyncQueue, cancelSync, toggleDeviceManaged } from '../lib/grpc/actions.js';
  import { getSyncQueue } from '../lib/stores/sync.svelte.js';
  import { getAllDevices } from '../lib/stores/devices.svelte.js';
  import { openCameraSettings, openLibrary } from '../lib/stores/ui.svelte.js';

  let { device } = $props();

  let displayName = $derived(
    device.wifiSsid?.trim() ? device.wifiSsid.substring(0, 12) : (device.name || 'Unknown GoPro')
  );

  let statusCode = $derived(getStatusCode(device));
  let syncEntry = $derived(getSyncQueue().find(e => e.cameraId === device.id));
  let isInSyncQueue = $derived(!!syncEntry);

  // The stream delivers the queue already sorted by priority then age, so
  // the position is the index among entries whose camera is not syncing.
  let queuePosition = $derived.by(() => {
    if (!isInSyncQueue || device.isSyncing) return 0;
    const devices = Object.values(getAllDevices());
    const waiting = getSyncQueue().filter(e =>
      !devices.find(d => d.id === e.cameraId)?.isSyncing);
    return waiting.findIndex(e => e.cameraId === device.id) + 1;
  });

  let downloadDetail = $derived.by(() => {
    const e = syncEntry;
    if (!e?.fileCount) return null;
    const parts = [`File ${e.fileIndex}/${e.fileCount}`];
    if (e.bytesTotal > 0) parts.push(`${formatBytes(e.bytesDone)} / ${formatBytes(e.bytesTotal)}`);
    if (e.rateBps > 0) parts.push(`${(e.rateBps / 1e6).toFixed(1)} MB/s`);
    if (e.rateBps > 0 && e.bytesTotal > e.bytesDone) {
      parts.push(formatEta((e.bytesTotal - e.bytesDone) / e.rateBps));
    }
    return parts.join(' · ');
  });

  function formatBytes(bytes) {
    if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
    return `${Math.max(1, Math.round(bytes / 1e6))} MB`;
  }

  function formatEta(seconds) {
    if (seconds < 90) return `~${Math.max(5, Math.round(seconds / 5) * 5)}s left`;
    return `~${Math.round(seconds / 60)} min left`;
  }

  let signalStrength = $derived(getSignalLevel(device.rssi || -100));
  let signalColor = $derived(
    signalStrength >= 3 ? 'var(--secondary-color)' :
    signalStrength >= 2 ? 'var(--warning-color)' : 'var(--danger-color)'
  );

  let storageText = $derived(formatStorage(device.remainingSpaceKb));
  let lastSyncedText = $derived(formatTimeAgo(device.lastSynced));

  let batteryLevel = $derived(Math.max(0, Math.min(100, device.batteryLevel ?? 0)));
  let batteryColor = $derived(
    batteryLevel < 20 ? 'var(--danger-color)' :
    batteryLevel < 50 ? 'var(--warning-color)' : 'var(--secondary-color)'
  );

  function formatStorage(kb) {
    if (!kb || kb <= 0) return null;
    const gb = kb / (1024 * 1024);
    return gb >= 1 ? `${gb.toFixed(1)} GB` : `${Math.round(kb / 1024)} MB`;
  }

  function formatTimeAgo(date) {
    if (!date || date.getTime() < 86400000) return null;
    const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    return `${days}d ago`;
  }

  function getStatusCode(d) {
    if (!d.isReachable) return 'unreachable';
    if (d.isPairing) return 'pairing';
    if (d.isSyncing) return 'syncing';
    if (d.isPaired && d.isManaged) return 'managed';
    if (d.isPaired) return 'available';
    return 'discovered';
  }

  function getSignalLevel(rssi) {
    if (rssi >= -50) return 4;
    if (rssi >= -65) return 3;
    if (rssi >= -80) return 2;
    return 1;
  }

  function handleSync() {
    if (isInSyncQueue) cancelSync(device.macAddress);
    else addToSyncQueue(device.macAddress);
  }

  function handleUnmanage() {
    toggleDeviceManaged(device.macAddress, false);
  }

  function handleBrowse() {
    openLibrary(device.id);
  }

  function handleSettings() {
    openCameraSettings({
      type: 'camera',
      id: device.id,
      name: displayName,
      referenceCameraId: device.id,
    });
  }
</script>

<div class="card status-{statusCode}">
  <div class="card-header">
    <div class="name-row">
      <span class="name">{displayName}</span>
      <span class="status-badge">{statusCode}</span>
      {#if device.numPhotos > 0 || device.numVideos > 0}
        <span class="media-counts">
          {#if device.numPhotos > 0}<i class="fas fa-image"></i> {device.numPhotos}{/if}
          {#if device.numPhotos > 0 && device.numVideos > 0} · {/if}
          {#if device.numVideos > 0}<i class="fas fa-video"></i> {device.numVideos}{/if}
        </span>
      {/if}
    </div>
    {#if device.model}
      <span class="model">{device.model}</span>
    {/if}
  </div>

  <div class="card-body">
    <div class="info-row">
      <div class="signal">
        <div class="signal-bars">
          {#each [1, 2, 3, 4] as level}
            <div class="bar" class:filled={level <= signalStrength}
                 style:--bar-color={signalColor}></div>
          {/each}
        </div>
      </div>
      {#if storageText}
        <span class="storage"><i class="fas fa-sd-card"></i> {storageText}</span>
      {/if}
      {#if lastSyncedText}
        <span class="last-synced"><i class="fas fa-sync"></i> {lastSyncedText}</span>
      {/if}
      {#if device.batteryLevel != null}
        <div class="battery">
          <div class="battery-icon">
            <div class="battery-fill" style:width="{batteryLevel}%"
                 style:background-color={batteryColor}></div>
          </div>
          <span class="battery-text">{batteryLevel}%</span>
        </div>
      {/if}
    </div>

    {#if device.lastSyncError && !device.isSyncing}
      <div class="sync-error">
        <i class="fas fa-exclamation-triangle"></i> {device.lastSyncError}
      </div>
    {/if}

    {#if device.isSyncing && syncEntry}
      <div class="sync-progress">
        <div class="progress-bar">
          <div class="progress-fill" style:width="{syncEntry.progressPercent || 0}%"></div>
        </div>
        <span class="progress-label">{syncEntry.currentOperation || 'Preparing...'}</span>
        {#if downloadDetail}
          <span class="progress-detail">{downloadDetail}</span>
        {/if}
      </div>
    {:else if isInSyncQueue}
      <div class="queue-badge">
        <i class="fas fa-clock"></i>
        {queuePosition === 1 ? 'Next in queue' : queuePosition > 1 ? `In queue · #${queuePosition}` : 'In queue'}
      </div>
    {/if}
  </div>

  <div class="card-actions">
    <button class="btn {isInSyncQueue ? 'btn-danger' : 'btn-sync'}" onclick={handleSync}>
      {isInSyncQueue ? 'Cancel' : 'Sync'}
    </button>
    <button class="btn btn-outline" onclick={handleBrowse} disabled={!device.id}
            title="Browse media" aria-label="Browse media">
      <i class="fas fa-photo-film"></i>
    </button>
    <button class="btn btn-outline" onclick={handleSettings} disabled={device.isSyncing}
            title="Camera settings" aria-label="Camera settings">
      <i class="fas fa-sliders-h"></i>
    </button>
    <button class="btn btn-outline" onclick={handleUnmanage}>Unmanage</button>
  </div>
</div>

<style>
  .card {
    background-color: var(--panel-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    border-left: 4px solid var(--border-color);
    transition: all 0.2s;
  }

  .card:hover {
    box-shadow: var(--shadow-md);
  }

  /* Status colors */
  .status-managed     { border-left-color: #3498db; }
  .status-syncing     { border-left-color: #f39c12; }
  .status-unreachable { border-left-color: #95a5a6; }
  .status-available   { border-left-color: #2ecc71; }
  .status-pairing     { border-left-color: #e67e22; }
  .status-discovered  { border-left-color: #9b59b6; }

  .card-header {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .name-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .name {
    font-weight: 600;
    font-size: 1.05rem;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .status-badge {
    font-size: 0.65rem;
    font-weight: 500;
    text-transform: uppercase;
    padding: 1px 6px;
    border-radius: 10px;
    color: white;
    background-color: var(--primary-color);
    flex-shrink: 0;
  }
  .status-syncing .status-badge { background-color: #f39c12; animation: pulse 1s infinite; }
  .status-unreachable .status-badge { background-color: #95a5a6; }
  .status-pairing .status-badge { background-color: #e67e22; animation: pulse 1s infinite; }

  .model {
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .card-body {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .signal-bars {
    display: flex;
    align-items: flex-end;
    gap: 2px;
    height: 14px;
  }

  .bar {
    width: 4px;
    background-color: var(--border-color);
    border-radius: 1px;
  }
  .bar:nth-child(1) { height: 4px; }
  .bar:nth-child(2) { height: 7px; }
  .bar:nth-child(3) { height: 10px; }
  .bar:nth-child(4) { height: 14px; }
  .bar.filled { background-color: var(--bar-color); }

  .battery {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .battery-icon {
    width: 22px;
    height: 10px;
    border: 1px solid var(--text-muted);
    border-radius: 2px;
    position: relative;
    overflow: hidden;
  }

  .battery-icon::after {
    content: '';
    position: absolute;
    right: -4px;
    top: 2px;
    width: 2px;
    height: 5px;
    background: var(--text-muted);
    border-radius: 0 2px 2px 0;
  }

  .battery-fill {
    height: 100%;
    transition: width 0.3s;
  }

  .battery-text {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .storage, .last-synced {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .sync-error {
    font-size: 0.7rem;
    color: var(--danger-color);
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .media-counts {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-left: auto;
    flex-shrink: 0;
  }

  .sync-progress {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .progress-bar {
    height: 4px;
    background-color: var(--border-color);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--warning-color), var(--secondary-color));
    border-radius: 2px;
    transition: width 0.3s;
    position: relative;
  }

  .progress-fill::after {
    content: '';
    position: absolute;
    top: 0;
    left: -50%;
    width: 50%;
    height: 100%;
    background: linear-gradient(90deg, transparent, rgba(255,255,255,0.4), transparent);
    animation: sync-progress-shimmer 1.5s infinite;
  }

  .progress-label {
    font-size: 0.65rem;
    color: var(--text-muted);
    font-style: italic;
    text-align: center;
  }

  .progress-detail {
    font-size: 0.7rem;
    color: var(--text-secondary);
    text-align: center;
    font-variant-numeric: tabular-nums;
  }

  .queue-badge {
    font-size: 0.75rem;
    color: var(--warning-color);
    display: flex;
    align-items: center;
    gap: 4px;
  }


  .card-actions {
    display: flex;
    gap: 8px;
    margin-top: auto;
  }

  .btn {
    flex: 1;
    padding: 8px 12px;
    border: none;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    color: white;
    min-height: 36px;
    transition: all 0.2s;
  }

  .btn:hover { transform: translateY(-1px); }
  .btn:active { transform: translateY(0); }

  .btn-sync { background-color: var(--secondary-color); }
  .btn-sync:hover { background-color: var(--secondary-dark); }

  .btn-danger { background-color: var(--danger-color); }
  .btn-danger:hover { background-color: var(--danger-dark); }

  .btn-outline {
    background-color: transparent;
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
  }
  .btn-outline:hover {
    border-color: var(--text-secondary);
  }
</style>
