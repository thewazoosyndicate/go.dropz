<script>
  import { getSettingsOpen, setSettingsOpen } from '../lib/stores/ui.svelte.js';
  import { getAppConfig } from '../lib/stores/config.svelte.js';
  import { updateSetting, resetAllSettings } from '../lib/grpc/actions.js';
  import Toggle from './ui/Toggle.svelte';
  import Button from './ui/Button.svelte';
  import IconButton from './ui/IconButton.svelte';

  const { ipcRenderer } = window.require('electron');

  let isOpen = $derived(getSettingsOpen());
  let config = $derived(getAppConfig());
  let advancedOpen = $state(false);
  let confirmingReset = $state(false);

  // Non-linear slider stops
  const MEDIA_AGE_STOPS = [1,2,3,4,5,6,7,8,9,10,20,30,40,50,60,70,80,90,120,150,180,210,240,270,300,330,360];
  const STATUS_CHECK_STOPS = [0,60,120,300,600,900,1800,2700,3600];

  function formatValue(value, setting) {
    if (setting === 'days_threshold') return value === 1 ? '1 day' : `${value} days`;
    if (setting === 'status_check_interval_seconds' && value === 0) return 'Off';
    if (value >= 3600) return (value / 3600) + 'h';
    if (value >= 120) return Math.round(value / 60) + ' min';
    return value + ' s';
  }

  function sliderToValue(sliderVal, setting) {
    if (setting === 'days_threshold') return MEDIA_AGE_STOPS[parseInt(sliderVal)] ?? 7;
    if (setting === 'status_check_interval_seconds') return STATUS_CHECK_STOPS[parseInt(sliderVal)] ?? 300;
    return parseInt(sliderVal);
  }

  function valueToSlider(value, setting) {
    if (setting === 'days_threshold') {
      let idx = MEDIA_AGE_STOPS.indexOf(value);
      if (idx === -1) idx = MEDIA_AGE_STOPS.findIndex(s => s >= value) || 0;
      return idx;
    }
    if (setting === 'status_check_interval_seconds') {
      let idx = STATUS_CHECK_STOPS.indexOf(value);
      if (idx === -1) idx = STATUS_CHECK_STOPS.findIndex(s => s >= value) || 0;
      return idx;
    }
    return value;
  }

  function handleSlider(event, setting) {
    updateSetting(setting, sliderToValue(event.target.value, setting));
  }

  function handleReset() {
    if (!confirmingReset) {
      confirmingReset = true;
      setTimeout(() => confirmingReset = false, 4000);
      return;
    }
    confirmingReset = false;
    resetAllSettings();
  }

  function browseFolderClick() {
    ipcRenderer.send('open-folder-dialog');
  }

  // Listen for folder selection
  ipcRenderer.on('selected-folder', (event, path) => {
    if (path) updateSetting('destination_folder', path);
  });

  function close() {
    setSettingsOpen(false);
    confirmingReset = false;
  }

  function handleOverlayClick(event) {
    if (event.target === event.currentTarget) close();
  }

  function onKeydown(e) {
    if (isOpen && e.key === 'Escape') close();
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if isOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="modal-overlay" onclick={handleOverlayClick}>
    <div class="modal" role="dialog" aria-modal="true" aria-labelledby="settings-title">
      <div class="modal-header">
        <h2 id="settings-title"><i class="fas fa-cog" aria-hidden="true"></i> Settings</h2>
        <IconButton icon="fa-times" title="Close settings" onclick={close} />
      </div>

      {#if config}
        <div class="modal-body">
          <div class="settings-section">
            <h3>Sync</h3>

            <div class="setting-item toggle-row">
              <div>
                <span class="setting-label">Check on return</span>
                <p class="setting-hint">Looks for new clips when a camera comes back in range</p>
              </div>
              <Toggle checked={config.checkOnReturn} onchange={(v) => updateSetting('check_on_return', v)} />
            </div>

            <div class="setting-item slider-row">
              <div class="setting-label-row">
                <span>Check every</span>
                <span class="setting-value">{formatValue(config.statusCheckIntervalSeconds, 'status_check_interval_seconds')}</span>
              </div>
              <p class="setting-hint">Pings cameras in range over Bluetooth to spot new clips</p>
              <input type="range" min="0" max="{STATUS_CHECK_STOPS.length - 1}" step="1"
                     aria-label="Check interval"
                     value={valueToSlider(config.statusCheckIntervalSeconds, 'status_check_interval_seconds')}
                     oninput={(e) => handleSlider(e, 'status_check_interval_seconds')} />
            </div>

            <div class="setting-item slider-row">
              <div class="setting-label-row">
                <span>Download clips from the last</span>
                <span class="setting-value">{formatValue(config.daysThreshold, 'days_threshold')}</span>
              </div>
              <input type="range" min="0" max="{MEDIA_AGE_STOPS.length - 1}" step="1"
                     aria-label="Media age"
                     value={valueToSlider(config.daysThreshold, 'days_threshold')}
                     oninput={(e) => handleSlider(e, 'days_threshold')} />
            </div>
          </div>

          <div class="settings-section">
            <h3>Storage</h3>
            <div class="setting-item">
              <span class="setting-label">Library folder</span>
              <p class="setting-hint">One subfolder per camera</p>
              <div class="folder-control">
                <input type="text" class="folder-input" value={config.destinationFolder || ''} readonly aria-label="Library folder" />
                <Button icon="fa-folder-open" onclick={browseFolderClick}>Change</Button>
              </div>
            </div>
          </div>

          <div class="settings-section">
            <button class="section-toggle" onclick={() => advancedOpen = !advancedOpen} aria-expanded={advancedOpen}>
              <h3>Advanced</h3>
              <i class="fas fa-chevron-down" class:open={advancedOpen} aria-hidden="true"></i>
            </button>

            {#if advancedOpen}
              <div class="setting-item toggle-row">
                <div>
                  <span class="setting-label">Set camera clock</span>
                  <p class="setting-hint">Syncs the camera time to this computer on connect</p>
                </div>
                <Toggle checked={config.setTimeEnabled} onchange={(v) => updateSetting('set_time_enabled', v)} />
              </div>

              <div class="setting-item toggle-row">
                <div>
                  <span class="setting-label">Turbo transfer</span>
                  <p class="setting-hint">Faster on some setups, slower on others. Off is the safe default.</p>
                </div>
                <Toggle checked={config.turboEnabled} onchange={(v) => updateSetting('turbo_enabled', v)} />
              </div>

              <div class="setting-item slider-row">
                <div class="setting-label-row">
                  <span>Bluetooth scan interval</span>
                  <span class="setting-value">{config.scanIntervalSeconds} s</span>
                </div>
                <input type="range" min="5" max="120" step="5" aria-label="Scan interval"
                       value={config.scanIntervalSeconds}
                       oninput={(e) => handleSlider(e, 'scan_interval_seconds')} />
              </div>

              <div class="setting-item slider-row">
                <div class="setting-label-row">
                  <span>Wi-Fi connect timeout</span>
                  <span class="setting-value">{config.connectTimeoutSeconds} s</span>
                </div>
                <input type="range" min="5" max="60" step="5" aria-label="Connect timeout"
                       value={config.connectTimeoutSeconds}
                       oninput={(e) => handleSlider(e, 'connect_timeout_seconds')} />
              </div>

              <div class="setting-item slider-row">
                <div class="setting-label-row">
                  <span>Inactivity timeout</span>
                  <span class="setting-value">{config.inactivityTimeoutSeconds} s</span>
                </div>
                <input type="range" min="30" max="300" step="15" aria-label="Inactivity timeout"
                       value={config.inactivityTimeoutSeconds}
                       oninput={(e) => handleSlider(e, 'inactivity_timeout_seconds')} />
              </div>
            {/if}
          </div>

          <div class="actions">
            <span class="shortcut-hint">Ctrl+1..3 switch tabs, Ctrl+, opens settings</span>
            <Button variant={confirmingReset ? 'danger' : 'danger-outline'} icon="fa-undo" onclick={handleReset}>
              {confirmingReset ? 'Confirm reset' : 'Reset all'}
            </Button>
          </div>
        </div>
      {:else}
        <div class="modal-body">
          <p class="loading">Loading configuration...</p>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background-color: var(--panel-bg);
    border-radius: 12px;
    box-shadow: 0 20px 60px rgba(0,0,0,0.3);
    width: 520px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border-color);
  }

  .modal-header h2 {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 1.1rem;
  }

  .modal-body {
    padding: 16px 20px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .settings-section {
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 14px;
  }

  .settings-section h3 {
    margin: 0 0 12px 0;
    font-size: 0.9rem;
    padding-bottom: 8px;
    border-bottom: 1px solid var(--border-color);
  }

  .section-toggle {
    width: 100%;
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: none;
    border: none;
    color: var(--text-primary);
    cursor: pointer;
    padding: 0;
  }
  .section-toggle h3 { flex: 1; text-align: left; }
  .section-toggle i { transition: transform 0.15s; color: var(--text-muted); margin-bottom: 12px; }
  .section-toggle i.open { transform: rotate(180deg); }

  .setting-item { padding: 8px 0; }

  .toggle-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }

  .slider-row { display: flex; flex-direction: column; gap: 6px; }

  .setting-label { font-size: 0.85rem; color: var(--text-primary); }

  .setting-hint {
    font-size: 0.75rem;
    color: var(--text-muted);
    margin: 2px 0 0 0;
    line-height: 1.3;
  }

  .setting-label-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    font-size: 0.85rem;
  }

  .setting-value {
    font-size: 0.75rem;
    font-weight: 600;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    color: var(--primary-color);
    background: rgba(52,152,219,0.1);
    padding: 1px 6px;
    border-radius: 4px;
  }

  input[type="range"] {
    -webkit-appearance: none;
    appearance: none;
    width: 100%;
    height: 6px;
    border-radius: 3px;
    background: var(--border-color);
    cursor: pointer;
  }

  input[type="range"]::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: white;
    border: 2px solid var(--primary-color);
    box-shadow: 0 1px 3px rgba(0,0,0,0.2);
    cursor: pointer;
  }

  :global([data-theme="dark"]) input[type="range"]::-webkit-slider-thumb { background: #2c2c2c; }

  .folder-control { display: flex; gap: 8px; margin-top: 6px; }

  .folder-input {
    flex: 1;
    padding: 6px 10px;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    background-color: var(--light-bg);
    color: var(--text-primary);
    font-size: 0.8rem;
  }

  .actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding-top: 8px;
  }

  .shortcut-hint { font-size: 0.72rem; color: var(--text-muted); }

  .loading { color: var(--text-muted); text-align: center; padding: 20px; }
</style>
