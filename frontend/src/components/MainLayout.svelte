<script>
  import ManagedPanel from './ManagedPanel.svelte';
  import DiscoveredPanel from './DiscoveredPanel.svelte';
  import VideoLibrary from './VideoLibrary.svelte';
  import ActivityPanel from './ActivityPanel.svelte';
  import { loadVideos } from '../lib/grpc/actions.js';
  import { getLibraryTarget, getActiveTab, setActiveTab, getLibraryLastVisit,
           setSettingsOpen, getSettingsOpen, getCameraSettingsTarget, getPlayerTarget } from '../lib/stores/ui.svelte.js';
  import { getSyncQueue, getNewFilePaths } from '../lib/stores/sync.svelte.js';

  let activeTab = $derived(getActiveTab());
  let videosLoaded = false;

  let newCount = $derived(getNewFilePaths(getLibraryLastVisit()).size);
  let queueLength = $derived(getSyncQueue().length);

  const TABS = [
    { id: 'cameras', icon: 'fa-video', label: 'Cameras' },
    { id: 'library', icon: 'fa-photo-film', label: 'Library' },
    { id: 'activity', icon: 'fa-wave-square', label: 'Activity' },
  ];

  // Card shortcuts land on the library tab; VideoLibrary consumes the target
  $effect(() => {
    if (getLibraryTarget()) switchTab('library');
  });

  function switchTab(tab) {
    setActiveTab(tab);
    if (tab === 'library' && !videosLoaded) {
      videosLoaded = true;
      loadVideos();
    }
  }

  // Ctrl/Cmd+1..3 switch tabs, Ctrl/Cmd+, opens settings; skipped while
  // an overlay has the keyboard or the user types in a field.
  function onKeydown(e) {
    const mod = e.ctrlKey || e.metaKey;
    if (!mod || e.altKey) return;
    const typing = ['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target?.tagName);
    if (typing) return;
    if (e.key === ',') {
      e.preventDefault();
      setSettingsOpen(true);
      return;
    }
    const n = Number(e.key);
    if (n >= 1 && n <= TABS.length && !getSettingsOpen() && !getCameraSettingsTarget() && !getPlayerTarget()) {
      e.preventDefault();
      switchTab(TABS[n - 1].id);
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="tab-bar" role="tablist">
  {#each TABS as tab, i (tab.id)}
    <button class="tab" role="tab" aria-selected={activeTab === tab.id}
            class:active={activeTab === tab.id} onclick={() => switchTab(tab.id)}
            title="{tab.label} (Ctrl+{i + 1})">
      <i class="fas {tab.icon}" aria-hidden="true"></i> {tab.label}
      {#if tab.id === 'library' && newCount > 0}
        <span class="count">{newCount} new</span>
      {:else if tab.id === 'activity' && queueLength > 0}
        <span class="dot" aria-label="Sync in progress"></span>
      {/if}
    </button>
  {/each}
</div>

{#if activeTab === 'cameras'}
  <main class="main-layout">
    <div class="managed-section">
      <ManagedPanel />
    </div>
    <div class="discovered-section">
      <DiscoveredPanel />
    </div>
  </main>
{:else if activeTab === 'library'}
  <main class="main-layout library-layout">
    <VideoLibrary />
  </main>
{:else}
  <main class="main-layout library-layout">
    <ActivityPanel />
  </main>
{/if}

<style>
  .tab-bar {
    display: flex;
    gap: 0;
    padding: 0 12px;
    border-bottom: 1px solid var(--border-color);
    background-color: var(--panel-bg);
    flex-shrink: 0;
  }

  .tab {
    padding: 10px 20px;
    border: none;
    background: none;
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    border-bottom: 2px solid transparent;
    transition: color 0.2s, border-color 0.2s;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .tab:hover {
    color: var(--text-primary);
  }

  .tab.active {
    color: var(--primary-color);
    border-bottom-color: var(--primary-color);
  }

  .count {
    font-size: 11px;
    font-weight: 600;
    color: white;
    background: var(--primary-color);
    border-radius: 8px;
    padding: 0 6px;
    line-height: 16px;
  }

  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--state-busy);
  }

  .main-layout {
    flex: 1;
    display: flex;
    gap: 12px;
    padding: 12px;
    overflow: hidden;
    min-height: 0;
  }

  .library-layout {
    flex-direction: column;
  }

  .managed-section {
    flex: 7;
    min-width: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .discovered-section {
    flex: 3;
    min-width: 280px;
    max-width: 420px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  @media (max-width: 900px) {
    .main-layout {
      flex-direction: column;
    }
    .discovered-section {
      max-width: none;
    }
  }
</style>
