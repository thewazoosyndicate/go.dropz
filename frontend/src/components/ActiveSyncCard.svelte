<script>
  // The running sync (or a waiting one) in the Activity tab: stepper with
  // timings, totals, and every file the transfer considered.
  import { cancelSync } from '../lib/grpc/actions.js';
  import { findDeviceById, displayName } from '../lib/stores/devices.svelte.js';
  import { getQueuePosition } from '../lib/stores/sync.svelte.js';
  import { formatBytes, formatRate, formatEta, formatClock, plural } from '../lib/format.js';
  import Button from './ui/Button.svelte';
  import PhaseStepper from './PhaseStepper.svelte';
  import SyncFileRow from './SyncFileRow.svelte';

  let { entry } = $props();

  let device = $derived(findDeviceById(entry.cameraId));
  let name = $derived(displayName(device));
  let running = $derived(!!device?.isSyncing);
  let position = $derived(running ? 0 : getQueuePosition(entry.cameraId));
  let isPreview = $derived(/preview/i.test(entry.currentOperation || ''));

  let totals = $derived.by(() => {
    const parts = [];
    if (entry.bytesTotal > 0) parts.push(`${formatBytes(entry.bytesDone)} of ${formatBytes(entry.bytesTotal)}`);
    if (entry.rateBps > 0) parts.push(formatRate(entry.rateBps));
    if (entry.rateBps > 0 && entry.bytesTotal > entry.bytesDone) parts.push(formatEta((entry.bytesTotal - entry.bytesDone) / entry.rateBps));
    return parts.join(' · ');
  });

  let files = $derived(entry.files || []);
  let skipped = $derived(files.filter(f => f.state === 'skipped').length);
  let queued = $derived(files.filter(f => f.state === 'queued'));
  let shown = $derived.by(() => {
    // Done, failed and the live file stay; the queue tail collapses to a count
    const visible = files.filter(f => f.state !== 'skipped' && f.state !== 'queued');
    return visible.concat(queued.slice(0, 2));
  });
  let hiddenQueued = $derived(Math.max(0, queued.length - 2));
  let catalogLine = $derived.by(() => {
    if (!files.length) return '';
    const fresh = files.length - skipped;
    return `${plural(fresh, 'new file')}, ${skipped} already here`;
  });
</script>

<div class="card" class:waiting={!running}>
  <div class="head">
    <span class="name">{name}</span>
    <span class="meta">
      {#if device?.model}{device.model} &middot; {/if}
      {#if running && entry.startedAt}started {formatClock(entry.startedAt)}{:else}queued {formatClock(entry.queuedAt)}{/if}
    </span>
    <span class="totals">
      {#if running}{totals || entry.currentOperation}{:else}{position === 1 ? 'Next in line' : `In line, #${position}`}{/if}
    </span>
    {#if running}
      <Button variant="danger-outline" size="sm" onclick={() => cancelSync(device.macAddress)}>Cancel</Button>
    {:else}
      <Button size="sm" onclick={() => cancelSync(device.macAddress)}>Remove</Button>
    {/if}
  </div>

  {#if running && !isPreview}
    <PhaseStepper phase={entry.phase} phases={entry.phases} {entry} />
    {#if catalogLine}<div class="catalog-line">{catalogLine}</div>{/if}
  {:else if running}
    <div class="catalog-line">{entry.currentOperation}</div>
  {/if}

  {#if running && shown.length > 0}
    <div class="files">
      {#each shown as file (file.cameraPath || file.name)}
        <SyncFileRow {file} rateBps={entry.rateBps} />
      {/each}
      {#if hiddenQueued > 0 || skipped > 0}
        <div class="foot">
          {#if hiddenQueued > 0}<span>{hiddenQueued} more queued</span>{/if}
          {#if hiddenQueued > 0 && skipped > 0}<span class="sep">&middot;</span>{/if}
          {#if skipped > 0}<span>{plural(skipped, 'file')} skipped, already in the library</span>{/if}
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .card {
    border: 1px solid var(--border-color);
    border-left: 4px solid var(--state-busy);
    border-radius: 8px;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .card.waiting { border-left-color: var(--state-idle); }

  .head { display: flex; align-items: center; gap: 12px; }
  .name { font-weight: 600; font-size: 1.05rem; }
  .meta { font-size: 0.75rem; color: var(--text-muted); }
  .totals {
    margin-left: auto;
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-variant-numeric: tabular-nums;
  }

  .catalog-line { font-size: 0.75rem; color: var(--text-secondary); }

  .files { display: flex; flex-direction: column; border-top: 1px solid var(--border-color); padding-top: 4px; }

  .foot {
    display: flex;
    align-items: center;
    gap: 10px;
    padding-top: 8px;
    font-size: 0.75rem;
    color: var(--text-secondary);
  }
  .sep { color: var(--border-color); }
</style>
