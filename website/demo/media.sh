#!/usr/bin/env bash
# Synthetic clips, previews, and thumbnails the fake backend hands to the UI.
# Real GoPro footage is not in the repo; moving gradients stand in for it.
# Labels sit inside the middle 4:3 of the frame so the card thumbnails keep them.
set -euo pipefail

OUT=${1:?usage: media.sh <out-dir>}
HERE=$(cd "$(dirname "$0")" && pwd)
FFMPEG=${FFMPEG:-$HERE/../../bin/ffmpeg}
# Brackets in a font path (variable fonts) break the filter parser
FONT=${FONT:-$(fc-list : file | grep -E 'LiberationSans-Regular|DejaVuSans\.ttf|Cantarell-Regular' | grep -v '\[' | head -1 | tr -d :)}
mkdir -p "$OUT"

# name  c0      c1      c2
CLIPS='
GX010042 0x1b4f72 0x5dade2 0xf5cba7
GX010043 0x0b5345 0x48c9b0 0xfdebd0
GX010044 0x4a235a 0xaf7ac5 0xf9e79f
GH010391 0x784212 0xf0b27a 0xd6eaf8
GH010392 0x1c2833 0x5d6d7e 0xf7dc6f
GX020117 0x0e6655 0x82e0aa 0xfad7a0
GX020118 0x6e2c00 0xf5b041 0xaed6f1
GX020119 0x154360 0x7fb3d5 0xf5eef8
GX010039 0x512e5f 0xc39bd3 0xfdedec
GX010040 0x145a32 0x7dcea0 0xfcf3cf
'

echo "$CLIPS" | while read -r name c0 c1 c2; do
  [ -z "$name" ] && continue
  [ -f "$OUT/$name.webm" ] && continue
  "$FFMPEG" -nostdin -hide_banner -loglevel error -y \
    -f lavfi -i "gradients=s=854x480:c0=$c0:c1=$c1:c2=$c2:nb_colors=3:speed=0.02:d=12:r=30" \
    -vf "vignette=PI/5,drawtext=fontfile=$FONT:text='$name.MP4':x=130:y=30:fontsize=28:fontcolor=white@0.85,drawtext=fontfile=$FONT:text='%{pts\:hms}':x=w-tw-130:y=h-th-30:fontsize=24:fontcolor=white@0.75" \
    -c:v libvpx-vp9 -deadline realtime -cpu-used 8 -row-mt 1 -b:v 900k -t 12 "$OUT/$name.webm"
  "$FFMPEG" -nostdin -hide_banner -loglevel error -y -ss 2 -i "$OUT/$name.webm" -frames:v 1 -vf scale=320:-1 -q:v 4 "$OUT/$name.jpg"
done

# Thumbnail of the cut the trim scene saves (in point 3 s)
[ -f "$OUT/GX010042_trim.jpg" ] || "$FFMPEG" -nostdin -hide_banner -loglevel error -y -ss 3 -i "$OUT/GX010042.webm" -frames:v 1 -vf scale=320:-1 -q:v 4 "$OUT/GX010042_trim.jpg"
echo "media ready in $OUT"
