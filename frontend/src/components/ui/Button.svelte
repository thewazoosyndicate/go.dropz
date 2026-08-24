<script>
  // variant: primary (blue), success (green), danger (red, filled),
  // danger-outline (red text, for cancel), outline (default)
  let {
    variant = 'outline',
    size = 'md',
    icon = '',
    title = '',
    ariaLabel = '',
    disabled = false,
    grow = false,
    type = 'button',
    onclick,
    children,
  } = $props();
</script>

<button
  {type}
  class="btn btn-{variant} size-{size}"
  class:grow
  class:icon-only={!children}
  {title}
  aria-label={ariaLabel || title || undefined}
  {disabled}
  {onclick}
>
  {#if icon}<i class="fas {icon}" aria-hidden="true"></i>{/if}
  {#if children}{@render children()}{/if}
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    border: 1px solid transparent;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 500;
    line-height: 1.2;
    cursor: pointer;
    color: white;
    transition: background-color 0.15s, border-color 0.15s, color 0.15s;
    white-space: nowrap;
  }

  .size-md { padding: 8px 12px; min-height: 36px; }
  .size-sm { padding: 5px 10px; min-height: 30px; font-size: 0.75rem; }
  .icon-only.size-md { width: 36px; padding: 8px; }
  .icon-only.size-sm { width: 30px; padding: 5px; }
  .grow { flex: 1; }

  .btn:disabled { opacity: 0.5; cursor: default; }

  .btn-primary { background-color: var(--primary-color); }
  .btn-primary:hover:not(:disabled) { background-color: var(--primary-dark); }

  .btn-success { background-color: var(--secondary-color); }
  .btn-success:hover:not(:disabled) { background-color: var(--secondary-dark); }

  .btn-danger { background-color: var(--danger-color); }
  .btn-danger:hover:not(:disabled) { background-color: var(--danger-dark); }

  .btn-danger-outline {
    background: none;
    border-color: var(--danger-color);
    color: var(--danger-color);
  }
  .btn-danger-outline:hover:not(:disabled) {
    background-color: var(--danger-color);
    color: white;
  }

  .btn-outline {
    background: none;
    border-color: var(--border-color);
    color: var(--text-secondary);
  }
  .btn-outline:hover:not(:disabled) {
    border-color: var(--text-secondary);
    color: var(--text-primary);
  }
</style>
