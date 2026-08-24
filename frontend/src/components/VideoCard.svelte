<script>
  // shell.openPath / showItemInFolder can block the UI on Linux;
  // main runs them detached instead (desktop-open / desktop-show)
  const { ipcRenderer } = window.require('electron');
  import { previewVideo } from '../lib/grpc/actions.js';
  import { addToast, openPlayer } from '../lib/stores/ui.svelte.js';
  import { formatBytes } from '../lib/format.js';

  let { video, cameraName, isNew = false } = $props();

  let isImage = $derived(video.mimeType?.startsWith('image/'));
  let icon = $derived(isImage ? 'fa-image' : 'fa-film');
  // Local cache of a preview generated this session; the store refreshes
  // previewPath only on the next library reload.
  let generatedPath = $state(null);
  let generating = $state(false);
  let playablePath = $derived(generatedPath || video.previewPath);

  let timeText = $derived(video.createdAt ? video.createdAt.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) : '');
  let metaText = $derived([cameraName, formatBytes(video.sizeBytes), timeText].filter(Boolean).join(' · '));

  function openFile() {
    ipcRenderer.send('desktop-open', video.path);
  }

  function showInFolder() {
    ipcRenderer.send('desktop-show', video.path);
  }

  // Full-res camera files are HEVC, which the renderer cannot decode;
  // playback goes through the transcoded WebM preview.
  async function play() {
    if (playablePath) return openPlayer(playablePath, video.name);
    generating = true;
    try {
      generatedPath = await previewVideo(video.path);
      openPlayer(generatedPath, video.name);
    } catch (e) {
      addToast(e.message || 'Preview generation failed', 'error');
    } finally {
      generating = false;
    }
  }
</script>

<!-- Mirrors CameraMediaBrowser's media-card so both library tabs read the same -->
<div class="media-card" class:is-new={isNew}>
  <div class="thumb">
    {#if video.thumbnailPath}
      <img src={'file://' + video.thumbnailPath} alt={video.name} loading="lazy" />
    {:else}
      <i class="fas {icon}" aria-hidden="true"></i>
    {/if}
    {#if isNew}<span class="new-badge">New</span>{/if}
  </div>
  <div class="media-info">
    <span class="media-name" title={video.name}>{video.name}</span>
    <span class="media-meta" title={metaText}>{metaText}</span>
  </div>
  <div class="card-actions">
    {#if isImage}
      <button class="mini-btn primary" onclick={openFile}>
        <i class="fas fa-image" aria-hidden="true"></i> Open
      </button>
    {:else}
      <button class="mini-btn primary" onclick={play} disabled={generating}
              title={playablePath ? 'Play preview' : 'Generate and play preview'}>
        <i class="fas {generating ? 'fa-spinner fa-spin' : 'fa-play'}" aria-hidden="true"></i>
        {generating ? 'Preparing...' : 'Play'}
      </button>
    {/if}
    <button class="mini-btn" onclick={showInFolder} title="Show in folder" aria-label="Show in folder">
      <i class="fas fa-folder-open" aria-hidden="true"></i>
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

  .media-card:hover { border-color: var(--text-secondary); }

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

  .thumb img { width: 100%; height: 100%; object-fit: cover; }

  .new-badge {
    position: absolute;
    top: 6px;
    left: 6px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    color: white;
    background: var(--primary-color);
    border-radius: 8px;
    padding: 0 6px;
    line-height: 16px;
  }

  .media-info { display: flex; flex-direction: column; padding: 6px 8px; }

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

  .card-actions { display: flex; gap: 6px; padding: 0 8px 8px; }

  .mini-btn {
    border: 1px solid var(--border-color);
    background: none;
    color: var(--text-primary);
    border-radius: 6px;
    padding: 4px 10px;
    min-height: 30px;
    font-size: 0.75rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 5px;
  }

  .mini-btn:disabled { opacity: 0.6; cursor: default; }

  .mini-btn.primary {
    background-color: var(--primary-color);
    border-color: var(--primary-color);
    color: white;
    flex: 1;
    justify-content: center;
  }
</style>
