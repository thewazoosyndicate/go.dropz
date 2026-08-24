<script>
  // Day-grouped grid shared by the library and the camera catalog.
  import MediaCard from './MediaCard.svelte';
  import { groupItems } from '../lib/media.js';

  let {
    items = [],          // already filtered and sorted
    sortBy = 'date',
    newKeys = new Set(), // item keys to badge as new
    selectedKeys = new Set(),
    pendingKey = null,   // item whose preview is being fetched
    onToggle,
    onPlay,
    emptyText = 'Nothing matches this filter.',
  } = $props();

  let groups = $derived(groupItems(items, sortBy));
</script>

{#if items.length === 0}
  <div class="empty-state"><p>{emptyText}</p></div>
{:else}
  <div class="groups">
    {#each groups as group (group.key)}
      <div class="group">
        {#if group.label}
          <div class="group-head">
            <span class="group-label">{group.label}</span>
            <span class="group-meta">{group.meta}</span>
          </div>
        {/if}
        <div class="media-grid">
          {#each group.items as item (item.key)}
            <MediaCard {item} isNew={newKeys.has(item.key)} selected={selectedKeys.has(item.key)}
                       pending={pendingKey === item.key} {onToggle} {onPlay} />
          {/each}
        </div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .groups { display: flex; flex-direction: column; gap: 14px; }
  .group { display: flex; flex-direction: column; gap: 8px; }
  .group-head { display: flex; align-items: baseline; gap: 10px; }
  .group-label { font-size: 0.85rem; font-weight: 600; }
  .group-meta { font-size: 0.75rem; color: var(--text-muted); }

  .media-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 10px;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 120px;
    color: var(--text-muted);
    text-align: center;
  }
</style>
