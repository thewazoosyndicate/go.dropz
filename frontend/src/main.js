const { app, BrowserWindow, ipcMain, dialog, shell } = require('electron');
const path = require('path');
const log = require('electron-log');
const fs = require('fs');
const { spawn } = require('child_process');

// GPU workarounds for Linux AppImage compatibility
app.commandLine.appendSwitch('disable-gpu-vsync');
app.commandLine.appendSwitch('disable-gpu-compositing');
app.commandLine.appendSwitch('enable-features', 'VaapiVideoDecoder');
app.commandLine.appendSwitch('disable-software-rasterizer');

// Configure logging
// Set to 'info' for normal operation, 'debug' for troubleshooting, 'warn' for minimal logging
log.transports.file.level = 'info';
log.transports.file.maxSize = 5 * 1024 * 1024; // Limit log file to 5MB
log.transports.console.level = 'info';
log.info('Application starting...');

// One backend owns 127.0.0.1:50051, so one app instance owns the backend.
// Without this lock a second launch spawns a second backend, that backend
// cannot bind, and the restart loop below hammers the port forever while
// the first backend keeps serving perfectly well.
if (!app.requestSingleInstanceLock()) {
  log.info('Another instance already owns the backend, exiting');
  app.quit();
  return;
}

// Keep a global reference of objects to prevent garbage collection
let mainWindow;
let goBinary;
let isQuitting = false;
let grpcReady = false;
// Every child ever spawned, not just the latest: a restart overwrote
// goBinary, so before-quit killed the doomed child and orphaned the healthy
// one, which then held the port against every future launch.
const goChildren = new Set();
let restartTimer = null;
let restartAttempts = 0;

// The backend exits with this code when the address is already bound.
// Restarting cannot help: something else owns the port.
const EXIT_ADDR_IN_USE = 3;
const MAX_RESTART_ATTEMPTS = 5;
// Uptime that proves a backend healthy and refills the crash budget. Keyed
// on time alive, not on "started serving": a backend that serves for two
// seconds and dies must burn through the budget, not loop forever at the
// minimum delay.
const STABLE_UPTIME_MS = 60000;

// The window may already be destroyed (quit teardown) when a child event
// fires; sending to a destroyed webContents throws inside the handler.
function sendToWindow(channel, payload) {
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send(channel, payload);
  }
}

function checkGrpcReady(entry) {
  if (!grpcReady && entry.msg === 'gRPC server started') {
    grpcReady = true;
    log.info('gRPC server ready, notifying renderer');
    sendToWindow('go-binary-status', { running: true });
  }
}

// Parse one slog JSON line from the Go backend into a structured entry.
// Non-JSON lines (library prints, panic traces) pass through as raw text.
function parseGoLogLine(line) {
  if (line.startsWith('{')) {
    try {
      const obj = JSON.parse(line);
      const { time, level, msg, component, ...attrs } = obj;
      return { time, level: (level || 'INFO').toUpperCase(), msg: msg || '', component: component || '', attrs };
    } catch (_) { /* fall through */ }
  }
  return { time: new Date().toISOString(), level: 'INFO', msg: line, component: '', attrs: {} };
}

// Get the path to the Go binary based on the platform
function getGoBinaryPath() {
  const isDev = !app.isPackaged;
  let binaryName = 'dropz';
  
  if (process.platform === 'win32') {
    binaryName += '.exe';
  }
  
  if (isDev) {
    // In development, use the binary from the project root/bin directory
    return path.join(__dirname, '..', '..', 'bin', binaryName);
  } else {
    // In production, binary is included in extraResources
    return path.join(process.resourcesPath, 'bin', binaryName);
  }
}

