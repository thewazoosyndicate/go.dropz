<script>
  // tone: ok | busy | error | info | idle. Filled badges carry white
  // text; idle is the quiet grey pill for states that need no attention.
  let { tone = 'idle', pulse = false, children } = $props();
</script>

<span class="badge tone-{tone}" class:pulse>{@render children()}</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.02em;
    text-transform: uppercase;
    line-height: 16px;
    padding: 1px 7px;
    border-radius: 10px;
    color: white;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .tone-ok { background-color: var(--state-ok); }
  .tone-busy { background-color: var(--state-busy); }
  .tone-error { background-color: var(--state-error); }
  .tone-info { background-color: var(--state-info); }
  .tone-idle { background-color: var(--state-idle-tint); color: var(--text-secondary); }

  .pulse { animation: pulse 1.2s infinite; }
</style>
