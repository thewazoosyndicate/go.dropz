// Renderer process script
const { ipcRenderer } = require('electron');
const { getClient } = require('./grpc-utils');
const { logToFile, LOG_LEVELS } = require('./fileLogger');

// DOM elements
const statusLight = document.getElementById('status-light');
const statusText = document.getElementById('status-text');
const restartButton = document.getElementById('restart-service');
const logContent = document.getElementById('log-content');
const clearLogsButton = document.getElementById('clear-logs');
const logLevelSelect = document.getElementById('log-level-select');
const showGoLogsToggle = document.getElementById('show-go-logs-toggle');
const themeToggle = document.getElementById('theme-toggle');

// GoPro sections DOM elements
const pairQueueContainer = document.getElementById('pair-queue-container');
const camerasPoolContainer = document.getElementById('cameras-pool-container');
const syncQueueContainer = document.getElementById('sync-queue-container');
const managedCountDisplay = document.getElementById('managed-count');
const unpairedCountDisplay = document.getElementById('unpaired-count');
const syncCountDisplay = document.getElementById('sync-count');
const pairAllToggle = document.getElementById('pair-all-toggle');
const syncQueueToggle = document.getElementById('sync-queue-toggle');

// View toggles
const viewToggleButtons = document.querySelectorAll('.view-button');

// Templates
const goProItemTemplate = document.getElementById('gopro-item-template');

// View state
let currentViewMode = 'grid'; // 'grid' or 'list'

// Theme state
let isDarkMode = false;

// Service state tracking
let serviceRunning = false;
let isRestarting = false;

// GoPro devices state
let allDevices = {}; // All detected devices by MAC address
let managedDevices = {}; // Devices that are managed (toggled on)
let syncQueue = []; // Devices waiting for sync
let pairingInProgress = {}; // Devices currently being paired
let syncingInProgress = {}; // Devices currently being synced

// Settings
let autoSync = false;
let autoPair = false;

// Streaming state
let activeStream = null;

// Variables to track connection state
let connectionState = {
  isConnecting: false,
  retryCount: 0,
  maxRetries: 10,
  baseDelay: 1000
};

// Group management state
let groups = [];

// Video management state
let videos = [];

// Configuration state
let appConfig = null;

// UI state tracking for efficient updates
let uiState = {
  currentDeviceCounts: {
    discovered: 0,
    managed: 0,
    sync: 0
  },
  deviceElements: {}, // Track DOM elements by MAC address for direct updates
  poolMembership: {} // Track which pools each device belongs to
};

// Log initial message
logToFile('Renderer process starting up');

// Initialize the UI
function init() {
  debugLog('Initializing frontend application', LOG_LEVELS.INFO);
  
  // Set initial status
  updateStatus(false);
  
  // Load theme preference from localStorage
  loadThemePreference();
  
  // Initialize toggle switches with default values
  // This ensures the UI is responsive immediately
  initializeToggles();
  
  // Set up event listeners
  restartButton.addEventListener('click', restartService);
  clearLogsButton.addEventListener('click', clearLogs);
  pairAllToggle.addEventListener('change', togglePairAll);
  logLevelSelect.addEventListener('change', changeLogLevel);
  themeToggle.addEventListener('click', toggleTheme);
  
  // Set up view toggle
  setupViewToggle();
  
  // Set up settings panel handlers
  setupSettingsHandlers();
  
  // Set up Go logs toggle if it exists
  if (showGoLogsToggle) {
    showGoLogsToggle.addEventListener('change', toggleGoLogs);
    // Initialize to checked by default
    showGoLogsToggle.checked = true;
  }
  
  // Load current log level from backend
  loadCurrentLogLevel();
  
  // Listen for IPC messages from the main process
  debugLog('Setting up IPC listeners', LOG_LEVELS.TRACE);
  setupIpcListeners();
  
  // Periodically update device lists (to handle devices that may have gone offline)
  setInterval(cleanupDisconnectedDevices, 30000); // Check every 30 seconds
  
  // Start streaming for gRPC-based device updates
  setTimeout(() => {
    debugLog('Starting device streaming', LOG_LEVELS.INFO);
    startDeviceStreaming();
    
    // Load additional data after connection is established
    setTimeout(() => {
      // Load app configuration
      loadConfig();
      
      // Load camera groups
      loadGroups();
      
      // Load videos
      loadVideos();
      
      debugLog('Initial data loading complete', LOG_LEVELS.INFO);
    }, 2000); // Give time for streams to stabilize
  }, 1000); // Delay initial connection slightly to allow backend to fully start
  
  // Handle app closing - cancel all streams
  window.addEventListener('beforeunload', () => {
    debugLog('App closing, cancelling all streams', LOG_LEVELS.INFO);
    cancelAllStreams();
  });
  
  debugLog('Frontend initialization complete', LOG_LEVELS.INFO);
}

// Initialize toggle switches with default values
function initializeToggles() {
  // Set default values for toggles
  autoPair = false;
  autoSync = false;
  
  // Update UI elements with defaults
  if (pairAllToggle) {
    pairAllToggle.checked = autoPair;
    pairAllToggle.addEventListener('change', togglePairAll);
  }
  
  if (syncQueueToggle) {
    syncQueueToggle.checked = autoSync;
    syncQueueToggle.addEventListener('change', toggleAutoSync);
  }
  
  debugLog('Toggle switches initialized with default values', LOG_LEVELS.TRACE);
}

// Setup view toggle buttons
function setupViewToggle() {
  if (viewToggleButtons) {
    viewToggleButtons.forEach(button => {
      button.addEventListener('click', () => {
        const viewMode = button.getAttribute('data-view');
        if (viewMode !== currentViewMode) {
          currentViewMode = viewMode;
          
          // Update active state on buttons
          viewToggleButtons.forEach(btn => {
            if (btn.getAttribute('data-view') === viewMode) {
              btn.classList.add('active');
            } else {
              btn.classList.remove('active');
            }
          });
          
          // Apply view mode to containers
          [camerasPoolContainer, pairQueueContainer, syncQueueContainer].forEach(container => {
            if (container) {
              container.classList.remove('devices-grid', 'devices-list');
              container.classList.add(`devices-${viewMode}`);
            }
          });
          
          debugLog(`View mode changed to: ${viewMode}`, LOG_LEVELS.INFO);
          
          // Refresh the UI to apply new view - use full rebuild for view changes
          updateDeviceLists();
        }
      });
    });
  }
}

// Start using gRPC streaming for device updates
function startDeviceStreaming() {
  // Prevent multiple simultaneous connection attempts
  if (connectionState.isConnecting) {
    debugLog('Connection attempt already in progress, skipping', LOG_LEVELS.INFO);
    return;
  }
  
  connectionState.isConnecting = true;
  debugLog('Starting device streaming with gRPC', LOG_LEVELS.INFO);
  
  try {
    // Get the client and proto definitions
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Start three separate streams for the three types of data
    startDiscoveredCamerasStream(client);
    startManagedCamerasStream(client);
    startSyncQueueStream(client);
    
    // Connection successful, reset retry counter
    connectionState.retryCount = 0;
    connectionState.isConnecting = false;
    
    // Update UI to show successful connection
    updateStatus(true);
    
  } catch (error) {
    debugLog(`Error starting device streams: ${error.message}`, LOG_LEVELS.ERROR);
    connectionState.isConnecting = false;
    
    // Handle connection failure with retry logic
    handleConnectionFailure(error);
  }
}

// Start streaming discovered cameras
function startDiscoveredCamerasStream(client) {
  try {
    const grpcUtils = require('./grpc-utils');
    const request = new grpcUtils.GetDiscoveredCamerasRequest();
    request.setChangeCounter(0);
    
    debugLog('Starting WatchDiscoveredCameras stream for real-time updates', LOG_LEVELS.TRACE);
    
    const call = client.watchDiscoveredCameras(request);
    
    // Store the stream so we can cancel it if needed
    if (!activeStream) activeStream = {};
    activeStream.discoveredCamerasStream = {
      call,
      cancel: () => {
        call.cancel();
        debugLog('DiscoveredCameras stream cancelled', LOG_LEVELS.INFO);
      },
      intentionalCancel: false
    };
    
    // Handle incoming data - this is now push-based (reactive)
    call.on('data', (response) => {
      // Skip heartbeats which are just keepalives
      if (response.getHeartbeat()) {
        debugLog('Received heartbeat from server', LOG_LEVELS.TRACE);
        return;
      }
      
      // If there are no changes, we can skip processing
      if (response.getNoChanges()) {
        debugLog('Received no-change notification from server', LOG_LEVELS.TRACE);
        return;
      }
      
      const cameras = response.getCamerasList();
      debugLog(`Received ${cameras.length} discovered cameras update via push notification`, LOG_LEVELS.TRACE);
      
      // Process each camera immediately
      cameras.forEach(camera => {
        const cameraObj = processDiscoveredCamera(camera);
        if (cameraObj) {
          addOrUpdateDevice(cameraObj);
          // addOrUpdateDevice now handles targeted updates automatically
        }
      });
    });
    
    call.on('error', (error) => {
      debugLog(`Error in discovered cameras stream: ${error}`, LOG_LEVELS.ERROR);
      addLogEntry(`Error receiving camera updates: ${error}`, 'error');
      
      // Attempt to restart the stream after a short delay
      setTimeout(() => {
        if (!activeStream.discoveredCamerasStream?.intentionalCancel) {
          debugLog('Attempting to restart discovered cameras stream', LOG_LEVELS.INFO);
          startDiscoveredCamerasStream(client);
        }
      }, 5000);
    });
    
    call.on('end', () => {
      debugLog('Discovered cameras stream ended', LOG_LEVELS.INFO);
      if (!activeStream.discoveredCamerasStream?.intentionalCancel) {
        // Stream ended unexpectedly, try to restart
        setTimeout(() => {
          debugLog('Attempting to restart discovered cameras stream after unexpected end', LOG_LEVELS.INFO);
          startDiscoveredCamerasStream(client);
        }, 5000);
      }
    });
  } catch (error) {
    debugLog(`Failed to start discovered cameras stream: ${error}`, LOG_LEVELS.ERROR);
    addLogEntry(`Failed to start camera updates: ${error}`, 'error');
  }
}

// Start streaming managed cameras
function startManagedCamerasStream(client) {
  try {
    const grpcUtils = require('./grpc-utils');
    const request = new grpcUtils.GetManagedCamerasRequest();
    // Set change counter to 0 for initial request to get all current managed cameras
    request.setChangeCounter(0);
    
    debugLog('Starting WatchManagedCameras stream for real-time updates', LOG_LEVELS.TRACE);
    
    const call = client.watchManagedCameras(request);
    
    // Store the stream so we can cancel it if needed
    if (!activeStream) activeStream = {};
    activeStream.managedCamerasStream = {
      call,
      cancel: () => {
        call.cancel();
        debugLog('ManagedCameras stream cancelled', LOG_LEVELS.INFO);
      },
      intentionalCancel: false
    };
    
    // Handle incoming data - now using real-time push
    call.on('data', (response) => {
      // Skip heartbeats which are just keepalives
      if (response.getHeartbeat()) {
        debugLog('Received heartbeat from managed cameras stream', LOG_LEVELS.TRACE);
        return;
      }
      
      // Skip if no changes
      if (response.getNoChanges()) {
        debugLog('Received no-change notification from managed cameras stream', LOG_LEVELS.TRACE);
        return;
      }
      
      const cameras = response.getCamerasList();
      debugLog(`Received ${cameras.length} managed cameras update via push notification`, LOG_LEVELS.TRACE);
      
      // Flag to track if we got any actual updates
      let hasUpdates = false;
      
      // Process each camera
      cameras.forEach(camera => {
        const cameraObj = processManagedCamera(camera);
        if (cameraObj) {
          addOrUpdateDevice(cameraObj);
          hasUpdates = true;
        }
      });
      
      // addOrUpdateDevice now handles targeted updates automatically
      if (hasUpdates) {
        debugLog('Updated managed camera data via targeted updates', LOG_LEVELS.TRACE);
      }
    });
    
    // Handle errors
    call.on('error', (error) => {
      debugLog(`ManagedCameras stream error: ${error.message}`, LOG_LEVELS.ERROR);
      
      // Check if this was an intentional cancellation
      if (activeStream.managedCamerasStream.intentionalCancel) {
        return;
      }
      
      // Attempt to reconnect
      setTimeout(() => {
        if (!isRestarting && serviceRunning) {
          debugLog('Reconnecting to ManagedCameras stream after error', LOG_LEVELS.INFO);
          startManagedCamerasStream(client);
        }
      }, calculateBackoffDelay());
    });
    
    // Handle end of stream
    call.on('end', () => {
      debugLog('ManagedCameras stream ended', LOG_LEVELS.INFO);
      
      // Attempt to reconnect if the stream ended unexpectedly
      if (!activeStream.managedCamerasStream.intentionalCancel && !isRestarting && serviceRunning) {
        debugLog('Reconnecting to ManagedCameras stream after end', LOG_LEVELS.INFO);
        setTimeout(() => startManagedCamerasStream(client), calculateBackoffDelay());
      }
    });
    
  } catch (error) {
    debugLog(`Error starting ManagedCameras stream: ${error.message}`, LOG_LEVELS.ERROR);
    // Attempt to reconnect after a delay
    setTimeout(() => {
      if (!isRestarting && serviceRunning) {
        debugLog('Retrying ManagedCameras stream after connection error', LOG_LEVELS.INFO);
        startManagedCamerasStream(getClient());
      }
    }, calculateBackoffDelay());
  }
}

// Start streaming sync queue
function startSyncQueueStream(client) {
  try {
    const grpcUtils = require('./grpc-utils');
    const request = new grpcUtils.GetSyncQueueRequest();
    // Set change counter to 0 for initial request to get all current queue entries
    request.setChangeCounter(0);
    
    debugLog('Starting WatchSyncQueue stream for real-time updates', LOG_LEVELS.TRACE);
    
    const call = client.watchSyncQueue(request);
    
    // Store the stream so we can cancel it if needed
    if (!activeStream) activeStream = {};
    activeStream.syncQueueStream = {
      call,
      cancel: () => {
        call.cancel();
        debugLog('SyncQueue stream cancelled', LOG_LEVELS.INFO);
      },
      intentionalCancel: false
    };
    
    // Handle incoming data - now using real-time push
    call.on('data', (response) => {
      // Skip heartbeats which are just keepalives
      if (response.getHeartbeat()) {
        debugLog('Received heartbeat from sync queue stream', LOG_LEVELS.TRACE);
        return;
      }
      
      // Skip if no changes
      if (response.getNoChanges()) {
        debugLog('Received no-change notification from sync queue stream', LOG_LEVELS.TRACE);
        return;
      }
      
      const queueEntries = response.getQueueList();
      debugLog(`Received sync queue update with ${queueEntries.length} entries via push notification`, LOG_LEVELS.TRACE);
      
      // Check if there are actual changes in the queue
      const oldQueueLength = syncQueue.length;
      
      // Update sync queue
      syncQueue = queueEntries.map(entry => processSyncQueueEntry(entry));
      
      // Update progress bars for all devices that are currently syncing
      updateAllSyncProgressBars();
      
      // Only update UI if the queue changed
      if (oldQueueLength !== syncQueue.length || queueEntries.length > 0) {
        debugLog('Updating UI with new sync queue data', LOG_LEVELS.TRACE);
        updateSyncQueueUI(); // Sync queue still needs full rebuild for now
        updateCounters();
      }
    });
    
    // Handle errors
    call.on('error', (error) => {
      debugLog(`SyncQueue stream error: ${error.message}`, LOG_LEVELS.ERROR);
      
      // Check if this was an intentional cancellation
      if (activeStream.syncQueueStream.intentionalCancel) {
        return;
      }
      
      // Attempt to reconnect
      setTimeout(() => {
        if (!isRestarting && serviceRunning) {
          debugLog('Reconnecting to SyncQueue stream after error', LOG_LEVELS.INFO);
          startSyncQueueStream(client);
        }
      }, calculateBackoffDelay());
    });
    
    // Handle end of stream
    call.on('end', () => {
      debugLog('SyncQueue stream ended', LOG_LEVELS.INFO);
      
      // Attempt to reconnect if the stream ended unexpectedly
      if (!activeStream.syncQueueStream.intentionalCancel && !isRestarting && serviceRunning) {
        setTimeout(() => startSyncQueueStream(client), calculateBackoffDelay());
      }
    });
    
  } catch (error) {
    debugLog(`Error starting SyncQueue stream: ${error.message}`, LOG_LEVELS.ERROR);
  }
}

