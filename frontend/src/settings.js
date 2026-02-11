// Settings panel — load/save/reset configuration
const { ipcRenderer } = require('electron');
const { LOG_LEVELS } = require('./fileLogger');
const state = require('./state');

let _debugLog = null;
let _addLogEntry = null;
let _showToast = null;
function debugLog(msg, level = LOG_LEVELS.INFO) { if (_debugLog) _debugLog(msg, level); }
function addLogEntry(msg, type) { if (_addLogEntry) _addLogEntry(msg, type); }
function showToast(msg, type) { if (_showToast) _showToast(msg, type); }

// Non-linear stops for media age: 1-10 by 1, 20-90 by 10, 120-360 by 30
const MEDIA_AGE_STOPS = [
  1,2,3,4,5,6,7,8,9,10,
  20,30,40,50,60,70,80,90,
  120,150,180,210,240,270,300,330,360
];

// Non-linear stops for auto-sync delay: 1-5min by 1, 10-30min by 5, 40-60min by 10
const SYNC_DELAY_STOPS = [
  60,120,180,240,300,
  600,900,1200,1500,1800,
  2400,3000,3600
];

function sliderValue(slider) {
  if (slider.dataset.setting === 'days_threshold')
    return MEDIA_AGE_STOPS[parseInt(slider.value)] || 7;
  if (slider.dataset.setting === 'inactivity_sync_interval_seconds')
    return SYNC_DELAY_STOPS[parseInt(slider.value)] || 600;
  return parseInt(slider.value);
}

function updateSliderFill(slider) {
  const pct = ((slider.value - slider.min) / (slider.max - slider.min)) * 100;
  // Compensate for 18px thumb: center travels from 9px to (width - 9px)
  const adj = `calc(${pct}% + ${9 - pct * 0.18}px)`;
  slider.style.background = `linear-gradient(to right, var(--warning-color), var(--secondary-color) ${adj}, var(--border-color) ${adj})`;
}

function formatValue(value, setting) {
  if (setting === 'days_threshold') return value + 'd';
  if (value >= 120) return Math.round(value / 60) + 'min';
  return value + 's';
}

function updateValueDisplay(slider) {
  const id = slider.id.replace('-input', '-value');
  const badge = document.getElementById(id);
  if (badge) badge.textContent = formatValue(sliderValue(slider), slider.dataset.setting);
}

// DOM references set from renderer
let pairAllToggle, syncQueueToggle, logLevelSelect;

function setDomRefs(refs) {
  pairAllToggle = refs.pairAllToggle;
  syncQueueToggle = refs.syncQueueToggle;
  logLevelSelect = refs.logLevelSelect;
}

function setupSettingsHandlers() {
  document.querySelectorAll('.setting-toggle').forEach(t => t.addEventListener('change', handleSettingChange));
  document.querySelectorAll('.setting-input[type="text"]').forEach(i => i.addEventListener('change', handleSettingChange));
  document.querySelectorAll('.setting-select').forEach(s => s.addEventListener('change', handleSettingChange));

  document.querySelectorAll('.setting-slider').forEach(slider => {
    updateSliderFill(slider);
    updateValueDisplay(slider);
    slider.addEventListener('input', (e) => {
      updateSliderFill(e.target);
      updateValueDisplay(e.target);
    });
    slider.addEventListener('change', handleSettingChange);
    slider.addEventListener('wheel', (e) => {
      e.preventDefault();
      const step = parseInt(slider.step) || 1;
      const delta = e.deltaY < 0 ? step : -step;
      const next = Math.max(+slider.min, Math.min(+slider.max, +slider.value + delta));
      if (next !== +slider.value) {
        slider.value = next;
        updateSliderFill(slider);
        updateValueDisplay(slider);
        slider.dispatchEvent(new Event('change'));
      }
    });
  });

  const browseButton = document.getElementById('browse-folder');
  if (browseButton) {
    browseButton.addEventListener('click', () => ipcRenderer.send('open-folder-dialog'));
    ipcRenderer.on('selected-folder', (event, path) => {
      const folderInput = document.getElementById('destination-folder-input');
      if (folderInput && path) {
        folderInput.value = path;
        folderInput.dispatchEvent(new Event('change'));
      }
    });
  }

  const resetButton = document.getElementById('reset-settings');
  if (resetButton) resetButton.addEventListener('click', resetAllSettings);

  debugLog('Settings handlers initialized', LOG_LEVELS.DEBUG);
}

function handleSettingChange(event) {
  const target = event.target;
  const settingName = target.dataset.setting;
  if (!settingName) return;

  let value;
  if (target.type === 'checkbox') value = target.checked;
  else if (target.type === 'range') value = sliderValue(target);
  else if (target.type === 'number') value = parseInt(target.value, 10);
  else value = target.value;

  debugLog(`Setting ${settingName} changed to ${value}`, LOG_LEVELS.INFO);

  const settingItem = target.closest('.setting-item');
  if (settingItem) {
    settingItem.classList.add('setting-changed');
    setTimeout(() => settingItem.classList.remove('setting-changed'), 2000);
  }

  updateSetting(settingName, value);
}

