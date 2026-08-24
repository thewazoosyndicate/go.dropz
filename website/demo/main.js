// Offscreen Electron driver: boots the real renderer against the fake
// backend, plays the storyboard with real input events, and writes one
// PNG per tick so the encoded video keeps honest timing.
const path = require('path');
const fs = require('fs');
const os = require('os');
const { app, BrowserWindow } = require('electron');
const { createFakeServer } = require('./fake-server.js');

const ROOT = path.resolve(__dirname, '..', '..');
const FRONTEND = path.join(ROOT, 'frontend');
const WORK = process.env.WORK || path.join(os.tmpdir(), 'dropz-demo');
const MEDIA = path.join(WORK, 'media');
const FRAMES = path.join(WORK, 'frames');
const STILLS = path.join(WORK, 'stills');
const FPS = Number(process.env.FPS || 24);
const ADDR = process.env.DROPZ_GRPC_ADDR || '127.0.0.1:50061';
const W = 1280, H = 800;

app.disableHardwareAcceleration();
app.commandLine.appendSwitch('force-device-scale-factor', '1');
process.env.DROPZ_GRPC_ADDR = ADDR;

const sleep = (ms) => new Promise(r => setTimeout(r, ms));
const MB = 1e6;
const LIB = '/home/alex/Videos';
const now = Date.now();
const ago = (min) => now - min * 60000;
const media = (f) => path.join(MEDIA, f);

// ---- world state -------------------------------------------------------

const cams = {
  c4231: { id: 'cam-4231', name: 'GoPro 4231', ssid: 'GoPro 4231', alias: 'Leo #9', ble: 'D8:3A:DD:41:23:10', rssi: -52,
    model: 'HERO13 Black', serial: 'C3501324500423', firmware: 'H24.01.02.20', battery: 78,
    numVideos: 41, numPhotos: 12, remainingKb: 93 * 1024 * 1024, isPaired: true, isManaged: true,
    isReachable: true, isSyncing: true, groupId: 'g1', lastSynced: ago(180), visible: false },
  c0882: { id: 'cam-0882', name: 'GoPro 0882', ssid: 'GoPro 0882', alias: 'Maya #4', ble: 'F4:5C:89:08:82:2A', rssi: -61,
    model: 'HERO12 Black', serial: 'C3471324988201', firmware: 'H23.01.02.32', battery: 54,
    numVideos: 27, numPhotos: 3, remainingKb: 41 * 1024 * 1024, isPaired: true, isManaged: true,
    isReachable: true, newMediaCount: 3, groupId: 'g1', lastSynced: ago(26 * 60), visible: false },
  c7115: { id: 'cam-7115', name: 'GoPro 7115', ssid: 'GoPro 7115', alias: 'Sam #11', ble: 'C0:98:E5:71:15:0F', rssi: -58,
    model: 'HERO11 Black', serial: 'C3461324711509', firmware: 'H22.01.02.32', battery: 91,
    numVideos: 66, numPhotos: 0, remainingKb: 118 * 1024 * 1024, isPaired: true, isManaged: true,
    isReachable: true, isSynced: true, groupId: 'g1', lastSynced: ago(95), visible: false },
  c5560: { id: 'cam-5560', name: 'GoPro 5560', ssid: 'GoPro 5560', ble: 'E2:11:70:55:60:B3', rssi: -47,
    model: 'HERO11 Black', serial: 'C3461324556003', firmware: 'H22.01.02.32', battery: 66,
    numVideos: 2, numPhotos: 0, remainingKb: 240 * 1024 * 1024, isReachable: true, inPairingMode: true, visible: false },
  c9014: { id: 'cam-9014', name: 'GoPro 9014', ssid: 'GoPro 9014', ble: 'A8:6D:AA:90:14:77', rssi: -77,
    model: 'HERO12 Black', battery: 32, isReachable: true, visible: false },
};

const clip = (id, name, cam, min, mb) => ({
  id, name: `${name}.MP4`, path: `${LIB}/${name}.MP4`, sizeBytes: mb * MB, createdAt: ago(min),
  cameraId: cam.id, thumbnailPath: media(`${name}.jpg`), previewPath: media(`${name}.webm`),
});

