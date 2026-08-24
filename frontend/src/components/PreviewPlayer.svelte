<script>
  import { getPlayerTarget, closePlayer } from '../lib/stores/ui.svelte.js';

  let target = $derived(getPlayerTarget());

  function onKeydown(e) {
    if (e.key === 'Escape' && target) closePlayer();
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if target}
  <div class="player-overlay" onclick={closePlayer} role="presentation">
    <div class="player" onclick={(e) => e.stopPropagation()} role="presentation">
      <div class="player-header">
        <span class="player-title">{target.title}</span>
        <button class="player-close" onclick={closePlayer} aria-label="Close player">&times;</button>
      </div>
      <!-- svelte-ignore a11y_media_has_caption -->
      <video src={'file://' + target.path} controls autoplay></video>
    </div>
  </div>
{/if}

<style>
  .player-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.75);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .player {
    background: var(--panel-bg);
    border-radius: 10px;
    overflow: hidden;
    width: min(880px, 92vw);
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow-md);
  }

  .player-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
  }

  .player-title {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .player-close {
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 1.3rem;
    cursor: pointer;
    line-height: 1;
  }
  .player-close:hover { color: var(--text-primary); }

  video {
    width: 100%;
    max-height: 70vh;
    background: black;
    display: block;
  }
</style>
