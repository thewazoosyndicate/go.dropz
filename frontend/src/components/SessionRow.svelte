<script>
  // One finished sync in the history list; expands to its files and
  // phase timings.
  import { findDeviceById, displayName } from '../lib/stores/devices.svelte.js';
  import { describeSession } from '../lib/stores/sync.svelte.js';
  import { addToSyncQueue } from '../lib/grpc/actions.js';
  import { openLibrary } from '../lib/stores/ui.svelte.js';
  import { formatDayLabel, formatClock, formatDuration, plural } from '../lib/format.js';
  import SyncFileRow from './SyncFileRow.svelte';
  import PhaseStepper from './PhaseStepper.svelte';

  let { session } = $props();

  let open = $state(false);
  let device = $derived(findDeviceById(session.cameraId));
  let name = $derived(displayName(device));
  let summary = $derived(describeSession(session));
  let when = $derived(`${formatDayLabel(session.finishedAt)} ${formatClock(session.finishedAt)}`);
  let elapsed = $derived(formatDuration(session.finishedAt - session.startedAt));

  let icon = $derived({
    complete: session.filesFailed ? { cls: 'fa-triangle-exclamation', color: 'var(--state-busy)' } : { cls: 'fa-check', color: 'var(--state-ok)' },
    up_to_date: { cls: 'fa-check', color: 'var(--text-muted)' },
    catalog_refreshed: { cls: 'fa-photo-film', color: 'var(--text-muted)' },
    failed: { cls: 'fa-triangle-exclamation', color: 'var(--state-error)' },
    cancelled: { cls: 'fa-ban', color: 'var(--text-muted)' },
  }[session.outcome] || { cls: 'fa-circle', color: 'var(--text-muted)' });

  let detail = $derived.by(() => {
    if (session.outcome === 'failed') {
      const step = session.stepIndex ? `step ${session.stepIndex} of ${session.stepCount}` : '';
      return [session.failedStep ? `stopped at ${session.failedStep.toLowerCase()}` : '', step].filter(Boolean).join(', ');
    }
    if (session.outcome === 'complete' && session.filesDownloaded > 0) return elapsed;
    if (session.outcome === 'up_to_date') return `verified over Bluetooth in ${elapsed}`;
    return '';
  });

  let hasFiles = $derived(session.files.length > 0);
  let canRetry = $derived(session.outcome === 'failed' && device?.isReachable && !device?.isSyncing);
  // The phase that was running when a failed sync stopped
  let failedPhase = $derived(session.outcome === 'failed' && session.phases.length
    ? session.phases[session.phases.length - 1].phase : '');

  function toggle() { open = !open; }
  function onRowKey(e) {
    if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggle(); }
  }
  function showInLibrary(e) {
    e.stopPropagation();
    openLibrary('local', { sessionId: session.id });
  }
  function retry(e) {
    e.stopPropagation();
    addToSyncQueue(device.macAddress);
  }
</script>

<div class="session" class:failed={session.outcome === 'failed'}>
  <div class="row" role="button" tabindex="0" aria-expanded={open} onclick={toggle} onkeydown={onRowKey}>
    <i class="fas {icon.cls}" style:color={icon.color} aria-hidden="true"></i>
    <span class="name">{name}</span>
    <span class="when">{when}</span>
    <span class="summary">
      <span class="text">{summary.text}</span>
      {#if detail}<span class="detail"> &middot; {detail}</span>{/if}
      {#if session.selection}<span class="detail"> &middot; selection</span>{/if}
    </span>
    <span class="action">
      {#if session.filesDownloaded > 0}
        <button class="link" onclick={showInLibrary}>Show in Library</button>
      {:else if canRetry}
        <button class="link" onclick={retry}>Retry now</button>
      {/if}
    </span>
    <i class="fas fa-chevron-down chevron" class:open aria-hidden="true"></i>
  </div>
  {#if open}
    <div class="body">
      {#if session.phases.length > 0}
        <PhaseStepper phase="" phases={session.phases} {failedPhase} />
      {/if}
      {#if session.outcome === 'failed' && session.error}
        <p class="explain">{session.error}. Press Retry now, or the next sync runs when new clips appear on the camera.</p>
      {/if}
      {#if hasFiles}
        <div class="files">
          {#each session.files as file (file.cameraPath || file.name)}
            <SyncFileRow {file} />
          {/each}
        </div>
        <div class="counts">
          {plural(session.filesDownloaded, 'file')} downloaded
          {#if session.filesFailed}&middot; {session.filesFailed} failed{/if}
          {#if session.filesSkipped}&middot; {session.filesSkipped} already in the library{/if}
        </div>
      {:else}
        <p class="explain muted">No files were considered in this run.</p>
      {/if}
    </div>
  {/if}
</div>

<style>
  .session { border-bottom: 1px solid var(--border-color); }
  .session:last-child { border-bottom: none; }
  .session.failed { background: var(--state-error-tint); }

  .row {
    display: grid;
    grid-template-columns: 20px 150px 150px minmax(0, 1fr) 130px 16px;
    gap: 12px;
    align-items: center;
    padding: 10px 14px;
    min-height: 44px;
    color: var(--text-primary);
    font-size: 0.82rem;
    cursor: pointer;
  }

  .row:hover { background: var(--hover-bg); }
  .row > i { text-align: center; }
  .name { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .when { color: var(--text-muted); }
  .summary { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .detail { color: var(--text-secondary); }
  .action { text-align: right; }

  .link {
    background: none;
    border: none;
    color: var(--primary-color);
    font-size: 0.82rem;
    cursor: pointer;
    padding: 6px 0;
  }
  .link:hover { text-decoration: underline; }

  .chevron { color: var(--text-muted); transition: transform 0.15s; font-size: 0.75rem; }
  .chevron.open { transform: rotate(180deg); }

  .body {
    padding: 4px 14px 14px 46px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .explain { font-size: 0.78rem; color: var(--text-primary); }
  .muted { color: var(--text-muted); }
  .files { display: flex; flex-direction: column; }
  .counts { font-size: 0.75rem; color: var(--text-secondary); }
</style>