const newClips = [clip('v42', 'GX010042', cams.c4231, 32, 58), clip('v43', 'GX010043', cams.c4231, 27, 71), clip('v44', 'GX010044', cams.c4231, 21, 32)];
const clips0882 = [clip('v17b', 'GX020118', cams.c0882, 48, 74), clip('v18b', 'GX020119', cams.c0882, 44, 44)];
const clip5560 = [clip('v55', 'GH010391', cams.c5560, 15, 55)];

const doneFile = (c, secs) => ({ name: c.name, sizeBytes: c.sizeBytes, state: 'done', bytesDone: c.sizeBytes, localPath: c.path, durationMs: secs * 1000 });
const skippedFile = (c) => ({ name: c.name, sizeBytes: c.sizeBytes, state: 'skipped', localPath: c.path });
const phasesOf = (t0, ...durs) => {
  const out = []; let t = t0;
  ['connect', 'link', 'catalog', 'transfer'].forEach((p, i) => { if (durs[i] == null) return; out.push({ phase: p, startedAt: t, finishedAt: t + durs[i] * 1000 }); t += durs[i] * 1000; });
  return out;
};

const state = {
  libraryDir: LIB,
  cameras: Object.values(cams),
  groups: [{ id: 'g1', name: 'U16 squad', cameraIds: ['cam-4231', 'cam-0882', 'cam-7115', 'cam-5560'], createdAt: ago(26 * 60) }],
  queue: [
    { cameraId: 'cam-4231', queuedAt: ago(0.5), priority: 5, phase: 'waiting' },
    { cameraId: 'cam-0882', queuedAt: ago(0.4), priority: 5, phase: 'waiting' },
  ],
  videos: [
    clip('v91', 'GH010391', cams.c7115, 125, 96), clip('v92', 'GH010392', cams.c7115, 110, 132),
    clip('v17', 'GX020117', cams.c0882, 26 * 60 + 30, 210), clip('v39', 'GX010039', cams.c4231, 27 * 60 + 10, 88),
    clip('v40', 'GX010040', cams.c4231, 27 * 60, 102),
  ],
  history: [],
};
{
  const v = Object.fromEntries(state.videos.map(x => [x.id, x]));
  state.history = [
    { id: 's5', cameraId: 'cam-7115', startedAt: ago(100), finishedAt: ago(99.9), outcome: 'up_to_date',
      phases: phasesOf(ago(100), 4, 2), files: [] },
    { id: 's4', cameraId: 'cam-7115', startedAt: ago(108), finishedAt: ago(106), outcome: 'complete',
      filesDownloaded: 2, bytesDownloaded: 228 * MB, phases: phasesOf(ago(108), 3, 5, 2, 110),
      files: [doneFile(v.v91, 48), doneFile(v.v92, 62)] },
    { id: 's3', cameraId: 'cam-4231', startedAt: ago(27 * 60 + 8), finishedAt: ago(27 * 60 + 6), outcome: 'complete',
      filesDownloaded: 2, bytesDownloaded: 190 * MB, phases: phasesOf(ago(27 * 60 + 8), 3, 4, 2, 105),
      files: [doneFile(v.v39, 44), doneFile(v.v40, 55)] },
    { id: 's2', cameraId: 'cam-4231', startedAt: ago(27 * 60 + 14), finishedAt: ago(27 * 60 + 13), outcome: 'failed',
      error: 'Wi-Fi join timed out', failedStep: 'Join camera Wi-Fi', stepIndex: 4, stepCount: 9,
      phases: phasesOf(ago(27 * 60 + 14), 3, 40), files: [] },
    { id: 's1', cameraId: 'cam-0882', startedAt: ago(26 * 60 + 34), finishedAt: ago(26 * 60 + 31), outcome: 'complete',
      filesDownloaded: 1, filesSkipped: 3, bytesDownloaded: 210 * MB, phases: phasesOf(ago(26 * 60 + 34), 3, 6, 2, 150),
      files: [doneFile(v.v17, 118), skippedFile(v.v39), skippedFile(v.v40), skippedFile(v.v91)] },
  ];
}

// ---- sync simulation ----------------------------------------------------

