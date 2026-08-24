<script>
  // Four user phases in place of nine backend labels. compact: one line
  // of labels for a card; full: bars with per-phase timings for Activity.
  import { formatDuration } from '../lib/format.js';

  let { phase = 'waiting', phases = [], compact = false, entry = null, failedPhase = '' } = $props();

  const STEPS = [
    { id: 'connect', label: 'Connect' },
    { id: 'link', label: 'Wi-Fi' },
    { id: 'catalog', label: 'Catalog' },
    { id: 'transfer', label: 'Transfer' },
  ];

  let steps = $derived(STEPS.map(s => {
    const timing = phases.find(p => p.phase === s.id);
    const failed = failedPhase === s.id;
    const done = !!timing?.finishedAt && phase !== s.id && !failed;
    const active = phase === s.id;
    let detail = '';
    if (!compact && timing) {
      const end = timing.finishedAt || new Date();
      detail = formatDuration(end - timing.startedAt);
    }
    if (!compact && s.id === 'transfer' && entry?.fileCount > 0) {
      detail = `${entry.fileIndex} of ${entry.fileCount}`;
    }
    return { ...s, done, active, failed, detail };
  }));
</script>

{#if compact}
  <div class="stepper compact" aria-label="Sync phase">
    {#each steps as s, i (s.id)}
      {#if i > 0}<span class="link" class:done={s.done || s.active}></span>{/if}
      <span class="step" class:done={s.done} class:active={s.active}>
        {#if s.done}<i class="fas fa-check" aria-hidden="true"></i>
        {:else if s.active}<span class="dot"></span>{/if}
        {s.label}
      </span>
    {/each}
  </div>
{:else}
  <div class="stepper full" aria-label="Sync phase">
    {#each steps as s (s.id)}
      <div class="col" class:done={s.done} class:active={s.active} class:failed={s.failed}>
        <div class="bar"></div>
        <div class="row">
          <span>{s.label}</span>
          <span class="detail">{s.detail}</span>
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .compact {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: var(--text-muted);
    white-space: nowrap;
  }

  .compact .step { display: flex; align-items: center; gap: 3px; }
  .compact .step.done { color: var(--state-ok); }
  .compact .step.active { color: var(--state-busy); font-weight: 600; }
  .compact .step i { font-size: 9px; }
  .compact .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--state-busy); }
  .compact .link { width: 6px; height: 1px; background: var(--border-color); }
  .compact .link.done { background: var(--state-ok); }

  .full {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 6px;
  }

  .col { display: flex; flex-direction: column; gap: 4px; }
  .col .bar { height: 4px; border-radius: 2px; background: var(--border-color); }
  .col.done .bar { background: var(--state-ok); }
  .col.active .bar { background: var(--state-busy); animation: pulse 1.2s infinite; }
  .col.failed .bar { background: var(--state-error); }
  .col.failed .row { color: var(--state-error); font-weight: 600; }
  .col .row {
    display: flex;
    justify-content: space-between;
    font-size: 11.5px;
    color: var(--text-secondary);
    font-variant-numeric: tabular-nums;
  }
  .col.active .row { color: var(--state-busy); font-weight: 600; }
  .detail { color: var(--text-muted); }
</style>