// Calculate delay for exponential backoff
function calculateBackoffDelay() {
  const delay = connectionState.baseDelay * Math.pow(2, connectionState.retryCount);
  connectionState.retryCount = Math.min(connectionState.retryCount + 1, connectionState.maxRetries);
  return Math.min(delay, 30000); // Cap at 30 seconds
}

// Process a discovered camera from gRPC
function processDiscoveredCamera(camera) {
  try {
    const cameraState = camera.getCameraState();
    if (!cameraState) return null;
    
    const cameraData = cameraState.getCamera();
    if (!cameraData) return null;
    
    const macAddress = cameraData.getMacAddress();
    if (!macAddress) return null;
    
    const status = cameraState.getStatus();
    const metadata = cameraState.getMetadata();
    
    return {
      macAddress: macAddress,
      id: cameraData.getId(),
      name: cameraData.getName() || 'Unnamed Camera',
      alias: cameraData.getAlias(),
      wifiSsid: cameraData.getWifiSsid(),
      wifiPassword: cameraData.getWifiPassword(),
      rssi: cameraData.getRssi(),
      
      // Status properties - directly use boolean properties instead of a status code
      lastSeen: status ? status.getLastSeen()?.toDate() : null,
      lastSynced: status ? status.getLastSynced()?.toDate() : null,
      isPairing: status ? status.getIsPairing() : false,
      isPaired: status ? status.getIsPaired() : false,
      isManaged: status ? status.getIsManaged() : false,
      isReachable: status ? status.getIsReachable() : false,
      isSynced: status ? status.getIsSynced() : false,
      isSyncing: status ? status.getIsSyncing() : false,
      
      // Group ID
      groupId: cameraState.getGroupId(),
      
      // Metadata if available
      batteryLevel: metadata ? metadata.getBatteryLevel() : null,
      firmwareVersion: metadata ? metadata.getFirmwareVersion() : null,
      model: metadata ? metadata.getModel() : null,
      serialNumber: metadata ? metadata.getSerialNumber() : null,
      hardwareVersion: metadata ? metadata.getHardwareVersion() : null,
      
      // Visual status determined based on flags
      visualStatus: determineVisualStatus(status)
    };
  } catch (error) {
    debugLog(`Error processing discovered camera: ${error.message}`, LOG_LEVELS.ERROR);
    return null;
  }
}

// Process a managed camera from gRPC
function processManagedCamera(camera) {
  try {
    const cameraState = camera.getCameraState();
    if (!cameraState) return null;
    
    const cameraData = cameraState.getCamera();
    if (!cameraData) return null;
    
    const macAddress = cameraData.getMacAddress();
    if (!macAddress) return null;
    
    const status = cameraState.getStatus();
    const metadata = cameraState.getMetadata();
    
    return {
      macAddress: macAddress,
      id: cameraData.getId(),
      name: cameraData.getName() || 'Unnamed Camera',
      alias: cameraData.getAlias(),
      wifiSsid: cameraData.getWifiSsid(),
      wifiPassword: cameraData.getWifiPassword(),
      rssi: cameraData.getRssi(),
      
      // Status properties - directly use boolean properties instead of a status code
      lastSeen: status ? status.getLastSeen()?.toDate() : null,
      lastSynced: status ? status.getLastSynced()?.toDate() : null,
      isPairing: status ? status.getIsPairing() : false,
      isPaired: status ? status.getIsPaired() : false,
      isManaged: status ? status.getIsManaged() : false,
      isReachable: status ? status.getIsReachable() : false,
      isSynced: status ? status.getIsSynced() : false,
      isSyncing: status ? status.getIsSyncing() : false,
      
      // Group ID
      groupId: cameraState.getGroupId(),
      
      // Metadata if available
      batteryLevel: metadata ? metadata.getBatteryLevel() : null,
      firmwareVersion: metadata ? metadata.getFirmwareVersion() : null,
      model: metadata ? metadata.getModel() : null,
      serialNumber: metadata ? metadata.getSerialNumber() : null,
      hardwareVersion: metadata ? metadata.getHardwareVersion() : null,
      
      // Visual status determined based on flags
      visualStatus: determineVisualStatus(status)
    };
  } catch (error) {
    debugLog(`Error processing managed camera: ${error.message}`, LOG_LEVELS.ERROR);
    return null;
  }
}

// Determine visual status for UI display based on camera status flags
function determineVisualStatus(status) {
  if (!status) return 'Unknown';
  
  // GRPC response object status uses getIsReachable(), etc.
  const isReachable = typeof status.getIsReachable === 'function' ? status.getIsReachable() : status.isReachable;
  const isPairing = typeof status.getIsPairing === 'function' ? status.getIsPairing() : status.isPairing;
  const isSyncing = typeof status.getIsSyncing === 'function' ? status.getIsSyncing() : status.isSyncing;
  const isPaired = typeof status.getIsPaired === 'function' ? status.getIsPaired() : status.isPaired;
  const isManaged = typeof status.getIsManaged === 'function' ? status.getIsManaged() : status.isManaged;
  
  // Follow the exact order of priority as defined in requirements
  if (!isReachable) {
    return 'Unreachable';
  }
  
  if (isPairing) {
    return 'Pairing';
  }
  
  if (isSyncing) {
    return 'Syncing';
  }
  
  // Visual-only statuses deduced from the combination of flags
  if (!isPaired) {
    return 'Discovered';
  }
  
  if (isPaired && !isManaged) {
    return 'Available';
  }
  
  if (!isPaired && isManaged) {
    return 'Unavailable';
  }
  
  if (isPaired && isManaged) {
    return 'Managed';
  }
  
  return 'Unknown';
}

// Check if a camera should be in the discovered pool
// For a camera to appear in "Discovered" it must either not be paired or not be managed (reachability not required)
function shouldShowInDiscoveredPool(device) {
  return !device.isPaired || !device.isManaged;
}

// Check if a camera should be in the managed pool
// For a camera to appear in "Managed pool" it must be reachable, paired, and managed
function shouldShowInManagedPool(device) {
  return device.isReachable && device.isPaired && device.isManaged;
}

// Check if a camera should be in the sync queue
// For a camera to appear in "Sync Queue" she will have to have is_synced false, is_paired true, is_managed true and is_reachable true
function shouldShowInSyncQueue(device) {
  return device.isReachable && device.isPaired && device.isManaged && !device.isSynced;
}

// Process a sync queue entry from gRPC
function processSyncQueueEntry(entry) {
  try {
    return {
      cameraId: entry.getCameraId(),
      queuedAt: entry.getQueuedAt()?.toDate(),
      priority: entry.getPriority(),
      progressPercent: entry.getProgressPercent(),
      currentOperation: entry.getCurrentOperation()
    };
  } catch (error) {
    debugLog(`Error processing sync queue entry: ${error.message}`, LOG_LEVELS.ERROR);
    return null;
  }
}

// Handle connection failure with retry
function handleConnectionFailure(error) {
  debugLog(`Connection failure: ${error.message}`, LOG_LEVELS.ERROR);
  addLogEntry(`Connection failure: ${error.message}`, 'error');
  
  // Update UI to show connection error
  if (statusText.textContent === 'Running') {
    statusText.textContent = 'Running (gRPC connection error)';
  }
  
  // Implement retry with exponential backoff
  if (connectionState.retryCount < connectionState.maxRetries && serviceRunning) {
    const delay = calculateBackoffDelay();
    debugLog(`Reconnecting in ${delay}ms (attempt ${connectionState.retryCount})`, LOG_LEVELS.INFO);
    addLogEntry(`Connection lost. Retry in ${Math.round(delay/1000)} seconds`, 'warn');
    
    setTimeout(() => {
      if (!isRestarting && serviceRunning) {
        startDeviceStreaming();
      }
    }, delay);
  }
}

// Cancel all active streams
function cancelAllStreams() {
  debugLog('Cancelling all active streams', LOG_LEVELS.INFO);
  
  if (activeStream) {
    if (activeStream.discoveredCamerasStream) {
      activeStream.discoveredCamerasStream.intentionalCancel = true;
      activeStream.discoveredCamerasStream.cancel();
    }
    
    if (activeStream.managedCamerasStream) {
      activeStream.managedCamerasStream.intentionalCancel = true;
      activeStream.managedCamerasStream.cancel();
    }
    
    if (activeStream.syncQueueStream) {
      activeStream.syncQueueStream.intentionalCancel = true;
      activeStream.syncQueueStream.cancel();
    }
    
    activeStream = null;
  }
}

// Add a debug function that will output to the terminal with different log levels
function debugLog(message, level = LOG_LEVELS.INFO, context = {}) {
  // Import current log level from fileLogger
  const { currentLogLevel } = require('./fileLogger');
  
  // Only proceed if the message's level is lower than or equal to the current log level
  if (level > currentLogLevel()) {
    // Message filtered by log level
    return;
  }
  
  // Get the log level name
  const levelName = getLogLevelName(level);
  
  // Format the message with context if provided
  let formattedMessage = message;
  if (Object.keys(context).length > 0) {
    const contextStr = Object.entries(context)
      .map(([key, value]) => `${key}=${JSON.stringify(value)}`)
      .join(' ');
    formattedMessage = `${message} ${contextStr}`;
  }
  
  // Format the message with the level name
  const finalMessage = `[FRONTEND-${levelName}] ${formattedMessage}`;
  
  // Choose appropriate console method based on level
  const consoleMethod = 
    level === LOG_LEVELS.ERROR ? console.error :
    level === LOG_LEVELS.WARN ? console.warn :
    console.log;
  consoleMethod(finalMessage);
  
  // Log to file with appropriate level
  logToFile(formattedMessage, level);
  
  // Send to main process
  try {
    ipcRenderer.send('debug-log', finalMessage);
  } catch (error) {
    logToFile(`Error sending debug message to main: ${error.message}`, LOG_LEVELS.ERROR);
  }
  
  // Add to the UI logs with appropriate styling
  const logStyle = 
    level === LOG_LEVELS.ERROR ? 'error' : 
    level === LOG_LEVELS.WARN ? 'warn' : 'info';
  addLogEntry(`${levelName}: ${formattedMessage}`, logStyle);
}

// Helper to get log level name
function getLogLevelName(level) {
  switch (level) {
    case LOG_LEVELS.ERROR: return 'Error';
    case LOG_LEVELS.WARN: return 'Warning';
    case LOG_LEVELS.INFO: return 'Info';
    case LOG_LEVELS.DEBUG: return 'Debug';
    case LOG_LEVELS.TRACE: return 'Trace';
    default: return 'Unknown';
  }
}

// Clean up devices that haven't been seen in a while
function cleanupDisconnectedDevices() {
  const now = new Date();
  const timeout = 30000; // 30 seconds - increased timeout for better stability
  
  for (const mac in allDevices) {
    const device = allDevices[mac];
    const lastSeen = new Date(device.lastSeen);
    
    if (now - lastSeen > timeout) {
      const membership = uiState.poolMembership[mac] || {};
      
      // Check if this is a managed camera
      if (device.isManaged && device.isPaired) {
        // For managed cameras, just mark as offline but keep them visible
        debugLog(`Managed camera ${device.name} (${mac}) is offline but keeping visible`, LOG_LEVELS.DEBUG);
        
        // Update device state to show as unreachable
        device.isReachable = false;
        device.rssi = -100; // Set to minimum RSSI to show red signal
        device.visualStatus = 'Unreachable';
        
        // Update the device element to show offline status
        if (membership.managed && uiState.deviceElements[mac] && uiState.deviceElements[mac].managed) {
          updateDeviceElement(uiState.deviceElements[mac].managed, device, false);
        }
        
        // Remove from discovery and sync pools only
        if (membership.discovered) {
          removeDeviceFromPool(mac, 'discovered');
        }
        if (membership.sync) {
          removeDeviceFromPool(mac, 'sync');
        }
        
        // Update pool membership to reflect changes
        uiState.poolMembership[mac] = {
          discovered: false,
          managed: true, // Keep managed status
          sync: false
        };
      } else {
        // For non-managed cameras, remove completely
        addLogEntry(`Device ${device.name} (${mac}) has not been seen for a while and was removed.`, 'info');
        
        // Remove from all UI pools
        if (membership.discovered) {
          removeDeviceFromPool(mac, 'discovered');
        }
        if (membership.managed) {
          removeDeviceFromPool(mac, 'managed');
        }
        if (membership.sync) {
          removeDeviceFromPool(mac, 'sync');
        }
        
        // Clean up tracking completely
        delete allDevices[mac];
        delete uiState.poolMembership[mac];
        delete uiState.deviceElements[mac];
      }
      
      // Remove from other collections as needed
      if (syncQueue.some(entry => entry.cameraId === device.id)) {
        syncQueue = syncQueue.filter(entry => entry.cameraId !== device.id);
      }
      
      if (pairingInProgress[mac]) {
        delete pairingInProgress[mac];
      }
      
      if (syncingInProgress[mac]) {
        delete syncingInProgress[mac];
      }
    }
  }
  
  // Update counters after cleanup
  updateCounters();
}

// Set up IPC listeners for communication with the main process
function setupIpcListeners() {
  // Listen for Go binary status updates
  ipcRenderer.on('go-binary-status', (event, data) => {
    debugLog(`Received go-binary-status: ${JSON.stringify(data)}`);
    
    // If we're in a restart process, we expect to receive a 'stopped' status
    // followed by a 'running' status, so handle it appropriately
    if (isRestarting && !data.running) {
      debugLog('Service stopped as part of restart process', LOG_LEVELS.INFO);
      updateStatus(false, data.exitCode);
      return;
    }
    
    updateStatus(data.running, data.exitCode);
    
    if (data.running) {
      addLogEntry('Service started', 'info');
      
      // If this is part of a restart, don't automatically start streaming
      // as the restart function will handle that after its timeout
      if (!isRestarting) {
        // Start streaming if service is running
        setTimeout(() => {
          // Give the gRPC server a moment to start up
          debugLog('Starting device streaming after service start', LOG_LEVELS.INFO);
          startDeviceStreaming();
        }, 2000);
      } else {
        debugLog('Service restarted, streaming will be initiated by restart handler', LOG_LEVELS.INFO);
      }
    } else {
      // Cancel any active streams when the service stops to prevent reconnection attempts
      if (activeStream) {
        debugLog('Service stopped, cancelling active stream', LOG_LEVELS.INFO);
        activeStream.intentionalCancel = true;
        activeStream.cancel();
        activeStream = null;
      }
      
      if (data.exitCode === 0) {
        addLogEntry('Service stopped gracefully', 'info');
      } else {
        addLogEntry(`Service stopped with exit code ${data.exitCode}`, 'error');
      }
    }
  });
  
  // Listen for go binary errors
  ipcRenderer.on('go-binary-error', (event, errorMessage) => {
    debugLog(`Go binary error: ${errorMessage}`, LOG_LEVELS.ERROR);
    addLogEntry(`Service error: ${errorMessage}`, 'error');
  });
  
  // Listen for Go binary logs
  ipcRenderer.on('go-binary-log', (event, log) => {
    // Parse log level from the log message
    const { level, style, prefix } = parseGoLogLevel(log);
    
    // Log to file only (avoid duplication with addLogEntry)
    logToFile(`${prefix} ${log}`, level);
    
    // Add to UI logs with 'go' source only (debugLog would duplicate this)
    addLogEntry(`${prefix} ${log}`, style, 'go');
  });
  
  // Listen for pair device responses
  ipcRenderer.on('pair-device-response', (event, result) => {
    debugLog(`Received pair-device-response: ${JSON.stringify(result)}`, LOG_LEVELS.INFO);
    
    // Process the response
    if (result.success) {
      // Find the device that was being paired
      const deviceMac = Object.keys(pairingInProgress).find(mac => pairingInProgress[mac]);
      
      if (deviceMac && allDevices[deviceMac]) {
        // Update the device status to paired
        updateDeviceStatus(deviceMac, 'paired');
        addLogEntry(`Device ${allDevices[deviceMac].name} paired successfully`, 'info');
      }
    } else {
      addLogEntry(`Pairing failed: ${result.message}`, 'error');
    }
    
    // Clear pairing in progress flags
    Object.keys(pairingInProgress).forEach(mac => {
      delete pairingInProgress[mac];
    });
    
    // updateDeviceStatus already handles UI updates via targeted updates
  });
}

