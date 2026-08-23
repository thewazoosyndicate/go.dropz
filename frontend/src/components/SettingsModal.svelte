<script>
  import { getSettingsOpen, setSettingsOpen } from '../lib/stores/ui.svelte.js';
  import { getAppConfig } from '../lib/stores/config.svelte.js';
  import { updateSetting, resetAllSettings } from '../lib/grpc/actions.js';

  const { ipcRenderer } = window.require('electron');

  let isOpen = $derived(getSettingsOpen());
  let config = $derived(getAppConfig());

  // Non-linear slider stops
  const MEDIA_AGE_STOPS = [1,2,3,4,5,6,7,8,9,10,20,30,40,50,60,70,80,90,120,150,180,210,240,270,300,330,360];
  const STATUS_CHECK_STOPS = [0,60,120,300,600,900,1800,2700,3600];

  function formatValue(value, setting) {
    if (setting === 'days_threshold') return value + 'd';
    if (setting === 'status_check_interval_seconds' && value === 0) return 'Off';
    if (value >= 3600) return (value / 3600) + 'h';
    if (value >= 120) return Math.round(value / 60) + 'min';
    return value + 's';
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
    const value = sliderToValue(event.target.value, setting);
    updateSetting(setting, value);
  }

  function handleToggle(event, setting) {
    updateSetting(setting, event.target.checked);
  }

  function handleReset() {
    if (confirm('Reset all settings to default values?')) {
      resetAllSettings();
    }
  }

  function browseFolderClick() {
    ipcRenderer.send('open-folder-dialog');
  }

  // Listen for folder selection
  ipcRenderer.on('selected-folder', (event, path) => {
    if (path) updateSetting('destination_folder', path);
  });

  function handleOverlayClick(event) {
    if (event.target === event.currentTarget) setSettingsOpen(false);
  }
</script>

