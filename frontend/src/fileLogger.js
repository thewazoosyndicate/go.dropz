// Simple file logger for debugging with log levels
const fs = require('fs');
const path = require('path');
const os = require('os');

const logFilePath = path.join(os.homedir(), '.dropz', 'logs', 'debug.log');

// Log levels: 0 = ERROR, 1 = WARN, 2 = INFO, 3 = DEBUG, 4 = TRACE
const LOG_LEVELS = {
  ERROR: 0,
  WARN: 1,
  INFO: 2,
  DEBUG: 3,
  TRACE: 4
};

// Current log level - change this to adjust logging verbosity
// Default: DEBUG (3) - logs everything except trace details
let currentLogLevel = LOG_LEVELS.DEBUG;

// Export a function to update the log level dynamically
function setLogLevel(level) {
  if (typeof level === 'string') {
    // Convert string level to number
    switch(level.toUpperCase()) {
      case 'ERROR': currentLogLevel = LOG_LEVELS.ERROR; break;
      case 'WARN': currentLogLevel = LOG_LEVELS.WARN; break;
      case 'INFO': currentLogLevel = LOG_LEVELS.INFO; break;
      case 'DEBUG': currentLogLevel = LOG_LEVELS.DEBUG; break;
      case 'TRACE': currentLogLevel = LOG_LEVELS.TRACE; break;
    }
  } else if (Number.isInteger(level) && level >= 0 && level <= 4) {
    currentLogLevel = level;
  }
  console.log(`Log level set to: ${getLogLevelName(currentLogLevel)} (${currentLogLevel})`);
  return currentLogLevel;
}

// Clear the log file on startup
try {
  // Ensure the log directory exists
  const logDir = path.dirname(logFilePath);
  fs.mkdirSync(logDir, { recursive: true });
  
  fs.writeFileSync(logFilePath, `Debug log started: ${new Date().toISOString()}, Log level: ${getLogLevelName(currentLogLevel)}\n`);
} catch (error) {
  console.error(`Failed to create log file: ${error.message}`);
}

function getLogLevelName(level) {
  return Object.keys(LOG_LEVELS).find(key => LOG_LEVELS[key] === level) || 'UNKNOWN';
}

// Log a message to the file if it meets the current log level
function logToFile(message, level = LOG_LEVELS.INFO) {
  if (level > currentLogLevel) return;
  
  try {
    // Ensure the log directory exists
    const logDir = path.dirname(logFilePath);
    fs.mkdirSync(logDir, { recursive: true });
    
    const timestamp = new Date().toISOString();
    const levelName = getLogLevelName(level);
    const logMessage = `[${timestamp}][${levelName}] ${message}\n`;
    fs.appendFileSync(logFilePath, logMessage);
  } catch (error) {
    console.error(`Failed to write to log file: ${error.message}`);
  }
}

module.exports = {
  logToFile,
  setLogLevel,
  currentLogLevel: () => currentLogLevel,
  LOG_LEVELS
}; 