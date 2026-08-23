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
Logs drawer (UI)          Live view of backend lines, in memory only
```

The backend emits structured lines with `component`, `camera`, and `err` attributes.
Electron spawns it with `--log-format=json` and parses each line for the drawer.
Verbosity has one knob: the `log_level` setting (drawer dropdown or gRPC), applied live.
Use `trace` to see scan cycles and download resume detail.

## License

Apache 2.0 — see [LICENSE](LICENSE).