// Update the service status UI
function updateStatus(running, exitCode) {
  // Only log status changes, not periodic status checks
  const statusChanged = serviceRunning !== running;
  if (statusChanged) {
    debugLog(`Service status changed to: ${running ? 'running' : 'stopped'}${exitCode ? ` (exit code: ${exitCode})` : ''}`, LOG_LEVELS.INFO);
  }
  
  serviceRunning = running;
  
  if (isRestarting) {
    statusLight.className = 'status-light restarting';
    statusText.textContent = 'Restarting...';
    restartButton.disabled = true;
    
    // Don't attempt connection checks or streaming while restarting
    return;
  } else if (running) {
    statusLight.className = 'status-light running';
    
    // Keep any existing error state in the status text if we're not explicitly checking connection
    if (statusText.textContent !== 'Running (Reconnecting...)' && 
        statusText.textContent !== 'Running (gRPC connection error)') {
      statusText.textContent = 'Running';
    }
    
    restartButton.disabled = false;
    
    // If the backend process is running but we can't connect to gRPC, show a warning
    // Only make this gRPC check after a status change to prevent excessive calls
    if (statusChanged) {
      checkGrpcConnection()
        .then(() => {
          // If connection succeeded, update any error status to Running
          if (statusText.textContent === 'Running (gRPC connection error)' || 
              statusText.textContent === 'Running (Reconnecting...)') {
            statusText.textContent = 'Running';
          }
        })
        .catch(error => {
          debugLog('gRPC connection error even though process is running: ' + error.message, LOG_LEVELS.ERROR);
          
          // Don't overwrite a more specific reconnecting status
          if (statusText.textContent !== 'Running (Reconnecting...)') {
            statusText.textContent = 'Running (gRPC connection error)';
          }
        });
    }
  } else {
    // Service not running
    statusLight.className = 'status-light';
    
    let statusMessage = 'Stopped';
    if (exitCode !== undefined && exitCode !== 0) {
      statusMessage = `Stopped (Exit code: ${exitCode})`;
      
      // If the service crashed, log it clearly
      debugLog(`Service crashed with exit code ${exitCode}`, LOG_LEVELS.ERROR);
    }
    
    statusText.textContent = statusMessage;
    restartButton.disabled = false;
  }
}

// Restart the Go binary service
function restartService() {
  // Set restart flag to prevent reconnection attempts during restart
  isRestarting = true;
  
  debugLog('Full application refresh requested', LOG_LEVELS.INFO);
  addLogEntry('Refreshing application...', 'info');
  
  // Reset connection state
  connectionState.retryCount = 0;
  connectionState.isConnecting = false;
  
  // Small delay to allow log messages to be displayed
  setTimeout(() => {
    debugLog('Performing full page refresh', LOG_LEVELS.INFO);
    // Force a full browser refresh (equivalent to Ctrl+R)
    window.location.reload(true); // true forces a reload from server, not from cache
  }, 500);
}

// Add a log entry to the log display
function addLogEntry(message, type, source = 'app') {
  // Filter out MapToStruct warnings related to BLE
  if (shouldFilterLogMessage(message, type, source)) {
    return; // Skip this log entry
  }
  
  const entry = document.createElement('div');
  entry.className = `log-entry ${type}`;
  
  // Format timestamp
  const timestamp = new Date().toLocaleTimeString();
  const formattedMessage = `[${timestamp}] ${message}`;
  
  // Set the text content
  entry.textContent = formattedMessage;
  
  // Add data attributes for filtering
  entry.dataset.logType = type;
  entry.dataset.source = source;
  
  // Check if Go logs are hidden
  if (source === 'go' && showGoLogsToggle && !showGoLogsToggle.checked) {
    entry.style.display = 'none';
  }
  
  // Add to log container at the top instead of the bottom
  logContent.insertBefore(entry, logContent.firstChild);
  // No need to scroll to bottom anymore as the new entries are at the top
  
  // Limit the number of entries to prevent performance issues
  const maxEntries = 500;
  while (logContent.children.length > maxEntries) {
    logContent.removeChild(logContent.lastChild);
  }
}

// Filter out specific BLE-related log messages
function shouldFilterLogMessage(message, type, source) {
  // Only filter warning messages from the Go backend
  if (type !== 'warn' || source !== 'go') {
    return false;
  }
  
  // Filter patterns for MapToStruct warnings
  const filterPatterns = [
    "MapToStruct: invalid field detected *adapter.Adapter1Properties.Connectable",
    "MapToStruct: invalid field detected *adapter.Adapter1Properties.PowerState",
    "MapToStruct: invalid field detected *adapter.Adapter1Properties.Version",
    "MapToStruct: invalid field detected *adapter.Adapter1Properties.Manufacturer",
    "MapToStruct: invalid field detected *device.Device1Properties.Bonded"
  ];
  
  // Check if message contains any of the filter patterns
  return filterPatterns.some(pattern => message.includes(pattern));
}

// Clear all log entries
function clearLogs() {
  logContent.innerHTML = '';
  addLogEntry('Logs cleared', 'info');
}

// Change the log level based on the selected value
function changeLogLevel() {
  const selectedLevel = logLevelSelect.value;
  
  try {
    // First, update the frontend log level immediately for responsive UI
    const { setLogLevel } = require('./fileLogger');
    const frontendLevelUpdated = setLogLevel(selectedLevel);
    
    // Show the change in the UI
    const logLevelName = getLogLevelName(frontendLevelUpdated);
    
    debugLog(`Frontend log level changing to: ${logLevelName}`, LOG_LEVELS.INFO);
    addLogEntry(`Setting log level to: ${logLevelName.toUpperCase()}`, 'info');
    
    // Test log messages at different levels to demonstrate the change
    debugLog('This is a DEBUG message - should show if level is DEBUG or TRACE', LOG_LEVELS.DEBUG);
    debugLog('This is a TRACE message - should only show if level is TRACE', LOG_LEVELS.TRACE);
    
    // Then update backend log level through gRPC
    const client = getClient();
    const { UpdateConfigRequest, Config } = require('./proto/config_pb');
    
    // First, get the current config
    const getConfigRequest = new (require('./proto/config_pb').GetConfigRequest)();
    
    client.getConfig(getConfigRequest, (getError, getResponse) => {
      if (getError) {
        debugLog(`Failed to get current config: ${getError.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to get current config: ${getError.message}`, 'error');
        return;
      }
      
      // Create a new config with the current settings
      const currentConfig = getResponse.getConfig();
      const newConfig = new Config();
      
      // Copy all existing values from the current config
      if (currentConfig.getScanIntervalSeconds()) {
        newConfig.setScanIntervalSeconds(currentConfig.getScanIntervalSeconds());
      }
      if (currentConfig.getConnectTimeoutSeconds()) {
        newConfig.setConnectTimeoutSeconds(currentConfig.getConnectTimeoutSeconds());
      }
      if (currentConfig.getVideoAgeDays()) {
        newConfig.setVideoAgeDays(currentConfig.getVideoAgeDays());
      }
      if (currentConfig.getSetTimeEnabled() !== undefined) {
        newConfig.setSetTimeEnabled(currentConfig.getSetTimeEnabled());
      }
      if (currentConfig.getInactivityTimeoutSeconds()) {
        newConfig.setInactivityTimeoutSeconds(currentConfig.getInactivityTimeoutSeconds());
      }
      
      // Set the new log level - ensure proper string format (e.g., "debug", "info")
      // Go's logrus uses lowercase log levels
      newConfig.setLogLevel(selectedLevel.toLowerCase());
      
      // Create and send the update request
      const request = new UpdateConfigRequest();
      request.setConfig(newConfig);
      
      client.updateConfig(request, (error, response) => {
        if (error) {
          debugLog(`Failed to change backend log level: ${error.message}`, LOG_LEVELS.ERROR);
          addLogEntry(`Failed to change backend log level: ${error.message}`, 'error');
        } else {
          debugLog(`Backend log level changed to: ${selectedLevel}`, LOG_LEVELS.INFO);
          addLogEntry(`Backend (Go) log level changed to: ${selectedLevel.toUpperCase()}`, 'info');
          
          // Generate some test logs to verify both frontend and backend are working
          debugLog(`Log synchronization complete - both frontend and backend now using ${logLevelName}`, LOG_LEVELS.INFO);
        }
      });
    });
  } catch (error) {
    debugLog(`Exception changing log level: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Exception changing log level: ${error.message}`, 'error');
  }
}

// Load current log level from backend
function loadCurrentLogLevel() {
  const select = document.getElementById('backend-log-level-select');
  if (!select) return;

  // Set default to info if not set
  const currentLevel = localStorage.getItem('logLevel') || 'info';
  select.value = currentLevel;

  // Update the backend log level
  updateSetting('log_level', currentLevel);
}

// Add or update a device in the global tracking
function addOrUpdateDevice(device) {
  if (!device || !device.macAddress) {
    debugLog('Attempted to add device with no MAC address', LOG_LEVELS.ERROR);
    return null;
  }
  
  // Validate required data
  if (!device.name) {
    device.name = device.model || 'Unnamed Camera';
  }
  
  // First call or moving from one list to another
  const existingDevice = allDevices[device.macAddress];
  const isNewDevice = !existingDevice;
  
  // Track RSSI changes for real-time signal strength updates
  let rssiChanged = false;
  let lastSeenChanged = false;
  let significantChange = false;
  
  if (!isNewDevice && existingDevice) {
    rssiChanged = existingDevice.rssi !== device.rssi;
    lastSeenChanged = existingDevice.lastSeen !== device.lastSeen;
    
    // Check for significant status changes
    significantChange = 
      existingDevice.isReachable !== device.isReachable ||
      existingDevice.isPaired !== device.isPaired ||
      existingDevice.isManaged !== device.isManaged ||
      existingDevice.isSyncing !== device.isSyncing ||
      existingDevice.name !== device.name ||
      existingDevice.visualStatus !== device.visualStatus;
    
    // Log RSSI changes for debugging real-time updates
    if (rssiChanged) {
      debugLog(`RSSI updated for ${device.name} (${device.macAddress}): ${existingDevice.rssi} -> ${device.rssi}`, LOG_LEVELS.TRACE);
    }
  }
  
  if (isNewDevice) {
    // Log new device
    debugLog(`New device discovered: ${device.name} (${device.macAddress}) RSSI: ${device.rssi}`, LOG_LEVELS.INFO);
    addLogEntry(`New device discovered: ${getDeviceDisplayName(device)}`, 'info');
    significantChange = true;
  } else if (significantChange) {
    // Log significant changes
    if (existingDevice.isReachable !== device.isReachable) {
      debugLog(`Device reachability changed: ${device.name} (${device.macAddress}) - now ${device.isReachable ? 'reachable' : 'unreachable'}`, LOG_LEVELS.INFO);
    }
    
    if (existingDevice.isPaired !== device.isPaired) {
      debugLog(`Device paired status changed: ${device.name} (${device.macAddress}) - now ${device.isPaired ? 'paired' : 'unpaired'}`, LOG_LEVELS.INFO);
      if (device.isPaired) {
        addLogEntry(`Device paired: ${getDeviceDisplayName(device)}`, 'success');
      }
    }
    
    if (existingDevice.isManaged !== device.isManaged) {
      debugLog(`Device management changed: ${device.name} (${device.macAddress}) - now ${device.isManaged ? 'managed' : 'unmanaged'}`, LOG_LEVELS.INFO);
    }
    
    if (existingDevice.isSyncing !== device.isSyncing) {
      debugLog(`Device syncing status changed: ${device.name} (${device.macAddress}) - now ${device.isSyncing ? 'syncing' : 'not syncing'}`, LOG_LEVELS.INFO);
    }
  }
  
  // Preserve timestamps if they're newer in the existing device
  if (existingDevice && existingDevice.lastSeen && device.lastSeen) {
    if (existingDevice.lastSeen > device.lastSeen) {
      device.lastSeen = existingDevice.lastSeen;
    }
  }
  
  // Calculate which pools this device should be in
  const poolMembership = {
    discovered: shouldShowInDiscoveredPool(device),
    managed: shouldShowInManagedPool(device),
    sync: shouldShowInSyncQueue(device)
  };
  
  // CRITICAL: Preserve the old membership BEFORE updating uiState.poolMembership
  const previousMembership = uiState.poolMembership[device.macAddress] || {};
  const membershipChanged = isNewDevice || 
    previousMembership.discovered !== poolMembership.discovered ||
    previousMembership.managed !== poolMembership.managed ||
    previousMembership.sync !== poolMembership.sync;
  
  // Debug membership changes
  if (membershipChanged && !isNewDevice) {
    debugLog(`Pool membership changed for ${device.name}: Discovered=${previousMembership.discovered}=>${poolMembership.discovered}, Managed=${previousMembership.managed}=>${poolMembership.managed}, Sync=${previousMembership.sync}=>${poolMembership.sync}`, LOG_LEVELS.DEBUG);
  }
  
  // Update or add the device
  allDevices[device.macAddress] = device;
  // NOTE: We update poolMembership AFTER calling updateDeviceInPools so it can access the old values
  // uiState.poolMembership[device.macAddress] = poolMembership;
  
  // Only log detailed pool membership info for new devices or when explicitly debugging
  if (isNewDevice) {
    debugLog(`Device ${device.name} pool memberships: Discovered=${poolMembership.discovered}, Managed=${poolMembership.managed}, SyncQueue=${poolMembership.sync}`, LOG_LEVELS.DEBUG);
    debugLog(`Device ${device.name} flags: isReachable=${device.isReachable}, isPaired=${device.isPaired}, isManaged=${device.isManaged}, isSynced=${device.isSynced}`, LOG_LEVELS.DEBUG);
  }
  
  // Debug device state for troubleshooting
  if (membershipChanged || isNewDevice) {
    debugDeviceState(device, isNewDevice ? 'NEW DEVICE' : 'MEMBERSHIP CHANGED');
  }
  
  // Determine what kind of update we need
  if (isNewDevice || membershipChanged || significantChange) {
    // Significant change - device needs to be added/removed/moved between pools
    updateDeviceInPools(device, poolMembership, isNewDevice, membershipChanged);
    // NOW update the UI state after the pools have been updated
    uiState.poolMembership[device.macAddress] = poolMembership;
    updateCounters();
  } else if (rssiChanged) {
    // Just RSSI update - only update signal strength indicators
    updateSignalStrengthForDevice(device.macAddress, device.rssi);
    debugLog(`RSSI-only update for ${device.name}: ${device.rssi} dBm`, LOG_LEVELS.TRACE);
  } else if (lastSeenChanged) {
    // Just timestamp update - only update the timestamp display
    updateDeviceTimestamp(device.macAddress, device.lastSeen);
  } else {
    // Minor update - update the existing device element in place
    updateDeviceElementInPlace(device);
    // For minor updates, also ensure pool membership is up to date
    uiState.poolMembership[device.macAddress] = poolMembership;
  }
  
  return device;
}

// Update a device across all pools where it should appear
function updateDeviceInPools(device, poolMembership, isNewDevice, membershipChanged) {
  const macAddress = device.macAddress;
  
  // If device changed pools, remove it from old pools first
  if (membershipChanged && !isNewDevice) {
    const oldMembership = uiState.poolMembership[macAddress] || {};
    
    if (oldMembership.discovered && !poolMembership.discovered) {
      removeDeviceFromPool(macAddress, 'discovered');
    }
    if (oldMembership.managed && !poolMembership.managed) {
      removeDeviceFromPool(macAddress, 'managed');
    }
    if (oldMembership.sync && !poolMembership.sync) {
      removeDeviceFromPool(macAddress, 'sync');
    }
  }
  
  // Add device to new pools or update existing ones
  if (poolMembership.discovered) {
    updateDeviceInPool(device, 'discovered');
  }
  if (poolMembership.managed) {
    updateDeviceInPool(device, 'managed');
  }
  if (poolMembership.sync) {
    updateDeviceInPool(device, 'sync');
  }

  // Ensure discovered element is removed if device should no longer appear
  if (!poolMembership.discovered) {
    removeDeviceFromPool(device.macAddress, 'discovered');
  }

  // Clean up tracking for elements that are no longer needed
  if (membershipChanged) {
    cleanupDeviceElements(macAddress, poolMembership);
  }
}

