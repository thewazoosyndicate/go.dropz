<script>
  // The camera's cached catalog: loads it, keeps the preview session, and
  // hands the items to the same grid the local library uses.
  import { onDestroy } from 'svelte';
  import { fetchCameraMedia, requestMediaDownload, previewMedia, setPreviewSession } from '../lib/grpc/actions.js';
  import { addToast, addLog, openPlayer } from '../lib/stores/ui.svelte.js';
  import { getSyncQueue } from '../lib/stores/sync.svelte.js';
  import { findDeviceById } from '../lib/stores/devices.svelte.js';
  import { fromCatalogItem, filterKind, sortItems } from '../lib/media.js';
  import { formatDayLabel, formatClock } from '../lib/format.js';
  import MediaToolbar from './MediaToolbar.svelte';
  import MediaGrid from './MediaGrid.svelte';
  import Button from './ui/Button.svelte';

  let { cameraId, cameraName } = $props();

  let loading = $state(false);
  let loadError = $state('');
  let items = $state([]);
  let updatedAt = $state(null);
  let selected = $state(new Set());
  let onlyOnCamera = $state(false);
  let kind = $state('all');
  let sortBy = $state('date');
  let sortAsc = $state(false);
  let loadedFor = $state(null);

  let syncEntry = $derived(getSyncQueue().find(e => e.cameraId === cameraId));
  let device = $derived(findDeviceById(cameraId));

  // Session state while armed: live link, connecting, or out of range
  let sessionState = $derived.by(() => {
    if (!device?.previewEnabled) return null;
    const op = syncEntry?.currentOperation || '';
    if (op.startsWith('Preview session') || op.startsWith('Fetching') || op.startsWith('Converting') || op === 'Preview ready') return 'live';
    if (device.isSyncing || syncEntry) return 'connecting';
    if (!device.isReachable) return 'waiting';
    return 'connecting';
  });

  async function toggleSession() {
    try {
      await setPreviewSession(cameraId, !device?.previewEnabled);
    } catch (e) {
      addToast(e.message || 'Preview session toggle failed', 'error');
    }
  }

  // The session only matters while this camera's library is on screen:
  // navigating away disarms. Deliberately NOT an $effect: the cameraId
  // prop chains to a derived that changes identity on every device
  // stream update, so an effect's teardown fired (and disarmed) every
  // few hundred ms. The component is keyed by camera in VideoLibrary,
  // so one instance = one camera; the initial value IS the value.
  // svelte-ignore state_referenced_locally
  const sessionCameraId = cameraId;
  onDestroy(() => {
    setPreviewSession(sessionCameraId, false).catch(() => {});
  });

  let normalized = $derived(items.map(i => fromCatalogItem(i, cameraId, cameraName)));
  let onCameraCount = $derived(normalized.filter(i => !i.downloaded).length);
  let shown = $derived(sortItems(
    filterKind(onlyOnCamera ? normalized.filter(i => !i.downloaded) : normalized, kind), sortBy, sortAsc));
  let selectableShown = $derived(shown.filter(i => !i.downloaded));

  $effect(() => {
    if (cameraId && loadedFor !== cameraId) {
      loadedFor = cameraId;
      selected = new Set();
      load();
    }
  });

  // Reload when a sync for this camera finishes: catalog and states changed.
  let hadSync = $state(false);
  $effect(() => {
    const syncing = !!syncEntry;
    if (hadSync && !syncing) load();
    hadSync = syncing;
  });

  // Preview flow: the session streams "Preview ready" when an LRV lands;
  // reload so the item gains its previewPath, then auto-open the one the
  // user asked for.
  let pendingPreview = $state(null);
  let lastOp = $state('');
  $effect(() => {
    const op = syncEntry?.currentOperation || '';
    if (op === lastOp) return;
    lastOp = op;
    if (op === 'Preview ready') load();
    if (op === 'Preview failed' && pendingPreview) {
      pendingPreview = null;
      addToast('Preview failed, see the diagnostics log', 'error');
    }
  });
  $effect(() => {
    if (!pendingPreview) return;
    const item = items.find(i => i.cameraPath === pendingPreview);
    if (item?.previewPath) {
      pendingPreview = null;
      openPlayer(item.previewPath, item.name, item.downloaded ? item.localPath : '', item.sizeBytes);
    }
  });

  async function load() {
    loading = true;
    loadError = '';
    try {
      const result = await fetchCameraMedia(cameraId);
      items = result.items;
      updatedAt = result.updatedAt;
    } catch (e) {
      loadError = e.message || 'Failed to load catalog';
      items = [];
    } finally {
      loading = false;
    }
  }

  function toggle(item) {
    const next = new Set(selected);
    if (next.has(item.key)) next.delete(item.key);
    else next.add(item.key);
    selected = next;
  }

  function selectShown() {
    selected = new Set(selectableShown.map(i => i.key));
  }

  async function downloadSelected() {
    const paths = [...selected];
    try {
      await requestMediaDownload(cameraId, paths);
      addToast(`${paths.length} files queued from ${cameraName}`, 'success');
      addLog(`Queued ${paths.length} files from ${cameraName}`, 'info');
      selected = new Set();
    } catch (e) {
      addToast('Failed to queue download', 'error');
      addLog(`Queue download failed: ${e.message}`, 'error');
    }
  }

  // Camera files are HEVC, which the renderer cannot decode; playback
  // always goes through the transcoded WebM proxy.
  async function play(item) {
    if (item.previewPath) {
      return openPlayer(item.previewPath, item.name, item.downloaded ? item.localPath : '', item.sizeBytes);
    }
    try {
      await previewMedia(cameraId, item.cameraPath);
      pendingPreview = item.cameraPath;
      addToast(item.downloaded ? 'Generating preview...' : 'Fetching preview from camera...', 'info');
    } catch (e) {
      addToast(e.message || 'Preview failed', 'error');
    }
  }

  async function refreshFromCamera() {
    try {
      await requestMediaDownload(cameraId, []);
      addToast(`Catalog refresh queued for ${cameraName}`, 'info');
    } catch (e) {
      addToast('Failed to queue refresh', 'error');
    }
  }
