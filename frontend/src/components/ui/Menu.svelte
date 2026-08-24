<script>
  // Kebab menu for secondary and destructive card actions. Items:
  // { label, icon, onclick, danger, confirm, disabled, checked, items }.
  // A confirm item asks once inline ("Confirm?") instead of a native
  // dialog; an item with `items` opens its children in place.
  let { items = [], title = 'More actions', icon = 'fa-ellipsis-vertical' } = $props();

  let open = $state(false);
  let confirming = $state(null);
  let expanded = $state(null); // label of the open sub-list
  let trigger = $state(null);
  // Fixed coordinates: the card grid scrolls and clips anything absolute
  let pos = $state({ top: 0, right: 0, up: false });

  function place() {
    const rect = trigger.getBoundingClientRect();
    const estimated = items.length * 38 + 10;
    const up = rect.bottom + estimated > window.innerHeight;
    pos = {
      right: window.innerWidth - rect.right,
      top: up ? rect.top - 6 : rect.bottom + 6,
      up,
    };
  }

  function toggle(e) {
    e.stopPropagation();
    if (!open) place();
    open = !open;
    confirming = null;
    expanded = null;
  }

  function close() {
    open = false;
    confirming = null;
    expanded = null;
  }

  function pick(item, e) {
    e.stopPropagation();
    if (item.disabled) return;
    if (item.items) {
      expanded = expanded === item.label ? null : item.label;
      return;
    }
    if (item.confirm && confirming !== item.label) {
      confirming = item.label;
      return;
    }
    close();
    item.onclick?.();
  }

  function onWindowClick(e) {
    if (open && trigger && !trigger.contains(e.target) && !e.target.closest?.('.menu-popover')) close();
  }

  function onKeydown(e) {
    if (open && e.key === 'Escape') close();
  }
</script>

<svelte:window onclick={onWindowClick} onkeydown={onKeydown} onresize={close} />

<div class="menu">
  <button class="trigger" class:open bind:this={trigger} onclick={toggle} {title} aria-label={title}
          aria-haspopup="menu" aria-expanded={open}>
    <i class="fas {icon}" aria-hidden="true"></i>
  </button>
  {#if open}
    <div class="popover menu-popover" role="menu"
         style:right="{pos.right}px"
         style:top={pos.up ? 'auto' : `${pos.top}px`}
         style:bottom={pos.up ? `${window.innerHeight - pos.top}px` : 'auto'}>
      {#each items as item (item.label)}
        <button class="item" class:danger={item.danger} role="menuitem"
                aria-haspopup={item.items ? 'menu' : undefined} aria-expanded={item.items ? expanded === item.label : undefined}
                disabled={item.disabled} onclick={(e) => pick(item, e)}>
          {#if item.icon}<i class="fas {item.icon}" aria-hidden="true"></i>{/if}
          <span class="label">{confirming === item.label ? `Confirm ${item.label.toLowerCase()}?` : item.label}</span>
          {#if item.items}<i class="fas fa-chevron-down caret" class:open={expanded === item.label} aria-hidden="true"></i>{/if}
          {#if item.checked}<i class="fas fa-check tick" aria-hidden="true"></i>{/if}
        </button>
        {#if item.items && expanded === item.label}
          <div class="sub" role="group">
            {#each item.items as child (child.label)}
              <button class="item child" class:danger={child.danger} role="menuitem"
                      disabled={child.disabled} onclick={(e) => pick(child, e)}>
                {#if child.icon}<i class="fas {child.icon}" aria-hidden="true"></i>{/if}
                <span class="label">{child.label}</span>
                {#if child.checked}<i class="fas fa-check tick" aria-hidden="true"></i>{/if}
              </button>
            {/each}
          </div>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<style>
  .menu { position: relative; }

  .trigger {
    width: 36px;
    min-height: 36px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.85rem;
  }

  .trigger:hover, .trigger.open {
    border-color: var(--text-secondary);
    color: var(--text-primary);
  }

  .popover {
    position: fixed;
    min-width: 200px;
    max-height: 60vh;
    overflow-y: auto;
    background: var(--panel-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    box-shadow: var(--shadow-md);
    padding: 4px;
    z-index: 50;
    display: flex;
    flex-direction: column;
  }

  .item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    min-height: 36px;
    background: none;
    border: none;
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 0.8rem;
    text-align: left;
    cursor: pointer;
    width: 100%;
  }

  .item > i:first-child { width: 14px; text-align: center; color: var(--text-secondary); }
  .item .label { flex: 1; }
  .item:hover:not(:disabled) { background-color: var(--hover-bg); }
  .item:disabled { opacity: 0.5; cursor: default; }
  .item.danger { color: var(--danger-color); }
  .item.danger i { color: var(--danger-color); }
  .caret { font-size: 0.7rem; color: var(--text-muted); transition: transform 0.15s; }
  .caret.open { transform: rotate(180deg); }
  .tick { color: var(--state-ok); font-size: 0.75rem; }

  .sub {
    display: flex;
    flex-direction: column;
    border-left: 2px solid var(--border-color);
    margin: 0 0 4px 16px;
  }
  .child { padding-left: 10px; }
</style>
