<script>
  // shell.openPath / showItemInFolder can block the UI on Linux;
  // main runs them detached instead (desktop-open / desktop-show)
  const { ipcRenderer } = window.require('electron');

  let { video, cameraName } = $props();

  let icon = $derived(video.mimeType?.startsWith('image/') ? 'fa-image' : 'fa-film');

  let sizeText = $derived(formatSize(video.sizeBytes));
  let dateText = $derived(video.createdAt ? video.createdAt.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) : '');
  let metaText = $derived([cameraName, sizeText, dateText].filter(Boolean).join(' · '));

  function formatSize(bytes) {
    if (!bytes) return '';
    if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
    if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(1)} MB`;
    return `${Math.round(bytes / 1024)} KB`;
  }

  function openFile() {
    ipcRenderer.send('desktop-open', video.path);
  }

  function showInFolder() {
    ipcRenderer.send('desktop-show', video.path);
  }
</script>

<!-- Mirrors CameraMediaBrowser's media-card so both library tabs read the same -->
<div class="media-card">
  <div class="thumb">
    {#if video.thumbnailPath}
      <img src={'file://' + video.thumbnailPath} alt={video.name} loading="lazy" />
    {:else}
      <i class="fas {icon}"></i>
    {/if}
  </div>
  <div class="media-info">
    <span class="media-name" title={video.name}>{video.name}</span>
    <span class="media-meta">{metaText}</span>
  </div>
  <div class="card-actions">
    <button class="mini-btn primary" onclick={openFile}>
      <i class="fas fa-play"></i> Open
    </button>
    <button class="mini-btn" onclick={showInFolder} title="Show in folder" aria-label="Show in folder">
      <i class="fas fa-folder-open"></i>
    </button>
  </div>
</div>

<style>
  .media-card {
    background-color: var(--panel-bg);
    border: 2px solid var(--border-color);
    border-radius: 8px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    transition: border-color 0.15s;
  }

  .media-card:hover {
    border-color: var(--text-secondary);
  }

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
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .card-actions {
    display: flex;
    gap: 6px;
    padding: 0 8px 8px;
  }

  .mini-btn {
    border: 1px solid var(--border-color);
    background: none;
    color: var(--text-primary);
    border-radius: 6px;
    padding: 3px 10px;
    font-size: 0.75rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .mini-btn.primary {
    background-color: var(--primary-color);
    border-color: var(--primary-color);
    color: white;
    flex: 1;
    justify-content: center;
  }
</style>