</script>

<div class="browser">
  <MediaToolbar bind:kind bind:sortBy bind:sortAsc>
    {#snippet leading()}
      <button class="chip" class:active={onlyOnCamera} disabled={onCameraCount === 0}
              onclick={() => onlyOnCamera = !onlyOnCamera}>
        <span class="chip-dot"></span> Not downloaded &middot; {onCameraCount}
      </button>
      {#if updatedAt}
        <span class="muted">catalog from {formatDayLabel(updatedAt).toLowerCase()} {formatClock(updatedAt)}</span>
      {/if}
    {/snippet}
    {#snippet trailing()}
      <Button icon="fa-satellite-dish" size="sm" onclick={toggleSession}
              title={device?.previewEnabled ? 'Close the camera link' : 'Keep a camera link up for instant previews'}>
        {#if sessionState === 'live'}Live
        {:else if sessionState === 'connecting'}Connecting...
        {:else if sessionState === 'waiting'}Waiting for camera
        {:else}Camera link{/if}
      </Button>
      {#if syncEntry}
        <span class="syncing-badge">
          <i class="fas fa-spinner fa-spin" aria-hidden="true"></i> {syncEntry.currentOperation || 'Syncing...'}
        </span>
      {:else}
        <Button icon="fa-rotate" size="sm" onclick={refreshFromCamera} title="Reconnect to the camera and refresh this catalog">Refresh</Button>
      {/if}
      <Button size="sm" onclick={selectShown} disabled={selectableShown.length === 0 || !!syncEntry}
              title="Select every shown file that is not downloaded yet">Select shown</Button>
      <Button variant="primary" size="sm" icon="fa-download" onclick={downloadSelected} disabled={selected.size === 0 || !!syncEntry}>
        Download{selected.size > 0 ? ` (${selected.size})` : ''}
      </Button>
    {/snippet}
  </MediaToolbar>

  {#if loading && items.length === 0}
    <div class="empty-state"><i class="fas fa-spinner fa-spin"></i><p>Loading catalog...</p></div>
  {:else if loadError}
    <div class="empty-state"><i class="fas fa-exclamation-triangle"></i><p>{loadError}</p></div>
  {:else if items.length === 0}
    <div class="empty-state">
      <i class="fas fa-camera"></i>
      <p>No catalog for this camera yet</p>
      <p class="hint">Sync it once, or press Refresh.</p>
    </div>
  {:else}
    <div class="scroll">
      <MediaGrid items={shown} {sortBy} selectedKeys={selected} pendingKey={pendingPreview}
                 onToggle={toggle} onPlay={play} />
    </div>
  {/if}
</div>

<style>
  .browser {
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
  }

  .syncing-badge {
    font-size: 0.8rem;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  /* The browser is pinned to the panel height; the grid scrolls inside */
  .scroll {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-muted);
    gap: 6px;
    padding: 30px 0;
  }

  .hint { font-size: 0.8rem; }
</style>
