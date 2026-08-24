<script>
  import { getServiceRunning, isAnyReconnecting } from '../lib/stores/connection.svelte.js';
  import { getDiscoveredDevices, getManagedDevices, getAllDevices } from '../lib/stores/devices.svelte.js';
  import { getSyncQueue } from '../lib/stores/sync.svelte.js';
  import { toggleTheme, getTheme, setSettingsOpen } from '../lib/stores/ui.svelte.js';

  let discoveredCount = $derived(Object.keys(getDiscoveredDevices()).length);
  let managedCount = $derived(Object.keys(getManagedDevices()).length);
  let syncingCount = $derived(getSyncQueue().length);
  let running = $derived(getServiceRunning());
  let reconnecting = $derived(isAnyReconnecting());
  let isDark = $derived(getTheme() === 'dark');

  // Healthy is the permanent state and says nothing; only the
  // exceptions (backend gone, streams reconnecting) earn chrome.
  let problem = $derived(
    !running ? { cls: 'stopped', label: 'Backend stopped' } :
    reconnecting ? { cls: 'reconnecting', label: 'Reconnecting...' } : null
  );

  // The radio is a single slot; show which camera holds it and why.
  let activeLink = $derived.by(() => {
    const devices = Object.values(getAllDevices());
    for (const e of getSyncQueue()) {
      const d = devices.find(x => x.id === e.cameraId);
      if (!d?.isSyncing) continue;
      const name = d.wifiSsid?.trim()?.substring(0, 12) || d.name || 'camera';
      const op = e.currentOperation || '';
      const isPreview = op.includes('preview') || op.startsWith('Preview');
      return isPreview
        ? { icon: 'fa-satellite-dish', label: `Connected to ${name}` }
        : { icon: 'fa-sync fa-spin', label: `Syncing ${name}` };
    }
    return null;
  });
</script>

<header class="status-bar">
  <div class="status-left">
    <img src="imgs/logo.svg" alt="Dropz" class="logo" />
    <span class="summary">
      {discoveredCount} seen &middot; {managedCount} managed &middot; {syncingCount} syncing
    </span>
  </div>
  <div class="status-right">
    {#if problem}
      <span class="status-pill {problem.cls}">{problem.label}</span>
    {:else if activeLink}
      <span class="active-link"><i class="fas {activeLink.icon}"></i> {activeLink.label}</span>
    {/if}
    <button class="icon-btn" onclick={() => setSettingsOpen(true)} title="Settings">
      <i class="fas fa-cog"></i>
    </button>
    <button class="icon-btn" onclick={toggleTheme} title="Toggle theme">
      <i class="fas {isDark ? 'fa-sun' : 'fa-moon'}"></i>
    </button>
  </div>
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

  .status-pill.stopped {
    background-color: var(--danger-color);
  }

  .status-pill.reconnecting {
    background-color: var(--warning-color);
    animation: blink 1.5s infinite;
  }

  .active-link {
    font-size: 0.8rem;
    color: var(--secondary-color);
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .icon-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .icon-btn:hover {
    background-color: rgba(0, 0, 0, 0.05);
  }

  :global([data-theme="dark"]) .icon-btn:hover {
    background-color: rgba(255, 255, 255, 0.1);
  }
</style>
