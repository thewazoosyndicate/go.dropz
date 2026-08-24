<script>
  let { checked = false, label = '', onchange, disabled = false } = $props();
</script>

<label class="toggle" class:disabled>
  <input type="checkbox" {checked} {disabled} onchange={(e) => onchange?.(e.target.checked)} />
  <span class="track" aria-hidden="true"></span>
  {#if label}<span class="label">{label}</span>{/if}
</label>

<style>
  .toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    user-select: none;
  }

  .toggle.disabled { opacity: 0.5; cursor: default; }

  .toggle input {
    position: absolute;
    opacity: 0;
    width: 1px;
    height: 1px;
  }

  .track {
    position: relative;
    width: 36px;
    height: 20px;
    background-color: #ccc;
    border-radius: 20px;
    transition: background-color 0.2s;
    flex-shrink: 0;
  }

  :global([data-theme="dark"]) .track { background-color: #555; }

  .track::before {
    content: '';
    position: absolute;
    width: 14px;
    height: 14px;
    left: 3px;
    bottom: 3px;
    background: white;
    border-radius: 50%;
    transition: transform 0.2s;
  }

  .toggle input:checked + .track { background-color: var(--secondary-color); }
  .toggle input:checked + .track::before { transform: translateX(16px); }
  .toggle input:focus-visible + .track {
    outline: 2px solid var(--primary-color);
    outline-offset: 2px;
  }

  .label {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }
</style>
