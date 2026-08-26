import { setLogLevel } from './ui.svelte.js';

// App configuration store
let appConfig = $state(null);
let autoSync = $state(false);

export function getAppConfig() { return appConfig; }
export function getAutoSync() { return autoSync; }

export function setAppConfig(config) {
  appConfig = config;
  if (config) {
    autoSync = config.syncEnabled || false;
    if (config.logLevel) setLogLevel(config.logLevel);
  }
}

export function setAutoSync(value) { autoSync = value; }

export function updateConfigField(key, value) {
  if (!appConfig) return;
  const mapping = {
    'sync_enabled': 'syncEnabled',
    'scan_interval_seconds': 'scanIntervalSeconds',
    'connect_timeout_seconds': 'connectTimeoutSeconds',
    'days_threshold': 'daysThreshold',
    'destination_folder': 'destinationFolder',
    'inactivity_timeout_seconds': 'inactivityTimeoutSeconds',
    'status_check_interval_seconds': 'statusCheckIntervalSeconds',
    'check_on_return': 'checkOnReturn',
    'turbo_enabled': 'turboEnabled',
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
  if (key === 'sync_enabled') autoSync = !!value;
  if (key === 'log_level') setLogLevel(value);
}
