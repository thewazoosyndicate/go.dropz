<script>
  import { getSearchQuery, setSearchQuery, getSortBy, setSortBy } from '../lib/stores/ui.svelte.js';

  let query = $derived(getSearchQuery());
  let sort = $derived(getSortBy());
</script>

<div class="search-bar">
  <div class="search-input-wrapper">
    <i class="fas fa-search search-icon"></i>
    <input
      type="text"
      class="search-input"
      placeholder="Search cameras..."
      value={query}
      oninput={(e) => setSearchQuery(e.target.value)}
    />
    {#if query}
      <button class="clear-btn" onclick={() => setSearchQuery('')} aria-label="Clear search">
        <i class="fas fa-times"></i>
      </button>
    {/if}
  </div>
  <select class="sort-select" value={sort} onchange={(e) => setSortBy(e.target.value)}>
    <option value="signal">Signal</option>
    <option value="name">Name</option>
  </select>
</div>

<style>
  .search-bar {
    display: flex;
    gap: 8px;
    padding: 8px 14px;
    border-bottom: 1px solid var(--border-color);
  }

  .search-input-wrapper {
    flex: 1;
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: 8px;
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .search-input {
    width: 100%;
    padding: 6px 28px 6px 26px;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    background-color: var(--light-bg);
    color: var(--text-primary);
    font-size: 0.8rem;
    outline: none;
    transition: border-color 0.2s;
  }

  .search-input:focus {
    border-color: var(--primary-color);
  }

  .clear-btn {
    position: absolute;
    right: 6px;
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 0.7rem;
    padding: 2px;
  }

  .sort-select {
    padding: 5px 8px;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    background-color: var(--light-bg);
    color: var(--text-primary);
    font-size: 0.75rem;
    cursor: pointer;
    outline: none;
  }

  .sort-select:focus {
    border-color: var(--primary-color);
  }
</style>
