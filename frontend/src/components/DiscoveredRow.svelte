<script>
  import { pairDevice, toggleDeviceManaged } from '../lib/grpc/actions.js';
  import { getPairingInProgress, displayName } from '../lib/stores/devices.svelte.js';
  import Button from './ui/Button.svelte';

  let { device } = $props();

  let name = $derived(displayName(device));
  let isPairing = $derived(!!getPairingInProgress()[device.macAddress]);
  let signalStrength = $derived(getSignalLevel(device.rssi || -100));
  let signalColor = $derived(
    signalStrength >= 3 ? 'var(--state-ok)' :
    signalStrength >= 2 ? 'var(--state-busy)' : 'var(--state-error)'
  );

  let batteryLevel = $derived(Math.max(0, Math.min(100, device.batteryLevel ?? 0)));
  let batteryColor = $derived(
    batteryLevel < 20 ? 'var(--state-error)' :
    batteryLevel < 50 ? 'var(--state-busy)' : 'var(--state-ok)'
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
  <div class="signal-bars" title="Signal">
    {#each [1, 2, 3, 4] as level}
      <div class="bar" class:filled={level <= signalStrength}
           style:--bar-color={signalColor}></div>
    {/each}
  </div>

  <span class="name" title={name}>{name}</span>

  {#if device.inPairingMode && !device.isPaired}
    <span class="pairing-mode-badge" title="Camera is showing its pairing screen">
      <i class="fas fa-link" aria-hidden="true"></i> ready to pair
    </span>
  {/if}

  {#if device.batteryLevel != null}
    <div class="battery-icon" title="Battery {batteryLevel}%">
      <div class="battery-fill" style:width="{batteryLevel}%" style:background-color={batteryColor}></div>
    </div>
  {/if}

  {#if isPairing}
    <Button size="sm" variant="primary" disabled icon="fa-spinner fa-spin" ariaLabel="Pairing" />
  {:else if device.isPaired && !device.isManaged}
    <Button size="sm" variant="success" onclick={handleManage}>Manage</Button>
  {:else if device.isPaired && device.isManaged}
    <Button size="sm" variant="success" disabled icon="fa-check" ariaLabel="Paired" />
  {:else}
    <Button size="sm" variant="primary" onclick={handlePair}>Pair</Button>
  {/if}
</div>

<style>
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 14px;
    border-bottom: 1px solid var(--border-color);
    min-height: 46px;
    transition: background-color 0.15s;
  }

  .row:hover { background-color: var(--light-bg); }
  .row.unreachable { opacity: 0.45; }

  .signal-bars { display: flex; align-items: flex-end; gap: 1px; height: 12px; }
  .bar { width: 3px; background-color: var(--border-color); border-radius: 1px; }
  .bar:nth-child(1) { height: 3px; }
  .bar:nth-child(2) { height: 6px; }
  .bar:nth-child(3) { height: 9px; }
  .bar:nth-child(4) { height: 12px; }
  .bar.filled { background-color: var(--bar-color); }

  .pairing-mode-badge {
    font-size: 0.7rem;
    color: var(--state-ok);
    border: 1px solid var(--state-ok);
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

  .battery-icon {
    width: 18px;
    height: 9px;
    border: 1px solid var(--text-muted);
    border-radius: 2px;
    overflow: hidden;
    position: relative;
    flex-shrink: 0;
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

  .battery-fill { height: 100%; transition: width 0.3s; }
</style>
