<script>
  const { shell } = window.require('electron');
  const path = window.require('path');

  let { video, cameraName } = $props();

  let icon = $derived(video.mimeType?.startsWith('image/') ? 'fa-image' : 'fa-film');

  let sizeText = $derived(formatSize(video.sizeBytes));
  let dateText = $derived(video.createdAt ? video.createdAt.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) : '');

  function formatSize(bytes) {
    if (!bytes) return '';
    if (bytes >= 1e9) return `${(bytes / 1e9).toFixed(1)} GB`;
    if (bytes >= 1e6) return `${(bytes / 1e6).toFixed(1)} MB`;
    return `${Math.round(bytes / 1024)} KB`;
  }

  function openFile() {
    shell.openPath(video.path);
  }

  function showInFolder() {
    shell.showItemInFolder(video.path);
  }
</script>

<div class="card">
  <div class="card-header">
    <i class="fas {icon} file-icon"></i>
    <div class="file-info">
      <span class="filename" title={video.name}>{video.name}</span>
      <span class="meta">{cameraName}</span>
    </div>
  </div>
  <div class="card-body">
    <span class="detail">{sizeText}</span>
    <span class="detail">{dateText}</span>
  </div>
  <div class="card-actions">
    <button class="btn btn-primary" onclick={openFile}>Open</button>
    <button class="btn btn-outline" onclick={showInFolder}>Show</button>
  </div>
</div>

<style>
  .card {
    background-color: var(--panel-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    transition: all 0.2s;
  }

  .card:hover {
    box-shadow: var(--shadow-md);
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .file-icon {
    font-size: 1.4rem;
    color: var(--primary-color);
    flex-shrink: 0;
  }

  .file-info {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .filename {
    font-weight: 600;
    font-size: 0.9rem;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .meta {
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .card-body {
    display: flex;
    gap: 12px;
  }

  .detail {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .card-actions {
    display: flex;
    gap: 8px;
    margin-top: auto;
  }

  .btn {
    flex: 1;
    padding: 8px 12px;
    border: none;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 500;
    cursor: pointer;
    color: white;
    min-height: 36px;
    transition: all 0.2s;
  }

  .btn:hover { transform: translateY(-1px); }
  .btn:active { transform: translateY(0); }

  .btn-primary { background-color: var(--primary-color); }
  .btn-primary:hover { background-color: var(--primary-dark); }

  .btn-outline {
    background-color: transparent;
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
  }
  .btn-outline:hover {
    border-color: var(--text-secondary);
  }
</style>
