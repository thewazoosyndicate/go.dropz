<script>
  import CameraMediaBrowser from './CameraMediaBrowser.svelte';
  import MediaToolbar from './MediaToolbar.svelte';
  import MediaGrid from './MediaGrid.svelte';
  import IconButton from './ui/IconButton.svelte';
  import { getVideos, getTotalCount, getLoading } from '../lib/stores/videos.svelte.js';
  import { getManagedDevices, getAllDevices, displayName } from '../lib/stores/devices.svelte.js';
  import { loadVideos, previewVideo } from '../lib/grpc/actions.js';
  import { getLibraryTarget, clearLibraryTarget, getLibraryLastVisit, addToast, openPlayer } from '../lib/stores/ui.svelte.js';
  import { getSyncQueue, getNewFilePaths, getSessionById } from '../lib/stores/sync.svelte.js';
  import { getAppConfig } from '../lib/stores/config.svelte.js';
  import { fromLibraryVideo, filterKind, sortItems } from '../lib/media.js';
  import { formatDayLabel, formatClock, plural } from '../lib/format.js';

  const { ipcRenderer } = window.require('electron');

  let videos = $derived(getVideos());
  let totalCount = $derived(getTotalCount());
  let loading = $derived(getLoading());

  // 'local' or a camera ID: the media source being browsed
  let source = $state('local');
  // 'all' | 'new' | { sessionId }
  let filter = $state('all');
  let kind = $state('all');
  let sortBy = $state('date');
  let sortAsc = $state(false);

  let cameraNames = $derived.by(() => {
    const names = {};
    for (const d of Object.values(getAllDevices())) {
      if (d.id) names[d.id] = displayName(d);
    }
    return names;
  });

  let newPaths = $derived(getNewFilePaths(getLibraryLastVisit()));
  let session = $derived(filter?.sessionId ? getSessionById(filter.sessionId) : null);
  let sessionPaths = $derived(new Set((session?.files || []).filter(f => f.localPath).map(f => f.localPath)));

  let normalized = $derived(videos.map(v => fromLibraryVideo(v, cameraNames[v.cameraId] || 'Unknown')));
  let newKeys = $derived(new Set(normalized.filter(i => newPaths.has(i.localPath)).map(i => i.key)));
  let shown = $derived(sortItems(filterKind(normalized.filter(i => {
    if (filter === 'new') return newPaths.has(i.localPath);
    if (filter?.sessionId) return sessionPaths.has(i.localPath);
    return true;
  }), kind), sortBy, sortAsc));

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
      .map(d => ({ id: d.id, name: displayName(d) }))
      .sort((a, b) => a.name.localeCompare(b.name))
  );

  $effect(() => {
    const target = getLibraryTarget();
    if (target) {
      source = target.source;
      filter = target.filter || 'all';
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

  let sessionLabel = $derived(session
    ? `From sync ${formatDayLabel(session.finishedAt).toLowerCase()} ${formatClock(session.finishedAt)} · ${plural(session.filesDownloaded, 'file')}`
    : 'From a sync');

  // Full-res camera files are HEVC, which the renderer cannot decode;
  // playback goes through the transcoded WebM preview.
  let pendingKey = $state(null);
  async function play(item) {
    if (item.previewPath) return openPlayer(item.previewPath, item.name, item.localPath, item.sizeBytes);
    pendingKey = item.key;
    try {
      const path = await previewVideo(item.localPath);
      openPlayer(path, item.name, item.localPath, item.sizeBytes);
      loadVideos();
    } catch (e) {
      addToast(e.message || 'Preview generation failed', 'error');
    } finally {
      pendingKey = null;
    }
  }

  function openFolder() {
    const folder = getAppConfig()?.destinationFolder;
    if (folder) ipcRenderer.send('desktop-open', folder);
  }
</script>

<section class="panel">
  <div class="panel-header">
    <h2><i class="fas fa-photo-film" aria-hidden="true"></i> Library</h2>
    <div class="header-actions">
      {#if source === 'local'}
        <span class="badge">{plural(totalCount, 'file')}</span>
        <IconButton icon="fa-folder-open" title="Open the library folder" onclick={openFolder} />
        <IconButton icon="fa-refresh" title="Rescan the library" spin={loading} disabled={loading} onclick={() => loadVideos()} />
      {/if}
    </div>
  </div>
  <div class="source-bar">
    <button class="source-chip" class:active={source === 'local'} onclick={() => source = 'local'}>
      <i class="fas fa-hard-drive" aria-hidden="true"></i> Local
    </button>
    {#each managedCameras as cam (cam.id)}
      <button class="source-chip" class:active={source === cam.id} onclick={() => source = cam.id}>
        <i class="fas fa-camera" aria-hidden="true"></i> {cam.name}
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
        <p>No media files yet</p>
        <p class="hint">Sync a camera and its clips land here.</p>
      </div>
    {:else}
      <MediaToolbar bind:kind bind:sortBy bind:sortAsc>
        {#snippet leading()}
          {#if filter?.sessionId}
            <button class="chip active" onclick={() => filter = 'all'} title="Clear">
              {sessionLabel} <i class="fas fa-times" aria-hidden="true"></i>
            </button>
          {:else}
            <button class="chip" class:active={filter === 'new'} disabled={newKeys.size === 0}
                    onclick={() => filter = filter === 'new' ? 'all' : 'new'}>
              <span class="chip-dot"></span> New since last visit &middot; {newKeys.size}
            </button>
          {/if}
        {/snippet}
      </MediaToolbar>
      <div class="scroll">
        <MediaGrid items={shown} {sortBy} {newKeys} {pendingKey} onPlay={play} />
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

  .header-actions { display: flex; align-items: center; gap: 8px; }

  .badge {
    font-size: 0.8rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 3px 10px;
    border-radius: 12px;
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
    min-height: 30px;
    font-size: 0.82rem;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: color 0.15s, background-color 0.15s;
  }

  .source-chip:hover { color: var(--text-primary); }
  .source-chip.active {
    background-color: var(--primary-color);
    border-color: var(--primary-color);
    color: white;
  }

  .panel-content {
    flex: 1;
    min-height: 0;
    padding: 12px 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

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
    height: 100%;
    min-height: 120px;
    color: var(--text-muted);
    text-align: center;
  }

  .empty-logo { height: 80px; opacity: 0.6; margin-bottom: 12px; }
  .hint { font-size: 0.8rem; margin-top: 4px; }
</style>
