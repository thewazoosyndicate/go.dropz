<script>
  import VideoCard from './VideoCard.svelte';
  import CameraMediaBrowser from './CameraMediaBrowser.svelte';
  import IconButton from './ui/IconButton.svelte';
  import { getVideos, getTotalCount, getLoading } from '../lib/stores/videos.svelte.js';
  import { getManagedDevices, getAllDevices, displayName } from '../lib/stores/devices.svelte.js';
  import { loadVideos } from '../lib/grpc/actions.js';
  import { getLibraryTarget, clearLibraryTarget, getLibraryLastVisit } from '../lib/stores/ui.svelte.js';
  import { getSyncQueue, getNewFilePaths, getSessionById } from '../lib/stores/sync.svelte.js';
  import { getAppConfig } from '../lib/stores/config.svelte.js';
  import { formatBytes, formatDayLabel, formatClock, dayKey, plural } from '../lib/format.js';

  const { ipcRenderer } = window.require('electron');

  let videos = $derived(getVideos());
  let totalCount = $derived(getTotalCount());
  let loading = $derived(getLoading());

  // 'local' or a camera ID: the media source being browsed
  let source = $state('local');
  // 'all' | 'new' | { sessionId }
  let filter = $state('all');
  let kind = $state('all'); // all | video | photo
  let sortBy = $state('date');
  let sortAsc = $state(false);

  let newPaths = $derived(getNewFilePaths(getLibraryLastVisit()));
  let session = $derived(filter?.sessionId ? getSessionById(filter.sessionId) : null);
  let sessionPaths = $derived(new Set((session?.files || []).filter(f => f.localPath).map(f => f.localPath)));

  let filtered = $derived(videos.filter(v => {
    if (kind === 'video' && !v.mimeType?.startsWith('video/')) return false;
    if (kind === 'photo' && !v.mimeType?.startsWith('image/')) return false;
    if (filter === 'new') return newPaths.has(v.path);
    if (filter?.sessionId) return sessionPaths.has(v.path);
    return true;
  }));

  let sorted = $derived.by(() => {
    const list = [...filtered];
    const dir = sortAsc ? 1 : -1;
    list.sort((a, b) => {
      if (sortBy === 'name') return dir * a.name.localeCompare(b.name);
      if (sortBy === 'size') return dir * (a.sizeBytes - b.sizeBytes);
      return dir * ((a.createdAt?.getTime() || 0) - (b.createdAt?.getTime() || 0));
    });
    return list;
  });

  // Day groups only make sense in date order; other sorts stay flat
  let groups = $derived.by(() => {
    if (sortBy !== 'date') return [{ key: 'flat', label: '', videos: sorted }];
    const out = [];
    const byKey = new Map();
    for (const v of sorted) {
      const key = dayKey(v.createdAt);
      let g = byKey.get(key);
      if (!g) {
        g = { key, label: formatDayLabel(v.createdAt), videos: [] };
        byKey.set(key, g);
        out.push(g);
      }
      g.videos.push(v);
    }
    return out.map(g => {
      const bytes = g.videos.reduce((n, v) => n + (v.sizeBytes || 0), 0);
      const cams = new Set(g.videos.map(v => v.cameraId));
      const from = cams.size === 1 ? cameraNames[g.videos[0].cameraId] : '';
      g.meta = [plural(g.videos.length, 'file'), formatBytes(bytes), from ? `from ${from}` : ''].filter(Boolean).join(' · ');
      return g;
    });
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

  let cameraNames = $derived.by(() => {
    const names = {};
    for (const d of Object.values(getAllDevices())) {
      if (d.id) names[d.id] = displayName(d);
    }
    return names;
  });

  let sessionLabel = $derived(session
    ? `From sync ${formatDayLabel(session.finishedAt).toLowerCase()} ${formatClock(session.finishedAt)} · ${plural(session.filesDownloaded, 'file')}`
    : 'From a sync');

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
      <div class="toolbar">
        <div class="chips">
          {#if filter?.sessionId}
            <button class="chip active" onclick={() => filter = 'all'} title="Clear">
              {sessionLabel} <i class="fas fa-times" aria-hidden="true"></i>
            </button>
          {:else}
            <button class="chip" class:active={filter === 'new'} disabled={newPaths.size === 0}
                    onclick={() => filter = filter === 'new' ? 'all' : 'new'}>
              <span class="chip-dot"></span> New since last visit &middot; {newPaths.size}
            </button>
          {/if}
          <button class="chip" class:active={kind === 'video'} onclick={() => kind = kind === 'video' ? 'all' : 'video'}>Videos</button>
          <button class="chip" class:active={kind === 'photo'} onclick={() => kind = kind === 'photo' ? 'all' : 'photo'}>Photos</button>
        </div>
        <div class="sort">
          <select bind:value={sortBy} aria-label="Sort by">
            <option value="date">Date</option>
            <option value="name">Name</option>
            <option value="size">Size</option>
          </select>
          <IconButton icon={sortAsc ? 'fa-arrow-up-short-wide' : 'fa-arrow-down-wide-short'}
                      title="Toggle sort direction" onclick={() => sortAsc = !sortAsc} />
        </div>
      </div>
      {#if sorted.length === 0}
        <div class="empty-state">
          <p>Nothing matches this filter.</p>
        </div>
      {:else}
        {#each groups as group (group.key)}
          <div class="group">
            {#if group.label}
              <div class="group-head">
                <span class="group-label">{group.label}</span>
                <span class="group-meta">{group.meta}</span>
              </div>
            {/if}
            <div class="media-grid">
              {#each group.videos as video (video.id)}
                <VideoCard {video} cameraName={cameraNames[video.cameraId] || 'Unknown'} isNew={newPaths.has(video.path)} />
              {/each}
            </div>
          </div>
        {/each}
      {/if}
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

  .source-chip, .chip {
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

  .source-chip:hover, .chip:hover:not(:disabled) { color: var(--text-primary); }
  .source-chip.active, .chip.active {
    background-color: var(--primary-color);
    border-color: var(--primary-color);
    color: white;
  }
  .chip:disabled { opacity: 0.5; cursor: default; }
  .chip-dot { width: 7px; height: 7px; border-radius: 50%; background: currentColor; }

  .panel-content {
    flex: 1;
    padding: 12px 16px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    flex-wrap: wrap;
  }

  .chips { display: flex; gap: 6px; flex-wrap: wrap; }
  .sort { display: flex; align-items: center; gap: 8px; }

  .toolbar select {
    background-color: var(--panel-bg);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 5px 8px;
    height: 32px;
    font-size: 0.85rem;
  }

  .group { display: flex; flex-direction: column; gap: 8px; }
  .group-head { display: flex; align-items: baseline; gap: 10px; }
  .group-label { font-size: 0.85rem; font-weight: 600; }
  .group-meta { font-size: 0.75rem; color: var(--text-muted); }

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
    min-height: 120px;
    color: var(--text-muted);
    text-align: center;
  }

  .empty-logo { height: 80px; opacity: 0.6; margin-bottom: 12px; }
  .hint { font-size: 0.8rem; margin-top: 4px; }
</style>