// Update or add a device in a specific pool
function updateDeviceInPool(device, poolType) {
  const macAddress = device.macAddress;
  const container = getPoolContainer(poolType);
  
  if (!container) {
    debugLog(`No pool container found for pool type: ${poolType}`, LOG_LEVELS.ERROR);
    return;
  }
  
  let gridContainer = container.querySelector('.gopro-grid-container');
  
  if (!gridContainer) {
    // Create grid container if it doesn't exist
    debugLog(`Creating grid container for pool: ${poolType}`, LOG_LEVELS.DEBUG);
    container.innerHTML = ''; // Clear any existing content
    gridContainer = document.createElement('div');
    gridContainer.className = 'gopro-grid-container';
    container.appendChild(gridContainer);
  }
  
  // Check if device already exists in this pool
  const existingElement = gridContainer.querySelector(`.device-card[data-mac="${macAddress}"]`);
  
  if (existingElement) {
    // Update existing element
    updateDeviceElement(existingElement, device, poolType === 'sync');
  } else {
    // Create new element
    const deviceElement = createGoProElement(device, poolType === 'sync');
    
    // Insert in the correct position (maintain sorting)
    insertDeviceInCorrectPosition(gridContainer, deviceElement, device, poolType);
    
    // Track the element
    if (!uiState.deviceElements[macAddress]) {
      uiState.deviceElements[macAddress] = {};
    }
    uiState.deviceElements[macAddress][poolType] = deviceElement;
    
    // Remove empty state if present
    removeEmptyState(gridContainer);
  }
}

// Remove a device from a specific pool
function removeDeviceFromPool(macAddress, poolType) {
  const container = getPoolContainer(poolType);
  
  if (!container) {
    debugLog(`No pool container found for pool type: ${poolType}`, LOG_LEVELS.ERROR);
    return;
  }
  
  const element = container.querySelector(`.device-card[data-mac="${macAddress}"]`);
  
  if (element) {
    element.remove();
    
    // Clean up tracking
    if (uiState.deviceElements[macAddress]) {
      delete uiState.deviceElements[macAddress][poolType];
      if (Object.keys(uiState.deviceElements[macAddress]).length === 0) {
        delete uiState.deviceElements[macAddress];
      }
    }
    
    // Check if we need to show empty state
    const gridContainer = container.querySelector('.gopro-grid-container');
    if (gridContainer && gridContainer.children.length === 0) {
      showEmptyState(gridContainer, poolType);
    }
  }
}

// Get the container element for a specific pool type
function getPoolContainer(poolType) {
  let container;
  switch (poolType) {
    case 'discovered': 
      container = pairQueueContainer;
      break;
    case 'managed': 
      container = camerasPoolContainer;
      break;
    case 'sync': 
      container = syncQueueContainer;
      break;
    default: 
      debugLog(`Unknown pool type: ${poolType}`, LOG_LEVELS.ERROR);
      return null;
  }
  
  if (!container) {
    debugLog(`Container for pool type '${poolType}' is null - DOM may not be ready`, LOG_LEVELS.ERROR);
  }
  
  return container;
}

// Insert device element in the correct sorted position
function insertDeviceInCorrectPosition(gridContainer, deviceElement, device, poolType) {
  const children = Array.from(gridContainer.children);
  let insertIndex = children.length;
  
  // Sort by signal strength for discovered pool, by name for others
  if (poolType === 'discovered') {
    // Sort by RSSI (highest first)
    for (let i = 0; i < children.length; i++) {
      const childMac = children[i].getAttribute('data-mac');
      const childDevice = allDevices[childMac];
      if (childDevice && (device.rssi || -100) > (childDevice.rssi || -100)) {
        insertIndex = i;
        break;
      }
    }
  } else {
    // Sort by name (alphabetical)
    for (let i = 0; i < children.length; i++) {
      const childMac = children[i].getAttribute('data-mac');
      const childDevice = allDevices[childMac];
      if (childDevice && (getDeviceDisplayName(device) || '').localeCompare(getDeviceDisplayName(childDevice) || '') < 0) {
        insertIndex = i;
        break;
      }
    }
  }
  
  if (insertIndex >= children.length) {
    gridContainer.appendChild(deviceElement);
  } else {
    gridContainer.insertBefore(deviceElement, children[insertIndex]);
  }
}

// Update an existing device element in place
function updateDeviceElement(element, device, inSyncQueue = false) {
  const macAddress = device.macAddress;
  
  // Update device name and status badge
  const nameElement = element.querySelector('.device-name');
  if (nameElement) {
    // Clear existing content
    nameElement.innerHTML = '';
    nameElement.textContent = getDeviceDisplayName(device);
    
    // Add status badge
    const statusText = formatStatus(getStatusShortcode(device));
    const statusBadge = document.createElement('span');
    statusBadge.className = `status-badge status-${getStatusShortcode(device).toLowerCase()}`;
    statusBadge.textContent = statusText;
    nameElement.appendChild(document.createTextNode(' '));
    nameElement.appendChild(statusBadge);
  }
  
  // Update device model
  const modelElement = element.querySelector('.device-model');
  if (modelElement) {
    modelElement.textContent = device.model || 'Unknown Model';
  }
  
  // Update firmware
  const firmwareElement = element.querySelector('.device-firmware');
  if (firmwareElement) {
    firmwareElement.textContent = device.firmwareVersion || 'Unknown FW';
  }
  
  // Update signal strength
  updateSignalStrengthInElement(element, device.rssi || -100);
  
  // Update battery if available
  const batteryElement = element.querySelector('.device-battery');
  if (batteryElement && device.batteryLevel !== undefined) {
    batteryElement.innerHTML = '';
    const batteryIndicator = createBatteryIndicator(device.batteryLevel, device.isCharging);
    batteryElement.appendChild(batteryIndicator);
  }
  
  // Update status/timestamp
  const statusElement = element.querySelector('.device-status');
  if (statusElement) {
    const lastSeen = device.lastSeen ? `Last seen: ${formatTimeSince(device.lastSeen)}` : '';
    statusElement.textContent = lastSeen;
    statusElement.className = `device-status status-${getStatusShortcode(device).toLowerCase()}`;
  }
  
  // Update status class on main element
  const statusCode = getStatusShortcode(device);
  element.className = element.className.replace(/status-\w+/g, '');
  element.classList.add(`status-${statusCode.toLowerCase()}`);
  
  // Update sync progress bar
  updateSyncProgress(element, device, inSyncQueue);
  
  // Update buttons and toggles
  updateDeviceControls(element, device, inSyncQueue);
}

// Update sync progress bar for a device
function updateSyncProgress(element, device, inSyncQueue = false) {
  const progressContainer = element.querySelector('.sync-progress');
  const progressBar = element.querySelector('.sync-progress-fill');
  const progressOperation = element.querySelector('.sync-progress-operation');
  
  if (!progressContainer || !progressBar || !progressOperation) {
    return; // Progress elements not found
  }
  
  // Check if device is currently syncing
  const isSyncing = device.isSyncing;
  
  if (isSyncing) {
    // Find the sync queue entry for this device to get progress details
    const syncEntry = syncQueue.find(entry => entry.cameraId === device.id);
    
    if (syncEntry) {
      const progressPercent = syncEntry.progressPercent || 0;
      const currentOperation = syncEntry.currentOperation || 'Preparing...';
      
      // Update progress bar
      progressBar.style.width = `${progressPercent}%`;
      
      // Show current operation
      progressOperation.textContent = currentOperation;
      
      // Show progress container
      progressContainer.classList.add('active');
      
      debugLog(`Updated sync progress for ${device.name}: ${progressPercent}% - ${currentOperation}`, LOG_LEVELS.TRACE);
    } else {
      // Device is syncing but no queue entry found, show indeterminate state
      progressBar.style.width = '100%';
      progressOperation.textContent = 'Processing...';
      progressContainer.classList.add('active');
    }
  } else {
    // Hide progress container when not syncing
    progressContainer.classList.remove('active');
  }
}

// Update sync progress bars for all syncing devices
function updateAllSyncProgressBars() {
  // Find all device cards that currently exist in the DOM
  const deviceCards = document.querySelectorAll('.device-card[data-mac]');
  
  deviceCards.forEach(card => {
    const macAddress = card.getAttribute('data-mac');
    const device = allDevices[macAddress];
    
    if (device && device.isSyncing) {
      // Check if we're in the sync queue (this determines if we use inSyncQueue parameter)
      const inSyncQueue = shouldShowInSyncQueue(device);
      updateSyncProgress(card, device, inSyncQueue);
    }
  });
}

// Update device controls (buttons and toggles)
function updateDeviceControls(element, device, inSyncQueue = false) {
  const macAddress = device.macAddress;
  const pairButton = element.querySelector('.pair-button');
  const syncButton = element.querySelector('.sync-button');
  const manageToggle = element.querySelector('.manage-toggle');
  
  // Check if device is in sync queue (has an entry in syncQueue array)
  const isInSyncQueue = syncQueue.some(entry => entry.cameraId === device.id);
  
  // Configure pair button
  if (pairButton) {
    // Clear existing event listeners by cloning the element
    const newPairButton = pairButton.cloneNode(true);
    pairButton.parentNode.replaceChild(newPairButton, pairButton);
    
    if (isInSyncQueue) {
      // Device is in sync queue - show Cancel button
      newPairButton.textContent = "Cancel";
      newPairButton.disabled = false;
      newPairButton.classList.remove('in-progress');
      newPairButton.classList.add('cancel-button');
      newPairButton.addEventListener('click', () => {
        newPairButton.textContent = "Cancelling...";
        newPairButton.disabled = true;
        newPairButton.classList.add('in-progress');
        cancelSync(macAddress);
      });
    } else if (pairingInProgress[macAddress]) {
      newPairButton.textContent = "Pairing...";
      newPairButton.disabled = true;
      newPairButton.classList.add('in-progress');
      newPairButton.classList.remove('cancel-button');
    } else if (!device.isPaired) {
      newPairButton.textContent = "Pair";
      newPairButton.disabled = false;
      newPairButton.classList.remove('in-progress', 'cancel-button');
      newPairButton.addEventListener('click', () => {
        newPairButton.textContent = "Pairing...";
        newPairButton.disabled = true;
        newPairButton.classList.add('in-progress');
        pairDevice(macAddress);
      });
    } else {
      newPairButton.textContent = "Paired";
      newPairButton.disabled = true;
      newPairButton.classList.remove('in-progress', 'cancel-button');
    }
  }
  
  // Configure sync button
  if (syncButton) {
    const newSyncButton = syncButton.cloneNode(true);
    syncButton.parentNode.replaceChild(newSyncButton, syncButton);
    
    if (inSyncQueue) {
      newSyncButton.textContent = "In Queue";
      newSyncButton.disabled = true;
      newSyncButton.classList.add('queued');
    } else if (device.isPaired && device.isManaged) {
      newSyncButton.textContent = "Sync";
      newSyncButton.disabled = false;
      newSyncButton.classList.remove('in-progress', 'queued');
      newSyncButton.addEventListener('click', () => {
        newSyncButton.textContent = "Syncing...";
        newSyncButton.disabled = true;
        newSyncButton.classList.add('in-progress');
        addToSyncQueue(macAddress);
      });
    } else {
      newSyncButton.disabled = true;
      newSyncButton.classList.remove('in-progress', 'queued');
    }
  }
  
  // Configure manage toggle
  if (manageToggle) {
    const newManageToggle = manageToggle.cloneNode(true);
    manageToggle.parentNode.replaceChild(newManageToggle, manageToggle);
    
    newManageToggle.checked = !!device.isManaged;
    newManageToggle.addEventListener('change', (event) => {
      toggleDeviceManaged(macAddress, event.target.checked);
    });
  }
}

// Update just the signal strength in a specific element
function updateSignalStrengthInElement(element, rssi) {
  const signalElement = element.querySelector('.device-signal');
  if (signalElement) {
    const signalBarsContainer = signalElement.querySelector('.signal-bars');
    const rssiValueElement = signalElement.querySelector('.rssi-value');
    
    if (signalBarsContainer) {
      const bars = signalBarsContainer.querySelectorAll('.signal-bar');
      
      // Clear existing filled state
      bars.forEach(bar => bar.classList.remove('filled'));
      
      // Calculate and fill bars based on RSSI
      const normalizedRssi = Math.min(Math.max(rssi, -100), -30);
      const signalStrength = Math.floor(((normalizedRssi + 100) / 70) * 4);
      
      for (let i = 0; i < signalStrength; i++) {
        if (bars[i]) {
          bars[i].classList.add('filled');
        }
      }
    }
    
    if (rssiValueElement) {
      rssiValueElement.textContent = `${rssi} dBm`;
    }
  }
}

// Update just the timestamp for a device
function updateDeviceTimestamp(macAddress, lastSeen) {
  const deviceElements = uiState.deviceElements[macAddress];
  if (deviceElements) {
    Object.values(deviceElements).forEach(element => {
      const statusElement = element.querySelector('.device-status');
      if (statusElement && lastSeen) {
        const lastSeenText = `Last seen: ${formatTimeSince(lastSeen)}`;
        statusElement.textContent = lastSeenText;
      }
    });
  }
}

// Update a device element in place for minor changes
function updateDeviceElementInPlace(device) {
  const macAddress = device.macAddress;
  const deviceElements = uiState.deviceElements[macAddress];
  
  if (deviceElements) {
    Object.entries(deviceElements).forEach(([poolType, element]) => {
      updateDeviceElement(element, device, poolType === 'sync');
    });
  }
}

// Clean up device element tracking
function cleanupDeviceElements(macAddress, currentMembership) {
  if (uiState.deviceElements[macAddress]) {
    Object.keys(uiState.deviceElements[macAddress]).forEach(poolType => {
      if (!currentMembership[poolType]) {
        delete uiState.deviceElements[macAddress][poolType];
      }
    });
    
    if (Object.keys(uiState.deviceElements[macAddress]).length === 0) {
      delete uiState.deviceElements[macAddress];
    }
  }
}

// Show empty state in a container
function showEmptyState(gridContainer, poolType) {
  if (!gridContainer) {
    debugLog(`Cannot show empty state: gridContainer is null for pool type: ${poolType}`, LOG_LEVELS.ERROR);
    return;
  }
  
  const emptyState = document.createElement('div');
  emptyState.className = 'empty-state';
  
  const logo = document.createElement('img');
  logo.src = 'imgs/3_dropz.svg';
  logo.alt = 'Dropz Logo';
  logo.className = 'empty-state-logo';
  
  const message = document.createElement('p');
  switch (poolType) {
    case 'discovered':
      message.textContent = 'No GoPro devices detected';
      break;
    case 'managed':
      message.textContent = 'No managed GoPro devices';
      break;
    case 'sync':
      message.textContent = 'No GoPro devices in sync queue';
      break;
    default:
      message.textContent = 'No devices';
  }
  
  emptyState.appendChild(logo);
  emptyState.appendChild(message);
  gridContainer.appendChild(emptyState);
}

// Remove empty state from a container
function removeEmptyState(gridContainer) {
  if (!gridContainer) {
    debugLog(`Cannot remove empty state: gridContainer is null`, LOG_LEVELS.ERROR);
    return;
  }
  
  const emptyState = gridContainer.querySelector('.empty-state');
  if (emptyState) {
    emptyState.remove();
  }
}

// Update the counters display for the three lists
function updateCounters() {
  // Count devices in each category using our helper functions
  let managedCount = Object.values(allDevices).filter(device => shouldShowInManagedPool(device)).length;
  let discoveredCount = Object.values(allDevices).filter(device => shouldShowInDiscoveredPool(device)).length;
  let syncCount = Object.values(allDevices).filter(device => shouldShowInSyncQueue(device)).length;
  
  // Update displays
  managedCountDisplay.textContent = managedCount;
  unpairedCountDisplay.textContent = discoveredCount;
  syncCountDisplay.textContent = syncCount;
  
  // Update our cached counts
  uiState.currentDeviceCounts = {
    total: Object.keys(allDevices).length,
    discovered: discoveredCount,
    managed: managedCount,
    sync: syncCount
  };
}

