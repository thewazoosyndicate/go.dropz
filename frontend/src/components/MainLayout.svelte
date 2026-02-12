<script>
  import ManagedPanel from './ManagedPanel.svelte';
  import DiscoveredPanel from './DiscoveredPanel.svelte';
</script>

<main class="main-layout">
  <div class="managed-section">
    <ManagedPanel />
  </div>
  <div class="discovered-section">
    <DiscoveredPanel />
  </div>
</main>

<style>
  .main-layout {
    flex: 1;
    display: flex;
    gap: 12px;
    padding: 12px;
    overflow: hidden;
    min-height: 0;
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
