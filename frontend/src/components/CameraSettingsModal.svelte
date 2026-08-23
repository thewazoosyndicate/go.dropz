<script>
  import { getCameraSettingsTarget, closeCameraSettings, addToast, addLog } from '../lib/stores/ui.svelte.js';
  import { fetchCameraSettings, applyCameraSettings } from '../lib/grpc/actions.js';
  import { getAllDevices } from '../lib/stores/devices.svelte.js';

  let target = $derived(getCameraSettingsTarget());

  let loading = $state(false);
  let loadError = $state('');
  let settings = $state([]);
  let pending = $state({});
  let applying = $state(false);
  let cameraResults = $state([]);
  let loadedFor = $state(null);

  let dirty = $derived(Object.keys(pending).length > 0);
  // Settings with a single option are informational; the camera offers no choice
  let editable = $derived(settings.filter(s => s.options.length > 1));
  let readOnly = $derived(settings.filter(s => s.options.length <= 1));

  $effect(() => {
    if (target && loadedFor !== target.referenceCameraId) {
      loadedFor = target.referenceCameraId;
      load();
    }
    if (!target) {
      loadedFor = null;
      settings = [];
      pending = {};
      cameraResults = [];
      loadError = '';
    }
  });

  async function load() {
    loading = true;
    loadError = '';
    cameraResults = [];
    pending = {};
    try {
      const result = await fetchCameraSettings(target.referenceCameraId, true);
      settings = result.settings;
      if (settings.length === 0) loadError = 'Camera returned no settings.';
    } catch (e) {
      loadError = e.message || 'Failed to read settings';
      settings = [];
    } finally {
      loading = false;
    }
  }

  function handleSelect(setting, event) {
    const value = Number(event.target.value);
    if (value === setting.value) delete pending[setting.id];
    else pending[setting.id] = value;
    pending = { ...pending };
  }

  function cameraName(id) {
    const d = Object.values(getAllDevices()).find(d => d.id === id);
    return d ? (d.wifiSsid?.trim()?.substring(0, 12) || d.name || id) : id;
  }

  async function apply() {
    applying = true;
    cameraResults = [];
    const changes = Object.entries(pending).map(([id, value]) => ({ id: Number(id), value }));
    try {
      const results = await applyCameraSettings({
        cameraId: target.type === 'camera' ? target.id : '',
        groupId: target.type === 'group' ? target.id : '',
        changes,
      });
      cameraResults = results;
      const failures = results.filter(r => r.error || r.results.some(x => x.error));
      if (failures.length === 0) {
        addToast('Settings applied', 'success');
        addLog(`Settings applied to ${target.name}`, 'success');
        pending = {};
        if (target.type === 'camera') await load();
      } else {
        addToast('Some settings were not applied', 'warning');
      }
    } catch (e) {
      addToast('Failed to apply settings', 'error');
      addLog(`Apply settings failed: ${e.message}`, 'error');
    } finally {
      applying = false;
    }
  }

  function handleOverlayClick(event) {
    if (event.target === event.currentTarget && !applying) closeCameraSettings();
  }

  function settingLabelFor(result, settingId) {
    const s = settings.find(x => x.id === settingId);
    return s ? s.name : `Setting ${settingId}`;
  }
</script>

