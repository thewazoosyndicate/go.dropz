<script>
  import VideoCard from './VideoCard.svelte';
  import CameraMediaBrowser from './CameraMediaBrowser.svelte';
  import { getVideos, getTotalCount, getLoading } from '../lib/stores/videos.svelte.js';
  import { getManagedDevices, getAllDevices } from '../lib/stores/devices.svelte.js';
  import { loadVideos } from '../lib/grpc/actions.js';
  import { getLibraryTarget, clearLibraryTarget } from '../lib/stores/ui.svelte.js';
  import { getSyncQueue } from '../lib/stores/sync.svelte.js';

  let videos = $derived(getVideos());
  let totalCount = $derived(getTotalCount());
  let loading = $derived(getLoading());

  // 'local' or a camera ID: the media source being browsed
  let source = $state('local');

  let sortBy = $state('date');
  let sortAsc = $state(false);

  let sorted = $derived.by(() => {
    const list = [...videos];
    const dir = sortAsc ? 1 : -1;
    list.sort((a, b) => {
      if (sortBy === 'name') return dir * a.name.localeCompare(b.name);
      if (sortBy === 'size') return dir * (a.sizeBytes - b.sizeBytes);
      return dir * ((a.createdAt?.getTime() || 0) - (b.createdAt?.getTime() || 0));
    });
    return list;
  });

  // Reload when a sync leaves the queue: its downloads are on disk now.
  // Cheaper and quieter than watching the filesystem while a download is
  // still writing into it; the manual button covers out-of-band changes.
  let prevQueueSize = $state(0);
  $effect(() => {
    const size = getSyncQueue().length;
    if (size < prevQueueSize) loadVideos();
    prevQueueSize = size;
  });

  // Returning to the Local tab rescans, so it never shows a stale list
  $effect(() => {
    if (source === 'local') loadVideos();
  });

  let managedCameras = $derived.by(() =>
    Object.values(getManagedDevices())
      .filter(d => d.id)
      .map(d => ({ id: d.id, name: d.wifiSsid?.trim()?.substring(0, 12) || d.name || 'Unknown' }))
      .sort((a, b) => a.name.localeCompare(b.name))
  );

  $effect(() => {
    const target = getLibraryTarget();
    if (target) {
      source = target.source;
      clearLibraryTarget();
    }
  });

  // A camera leaving the managed pool drops the tab back to local
  $effect(() => {
    if (source !== 'local' && !managedCameras.some(c => c.id === source)) {
      source = 'local';
    }
  });

  let sourceCamera = $derived(managedCameras.find(c => c.id === source));

  // Build camera ID → display name map
  let cameraNames = $derived.by(() => {
    const names = {};
    const devices = getAllDevices();
    for (const d of Object.values(devices)) {
      if (d.id) {
        names[d.id] = d.wifiSsid?.trim()?.substring(0, 12) || d.name || 'Unknown';
      }
    }
    return names;
  });
</script>

<section class="panel">
  <div class="panel-header">
    <h2><i class="fas fa-photo-film"></i> Library</h2>
    <div class="header-actions">
      {#if source === 'local'}
        <span class="badge">{totalCount} files</span>
        <button class="refresh-btn" onclick={() => loadVideos()} disabled={loading} aria-label="Refresh library">
          <i class="fas fa-refresh" class:spinning={loading}></i>
        </button>
      {/if}
    </div>
  </div>
  <div class="source-bar">
    <button class="source-chip" class:active={source === 'local'} onclick={() => source = 'local'}>
      <i class="fas fa-hard-drive"></i> Local
    </button>
    {#each managedCameras as cam (cam.id)}
      <button class="source-chip" class:active={source === cam.id} onclick={() => source = cam.id}>
        <i class="fas fa-camera"></i> {cam.name}
      </button>
    {/each}
  </div>
  <div class="panel-content">
    {#if source !== 'local' && sourceCamera}
      <!-- Keyed so a camera switch destroys the old browser, firing its
           disarm-on-destroy for the previous camera's session -->
      {#key sourceCamera.id}
        <CameraMediaBrowser cameraId={sourceCamera.id} cameraName={sourceCamera.name} />
      {/key}
    {:else if loading && videos.length === 0}
      <div class="empty-state">
        <i class="fas fa-spinner fa-spin"></i>
        <p>Scanning files...</p>
      </div>
    {:else if videos.length === 0}
      <div class="empty-state">
        <img src="imgs/3_dropz.svg" alt="Dropz" class="empty-logo" />
        <p>No media files found</p>
        <p class="hint">Sync a camera to see files here</p>
      </div>
    {:else}
      <div class="toolbar">
        <select bind:value={sortBy} aria-label="Sort by">
          <option value="date">Date</option>
          <option value="name">Name</option>
          <option value="size">Size</option>
        </select>
        <button class="btn-outline" onclick={() => sortAsc = !sortAsc}
                title="Toggle sort direction" aria-label="Toggle sort direction">
          <i class="fas {sortAsc ? 'fa-arrow-up-short-wide' : 'fa-arrow-down-wide-short'}"></i>
        </button>
      </div>
      <div class="media-grid">
        {#each sorted as video (video.id)}
          <VideoCard {video} cameraName={cameraNames[video.cameraId] || 'Unknown'} />
        {/each}
      </div>
    {/if}
  </div>
</section>

<style>
  .panel {
    background-color: var(--panel-bg);
    border-radius: var(--border-radius);
    box-shadow: var(--shadow-sm);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    flex: 1;
    transition: background-color 0.3s ease;
  }

  .panel-header {
    padding: 12px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--border-color);
  }

  .panel-header h2 {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 1rem;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .badge {
    font-size: 0.8rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 3px 10px;
    border-radius: 12px;
  }

  .refresh-btn {
    background: none;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 4px 8px;
    cursor: pointer;
    color: var(--text-secondary);
    font-size: 0.8rem;
    transition: all 0.2s;
  }

  .refresh-btn:hover:not(:disabled) {
    border-color: var(--text-secondary);
    color: var(--text-primary);
  }

  .refresh-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .spinning {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .source-bar {
    display: flex;
    gap: 6px;
    padding: 8px 16px 0;
    flex-wrap: wrap;
  }

  .source-chip {
    background: none;
    border: 1px solid var(--border-color);
    border-radius: 14px;
    padding: 4px 12px;
    font-size: 0.82rem;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.15s;
  }

  .source-chip:hover { color: var(--text-primary); }

  .source-chip.active {
    background-color: var(--primary-color);
    border-color: var(--primary-color);
    color: white;
  }

  .panel-content {
    flex: 1;
    padding: 12px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding-bottom: 10px;
  }

  .toolbar select {
    background-color: var(--panel-bg);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 5px 8px;
    font-size: 0.85rem;
  }

  .btn-outline {
    background: none;
    border: 1px solid var(--border-color);
    color: var(--text-primary);
    border-radius: 6px;
    padding: 6px 12px;
    font-size: 0.85rem;
    cursor: pointer;
  }

  /* Same sizing as CameraMediaBrowser's grid so both tabs read the same */
  .media-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 10px;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-muted);
    text-align: center;
  }

  .empty-logo {
    height: 80px;
    opacity: 0.6;
    margin-bottom: 12px;
  }

  .hint {
    font-size: 0.8rem;
    margin-top: 4px;
  }
</style>