let server;
let sessionSeq = 10;

// Advances one queue entry through the four phases on wall-clock time.
// files: clips to download; skipped: clips already in the library.
function simulateSync(cam, { connect, link, catalog, rateBps, files, skipped = [], onDone }) {
  const entry = state.queue.find(e => e.cameraId === cam.id);
  const t0 = Date.now();
  const tLink = t0 + connect * 1000, tCatalog = tLink + link * 1000, tTransfer = tCatalog + catalog * 1000;
  const total = files.reduce((n, f) => n + f.sizeBytes, 0);
  Object.assign(cam, { isSyncing: true, isSynced: false });
  Object.assign(entry, { startedAt: t0, stepCount: 9 });
  const ops = { connect: ['Connecting over Bluetooth', 5, 1], link: ['Joining camera Wi-Fi', 15, 4], catalog: ['Reading the media list', 25, 6] };

  const timer = setInterval(() => {
    const t = Date.now();
    const phase = t < tLink ? 'connect' : t < tCatalog ? 'link' : t < tTransfer ? 'catalog' : 'transfer';
    const bounds = { connect: [t0, tLink], link: [tLink, tCatalog], catalog: [tCatalog, tTransfer], transfer: [tTransfer, null] };
    entry.phase = phase;
    entry.phases = Object.entries(bounds).filter(([, [s]]) => s <= t)
      .map(([p, [s, e]]) => ({ phase: p, startedAt: s, finishedAt: e && e <= t ? e : null }));
    if (phase !== 'transfer') {
      const [op, pct, step] = ops[phase];
      Object.assign(entry, { currentOperation: op, progressPercent: pct, stepIndex: step, fileCount: 0, rateBps: 0 });
      entry.files = phase === 'catalog' ? files.map(f => ({ name: f.name, sizeBytes: f.sizeBytes, state: 'queued' })).concat(skipped.map(skippedFile)) : [];
    } else {
      let done = Math.min(total, (t - tTransfer) / 1000 * rateBps);
      let left = done, idx = 0;
      entry.files = files.map((f, i) => {
        const bytes = Math.min(f.sizeBytes, Math.max(0, left)); left -= f.sizeBytes;
        const st = bytes >= f.sizeBytes ? 'done' : bytes > 0 || (i === 0 && idx === 0) ? 'downloading' : 'queued';
        if (st === 'downloading') { idx = i + 1; Object.assign(entry, { fileName: f.name, fileBytes: bytes, fileTotal: f.sizeBytes }); }
        return { name: f.name, sizeBytes: f.sizeBytes, state: st, bytesDone: bytes, localPath: st === 'done' ? f.path : '', durationMs: st === 'done' ? f.sizeBytes / rateBps * 1000 : 0 };
      }).concat(skipped.map(skippedFile));
      Object.assign(entry, { currentOperation: 'Downloading', progressPercent: 25 + 75 * done / total, stepIndex: 7,
        fileIndex: idx || files.length, fileCount: files.length, bytesDone: done, bytesTotal: total, rateBps });
      if (done >= total) {
        clearInterval(timer);
        state.queue.splice(state.queue.indexOf(entry), 1);
        Object.assign(cam, { isSyncing: false, isSynced: true, lastSynced: t, newMediaCount: 0 });
        state.history.unshift({ id: `s${sessionSeq++}`, cameraId: cam.id, startedAt: t0, finishedAt: t, outcome: 'complete',
          filesDownloaded: files.length, filesSkipped: skipped.length, bytesDownloaded: total,
          phases: entry.phases.map(p => ({ ...p, finishedAt: p.finishedAt || t })),
          files: files.map(f => doneFile(f, f.sizeBytes / rateBps)).concat(skipped.map(skippedFile)) });
        state.videos.push(...files);
        server.notify('queue', 'managed');
        if (onDone) onDone();
        return;
      }
    }
    server.notify('queue');
  }, 200);
}

// Ideal-workflow speeds: fast phases, Turbo Transfer rates
const FAST = { connect: 1.2, link: 1.4, catalog: 1.0, rateBps: 28 * MB };
const sync4231 = () => simulateSync(cams.c4231, { ...FAST, files: newClips, skipped: [state.videos[3], state.videos[4]] });
const sync0882 = () => simulateSync(cams.c0882, { ...FAST, files: clips0882, skipped: [state.videos[2]] });
const sync5560 = () => simulateSync(cams.c5560, { ...FAST, files: clip5560 });

