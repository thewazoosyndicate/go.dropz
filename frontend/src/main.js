const { app, BrowserWindow, ipcMain, dialog } = require('electron');
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

// Keep a global reference of objects to prevent garbage collection
let mainWindow;
let goBinary;
let isQuitting = false;
let grpcReady = false;

function checkGrpcReady(entry) {
  if (!grpcReady && entry.msg === 'gRPC server started') {
    grpcReady = true;
    log.info('gRPC server ready, notifying renderer');
    if (mainWindow && mainWindow.webContents) {
      mainWindow.webContents.send('go-binary-status', { running: true });
    }
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
    
    log.info(`Go binary process started with PID: ${childProcess.pid}`);
    
    function processGoOutput(data) {
      const output = data.toString().trim();
      if (!output) return;
      output.split('\n').forEach(line => {
        const trimmedLine = line.trim();
        if (!trimmedLine) return;

        const entry = parseGoLogLine(trimmedLine);
        checkGrpcReady(entry);

        if (mainWindow && mainWindow.webContents) {
          mainWindow.webContents.send('go-binary-log', entry);
        }
      });
    }

    childProcess.stdout.on('data', processGoOutput);
    childProcess.stderr.on('data', processGoOutput);
    
    // Only track process lifecycle events
    childProcess.on('close', (code) => {
      grpcReady = false;
      log.info(`Go binary process exited with code ${code}`);
      if (mainWindow) {
        mainWindow.webContents.send('go-binary-status', { running: false, exitCode: code });
      }
      
      // Restart the process if it crashed and we're not quitting the app
      if (code !== 0 && !isQuitting) {
        log.info('Go binary crashed, restarting...');
        setTimeout(() => {
          goBinary = startGoBinary();
        }, 1000); // Wait a second before restarting
      }
    });
    
    childProcess.on('error', (err) => {
      log.error(`Failed to start Go binary: ${err.message}`);
      if (mainWindow) {
        mainWindow.webContents.send('go-binary-error', err.message);
      }
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
  mainWindow = new BrowserWindow({
    width: 1440,
    height: 800,
    icon: path.join(process.resourcesPath, 'icon.png'),
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
      mainWindow.webContents.send('go-binary-status', { running: true });
    }
  });

  // Show window when ready
  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
    
    // Start the Go binary once the window is ready
    goBinary = startGoBinary();
  });
  
  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// Handle app events
app.on('ready', createWindow);

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

app.on('before-quit', () => {
  isQuitting = true;
  
  // Kill the Go binary process when the app is closing
  if (goBinary) {
    try {
      log.info('Terminating Go binary process...');
      if (process.platform === 'win32') {
        spawn('taskkill', ['/pid', goBinary.pid, '/f', '/t']);
      } else {
        goBinary.kill('SIGTERM');
      }
    } catch (err) {
      log.error(`Error terminating Go binary: ${err.message}`);
    }
  }
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
