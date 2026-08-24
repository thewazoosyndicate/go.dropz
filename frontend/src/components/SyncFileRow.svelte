<script>
  // One file of a running or finished sync. The same row serves both, so
  // history reads exactly like the live view did.
  import { formatBytes, formatDuration } from '../lib/format.js';
  import ProgressBar from './ui/ProgressBar.svelte';

  const { ipcRenderer } = window.require('electron');

  let { file, rateBps = 0 } = $props();

  let icon = $derived({
    done: { cls: 'fa-check', color: 'var(--state-ok)' },
    downloading: { cls: 'fa-download', color: 'var(--state-busy)' },
    failed: { cls: 'fa-triangle-exclamation', color: 'var(--state-error)' },
    skipped: { cls: 'fa-check', color: 'var(--text-muted)' },
    queued: { cls: 'fa-clock', color: 'var(--border-color)' },
  }[file.state] || { cls: 'fa-clock', color: 'var(--border-color)' });

  let percent = $derived(file.sizeBytes > 0 ? (file.bytesDone / file.sizeBytes) * 100 : 0);
  let eta = $derived.by(() => {
    if (file.state !== 'downloading' || rateBps <= 0 || file.sizeBytes <= file.bytesDone) return '';
    const s = (file.sizeBytes - file.bytesDone) / rateBps;
    return s < 5 ? 'almost done' : `${Math.round(s)} s left`;
  });

  function showInFolder() {
    if (file.localPath) ipcRenderer.send('desktop-show', file.localPath);
  }
</script>

<div class="row state-{file.state}">
  <i class="fas {icon.cls}" style:color={icon.color} aria-hidden="true"></i>
  <span class="name" title={file.cameraPath || file.name}>{file.name}</span>
  <span class="size">{formatBytes(file.sizeBytes)}</span>
  <div class="status">
    {#if file.state === 'done'}
      <span>Done{file.durationMs ? ` · ${formatDuration(file.durationMs)}` : ''}</span>
    {:else if file.state === 'downloading'}
      <ProgressBar value={percent} height={4} label="File progress" />
      <span class="pct">{Math.round(percent)}%</span>
    {:else if file.state === 'failed'}
      <span class="failed" title={file.error}>Failed{file.error ? ` · ${file.error}` : ''}</span>
    {:else if file.state === 'skipped'}
      <span>Already in library</span>
    {:else}
      <span>Queued</span>
    {/if}
  </div>
  <div class="action">
    {#if file.state === 'downloading'}
      <span class="eta">{eta}</span>
    {:else if file.localPath}
      <button class="link" onclick={showInFolder}>Show in folder</button>
    {/if}
  </div>
</div>

<style>
  .row {
    display: grid;
    grid-template-columns: 20px minmax(0, 1fr) 80px 200px 110px;
    gap: 12px;
    align-items: center;
    padding: 6px 0;
    border-bottom: 1px dashed var(--border-color);
    font-size: 0.78rem;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }

  .row:last-child { border-bottom: none; }
  .row.state-queued, .row.state-skipped { color: var(--text-muted); }
  .row.state-downloading { background: var(--state-busy-tint); border-radius: 4px; padding: 6px 4px; }
  .row i { font-size: 0.85rem; text-align: center; }

  .name {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 0.75rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .state-downloading .name { font-weight: 600; }

  .size { text-align: right; color: var(--text-secondary); }

  .status { display: flex; align-items: center; gap: 8px; color: var(--text-secondary); min-width: 0; }
  .status span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .pct { flex-shrink: 0; font-size: 0.72rem; }
  .failed { color: var(--state-error); }

  .action { text-align: right; }
  .eta { color: var(--text-secondary); }

  .link {
    background: none;
    border: none;
    color: var(--primary-color);
    font-size: 0.78rem;
    cursor: pointer;
    padding: 4px 0;
  }
  .link:hover { text-decoration: underline; }
</style>
