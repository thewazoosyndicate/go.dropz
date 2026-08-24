<script>
  // Square outline control for toolbars and headers; 32px so it lines up
  // with inputs and chips. Card action rows use Button instead.
  let { icon, title, active = false, disabled = false, spin = false, onclick } = $props();
</script>

<button class="icon-btn" class:active {title} aria-label={title} {disabled} {onclick}>
  <i class="fas {icon}" class:fa-spin={spin} aria-hidden="true"></i>
</button>

<style>
  .icon-btn {
    width: 32px;
    height: 32px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.85rem;
    transition: background-color 0.15s, border-color 0.15s, color 0.15s;
  }

  .icon-btn:hover:not(:disabled) {
    background-color: var(--hover-bg);
    color: var(--text-primary);
  }

  .icon-btn.active {
    border-color: var(--primary-color);
    color: var(--primary-color);
  }

  .icon-btn:disabled { opacity: 0.5; cursor: default; }
</style>
