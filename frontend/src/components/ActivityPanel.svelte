<script>
  import ActiveSyncCard from './ActiveSyncCard.svelte';
  import SessionRow from './SessionRow.svelte';
  import DiagnosticsLog from './DiagnosticsLog.svelte';
  import Button from './ui/Button.svelte';
  import { getSyncQueue, getSyncHistory, isHistoryLoaded } from '../lib/stores/sync.svelte.js';
  import { getManagedDevices, findDeviceById, displayName } from '../lib/stores/devices.svelte.js';
  import { getActivityTarget, clearActivityTarget, getLogsExpanded, setLogsExpanded } from '../lib/stores/ui.svelte.js';
  import { loadSyncHistory } from '../lib/grpc/history.js';

  // 'all' | 'attention' | cameraId
  let filter = $state('all');
  let logsOpen = $derived(getLogsExpanded());

  $effect(() => {
    const target = getActivityTarget();
    if (target) {
      filter = target.cameraId || 'all';
      clearActivityTarget();
    }
  });

  // History is loaded on start and after each sync; a manual open still
  // refreshes in case the stream missed a shrink while reconnecting.
  $effect(() => { loadSyncHistory(); });

  let cameras = $derived(Object.values(getManagedDevices())
    .filter(d => d.id)
    .map(d => ({ id: d.id, name: displayName(d) }))
    .sort((a, b) => a.name.localeCompare(b.name)));

  let queue = $derived(getSyncQueue().filter(e => filter === 'all' || filter === 'attention' ? true : e.cameraId === filter));
  // Running first, then waiting in queue order
  let now = $derived([...queue].sort((a, b) => {
    const ra = findDeviceById(a.cameraId)?.isSyncing ? 0 : 1;
    const rb = findDeviceById(b.cameraId)?.isSyncing ? 0 : 1;
    return ra - rb;
  }));

  let sessions = $derived(getSyncHistory().filter(s => {
    if (filter === 'attention') return s.outcome === 'failed' || s.filesFailed > 0;
    if (filter !== 'all') return s.cameraId === filter;
    return true;
  }));
</script>

<section class="panel">
  <div class="panel-header">
    <h2><i class="fas fa-wave-square" aria-hidden="true"></i> Activity</h2>
    <div class="header-actions">
      <div class="chips" role="tablist" aria-label="Filter">
        <button class="chip" class:active={filter === 'all'} onclick={() => filter = 'all'}>All</button>
        <button class="chip" class:active={filter === 'attention'} onclick={() => filter = 'attention'}>Needs attention</button>
        {#each cameras as cam (cam.id)}
          <button class="chip" class:active={filter === cam.id} onclick={() => filter = cam.id}>{cam.name}</button>
        {/each}
      </div>
      <span class="divider"></span>
      <Button size="sm" icon="fa-terminal" onclick={() => setLogsExpanded(!logsOpen)}>
        {logsOpen ? 'Hide diagnostics' : 'Diagnostics log'}
      </Button>
    </div>
  </div>

  <div class="panel-content">
    <div class="section">
      <div class="section-title">Now</div>
      {#if now.length === 0}
        <p class="quiet">Nothing running. Syncs start on their own when a managed camera has new clips.</p>
      {:else}
        <div class="stack">
          {#each now as entry (entry.cameraId)}
            <ActiveSyncCard {entry} />
          {/each}
        </div>
      {/if}
    </div>

    <div class="section">
      <div class="section-title">Earlier</div>
      {#if sessions.length === 0}
        <p class="quiet">{isHistoryLoaded() ? 'No syncs recorded yet.' : 'Loading...'}</p>
      {:else}
        <div class="list">
          {#each sessions as session (session.id)}
            <SessionRow {session} />
          {/each}
        </div>
      {/if}
    </div>

    {#if logsOpen}
      <div class="section">
        <div class="section-title">Diagnostics</div>
        <DiagnosticsLog />
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
  }

  .panel-header {
    padding: 12px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    border-bottom: 1px solid var(--border-color);
  }

  .panel-header h2 {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 1rem;
    white-space: nowrap;
  }

  .header-actions { display: flex; align-items: center; gap: 8px; min-width: 0; }
  .chips { display: flex; gap: 6px; flex-wrap: wrap; }

  .chip {
    background: none;
    border: 1px solid var(--border-color);
    border-radius: 14px;
    padding: 4px 12px;
    min-height: 30px;
    font-size: 0.8rem;
    color: var(--text-secondary);
    cursor: pointer;
  }
  .chip:hover { color: var(--text-primary); }
  .chip.active { background: var(--primary-color); border-color: var(--primary-color); color: white; }

  .divider { width: 1px; height: 20px; background: var(--border-color); }

  .panel-content {
    flex: 1;
    padding: 12px 16px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 18px;
  }

  .section { display: flex; flex-direction: column; gap: 8px; }

  .section-title {
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-muted);
  }

  .stack { display: flex; flex-direction: column; gap: 10px; }

  .list {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border-color);
    border-radius: 8px;
    overflow: hidden;
  }

  .quiet { font-size: 0.82rem; color: var(--text-muted); }
</style>
