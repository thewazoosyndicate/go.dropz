# Dropz Protocol Buffers

This directory contains the Protocol Buffer definitions for the Dropz application, which manages a pool of GoPro cameras. The files have been organized by domain for better maintainability.

## File Organization

- `common.proto`: Common package definitions, status enums, and shared types
- `gopro.proto`: Camera data models, lists, and related operations
- `video.proto`: Video file data models and related operations
- `logs.proto`: Logging data models and related operations
- `config.proto`: Configuration data models and settings
- `service.proto`: Main service definition that combines all domains

## Core Concepts and Workflow

### Camera Lists and Workflow

Dropz manages cameras through three distinct lists that represent different stages in the camera lifecycle:

1. **Discovered Cameras**: All reachable GoPro cameras detected by the system
   - A camera in this list has status `STATUS_DISCOVERED`
   - This is the entry point for all cameras into the system

2. **Camera Pool**: All cameras that are both managed and paired
   - Cameras here can have various statuses (`STATUS_MANAGED`, `STATUS_PAIRING`, `STATUS_PAIRED_REACHABLE`, etc.)
   - This list represents all cameras the system knows about and intends to manage
   - Cameras may be unreachable but remain in this list if they have been previously paired

3. **Sync Queue**: Cameras waiting to be synced
   - A camera in this list has status `STATUS_SYNC_PENDING` or `STATUS_SYNCING`
   - This is a subset of the Camera Pool

### Camera Status Lifecycle

Cameras progress through the following statuses:

- `STATUS_DISCOVERED`: Initial status when a camera is first detected
- `STATUS_MANAGED`: Camera has been added to the management pool
- `STATUS_PAIRING`: Camera is in the process of being paired
- `STATUS_PAIRED_REACHABLE`: Camera is paired and currently reachable
- `STATUS_PAIRED_UNREACHABLE`: Camera is paired but currently not reachable
- `STATUS_SYNC_PENDING`: Camera is in the sync queue waiting to be synced
- `STATUS_SYNCING`: Camera is actively being synced
- `STATUS_ERROR`: Camera has encountered an error

### Service Operations

The system provides operations for:

- Camera discovery and management
- Camera pairing
- Content synchronization
- Group management
- Video access
- Configuration and settings
- Logging

## Data Structure Overview

### Core Types

- **Camera**: Base information about a camera including ID, name, connectivity details
- **CameraMetadata**: Technical details about a camera (firmware, model, etc.)
- **DiscoveredCamera**: A camera that has been discovered but may not be managed
- **ManagedCamera**: A camera in the management pool
- **SyncQueueEntry**: A record in the sync queue
- **Group**: A collection of cameras that can be managed together
- **VideoFile**: Metadata about a synchronized video file
- **Config**: System-wide configuration settings
- **LogEntry**: System logs

### Real-time Updates

The protocol supports real-time updates through streaming RPCs for:
- Discovered cameras (`WatchDiscoveredCameras`)
- Managed cameras (`WatchManagedCameras`)
- Sync queue (`WatchSyncQueue`)

This allows the client to maintain up-to-date information without polling.

## Implementation Notes

- All timestamps use `google.protobuf.Timestamp` for standardization
- Change counters are used for tracking updates efficiently
- Camera IDs are used as references between different message types

## Compilation

When compiling these proto files, ensure you're in the root directory of the project so the import paths resolve correctly.

### Prerequisites

1. Install the Protocol Buffers compiler (protoc):

```bash
# For Fedora/RHEL/CentOS
sudo dnf install protobuf-compiler

# For Ubuntu/Debian
sudo apt-get install protobuf-compiler

# For macOS
brew install protobuf

# Alternatively, download from GitHub releases:
# https://github.com/protocolbuffers/protobuf/releases
```

2. Ensure you have the well-known type definitions (like `google/protobuf/timestamp.proto`):

If you see errors like `google/protobuf/timestamp.proto: File not found`, you need to include the well-known type definitions. You can:

```bash
# Option 1: Download the proto files from GitHub
mkdir -p include/google/protobuf
curl -L https://github.com/protocolbuffers/protobuf/raw/main/src/google/protobuf/timestamp.proto -o include/google/protobuf/timestamp.proto

# Option 2: On many systems, these files are installed with protoc in
# /usr/include or /usr/local/include. Add these to your include path.
```

When running protoc, add the include directory to your command:
```bash
protoc -I=proto -I=include -I=/usr/local/include -I=/usr/include ...
```

### Go Code Generation

To generate Go code from these proto files:

1. Install the required protoc plugins:

```bash
go get -u google.golang.org/protobuf/cmd/protoc-gen-go
go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

2. Generate the code:

```bash
mkdir -p pkg/protocol
protoc -I=proto -I=include -I=/usr/local/include -I=/usr/include \
    --go_out=pkg/protocol/ --go_opt=paths=source_relative \
    --go-grpc_out=pkg/protocol/ --go-grpc_opt=paths=source_relative \
    --go_opt=Mservice.proto=github.com/dropz/dropz/pkg/protocol \
    --go_opt=Mgopro.proto=github.com/dropz/dropz/pkg/protocol \
    --go_opt=Mconfig.proto=github.com/dropz/dropz/pkg/protocol \
    --go_opt=Mlogs.proto=github.com/dropz/dropz/pkg/protocol \
    --go_opt=Mvideo.proto=github.com/dropz/dropz/pkg/protocol \
    --go_opt=Mcommon.proto=github.com/dropz/dropz/pkg/protocol \
    --go-grpc_opt=Mservice.proto=github.com/dropz/dropz/pkg/protocol \
    --go-grpc_opt=Mgopro.proto=github.com/dropz/dropz/pkg/protocol \
    --go-grpc_opt=Mconfig.proto=github.com/dropz/dropz/pkg/protocol \
    --go-grpc_opt=Mlogs.proto=github.com/dropz/dropz/pkg/protocol \
    --go-grpc_opt=Mvideo.proto=github.com/dropz/dropz/pkg/protocol \
    --go-grpc_opt=Mcommon.proto=github.com/dropz/dropz/pkg/protocol \
    proto/*.proto
```

3. If needed, move generated files to the correct location:

```bash
# Move any generated files that ended up in the wrong place
if [ -f proto/*_grpc.pb.go ] || [ -f proto/*.pb.go ]; then
    mv proto/*.pb.go pkg/protocol/ 2>/dev/null || true
    mv proto/*_grpc.pb.go pkg/protocol/ 2>/dev/null || true
fi
rm -rf github.com  # Clean up any stray directories
```

### JavaScript/Node.js Code Generation

To generate JavaScript code for the frontend:

```bash
cd frontend && npx grpc_tools_node_protoc \
    --js_out=import_style=commonjs:./src/proto \
    --grpc_out=grpc_js:./src/proto \
    --proto_path=../proto \
    --proto_path=../include \
    --proto_path=/usr/local/include \
    --proto_path=/usr/include \
    ../proto/*.proto
```

### Other Languages

For other languages, refer to the Protocol Buffers documentation for your target language:
- [Protocol Buffers Documentation](https://developers.google.com/protocol-buffers/docs/tutorials)
- [gRPC Documentation](https://grpc.io/docs/)
