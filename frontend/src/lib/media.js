// One item shape for the library and the camera catalog, so both views
// share the same card, grid, sort, and grouping.
import { formatBytes, formatDayLabel, dayKey, plural } from './format.js';

export function fromLibraryVideo(v, cameraName) {
  return {
    key: v.id,
    name: v.name,
    sizeBytes: v.sizeBytes,
    createdAt: v.createdAt,
    thumbnailPath: v.thumbnailPath,
    previewPath: v.previewPath,
    localPath: v.path,
    cameraPath: '',
    isImage: !!v.mimeType?.startsWith('image/'),
    downloaded: true,
    canPreview: !v.mimeType?.startsWith('image/'),
    cameraId: v.cameraId,
    cameraName,
  };
}

export function fromCatalogItem(i, cameraId, cameraName) {
  return {
    key: i.cameraPath,
    name: i.name,
    sizeBytes: i.sizeBytes,
    createdAt: i.createdAt,
    thumbnailPath: i.thumbnailPath,
    previewPath: i.previewPath,
    localPath: i.localPath,
    cameraPath: i.cameraPath,
    isImage: i.name.toLowerCase().endsWith('.jpg'),
    downloaded: i.downloaded,
    // Only GX/GH videos carry an LRV proxy on the card
    canPreview: /^G[XH].*\.MP4$/i.test(i.name),
    cameraId,
    cameraName,
  };
}

export function filterKind(items, kind) {
  if (kind === 'video') return items.filter(i => !i.isImage);
  if (kind === 'photo') return items.filter(i => i.isImage);
  return items;
}

export function sortItems(items, sortBy, asc) {
  const list = [...items];
  const dir = asc ? 1 : -1;
  list.sort((a, b) => {
    if (sortBy === 'name') return dir * a.name.localeCompare(b.name);
    if (sortBy === 'size') return dir * (a.sizeBytes - b.sizeBytes);
    return dir * ((a.createdAt?.getTime() || 0) - (b.createdAt?.getTime() || 0));
  });
  return list;
}

// Day groups only make sense in date order; other sorts stay flat.
// The header names the camera when every file in the day came from one.
export function groupItems(sorted, sortBy) {
  if (sortBy !== 'date') return [{ key: 'flat', label: '', meta: '', items: sorted }];
  const out = [];
  const byKey = new Map();
  for (const item of sorted) {
    const key = dayKey(item.createdAt);
    let g = byKey.get(key);
    if (!g) {
      g = { key, label: formatDayLabel(item.createdAt), items: [] };
      byKey.set(key, g);
      out.push(g);
    }
    g.items.push(item);
  }
  for (const g of out) {
    const bytes = g.items.reduce((n, i) => n + (i.sizeBytes || 0), 0);
    const cams = new Set(g.items.map(i => i.cameraName));
    const from = cams.size === 1 && g.items[0].cameraName ? `from ${g.items[0].cameraName}` : '';
    g.meta = [plural(g.items.length, 'file'), formatBytes(bytes), from].filter(Boolean).join(' · ');
  }
  return out;
}