// Update UI for pair queue
function updatePairQueueUI() {
  // Clear existing content
  pairQueueContainer.innerHTML = '';
  
  // Add wrapper for grid layout
  const gridContainer = document.createElement('div');
  gridContainer.className = 'gopro-grid-container';
  pairQueueContainer.appendChild(gridContainer);
  
  // Use exact boolean-based filtering from database criteria
  let discoveredDevices = Object.values(allDevices).filter(device => 
    device.isReachable && 
    (!device.isPaired || !device.isManaged)
  );
  
  // Add debugging to see which devices are being considered and why (only log when filter changes)
  const filterKey = `discovered-${discoveredDevices.length}-${Object.values(allDevices).length}`;
  if (uiState.lastFilterKey !== filterKey) {
    debugLog(`Discovered pool filtering - Total devices: ${Object.values(allDevices).length}, Shown: ${discoveredDevices.length}`, LOG_LEVELS.DEBUG);
    uiState.lastFilterKey = filterKey;
  }
  
  // Sort by signal strength
  discoveredDevices.sort((a, b) => (b.rssi || -100) - (a.rssi || -100));
  
  if (discoveredDevices.length === 0) {
    showEmptyState(gridContainer, 'discovered');
  } else {
    // Add each discovered device to the UI
    discoveredDevices.forEach(device => {
      const deviceElement = createGoProElement(device);
      gridContainer.appendChild(deviceElement);
      
      // Track the element for future targeted updates
      if (!uiState.deviceElements[device.macAddress]) {
        uiState.deviceElements[device.macAddress] = {};
      }
      uiState.deviceElements[device.macAddress].discovered = deviceElement;
    });
  }
}

// Update UI for cameras pool
function updateCamerasPoolUI() {
  // Clear existing content
  camerasPoolContainer.innerHTML = '';
  
  // Add wrapper for grid layout
  const gridContainer = document.createElement('div');
  gridContainer.className = 'gopro-grid-container';
  camerasPoolContainer.appendChild(gridContainer);
  
  // Use exact boolean-based filtering from database criteria
  let managedDevices = Object.values(allDevices).filter(device => 
    device.isReachable && 
    device.isPaired && 
    device.isManaged
  );
  
  // Sort by name
  managedDevices.sort((a, b) => (a.name || '').localeCompare(b.name || ''));
  
  if (managedDevices.length === 0) {
    showEmptyState(gridContainer, 'managed');
  } else {
    // Add each managed device to the UI
    managedDevices.forEach(device => {
      const deviceElement = createGoProElement(device);
      gridContainer.appendChild(deviceElement);
      
      // Track the element for future targeted updates
      if (!uiState.deviceElements[device.macAddress]) {
        uiState.deviceElements[device.macAddress] = {};
      }
      uiState.deviceElements[device.macAddress].managed = deviceElement;
    });
  }
}

// Update UI for sync queue
function updateSyncQueueUI() {
  // Clear existing content
  syncQueueContainer.innerHTML = '';
  
  // Add wrapper for grid layout
  const gridContainer = document.createElement('div');
  gridContainer.className = 'gopro-grid-container';
  syncQueueContainer.appendChild(gridContainer);
  
  // Use new boolean-based filtering from database criteria
  let syncDevices = Object.values(allDevices).filter(device => 
    device.isReachable && 
    device.isPaired && 
    device.isManaged && 
    !device.isSynced
  );

  if (syncDevices.length === 0) {
    showEmptyState(gridContainer, 'sync');
  } else {
    // Add each device in sync queue to the UI
    syncDevices.forEach(device => {
      // Create and add UI element
      const deviceElement = createGoProElement(device, true);
      gridContainer.appendChild(deviceElement);
      
      // Track the element for future targeted updates
      if (!uiState.deviceElements[device.macAddress]) {
        uiState.deviceElements[device.macAddress] = {};
      }
      uiState.deviceElements[device.macAddress].sync = deviceElement;
    });
  }
}

// Update to createGoProElement function to match the new template structure
function createGoProElement(device, inSyncQueue = false) {
  // Clone the template
  const template = goProItemTemplate.content.cloneNode(true);
  const element = template.querySelector('.device-card');
  
  // Get device status
  const statusCode = getStatusShortcode(device);
  const formattedStatus = formatStatus(statusCode);
  debugLog(`Device ${device.name}: visualStatus=${device.visualStatus}, formatted=${formattedStatus}`, LOG_LEVELS.TRACE);
  
  // Add status class to the device card for status-specific styling
  element.classList.add(`status-${statusCode.toLowerCase()}`);
  
  // Set data attribute for the device MAC
  element.setAttribute('data-mac', device.macAddress);
  
  // Set device name
  const nameElement = element.querySelector('.device-name');
  if (nameElement) {
    nameElement.textContent = getDeviceDisplayName(device);
    
    // Add status badge next to the device name
    const statusText = formatStatus(getStatusShortcode(device));
    const statusBadge = document.createElement('span');
    statusBadge.className = `status-badge status-${getStatusShortcode(device).toLowerCase()}`;
    statusBadge.textContent = statusText;
    nameElement.appendChild(document.createTextNode(' '));
    nameElement.appendChild(statusBadge);
  }
  
  // Set device MAC address
  const macElement = element.querySelector('.device-mac');
  if (macElement) {
    macElement.textContent = formatMacAddress(device.macAddress);
  }
  
  // Set model info if available
  const modelElement = element.querySelector('.device-model');
  if (modelElement) {
    modelElement.textContent = device.model || 'Unknown Model';
  }
  
  // Set firmware if available
  const firmwareElement = element.querySelector('.device-firmware');
  if (firmwareElement) {
    firmwareElement.textContent = device.firmwareVersion || 'Unknown FW';
  }
  
  // Create and append signal strength indicator
  const signalElement = element.querySelector('.device-signal');
  if (signalElement) {
    const rssi = device.rssi || -100;
    const signalBars = createSignalStrengthIndicator(rssi);
    const rssiValue = document.createElement('span');
    rssiValue.textContent = `${rssi} dBm`;
    rssiValue.classList.add('rssi-value');
    signalElement.appendChild(signalBars);
    signalElement.appendChild(rssiValue);
  }
  
  // Create and append battery indicator if available
  const batteryElement = element.querySelector('.device-battery');
  if (batteryElement && device.batteryLevel !== undefined) {
    const batteryIndicator = createBatteryIndicator(device.batteryLevel, device.isCharging);
    batteryElement.appendChild(batteryIndicator);
  }
  
  // Set status details (last seen)
  const statusElement = element.querySelector('.device-status');
  if (statusElement) {
    // We'll use this for additional status details
    const status = getStatusShortcode(device);
    const lastSeen = device.lastSeen ? `Last seen: ${formatTimeSince(device.lastSeen)}` : '';
    statusElement.textContent = lastSeen;
    statusElement.classList.add(`status-${status.toLowerCase()}`);
  }
  
  // Configure buttons and toggles based on device state
  const pairButton = element.querySelector('.pair-button');
  const syncButton = element.querySelector('.sync-button');
  const manageToggle = element.querySelector('.manage-toggle');
  
  // Configure pair button
  if (pairButton) {
    // Check if device is in sync queue (has an entry in syncQueue array)
    const isInSyncQueue = syncQueue.some(entry => entry.cameraId === device.id);
    
    if (isInSyncQueue) {
      // Device is in sync queue - show Cancel button
      pairButton.textContent = "Cancel";
      pairButton.disabled = false;
      pairButton.classList.remove('in-progress');
      pairButton.classList.add('cancel-button');
      pairButton.addEventListener('click', () => {
        pairButton.textContent = "Cancelling...";
        pairButton.disabled = true;
        pairButton.classList.add('in-progress');
        cancelSync(device.macAddress);
      });
    }
    // Check if we're already pairing this device
    else if (pairingInProgress[device.macAddress]) {
      pairButton.textContent = "Pairing...";
      pairButton.disabled = true;
      pairButton.classList.add('in-progress');
      pairButton.classList.remove('cancel-button');
    }
    // Check if device is discovered (not paired)
    else if (!device.isPaired) {
      pairButton.textContent = "Pair";
      pairButton.disabled = false;
      pairButton.classList.remove('in-progress', 'cancel-button');
      pairButton.addEventListener('click', () => {
        // Update button state immediately
        pairButton.textContent = "Pairing...";
        pairButton.disabled = true;
        pairButton.classList.add('in-progress');
        // Initiate pairing
        pairDevice(device.macAddress);
      });
    } else if (device.isPairing) {
      pairButton.textContent = "Pairing...";
      pairButton.disabled = true;
      pairButton.classList.add('in-progress');
      pairButton.classList.remove('cancel-button');
    } else {
      pairButton.textContent = "Paired";
      pairButton.disabled = true;
      pairButton.classList.remove('in-progress', 'cancel-button');
    }
  }
  
  // Configure sync button
  if (syncButton) {
    if (inSyncQueue) {
      syncButton.textContent = "In Queue";
      syncButton.disabled = true;
      syncButton.classList.add('queued');
    } else if (device.isPaired && device.isManaged) {
      syncButton.textContent = "Sync";
      syncButton.disabled = false;
      syncButton.classList.remove('in-progress', 'queued');
      syncButton.addEventListener('click', () => {
        syncButton.textContent = "Syncing...";
        syncButton.disabled = true;
        syncButton.classList.add('in-progress');
        addToSyncQueue(device.macAddress);
      });
    } else {
      syncButton.disabled = true;
      syncButton.classList.remove('in-progress', 'queued');
    }
  }
  
  // Configure manage toggle
  if (manageToggle) {
    manageToggle.checked = !!device.isManaged;
    manageToggle.addEventListener('change', (event) => {
      toggleDeviceManaged(device.macAddress, event.target.checked);
    });
  }
  
  // Initialize sync progress bar
  updateSyncProgress(element, device, inSyncQueue);
  
  return element;
}

// Helper function to create signal strength indicator
function createSignalStrengthIndicator(rssi) {
  const signalContainer = document.createElement('div');
  signalContainer.classList.add('signal-bars');
  
  // Create 4 bars
  for (let i = 0; i < 4; i++) {
    const bar = document.createElement('div');
    bar.classList.add('signal-bar');
    signalContainer.appendChild(bar);
  }
  
  // Fill bars based on signal strength
  const bars = signalContainer.querySelectorAll('.signal-bar');
  const normalizedRssi = Math.min(Math.max(rssi, -100), -30);
  const signalStrength = Math.floor(((normalizedRssi + 100) / 70) * 4);
  
  for (let i = 0; i < signalStrength; i++) {
    bars[i].classList.add('filled');
  }
  
  return signalContainer;
}

// Helper function to create battery indicator
function createBatteryIndicator(level, isCharging = false) {
  const batteryContainer = document.createElement('div');
  batteryContainer.classList.add('battery-indicator');
  
  const batteryIcon = document.createElement('div');
  batteryIcon.classList.add('battery-icon');
  if (isCharging) {
    batteryIcon.classList.add('charging');
  }
  
  const batteryFill = document.createElement('div');
  batteryFill.classList.add('battery-fill');
  
  // Set battery fill level and color class
  const fillLevel = Math.max(0, Math.min(100, level));
  batteryFill.style.width = `${fillLevel}%`;
  
  if (fillLevel < 20) {
    batteryFill.classList.add('battery-low');
  } else if (fillLevel < 50) {
    batteryFill.classList.add('battery-medium');
  } else {
    batteryFill.classList.add('battery-good');
  }
  
  batteryIcon.appendChild(batteryFill);
  batteryContainer.appendChild(batteryIcon);
  
  const batteryText = document.createElement('span');
  batteryText.textContent = `${fillLevel}%`;
  batteryContainer.appendChild(batteryText);
  
  return batteryContainer;
}

// Helper function to format MAC address with colons
function formatMacAddress(macAddress) {
  if (!macAddress) return 'Unknown';
  // Format MAC address with colons (if not already formatted)
  if (macAddress.includes(':')) return macAddress;
  return macAddress.match(/.{1,2}/g).join(':');
}

// Helper function to get the display name for a device
// Uses wifi_ssid (max 12 chars) if available, otherwise falls back to device name
function getDeviceDisplayName(device) {
  if (device.wifiSsid && device.wifiSsid.trim() !== '') {
    // Use WiFi SSID but limit to 12 characters
    return device.wifiSsid.substring(0, 12);
  }
  // Fall back to device name or default
  return device.name || 'Unknown GoPro';
}

// Helper function to get a short status code for compact display
function getStatusShortcode(device) {
  // First check for special statuses
  if (!device.isReachable) return 'unreachable';
  if (device.isPairing) return 'pairing';
  if (device.isSyncing) return 'syncing';
  
  // Then determine visual status based on paired/managed states
  if (!device.isPaired && device.isManaged) return 'unavailable';
  if (!device.isPaired) return 'discovered';
  if (device.isPaired && !device.isManaged) return 'available';
  if (device.isPaired && device.isManaged) return 'managed';
  
  // Fallback to standard visuals
  const status = (device.visualStatus || '').toLowerCase();
  switch (status) {
    case 'discovered': return 'discovered';
    case 'managed': return 'managed';
    case 'pairing': return 'pairing';
    case 'paired': return 'paired';
    case 'unavailable': return 'unavailable';
    case 'available': return 'available';
    case 'syncing': return 'syncing';
    case 'unreachable': return 'unreachable';
    default: return 'unknown';
  }
}

// Format camera status for display
function formatStatus(status) {
  // Accept either a visualStatus string or a shortcode
  if (typeof status === 'string') {
    switch (status.toLowerCase()) {
      case 'd': 
      case 'discovered': 
        return 'Discovered';
        
      case 'm': 
      case 'managed': 
        return 'Managed';
        
      case 'pairing': 
        return 'Pairing';
        
      case 'paired': 
        return 'Paired';
        
      case 'unavailable': 
        return 'Unavailable';
        
      case 'available': 
        return 'Available';
        
      case 'syncing': 
        return 'Syncing';
        
      case 'unreachable': 
        return 'Unreachable';
        
      case 'unknown':
        return 'Unknown';
        
      default:
        if (status.includes('_')) {
          return status.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()).join(' ');
        }
        return status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();
    }
  }
  return 'Unknown';
}

