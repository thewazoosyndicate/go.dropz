<script>
  import { getLogs, getLogsExpanded, setLogsExpanded, clearLogs,
           getShowBackendLogs, setShowBackendLogs, getLogLevel } from '../lib/stores/ui.svelte.js';
  import { updateSetting } from '../lib/grpc/actions.js';

  let expanded = $derived(getLogsExpanded());
  let showBackend = $derived(getShowBackendLogs());
  let level = $derived(getLogLevel());
  let logs = $derived(getLogs());

  let filteredLogs = $derived.by(() => {
    return logs.filter(log => {
      if (!showBackend && log.source === 'go') return false;
      return true;
    });
  });

  let logContainer = $state(null);

  $effect(() => {
    // Auto-scroll when new logs arrive
    if (logContainer && expanded && filteredLogs.length) {
      requestAnimationFrame(() => {
        if (logContainer) logContainer.scrollTop = logContainer.scrollHeight;
      });
    }
  });
</script>

<div class="log-drawer" class:expanded>
  <button class="drawer-tab" onclick={() => setLogsExpanded(!expanded)}>
    <i class="fas fa-terminal"></i>
    Logs {expanded ? '▾' : '▸'}
  </button>

  {#if expanded}
    <div class="drawer-header">
      <div class="drawer-controls">
        <select class="level-select" value={level} onchange={(e) => updateSetting('log_level', e.target.value)}>
          <option value="trace">Trace</option>
          <option value="debug">Debug</option>
          <option value="info">Info</option>
          <option value="warn">Warn</option>
          <option value="error">Error</option>
        </select>
        <label class="toggle-small">
          <input type="checkbox" checked={showBackend} onchange={(e) => setShowBackendLogs(e.target.checked)} />
          <span>Backend</span>
        </label>
        <button class="clear-btn" onclick={clearLogs}>
          <i class="fas fa-trash-alt"></i> Clear
        </button>
      </div>
    </div>
    <div class="log-content" bind:this={logContainer}>
      {#each filteredLogs as log}
        <div class="log-entry {log.type}">
          <span class="log-time">[{log.time.toLocaleTimeString()}]</span>
          {log.message}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .log-drawer {
    background-color: var(--panel-bg);
    border-top: 1px solid var(--border-color);
    flex-shrink: 0;
    transition: background-color 0.3s ease;
  }

  .log-drawer.expanded {
    height: 35vh;
    display: flex;
    flex-direction: column;
  }

  .drawer-tab {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 16px;
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 0.8rem;
    cursor: pointer;
    width: 100%;
    text-align: left;
  }

  .drawer-tab:hover {
    color: var(--text-primary);
  }

  .drawer-header {
    padding: 6px 14px;
    border-bottom: 1px solid var(--border-color);
  }

  .drawer-controls {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .level-select {
    padding: 3px 6px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    background-color: var(--panel-bg);
    color: var(--text-primary);
    font-size: 0.75rem;
    cursor: pointer;
  }

  .toggle-small {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.75rem;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .toggle-small input {
    cursor: pointer;
  }

  .clear-btn {
    padding: 3px 8px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    background: none;
    color: var(--text-secondary);
    font-size: 0.7rem;
    cursor: pointer;
    margin-left: auto;
  }

  .clear-btn:hover {
    color: var(--text-primary);
    border-color: var(--text-secondary);
  }

  .log-content {
    flex: 1;
    overflow-y: auto;
    padding: 8px 14px;
    font-family: monospace;
    font-size: 0.8rem;
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

  .log-time {
    color: var(--text-muted);
    margin-right: 4px;
  }
</style>
