<script>
  import StatusBar from './components/StatusBar.svelte';
  import MainLayout from './components/MainLayout.svelte';
  import SettingsModal from './components/SettingsModal.svelte';
  import CameraSettingsModal from './components/CameraSettingsModal.svelte';
  import PreviewPlayer from './components/PreviewPlayer.svelte';
  import ToastContainer from './components/ToastContainer.svelte';
  import { loadThemePreference } from './lib/stores/ui.svelte.js';
  import { startDeviceStreaming, cancelAllStreams } from './lib/grpc/streams.js';
  import { loadConfig } from './lib/grpc/actions.js';
  import { setServiceRunning } from './lib/stores/connection.svelte.js';
  import { addLog } from './lib/stores/ui.svelte.js';
  import { cleanupStaleDevices } from './lib/stores/devices.svelte.js';

  const { ipcRenderer } = window.require('electron');

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

  const levelStyles = { ERROR: 'error', WARN: 'warn', DEBUG: 'debug', TRACE: 'debug' };

  ipcRenderer.on('go-binary-log', (event, entry) => {
    const style = levelStyles[entry.level] || 'info';
    const attrs = Object.entries(entry.attrs || {})
      .map(([k, v]) => `${k}=${v}`)
      .join(' ');
    const prefix = entry.component ? `[${entry.component}] ` : '';
    addLog(`${prefix}${entry.msg}${attrs ? ' ' + attrs : ''}`, style, 'go');
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
  <SettingsModal />
  <CameraSettingsModal />
  <PreviewPlayer />
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