// Pair a device
function pairDevice(macAddress) {
  console.log("pairDevice function called with macAddress:", macAddress);
  
  const device = allDevices[macAddress];
  if (!device || !device.id) {
    console.log("Device not found or missing ID:", device);
    debugLog(`Cannot pair device, no device ID found for ${macAddress}`, LOG_LEVELS.ERROR);
    resetPairButtonState(macAddress);
    return;
  }
  
  console.log("Attempting to pair device:", device.name, "ID:", device.id, "Status:", device.status);
  debugLog(`Pairing device: ${device.name} (${device.id})`, LOG_LEVELS.INFO);
  addLogEntry(`Attempting to pair ${getDeviceDisplayName(device)}`, 'info');
  
  // Mark device as pairing in progress
  pairingInProgress[macAddress] = true;
  
  // Update device state in UI immediately
  updateDeviceStatus(macAddress, 'pairing');
  updateDeviceLists();
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the pairing request - this should work whether the device is managed or not
    const pairRequest = new grpcUtils.PairCameraRequest();
    pairRequest.setCameraId(device.id);
    
    // Call the pairing service
    client.pairCamera(pairRequest, (pairError, pairResponse) => {
      // No longer pairing
      pairingInProgress[macAddress] = false;
      
      if (pairError) {
        debugLog(`Error pairing device: ${pairError.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to pair ${getDeviceDisplayName(device)}: ${pairError.message}`, 'error');
        resetPairButtonState(macAddress);
        updateDeviceStatus(macAddress, 'discovered');
        updateDeviceLists();
        return;
      }
      
      if (!pairResponse.getSuccess()) {
        debugLog(`Pairing failed: ${pairResponse.getMessage()}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to pair ${getDeviceDisplayName(device)}: ${pairResponse.getMessage()}`, 'error');
        resetPairButtonState(macAddress);
        updateDeviceStatus(macAddress, 'discovered');
        updateDeviceLists();
        return;
      }
      
      // Successfully paired
      debugLog(`Successfully paired device: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`Successfully paired ${getDeviceDisplayName(device)}`, 'success');
      
      // Process the returned camera object
      const pairResultCamera = pairResponse.getCamera();
      if (pairResultCamera) {
        debugLog(`Received updated camera object from pairing response`, LOG_LEVELS.DEBUG);
        const updatedDevice = processManagedCamera(pairResultCamera);
        if (updatedDevice) {
          // Log the updated device state for debugging
          debugLog(`Updated device state: isPaired=${updatedDevice.isPaired}, isManaged=${updatedDevice.isManaged}, isReachable=${updatedDevice.isReachable}`, LOG_LEVELS.DEBUG);
          addOrUpdateDevice(updatedDevice);
        } else {
          debugLog(`Failed to process paired camera - result was null`, LOG_LEVELS.ERROR);
          // Still update the local device state since backend reported success
          device.isPaired = true;
          device.isPairing = false;
          addOrUpdateDevice(device);
        }
      } else {
        debugLog(`Pairing succeeded but no camera object was returned`, LOG_LEVELS.WARN);
        // Update local state since backend reported success
        device.isPaired = true;
        device.isPairing = false;
        addOrUpdateDevice(device);
      }
      
      // Update UI
      updateDeviceLists();
      
      // Check if we should automatically sync
      if (autoSync && device.isManaged) {
        debugLog(`Auto-sync enabled for managed device, adding ${getDeviceDisplayName(device)} to sync queue`, LOG_LEVELS.INFO);
        addToSyncQueue(macAddress);
      }
    });
  } catch (error) {
    debugLog(`Exception during pairing: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error pairing ${getDeviceDisplayName(device)}: ${error.message}`, 'error');
    pairingInProgress[macAddress] = false;
    resetPairButtonState(macAddress);
    updateDeviceStatus(macAddress, 'discovered');
    updateDeviceLists();
  }
}

// Helper function to reset pair button state after error
function resetPairButtonState(macAddress) {
  // Find the pair button for this device and reset it
  const deviceElement = document.querySelector(`.device-card[data-mac="${macAddress}"]`);
  if (deviceElement) {
    const pairButton = deviceElement.querySelector('.pair-button');
    if (pairButton) {
      pairButton.textContent = "Pair";
      pairButton.disabled = false;
      pairButton.classList.remove('in-progress');
    }
  }
}

// Add to sync queue
function addToSyncQueue(macAddress) {
  const device = allDevices[macAddress];
  if (!device || !device.id) {
    debugLog(`Cannot add to sync queue, no device ID found for ${macAddress}`, LOG_LEVELS.ERROR);
    addLogEntry(`Failed to add device to sync queue: device not found or missing ID`, 'error');
    resetSyncButtonState(macAddress);
    return;
  }
  
  debugLog(`Adding device to sync queue: ${device.name} (${device.id})`, LOG_LEVELS.INFO);
  debugLog(`Device details - MAC: ${macAddress}, isPaired: ${device.isPaired}, isManaged: ${device.isManaged}, ID: ${device.id}`, LOG_LEVELS.DEBUG);
  addLogEntry(`Adding ${getDeviceDisplayName(device)} to sync queue`, 'info');
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.ForceSyncRequest();
    request.setCameraId(device.id);
    
    debugLog(`Sending ForceSync request for camera ID: ${device.id}`, LOG_LEVELS.DEBUG);
    
    // Call the service - this will automatically set isSynced to false in the backend
    client.forceSync(request, (error, response) => {
      if (error) {
        debugLog(`Error adding to sync queue: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to add ${getDeviceDisplayName(device)} to sync queue: ${error.message}`, 'error');
        resetSyncButtonState(macAddress);
        return;
      }
      
      if (!response.getSuccess()) {
        debugLog(`Adding to sync queue failed: ${response.getMessage()}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to add ${getDeviceDisplayName(device)} to sync queue: ${response.getMessage()}`, 'error');
        resetSyncButtonState(macAddress);
        return;
      }
      
      // Successfully added to sync queue
      debugLog(`Successfully added device to sync queue: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`Added ${getDeviceDisplayName(device)} to sync queue`, 'success');
      
      // Update local device state
      device.isSynced = false;
      
      // Process the returned queue entry
      const queueEntry = response.getQueueEntry();
      if (queueEntry) {
        const processedEntry = processSyncQueueEntry(queueEntry);
        if (processedEntry) {
          // Check if the entry is already in the queue
          const existingIndex = syncQueue.findIndex(entry => entry.cameraId === processedEntry.cameraId);
          
          if (existingIndex >= 0) {
            // Update existing entry
            syncQueue[existingIndex] = processedEntry;
          } else {
            // Add new entry
            syncQueue.push(processedEntry);
          }
          
          // Update UI
          updateSyncQueueUI();
          updateCounters();
        }
      }
    });
  } catch (error) {
    debugLog(`Exception during adding to sync queue: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error adding ${getDeviceDisplayName(device)} to sync queue: ${error.message}`, 'error');
    resetSyncButtonState(macAddress);
  }
}

// Helper function to reset sync button state after error
function resetSyncButtonState(macAddress) {
  // Find the sync button for this device and reset it
  const deviceElement = document.querySelector(`.device-card[data-mac="${macAddress}"]`);
  if (deviceElement) {
    const syncButton = deviceElement.querySelector('.sync-button');
    if (syncButton) {
      syncButton.textContent = "Sync Now";  // Changed from "Sync" to "Sync Now"
      syncButton.disabled = false;
      syncButton.classList.remove('in-progress');
    }
  }
}

