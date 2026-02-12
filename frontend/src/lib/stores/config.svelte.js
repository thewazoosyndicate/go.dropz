// App configuration store
let appConfig = $state(null);
let autoPair = $state(false);
let autoSync = $state(false);

export function getAppConfig() { return appConfig; }
export function getAutoPair() { return autoPair; }
export function getAutoSync() { return autoSync; }

export function setAppConfig(config) {
  appConfig = config;
  if (config) {
    autoPair = config.pairModeEnabled || false;
    autoSync = config.syncEnabled || false;
  }
}

export function setAutoPair(value) { autoPair = value; }
export function setAutoSync(value) { autoSync = value; }

export function updateConfigField(key, value) {
  if (!appConfig) return;
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
  if (configKey && configKey in appConfig) {
    let typedValue = value;
    if (typeof appConfig[configKey] === 'boolean') typedValue = value === 'true' || value === true;
    else if (typeof appConfig[configKey] === 'number') typedValue = Number(value);
    appConfig = { ...appConfig, [configKey]: typedValue };
  }
  if (key === 'pair_mode_enabled') autoPair = !!value;
  if (key === 'sync_enabled') autoSync = !!value;
}
