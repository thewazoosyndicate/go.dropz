<script>
  import { addToSyncQueue, cancelSync, toggleDeviceManaged, forgetDevice, setCameraAlias, moveCamerasToGroup } from '../lib/grpc/actions.js';
  import { getSyncEntryForCamera, getQueuePosition, getActiveSyncEntry } from '../lib/stores/sync.svelte.js';
  import { getGroups } from '../lib/stores/groups.svelte.js';
  import { displayName, factoryName, findDeviceById } from '../lib/stores/devices.svelte.js';
  import { openCameraSettings, openLibrary, openActivity, addToast } from '../lib/stores/ui.svelte.js';
  import { formatBytes, formatRate, formatEta, formatTimeAgo, plural } from '../lib/format.js';
  import Button from './ui/Button.svelte';
  import Badge from './ui/Badge.svelte';
  import ProgressBar from './ui/ProgressBar.svelte';
  import Menu from './ui/Menu.svelte';
  import PhaseStepper from './PhaseStepper.svelte';

  let { device } = $props();

  let name = $derived(displayName(device));
  let factory = $derived(factoryName(device));
  let subtitle = $derived([device.model, factory !== name ? factory : ''].filter(Boolean).join(' · '));

  let syncEntry = $derived(getSyncEntryForCamera(device.id));
  let inQueue = $derived(!!syncEntry);
  let queuePosition = $derived(inQueue && !device.isSyncing ? getQueuePosition(device.id) : 0);
  let activeName = $derived.by(() => {
    const active = getActiveSyncEntry();
    return active ? displayName(findDeviceById(active.cameraId)) : '';
  });

  // One state at a time, most urgent first
  let state = $derived.by(() => {
    if (!device.isReachable) return 'away';
    if (device.isPairing) return 'pairing';
    if (device.isSyncing && syncEntry) return 'syncing';
    if (inQueue) return 'queued';
    if (device.lastSyncError) return 'failed';
    if (device.newMediaCount > 0) return 'new';
    return 'ok';
  });

  const BADGE = {
    ok: { tone: 'ok', label: 'Up to date' },
    new: { tone: 'info', label: 'new' },
    queued: { tone: 'idle', label: 'Queued' },
    syncing: { tone: 'busy', label: 'Syncing' },
    failed: { tone: 'error', label: 'Failed' },
    away: { tone: 'idle', label: 'Away' },
    pairing: { tone: 'busy', label: 'Pairing' },
  };
  let badge = $derived(state === 'new'
    ? { tone: 'info', label: `${device.newMediaCount} new` }
    : BADGE[state]);

  let transfer = $derived.by(() => {
    const e = syncEntry;
    if (!e?.fileCount) return null;
    const parts = [];
    if (e.bytesTotal > 0) parts.push(`${formatBytes(e.bytesDone)} of ${formatBytes(e.bytesTotal)}`);
    if (e.rateBps > 0) parts.push(formatRate(e.rateBps));
    if (e.rateBps > 0 && e.bytesTotal > e.bytesDone) parts.push(formatEta((e.bytesTotal - e.bytesDone) / e.rateBps));
    return { file: e.fileName, index: e.fileIndex, count: e.fileCount, detail: parts.join(' · ') };
  });

  let lastSyncedText = $derived(formatTimeAgo(device.lastSynced));
  let lastSeenText = $derived(formatTimeAgo(device.lastSeen));
  let storageText = $derived(formatStorage(device.remainingSpaceKb));
  let signalStrength = $derived(getSignalLevel(device.rssi || -100));
  let signalColor = $derived(
    signalStrength >= 3 ? 'var(--state-ok)' :
    signalStrength >= 2 ? 'var(--state-busy)' : 'var(--state-error)'
  );
  let batteryLevel = $derived(Math.max(0, Math.min(100, device.batteryLevel ?? 0)));
  let batteryColor = $derived(
    batteryLevel < 20 ? 'var(--state-error)' :
    batteryLevel < 50 ? 'var(--state-busy)' : 'var(--state-ok)'
  );

  function formatStorage(kb) {
    if (!kb || kb <= 0) return null;
    const gb = kb / (1024 * 1024);
    return gb >= 1 ? `${gb.toFixed(1)} GB` : `${Math.round(kb / 1024)} MB`;
  }

  function getSignalLevel(rssi) {
    if (rssi >= -50) return 4;
    if (rssi >= -65) return 3;
    if (rssi >= -80) return 2;
    return 1;
  }

  // Inline rename: the name becomes an input, Enter saves, Escape cancels
  let renaming = $state(false);
  let draft = $state('');
  let nameInput = $state(null);

  function startRename() {
    draft = device.alias || '';
    renaming = true;
    setTimeout(() => nameInput?.focus(), 0);
  }

  async function commitRename() {
    if (!renaming) return;
    renaming = false;
    const alias = draft.trim();
    if (alias === (device.alias || '')) return;
    try {
      await setCameraAlias(device.id, alias);
    } catch (e) {
      addToast(e.message || 'Rename failed', 'error');
    }
  }

  function onRenameKey(e) {
    if (e.key === 'Enter') commitRename();
    else if (e.key === 'Escape') renaming = false;
  }

  let groupItems = $derived.by(() => {
    const items = getGroups().map(g => ({
      label: g.name, icon: g.syncPaused ? 'fa-pause' : 'fa-layer-group',
      checked: device.groupId === g.id, disabled: device.groupId === g.id,
      onclick: () => moveCamerasToGroup([device.id], g.id),
    }));
    if (device.groupId) {
      items.push({ label: 'No group', icon: 'fa-minus', onclick: () => moveCamerasToGroup([device.id], '') });
    }
    return items;
  });

  let menuItems = $derived([
    { label: 'Rename', icon: 'fa-pen', onclick: startRename },
    { label: 'Move to group', icon: 'fa-layer-group', items: groupItems, disabled: groupItems.length === 0 || !device.id },
    { label: 'Browse media', icon: 'fa-photo-film', onclick: () => openLibrary(device.id), disabled: !device.id },
    { label: 'Camera settings', icon: 'fa-sliders-h', onclick: handleSettings, disabled: device.isSyncing || !device.isReachable },
    { label: 'Show activity', icon: 'fa-wave-square', onclick: () => openActivity(device.id) },
    { label: 'Unmanage', icon: 'fa-link-slash', danger: true, confirm: true,
      onclick: () => toggleDeviceManaged(device.macAddress, false) },
    { label: 'Forget pairing', icon: 'fa-eraser', danger: true, confirm: true,
      onclick: () => forgetDevice(device.macAddress) },
  ]);

  function handleSettings() {
    openCameraSettings({ type: 'camera', id: device.id, name, referenceCameraId: device.id });
  }
