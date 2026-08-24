#!/usr/bin/env bash
# Point the download links at the latest GitHub release.
# The DMG name carries the version, so its link cannot use /releases/latest.
set -euo pipefail

repo=thewazoosyndicate/go.dropz
html="$(dirname "$0")/content/index.html"

tag=$(gh release view -R "$repo" --json tagName -q .tagName)
dmg=$(gh release view -R "$repo" --json assets -q '.assets[].name | select(endswith(".dmg"))' | head -1)
[ -n "$tag" ] && [ -n "$dmg" ] || { echo "no release or no dmg asset" >&2; exit 1; }

sed -i \
  -e "s#releases/download/v[0-9.]*/Dropz-[0-9.]*-arm64\.dmg#releases/download/$tag/$dmg#g" \
  -e "s#data-release=\"v[0-9.]*\">v[0-9.]*<#data-release=\"$tag\">$tag<#" \
  "$html"

echo "index.html now points at $tag ($dmg)"
