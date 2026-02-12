<script>
  import { getToasts, removeToast } from '../lib/stores/ui.svelte.js';

  let toasts = $derived(getToasts());
</script>

<div class="toast-container">
  {#each toasts as toast (toast.id)}
    <div class="toast {toast.type}">
      <span>{toast.message}</span>
      <button class="toast-close" onclick={() => removeToast(toast.id)}>&times;</button>
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
  }

  .toast-close:hover { opacity: 1; }
</style>
