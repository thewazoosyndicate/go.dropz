// Shared application state — single source of truth
const allDevices = {};
let syncQueue = [];
const pairingInProgress = {};
const pairingTimeouts = {};
let autoSync = false;
let autoPair = false;

let activeStream = null;

let serviceRunning = false;

let appConfig = null;

const streamStatus = {};

const connectionState = {
  isConnecting: false,
  retryCount: 0,
  maxRetries: 10,
  baseDelay: 1000
};

const uiState = {
  currentDeviceCounts: { discovered: 0, managed: 0, sync: 0 },
  deviceElements: {},
  poolMembership: {}
};

module.exports = {
  allDevices,
  get syncQueue() { return syncQueue; },
  set syncQueue(val) { syncQueue = val; },
  pairingInProgress,
  pairingTimeouts,
  get autoSync() { return autoSync; },
  set autoSync(val) { autoSync = val; },
  get autoPair() { return autoPair; },
  set autoPair(val) { autoPair = val; },
  get activeStream() { return activeStream; },
  set activeStream(val) { activeStream = val; },
  get serviceRunning() { return serviceRunning; },
  set serviceRunning(val) { serviceRunning = val; },
  get appConfig() { return appConfig; },
  set appConfig(val) { appConfig = val; },
  connectionState,
  uiState,
  streamStatus
};