// Cancel sync for a device and remove from sync queue
function cancelSync(macAddress) {
  const device = allDevices[macAddress];
  if (!device || !device.id) {
    debugLog(`Cannot cancel sync, no device ID found for ${macAddress}`, LOG_LEVELS.ERROR);
    addLogEntry(`Failed to cancel sync: device not found or missing ID`, 'error');
    return;
  }
  
  debugLog(`Cancelling sync for device: ${device.name} (${device.id})`, LOG_LEVELS.INFO);
  addLogEntry(`Cancelling sync for ${getDeviceDisplayName(device)}`, 'info');
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.CancelSyncRequest();
    request.setCameraId(device.id);
    
    debugLog(`Sending CancelSync request for camera ID: ${device.id}`, LOG_LEVELS.DEBUG);
    
    // Call the service
    client.cancelSync(request, (error, response) => {
      if (error) {
        debugLog(`Error cancelling sync: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to cancel sync for ${getDeviceDisplayName(device)}: ${error.message}`, 'error');
        return;
      }
      
      if (!response.getSuccess()) {
        debugLog(`Cancelling sync failed: ${response.getMessage()}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to cancel sync for ${getDeviceDisplayName(device)}: ${response.getMessage()}`, 'error');
        return;
      }
      
      // Successfully cancelled sync
      debugLog(`Successfully cancelled sync for device: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`Cancelled sync for ${getDeviceDisplayName(device)}`, 'success');
      
      // Update local device state - device should be synced now (removed from queue)
      device.isSynced = true;
      device.isSyncing = false;
      
      // Remove from sync queue if present
      const existingIndex = syncQueue.findIndex(entry => entry.cameraId === device.id);
      if (existingIndex >= 0) {
        syncQueue.splice(existingIndex, 1);
      }
      
      // Update UI
      addOrUpdateDevice(device);
      updateSyncQueueUI();
      updateCounters();
    });
  } catch (error) {
    debugLog(`Exception during cancelling sync: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error cancelling sync for ${getDeviceDisplayName(device)}: ${error.message}`, 'error');
  }
}

// Toggle whether a device is managed
function toggleDeviceManaged(macAddress, isManaged) {
  const device = allDevices[macAddress];
  if (!device || !device.id) {
    debugLog(`Cannot ${isManaged ? 'manage' : 'unmanaging'} device, no device ID found for ${macAddress}`, LOG_LEVELS.ERROR);
    return;
  }
  
  debugLog(`${isManaged ? 'Managing' : 'Unmanaging'} device: ${device.name} (${device.id})`, LOG_LEVELS.INFO);
  addLogEntry(`${isManaged ? 'Adding' : 'Removing'} ${getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} camera pool`, 'info');
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    if (isManaged) {
      // Managing device
      const request = new grpcUtils.ManageCameraRequest();
      request.setCameraId(device.id);
      
      client.manageCamera(request, handleResponse);
    } else {
      // Unmanaging device
      const request = new grpcUtils.UnmanageCameraRequest();
      request.setCameraId(device.id);
      
      client.unmanageCamera(request, handleResponse);
    }
    
    function handleResponse(error, response) {
      if (error) {
        debugLog(`Error ${isManaged ? 'managing' : 'unmanaging'} device: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to ${isManaged ? 'add' : 'remove'} ${getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} camera pool: ${error.message}`, 'error');
        return;
      }
      
      if (!response.getSuccess()) {
        debugLog(`${isManaged ? 'Managing' : 'Unmanaging'} failed: ${response.getMessage()}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to ${isManaged ? 'add' : 'remove'} ${getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} camera pool: ${response.getMessage()}`, 'error');
        return;
      }
      
      // Success!
      debugLog(`Successfully ${isManaged ? 'managed' : 'unmanaged'} device: ${device.name}`, LOG_LEVELS.INFO);
      addLogEntry(`${isManaged ? 'Added' : 'Removed'} ${getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} camera pool`, 'success');
      
      // Update local device state immediately to ensure UI reflects the change
      device.isManaged = isManaged;
      if (isManaged) {
        // Managing implies paired
        device.isPaired = true;
      }
      
      // If this was a manage request, process the returned camera
      if (isManaged && response.getCamera) {
        const managedCamera = response.getCamera();
        if (managedCamera) {
          const updatedDevice = processManagedCamera(managedCamera);
          if (updatedDevice) {
            addOrUpdateDevice(updatedDevice);
          }
        }
      } else {
        // For unmanage operations or manage operations without returned camera data,
        // update the device using the local state change
        addOrUpdateDevice(device);
      }
      
      // Auto-pair if enabled
      if (isManaged && autoPair && !(device.isPaired)) {
        debugLog(`Auto-pair enabled, pairing ${getDeviceDisplayName(device)}`, LOG_LEVELS.INFO);
        pairDevice(macAddress);
      }
    }
  } catch (error) {
    debugLog(`Exception during ${isManaged ? 'managing' : 'unmanaging'} device: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error ${isManaged ? 'adding' : 'removing'} ${getDeviceDisplayName(device)} ${isManaged ? 'to' : 'from'} camera pool: ${error.message}`, 'error');
  }
}

// Toggle automatic pairing of all devices
function togglePairAll(event) {
  const enabled = event.target.checked;
  const previousValue = autoPair;
  
  debugLog(`Toggling pair-all mode: ${enabled}`, LOG_LEVELS.INFO);
  
  // Update local state optimistically
  autoPair = enabled;
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.UpdateSettingRequest();
    request.setSettingName('pair_mode_enabled');
    request.setBoolValue(enabled);
    
    // Call the service and handle response
    client.updateSetting(request, (error, response) => {
      if (error) {
        debugLog(`Error updating pair_mode_enabled: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to update auto-pair setting: ${error.message}`, 'error');
        
        // Revert local state and UI on error
        autoPair = previousValue;
        if (pairAllToggle) {
          pairAllToggle.checked = previousValue;
        }
        return;
      }
      
      // Update was successful
      debugLog(`Successfully updated pair_mode_enabled to ${enabled}`, LOG_LEVELS.INFO);
      
      if (enabled) {
        addLogEntry('Auto-pairing enabled', 'info');
        // Start pairing all unpaired devices
        Object.values(allDevices)
          .filter(d => !d.isPaired && !pairingInProgress[d.macAddress])
          .forEach(d => {
            pairDevice(d.macAddress);
          });
      } else {
        addLogEntry('Auto-pairing disabled', 'info');
      }
    });
  } catch (error) {
    debugLog(`Exception updating pair_mode_enabled: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error updating auto-pair setting: ${error.message}`, 'error');
    
    // Revert local state and UI on exception
    autoPair = previousValue;
    if (pairAllToggle) {
      pairAllToggle.checked = previousValue;
    }
  }
}

// Toggle automatic syncing of all devices
function toggleAutoSync(event) {
  const enabled = event.target.checked;
  const previousValue = autoSync;
  
  debugLog(`Toggling auto-sync mode: ${enabled}`, LOG_LEVELS.INFO);
  
  // Update local state optimistically
  autoSync = enabled;
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.UpdateSettingRequest();
    request.setSettingName('sync_enabled');
    request.setBoolValue(enabled);
    
    // Call the service and handle response
    client.updateSetting(request, (error, response) => {
      if (error) {
        debugLog(`Error updating sync_enabled: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to update auto-sync setting: ${error.message}`, 'error');
        
        // Revert local state and UI on error
        autoSync = previousValue;
        if (syncQueueToggle) {
          syncQueueToggle.checked = previousValue;
        }
        return;
      }
      
      // Update was successful
      debugLog(`Successfully updated sync_enabled to ${enabled}`, LOG_LEVELS.INFO);
      
      if (enabled) {
        addLogEntry('Auto-sync enabled', 'info');
        // Start syncing if there are devices in the queue
        if (syncQueue.length > 0 && Object.keys(syncingInProgress).length === 0) {
          startNextSync();
        }
      } else {
        addLogEntry('Auto-sync disabled', 'info');
      }
    });
  } catch (error) {
    debugLog(`Exception updating sync_enabled: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error updating auto-sync setting: ${error.message}`, 'error');
    
    // Revert local state and UI on exception
    autoSync = previousValue;
    if (syncQueueToggle) {
      syncQueueToggle.checked = previousValue;
    }
  }
}

// Update device timestamps for UI refresh without regenerating DOM elements
function updateDeviceTimestamps() {
  // Only update the timestamps in the UI without redrawing everything
  Object.values(allDevices).forEach(device => {
    const element = document.querySelector(`.gopro-item[data-mac="${device.macAddress}"] .gopro-status`);
    if (element) {
      const lastSeenText = formatTimeSince(device.lastSeen);
      const statusText = `Status: ${formatStatus(device.status)} (seen ${lastSeenText})`;
      element.textContent = statusText;
    }
  });
}

// Parse Go log message to determine log level
function parseGoLogLevel(logMessage) {
  // Default level and style
  let result = {
    level: LOG_LEVELS.INFO,
    style: 'info',
    prefix: '[Go]'
  };
  
  // Check for time prefix (common in Go logs) - e.g., 2023/05/20 15:04:05
  const timePrefix = logMessage.match(/^\d{4}[/-]\d{2}[/-]\d{2}[ T]\d{2}:\d{2}:\d{2}/);
  
  // Handle standard Go log format with level prefix
  // Our custom formatter adds level prefixes like [INFO], [DEBUG], etc.
  if (logMessage.includes('[TRACE]')) {
    result.level = LOG_LEVELS.TRACE;
    result.style = 'info';
    result.prefix = '[Go-TRACE]';
    return result;
  }
  
  if (logMessage.includes('[DEBUG]')) {
    result.level = LOG_LEVELS.DEBUG;
    result.style = 'info';
    result.prefix = '[Go-DEBUG]';
    return result;
  }
  
  if (logMessage.includes('[INFO]')) {
    result.level = LOG_LEVELS.INFO;
    result.style = 'info';
    result.prefix = '[Go-INFO]';
    return result;
  }
  
  if (logMessage.includes('[WARN]')) {
    result.level = LOG_LEVELS.WARN;
    result.style = 'warn';
    result.prefix = '[Go-WARN]';
    return result;
  }
  
  if (logMessage.includes('[ERROR]')) {
    result.level = LOG_LEVELS.ERROR;
    result.style = 'error';
    result.prefix = '[Go-ERROR]';
    return result;
  }
  
  if (logMessage.includes('[FATAL]') || logMessage.includes('[PANIC]')) {
    result.level = LOG_LEVELS.ERROR;
    result.style = 'error';
    result.prefix = '[Go-FATAL]';
    return result;
  }
  
  // Handle JSON formatted logs (if somehow our custom formatter isn't used)
  try {
    if (logMessage.trim().startsWith('{') && logMessage.includes('"level"')) {
      const logObj = JSON.parse(logMessage);
      if (logObj.level) {
        switch (logObj.level.toLowerCase()) {
          case 'trace':
            result.level = LOG_LEVELS.TRACE;
            result.style = 'info';
            result.prefix = '[Go-TRACE]';
            break;
          case 'debug':
            result.level = LOG_LEVELS.DEBUG;
            result.style = 'info';
            result.prefix = '[Go-DEBUG]';
            break;
          case 'info':
            result.level = LOG_LEVELS.INFO;
            result.style = 'info';
            result.prefix = '[Go-INFO]';
            break;
          case 'warn':
          case 'warning':
            result.level = LOG_LEVELS.WARN;
            result.style = 'warn';
            result.prefix = '[Go-WARN]';
            break;
          case 'error':
            result.level = LOG_LEVELS.ERROR;
            result.style = 'error';
            result.prefix = '[Go-ERROR]';
            break;
          case 'fatal':
          case 'panic':
            result.level = LOG_LEVELS.ERROR;
            result.style = 'error';
            result.prefix = '[Go-FATAL]';
            break;
        }
        return result;
      }
    }
  } catch (e) {
    // Not valid JSON, continue with pattern matching
  }
  
  // Fall back to pattern matching for non-standard log formats
  // Common Go logging patterns to detect
  if (logMessage.match(/\bERROR\b/i) ||
      logMessage.match(/\berror:/i) ||
      logMessage.includes('panic:') || 
      logMessage.includes('fatal:')) {
    result.level = LOG_LEVELS.ERROR;
    result.style = 'error';
    result.prefix = '[Go-ERROR]';
  } else if (logMessage.match(/\bWARN\b/i) ||
             logMessage.match(/\bwarning:/i)) {
    result.level = LOG_LEVELS.WARN;
    result.style = 'warn';
    result.prefix = '[Go-WARN]';
  } else if (logMessage.match(/\bDEBUG\b/i)) {
    result.level = LOG_LEVELS.DEBUG;
    result.style = 'info';
    result.prefix = '[Go-DEBUG]';
  } else if (logMessage.match(/\bTRACE\b/i) ||
             logMessage.match(/\bVERBOSE\b/i)) {
    result.level = LOG_LEVELS.TRACE;
    result.style = 'info';
    result.prefix = '[Go-TRACE]';
  } else if (logMessage.match(/\bINFO\b/i)) {
    // Explicitly mark as INFO
    result.prefix = '[Go-INFO]';
  }
  
  return result;
}

// Test logging at all levels - useful for debugging
function testLogLevels() {
  // Add a header log entry
  addLogEntry('--- Testing all log levels ---', 'info');
  
  // Log at each level
  debugLog('This is an ERROR level message', LOG_LEVELS.ERROR);
  debugLog('This is a WARNING level message', LOG_LEVELS.WARN);
  debugLog('This is an INFO level message', LOG_LEVELS.INFO);
  debugLog('This is a DEBUG level message', LOG_LEVELS.DEBUG);
  debugLog('This is a TRACE level message', LOG_LEVELS.TRACE);
  
  // Test special formatted Go log style detection
  // This helps verify our Go log parsing is working
  debugLog('[ERROR] This is a test of Go-style ERROR log message format', LOG_LEVELS.INFO);
  debugLog('[WARN] This is a test of Go-style WARN log message format', LOG_LEVELS.INFO);
  debugLog('[INFO] This is a test of Go-style INFO log message format', LOG_LEVELS.INFO);
  debugLog('[DEBUG] This is a test of Go-style DEBUG log message format', LOG_LEVELS.INFO);
  debugLog('[TRACE] This is a test of Go-style TRACE log message format', LOG_LEVELS.INFO);
  
  // Add a separator log entry
  addLogEntry('--- End of log level test ---', 'info');
  
  // Return the current log level for confirmation
  const { currentLogLevel, getLogLevelName } = require('./fileLogger');
  const level = currentLogLevel();
  const levelName = getLogLevelName(level);
  
  addLogEntry(`Current log level: ${levelName} (${level})`, 'info');
  
  return levelName;
}

// Add a debug test button to test log levels
function addDebugControlsToUI() {
  // Create a button to test logs
  const testButton = document.createElement('button');
  testButton.textContent = 'Test Log Levels';
  testButton.className = 'button test-logs-button';
  testButton.addEventListener('click', () => {
    testLogLevels();
  });
  
  // Find a good location for it
  const logLevelContainer = logLevelSelect.parentElement;
  if (logLevelContainer) {
    logLevelContainer.appendChild(testButton);
  }
}

// Toggle Go logs visibility
function toggleGoLogs(event) {
  const showGoLogs = event.target.checked;
  const goLogEntries = document.querySelectorAll('.log-entry[data-source="go"]');
  
  goLogEntries.forEach(entry => {
    entry.style.display = showGoLogs ? 'block' : 'none';
  });
  
  debugLog(`Go logs ${showGoLogs ? 'shown' : 'hidden'}`, LOG_LEVELS.INFO);
}

// Ensure init is called when the page loads
document.addEventListener('DOMContentLoaded', () => {
  debugLog('DOM fully loaded, calling init()');
  init();
  
  // Add the debug controls after a short delay to let everything initialize
  setTimeout(() => {
    addDebugControlsToUI();
  }, 1000);
});

// Export for testing (if needed)
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    startDeviceStreaming,
    handleDeviceStreamUpdate,
    checkGrpcConnection
  };
}

// Get all camera groups
function loadGroups() {
  debugLog('Loading camera groups', LOG_LEVELS.INFO);
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.GetGroupsRequest();
    
    // Call the service
    client.getGroups(request, (error, response) => {
      if (error) {
        debugLog(`Error loading groups: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to load camera groups: ${error.message}`, 'error');
        return;
      }
      
      // Process the groups
      const groupsList = response.getGroupsList();
      groups = groupsList.map(group => ({
        id: group.getId(),
        name: group.getName(),
        cameraIds: group.getCameraIdsList(),
        createdAt: group.getCreatedAt()?.toDate(),
        updatedAt: group.getUpdatedAt()?.toDate()
      }));
      
      debugLog(`Loaded ${groups.length} camera groups`, LOG_LEVELS.INFO);
      
      // Update UI
      updateGroupsUI();
    });
  } catch (error) {
    debugLog(`Exception loading groups: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error loading camera groups: ${error.message}`, 'error');
  }
}

// Create a new camera group
function createGroup(name, cameraIds = []) {
  debugLog(`Creating new group: ${name} with ${cameraIds.length} cameras`, LOG_LEVELS.INFO);
  addLogEntry(`Creating new camera group: ${name}`, 'info');
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.CreateGroupRequest();
    request.setName(name);
    
    // Add camera IDs if provided
    if (cameraIds.length > 0) {
      request.setCameraIdsList(cameraIds);
    }
    
    // Call the service
    client.createGroup(request, (error, response) => {
      if (error) {
        debugLog(`Error creating group: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to create group: ${error.message}`, 'error');
        return;
      }
      
      // Process the created group
      const newGroup = {
        id: response.getId(),
        name: response.getName(),
        cameraIds: response.getCameraIdsList(),
        createdAt: response.getCreatedAt()?.toDate(),
        updatedAt: response.getUpdatedAt()?.toDate()
      };
      
      // Add to groups list
      groups.push(newGroup);
      
      debugLog(`Group created: ${name} (${newGroup.id})`, LOG_LEVELS.INFO);
      addLogEntry(`Created group: ${name}`, 'success');
      
      // Update UI
      updateGroupsUI();
    });
  } catch (error) {
    debugLog(`Exception creating group: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error creating group: ${error.message}`, 'error');
  }
}

// Update an existing camera group
function updateGroup(groupId, name, cameraIds) {
  debugLog(`Updating group ${groupId}: name=${name}, cameras=${cameraIds.length}`, LOG_LEVELS.INFO);
  addLogEntry(`Updating camera group: ${name}`, 'info');
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.UpdateGroupRequest();
    request.setGroupId(groupId);
    request.setName(name);
    request.setCameraIdsList(cameraIds);
    
    // Call the service
    client.updateGroup(request, (error, response) => {
      if (error) {
        debugLog(`Error updating group: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to update group: ${error.message}`, 'error');
        return;
      }
      
      // Process the updated group
      const updatedGroup = {
        id: response.getId(),
        name: response.getName(),
        cameraIds: response.getCameraIdsList(),
        createdAt: response.getCreatedAt()?.toDate(),
        updatedAt: response.getUpdatedAt()?.toDate()
      };
      
      // Update in groups list
      const index = groups.findIndex(g => g.id === groupId);
      if (index >= 0) {
        groups[index] = updatedGroup;
      } else {
        groups.push(updatedGroup);
      }
      
      debugLog(`Group updated: ${name} (${groupId})`, LOG_LEVELS.INFO);
      addLogEntry(`Updated group: ${name}`, 'success');
      
      // Update UI
      updateGroupsUI();
    });
  } catch (error) {
    debugLog(`Exception updating group: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error updating group: ${error.message}`, 'error');
  }
}

// Delete a camera group
function deleteGroup(groupId) {
  const group = groups.find(g => g.id === groupId);
  const groupName = group ? group.name : groupId;
  
  debugLog(`Deleting group: ${groupName} (${groupId})`, LOG_LEVELS.INFO);
  addLogEntry(`Deleting camera group: ${groupName}`, 'info');
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.DeleteGroupRequest();
    request.setGroupId(groupId);
    
    // Call the service
    client.deleteGroup(request, (error, response) => {
      if (error) {
        debugLog(`Error deleting group: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to delete group: ${error.message}`, 'error');
        return;
      }
      
      if (!response.getSuccess()) {
        debugLog(`Deleting group failed: ${response.getMessage()}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to delete group: ${response.getMessage()}`, 'error');
        return;
      }
      
      // Remove from groups list
      groups = groups.filter(g => g.id !== groupId);
      
      debugLog(`Group deleted: ${groupName} (${groupId})`, LOG_LEVELS.INFO);
      addLogEntry(`Deleted group: ${groupName}`, 'success');
      
      // Update UI
      updateGroupsUI();
    });
  } catch (error) {
    debugLog(`Exception deleting group: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error deleting group: ${error.message}`, 'error');
  }
}

// Update the groups UI
function updateGroupsUI() {
  // This is a placeholder - you'll need to implement the actual UI update
  // based on your application's needs and DOM structure
  debugLog(`Groups UI would be updated with ${groups.length} groups`, LOG_LEVELS.TRACE);
}

// Get all synchronized videos
function loadVideos() {
  debugLog('Loading synchronized videos', LOG_LEVELS.INFO);
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.GetVideosRequest();
    // You can add filter parameters here if needed:
    // request.setCameraId(cameraId);
    // request.setStartTime(startTimestamp);
    // request.setEndTime(endTimestamp);
    
    // Call the service
    client.getVideos(request, (error, response) => {
      if (error) {
        debugLog(`Error loading videos: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to load videos: ${error.message}`, 'error');
        return;
      }
      
      // Process the videos
      const videosList = response.getVideosList();
      videos = videosList.map(video => ({
        id: video.getId(),
        cameraId: video.getCameraId(),
        path: video.getPath(),
        filename: video.getFilename(),
        size: video.getSize(),
        duration: video.getDuration(),
        createdAt: video.getCreatedAt()?.toDate(),
        syncedAt: video.getSyncedAt()?.toDate(),
        thumbnailPath: video.getThumbnailPath()
      }));
      
      debugLog(`Loaded ${videos.length} videos`, LOG_LEVELS.INFO);
      
      // Update UI
      updateVideosUI();
    });
  } catch (error) {
    debugLog(`Exception loading videos: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error loading videos: ${error.message}`, 'error');
  }
}

// Get videos for a specific camera
function getVideosForCamera(cameraId) {
  debugLog(`Loading videos for camera: ${cameraId}`, LOG_LEVELS.INFO);
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request with camera filter
    const request = new grpcUtils.GetVideosRequest();
    request.setCameraId(cameraId);
    
    // Call the service
    client.getVideos(request, (error, response) => {
      if (error) {
        debugLog(`Error loading videos for camera: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to load videos for camera: ${error.message}`, 'error');
        return;
      }
      
      // Process the videos
      const videosList = response.getVideosList();
      const cameraVideos = videosList.map(video => ({
        id: video.getId(),
        cameraId: video.getCameraId(),
        path: video.getPath(),
        filename: video.getFilename(),
        size: video.getSize(),
        duration: video.getDuration(),
        createdAt: video.getCreatedAt()?.toDate(),
        syncedAt: video.getSyncedAt()?.toDate(),
        thumbnailPath: video.getThumbnailPath()
      }));
      
      debugLog(`Loaded ${cameraVideos.length} videos for camera ${cameraId}`, LOG_LEVELS.INFO);
      
      // Return the videos for this camera
      return cameraVideos;
    });
  } catch (error) {
    debugLog(`Exception loading videos for camera: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error loading videos for camera: ${error.message}`, 'error');
    return [];
  }
}

// Update the videos UI
function updateVideosUI() {
  // This is a placeholder - you'll need to implement the actual UI update
  // based on your application's needs and DOM structure
  debugLog(`Videos UI would be updated with ${videos.length} videos`, LOG_LEVELS.TRACE);
}

// Load full application configuration
function loadConfig() {
  debugLog('Loading application configuration', LOG_LEVELS.INFO);
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.GetConfigRequest();
    
    // Call the service
    client.getConfig(request, (error, response) => {
      if (error) {
        debugLog(`Error loading configuration: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to load configuration: ${error.message}`, 'error');
        return;
      }
      
      // Process the config
      const config = response.getConfig();
      try {
        appConfig = {
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
        
        // Update the frontend toggle states based on config values
        autoSync = appConfig.syncEnabled;
        autoPair = appConfig.pairModeEnabled;
        
        // Update the UI toggle elements
        if (syncQueueToggle) {
          syncQueueToggle.checked = autoSync;
        }
        if (pairAllToggle) {
          pairAllToggle.checked = autoPair;
        }
        
        debugLog(`Settings synced from backend: autoSync=${autoSync}, autoPair=${autoPair}`, LOG_LEVELS.INFO);
        
      } catch (configError) {
        // If any getter fails, create a partial config
        debugLog(`Error getting some config values: ${configError.message}`, LOG_LEVELS.WARN);
        appConfig = {
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
        
        // Still try to update the toggle states with what we have
        autoSync = appConfig.syncEnabled;
        autoPair = appConfig.pairModeEnabled;
        
        // Update the UI toggle elements
        if (syncQueueToggle) {
          syncQueueToggle.checked = autoSync;
        }
        if (pairAllToggle) {
          pairAllToggle.checked = autoPair;
        }
      }
      
      debugLog(`Loaded configuration: ${JSON.stringify(appConfig)}`, LOG_LEVELS.INFO);
      
      // Update UI
      updateConfigUI();
    });
  } catch (error) {
    debugLog(`Exception loading configuration: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error loading configuration: ${error.message}`, 'error');
  }
}

// Reset all settings to defaults
function resetAllSettings() {
  debugLog('Resetting all settings to defaults', LOG_LEVELS.INFO);
  addLogEntry('Resetting all settings to defaults', 'info');
  
  // Confirm with user
  if (!confirm('Reset all settings to default values?')) {
    return;
  }
  
  // Get all settings from the UI
  const settings = [
    'pair_mode_enabled',
    'sync_enabled',
    'scan_interval_seconds',
    'connect_timeout_seconds',
    'days_threshold',
    'destination_folder',
    'inactivity_timeout_seconds',
    'inactivity_sync_interval_seconds',
    'set_time_enabled',
    'log_level'
  ];
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Reset each setting one by one
    let resetCount = 0;
    
    settings.forEach(settingName => {
      const request = new grpcUtils.ResetSettingRequest();
      request.setSettingName(settingName);
      
      client.resetSetting(request, (error, response) => {
        resetCount++;
        
        if (error) {
          debugLog(`Error resetting setting ${settingName}: ${error.message}`, LOG_LEVELS.ERROR);
          addLogEntry(`Failed to reset setting ${settingName}: ${error.message}`, 'error');
        } else {
          debugLog(`Reset setting ${settingName} to default value`, LOG_LEVELS.INFO);
        }
        
        // When all settings have been reset, reload the config
        if (resetCount === settings.length) {
          addLogEntry('All settings reset to defaults', 'success');
          // Reload the config to update the UI
          loadConfig();
        }
      });
    });
  } catch (error) {
    debugLog(`Exception resetting settings: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error resetting settings: ${error.message}`, 'error');
  }
}

// Update the configuration UI
function updateConfigUI() {
  debugLog('Updating configuration UI', LOG_LEVELS.DEBUG);
  
  // Update the settings panel
  updateSettingsUI();
  
  // Update log level select if it exists
  if (appConfig && appConfig.logLevel && logLevelSelect) {
    const levelValue = appConfig.logLevel.toLowerCase();
    // Find the option with this value
    for (let i = 0; i < logLevelSelect.options.length; i++) {
      if (logLevelSelect.options[i].value.toLowerCase() === levelValue) {
        logLevelSelect.selectedIndex = i;
        break;
      }
    }
  }
  
  // Update backend log level select if it exists
  if (appConfig && appConfig.logLevel) {
    const backendLogLevelSelect = document.getElementById('backend-log-level-select');
    if (backendLogLevelSelect) {
      backendLogLevelSelect.value = appConfig.logLevel.toLowerCase();
    }
  }
}

// Check gRPC connection with a test call
function checkGrpcConnection() {
  return new Promise((resolve, reject) => {
    try {
      const grpcUtils = require('./grpc-utils');
      const client = grpcUtils.getClient();
      
      // Use a simple GetConfig call to check connection
      const request = new grpcUtils.GetConfigRequest();
      
      client.getConfig(request, (error, response) => {
        if (error) {
          reject(error);
        } else {
          resolve(response);
        }
      });
    } catch (error) {
      reject(error);
    }
  });
}

// Simple wrapper function to handle device stream updates
function handleDeviceStreamUpdate(cameras, callback) {
  if (!cameras || cameras.length === 0) {
    return;
  }
  
  cameras.forEach(camera => {
    const cameraObj = callback(camera);
    if (cameraObj) {
      addOrUpdateDevice(cameraObj);
      // addOrUpdateDevice now handles targeted updates and counter updates automatically
    }
  });
}

// Toggle between light and dark themes
function toggleTheme() {
  isDarkMode = !isDarkMode;
  updateTheme();
  saveThemePreference();
}

// Update the theme based on the current state
function updateTheme() {
  const htmlElement = document.documentElement;
  const themeIcon = themeToggle.querySelector('i');
  
  if (isDarkMode) {
    htmlElement.setAttribute('data-theme', 'dark');
    themeIcon.className = 'fas fa-sun';
  } else {
    htmlElement.removeAttribute('data-theme');
    themeIcon.className = 'fas fa-moon';
  }
  
  debugLog(`Theme switched to ${isDarkMode ? 'dark' : 'light'} mode`, LOG_LEVELS.INFO);
}

// Save the theme preference to localStorage
function saveThemePreference() {
  try {
    localStorage.setItem('darkMode', isDarkMode ? 'true' : 'false');
  } catch (error) {
    debugLog(`Error saving theme preference: ${error.message}`, LOG_LEVELS.ERROR);
  }
}

// Load theme preference from localStorage
function loadThemePreference() {
  try {
    const savedPreference = localStorage.getItem('darkMode');
    if (savedPreference !== null) {
      isDarkMode = savedPreference === 'true';
    } else {
      // If no saved preference, check system preference
      const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
      isDarkMode = prefersDark;
    }
    
    // Apply the theme
    updateTheme();
  } catch (error) {
    debugLog(`Error loading theme preference: ${error.message}`, LOG_LEVELS.ERROR);
  }
}

// Format time since a given date for a user-friendly display
function formatTimeSince(date) {
  if (!date) return 'unknown';
  
  // Convert to date object if it's a string
  if (typeof date === 'string') {
    date = new Date(date);
  }
  
  // Get seconds difference
  const seconds = Math.floor((new Date() - date) / 1000);
  
  // Less than a minute
  if (seconds < 60) {
    return 'just now';
  }
  
  // Less than an hour
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes} ${minutes === 1 ? 'minute' : 'minutes'} ago`;
  }
  
  // Less than a day
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours} ${hours === 1 ? 'hour' : 'hours'} ago`;
  }
  
  // Less than a week
  const days = Math.floor(hours / 24);
  if (days < 7) {
    return `${days} ${days === 1 ? 'day' : 'days'} ago`;
  }
  
  // Format as date if more than a week ago
  return date.toLocaleDateString();
}

// Setup settings panel handlers
function setupSettingsHandlers() {
  // Get all toggle settings
  const toggleSettings = document.querySelectorAll('.setting-toggle');
  toggleSettings.forEach(toggle => {
    toggle.addEventListener('change', handleSettingChange);
  });
  
  // Get all number inputs
  const numberInputs = document.querySelectorAll('.setting-input[type="number"]');
  numberInputs.forEach(input => {
    input.addEventListener('change', handleSettingChange);
  });
  
  // Get all text inputs
  const textInputs = document.querySelectorAll('.setting-input[type="text"]');
  textInputs.forEach(input => {
    input.addEventListener('change', handleSettingChange);
  });
  
  // Get all select elements
  const selectInputs = document.querySelectorAll('.setting-select');
  selectInputs.forEach(select => {
    select.addEventListener('change', handleSettingChange);
  });
  
  // Setup folder browse button
  const browseButton = document.getElementById('browse-folder');
  if (browseButton) {
    browseButton.addEventListener('click', () => {
      // Use Electron's dialog to open a folder picker
      ipcRenderer.send('open-folder-dialog');
    });
    
    // Listen for the selected folder
    ipcRenderer.on('selected-folder', (event, path) => {
      // Update the input element
      const folderInput = document.getElementById('destination-folder-input');
      if (folderInput && path) {
        folderInput.value = path;
        // Trigger the change event to save the setting
        folderInput.dispatchEvent(new Event('change'));
      }
    });
  }
  
  // Setup reset settings button
  const resetButton = document.getElementById('reset-settings');
  if (resetButton) {
    resetButton.addEventListener('click', resetAllSettings);
  }
  
  debugLog('Settings handlers initialized', LOG_LEVELS.DEBUG);
}

// Handle setting change
function handleSettingChange(event) {
  const target = event.target;
  const settingName = target.dataset.setting;
  
  if (!settingName) {
    debugLog('Setting name not found for element', LOG_LEVELS.ERROR);
    return;
  }
  
  let value;
  
  // Get the appropriate value based on input type
  if (target.type === 'checkbox') {
    value = target.checked;
  } else if (target.type === 'number') {
    value = parseInt(target.value, 10);
  } else {
    value = target.value;
  }
  
  debugLog(`Setting ${settingName} changed to ${value}`, LOG_LEVELS.INFO);
  
  // Highlight the changed setting
  const settingItem = target.closest('.setting-item');
  if (settingItem) {
    settingItem.classList.add('setting-changed');
    setTimeout(() => {
      settingItem.classList.remove('setting-changed');
    }, 2000);
  }
  
  // Save the setting to the backend
  updateSetting(settingName, value);
}

// Update a setting in the backend
function updateSetting(key, value) {
  debugLog(`Updating setting: ${key} = ${value}`, LOG_LEVELS.INFO);
  
  try {
    const grpcUtils = require('./grpc-utils');
    const client = grpcUtils.getClient();
    
    // Create the request
    const request = new grpcUtils.UpdateSettingRequest();
    request.setSettingName(key);
    
    // Set the appropriate value field based on type
    if (typeof value === 'boolean') {
      request.setBoolValue(value);
    } else if (typeof value === 'number') {
      request.setIntValue(value);
    } else {
      request.setStringValue(String(value));
    }
    
    // Call the service
    client.updateSetting(request, (error, response) => {
      if (error) {
        debugLog(`Error updating setting ${key}: ${error.message}`, LOG_LEVELS.ERROR);
        addLogEntry(`Failed to update setting ${key}: ${error.message}`, 'error');
        return;
      }
      
      debugLog(`Setting ${key} updated to ${value}`, LOG_LEVELS.INFO);
      addLogEntry(`Setting ${key} updated successfully`, 'success');
      
      // Update local config if available
      if (appConfig) {
        // Map backend setting key to frontend config property
        const configMapping = {
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
        
        // Update the local config if key mapping exists
        const configKey = configMapping[key];
        if (configKey && configKey in appConfig) {
          // Convert value to the appropriate type
          let typedValue = value;
          if (typeof appConfig[configKey] === 'boolean') {
            typedValue = value === 'true' || value === true;
          } else if (typeof appConfig[configKey] === 'number') {
            typedValue = Number(value);
          }
          
          appConfig[configKey] = typedValue;
          debugLog(`Updated local config: ${configKey}=${typedValue}`, LOG_LEVELS.DEBUG);
          
          // Update local state variables and UI based on the setting
          if (key === 'pair_mode_enabled') {
            autoPair = typedValue;
            if (pairAllToggle) {
              pairAllToggle.checked = autoPair;
            }
            debugLog(`Updated autoPair state to ${autoPair}`, LOG_LEVELS.DEBUG);
          } else if (key === 'sync_enabled') {
            autoSync = typedValue;
            if (syncQueueToggle) {
              syncQueueToggle.checked = autoSync;
            }
            debugLog(`Updated autoSync state to ${autoSync}`, LOG_LEVELS.DEBUG);
          } else if (key === 'log_level') {
            // Sync the frontend log level selector
            if (logLevelSelect) {
              logLevelSelect.value = value;
            }
            
            // Update the frontend log level
            const { setLogLevel } = require('./fileLogger');
            setLogLevel(value.toUpperCase());
          }
          
          // Update any other UI elements as needed
          updateConfigUI();
        }
      }
      
      // Special handling for certain settings
      if (key === 'pair_mode_enabled') {
        autoPair = value;
        if (pairAllToggle) {
          pairAllToggle.checked = value;
        }
      } else if (key === 'sync_enabled') {
        autoSync = value;
        if (syncQueueToggle) {
          syncQueueToggle.checked = value;
        }
      } else if (key === 'log_level') {
        // Sync the frontend log level selector
        if (logLevelSelect) {
          logLevelSelect.value = value;
        }
        
        // Update the frontend log level
        const { setLogLevel } = require('./fileLogger');
        setLogLevel(value.toUpperCase());
      }
    });
  } catch (error) {
    debugLog(`Exception updating setting: ${error.message}`, LOG_LEVELS.ERROR);
    addLogEntry(`Error updating setting ${key}: ${error.message}`, 'error');
  }
}

// Update the settings UI based on current config
function updateSettingsUI() {
  debugLog('Updating settings UI', LOG_LEVELS.DEBUG);
  
  if (!appConfig) {
    debugLog('No config available, skipping settings UI update', LOG_LEVELS.WARN);
    return;
  }
  
  // Map the config properties to setting names
  const configMapping = {
    'pairModeEnabled': 'pair_mode_enabled',
    'syncEnabled': 'sync_enabled',
    'scanIntervalSeconds': 'scan_interval_seconds',
    'connectTimeoutSeconds': 'connect_timeout_seconds',
    'daysThreshold': 'days_threshold',
    'destinationFolder': 'destination_folder',
    'inactivityTimeoutSeconds': 'inactivity_timeout_seconds',
    'inactivitySyncIntervalSeconds': 'inactivity_sync_interval_seconds',
    'setTimeEnabled': 'set_time_enabled',
    'logLevel': 'log_level',
    'debugMode': 'debug_mode'
  };
  
  // Update each setting in the UI
  for (const [configKey, settingName] of Object.entries(configMapping)) {
    const value = appConfig[configKey];
    if (value !== undefined) {
      updateSettingUI(settingName, value);
    }
  }
  
  debugLog('Settings UI updated', LOG_LEVELS.DEBUG);
}

// Update a single setting element in the UI
function updateSettingUI(settingName, value) {
  // Find the element with the data-setting attribute
  const element = document.querySelector(`[data-setting="${settingName}"]`);
  
  // Also check for special toggle elements
  if (settingName === 'pair_mode_enabled' && pairAllToggle) {
    pairAllToggle.checked = Boolean(value);
  } else if (settingName === 'sync_enabled' && syncQueueToggle) {
    syncQueueToggle.checked = Boolean(value);
  }
  
  // Update standard settings elements if they exist
  if (element) {
    if (element.type === 'checkbox') {
      element.checked = Boolean(value);
    } else if (element.type === 'number') {
      element.value = value;
    } else if (element.tagName === 'SELECT') {
      element.value = value;
    } else {
      element.value = value;
    }
    
    debugLog(`Updated UI for setting ${settingName} = ${value}`, LOG_LEVELS.DEBUG);
  } else {
    debugLog(`No standard UI element found for setting ${settingName}`, LOG_LEVELS.DEBUG);
  }
}

// Update signal strength indicators for a specific device in real-time
function updateSignalStrengthForDevice(macAddress, rssi) {
  debugLog(`Updating signal strength for ${macAddress}: ${rssi} dBm`, LOG_LEVELS.TRACE);
  
  // Find all device cards with this MAC address (there might be multiple in different panels)
  const deviceCards = document.querySelectorAll(`.device-card[data-mac="${macAddress}"]`);
  
  deviceCards.forEach(deviceCard => {
    // Update signal bars
    const signalBarsContainer = deviceCard.querySelector('.signal-bars');
    if (signalBarsContainer) {
      const bars = signalBarsContainer.querySelectorAll('.signal-bar');
      
      // Clear existing filled state
      bars.forEach(bar => bar.classList.remove('filled'));
      
      // Calculate and fill bars based on new RSSI
      const normalizedRssi = Math.min(Math.max(rssi, -100), -30);
      const signalStrength = Math.floor(((normalizedRssi + 100) / 70) * 4);
      
      for (let i = 0; i < signalStrength; i++) {
        if (bars[i]) {
          bars[i].classList.add('filled');
        }
      }
    }
    
    // Update RSSI value text
    const rssiValueElement = deviceCard.querySelector('.rssi-value');
    if (rssiValueElement) {
      rssiValueElement.textContent = `${rssi} dBm`;
    }
    

  });
}

// Update a device's status in the tracking
function updateDeviceStatus(macAddress, status) {
  const device = allDevices[macAddress];
  if (!device) {
    debugLog(`Cannot update status for unknown device: ${macAddress}`, LOG_LEVELS.ERROR);
    return;
  }
  // Update pairing flags
  if (status === 'paired') {
    device.isPairing = false;
    device.isPaired = true;
  }
  // Update visual status
  device.visualStatus = status;
  // Rerun pool logic to remove from Discovered and place into Available/Managed
  addOrUpdateDevice(device);
  // Highlight status change
  highlightStatusChange(macAddress);
}

// Add debug function to log device state and pool memberships
function debugDeviceState(device, action) {
  // Use the existing debug logging system instead of undefined debuggingEnabled
  debugLog(`[DEVICE STATE DEBUG] ${action}: ${device.name} (${device.macAddress})`, LOG_LEVELS.DEBUG);
  debugLog(`Device flags: isReachable=${device.isReachable}, isPaired=${device.isPaired}, isManaged=${device.isManaged}, isSynced=${device.isSynced}, isSyncing=${device.isSyncing}`, LOG_LEVELS.DEBUG);
  
  debugLog(`Pool membership should be: Discovered=${shouldShowInDiscoveredPool(device)}, Managed=${shouldShowInManagedPool(device)}, Sync=${shouldShowInSyncQueue(device)}`, LOG_LEVELS.DEBUG);
  
  const currentMembership = uiState.poolMembership[device.macAddress] || {};
  debugLog(`Current UI pool membership: Discovered=${currentMembership.discovered}, Managed=${currentMembership.managed}, Sync=${currentMembership.sync}`, LOG_LEVELS.DEBUG);
}

// Update all device lists in the UI (kept for full rebuilds when needed)
function updateDeviceLists() {
  // Only log on significant changes, not every update
  const totalDevices = Object.keys(allDevices).length;
  const discoveredCount = Object.values(allDevices).filter(device => shouldShowInDiscoveredPool(device)).length;
  const managedCount = Object.values(allDevices).filter(device => shouldShowInManagedPool(device)).length;
  const syncCount = Object.values(allDevices).filter(device => shouldShowInSyncQueue(device)).length;
  
  // Only log if this is a significant change (new devices, status changes, etc.)
  // We track the previous counts to avoid excessive logging
  if (!uiState.currentDeviceCounts || 
      uiState.currentDeviceCounts.total !== totalDevices ||
      uiState.currentDeviceCounts.discovered !== discoveredCount ||
      uiState.currentDeviceCounts.managed !== managedCount ||
      uiState.currentDeviceCounts.sync !== syncCount) {
    
    debugLog(`Device lists updated - Total: ${totalDevices}, Discovered: ${discoveredCount}, Managed: ${managedCount}, Sync: ${syncCount}`, LOG_LEVELS.TRACE);
    
    // Store current counts for next comparison
    uiState.currentDeviceCounts = {
      total: totalDevices,
      discovered: discoveredCount,
      managed: managedCount,
      sync: syncCount
    };
  }
  
  // Clear UI state tracking since we're doing a full rebuild
  uiState.deviceElements = {};
  uiState.poolMembership = {};
  
  // Update UI for each list
  updatePairQueueUI();
  updateCamerasPoolUI();
  updateSyncQueueUI();
  
  // Update counters
  updateCounters();
}

// Highlight a device card when its status changes
function highlightStatusChange(macAddress) {
  setTimeout(() => {
    // Find all instances of this device across pools
    const deviceCards = document.querySelectorAll(`.device-card[data-mac="${macAddress}"]`);
    deviceCards.forEach(deviceCard => {
      // Add highlight class to the device card for status-specific styling
      deviceCard.classList.add(`status-changed`);
      
      // Remove highlight class after animation completes
      setTimeout(() => {
        deviceCard.classList.remove(`status-changed`);
      }, 1500);
    });
  }, 100); // Short delay to ensure the DOM has updated
}