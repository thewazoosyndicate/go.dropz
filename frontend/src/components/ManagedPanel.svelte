<script>
  import ManagedCard from './ManagedCard.svelte';
  import GroupHeader from './GroupHeader.svelte';
  import IconButton from './ui/IconButton.svelte';
  import Menu from './ui/Menu.svelte';
  import { getManagedDevices, displayName } from '../lib/stores/devices.svelte.js';
  import { getGroups } from '../lib/stores/groups.svelte.js';
  import { createGroup, syncAllManaged } from '../lib/grpc/actions.js';
  import { isGroupCollapsed, addToast } from '../lib/stores/ui.svelte.js';
  import { plural } from '../lib/format.js';

  let devices = $derived(Object.values(getManagedDevices()).sort((a, b) =>
    displayName(a).localeCompare(displayName(b))
  ));
  let anyInRange = $derived(devices.some(d => d.isReachable && !d.isSyncing));
  let inRangeIds = $derived(devices.filter(d => d.isReachable && d.id).map(d => d.id));

  // Active groups first, paused ones after, ungrouped last. A group with
  // no managed camera left still shows, so it can be filled or deleted.
  let sections = $derived.by(() => {
    const groups = [...getGroups()].sort((a, b) =>
      (a.syncPaused - b.syncPaused) || a.name.localeCompare(b.name));
    const placed = new Set();
    const out = groups.map(g => {
      const members = devices.filter(d => g.cameraIds.includes(d.id));
      members.forEach(d => placed.add(d.macAddress));
      return { id: g.id, group: g, devices: members };
    });
    const loose = devices.filter(d => !placed.has(d.macAddress));
    if (loose.length > 0 || out.length === 0) out.push({ id: 'ungrouped', group: null, devices: loose });
    return out;
  });

  // New group: pick the seed, then name it inline
  let naming = $state(null); // 'range' | 'all' | 'empty'
  let groupName = $state('');
  let nameInput = $state(null);

  function startNaming(seed) {
    naming = seed;
    groupName = '';
    setTimeout(() => nameInput?.focus(), 0);
  }

  async function submitGroupName() {
    const name = groupName.trim();
    if (!name) return;
    const ids = naming === 'range' ? inRangeIds : naming === 'all' ? devices.filter(d => d.id).map(d => d.id) : [];
    naming = null;
    try {
      await createGroup(name, ids);
      addToast(ids.length ? `Group "${name}" created with ${plural(ids.length, 'camera')}` : `Group "${name}" created`, 'success');
    } catch (_) {}
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') submitGroupName();
    else if (e.key === 'Escape') naming = null;
  }

  let newGroupItems = $derived([
    { label: `From cameras in range (${inRangeIds.length})`, icon: 'fa-wifi', disabled: inRangeIds.length === 0, onclick: () => startNaming('range') },
    { label: `From all managed (${devices.length})`, icon: 'fa-video', disabled: devices.length === 0, onclick: () => startNaming('all') },
    { label: 'Empty group', icon: 'fa-folder-plus', onclick: () => startNaming('empty') },
  ]);
</script>

<section class="panel">
  <div class="panel-header">
    <h2><i class="fas fa-video" aria-hidden="true"></i> Managed</h2>
    <div class="header-actions">
      <span class="badge">{plural(devices.length, 'camera')}</span>
      {#if naming}
        <div class="name-input-row">
          <input class="name-input" type="text" placeholder="Group name" bind:this={nameInput}
                 bind:value={groupName} onkeydown={handleKeydown} aria-label="New group name" />
          <IconButton icon="fa-check" title="Create group" onclick={submitGroupName} />
          <IconButton icon="fa-times" title="Cancel" onclick={() => naming = null} />
        </div>
      {:else}
        <IconButton icon="fa-rotate" title="Sync every camera in range" disabled={!anyInRange} onclick={syncAllManaged} />
        <Menu items={newGroupItems} title="New group" icon="fa-folder-plus" />
      {/if}
    </div>
  </div>
  <div class="panel-content">
    {#if devices.length === 0 && getGroups().length === 0}
      <div class="empty-state">
        <img src="imgs/3_dropz.svg" alt="Dropz" class="empty-logo" />
        <p>No managed cameras</p>
        <p class="hint">Turn on a GoPro nearby and press Pair in the Nearby list.</p>
      </div>
    {:else}
      {#each sections as section (section.id)}
        <div class="section" class:paused={section.group?.syncPaused}>
          <GroupHeader group={section.group} devices={section.devices} />
          {#if !isGroupCollapsed(section.id)}
            {#if section.devices.length === 0}
              <p class="empty-group">No cameras yet. Use the group menu to add the cameras in range, or move cameras here from their card menu.</p>
            {:else}
              <div class="cards-grid">
                {#each section.devices as device (device.macAddress)}
                  <ManagedCard {device} />
                {/each}
              </div>
            {/if}
          {/if}
        </div>
      {/each}
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
    padding: 12px 16px;
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

  .header-actions { display: flex; align-items: center; gap: 8px; }
  .header-actions :global(.trigger) { width: 32px; min-height: 32px; }

  .badge {
    font-size: 0.8rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 3px 10px;
    border-radius: 12px;
  }

  .name-input-row { display: flex; align-items: center; gap: 4px; }

  .name-input {
    background: var(--light-bg);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 6px 8px;
    font-size: 0.8rem;
    color: var(--text-primary);
    width: 180px;
    height: 32px;
  }

  .name-input:focus { border-color: var(--primary-color); outline: none; }

  .panel-content {
    flex: 1;
    padding: 8px 12px 12px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .section { display: flex; flex-direction: column; gap: 6px; padding-bottom: 8px; }
  .section.paused .cards-grid { opacity: 0.85; }

  .cards-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 12px;
  }

  .empty-group {
    font-size: 0.78rem;
    color: var(--text-muted);
    padding: 0 0 4px 40px;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-muted);
    text-align: center;
  }

  .empty-logo { height: 80px; opacity: 0.6; margin-bottom: 12px; }
  .hint { font-size: 0.8rem; margin-top: 4px; }
</style>