</script>

<div class="card state-{state}">
  <div class="card-header">
    <div class="name-row">
      {#if renaming}
        <input class="name-input" bind:this={nameInput} bind:value={draft} maxlength="40"
               placeholder={factory} aria-label="Camera name"
               onkeydown={onRenameKey} onblur={commitRename} />
      {:else}
        <button class="name" title="Rename" onclick={startRename}>{name}</button>
      {/if}
      <Badge tone={badge.tone} pulse={state === 'syncing' || state === 'pairing'}>{badge.label}</Badge>
      {#if device.numPhotos > 0 || device.numVideos > 0}
        <span class="media-counts" title="Files on the camera">
          {#if device.numVideos > 0}<i class="fas fa-video" aria-hidden="true"></i> {device.numVideos}{/if}
          {#if device.numPhotos > 0 && device.numVideos > 0} &middot; {/if}
          {#if device.numPhotos > 0}<i class="fas fa-image" aria-hidden="true"></i> {device.numPhotos}{/if}
        </span>
      {/if}
    </div>
    {#if subtitle}<span class="subtitle">{subtitle}</span>{/if}
  </div>

  <div class="card-body">
    <div class="info-row">
      <div class="signal-bars" title="Signal">
        {#each [1, 2, 3, 4] as level}
          <div class="bar" class:filled={level <= signalStrength && device.isReachable}
               style:--bar-color={signalColor}></div>
        {/each}
      </div>
      {#if storageText}
        <span class="meta" title="Free space on the card"><i class="fas fa-sd-card" aria-hidden="true"></i> {storageText}</span>
      {/if}
      {#if state === 'away'}
        <span class="meta">last seen {lastSeenText || 'a while ago'}</span>
      {:else if lastSyncedText}
        <span class="meta" title="Last sync"><i class="fas fa-rotate" aria-hidden="true"></i> {lastSyncedText}</span>
      {/if}
      {#if device.batteryLevel != null}
        <div class="battery" title="Battery {batteryLevel}%">
          <div class="battery-icon">
            <div class="battery-fill" style:width="{batteryLevel}%" style:background-color={batteryColor}></div>
          </div>
          <span class="meta">{batteryLevel}%</span>
        </div>
      {/if}
    </div>

    {#if state === 'syncing'}
      <div class="block busy" aria-live="polite">
        <PhaseStepper compact phase={syncEntry.phase} phases={syncEntry.phases} />
        <ProgressBar value={syncEntry.progressPercent || 0} shimmer label="Sync progress" />
        {#if transfer}
          <div class="file-row">
            <span class="file-name">{transfer.file}</span>
            <span class="muted">{transfer.index} of {transfer.count}</span>
          </div>
          {#if transfer.detail}<span class="detail">{transfer.detail}</span>{/if}
        {:else}
          <span class="detail">{syncEntry.currentOperation || 'Preparing...'}</span>
        {/if}
      </div>
    {:else if state === 'queued'}
      <div class="block idle">
        <div class="headline"><i class="fas fa-clock" aria-hidden="true"></i> {queuePosition === 1 ? 'Next in line' : `In line, #${queuePosition}`}</div>
        <span class="detail">
          {#if device.newMediaCount > 0}{plural(device.newMediaCount, 'new clip')}. {/if}
          {#if activeName}Starts after {activeName}.{:else}Starts when the radio is free.{/if}
        </span>
      </div>
    {:else if state === 'failed'}
      <div class="block error">
        <div class="headline">
          <i class="fas fa-triangle-exclamation" aria-hidden="true"></i> {device.lastSyncError}
        </div>
        <span class="detail">Press Retry now, or wait for the next new clip.</span>
      </div>
    {:else if state === 'new'}
      <div class="block info">
        <div class="headline"><i class="fas fa-video" aria-hidden="true"></i> {plural(device.newMediaCount, 'new clip')} on the camera</div>
        <span class="detail">Syncs on its own once the camera is idle.</span>
      </div>
    {:else if state === 'away'}
      <div class="block idle">
        <div class="headline">
          {#if device.lastSyncError}
            <i class="fas fa-triangle-exclamation" aria-hidden="true"></i> Last sync failed
          {:else if device.newMediaCount > 0}
            <i class="fas fa-video" aria-hidden="true"></i> {plural(device.newMediaCount, 'new clip')} when it left
          {:else}
            <i class="fas fa-check" aria-hidden="true"></i> Up to date when it left
          {/if}
        </div>
        <span class="detail">
          {#if device.lastSyncError}{device.lastSyncError}. {/if}Checks for new clips when it returns.
        </span>
      </div>
    {:else if state === 'pairing'}
      <div class="block busy">
        <div class="headline"><i class="fas fa-spinner fa-spin" aria-hidden="true"></i> Pairing</div>
        <span class="detail">Confirm on the camera screen if it asks.</span>
      </div>
    {:else}
      <div class="block ok">
        <div class="headline"><i class="fas fa-check" aria-hidden="true"></i> Everything in your library</div>
        <span class="detail">{lastSyncedText ? `Last sync ${lastSyncedText}.` : 'Not synced yet.'}</span>
      </div>
    {/if}
  </div>

  <div class="card-actions">
    {#if state === 'syncing'}
      <Button variant="danger-outline" grow onclick={() => cancelSync(device.macAddress)}>Cancel</Button>
      <Button icon="fa-wave-square" grow onclick={() => openActivity(device.id)}>Details</Button>
    {:else if state === 'queued'}
      <Button grow onclick={() => cancelSync(device.macAddress)}>Remove from queue</Button>
      <Button icon="fa-photo-film" title="Browse media" onclick={() => openLibrary(device.id)} disabled={!device.id} />
      <Button icon="fa-sliders-h" title="Camera settings" onclick={handleSettings} />
    {:else if state === 'failed'}
      <Button variant="primary" icon="fa-rotate" grow onclick={() => addToSyncQueue(device.macAddress)}>Retry now</Button>
      <Button grow onclick={() => openActivity(device.id)}>What happened</Button>
    {:else if state === 'new'}
      <Button variant="success" grow onclick={() => addToSyncQueue(device.macAddress)}>Sync now</Button>
      <Button icon="fa-photo-film" title="Browse media" onclick={() => openLibrary(device.id)} disabled={!device.id} />
      <Button icon="fa-sliders-h" title="Camera settings" onclick={handleSettings} />
    {:else if state === 'away'}
      <Button grow disabled title="The camera is out of range">Out of range</Button>
      <Button icon="fa-photo-film" title="Browse media" onclick={() => openLibrary(device.id)} disabled={!device.id} />
    {:else if state === 'pairing'}
      <Button grow disabled>Pairing...</Button>
    {:else}
      <Button grow onclick={() => addToSyncQueue(device.macAddress)}>Check now</Button>
      <Button icon="fa-photo-film" title="Browse media" onclick={() => openLibrary(device.id)} disabled={!device.id} />
      <Button icon="fa-sliders-h" title="Camera settings" onclick={handleSettings} />
    {/if}
    <Menu items={menuItems} />
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
    border-left: 4px solid var(--state-idle);
    transition: box-shadow 0.2s, border-color 0.2s;
  }

  .card:hover { box-shadow: var(--shadow-md); }

  /* Border color means state only */
  .state-ok { border-left-color: var(--state-ok); }
  .state-new { border-left-color: var(--state-info); }
  .state-syncing, .state-pairing { border-left-color: var(--state-busy); }
  .state-failed { border-left-color: var(--state-error); }
  .state-away { opacity: 0.7; }

  .card-header { display: flex; flex-direction: column; gap: 2px; }

  .name-row { display: flex; align-items: center; gap: 8px; min-height: 28px; }

  .name {
    font-weight: 600;
    font-size: 1.05rem;
    color: var(--text-primary);
    background: none;
    border: none;
    padding: 0;
    cursor: text;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
  }

  .name-input {
    font-weight: 600;
    font-size: 1.05rem;
    color: var(--text-primary);
    background: var(--light-bg);
    border: 1px solid var(--primary-color);
    border-radius: 6px;
    padding: 2px 6px;
    min-width: 0;
    flex: 1;
  }

  .subtitle { font-size: 0.75rem; color: var(--text-muted); }

  .media-counts {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-left: auto;
    flex-shrink: 0;
  }

  .card-body { display: flex; flex-direction: column; gap: 8px; flex: 1; }

  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
  }

  .meta { font-size: 0.75rem; color: var(--text-secondary); white-space: nowrap; }

  .signal-bars { display: flex; align-items: flex-end; gap: 2px; height: 14px; }
  .bar { width: 4px; background-color: var(--border-color); border-radius: 1px; }
  .bar:nth-child(1) { height: 4px; }
  .bar:nth-child(2) { height: 7px; }
  .bar:nth-child(3) { height: 10px; }
  .bar:nth-child(4) { height: 14px; }
  .bar.filled { background-color: var(--bar-color); }

  .battery { display: flex; align-items: center; gap: 4px; }
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
  .battery-fill { height: 100%; transition: width 0.3s; }

  .block {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 8px 10px;
    border-radius: 6px;
    font-size: 0.75rem;
    color: var(--text-primary);
  }
  .block.ok { background: var(--state-ok-tint); }
  .block.busy { background: var(--state-busy-tint); }
  .block.error { background: var(--state-error-tint); }
  .block.info { background: var(--state-info-tint); }
  .block.idle { background: var(--state-idle-tint); }

  .headline { display: flex; align-items: center; gap: 8px; font-weight: 600; }
  .headline i { width: 14px; text-align: center; }
  .block.ok .headline i { color: var(--state-ok); }
  .block.error .headline i { color: var(--state-error); }
  .block.info .headline i { color: var(--state-info); }
  .block.idle .headline i, .block.busy .headline i { color: var(--text-secondary); }

  .detail { color: var(--text-secondary); font-variant-numeric: tabular-nums; }
  .block.error .detail, .block.ok .detail, .block.info .detail, .block.idle .detail { padding-left: 22px; }

  .file-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    gap: 8px;
    font-variant-numeric: tabular-nums;
  }
  .file-name {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 0.72rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .muted { color: var(--text-secondary); white-space: nowrap; }

  .card-actions { display: flex; gap: 8px; margin-top: auto; }
</style>