// Start the Go binary process
function startGoBinary() {
  const binaryPath = getGoBinaryPath();
  
  log.info(`Starting Go binary from: ${binaryPath}`);
  
  if (!fs.existsSync(binaryPath)) {
    log.error(`Go binary not found at: ${binaryPath}`);
    dialog.showErrorBox('Error', `Go binary not found at: ${binaryPath}`);
    return null;
  }
  
  try {
    log.info('Executing Go binary with environment:', JSON.stringify({
      NODE_ENV: app.isPackaged ? 'production' : 'development'
    }));
    
    // JSON logs: the backend owns its rotating file; we only parse and
    // forward lines to the renderer, never write them to a second file.
    const childProcess = spawn(binaryPath, ['--log-format=json'], {
      env: { ...process.env, NODE_ENV: app.isPackaged ? 'production' : 'development' }
    });
    
    goChildren.add(childProcess);
    const startedAt = Date.now();
    log.info(`Go binary process started with PID: ${childProcess.pid}`);

    function processGoOutput(data) {
      const output = data.toString().trim();
      if (!output) return;
      output.split('\n').forEach(line => {
        const trimmedLine = line.trim();
        if (!trimmedLine) return;

        const entry = parseGoLogLine(trimmedLine);
        checkGrpcReady(entry);

        sendToWindow('go-binary-log', entry);
      });
    }

    childProcess.stdout.on('data', processGoOutput);
    childProcess.stderr.on('data', processGoOutput);
    
    // Only track process lifecycle events
    childProcess.on('close', (code) => {
      goChildren.delete(childProcess);
      grpcReady = false;
      log.info(`Go binary process exited with code ${code}`);
      sendToWindow('go-binary-status', { running: false, exitCode: code });

      if (isQuitting) {
        // Last child reaped: stop holding the quit open for the grace period.
        if (goChildren.size === 0) app.quit();
        return;
      }
      if (code === 0) return;

      if (code === EXIT_ADDR_IN_USE) {
        log.error('Backend address already in use; not restarting');
        sendToWindow('go-binary-error',
          'Another Dropz backend already owns 127.0.0.1:50051. Quit it and relaunch.');
        return;
      }

      // A long-lived backend was healthy: refill the crash budget so a
      // crash days later gets its full allowance of restarts.
      if (Date.now() - startedAt > STABLE_UPTIME_MS) restartAttempts = 0;

      restartAttempts++;
      if (restartAttempts > MAX_RESTART_ATTEMPTS) {
        log.error(`Go binary crashed ${restartAttempts} times, giving up`);
        sendToWindow('go-binary-error',
          `Backend crashed ${restartAttempts} times in a row. See ~/.dropz/logs/dropz.log.`);
        return;
      }

      // Exponential backoff: the old flat 1s retry produced ~90 processes in
      // 100 seconds, each creating and abandoning a CoreBluetooth central
      // manager on the way out.
      const delay = Math.min(1000 * 2 ** (restartAttempts - 1), 30000);
      log.info(`Go binary crashed, restarting in ${delay}ms (attempt ${restartAttempts}/${MAX_RESTART_ATTEMPTS})`);
      restartTimer = setTimeout(() => {
        restartTimer = null;
        // Re-checked here, not only at close time: quitting during the delay
        // used to spawn a fresh backend that outlived the app.
        if (isQuitting) return;
        goBinary = startGoBinary();
      }, delay);
    });
    
    childProcess.on('error', (err) => {
      log.error(`Failed to start Go binary: ${err.message}`);
      sendToWindow('go-binary-error', err.message);
    });
    
    return childProcess;
  } catch (err) {
    log.error(`Exception starting Go binary: ${err.message}`);
    dialog.showErrorBox('Error', `Failed to start Go binary: ${err.message}`);
    return null;
  }
}

// Create the main application window
function createWindow() {
  // process.resourcesPath points inside node_modules/electron until packaged
  const iconPath = app.isPackaged
    ? path.join(process.resourcesPath, 'icon.png')
    : path.join(__dirname, '..', 'resources', 'icon.png');
  mainWindow = new BrowserWindow({
    width: 1440,
    height: 800,
    icon: iconPath,
    autoHideMenuBar: true,
    webPreferences: {
      nodeIntegration: true,
      contextIsolation: false,
      preload: path.join(__dirname, 'preload.js'),
      webSecurity: true
    },
    show: false
  });

  // Set Content-Security-Policy before loading the file
  mainWindow.webContents.session.webRequest.onHeadersReceived((details, callback) => {
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        'Content-Security-Policy': ['script-src \'self\' \'unsafe-eval\'']
      }
    });
  });
  
  // Load Vite build output (Svelte app)
  mainWindow.loadFile(path.join(__dirname, '..', 'dist-svelte', 'index.html'));

  // Re-send backend status after reload (Ctrl+R)
  mainWindow.webContents.on('did-finish-load', () => {
    if (grpcReady) {
      sendToWindow('go-binary-status', { running: true });
    }
  });

  // Show window when ready. Deliberately does NOT start the backend: on
  // macOS, closing the window and clicking the dock icon runs createWindow
  // again, which spawned a second backend onto the port the first one still
  // held. The backend's lifetime belongs to the app, not to the window.
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// Handle app events
app.on('ready', () => {
  goBinary = startGoBinary();
  createWindow();
});