{#if isOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="modal-overlay" onclick={handleOverlayClick}>
    <div class="modal">
      <div class="modal-header">
        <h2><i class="fas fa-cog"></i> Settings</h2>
        <button class="close-btn" onclick={() => setSettingsOpen(false)} aria-label="Close settings">
          <i class="fas fa-times"></i>
        </button>
      </div>

      {#if config}
        <div class="modal-body">
          <!-- Scanning & Connectivity -->
          <div class="settings-section">
            <h3>Scanning & Connectivity</h3>

            <div class="setting-item toggle-row">
              <span class="setting-label">Set Camera Time</span>
              <label class="toggle">
                <input type="checkbox" checked={config.setTimeEnabled}
                       onchange={(e) => handleToggle(e, 'set_time_enabled')} />
                <span class="toggle-slider"></span>
              </label>
            </div>

            <div class="setting-item slider-row">
              <div class="setting-label-row">
                <span>Scan Interval</span>
                <span class="setting-value">{config.scanIntervalSeconds}s</span>
              </div>
              <input type="range" min="5" max="120" step="5"
                     value={config.scanIntervalSeconds}
                     oninput={(e) => handleSlider(e, 'scan_interval_seconds')} />
            </div>

            <div class="setting-item slider-row">
              <div class="setting-label-row">
                <span>Connect Timeout</span>
                <span class="setting-value">{config.connectTimeoutSeconds}s</span>
              </div>
              <input type="range" min="5" max="60" step="5"
                     value={config.connectTimeoutSeconds}
                     oninput={(e) => handleSlider(e, 'connect_timeout_seconds')} />
            </div>

            <div class="setting-item slider-row">
              <div class="setting-label-row">
                <span>Inactivity Timeout</span>
                <span class="setting-value">{config.inactivityTimeoutSeconds}s</span>
              </div>
              <input type="range" min="30" max="300" step="15"
                     value={config.inactivityTimeoutSeconds}
                     oninput={(e) => handleSlider(e, 'inactivity_timeout_seconds')} />
            </div>
          </div>

          <!-- Download -->
          <div class="settings-section">
            <h3>Download</h3>

            <div class="setting-item toggle-row">
              <div>
                <span class="setting-label">Check on Return</span>
                <p class="setting-hint">Checks for new media when a camera reappears</p>
              </div>
              <label class="toggle">
                <input type="checkbox" checked={config.checkOnReturn}
                       onchange={(e) => handleToggle(e, 'check_on_return')} />
                <span class="toggle-slider"></span>
              </label>
            </div>

            <div class="setting-item">
              <div class="setting-label">
                <span>Turbo Transfer</span>
                <p class="setting-hint">Faster on some setups, slower on others; measure with ble-probe -speed</p>
              </div>
              <label class="toggle">
                <input type="checkbox" checked={config.turboEnabled}
                       onchange={(e) => handleToggle(e, 'turbo_enabled')} />
                <span class="toggle-slider"></span>
              </label>
            </div>

            <div class="setting-item slider-row">
              <div class="setting-label-row">
                <span>Check Every</span>
                <span class="setting-value">{formatValue(config.statusCheckIntervalSeconds, 'status_check_interval_seconds')}</span>
              </div>
              <p class="setting-hint">Periodically pings cameras to spot new media</p>
              <input type="range" min="0" max="{STATUS_CHECK_STOPS.length - 1}" step="1"
                     value={valueToSlider(config.statusCheckIntervalSeconds, 'status_check_interval_seconds')}
                     oninput={(e) => handleSlider(e, 'status_check_interval_seconds')} />
            </div>

            <div class="setting-item slider-row">
              <div class="setting-label-row">
                <span>Media Age</span>
                <span class="setting-value">{formatValue(config.daysThreshold, 'days_threshold')}</span>
              </div>
              <input type="range" min="0" max="{MEDIA_AGE_STOPS.length - 1}" step="1"
                     value={valueToSlider(config.daysThreshold, 'days_threshold')}
                     oninput={(e) => handleSlider(e, 'days_threshold')} />
            </div>
          </div>

          <!-- Storage -->
          <div class="settings-section">
            <h3>Storage</h3>
            <div class="setting-item">
              <span class="setting-label">Destination Folder</span>
              <div class="folder-control">
                <input type="text" class="folder-input" value={config.destinationFolder || ''} readonly />
                <button class="browse-btn" onclick={browseFolderClick} aria-label="Browse folder">
                  <i class="fas fa-folder-open"></i>
                </button>
              </div>
            </div>
          </div>

          <div class="actions">
            <button class="reset-btn" onclick={handleReset}>
              <i class="fas fa-undo"></i> Reset All
            </button>
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

  .close-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 1.1rem;
    cursor: pointer;
    padding: 4px 8px;
  }

  .close-btn:hover {
    color: var(--text-primary);
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

  .setting-item {
    padding: 8px 0;
  }

  .toggle-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .slider-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .setting-label {
    font-size: 0.85rem;
    color: var(--text-primary);
  }

  .setting-hint {
    font-size: 0.72rem;
    color: var(--text-muted);
    margin: 2px 0 0 0;
    line-height: 1.3;
  }

  .setting-label-row {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }

  .setting-value {
    font-size: 0.75rem;
    font-weight: 600;
    font-family: monospace;
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
    outline: none;
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

  :global([data-theme="dark"]) input[type="range"]::-webkit-slider-thumb {
    background: #2c2c2c;
  }

  .toggle {
    position: relative;
    display: inline-flex;
    cursor: pointer;
  }

  .toggle input {
    opacity: 0;
    width: 0;
    height: 0;
    position: absolute;
  }

  .toggle-slider {
    width: 36px;
    height: 20px;
    background-color: #ccc;
    border-radius: 20px;
    position: relative;
    transition: 0.3s;
  }

  :global([data-theme="dark"]) .toggle-slider {
    background-color: #555;
  }

  .toggle-slider::before {
    content: '';
    position: absolute;
    width: 14px;
    height: 14px;
    left: 3px;
    bottom: 3px;
    background: white;
    border-radius: 50%;
    transition: 0.3s;
  }

  .toggle input:checked + .toggle-slider {
    background-color: var(--secondary-color);
  }

  .toggle input:checked + .toggle-slider::before {
    transform: translateX(16px);
  }

  .folder-control {
    display: flex;
    gap: 8px;
    margin-top: 6px;
  }

  .folder-input {
    flex: 1;
    padding: 6px 10px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    background-color: var(--light-bg);
    color: var(--text-primary);
    font-size: 0.8rem;
  }

  .browse-btn {
    padding: 6px 12px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    background: none;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .browse-btn:hover {
    border-color: var(--primary-color);
    color: var(--primary-color);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    padding-top: 8px;
  }

  .reset-btn {
    padding: 8px 16px;
    border: 1px solid var(--danger-color);
    border-radius: 6px;
    background: none;
    color: var(--danger-color);
    font-size: 0.8rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .reset-btn:hover {
    background-color: var(--danger-color);
    color: white;
  }

  .loading {
    color: var(--text-muted);
    text-align: center;
    padding: 20px;
  }
</style>
