// Connection state store
let serviceRunning = $state(false);
let streamStatus = $state({});
let connectionState = $state({
  isConnecting: false,
  retryCount: 0,
  maxRetries: 10,
  baseDelay: 1000
});

export function getServiceRunning() { return serviceRunning; }
export function getStreamStatus() { return streamStatus; }
export function getConnectionState() { return connectionState; }

export function setServiceRunning(value) { serviceRunning = value; }

export function setStreamStatus(key, status) {
  streamStatus = { ...streamStatus, [key]: status };
}

export function resetStreamStatuses() {
  streamStatus = {};
}

export function isAnyReconnecting() {
  return Object.values(streamStatus).some(s => s === 'reconnecting');
}

export function allStreamsConnected() {
  const statuses = Object.values(streamStatus);
  return statuses.length > 0 && statuses.every(s => s === 'connected');
}