function updateSetting(key, value) {
  debugLog(`Updating setting: ${key} = ${value}`, LOG_LEVELS.INFO);

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    const request = new grpcUtils.UpdateSettingRequest();
    request.setSettingName(key);

    if (typeof value === 'boolean') request.setBoolValue(value);
    else if (typeof value === 'number') request.setIntValue(value);
    else request.setStringValue(String(value));

    client.updateSetting(request, (error) => {
      if (error) {
        debugLog(`Error updating setting ${key}: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to update setting ${key}: ${error.message}`, 'error');
        showToast(`Failed to update setting`, 'error');
        return;
      }

      debugLog(`Setting ${key} updated to ${value}`, LOG_LEVELS.INFO);

      // Update local config
      if (state.appConfig) {
        const mapping = {
          'pair_mode_enabled': 'pairModeEnabled',
          'sync_enabled': 'syncEnabled',
          'scan_interval_seconds': 'scanIntervalSeconds',
          'connect_timeout_seconds': 'connectTimeoutSeconds',
          'days_threshold': 'daysThreshold',
          'destination_folder': 'destinationFolder',
          'inactivity_timeout_seconds': 'inactivityTimeoutSeconds',
          'inactivity_sync_interval_seconds': 'inactivitySyncIntervalSeconds',
          'set_time_enabled': 'setTimeEnabled',
          'log_level': 'logLevel'
        };
        const configKey = mapping[key];
        if (configKey && configKey in state.appConfig) {
          let typedValue = value;
          if (typeof state.appConfig[configKey] === 'boolean') typedValue = value === 'true' || value === true;
          else if (typeof state.appConfig[configKey] === 'number') typedValue = Number(value);
          state.appConfig[configKey] = typedValue;
        }
      }

      // Sync UI for special settings
      if (key === 'pair_mode_enabled') {
        state.autoPair = value;
        if (pairAllToggle) pairAllToggle.checked = value;
      } else if (key === 'sync_enabled') {
        state.autoSync = value;
        if (syncQueueToggle) syncQueueToggle.checked = value;
      } else if (key === 'log_level') {
        if (logLevelSelect) logLevelSelect.value = value;
        const { setLogLevel } = require('./fileLogger');
        setLogLevel(String(value).toUpperCase());
      }
    });
  } catch (error) {
    debugLog(`Exception updating setting: ${error.message}`, LOG_LEVELS.ERROR);
  }
}

function loadConfig() {
  debugLog('Loading application configuration', LOG_LEVELS.INFO);

  return new Promise((resolve) => {
    try {
      const grpcUtils = require('./grpc-utils');
      const client = grpcUtils.getClient();
      const request = new grpcUtils.GetConfigRequest();

      client.getConfig(request, (error, response) => {
        if (error) {
          debugLog(`Error loading configuration: ${error.message}`, LOG_LEVELS.ERROR);
          resolve();
          return;
        }

        const config = response.getConfig();
        try {
          state.appConfig = {
            pairModeEnabled: config.getPairModeEnabled(),
            syncEnabled: config.getSyncEnabled(),
            scanIntervalSeconds: config.getScanIntervalSeconds(),
            connectTimeoutSeconds: config.getConnectTimeoutSeconds(),
            daysThreshold: config.getDaysThreshold(),
            destinationFolder: config.getDestinationFolder(),
            setTimeEnabled: config.getSetTimeEnabled(),
            inactivityTimeoutSeconds: config.getInactivityTimeoutSeconds(),
            inactivitySyncIntervalSeconds: config.getInactivitySyncIntervalSeconds(),
            logLevel: config.getLogLevel()
          };
        } catch (configError) {
          debugLog(`Error getting some config values: ${configError.message}`, LOG_LEVELS.WARN);
          state.appConfig = {
            pairModeEnabled: config.getPairModeEnabled ? config.getPairModeEnabled() : false,
            syncEnabled: config.getSyncEnabled ? config.getSyncEnabled() : false,
            scanIntervalSeconds: config.getScanIntervalSeconds ? config.getScanIntervalSeconds() : 30,
            connectTimeoutSeconds: config.getConnectTimeoutSeconds ? config.getConnectTimeoutSeconds() : 30,
            daysThreshold: config.getDaysThreshold ? config.getDaysThreshold() : 7,
            destinationFolder: config.getDestinationFolder ? config.getDestinationFolder() : '',
            setTimeEnabled: config.getSetTimeEnabled ? config.getSetTimeEnabled() : true,
            inactivityTimeoutSeconds: config.getInactivityTimeoutSeconds ? config.getInactivityTimeoutSeconds() : 60,
            inactivitySyncIntervalSeconds: config.getInactivitySyncIntervalSeconds ? config.getInactivitySyncIntervalSeconds() : 600,
            logLevel: config.getLogLevel ? config.getLogLevel() : 'info'
          };
        }

        // Sync toggle state
        state.autoSync = state.appConfig.syncEnabled;
        state.autoPair = state.appConfig.pairModeEnabled;
        if (syncQueueToggle) syncQueueToggle.checked = state.autoSync;
        if (pairAllToggle) pairAllToggle.checked = state.autoPair;

        debugLog(`Config loaded: autoSync=${state.autoSync}, autoPair=${state.autoPair}`, LOG_LEVELS.INFO);
        updateSettingsUI();
        resolve();
      });
    } catch (error) {
      debugLog(`Exception loading configuration: ${error.message}`, LOG_LEVELS.ERROR);
      resolve();
    }
  });
}

