<script>
  import { getToasts, removeToast } from '../lib/stores/ui.svelte.js';

  let toasts = $derived(getToasts());
</script>

<!-- Polite: outcomes are announced without interrupting what a screen
     reader is in the middle of -->
<div class="toast-container" aria-live="polite">
  {#each toasts as toast (toast.id)}
    <div class="toast {toast.type}" role="status">
      <span>{toast.message}</span>
      <button class="toast-close" onclick={() => removeToast(toast.id)} aria-label="Dismiss">&times;</button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: 20px;
    right: 20px;
    z-index: 1000;
    display: flex;
    flex-direction: column-reverse;
    gap: 8px;
    max-width: 360px;
  }

  .toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-radius: 6px;
    color: white;
    font-size: 0.85rem;
    box-shadow: var(--shadow-md);
    animation: toast-slide-in 0.3s ease;
  }

  .toast.success { background-color: var(--secondary-dark); }
  .toast.error   { background-color: var(--danger-dark); }
  .toast.warning { background-color: var(--warning-dark); }
  .toast.info    { background-color: var(--primary-dark); }

  .toast-close {
    background: none;
    border: none;
    color: white;
    font-size: 1.1rem;
    cursor: pointer;
    opacity: 0.7;
    margin-left: 12px;
    padding: 0 4px;
    min-width: 24px;
    min-height: 24px;
  }

  .toast-close:hover { opacity: 1; }
</style>
