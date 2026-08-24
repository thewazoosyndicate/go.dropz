<script>
  // Section header for one group on the Cameras tab: live summary,
  // fold toggle, and the group's actions. `group` is null for the
  // ungrouped section.
  import { renameGroup, deleteGroup, setGroupSync, moveCamerasToGroup, addToSyncQueue } from '../lib/grpc/actions.js';
  import { getAllDevices } from '../lib/stores/devices.svelte.js';
  import { openCameraSettings, addToast, isGroupCollapsed, toggleGroupCollapsed } from '../lib/stores/ui.svelte.js';
  import { getGroups } from '../lib/stores/groups.svelte.js';
  import { plural } from '../lib/format.js';
  import Badge from './ui/Badge.svelte';
  import IconButton from './ui/IconButton.svelte';
  import Menu from './ui/Menu.svelte';

  let { group = null, devices = [] } = $props();

  let id = $derived(group?.id || 'ungrouped');
  let collapsed = $derived(isGroupCollapsed(id));
  let inRange = $derived(devices.filter(d => d.isReachable));
  let syncing = $derived(devices.filter(d => d.isSyncing).length);
  let attention = $derived(devices.filter(d => d.lastSyncError && !d.isSyncing).length);
  let fresh = $derived(devices.reduce((n, d) => n + (d.newMediaCount || 0), 0));
  let otherGroups = $derived(getGroups().filter(g => g.id !== group?.id).length);

  let summary = $derived.by(() => {
    const parts = [`${inRange.length} of ${devices.length} in range`];
    if (syncing) parts.push(`${syncing} syncing`);
    if (attention) parts.push(`${attention} need${attention === 1 ? 's' : ''} attention`);
    else if (fresh) parts.push(`${fresh} new on camera`);
    return parts.join(' · ');
  });

  // Cameras switched on nearby that are not in this group yet: on a
  // shoot day that is the rig
  let nearbyOthers = $derived(Object.values(getAllDevices())
    .filter(d => d.isManaged && d.isPaired && d.isReachable && d.id && !devices.some(x => x.id === d.id)));

  let renaming = $state(false);
  let draft = $state('');
  let nameInput = $state(null);

  function startRename() {
    draft = group.name;
    renaming = true;
    setTimeout(() => nameInput?.focus(), 0);
  }

  async function commitRename() {
    if (!renaming) return;
    renaming = false;
    const name = draft.trim();
    if (!name || name === group.name) return;
    try {
      await renameGroup(group.id, name, group.cameraIds);
    } catch (_) {}
  }

  function onRenameKey(e) {
    if (e.key === 'Enter') commitRename();
    else if (e.key === 'Escape') renaming = false;
  }

  function syncGroup() {
    const targets = inRange.filter(d => !d.isSyncing);
    if (targets.length === 0) {
      addToast('No camera of this group is in range', 'info');
      return;
    }
    targets.forEach(d => addToSyncQueue(d.macAddress));
    addToast(`Queued ${plural(targets.length, 'camera')}`, 'info');
  }

  function groupSettings() {
    const reference = devices.find(d => d.isReachable && d.isPaired);
    if (!reference) {
      addToast('No reachable camera in this group', 'error');
      return;
    }
    openCameraSettings({ type: 'group', id: group.id, name: group.name, referenceCameraId: reference.id });
  }

  async function addNearby() {
    try {
      await moveCamerasToGroup(nearbyOthers.map(d => d.id), group.id);
      addToast(`Added ${plural(nearbyOthers.length, 'camera')} to ${group.name}`, 'success');
    } catch (_) {}
  }

  let menuItems = $derived(!group ? [] : [
    { label: 'Rename', icon: 'fa-pen', onclick: startRename },
    group.syncPaused
      ? { label: 'Resume auto-sync', icon: 'fa-play', onclick: () => setGroupSync(group.id, false) }
      : { label: 'Pause auto-sync', icon: 'fa-pause', onclick: () => setGroupSync(group.id, true) },
    { label: 'Use only this group', icon: 'fa-bullseye', disabled: otherGroups === 0,
      onclick: () => setGroupSync(group.id, false, true) },
    { label: 'Add cameras in range', icon: 'fa-plus', disabled: nearbyOthers.length === 0, onclick: addNearby },
    { label: 'Delete group', icon: 'fa-trash-alt', danger: true, confirm: true, onclick: () => deleteGroup(group.id) },
  ]);
</script>

<div class="group-header" class:paused={group?.syncPaused}>
  <button class="fold" onclick={() => toggleGroupCollapsed(id)} aria-expanded={!collapsed}
          title={collapsed ? 'Show cameras' : 'Hide cameras'}>
    <i class="fas fa-chevron-down" class:collapsed aria-hidden="true"></i>
  </button>

  {#if group && renaming}
    <input class="name-input" bind:this={nameInput} bind:value={draft} maxlength="40"
           aria-label="Group name" onkeydown={onRenameKey} onblur={commitRename} />
  {:else if group}
    <button class="name" onclick={startRename} title="Rename">{group.name}</button>
  {:else}
    <span class="name muted">Ungrouped</span>
  {/if}

  {#if group?.syncPaused}
    <Badge tone="idle">Paused</Badge>
  {:else if syncing > 0}
    <Badge tone="busy" pulse>Syncing</Badge>
  {/if}

  <span class="summary">{summary}</span>

  {#if group}
    <div class="actions">
      <IconButton icon="fa-rotate" title="Sync every camera of this group in range" disabled={inRange.length === 0} onclick={syncGroup} />
      <IconButton icon="fa-sliders-h" title="Apply settings to this group" disabled={inRange.length === 0} onclick={groupSettings} />
      <Menu items={menuItems} title="Group actions" />
    </div>
  {/if}
</div>

<style>
  .group-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 4px 0 4px 2px;
    min-height: 40px;
  }

  .group-header.paused .name, .group-header.paused .summary { color: var(--text-muted); }

  .fold {
    width: 28px;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: 6px;
  }
  .fold:hover { background: var(--hover-bg); color: var(--text-primary); }
  .fold i { transition: transform 0.15s; font-size: 0.75rem; }
  .fold i.collapsed { transform: rotate(-90deg); }

  .name {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-primary);
    background: none;
    border: none;
    padding: 0;
    cursor: text;
  }
  .name.muted { color: var(--text-muted); cursor: default; }

  .name-input {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-primary);
    background: var(--light-bg);
    border: 1px solid var(--primary-color);
    border-radius: 6px;
    padding: 2px 6px;
    width: 220px;
  }

  .summary { font-size: 0.78rem; color: var(--text-secondary); }

  .actions { margin-left: auto; display: flex; align-items: center; gap: 6px; }
  .actions :global(.trigger) { width: 32px; min-height: 32px; }
</style>
