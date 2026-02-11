// Shared log filter patterns for both main and renderer processes
const FILTER_PATTERNS = [
  "MapToStruct: invalid field detected *adapter.Adapter1Properties.Connectable",
  "MapToStruct: invalid field detected *adapter.Adapter1Properties.PowerState",
  "MapToStruct: invalid field detected *adapter.Adapter1Properties.Version",
  "MapToStruct: invalid field detected *adapter.Adapter1Properties.Manufacturer",
  "MapToStruct: invalid field detected *device.Device1Properties.Bonded"
];

function shouldFilterLogMessage(message) {
  if (!message.includes('[WARN]') && !message.includes('WARN')) {
    return false;
  }
  return FILTER_PATTERNS.some(pattern => message.includes(pattern));
}

module.exports = { shouldFilterLogMessage };
