// Device stores — discovered and managed cameras keyed by MAC address
let discoveredDevices = $state({});
let managedDevices = $state({});
let allDevices = $state({});
let pairingInProgress = $state({});

export function getDiscoveredDevices() { return discoveredDevices; }
export function getManagedDevices() { return managedDevices; }
export function getAllDevices() { return allDevices; }
export function getPairingInProgress() { return pairingInProgress; }

export function getSeenCount() {
  return Object.values(allDevices).filter(d => d.isReachable).length;
}

// The user's name first; the SSID is the stable factory identity
// ("GoPro 7115"), the advertised name a last resort.
export function displayName(device) {
  if (!device) return 'Camera';
  if (device.alias?.trim()) return device.alias.trim();
  if (device.wifiSsid?.trim()) return device.wifiSsid.substring(0, 12);
  return device.name || 'Unknown GoPro';
}

// The factory identity, shown under an alias so the camera stays
// recognisable on its own screen
export function factoryName(device) {
  if (!device) return '';
  if (device.wifiSsid?.trim()) return device.wifiSsid.substring(0, 12);
  return device.name || '';
}

export function findDeviceById(id) {
  return Object.values(allDevices).find(d => d.id === id) || null;
}

export function updateDevice(device) {
  if (!device?.macAddress) return;
  allDevices[device.macAddress] = device;

  // Pool membership mirrors the backend (InManagedPool): paired + managed.
  // Reachability must not decide membership; a managed camera that is
  // asleep or briefly out of range stays on its card as "unreachable".
  const inManagedPool = device.isPaired && device.isManaged;

  if (inManagedPool) {
    managedDevices[device.macAddress] = device;
    delete discoveredDevices[device.macAddress];
  } else {
    discoveredDevices[device.macAddress] = device;
    delete managedDevices[device.macAddress];
  }
}

export function removeDevice(mac) {
  delete allDevices[mac];
  delete discoveredDevices[mac];
  delete managedDevices[mac];
  delete pairingInProgress[mac];
}

export function setPairingInProgress(mac, value) {
  if (value) pairingInProgress[mac] = true;
  else delete pairingInProgress[mac];
}

export function cleanupStaleDevices(timeoutMs = 30000) {
  const now = Date.now();
  for (const mac in allDevices) {
    const device = allDevices[mac];
    const lastSeen = device.lastSeen ? new Date(device.lastSeen).getTime() : 0;
    if (now - lastSeen <= timeoutMs) continue;

    if (device.isManaged && device.isPaired) {
      // Keep managed devices but mark unreachable
      device.isReachable = false;
      device.rssi = -100;
      allDevices[mac] = { ...device };
      delete discoveredDevices[mac];
      // Keep in managedDevices so card shows as unreachable
    } else {
      removeDevice(mac);
    }
  }
}
