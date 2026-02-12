// Preload script — exposes app root path for Svelte's window.require() calls
const path = require('path');

// __dirname here is the directory of preload.js (frontend/src/)
// Go up one level to get the frontend app root
window.__appRoot = path.resolve(__dirname, '..');
