<script>
  import { loadGroup, deleteGroup } from '../lib/grpc/actions.js';
  import { getAllDevices } from '../lib/stores/devices.svelte.js';
  import { openCameraSettings, addToast } from '../lib/stores/ui.svelte.js';

  let { group } = $props();
  let confirming = $state(false);

  let cameraNames = $derived.by(() => {
    const devices = getAllDevices();
    return group.cameraIds.map(id => {
      const d = Object.values(devices).find(d => d.id === id);
      if (!d) return 'Unknown';
      return d.wifiSsid?.trim()?.substring(0, 12) || d.name || 'Unknown';
    });
  });

  function handleLoad() {
    loadGroup(group.id);
  }

  function handleSettings() {
    // Options are read from one reachable member; each camera applies
    // what its model supports.
    const devices = Object.values(getAllDevices());
    const reference = group.cameraIds.find(id =>
      devices.some(d => d.id === id && d.isReachable && d.isPaired)
    );
    if (!reference) {
      addToast('No reachable camera in this group', 'error');
      return;
    }
    openCameraSettings({
      type: 'group',
      id: group.id,
      name: group.name,
      referenceCameraId: reference,
    });
  }

  function handleDelete() {
    if (!confirming) {
      confirming = true;
      setTimeout(() => confirming = false, 3000);
      return;
    }
    deleteGroup(group.id);
    confirming = false;
  }
</script>

<div class="card">
  <div class="card-header">
    <span class="name">{group.name}</span>
    <span class="count">{group.cameraIds.length} cameras</span>
  </div>
  <div class="card-body">
    <div class="camera-list">
      {#each cameraNames as name}
        <span class="camera-tag">{name}</span>
      {/each}
    </div>
  </div>
  <div class="card-actions">
    <button class="btn btn-primary" onclick={handleLoad}>Load</button>
    <button class="btn btn-outline" onclick={handleSettings}
            title="Group settings" aria-label="Group settings">
      <i class="fas fa-sliders-h"></i>
    </button>
    <button class="btn {confirming ? 'btn-danger' : 'btn-outline'}" onclick={handleDelete}>
      {confirming ? 'Confirm?' : 'Delete'}
    </button>
  </div>
</div>

<style>
  .card {
    background-color: var(--panel-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    border-left: 4px solid var(--primary-color);
    transition: all 0.2s;
  }

  .card:hover {
    box-shadow: var(--shadow-md);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .name {
    font-weight: 600;
    font-size: 1.05rem;
    color: var(--text-primary);
  }

  .count {
    font-size: 0.75rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 2px 8px;
    border-radius: 10px;
  }

  .card-body {
    flex: 1;
  }

  .camera-list {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .camera-tag {
    font-size: 0.7rem;
    color: var(--text-secondary);
    background-color: var(--light-bg);
    padding: 2px 8px;
    border-radius: 4px;
  }

  .card-actions {
    display: flex;
    gap: 8px;
    margin-top: auto;
  }

  .btn {
    flex: 1;
    padding: 8px 12px;
    border: none;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    color: white;
    min-height: 36px;
    transition: all 0.2s;
  }

  .btn:hover { transform: translateY(-1px); }
  .btn:active { transform: translateY(0); }

  .btn-primary { background-color: var(--primary-color); }
  .btn-primary:hover { background-color: var(--primary-dark); }

  .btn-danger { background-color: var(--danger-color); }
  .btn-danger:hover { background-color: var(--danger-dark); }

  .btn-outline {
    background-color: transparent;
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
  }
  .btn-outline:hover {
    border-color: var(--text-secondary);
  }
</style>