{#if target}
  <div class="modal-overlay" onclick={handleOverlayClick}>
    <div class="modal">
      <div class="modal-header">
        <h2>
          <i class="fas fa-sliders-h"></i>
          {target.name}
          {#if target.type === 'group'}<span class="group-note">group</span>{/if}
        </h2>
        <button class="close-btn" onclick={closeCameraSettings} aria-label="Close camera settings">
          <i class="fas fa-times"></i>
        </button>
      </div>

      <div class="modal-body">
        {#if target.type === 'group'}
          <p class="hint">
            Options read from {cameraName(target.referenceCameraId)}.
            Each camera applies what its model supports; results are per camera.
          </p>
        {/if}

        {#if loading}
          <div class="state-line"><i class="fas fa-spinner fa-spin"></i> Reading from camera over Bluetooth...</div>
        {:else if loadError}
          <div class="state-line error">
            <i class="fas fa-exclamation-triangle"></i> {loadError}
            <button class="btn btn-outline" onclick={load}>Retry</button>
          </div>
        {:else}
          {#each editable as setting (setting.id)}
            <div class="setting-row" class:changed={setting.id in pending}>
              <label for="cs-{setting.id}">{setting.name}</label>
              <select id="cs-{setting.id}" value={pending[setting.id] ?? setting.value}
                      onchange={(e) => handleSelect(setting, e)} disabled={applying}>
                {#if !setting.options.some(o => o.value === setting.value)}
                  <option value={setting.value}>{setting.valueName}</option>
                {/if}
                {#each setting.options as opt (opt.value)}
                  <option value={opt.value}>{opt.name}</option>
                {/each}
              </select>
            </div>
          {/each}

          {#if readOnly.length > 0}
            <details class="readonly-block">
              <summary>{readOnly.length} settings without selectable options</summary>
              {#each readOnly as setting (setting.id)}
                <div class="setting-row readonly">
                  <span>{setting.name}</span>
                  <span class="value">{setting.valueName}</span>
                </div>
              {/each}
            </details>
          {/if}
        {/if}

        {#if cameraResults.length > 0}
          <div class="results">
            {#each cameraResults as r (r.cameraId)}
              {#if r.error}
                <div class="result-line error">
                  <i class="fas fa-times-circle"></i> {cameraName(r.cameraId)}: {r.error}
                </div>
              {:else if r.results.some(x => x.error)}
                {#each r.results.filter(x => x.error) as f (f.id)}
                  <div class="result-line warning">
                    <i class="fas fa-exclamation-triangle"></i>
                    {cameraName(r.cameraId)}: {settingLabelFor(r, f.id)}: {f.error}
                  </div>
                {/each}
              {:else}
                <div class="result-line success">
                  <i class="fas fa-check-circle"></i> {cameraName(r.cameraId)}: applied
                </div>
              {/if}
            {/each}
          </div>
        {/if}
      </div>

      <div class="modal-footer">
        <button class="btn btn-outline" onclick={load} disabled={loading || applying}>
          <i class="fas fa-rotate"></i> Refresh
        </button>
        <button class="btn btn-primary" onclick={apply} disabled={!dirty || applying || loading}>
          {#if applying}<i class="fas fa-spinner fa-spin"></i> Applying...{:else}Apply{/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background-color: var(--panel-bg);
    border-radius: 10px;
    width: min(560px, 92vw);
    max-height: 84vh;
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow-md);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 18px;
    border-bottom: 1px solid var(--border-color);
  }

  .modal-header h2 {
    font-size: 1.1rem;
    margin: 0;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .group-note {
    font-size: 0.75rem;
    padding: 2px 8px;
    border-radius: 10px;
    background-color: var(--primary-color);
    color: white;
  }

  .close-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 1.1rem;
  }

  .modal-body {
    padding: 14px 18px;
    overflow-y: auto;
    flex: 1;
  }

  .hint {
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin: 0 0 10px;
  }

  .state-line {
    padding: 18px 0;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .state-line.error { color: var(--danger-color); }

  .setting-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding: 7px 0;
    border-bottom: 1px solid var(--border-color);
  }

  .setting-row label { color: var(--text-primary); }

  .setting-row.changed label { color: var(--primary-color); font-weight: 600; }

  .setting-row select {
    background-color: var(--panel-bg);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 4px 8px;
    max-width: 55%;
  }

  .setting-row.readonly {
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .readonly-block { margin-top: 10px; }

  .readonly-block summary {
    cursor: pointer;
    color: var(--text-secondary);
    font-size: 0.85rem;
    padding: 6px 0;
  }

  .results {
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .result-line {
    font-size: 0.88rem;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .result-line.success { color: var(--secondary-color); }
  .result-line.warning { color: var(--warning-color); }
  .result-line.error { color: var(--danger-color); }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 12px 18px;
    border-top: 1px solid var(--border-color);
  }

  .btn {
    padding: 7px 16px;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .btn:disabled { opacity: 0.5; cursor: default; }

  .btn-primary {
    background-color: var(--primary-color);
    color: white;
  }

  .btn-outline {
    background: none;
    border: 1px solid var(--border-color);
    color: var(--text-primary);
  }
</style>
