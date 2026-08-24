<script>
  import SearchBar from './SearchBar.svelte';
  import DiscoveredRow from './DiscoveredRow.svelte';
  import Toggle from './ui/Toggle.svelte';
  import { getDiscoveredDevices, getNearbyCount } from '../lib/stores/devices.svelte.js';
  import { getSearchQuery, getSortBy } from '../lib/stores/ui.svelte.js';
  import { togglePairAll } from '../lib/grpc/actions.js';
  import { getAutoPair } from '../lib/stores/config.svelte.js';

  let autoPairEnabled = $derived(getAutoPair());
  let seenCount = $derived(getNearbyCount());

  let filteredDevices = $derived.by(() => {
    const devices = Object.values(getDiscoveredDevices());
    const query = getSearchQuery().toLowerCase();
    const sort = getSortBy();

    let result = devices;
    if (query) {
      result = result.filter(d =>
        (d.name || '').toLowerCase().includes(query) ||
        (d.wifiSsid || '').toLowerCase().includes(query) ||
        (d.serialNumber || '').toLowerCase().includes(query) ||
        (d.macAddress || '').toLowerCase().includes(query)
      );
    }

    if (sort === 'name') {
      result.sort((a, b) => (a.name || '').localeCompare(b.name || ''));
    } else {
      // Default: signal strength descending
      const SORT_HYSTERESIS = 8;
      result.sort((a, b) => {
        const diff = (b.rssi || -100) - (a.rssi || -100);
        if (Math.abs(diff) < SORT_HYSTERESIS) return 0;
        return diff;
      });
    }
    return result;
  });
</script>

<section class="panel">
  <div class="panel-header">
    <h2><i class="fas fa-search" aria-hidden="true"></i> Nearby</h2>
    <div class="header-controls">
      <span class="badge" title="Cameras in range">{seenCount}</span>
      <Toggle checked={autoPairEnabled} label="Pair all" onchange={togglePairAll} />
    </div>
  </div>
  <SearchBar />
  <div class="panel-content">
    {#if filteredDevices.length === 0}
      <div class="empty-state">
        <img src="imgs/3_dropz.svg" alt="Dropz" class="empty-logo" />
        <p>No cameras nearby</p>
        <p class="hint">Cameras appear here when they are on and within Bluetooth range.</p>
      </div>
    {:else}
      <div class="device-list">
        {#each filteredDevices as device (device.macAddress)}
          <DiscoveredRow {device} />
        {/each}
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
    transition: background-color 0.3s ease;
  }

  .panel-header {
    padding: 10px 14px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--border-color);
  }

  .panel-header h2 {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 1rem;
  }

  .header-controls { display: flex; align-items: center; gap: 10px; }

  .badge {
    font-size: 0.8rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 2px 8px;
    border-radius: 10px;
  }

  .panel-content { flex: 1; overflow-y: auto; padding: 0; }

  .device-list { display: flex; flex-direction: column; }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    padding: 32px;
    color: var(--text-muted);
    text-align: center;
  }

  .empty-logo { height: 60px; opacity: 0.5; margin-bottom: 8px; }
  .hint { font-size: 0.78rem; margin-top: 4px; }
</style>