function resetAllSettings() {
  if (!confirm('Reset all settings to default values?')) return;

  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    const request = new grpcUtils.ResetSettingRequest();
    request.setSettingName('all');

    client.resetSetting(request, (error, config) => {
      if (error) {
        debugLog(`Error resetting settings: ${error.message}`, LOG_LEVELS.ERROR);
        showToast('Failed to reset settings', 'error');
        return;
      }

      state.appConfig = {
        pairModeEnabled: config.getPairModeEnabled(),
        syncEnabled: config.getSyncEnabled(),
        scanIntervalSeconds: config.getScanIntervalSeconds(),
        connectTimeoutSeconds: config.getConnectTimeoutSeconds(),
        daysThreshold: config.getDaysThreshold(),
        destinationFolder: config.getDestinationFolder(),
        setTimeEnabled: config.getSetTimeEnabled(),
        inactivityTimeoutSeconds: config.getInactivityTimeoutSeconds(),
        inactivitySyncIntervalSeconds: config.getInactivitySyncIntervalSeconds(),
        logLevel: config.getLogLevel()
      };

      state.autoSync = state.appConfig.syncEnabled;
      state.autoPair = state.appConfig.pairModeEnabled;
      if (syncQueueToggle) syncQueueToggle.checked = state.autoSync;
      if (pairAllToggle) pairAllToggle.checked = state.autoPair;

      updateSettingsUI();
      addLogEntry('All settings reset to defaults', 'success');
      showToast('Settings reset to defaults', 'success');
    });
  } catch (error) {
    debugLog(`Exception resetting settings: ${error.message}`, LOG_LEVELS.ERROR);
  }
}

function updateSettingsUI() {
  if (!state.appConfig) return;

  const mapping = {
    'pairModeEnabled': 'pair_mode_enabled',
    'syncEnabled': 'sync_enabled',
    'scanIntervalSeconds': 'scan_interval_seconds',
    'connectTimeoutSeconds': 'connect_timeout_seconds',
    'daysThreshold': 'days_threshold',
    'destinationFolder': 'destination_folder',
    'inactivityTimeoutSeconds': 'inactivity_timeout_seconds',
    'inactivitySyncIntervalSeconds': 'inactivity_sync_interval_seconds',
    'setTimeEnabled': 'set_time_enabled',
    'logLevel': 'log_level'
  };

  for (const [configKey, settingName] of Object.entries(mapping)) {
    const value = state.appConfig[configKey];
    if (value === undefined) continue;

    if (settingName === 'pair_mode_enabled' && pairAllToggle) pairAllToggle.checked = Boolean(value);
    else if (settingName === 'sync_enabled' && syncQueueToggle) syncQueueToggle.checked = Boolean(value);

    const el = document.querySelector(`[data-setting="${settingName}"]`);
    if (el) {
      if (el.type === 'checkbox') el.checked = Boolean(value);
      else if (settingName === 'days_threshold') {
        let idx = MEDIA_AGE_STOPS.indexOf(value);
        if (idx === -1) idx = MEDIA_AGE_STOPS.findIndex(s => s >= value) || 0;
        el.value = idx;
      } else if (settingName === 'inactivity_sync_interval_seconds') {
        let idx = SYNC_DELAY_STOPS.indexOf(value);
        if (idx === -1) idx = SYNC_DELAY_STOPS.findIndex(s => s >= value) || 0;
        el.value = idx;
      } else el.value = value;
      if (el.classList.contains('setting-slider')) {
        updateSliderFill(el);
        updateValueDisplay(el);
      }
    }
  }

  // Sync log level select
  if (state.appConfig.logLevel && logLevelSelect) {
    for (let i = 0; i < logLevelSelect.options.length; i++) {
      if (logLevelSelect.options[i].value.toLowerCase() === state.appConfig.logLevel.toLowerCase()) {
        logLevelSelect.selectedIndex = i; break;
      }
    }
  }
}

module.exports = {
  setDomRefs,
  setupSettingsHandlers,
  loadConfig,
  updateSetting,
  resetAllSettings,
  updateSettingsUI,
  setDebugLog: (fn) => { _debugLog = fn; },
  setAddLogEntry: (fn) => { _addLogEntry = fn; },
  setShowToast: (fn) => { _showToast = fn; }
};
