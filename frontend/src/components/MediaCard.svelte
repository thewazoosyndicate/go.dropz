<script>
  // One card for a library file and a catalog entry. A downloaded item
  // plays and opens its folder; an item still on the camera selects for
  // download and can fetch its low-res preview.
  // shell.openPath / showItemInFolder can block the UI on Linux;
  // main runs them detached instead (desktop-open / desktop-show)
  const { ipcRenderer } = window.require('electron');
  import { formatBytes } from '../lib/format.js';

  let { item, isNew = false, selected = false, pending = false, onToggle, onPlay } = $props();

  let selectable = $derived(!item.downloaded);
  let timeText = $derived(item.createdAt ? item.createdAt.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) : '');
  let metaText = $derived([item.cameraName, formatBytes(item.sizeBytes), timeText].filter(Boolean).join(' · '));

  function openFile() {
    if (item.localPath) ipcRenderer.send('desktop-open', item.localPath);
  }

  function showInFolder() {
    if (item.localPath) ipcRenderer.send('desktop-show', item.localPath);
  }

</script>

{#snippet thumbContent()}
  {#if item.thumbnailPath}
    <img src={'file://' + item.thumbnailPath} alt={item.name} loading="lazy" />
  {:else}
    <i class="fas {item.isImage ? 'fa-image' : 'fa-film'}" aria-hidden="true"></i>
  {/if}
  {#if isNew}
    <span class="badge new">New</span>
  {:else if !item.downloaded}
    <span class="badge on-camera">On camera</span>
  {/if}
  {#if selected}
    <span class="check" aria-hidden="true"><i class="fas fa-check"></i></span>
  {:else if item.cameraPath && item.downloaded}
    <span class="check in-library" title="In your library" aria-hidden="true"><i class="fas fa-check"></i></span>
  {/if}
{/snippet}

<div class="media-card" class:selected class:on-camera={!item.downloaded}>
  {#if selectable}
    <!-- Clicking the picture selects: the fast path for picking many clips -->
    <button class="thumb selectable" onclick={() => onToggle?.(item)} aria-pressed={selected}
            title={selected ? 'Selected for download' : 'Select for download'}>
      {@render thumbContent()}
    </button>
  {:else}
    <div class="thumb">{@render thumbContent()}</div>
  {/if}
  <div class="media-info">
    <span class="media-name" title={item.name}>{item.name}</span>
    <span class="media-meta" title={metaText}>{metaText}</span>
  </div>
  <div class="card-actions">
    {#if item.downloaded}
      {#if item.isImage}
        <button class="mini-btn primary" onclick={openFile}>
          <i class="fas fa-image" aria-hidden="true"></i> Open
        </button>
      {:else}
        <button class="mini-btn primary" onclick={() => onPlay?.(item)} disabled={pending}
                title={item.previewPath ? 'Play preview' : 'Generate and play preview'}>
          <i class="fas {pending ? 'fa-spinner fa-spin' : 'fa-play'}" aria-hidden="true"></i>
          {pending ? 'Preparing...' : 'Play'}
        </button>
      {/if}
      <button class="mini-btn" onclick={showInFolder} title="Show in folder" aria-label="Show in folder">
        <i class="fas fa-folder-open" aria-hidden="true"></i>
      </button>
    {:else}
      <button class="mini-btn" class:primary={selected} onclick={() => onToggle?.(item)} aria-pressed={selected}>
        <i class="fas {selected ? 'fa-check' : 'fa-download'}" aria-hidden="true"></i>
        {selected ? 'Selected' : 'Select'}
      </button>
      {#if item.canPreview}
        <button class="mini-btn" onclick={() => onPlay?.(item)} disabled={pending}
                title={item.previewPath ? 'Play preview' : 'Preview from camera'} aria-label="Preview">
          <i class="fas {pending ? 'fa-spinner fa-spin' : 'fa-play'}" aria-hidden="true"></i>
        </button>
      {/if}
    {/if}
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
  .media-card.selected { border-color: var(--primary-color); }

  .thumb {
    position: relative;
    aspect-ratio: 4 / 3;
    width: 100%;
    background-color: var(--light-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
    font-size: 1.6rem;
    border: none;
    padding: 0;
  }

  .thumb.selectable { cursor: pointer; }
  .thumb.selectable:hover::after {
    content: '';
    position: absolute;
    inset: 0;
    background: rgba(52, 152, 219, 0.12);
  }
  .thumb.selectable:focus-visible { outline-offset: -3px; }

  .thumb img { width: 100%; height: 100%; object-fit: cover; }

  .badge {
    position: absolute;
    top: 6px;
    left: 6px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.02em;
    color: white;
    border-radius: 8px;
    padding: 0 6px;
    line-height: 16px;
  }
  .badge.new { background: var(--primary-color); }
  .badge.on-camera { background: rgba(0, 0, 0, 0.55); }

  .check {
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
    background-color: var(--primary-color);
  }
  .check.in-library { background-color: var(--state-ok); }

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
    justify-content: center;
    gap: 5px;
  }

  .mini-btn:first-child { flex: 1; }
  .mini-btn:disabled { opacity: 0.6; cursor: default; }

  .mini-btn.primary {
    background-color: var(--primary-color);
    border-color: var(--primary-color);
    color: white;
  }
</style>
