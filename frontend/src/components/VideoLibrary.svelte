<script>
  import VideoCard from './VideoCard.svelte';
  import CameraMediaBrowser from './CameraMediaBrowser.svelte';
  import { getVideos, getTotalCount, getLoading } from '../lib/stores/videos.svelte.js';
  import { getManagedDevices, getAllDevices } from '../lib/stores/devices.svelte.js';
  import { loadVideos } from '../lib/grpc/actions.js';

  let videos = $derived(getVideos());
  let totalCount = $derived(getTotalCount());
  let loading = $derived(getLoading());

  // 'local' or a camera ID: the media source being browsed
  let source = $state('local');

  let managedCameras = $derived.by(() =>
    Object.values(getManagedDevices())
      .filter(d => d.id)
      .map(d => ({ id: d.id, name: d.wifiSsid?.trim()?.substring(0, 12) || d.name || 'Unknown' }))
      .sort((a, b) => a.name.localeCompare(b.name))
  );

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
        <button class="refresh-btn" onclick={() => loadVideos()} disabled={loading}>
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
      <CameraMediaBrowser cameraId={sourceCamera.id} cameraName={sourceCamera.name} />
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
      <div class="cards-grid">
        {#each videos as video (video.id)}
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

  .cards-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 12px;
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
