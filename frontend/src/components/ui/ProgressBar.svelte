<script>
  // value: 0..100. shimmer marks the one live transfer on screen; keep it
  // off for static bars (history, per-file rows).
  let { value = 0, height = 6, shimmer = false, label = '' } = $props();
  let width = $derived(Math.max(0, Math.min(100, value)));
</script>

<div class="bar" style:height="{height}px" role="progressbar"
     aria-valuenow={Math.round(width)} aria-valuemin="0" aria-valuemax="100" aria-label={label || undefined}>
  <div class="fill" class:shimmer style:width="{width}%"></div>
</div>

<style>
  .bar {
    background-color: var(--border-color);
    border-radius: 3px;
    overflow: hidden;
    width: 100%;
  }

  .fill {
    height: 100%;
    background: linear-gradient(90deg, var(--state-busy), var(--state-ok));
    border-radius: 3px;
    transition: width 0.3s;
    position: relative;
  }

  .fill.shimmer::after {
    content: '';
    position: absolute;
    top: 0;
    left: -50%;
    width: 50%;
    height: 100%;
    background: linear-gradient(90deg, transparent, rgba(255,255,255,0.4), transparent);
    animation: sync-progress-shimmer 1.5s infinite;
  }
</style>
