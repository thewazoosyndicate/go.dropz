# Dropz

Fleet management system for GoPro cameras. Handles BLE discovery, pairing, WiFi media sync, and bulk operations across large deployments.

Go backend communicates with an Electron frontend over gRPC.

## Requirements

- Linux with NetworkManager
- Bluetooth Low Energy adapter
- Go 1.24+
- Node.js
- protoc (Protocol Buffers compiler)

## Build & Run

```bash
# Generate protobuf code and build backend
make all

# Install frontend dependencies
cd frontend && npm install
```

Start the backend, then the frontend:

```bash
./bin/dropz
cd frontend && npm start
```

## Flags

```bash
-data-dir       Data directory (default: ~/.dropz/data)
-video-dir      Video download directory (default: ~/Videos)
-log-dir        Log directory (default: ~/.dropz/logs)
-log-level      trace, debug, info, warn, error (default: info)
-log-format     text or json (default: text; Electron uses json)
-server-addr    gRPC listen address (default: 127.0.0.1:50051)
-pair-mode      Enable automatic pairing (default: true)
-sync-enabled   Enable automatic sync (default: true)
-version        Show version and exit
```

## Logging

One writer per line; each log lives in exactly one file.

```
~/.dropz/logs/dropz.log   Backend (slog; rotated at 5MB, 3 backups kept)
Electron userData logs    App lifecycle only (window, spawn, restart)
Activity > Diagnostics    Live view of backend lines in the UI, in memory only
```

The backend emits structured lines with `component`, `camera`, and `err` attributes.
Electron spawns it with `--log-format=json` and parses each line for the diagnostics view.
Verbosity has one knob: the `log_level` setting (diagnostics dropdown or gRPC), applied live.
Use `trace` to see scan cycles and download resume detail.

## Groups

A group is the day's rig. Cameras belong to at most one group and stay
managed whatever the group does; the Cameras tab shows one section per
group, paused groups after active ones, ungrouped cameras last.

```
Pause auto-sync        Members stay visible, get no status checks and no
                       auto-sync; a manual Sync still runs
Use only this group    Resumes this group and pauses every other one:
                       the rig switch, without unmanaging anything
Add cameras in range   Pulls every managed camera that is switched on
                       nearby into the group
New group              From cameras in range, from all managed, or empty
```

## Sync visibility

Every sync is visible at three depths, all fed by the same queue stream:

```
Status bar     Camera on the radio, file x of y, ETA, how many wait
Camera card    One state at a time: up to date, new on camera, queued,
               syncing (phase, file, totals), failed (step, retry), away
Activity tab   Per-file rows for the running sync, then a kept history
               (SyncSession in the store, 200 most recent) with the
               outcome, files, phase timings, and a retry or library link
```

The nine backend steps collapse onto four user phases (connect, Wi-Fi,
catalog, transfer). A failed download is retried once after the rest of
the pass; a failed step is recorded with its index so the history can say
where it stopped.

## License

Apache 2.0 — see [LICENSE](LICENSE).
