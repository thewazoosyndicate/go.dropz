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

export function updateDevice(device) {
  if (!device?.macAddress) return;
  allDevices[device.macAddress] = device;

  const isDiscovered = !device.isPaired || !device.isManaged;
  const isManaged = device.isReachable && device.isPaired && device.isManaged;

  if (isDiscovered) {
    discoveredDevices[device.macAddress] = device;
  } else {
    delete discoveredDevices[device.macAddress];
  }

  if (isManaged) {
    managedDevices[device.macAddress] = device;
  } else {
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
