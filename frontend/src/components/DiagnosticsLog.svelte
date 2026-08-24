<script>
  // The raw log stream, folded into the Activity tab. It answers "why"
  // once the session rows have answered "what".
  import { getLogs, clearLogs, getShowBackendLogs, setShowBackendLogs, getLogLevel } from '../lib/stores/ui.svelte.js';
  import { updateSetting } from '../lib/grpc/actions.js';
  import Toggle from './ui/Toggle.svelte';
  import Button from './ui/Button.svelte';

  let showBackend = $derived(getShowBackendLogs());
  let level = $derived(getLogLevel());
  let logs = $derived(getLogs());

  let filteredLogs = $derived(logs.filter(log => showBackend || log.source !== 'go'));

  let logContainer = $state(null);

  $effect(() => {
    // Auto-scroll when new logs arrive
    if (logContainer && filteredLogs.length) {
      requestAnimationFrame(() => {
        if (logContainer) logContainer.scrollTop = logContainer.scrollHeight;
      });
    }
  });
</script>

<div class="diagnostics">
  <div class="controls">
    <select class="level-select" value={level} aria-label="Log level"
            onchange={(e) => updateSetting('log_level', e.target.value)}>
      <option value="trace">Trace</option>
      <option value="debug">Debug</option>
      <option value="info">Info</option>
      <option value="warn">Warn</option>
      <option value="error">Error</option>
    </select>
    <Toggle checked={showBackend} label="Backend" onchange={setShowBackendLogs} />
    <span class="spacer"></span>
    <Button size="sm" icon="fa-trash-alt" onclick={clearLogs}>Clear</Button>
  </div>
  <div class="log-content" bind:this={logContainer}>
    {#each filteredLogs as log}
      <div class="log-entry {log.type}">
        <span class="log-time">[{log.time.toLocaleTimeString()}]</span>
        {log.message}
      </div>
    {/each}
  </div>
</div>

<style>
  .diagnostics {
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border-color);
    border-radius: 8px;
    overflow: hidden;
    height: 320px;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 12px;
    border-bottom: 1px solid var(--border-color);
  }

  .spacer { flex: 1; }

  .level-select {
    padding: 4px 6px;
    height: 30px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    background-color: var(--panel-bg);
    color: var(--text-primary);
    font-size: 0.75rem;
    cursor: pointer;
  }

  .log-content {
    flex: 1;
    overflow-y: auto;
    padding: 8px 12px;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 0.75rem;
    background: var(--light-bg);
  }

  .log-entry {
    padding: 2px 0;
    border-bottom: 1px dashed var(--border-color);
    white-space: pre-wrap;
    word-break: break-all;
  }

  .log-entry.error { color: var(--danger-color); }
  .log-entry.debug { color: var(--text-secondary); opacity: 0.8; }
  .log-entry.warn  { color: var(--warning-color); }

  .log-time { color: var(--text-muted); margin-right: 4px; }
</style>
