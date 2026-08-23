<script>
  import { fetchCameraMedia, requestMediaDownload } from '../lib/grpc/actions.js';
  import { addToast, addLog } from '../lib/stores/ui.svelte.js';
  import { getSyncQueue } from '../lib/stores/sync.svelte.js';

  let { cameraId, cameraName } = $props();

  let loading = $state(false);
  let loadError = $state('');
  let items = $state([]);
  let updatedAt = $state(null);
  let selected = $state({});
  let sortBy = $state('date');
  let sortAsc = $state(false);
  let loadedFor = $state(null);

  let syncEntry = $derived(getSyncQueue().find(e => e.cameraId === cameraId));
  let selectedNames = $derived(Object.keys(selected).filter(n => selected[n]));

  let sorted = $derived.by(() => {
    const list = [...items];
    const dir = sortAsc ? 1 : -1;
    list.sort((a, b) => {
      if (sortBy === 'name') return dir * a.name.localeCompare(b.name);
      if (sortBy === 'size') return dir * (a.sizeBytes - b.sizeBytes);
      return dir * ((a.createdAt?.getTime() || 0) - (b.createdAt?.getTime() || 0));
    });
    return list;
  });

  $effect(() => {
    if (cameraId && loadedFor !== cameraId) {
      loadedFor = cameraId;
      selected = {};
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

  function toggle(name) {
    selected[name] = !selected[name];
    selected = { ...selected };
  }

  async function downloadSelected() {
    try {
      await requestMediaDownload(cameraId, selectedNames);
      addToast(`${selectedNames.length} files queued from ${cameraName}`, 'success');
      addLog(`Queued ${selectedNames.length} files from ${cameraName}`, 'info');
      selected = {};
    } catch (e) {
      addToast('Failed to queue download', 'error');
      addLog(`Queue download failed: ${e.message}`, 'error');
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

  function formatSize(bytes) {
    if (!bytes) return '';
    if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
    if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(1)} MB`;
    return `${Math.round(bytes / 1024)} KB`;
  }

  function formatDate(d) {
    return d ? d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) : '';
  }
</script>

<div class="browser">
  <div class="toolbar">
    <div class="toolbar-left">
      <select bind:value={sortBy} aria-label="Sort by">
        <option value="date">Date</option>
        <option value="name">Name</option>
        <option value="size">Size</option>
      </select>
      <button class="btn btn-outline" onclick={() => sortAsc = !sortAsc}
              title="Toggle sort direction" aria-label="Toggle sort direction">
        <i class="fas {sortAsc ? 'fa-arrow-up-short-wide' : 'fa-arrow-down-wide-short'}"></i>
      </button>
      {#if updatedAt}
        <span class="catalog-age">catalog from {formatDate(updatedAt)}</span>
      {/if}
    </div>
    <div class="toolbar-right">
      {#if syncEntry}
        <span class="syncing-badge">
          <i class="fas fa-spinner fa-spin"></i> {syncEntry.currentOperation || 'Syncing...'}
        </span>
      {:else}
        <button class="btn btn-outline" onclick={refreshFromCamera} title="Reconnect to the camera and refresh this catalog">
          <i class="fas fa-rotate"></i> Refresh from camera
        </button>
      {/if}
      <button class="btn btn-primary" onclick={downloadSelected} disabled={selectedNames.length === 0 || !!syncEntry}>
        <i class="fas fa-download"></i> Download {selectedNames.length > 0 ? `(${selectedNames.length})` : ''}
      </button>
    </div>
  </div>

  {#if loading && items.length === 0}
    <div class="empty-state"><i class="fas fa-spinner fa-spin"></i><p>Loading catalog...</p></div>
  {:else if loadError}
    <div class="empty-state"><i class="fas fa-exclamation-triangle"></i><p>{loadError}</p></div>
  {:else if items.length === 0}
    <div class="empty-state">
      <i class="fas fa-camera"></i>
      <p>No catalog for this camera yet</p>
      <p class="hint">Sync it once, or use "Refresh from camera"</p>
    </div>
  {:else}
    <div class="media-grid">
      {#each sorted as item (item.name)}
        <button class="media-card" class:selected={selected[item.name]} class:downloaded={item.downloaded}
             onclick={() => !item.downloaded && toggle(item.name)}
             title={item.downloaded ? 'Already downloaded' : 'Select for download'}>
          <div class="thumb">
            {#if item.thumbnailPath}
              <img src={'file://' + item.thumbnailPath} alt={item.name} loading="lazy" />
            {:else}
              <i class="fas {item.name.toLowerCase().endsWith('.jpg') ? 'fa-image' : 'fa-film'}"></i>
            {/if}
            {#if item.downloaded}
              <span class="state-badge downloaded-badge"><i class="fas fa-check"></i></span>
            {:else if selected[item.name]}
              <span class="state-badge selected-badge"><i class="fas fa-check"></i></span>
            {/if}
          </div>
          <div class="media-info">
            <span class="media-name" title={item.name}>{item.name}</span>
            <span class="media-meta">{formatSize(item.sizeBytes)}{item.sizeBytes && item.createdAt ? ' · ' : ''}{formatDate(item.createdAt)}</span>
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .browser {
    display: flex;
    flex-direction: column;
    gap: 10px;
    height: 100%;
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .toolbar-left, .toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .toolbar select {
    background-color: var(--panel-bg);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 5px 8px;
    font-size: 0.85rem;
  }

  .catalog-age {
    font-size: 0.78rem;
    color: var(--text-muted);
  }

  .syncing-badge {
    font-size: 0.85rem;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .btn {
    padding: 6px 12px;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .btn:disabled { opacity: 0.5; cursor: default; }

  .btn-primary {
    background-color: var(--primary-color);
    color: white;
  }

  .btn-outline {
    background: none;
    border: 1px solid var(--border-color);
    color: var(--text-primary);
  }

  .media-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 10px;
    overflow-y: auto;
  }

  .media-card {
    background-color: var(--panel-bg);
    border: 2px solid var(--border-color);
    border-radius: 8px;
    overflow: hidden;
    cursor: pointer;
    padding: 0;
    text-align: left;
    display: flex;
    flex-direction: column;
    transition: border-color 0.15s;
  }

  .media-card.selected { border-color: var(--primary-color); }
  .media-card.downloaded { cursor: default; opacity: 0.85; }

  .thumb {
    position: relative;
    aspect-ratio: 4 / 3;
    background-color: var(--light-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
    font-size: 1.6rem;
  }

  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .state-badge {
    position: absolute;
    top: 6px;
    right: 6px;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    color: white;
  }

  .downloaded-badge { background-color: var(--secondary-color); }
  .selected-badge { background-color: var(--primary-color); }

  .media-info {
    display: flex;
    flex-direction: column;
    padding: 6px 8px;
  }

  .media-name {
    font-size: 0.82rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .media-meta {
    font-size: 0.72rem;
    color: var(--text-muted);
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
