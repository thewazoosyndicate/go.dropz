<script>
  import { getServiceRunning, isAnyReconnecting } from '../lib/stores/connection.svelte.js';
  import { getNearbyCount, getManagedDevices, findDeviceById, displayName } from '../lib/stores/devices.svelte.js';
  import { getSyncQueue, getActiveSyncEntry } from '../lib/stores/sync.svelte.js';
  import { toggleTheme, getTheme, setSettingsOpen, openActivity } from '../lib/stores/ui.svelte.js';
  import { formatEta, formatTimeAgo } from '../lib/format.js';
  import IconButton from './ui/IconButton.svelte';

  let discoveredCount = $derived(getNearbyCount());
  let managedCount = $derived(Object.keys(getManagedDevices()).length);
  let running = $derived(getServiceRunning());
  let reconnecting = $derived(isAnyReconnecting());
  let isDark = $derived(getTheme() === 'dark');

  // Healthy is the permanent state and says nothing; only the
  // exceptions (backend gone, streams reconnecting) earn chrome.
  let problem = $derived(
    !running ? { cls: 'stopped', label: 'Backend stopped' } :
    reconnecting ? { cls: 'reconnecting', label: 'Reconnecting...' } : null
  );

  let active = $derived(getActiveSyncEntry());
  let activeDevice = $derived(active ? findDeviceById(active.cameraId) : null);
  let waitingCount = $derived(getSyncQueue().length - (active ? 1 : 0));

  // The radio is a single slot; the pill says who holds it and how far along.
  let activity = $derived.by(() => {
    if (!active) return null;
    const name = displayName(activeDevice);
    const op = active.currentOperation || '';
    if (op.includes('preview') || op.startsWith('Preview') || op.startsWith('Fetching') || op.startsWith('Converting')) {
      return { kind: 'preview', name, detail: 'camera link open', fraction: 0 };
    }
    if (active.fileCount > 0) {
      const parts = [`${active.fileIndex} of ${active.fileCount}`];
      if (active.rateBps > 0 && active.bytesTotal > active.bytesDone) {
        parts.push(formatEta((active.bytesTotal - active.bytesDone) / active.rateBps));
      }
      const fraction = active.bytesTotal > 0 ? active.bytesDone / active.bytesTotal : (active.fileIndex - 1) / active.fileCount;
      return { kind: 'transfer', name, detail: parts.join(' · '), fraction };
    }
    return { kind: 'busy', name, detail: op.toLowerCase(), fraction: (active.progressPercent || 0) / 100 };
  });

  // Resting summary: the one fact a glance needs
  let resting = $derived.by(() => {
    const managed = Object.values(getManagedDevices());
    if (managed.length === 0) return null;
    const failed = managed.filter(d => d.lastSyncError && !d.isSyncing).length;
    if (failed > 0) return { icon: 'fa-triangle-exclamation', cls: 'attention', label: `${failed} ${failed === 1 ? 'camera needs' : 'cameras need'} attention` };
    const fresh = managed.reduce((n, d) => n + (d.newMediaCount || 0), 0);
    if (fresh > 0) return { icon: 'fa-video', cls: 'fresh', label: `${fresh} new on camera` };
    const latest = managed.map(d => d.lastSynced).filter(Boolean).sort((a, b) => b - a)[0];
    const ago = formatTimeAgo(latest);
    return { icon: 'fa-check', cls: 'ok', label: ago ? `Everything synced · ${ago}` : 'Ready' };
  });
</script>

<header class="status-bar">
  <div class="status-left">
    <img src="imgs/logo.svg" alt="Dropz" class="logo" />
    <span class="summary">
      {managedCount} managed &middot; {discoveredCount} nearby
    </span>
  </div>
  <div class="status-right">
    {#if problem}
      <span class="status-pill {problem.cls}">{problem.label}</span>
    {:else if activity}
      <button class="activity-pill" onclick={() => openActivity(active.cameraId)} title="Open activity">
        <i class="fas {activity.kind === 'preview' ? 'fa-satellite-dish' : 'fa-rotate fa-spin'}" aria-hidden="true"></i>
        <span class="pill-name">{activity.name}</span>
        <span class="pill-detail">{activity.detail}</span>
        {#if waitingCount > 0}
          <span class="pill-sep"></span>
          <span class="pill-detail">{waitingCount} queued</span>
        {/if}
      </button>
    {:else if waitingCount > 0}
      <button class="activity-pill quiet" onclick={() => openActivity()} title="Open activity">
        <i class="fas fa-clock" aria-hidden="true"></i>
        <span class="pill-detail">{waitingCount} waiting for a camera in range</span>
      </button>
    {:else if resting}
      <span class="resting {resting.cls}">
        <i class="fas {resting.icon}" aria-hidden="true"></i> {resting.label}
      </span>
    {/if}
    <IconButton icon="fa-cog" title="Settings" onclick={() => setSettingsOpen(true)} />
    <IconButton icon={isDark ? 'fa-sun' : 'fa-moon'} title="Toggle theme" onclick={toggleTheme} />
  </div>
  {#if activity && activity.kind !== 'preview'}
    <div class="global-line" aria-hidden="true">
      <div class="global-fill" style:width="{Math.round(activity.fraction * 100)}%"></div>
    </div>
  {/if}
</header>

<style>
  .status-bar {
    background-color: var(--panel-bg);
    box-shadow: var(--shadow-sm);
    height: 44px;
    padding: 0 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    position: relative;
    z-index: 10;
    flex-shrink: 0;
    transition: background-color 0.3s ease;
  }

  .status-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo {
    height: 28px;
    width: auto;
  }

  .summary {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .status-right {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .status-pill {
    font-size: 0.75rem;
    font-weight: 500;
    color: white;
    padding: 3px 10px;
    border-radius: 10px;
  }

  .status-pill.stopped { background-color: var(--danger-color); }
  .status-pill.reconnecting { background-color: var(--warning-color); animation: blink 1.5s infinite; }

  .activity-pill {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 12px 4px 10px;
    min-height: 28px;
    border-radius: 16px;
    background: var(--state-busy-tint);
    border: 1px solid var(--state-busy);
    color: var(--text-primary);
    font-size: 0.8rem;
    cursor: pointer;
  }

  .activity-pill i { color: var(--state-busy); font-size: 0.85rem; }
  .activity-pill.quiet { background: var(--state-idle-tint); border-color: var(--border-color); }
  .activity-pill.quiet i { color: var(--text-secondary); }
  .pill-name { font-weight: 600; }
  .pill-detail { color: var(--text-secondary); font-variant-numeric: tabular-nums; }
  .pill-sep { width: 1px; height: 14px; background: var(--border-color); }

  .resting {
    font-size: 0.8rem;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .resting.ok i { color: var(--state-ok); }
  .resting.attention i { color: var(--state-error); }
  .resting.fresh i { color: var(--state-info); }

  .global-line {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 3px;
    background: var(--border-color);
  }

  .global-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--state-busy), var(--state-ok));
    transition: width 0.5s;
  }
</style>