app.on('second-instance', () => {
  if (!mainWindow) {
    createWindow();
    return;
  }
  if (mainWindow.isMinimized()) mainWindow.restore();
  mainWindow.focus();
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  }
});

// Grace period between SIGTERM and SIGKILL. A backend wedged in a stalled
// BLE teardown ignores SIGTERM: it has already released the gRPC listener
// but never finishes goProManager.Stop(), so it survives as an orphan.
const SHUTDOWN_GRACE_MS = 3000;

function killChild(child) {
  try {
    log.info(`Terminating Go binary process ${child.pid}...`);
    if (process.platform === 'win32') {
      spawn('taskkill', ['/pid', child.pid, '/f', '/t']);
      return;
    }
    child.kill('SIGTERM');
  } catch (err) {
    log.error(`Error terminating Go binary: ${err.message}`);
  }
}

function forceKillChild(child) {
  try {
    if (child.exitCode === null && child.signalCode === null) {
      log.warn(`Go binary ${child.pid} ignored SIGTERM, sending SIGKILL`);
      child.kill('SIGKILL');
    }
  } catch (err) {
    log.error(`Error killing Go binary: ${err.message}`);
  }
}

app.on('before-quit', (event) => {
  if (restartTimer) {
    clearTimeout(restartTimer);
    restartTimer = null;
  }

  if (isQuitting) return; // second pass: children already handled, let it go
  isQuitting = true;

  if (goChildren.size === 0) return;

  // Hold the quit open long enough to actually reap the children. Without
  // this Electron exits immediately and any child that ignores SIGTERM is
  // orphaned onto the port.
  event.preventDefault();

  // Every child, not just the latest: a restart overwrote goBinary, so the
  // healthy backend was never the one that got signalled.
  for (const child of goChildren) killChild(child);

  setTimeout(() => {
    for (const child of goChildren) forceKillChild(child);
    app.quit();
  }, SHUTDOWN_GRACE_MS);
});

// On Linux, Electron's shell.openPath waits for xdg-open to exit and
// showItemInFolder blocks on a D-Bus round-trip; a slow desktop handler
// froze the UI for the full timeout. Detached spawns never block.
function spawnDetached(cmd, args, onFail) {
  let failed = false;
  const fail = (msg) => {
    log.warn(`${cmd} ${msg}`);
    if (onFail && !failed) { failed = true; onFail(); }
  };
  const child = spawn(cmd, args, { detached: true, stdio: 'ignore' });
  child.on('error', (err) => fail(`failed: ${err.message}`));
  child.on('exit', (code) => { if (code !== 0) fail(`exited with ${code}`); });
  child.unref();
}

ipcMain.on('desktop-open', (_event, target) => {
  if (typeof target !== 'string' || !target) return;
  log.info(`Desktop open: ${target}`);
  if (process.platform !== 'linux') { shell.openPath(target); return; }
  spawnDetached('xdg-open', [target]);
});

ipcMain.on('desktop-show', (_event, target) => {
  if (typeof target !== 'string' || !target) return;
  log.info(`Desktop show: ${target}`);
  if (process.platform !== 'linux') { shell.showItemInFolder(target); return; }
  const uri = require('url').pathToFileURL(target).href;
  spawnDetached('gdbus', [
    'call', '--session',
    '--dest', 'org.freedesktop.FileManager1',
    '--object-path', '/org/freedesktop/FileManager1',
    '--method', 'org.freedesktop.FileManager1.ShowItems',
    `['${uri}']`, '',
  ], () => spawnDetached('xdg-open', [path.dirname(target)]));
});

// Handle folder selection dialog for destination folder setting
ipcMain.on('open-folder-dialog', (event) => {
  log.info('Open folder dialog requested');
  
  if (!mainWindow) {
    log.warn('Cannot open folder dialog: main window not available');
    return;
  }
  
  dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory', 'createDirectory'],
    title: 'Select Destination Folder for Media Downloads'
  }).then(result => {
    if (!result.canceled && result.filePaths.length > 0) {
      log.info(`Folder selected: ${result.filePaths[0]}`);
      event.reply('selected-folder', result.filePaths[0]);
    } else {
      log.info('Folder selection canceled');
    }
  }).catch(err => {
    log.error(`Error showing folder dialog: ${err.message}`);
  });
});
