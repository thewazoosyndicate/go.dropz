<script>
  // In-app player for the 480p proxy, with a lossless trim mode: two
  // handles on the scrubber, the in-handle snapping to the full-res
  // file's real keyframes so the saved cut is what the timeline showed.
  import { getPlayerTarget, closePlayer, openPlayer, addToast, addLog } from '../lib/stores/ui.svelte.js';
  import { getVideoKeyframes, trimVideo, loadVideos } from '../lib/grpc/actions.js';
  import { formatBytes } from '../lib/format.js';
  import Button from './ui/Button.svelte';
  import IconButton from './ui/IconButton.svelte';

  const { ipcRenderer } = window.require('electron');

  let target = $derived(getPlayerTarget());

  let video = $state(null);
  let player = $state(null);
  let timeline = $state(null);
  let duration = $state(0); // seconds, proxy timeline
  let currentTime = $state(0);
  let playing = $state(false);
  let muted = $state(false);
  let volume = $state(1);
  let hoverTime = $state(null);
  // Controls overlay the picture, windowed or fullscreen, and fade while
  // the clip plays and the mouse rests; paused or trimming keeps them.
  // Fullscreen takes the whole player so the trim handles stay usable.
  let fullscreen = $state(false);
  let idle = $state(false);
  let idleTimer = null;

  function toggleFullscreen() {
    if (!player) return;
    if (document.fullscreenElement) document.exitFullscreen?.();
    else player.requestFullscreen?.();
  }

  function onFullscreenChange() {
    fullscreen = !!document.fullscreenElement;
    wake();
  }

  function wake() {
    idle = false;
    clearTimeout(idleTimer);
    idleTimer = setTimeout(() => {
      if (playing && !trimming && !dragging && !saving) idle = true;
    }, 2500);
  }

  $effect(() => {
    if (video) video.volume = volume;
  });

  // Trim state, in seconds on the same timeline
  let trimming = $state(false);
  let inTime = $state(0);
  let outTime = $state(0);
  let keyframes = $state([]); // seconds, from the full-res file
  let keyframesLoaded = $state(false);
  let dragging = $state(null); // 'in' | 'out' | 'play' | null
  let loopSelection = $state(false);
  let saving = $state(false);
  let saved = $state(null);

  const MIN_LEN = 0.5;
  let canTrim = $derived(!!target?.sourcePath);
  let selectionLength = $derived(Math.max(0, outTime - inTime));
  let estimatedBytes = $derived(target?.sizeBytes && duration > 0 ? target.sizeBytes * selectionLength / duration : 0);
  let snappedIn = $derived(snapDown(inTime));
  // Ticks only when they are far enough apart to read as marks
  let tickSpacing = $derived(keyframes.length > 1 && timeline ? timeline.clientWidth / (duration / (keyframes[1] - keyframes[0])) : 0);
  let showTicks = $derived(tickSpacing >= 5);

  // Reset per clip, and take focus so the shortcuts work at once
  $effect(() => {
    if (!target) return;
    setTimeout(() => player?.focus(), 0);
    trimming = false;
    saved = null;
    saving = false;
    keyframes = [];
    keyframesLoaded = false;
    inTime = 0;
    outTime = 0;
    loopSelection = false;
  });

  function snapDown(t) {
    if (keyframes.length === 0) return t;
    let best = keyframes[0];
    for (const k of keyframes) {
      if (k > t + 1e-6) break;
      best = k;
    }
    return best;
  }

  function snapNearest(t) {
    if (keyframes.length === 0) return t;
    let best = keyframes[0];
    for (const k of keyframes) {
      if (Math.abs(k - t) < Math.abs(best - t)) best = k;
    }
    return best;
  }

  function neighborKeyframe(t, dir) {
    if (keyframes.length === 0) return clamp(t + dir);
    if (dir > 0) {
      const next = keyframes.find(k => k > t + 1e-3);
      return next ?? t;
    }
    let prev = t;
    for (const k of keyframes) {
      if (k < t - 1e-3) prev = k;
      else break;
    }
    return prev;
  }

  function clamp(t) { return Math.max(0, Math.min(duration, t)); }

  async function toggleTrim() {
    if (!canTrim) return;
    if (trimming) { trimming = false; loopSelection = false; wake(); return; }
    trimming = true;
    saved = null;
    wake();
    if (outTime === 0) {
      inTime = 0;
      outTime = duration;
    }
    if (!keyframesLoaded) {
      try {
        const r = await getVideoKeyframes(target.sourcePath);
        keyframes = r.keyframesMs.map(ms => ms / 1000);
        keyframesLoaded = true;
        if (r.durationMs > 0 && outTime >= duration) outTime = Math.min(duration, r.durationMs / 1000);
      } catch (e) {
        addToast(e.message || 'Could not read keyframes', 'error');
      }
    }
  }

  function setIn(t) {
    const v = snapNearest(clamp(t));
    inTime = Math.min(v, outTime - MIN_LEN);
    if (inTime < 0) inTime = 0;
  }

  // Both handles live on the keyframe grid. The end could technically
  // fall anywhere, but two handles with different rules on one bar read
  // as a bug; the clip's very end stays reachable past the last keyframe.
  function setOut(t) {
    const c = clamp(t);
    const v = c >= duration - 0.05 ? duration : snapNearest(c);
    outTime = Math.max(v, inTime + MIN_LEN);
  }

  // Transport
  function togglePlay() {
    if (!video) return;
    if (video.paused) video.play(); else video.pause();
  }

  function seek(t, scrub = false) {
    if (!video) return;
    const v = clamp(t);
    video.currentTime = v;
    currentTime = v;
    if (scrub && !video.paused) video.pause();
  }

  function playSelection() {
    loopSelection = true;
    seek(inTime);
    video?.play();
  }

  function onTimeUpdate() {
    if (!video || dragging) return;
    currentTime = video.currentTime;
    if (loopSelection && trimming && currentTime >= outTime - 0.05) {
      seek(inTime);
      if (video.paused) video.play();
    }
  }

  function onLoaded() {
    duration = video?.duration || 0;
    if (trimming && outTime === 0) outTime = duration;
  }

  // Pointer interaction on the timeline: handles drag, the track seeks.
  function timeAt(clientX) {
    const rect = timeline.getBoundingClientRect();
    const x = Math.max(0, Math.min(rect.width, clientX - rect.left));
    return duration * x / rect.width;
  }

  function onPointerDown(e, what) {
    e.preventDefault();
    e.stopPropagation();
    dragging = what;
    timeline.setPointerCapture(e.pointerId);
    onPointerMove(e);
  }

  let scrubPending = false;
  function onPointerMove(e) {
    if (dragging) {
      const t = timeAt(e.clientX);
      if (dragging === 'in') setIn(t);
      else if (dragging === 'out') setOut(t);
      // Show the frame under the handle while it moves, throttled to
      // one seek per animation frame so the decoder keeps up
      const preview = dragging === 'in' ? inTime : dragging === 'out' ? outTime : t;
      if (!scrubPending) {
        scrubPending = true;
        requestAnimationFrame(() => { scrubPending = false; seek(preview, true); });
      }
    } else {
      hoverTime = timeAt(e.clientX);
    }
  }

  function onPointerUp(e) {
    if (!dragging) return;
    try { timeline.releasePointerCapture(e.pointerId); } catch (_) {}
    dragging = null;
  }

  function onHandleKey(e, which) {
    const big = e.shiftKey ? 5 : 1;
    let delta = 0;
    if (e.key === 'ArrowLeft') delta = -1;
    else if (e.key === 'ArrowRight') delta = 1;
    else if (e.key === 'Home') { which === 'in' ? setIn(0) : setOut(inTime + MIN_LEN); e.preventDefault(); return; }
    else if (e.key === 'End') { which === 'in' ? setIn(outTime - MIN_LEN) : setOut(duration); e.preventDefault(); return; }
    else return;
    e.preventDefault();
    e.stopPropagation();
    // Handles walk keyframe to keyframe; the end also reaches the clip's tail
    let t = which === 'in' ? inTime : outTime;
    for (let i = 0; i < big; i++) {
      const n = neighborKeyframe(t, delta);
      t = (which === 'out' && delta > 0 && n === t) ? duration : n;
    }
    if (which === 'in') { setIn(t); seek(inTime, true); }
    else { setOut(t); seek(outTime, true); }
  }

  function onKeydown(e) {
    if (!target) return;
    if (['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target?.tagName)) return;
    switch (e.key) {
      case 'Escape':
        // The browser already left fullscreen on this Esc; do not also close
        if (fullscreen) break;
        if (trimming && !saving) { trimming = false; loopSelection = false; }
        else closePlayer();
        break;
      case ' ':
        e.preventDefault();
        togglePlay();
        break;
      case 'ArrowLeft':
        if (e.target?.getAttribute?.('role') === 'slider') return;
        e.preventDefault();
        seek(currentTime - (e.shiftKey ? 5 : 1));
        break;
      case 'ArrowRight':
        if (e.target?.getAttribute?.('role') === 'slider') return;
        e.preventDefault();
        seek(currentTime + (e.shiftKey ? 5 : 1));
        break;
      case 'Home': seek(0); break;
      case 'End': seek(duration); break;
      case 'i': case 'I':
        if (!trimming) toggleTrim();
        setIn(currentTime);
        break;
      case 'o': case 'O':
        if (!trimming) toggleTrim();
        setOut(currentTime);
        break;
      case '[': if (trimming) seek(inTime); break;
      case ']': if (trimming) seek(outTime); break;
      case 't': case 'T': toggleTrim(); break;
      case 'm': case 'M': muted = !muted; break;
      case 'f': case 'F': toggleFullscreen(); break;
      case 'Enter': if (trimming && !saving) save(); break;
    }
  }

  async function save() {
    if (!canTrim || saving) return;
    if (selectionLength < MIN_LEN) {
      addToast('Selection is shorter than half a second', 'error');
      return;
    }
    if (snappedIn <= 0 && outTime >= duration - 0.05) {
      addToast('The selection covers the whole clip', 'info');
      return;
    }
    saving = true;
    video?.pause();
    try {
      const r = await trimVideo(target.sourcePath, snappedIn * 1000, outTime * 1000);
      saved = r;
      addToast(`Saved ${r.name} (${formatBytes(r.sizeBytes)})`, 'success');
      addLog(`Trimmed ${target.title} to ${r.name}`, 'info');
      loadVideos();
    } catch (e) {
      addToast(e.message || 'Trim failed', 'error');
      addLog(`Trim failed: ${e.message}`, 'error');
    } finally {
      saving = false;
    }
  }

  function playSaved() {
    if (!saved?.previewPath) return;
    const s = saved;
    closePlayer();
    // Re-open on the next tick so the reset effect runs for the new clip
    setTimeout(() => openPlayer(s.previewPath, s.name, s.outputPath, s.sizeBytes), 0);
  }

  function showSaved() {
    if (saved?.outputPath) ipcRenderer.send('desktop-show', saved.outputPath);
  }

  function fmt(t) {
    if (!isFinite(t) || t < 0) t = 0;
    const m = Math.floor(t / 60);
    const s = t - m * 60;
    return `${String(m).padStart(2, '0')}:${s.toFixed(1).padStart(4, '0')}`;
  }

  function pct(t) { return duration > 0 ? (100 * t / duration) : 0; }
</script>

<svelte:window onkeydown={onKeydown} />
<svelte:document onfullscreenchange={onFullscreenChange} />

{#if target}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="player-overlay" onclick={(e) => { if (e.target === e.currentTarget) closePlayer(); }}>
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div class="player" class:fullscreen class:idle bind:this={player} tabindex="-1"
         onmousemove={wake} onpointerdown={wake}
         role="dialog" aria-modal="true" aria-label={target.title}>
      <div class="player-header">
        <span class="player-title">{target.title}</span>
        <div class="header-actions">
          {#if canTrim}
            <Button size="sm" icon="fa-scissors" variant={trimming ? 'primary' : 'outline'} onclick={toggleTrim}
                    title="Trim (T)">{trimming ? 'Trimming' : 'Trim'}</Button>
          {/if}
          <IconButton icon={fullscreen ? 'fa-compress' : 'fa-expand'} title={fullscreen ? 'Exit fullscreen (F)' : 'Fullscreen (F)'} onclick={toggleFullscreen} />
          <IconButton icon="fa-times" title="Close (Esc)" onclick={closePlayer} />
        </div>
      </div>

      <!-- svelte-ignore a11y_media_has_caption -->
      <video bind:this={video} src={'file://' + target.path} autoplay {muted}
             onloadedmetadata={onLoaded} ontimeupdate={onTimeUpdate}
             onplay={() => { playing = true; wake(); }} onpause={() => { playing = false; wake(); }}
             onclick={togglePlay} ondblclick={toggleFullscreen}></video>

      <div class="controls">
      <div class="transport" class:dragging={!!dragging}>
        <IconButton icon={playing ? 'fa-pause' : 'fa-play'} title={playing ? 'Pause (Space)' : 'Play (Space)'} onclick={togglePlay} />
        <span class="time" aria-live="off">{fmt(currentTime)} <span class="muted">/ {fmt(duration)}</span></span>

        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="timeline" class:trimming bind:this={timeline}
             onpointermove={onPointerMove} onpointerup={onPointerUp} onpointercancel={onPointerUp}
             onpointerleave={() => { if (!dragging) hoverTime = null; }}
             onpointerdown={(e) => onPointerDown(e, 'play')}>
          <div class="track">
            {#if trimming}
              <div class="dim" style:left="0" style:width="{pct(inTime)}%"></div>
              <div class="range" style:left="{pct(inTime)}%" style:width="{pct(outTime) - pct(inTime)}%"></div>
              <div class="dim" style:left="{pct(outTime)}%" style:width="{100 - pct(outTime)}%"></div>
            {:else}
              <div class="played" style:width="{pct(currentTime)}%"></div>
            {/if}
          </div>
          {#if trimming && showTicks}
            <div class="ticks" aria-hidden="true">
              {#each keyframes as k (k)}
                <span class="tick" style:left="{pct(k)}%"></span>
              {/each}
            </div>
          {/if}
          <div class="playhead" style:left="{pct(currentTime)}%" aria-hidden="true"></div>
          {#if hoverTime != null && !dragging}
            <div class="hover-label" style:left="{pct(hoverTime)}%">{fmt(hoverTime)}</div>
          {/if}
          {#if trimming}
            <div class="handle in" class:active={dragging === 'in'} style:left="{pct(inTime)}%"
                 role="slider" tabindex="0" aria-label="Start of trim" aria-valuemin="0" aria-valuemax={duration}
                 aria-valuenow={inTime} aria-valuetext={fmt(inTime)}
                 onpointerdown={(e) => onPointerDown(e, 'in')} onkeydown={(e) => onHandleKey(e, 'in')}>
              <span class="grip"></span>
              <span class="label">{fmt(inTime)}</span>
            </div>
            <div class="handle out" class:active={dragging === 'out'} style:left="{pct(outTime)}%"
                 role="slider" tabindex="0" aria-label="End of trim" aria-valuemin="0" aria-valuemax={duration}
                 aria-valuenow={outTime} aria-valuetext={fmt(outTime)}
                 onpointerdown={(e) => onPointerDown(e, 'out')} onkeydown={(e) => onHandleKey(e, 'out')}>
              <span class="grip"></span>
              <span class="label">{fmt(outTime)}</span>
            </div>
          {/if}
        </div>

        <div class="volume">
          <IconButton icon={muted || volume === 0 ? 'fa-volume-xmark' : volume < 0.5 ? 'fa-volume-low' : 'fa-volume-high'}
                      title={muted ? 'Unmute (M)' : 'Mute (M)'} onclick={() => muted = !muted} />
          <input type="range" min="0" max="1" step="0.05" bind:value={volume} aria-label="Volume"
                 oninput={() => { if (muted && volume > 0) muted = false; }} />
        </div>
      </div>

      {#if trimming}
        <div class="trim-bar">
          {#if saved}
            <div class="saved">
              <i class="fas fa-check" aria-hidden="true"></i>
              <span class="saved-text">Saved <strong>{saved.name}</strong> · {formatBytes(saved.sizeBytes)}</span>
              <span class="spacer"></span>
              {#if saved.previewPath}
                <Button size="sm" icon="fa-play" onclick={playSaved}>Play it</Button>
              {/if}
              <Button size="sm" icon="fa-folder-open" onclick={showSaved}>Show in folder</Button>
              <Button size="sm" variant="primary" onclick={() => { saved = null; }}>Trim again</Button>
            </div>
          {:else}
            <div class="points">
              <button class="point" onclick={() => setIn(currentTime)} title="Set start to the playhead (I)">
                <span class="k">In</span> <span class="v">{fmt(inTime)}</span>
              </button>
              <button class="point" onclick={() => setOut(currentTime)} title="Set end to the playhead (O)">
                <span class="k">Out</span> <span class="v">{fmt(outTime)}</span>
              </button>
              <span class="length">
                <span class="k">Length</span> <span class="v">{selectionLength.toFixed(1)} s</span>
                {#if estimatedBytes > 0}<span class="muted">· about {formatBytes(estimatedBytes)}</span>{/if}
              </span>
            </div>
            <span class="hint" title="No re-encode: the video, audio, and GPS telemetry are copied as they are. Cuts land on keyframes, about one per second on a GoPro, so both handles snap to them.">
              <i class="fas fa-lock-open" aria-hidden="true"></i>
              Lossless · on keyframes
            </span>
            <span class="spacer"></span>
            <Button size="sm" icon={loopSelection && playing ? 'fa-pause' : 'fa-repeat'}
                    onclick={() => loopSelection && playing ? video.pause() : playSelection()} title="Loop the selection">
              {loopSelection && playing ? 'Stop' : 'Loop selection'}
            </Button>
            <Button size="sm" variant="primary" icon={saving ? 'fa-spinner fa-spin' : 'fa-scissors'}
                    disabled={saving || selectionLength < MIN_LEN} onclick={save} title="Save the selection as a new file (Enter)">
              {saving ? 'Saving...' : 'Save trim'}
            </Button>
          {/if}
        </div>
        <div class="keys">
          <span><kbd>I</kbd> <kbd>O</kbd> set in / out</span>
          <span><kbd>[</kbd> <kbd>]</kbd> jump</span>
          <span><kbd>←</kbd> <kbd>→</kbd> next keyframe, <kbd>Shift</kbd> for 5</span>
          <span><kbd>Space</kbd> play</span>
          <span><kbd>Enter</kbd> save</span>
          <span><kbd>Esc</kbd> leave trim</span>
        </div>
      {/if}
      </div>
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

  /* The picture is the box; header and controls float over it */
  .player {
    position: relative;
    background: black;
    border-radius: 10px;
    overflow: hidden;
    width: min(1100px, 94vw);
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow-md);
    color: white;
    outline: none;
  }

  .player.fullscreen {
    width: 100vw;
    height: 100vh;
    border-radius: 0;
    justify-content: center;
  }

  video {
    width: 100%;
    max-height: 84vh;
    background: black;
    display: block;
    cursor: pointer;
  }
  .player.fullscreen video { max-height: 100vh; height: 100vh; object-fit: contain; }

  .player-header, .controls {
    position: absolute;
    left: 0;
    right: 0;
    transition: opacity 0.25s;
  }

  .player-header {
    top: 0;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    padding: 10px 12px 28px;
    background: linear-gradient(rgba(0, 0, 0, 0.65), transparent);
  }

  .header-actions { display: flex; align-items: center; gap: 8px; }

  .player-title {
    font-size: 0.85rem;
    font-weight: 600;
    color: white;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.6);
  }

  .controls {
    bottom: 0;
    padding: 28px 8px 8px;
    background: linear-gradient(transparent, rgba(0, 0, 0, 0.8) 45%);
  }

  .player.idle .player-header, .player.idle .controls { opacity: 0; pointer-events: none; }
  .player.idle { cursor: none; }

  /* Buttons read on the dark stage */
  .player :global(.icon-btn), .player :global(.btn-outline) {
    color: white;
    border-color: rgba(255, 255, 255, 0.35);
  }
  .player :global(.icon-btn:hover:not(:disabled)), .player :global(.btn-outline:hover:not(:disabled)) {
    background: rgba(255, 255, 255, 0.15);
    border-color: white;
    color: white;
  }

  .transport {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 4px 2px;
  }

  .time {
    font-variant-numeric: tabular-nums;
    font-size: 0.8rem;
    color: white;
    white-space: nowrap;
    min-width: 110px;
    text-shadow: 0 1px 2px rgba(0, 0, 0, 0.6);
  }
  .muted { color: rgba(255, 255, 255, 0.6); }

  /* Volume: a short slider that is always there, so nothing pops up
     under the pointer on its way to the end of the timeline */
  .volume { display: flex; align-items: center; gap: 6px; }
  .volume input[type="range"] {
    -webkit-appearance: none;
    appearance: none;
    width: 72px;
    height: 4px;
    margin: 0;
    border-radius: 2px;
    background: rgba(255, 255, 255, 0.35);
    cursor: pointer;
  }
  .volume input[type="range"]::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: white;
  }
  .transport.dragging .volume { pointer-events: none; }

  /* Timeline: a 44px tall hit area around a thin track. The side margins
     keep a handle at either end clear of the neighbouring buttons. */
  .timeline {
    position: relative;
    flex: 1;
    height: 44px;
    margin: 0 14px;
    cursor: pointer;
    touch-action: none;
    user-select: none;
  }

  .track {
    position: absolute;
    left: 0;
    right: 0;
    top: 19px;
    height: 6px;
    border-radius: 3px;
    background: rgba(255, 255, 255, 0.3);
    overflow: hidden;
    transition: height 0.1s, top 0.1s;
  }
  .timeline:hover .track { top: 18px; height: 8px; }

  .played { height: 100%; background: var(--primary-color); }
  .range { position: absolute; top: 0; height: 100%; background: var(--primary-color); opacity: 0.7; }
  .dim { position: absolute; top: 0; height: 100%; background: transparent; }

  .ticks { position: absolute; left: 0; right: 0; top: 27px; height: 4px; pointer-events: none; }
  .tick { position: absolute; width: 1px; height: 4px; background: rgba(255, 255, 255, 0.55); }

  .playhead {
    position: absolute;
    top: 14px;
    width: 2px;
    height: 16px;
    margin-left: -1px;
    background: white;
    border-radius: 1px;
    pointer-events: none;
  }
  .playhead::after {
    content: '';
    position: absolute;
    left: -5px;
    top: -6px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: white;
  }
  .trimming .playhead::after { display: none; }

  .hover-label, .handle .label {
    position: absolute;
    bottom: 100%;
    transform: translateX(-50%);
    font-size: 0.7rem;
    font-variant-numeric: tabular-nums;
    color: white;
    background: rgba(0, 0, 0, 0.75);
    border: 1px solid rgba(255, 255, 255, 0.3);
    border-radius: 4px;
    padding: 1px 6px;
    white-space: nowrap;
    pointer-events: none;
  }
  .hover-label { margin-bottom: -6px; color: rgba(255, 255, 255, 0.85); }

  /* Handles: 14px visible, 32px hit width, the whole 44px height */
  .handle {
    position: absolute;
    top: 0;
    width: 32px;
    height: 44px;
    margin-left: -16px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: ew-resize;
    outline: none;
  }
  .handle .grip {
    width: 14px;
    height: 28px;
    border-radius: 5px;
    background: var(--primary-color);
    box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.6);
    position: relative;
    transition: transform 0.1s;
  }
  .handle .grip::before, .handle .grip::after {
    content: '';
    position: absolute;
    top: 8px;
    width: 1px;
    height: 12px;
    background: rgba(255, 255, 255, 0.7);
  }
  .handle .grip::before { left: 5px; }
  .handle .grip::after { left: 8px; }
  .handle:hover .grip, .handle.active .grip, .handle:focus-visible .grip { transform: scaleY(1.12); }
  .handle:focus-visible .grip { box-shadow: 0 0 0 2px black, 0 0 0 4px white; }
  .handle .label { left: 50%; margin-bottom: 2px; opacity: 0; transition: opacity 0.1s; }
  .handle:hover .label, .handle.active .label, .handle:focus-visible .label { opacity: 1; }
  .handle.in .label { transform: translateX(-100%); left: 16px; }
  .handle.out .label { transform: none; left: 16px; }

  .trim-bar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 4px 6px;
    border-top: 1px solid rgba(255, 255, 255, 0.15);
    flex-wrap: wrap;
  }

  .points { display: flex; align-items: center; gap: 8px; }
  .point {
    display: flex;
    align-items: baseline;
    gap: 6px;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.25);
    border-radius: 6px;
    padding: 4px 10px;
    min-height: 30px;
    cursor: pointer;
    color: white;
    font-size: 0.8rem;
  }
  .point:hover { border-color: white; }
  .length { display: flex; align-items: baseline; gap: 6px; font-size: 0.8rem; padding: 4px 4px; }
  .k { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.04em; color: rgba(255, 255, 255, 0.65); }
  .v { font-variant-numeric: tabular-nums; font-weight: 600; }

  .hint { display: flex; align-items: center; gap: 6px; font-size: 0.75rem; color: rgba(255, 255, 255, 0.85); cursor: help; }
  .hint i { color: var(--state-ok); }
  .spacer { flex: 1; }

  .saved { display: flex; align-items: center; gap: 10px; width: 100%; font-size: 0.8rem; }
  .saved > i { color: var(--state-ok); }
  .saved-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .keys {
    display: flex;
    gap: 14px;
    flex-wrap: wrap;
    padding: 0 4px 2px;
    font-size: 0.7rem;
    color: rgba(255, 255, 255, 0.7);
  }
  kbd {
    display: inline-block;
    min-width: 16px;
    padding: 0 4px;
    border: 1px solid rgba(255, 255, 255, 0.3);
    border-bottom-width: 2px;
    border-radius: 4px;
    font-family: inherit;
    font-size: 0.68rem;
    line-height: 16px;
    text-align: center;
    color: white;
    background: rgba(255, 255, 255, 0.1);
  }
</style>
