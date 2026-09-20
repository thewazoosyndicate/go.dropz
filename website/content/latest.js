// The DMG name carries the version, so a static link goes stale at each release.
// On any failure (rate limit, offline, private repo) the baked-in links stay.
(async () => {
  const api = 'https://api.github.com/repos/thewazoosyndicate/go.dropz/releases/latest';
  const ext = { dmg: '.dmg', appimage: '.AppImage' };
  try {
    const res = await fetch(api, { headers: { Accept: 'application/vnd.github+json' } });
    if (!res.ok) return;
    const rel = await res.json();
    for (const a of document.querySelectorAll('[data-dl]')) {
      const asset = rel.assets.find(x => x.name.endsWith(ext[a.dataset.dl]));
      if (asset) a.href = asset.browser_download_url;
    }
    for (const v of document.querySelectorAll('[data-release]')) {
      v.dataset.release = rel.tag_name;
      v.textContent = rel.tag_name;
    }
  } catch {}
})();
