# Demo recorder

Regenerates the website video and stills from the real Electron UI.
No camera and no on-screen window: Electron renders offscreen against a
fake gRPC backend that plays the storyboard.

## Run

    cd frontend && npx vite build            # dist-svelte must exist
    WORK=/tmp/dropz-demo bash website/demo/media.sh /tmp/dropz-demo/media
    cd frontend && WORK=/tmp/dropz-demo npx electron ../website/demo/main.js
    ../bin/ffmpeg -y -framerate 24 -i /tmp/dropz-demo/frames/f_%05d.png \
      -vf format=yuv420p -c:v libx264 -preset slow -crf 27 -movflags +faststart \
      -an website/content/assets/demo.mp4

Stills land in `$WORK/stills`; copy the four `shot-*.png` and make the
poster from `final.png`. `PROBE=1` renders one frame and exits.

## Files

- `media.sh` synthetic clips, previews, thumbnails (gradients stand in for footage)
- `fake-server.js` the fake DropzService; plain state objects pushed on every watch stream
- `main.js` offscreen driver: storyboard, on-screen cursor, wall-clock frame capture
