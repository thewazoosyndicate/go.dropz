<script>
  import { pairDevice, toggleDeviceManaged } from '../lib/grpc/actions.js';
  import { getPairingInProgress } from '../lib/stores/devices.svelte.js';

  let { device } = $props();

  let displayName = $derived(
    device.wifiSsid?.trim() ? device.wifiSsid.substring(0, 12) : (device.name || 'Unknown GoPro')
  );

  let isPairing = $derived(!!getPairingInProgress()[device.macAddress]);
  let signalStrength = $derived(getSignalLevel(device.rssi || -100));
  let signalColor = $derived(
    signalStrength >= 3 ? 'var(--secondary-color)' :
    signalStrength >= 2 ? 'var(--warning-color)' : 'var(--danger-color)'
  );

  let batteryLevel = $derived(Math.max(0, Math.min(100, device.batteryLevel ?? 0)));
  let batteryColor = $derived(
    batteryLevel < 20 ? 'var(--danger-color)' :
    batteryLevel < 50 ? 'var(--warning-color)' : 'var(--secondary-color)'
  );

  function getSignalLevel(rssi) {
    if (rssi >= -50) return 4;
    if (rssi >= -65) return 3;
    if (rssi >= -80) return 2;
    return 1;
  }

  function handlePair() {
    if (!isPairing && !device.isPaired) pairDevice(device.macAddress);
  }

  function handleManage() {
    toggleDeviceManaged(device.macAddress, true);
  }
</script>

<div class="row" class:unreachable={!device.isReachable}>
  <div class="signal">
    <div class="signal-bars">
      {#each [1, 2, 3, 4] as level}
        <div class="bar" class:filled={level <= signalStrength}
             style:--bar-color={signalColor}></div>
      {/each}
    </div>
  </div>

  <span class="name" title={displayName}>{displayName}</span>

  {#if device.inPairingMode && !device.isPaired}
    <span class="pairing-mode-badge" title="Camera is showing its pairing screen">
      <i class="fas fa-link"></i> ready to pair
    </span>
  {/if}

  {#if device.batteryLevel != null}
    <div class="battery">
      <div class="battery-icon">
        <div class="battery-fill" style:width="{batteryLevel}%"
             style:background-color={batteryColor}></div>
      </div>
    </div>
  {/if}

  {#if isPairing}
    <button class="pair-btn pairing" disabled aria-label="Pairing">
      <i class="fas fa-spinner fa-spin"></i>
    </button>
  {:else if device.isPaired && !device.isManaged}
    <button class="pair-btn manage" onclick={handleManage}>
      Manage
    </button>
  {:else if device.isPaired && device.isManaged}
    <button class="pair-btn paired" disabled aria-label="Paired">
      <i class="fas fa-check"></i>
    </button>
  {:else}
    <button class="pair-btn" onclick={handlePair}>
      Pair
    </button>
  {/if}
</div>

<style>
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 14px;
    border-bottom: 1px solid var(--border-color);
    min-height: 44px;
    transition: background-color 0.15s;
  }

  .row:hover {
    background-color: var(--light-bg);
  }

  .row.unreachable {
    opacity: 0.45;
  }

  .signal-bars {
    display: flex;
    align-items: flex-end;
    gap: 1px;
    height: 12px;
  }

  .bar {
    width: 3px;
    background-color: var(--border-color);
    border-radius: 1px;
  }
  .bar:nth-child(1) { height: 3px; }
  .bar:nth-child(2) { height: 6px; }
  .bar:nth-child(3) { height: 9px; }
  .bar:nth-child(4) { height: 12px; }
  .bar.filled { background-color: var(--bar-color); }

  .pairing-mode-badge {
    font-size: 0.7rem;
    color: var(--secondary-color);
    border: 1px solid var(--secondary-color);
    border-radius: 8px;
    padding: 1px 6px;
    white-space: nowrap;
  }

  .name {
    flex: 1;
    font-size: 0.85rem;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .battery {
    flex-shrink: 0;
  }

  .battery-icon {
    width: 18px;
    height: 9px;
    border: 1px solid var(--text-muted);
    border-radius: 2px;
    overflow: hidden;
    position: relative;
  }

  .battery-icon::after {
    content: '';
    position: absolute;
    right: -3px;
    top: 2px;
    width: 2px;
    height: 4px;
    background: var(--text-muted);
    border-radius: 0 1px 1px 0;
  }

  .battery-fill {
    height: 100%;
    transition: width 0.3s;
  }

  .pair-btn {
    flex-shrink: 0;
    padding: 5px 14px;
    border: none;
    border-radius: 5px;
    background-color: var(--primary-color);
    color: white;
    font-size: 0.75rem;
    font-weight: 500;
    cursor: pointer;
    min-width: 50px;
    min-height: 30px;
    transition: all 0.2s;
  }

  .pair-btn:hover:not(:disabled) {
    background-color: var(--primary-dark);
    transform: translateY(-1px);
  }

  .pair-btn:disabled {
    cursor: default;
  }

  .pair-btn.paired {
    background-color: var(--secondary-color);
    opacity: 0.7;
  }

  .pair-btn.manage {
    background-color: var(--secondary-color);
  }

  .pair-btn.pairing {
    background-color: var(--warning-color);
    opacity: 0.8;
  }
</style>
