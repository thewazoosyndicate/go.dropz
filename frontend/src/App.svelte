<script>
  import StatusBar from './components/StatusBar.svelte';
  import MainLayout from './components/MainLayout.svelte';
  import LogDrawer from './components/LogDrawer.svelte';
  import SettingsModal from './components/SettingsModal.svelte';
  import ToastContainer from './components/ToastContainer.svelte';
  import { loadThemePreference } from './lib/stores/ui.svelte.js';
  import { startDeviceStreaming, cancelAllStreams } from './lib/grpc/streams.js';
  import { loadConfig } from './lib/grpc/actions.js';
  import { setServiceRunning } from './lib/stores/connection.svelte.js';
  import { addLog } from './lib/stores/ui.svelte.js';
  import { cleanupStaleDevices } from './lib/stores/devices.svelte.js';

  const { ipcRenderer } = window.require('electron');
  const path = window.require('path');
  const { shouldFilterLogMessage } = window.require(path.join(window.__appRoot, 'src', 'log-filters'));

  let grpcReadyResolve;
  let grpcReady = new Promise(resolve => { grpcReadyResolve = resolve; });

  loadThemePreference();

  // IPC listeners
  ipcRenderer.on('go-binary-status', (event, data) => {
    setServiceRunning(data.running);
    if (data.running) {
      addLog('Service started', 'info');
      grpcReadyResolve();
    } else {
      cancelAllStreams();
      addLog(data.exitCode === 0 ? 'Service stopped' : `Service stopped (exit ${data.exitCode})`, data.exitCode === 0 ? 'info' : 'error');
    }
  });

  ipcRenderer.on('go-binary-error', (event, msg) => {
    addLog(`Service error: ${msg}`, 'error');
  });

  ipcRenderer.on('go-binary-log', (event, log) => {
    if (shouldFilterLogMessage(log)) return;
    const style = log.includes('[ERROR]') || log.includes('[FATAL]') ? 'error' :
                  log.includes('[WARN]') ? 'warn' : 'info';
    const msg = log.replace(/^\d{4}\/\d{2}\/\d{2}\s+\d{2}:\d{2}:\d{2}\s+/, '');
    addLog(`[Go] ${msg}`, style, 'go');
  });

  // Start streaming when gRPC is ready
  grpcReady.then(() => {
    startDeviceStreaming();
    loadConfig();
  });

  // Cleanup stale devices every 30s
  const cleanupInterval = setInterval(() => cleanupStaleDevices(), 30000);

  // Cleanup on window close
  window.addEventListener('beforeunload', () => {
    clearInterval(cleanupInterval);
    cancelAllStreams();
  });
</script>

<div class="app-container">
  <StatusBar />
  <MainLayout />
  <LogDrawer />
  <SettingsModal />
  <ToastContainer />
</div>

<style>
  .app-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }
</style>
