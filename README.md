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

```
-data-dir       Data directory (default: ~/.dropz/data)
-video-dir      Video download directory (default: ~/Videos)
-log-dir        Log directory (default: ~/.dropz/logs)
-log-level      debug, info, warn, error (default: info)
-server-addr    gRPC listen address (default: 127.0.0.1:50051)
-pair-mode      Enable automatic pairing (default: true)
-sync-enabled   Enable automatic sync (default: true)
-version        Show version and exit
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
