<script>
  // One card for a library file and a catalog entry: a thumbnail with
  // small overlay controls, then name and meta. A downloaded item plays
  // and opens its folder; an item still on the camera selects for
  // download (click the picture) and can fetch its low-res preview.
  // shell.openPath / showItemInFolder can block the UI on Linux;
  // main runs them detached instead (desktop-open / desktop-show)
  const { ipcRenderer } = window.require('electron');
  import { formatBytes } from '../lib/format.js';

  let { item, isNew = false, selected = false, pending = false, onToggle, onPlay } = $props();

  let selectable = $derived(!item.downloaded);
  let timeText = $derived(item.createdAt ? item.createdAt.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) : '');
  let metaText = $derived([item.cameraName, formatBytes(item.sizeBytes), timeText].filter(Boolean).join(' · '));
  let playTitle = $derived(
    item.previewPath ? 'Play preview' :
    item.downloaded ? 'Generate and play preview' : 'Preview from camera');

  function openFile() {
    if (item.localPath) ipcRenderer.send('desktop-open', item.localPath);
  }

  function showInFolder() {
    if (item.localPath) ipcRenderer.send('desktop-show', item.localPath);
  }
</script>

<div class="media-card" class:selected>
  <div class="thumb">
    {#if item.thumbnailPath}
      <img src={'file://' + item.thumbnailPath} alt={item.name} loading="lazy" />
    {:else}
      <i class="fas {item.isImage ? 'fa-image' : 'fa-film'} placeholder" aria-hidden="true"></i>
    {/if}

    {#if selectable}
      <!-- Clicking the picture selects: the fast path for picking many clips -->
      <button class="select-area" onclick={() => onToggle?.(item)} aria-pressed={selected}
              title={selected ? 'Selected for download' : 'Select for download'}>
        <span class="sr-only">{selected ? 'Selected' : 'Select'} {item.name}</span>
      </button>
    {/if}

    {#if isNew}
      <span class="badge new">New</span>
    {:else if selectable}
      <span class="badge on-camera">On camera</span>
    {/if}
    {#if selected}
      <span class="check" aria-hidden="true"><i class="fas fa-check"></i></span>
    {:else if item.cameraPath && item.downloaded}
      <span class="check in-library" title="In your library" aria-hidden="true"><i class="fas fa-check"></i></span>
    {/if}

    <div class="overlay">
      {#if item.downloaded && item.isImage}
        <button class="ctl" onclick={openFile} title="Open" aria-label="Open {item.name}">
          <i class="fas fa-image" aria-hidden="true"></i>
        </button>
      {:else if item.canPreview}
        <button class="ctl" class:busy={pending} onclick={() => onPlay?.(item)} disabled={pending}
                title={playTitle} aria-label="{playTitle}, {item.name}">
          <i class="fas {pending ? 'fa-spinner fa-spin' : 'fa-play'}" aria-hidden="true"></i>
        </button>
      {/if}
      {#if item.downloaded}
        <button class="ctl right" onclick={showInFolder} title="Show in folder" aria-label="Show {item.name} in folder">
          <i class="fas fa-folder-open" aria-hidden="true"></i>
        </button>
      {/if}
    </div>
  </div>
  <div class="media-info">
    <span class="media-name" title={item.name}>{item.name}</span>
    <span class="media-meta" title={metaText}>{metaText}</span>
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
    background-color: var(--light-bg);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
  }

  .thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .placeholder { font-size: 1.6rem; }

  .select-area {
    position: absolute;
    inset: 0;
    background: transparent;
    border: none;
    cursor: pointer;
    padding: 0;
  }
  .select-area:hover { background: rgba(52, 152, 219, 0.12); }
  .select-area:focus-visible { outline-offset: -3px; }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }

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
    pointer-events: none;
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
    pointer-events: none;
  }
  .check.in-library { background-color: var(--state-ok); }

  /* Quiet until the pointer or focus is on the card; never hidden, so
     keyboard and touch users can find them */
  .overlay {
    position: absolute;
    left: 6px;
    right: 6px;
    bottom: 6px;
    display: flex;
    justify-content: space-between;
    pointer-events: none;
  }

  .ctl {
    pointer-events: auto;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: none;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    color: white;
    background-color: rgba(0, 0, 0, 0.45);
    opacity: 0.75;
    cursor: pointer;
    transition: opacity 0.15s, background-color 0.15s;
  }

  .ctl.right { margin-left: auto; }
  .media-card:hover .ctl, .ctl:focus-visible, .ctl.busy { opacity: 1; }
  .ctl:hover:not(:disabled) { background-color: var(--primary-color); }
  .ctl:disabled { cursor: default; }

  .media-info { display: flex; flex-direction: column; padding: 5px 8px 6px; }

  .media-name {
    font-size: 0.8rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .media-meta {
    font-size: 0.7rem;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
