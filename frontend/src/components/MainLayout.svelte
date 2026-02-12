<script>
  import ManagedPanel from './ManagedPanel.svelte';
  import DiscoveredPanel from './DiscoveredPanel.svelte';
  import VideoLibrary from './VideoLibrary.svelte';
  import GroupsPanel from './GroupsPanel.svelte';
  import { loadVideos, loadGroups } from '../lib/grpc/actions.js';

  let activeTab = $state('cameras');
  let videosLoaded = false;
  let groupsLoaded = false;

  function switchTab(tab) {
    activeTab = tab;
    if (tab === 'library' && !videosLoaded) {
      videosLoaded = true;
      loadVideos();
    }
    if (tab === 'groups' && !groupsLoaded) {
      groupsLoaded = true;
      loadGroups();
    }
  }
</script>

<div class="tab-bar">
  <button class="tab" class:active={activeTab === 'cameras'} onclick={() => switchTab('cameras')}>
    <i class="fas fa-video"></i> Cameras
  </button>
  <button class="tab" class:active={activeTab === 'library'} onclick={() => switchTab('library')}>
    <i class="fas fa-photo-film"></i> Library
  </button>
  <button class="tab" class:active={activeTab === 'groups'} onclick={() => switchTab('groups')}>
    <i class="fas fa-layer-group"></i> Groups
  </button>
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
    <GroupsPanel />
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
    transition: all 0.2s;
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
