<script>
  // Filter chips and sort controls shared by both media views. The
  // source adds its own chips before the kind filters (leading) and its
  // own buttons after the sort (trailing).
  import IconButton from './ui/IconButton.svelte';

  let { kind = $bindable('all'), sortBy = $bindable('date'), sortAsc = $bindable(false), leading, trailing } = $props();

  function toggleKind(k) {
    kind = kind === k ? 'all' : k;
  }
</script>

<div class="toolbar">
  <div class="chips">
    {#if leading}{@render leading()}{/if}
    <button class="chip" class:active={kind === 'video'} onclick={() => toggleKind('video')}>Videos</button>
    <button class="chip" class:active={kind === 'photo'} onclick={() => toggleKind('photo')}>Photos</button>
  </div>
  <div class="trailing">
    <select bind:value={sortBy} aria-label="Sort by">
      <option value="date">Date</option>
      <option value="name">Name</option>
      <option value="size">Size</option>
    </select>
    <IconButton icon={sortAsc ? 'fa-arrow-up-short-wide' : 'fa-arrow-down-wide-short'}
                title="Toggle sort direction" onclick={() => sortAsc = !sortAsc} />
    {#if trailing}{@render trailing()}{/if}
  </div>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    flex-wrap: wrap;
  }

  .chips { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
  .trailing { display: flex; align-items: center; gap: 8px; }

  /* Chip style is global to the toolbar so a source's leading chips match */
  .toolbar :global(.chip) {
    background: none;
    border: 1px solid var(--border-color);
    border-radius: 14px;
    padding: 4px 12px;
    min-height: 30px;
    font-size: 0.82rem;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: color 0.15s, background-color 0.15s;
  }
  .toolbar :global(.chip:hover:not(:disabled)) { color: var(--text-primary); }
  .toolbar :global(.chip.active) {
    background-color: var(--primary-color);
    border-color: var(--primary-color);
    color: white;
  }
  .toolbar :global(.chip:disabled) { opacity: 0.5; cursor: default; }
  .toolbar :global(.chip-dot) { width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
  .toolbar :global(.muted) { font-size: 0.78rem; color: var(--text-muted); }

  select {
    background-color: var(--panel-bg);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 5px 8px;
    height: 32px;
    font-size: 0.85rem;
  }
</style>
