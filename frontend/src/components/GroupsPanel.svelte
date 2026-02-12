<script>
  import GroupCard from './GroupCard.svelte';
  import { getGroups } from '../lib/stores/groups.svelte.js';

  let groups = $derived(getGroups());
</script>

<section class="panel">
  <div class="panel-header">
    <h2><i class="fas fa-layer-group"></i> Groups</h2>
    <span class="badge">{groups.length} groups</span>
  </div>
  <div class="panel-content">
    {#if groups.length === 0}
      <div class="empty-state">
        <img src="imgs/3_dropz.svg" alt="Dropz" class="empty-logo" />
        <p>No saved groups</p>
        <p class="hint">Save your managed cameras as a group from the Cameras tab</p>
      </div>
    {:else}
      <div class="cards-grid">
        {#each groups as group (group.id)}
          <GroupCard {group} />
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

  .badge {
    font-size: 0.8rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 3px 10px;
    border-radius: 12px;
  }

  .panel-content {
    flex: 1;
    padding: 12px;
    overflow-y: auto;
  }

  .cards-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
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

  .hint {
    font-size: 0.8rem;
    margin-top: 4px;
  }
</style>