const hooks = {
  onPaired: (cam) => { Object.assign(cam, { newMediaCount: 2, groupId: 'g1', inPairingMode: false }); },
  onTrim: (videoPath, startMs, endMs) => {
    const src = state.videos.find(v => v.path === videoPath);
    if (!src) return null;
    const mmss = (ms) => { const s = Math.round(ms / 1000); return `${String(Math.floor(s / 60)).padStart(2, '0')}${String(s % 60).padStart(2, '0')}`; };
    const base = src.name.replace(/\.MP4$/i, '');
    const name = `${base}_trim-${mmss(startMs)}-${mmss(endMs)}.MP4`;
    const out = { id: `trim-${base}`, name, path: `${LIB}/${name}`, sizeBytes: Math.round(src.sizeBytes * (endMs - startMs) / 12000),
      createdAt: Date.now(), cameraId: src.cameraId, thumbnailPath: media(`${base}_trim.jpg`), previewPath: src.previewPath };
    state.videos.push(out);
    return { ...out, startMs, endMs };
  },
};

// ---- driver ------------------------------------------------------------

let win;
const page = (js) => win.webContents.executeJavaScript(js, true);
let cur = { x: W / 2, y: H / 2 };
const ease = (k) => k < 0.5 ? 2 * k * k : 1 - Math.pow(-2 * k + 2, 2) / 2;

const CURSOR_JS = `(() => {
  const c = document.createElement('div'); c.id = 'demo-cursor';
  c.style.cssText = 'position:fixed;left:-50px;top:-50px;width:22px;height:30px;z-index:2147483647;pointer-events:none;';
  c.innerHTML = '<svg viewBox="0 0 22 30" width="22" height="30"><path d="M2 2 L2 23 L7.5 18 L11 27 L15 25.5 L11.5 17 L19 17 Z" fill="#fff" stroke="#111" stroke-width="1.6" stroke-linejoin="round"/></svg>';
  document.body.appendChild(c);
  window.__cur = (x, y) => { c.style.left = (x - 3) + 'px'; c.style.top = (y - 2) + 'px'; };
  window.__curShow = (on) => { c.style.display = on ? '' : 'none'; };
  window.__ripple = (x, y) => {
    const r = document.createElement('div');
    r.style.cssText = 'position:fixed;width:34px;height:34px;border-radius:50%;border:2px solid rgba(52,152,219,.9);z-index:2147483646;pointer-events:none;left:' + (x - 17) + 'px;top:' + (y - 17) + 'px;transform:scale(.3);opacity:1;transition:transform .35s ease-out,opacity .35s ease-out';
    document.body.appendChild(r);
    requestAnimationFrame(() => { r.style.transform = 'scale(1)'; r.style.opacity = '0'; });
    setTimeout(() => r.remove(), 400);
  };
})()`;

async function pointer(x, y, modifiers = []) {
  win.webContents.sendInputEvent({ type: 'mouseMove', x: Math.round(x), y: Math.round(y), modifiers });
  await page(`__cur(${x.toFixed(1)}, ${y.toFixed(1)})`);
}

async function moveTo(x, y, ms = 650, modifiers = []) {
  const from = { ...cur };
  // ~30 ms per step: one IPC round-trip per step already costs ~10-15 ms,
  // so a finer cadence would just make the move drift slower than ms
  const steps = Math.max(1, Math.round(ms / 30));
  for (let i = 1; i <= steps; i++) {
    const k = ease(i / steps);
    await pointer(from.x + (x - from.x) * k, from.y + (y - from.y) * k, modifiers);
    await sleep(14);
  }
  cur = { x, y };
}

async function click() {
  const { x, y } = cur;
  await page(`__ripple(${x.toFixed(1)}, ${y.toFixed(1)})`);
  win.webContents.sendInputEvent({ type: 'mouseDown', x: Math.round(x), y: Math.round(y), button: 'left', clickCount: 1 });
  await sleep(80);
  win.webContents.sendInputEvent({ type: 'mouseUp', x: Math.round(x), y: Math.round(y), button: 'left', clickCount: 1 });
}

