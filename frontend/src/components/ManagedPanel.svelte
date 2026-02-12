<script>
  import ManagedCard from './ManagedCard.svelte';
  import { getManagedDevices } from '../lib/stores/devices.svelte.js';
  import { saveManagedAsGroup } from '../lib/grpc/actions.js';

  let devices = $derived(Object.values(getManagedDevices()).sort((a, b) =>
    (a.name || '').localeCompare(b.name || '')
  ));

  let showNameInput = $state(false);
  let groupName = $state('');

  function handleSaveClick() {
    showNameInput = true;
    groupName = '';
  }

  function submitGroupName() {
    if (groupName.trim()) {
      saveManagedAsGroup(groupName.trim());
      showNameInput = false;
      groupName = '';
    }
  }

  function cancelGroupName() {
    showNameInput = false;
    groupName = '';
  }

  function handleKeydown(e) {
    if (e.key === 'Enter') submitGroupName();
    else if (e.key === 'Escape') cancelGroupName();
  }
</script>

<section class="panel">
  <div class="panel-header">
    <h2><i class="fas fa-video"></i> Managed</h2>
    <div class="header-actions">
      <span class="badge">{devices.length} cameras</span>
      {#if devices.length > 0}
        {#if showNameInput}
          <div class="name-input-row">
            <!-- svelte-ignore a11y_autofocus -->
            <input class="name-input" type="text" placeholder="Group name"
                   bind:value={groupName} onkeydown={handleKeydown} autofocus />
            <button class="save-group-btn" onclick={submitGroupName} title="Save">
              <i class="fas fa-check"></i>
            </button>
            <button class="save-group-btn" onclick={cancelGroupName} title="Cancel">
              <i class="fas fa-times"></i>
            </button>
          </div>
        {:else}
          <button class="save-group-btn" onclick={handleSaveClick} title="Save as Group">
            <i class="fas fa-save"></i>
          </button>
        {/if}
      {/if}
    </div>
  </div>
  <div class="panel-content">
    {#if devices.length === 0}
      <div class="empty-state">
        <img src="imgs/3_dropz.svg" alt="Dropz" class="empty-logo" />
        <p>No managed cameras</p>
      </div>
    {:else}
      <div class="cards-grid">
        {#each devices as device (device.macAddress)}
          <ManagedCard {device} />
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

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .badge {
    font-size: 0.8rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 3px 10px;
    border-radius: 12px;
  }

  .save-group-btn {
    background: none;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 4px 8px;
    cursor: pointer;
    color: var(--text-secondary);
    font-size: 0.8rem;
    transition: all 0.2s;
  }

  .save-group-btn:hover {
    border-color: var(--text-secondary);
    color: var(--text-primary);
  }

  .name-input-row {
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .name-input {
    background: var(--light-bg);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 4px 8px;
    font-size: 0.8rem;
    color: var(--text-primary);
    width: 140px;
    outline: none;
  }

  .name-input:focus {
    border-color: var(--primary-color);
  }

  .panel-content {
    flex: 1;
    padding: 12px;
    overflow-y: auto;
  }

  .cards-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 12px;
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

  .empty-logo {
    height: 80px;
    opacity: 0.6;
    margin-bottom: 12px;
  }
</style>
