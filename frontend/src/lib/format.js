// Shared number and time formatting so every panel reads the same.

export function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 MB';
  if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
  if (bytes >= 1e6) return `${Math.max(1, Math.round(bytes / 1e6))} MB`;
  return `${Math.max(1, Math.round(bytes / 1024))} KB`;
}

export function formatRate(bps) {
  if (!bps || bps <= 0) return '';
  return `${(bps / 1e6).toFixed(1)} MB/s`;
}

// Coarse on purpose: a jittery ETA reads as broken
export function formatEta(seconds) {
  if (!isFinite(seconds) || seconds < 0) return '';
  if (seconds < 20) return 'a few seconds left';
  if (seconds < 90) return `about ${Math.round(seconds / 10) * 10} s left`;
  const min = Math.round(seconds / 60);
  return `about ${min} min left`;
}

export function formatDuration(ms) {
  if (!ms || ms <= 0) return '';
  const s = Math.round(ms / 1000);
  if (s < 60) return `${s} s`;
  const m = Math.floor(s / 60);
  return `${m} min ${String(s % 60).padStart(2, '0')} s`;
}

export function formatTimeAgo(date) {
  if (!date || date.getTime() < 86400000) return null;
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} min ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function formatClock(date) {
  return date ? date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) : '';
}

// "Today", "Yesterday", else a short date; used for day groups and history
export function formatDayLabel(date) {
  if (!date) return '';
  const day = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const diff = Math.round((today - day) / 86400000);
  if (diff === 0) return 'Today';
  if (diff === 1) return 'Yesterday';
  return date.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short' });
}

export function dayKey(date) {
  if (!date) return 'unknown';
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
}

export function plural(n, singular, pluralForm = singular + 's') {
  return `${n} ${n === 1 ? singular : pluralForm}`;
}