async function drag(x, y, ms = 1500) {
  const { x: sx, y: sy } = cur;
  win.webContents.sendInputEvent({ type: 'mouseDown', x: Math.round(sx), y: Math.round(sy), button: 'left', clickCount: 1 });
  await sleep(120);
  await moveTo(x, y, ms, ['leftButtonDown']);
  await sleep(120);
  win.webContents.sendInputEvent({ type: 'mouseUp', x: Math.round(x), y: Math.round(y), button: 'left', clickCount: 1 });
}

function rectJs(sel, text, inner) {
  return `(() => {
    const els = [...document.querySelectorAll(${JSON.stringify(sel)})];
    let el = ${text ? `els.find(e => (e.querySelector('[class*=name]') || e).textContent.trim().startsWith(${JSON.stringify(text)}))` : 'els[0]'};
    if (el && ${JSON.stringify(inner || '')}) el = el.querySelector(${JSON.stringify(inner || '')});
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: r.left + r.width / 2, y: r.top + r.height / 2, l: r.left, t: r.top, w: r.width, h: r.height };
  })()`;
}

async function find(sel, text, inner, timeoutMs = 4000) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const r = await page(rectJs(sel, text, inner));
    if (r) return r;
    if (Date.now() > deadline) throw new Error(`not found: ${sel} ${text || ''} ${inner || ''}`);
    await sleep(60);
  }
}

async function moveToEl(sel, text, inner, ms) {
  const r = await find(sel, text, inner);
  await moveTo(r.x, r.y, ms);
  return r;
}

function pressKey(keyCode, modifiers = []) {
  win.webContents.sendInputEvent({ type: 'keyDown', keyCode, modifiers });
  if (keyCode.length === 1) win.webContents.sendInputEvent({ type: 'char', keyCode, modifiers });
  win.webContents.sendInputEvent({ type: 'keyUp', keyCode, modifiers });
}

async function still(name) {
  await page('__curShow(false)');
  await sleep(40);
  const img = await win.webContents.capturePage();
  await fs.promises.writeFile(path.join(STILLS, `${name}.png`), img.toPNG());
  await page('__curShow(true)');
}

// ---- recording ---------------------------------------------------------

const rec = { running: false, start: 0, next: 0, writes: [] };
const framePath = (i) => path.join(FRAMES, `f_${String(i).padStart(5, '0')}.png`);

// Frame index follows the wall clock, not the loop: a slow capture fills
// the ticks it missed with the frame it produced, so playback keeps real time
function startRecording() {
  rec.running = true; rec.start = Date.now(); rec.next = 0;
  (async () => {
    while (rec.running) {
      const img = await win.webContents.capturePage();
      const png = img.toPNG();
      const idx = Math.floor((Date.now() - rec.start) * FPS / 1000);
      for (let j = rec.next; j <= idx; j++) rec.writes.push(fs.promises.writeFile(framePath(j), png));
      rec.next = idx + 1;
      await sleep(5);
    }
  })();
}

async function stopRecording() {
  rec.running = false;
  await sleep(300);
  await Promise.all(rec.writes);
  return rec.next;
}

// ---- storyboard --------------------------------------------------------

