<script>
  import { getServiceRunning, isAnyReconnecting } from '../lib/stores/connection.svelte.js';
  import { getDiscoveredDevices, getManagedDevices } from '../lib/stores/devices.svelte.js';
  import { getSyncQueue } from '../lib/stores/sync.svelte.js';
  import { toggleTheme, getTheme, setSettingsOpen } from '../lib/stores/ui.svelte.js';

  let discoveredCount = $derived(Object.keys(getDiscoveredDevices()).length);
  let managedCount = $derived(Object.keys(getManagedDevices()).length);
  let syncingCount = $derived(getSyncQueue().length);
  let running = $derived(getServiceRunning());
  let reconnecting = $derived(isAnyReconnecting());
  let isDark = $derived(getTheme() === 'dark');

  let statusClass = $derived(
    running ? (reconnecting ? 'reconnecting' : 'running') : ''
  );
  let statusLabel = $derived(
    running ? (reconnecting ? 'Reconnecting...' : 'Running') : 'Stopped'
  );
</script>

<header class="status-bar">
  <div class="status-left">
    <img src="imgs/logo.svg" alt="Dropz" class="logo" />
    <span class="summary">
      {discoveredCount} seen &middot; {managedCount} managed &middot; {syncingCount} syncing
    </span>
  </div>
  <div class="status-right">
    <div class="connection-status">
      <div class="status-dot {statusClass}"></div>
      <span class="status-label">{statusLabel}</span>
    </div>
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

  .connection-status {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background-color: var(--danger-color);
  }

  .status-dot.running {
    background-color: var(--secondary-color);
  }

  .status-dot.reconnecting {
    background-color: var(--warning-color);
    animation: blink 1.5s infinite;
  }

  .status-label {
    font-size: 0.8rem;
    color: var(--text-secondary);
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