// Sequence-driven, not deadline-driven: each animated move does an IPC
// round-trip per step and runs slower than any absolute schedule would
// predict, so beats run in order with explicit holds. Every step renders.
async function storyboard() {
  const hold = (ms) => sleep(ms);
  const show = (...cs) => { cs.forEach(c => { c.visible = true; }); server.notify('discovered', 'managed'); };

  show(cams.c4231, cams.c0882, cams.c7115); sync4231();
  await sleep(400);
  startRecording();

  // 1. Two more cameras wake up nearby, a second sync starts
  await hold(700); show(cams.c5560);
  await hold(700); show(cams.c9014);
  await hold(500); sync0882();
  await hold(900);

  // 2. Pair the camera that is in pairing mode
  await moveToEl('.discovered-section .row', 'GoPro 5560', 'button', 650);
  await click();
  await hold(1600); // pairing spinner, then it lands in the group
  await moveToEl('.cards-grid .card', 'Leo #9', null, 700);
  state.queue.push({ cameraId: 'cam-5560', queuedAt: Date.now(), priority: 5, phase: 'waiting' }); server.notify('queue');
  await hold(1400); await still('shot-cameras');

  // 3. A two-second glance at Activity (still is the real deliverable)
  await moveToEl('.tab-bar .tab', 'Activity', null, 500);
  await click();
  await hold(1400); await still('shot-activity');

  // 4. The library fills as clips land
  await moveToEl('.tab-bar .tab', 'Library', null, 450);
  await click();
  await hold(2200); await still('shot-library');

  // 5. One lossless trim, in and out on the keyframe grid
  await moveToEl('.media-card', 'GX010042.MP4', '.ctl', 650);
  await click();
  await hold(900);
  // Keyboard, not clicks: the Trim/Close buttons sit over the <video>, whose
  // onclick=togglePlay would eat a real mouse click at those coordinates.
  pressKey('t');
  await hold(700);
  const tl = await find('.timeline');
  const xAt = (sec) => tl.l + tl.w * sec / 12;
  await moveToEl('.handle.in', null, null, 350);
  await drag(xAt(3), cur.y, 900);
  await hold(400); sync5560();
  await moveToEl('.handle.out', null, null, 350);
  await drag(xAt(8), cur.y, 900);
  await hold(900); await still('shot-trim');
  pressKey('Return');
  await hold(2600); // trim runs, the Saved bar appears
  // First Escape leaves trim mode, the second closes the player
  pressKey('Escape'); await hold(700);
  pressKey('Escape'); await hold(900);

  // 6. Back to a clean, everything-synced Cameras tab (the poster frame).
  // Ctrl+1 is the app's own tab shortcut; the player is closed now so it
  // takes. The cursor still travels to the tab so the move reads as intent.
  await moveToEl('.tab-bar .tab', 'Cameras', null, 450);
  pressKey('1', ['control']);
  await moveTo(W * 0.6, H * 0.88, 700);
  await hold(2400);
  const frames = await stopRecording();
  await still('final');
  return frames;
}

async function main() {
  for (const d of [FRAMES, STILLS]) fs.mkdirSync(d, { recursive: true });
  server = createFakeServer({ address: ADDR, state, hooks });
  await server.start();
  const heartbeat = setInterval(() => server.notify('discovered', 'managed'), 8000);

  win = new BrowserWindow({
    width: W, height: H, useContentSize: true, show: false,
    webPreferences: {
      offscreen: true, nodeIntegration: true, contextIsolation: false,
      preload: path.join(FRONTEND, 'src', 'preload.js'), webSecurity: true, backgroundThrottling: false,
    },
  });
  win.webContents.setFrameRate(30);
  win.webContents.on('console-message', (_e, level, msg) => { if (level >= 2) console.log('[renderer]', msg); });

  const loaded = () => new Promise(r => win.webContents.once('did-finish-load', r));
  const p1 = loaded();
  win.loadFile(path.join(FRONTEND, 'dist-svelte', 'index.html'));
  await p1;
  // Theme and "new since" marker live in localStorage, read at module init
  await page(`localStorage.setItem('darkMode', 'true'); localStorage.setItem('libraryLastVisit', String(Date.now()));`);
  const p2 = loaded();
  win.webContents.reload();
  await p2;
  await page(CURSOR_JS);
  win.webContents.send('go-binary-status', { running: true });
  await sleep(600);

  if (process.env.PROBE) {
    const t = Date.now();
    for (let i = 0; i < 20; i++) { const img = await win.webContents.capturePage(); img.toPNG(); }
    console.log(`capture+png: ${(Date.now() - t) / 20} ms avg`);
    await still('probe');
    clearInterval(heartbeat); app.exit(0); return;
  }

  const frames = await storyboard();
  console.log(`frames: ${frames} at ${FPS} fps (${(frames / FPS).toFixed(1)} s)`);
  clearInterval(heartbeat);
  server.stop();
  app.exit(0);
}

app.whenReady().then(main).catch(err => { console.error(err); app.exit(1); });
