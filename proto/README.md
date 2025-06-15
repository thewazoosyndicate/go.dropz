# Dropz Protocol Buffers

[![Protocol Buffers](https://img.shields.io/badge/protobuf-3.15%2B-blue.svg)](https://developers.google.com/protocol-buffers)
[![gRPC](https://img.shields.io/badge/gRPC-1.40%2B-green.svg)](https://grpc.io/)

This directory contains the Protocol Buffer definitions that serve as the API contract between the Dropz backend service and frontend applications. The files are organized by domain for better maintainability and clear separation of concerns.

## 📁 File Organization

### Domain-Specific Schemas

| File | Domain | Description |
|------|---------|-------------|
| `common.proto` | Core Types | Shared enums, status codes, and base types used across all domains |
| `gopro.proto` | Camera Management | Camera models, device lists, discovery, and management operations |  
| `video.proto` | Media Operations | Video file metadata, download status, and media-related operations |
| `logs.proto` | System Logging | Log entries, severity levels, and logging configuration |
| `config.proto` | Configuration | Application settings, user preferences, and system configuration |
| `service.proto` | Main API | Primary service definition that combines all domains into a unified API |

### Schema Dependencies

```
service.proto (Main API)
    ├── common.proto (Base types)
    ├── gopro.proto (Camera operations)
    ├── video.proto (Media operations)  
    ├── logs.proto (Logging)
    └── config.proto (Configuration)
```

## 🔄 Core Concepts and Workflow

### Camera Lifecycle Management

Dropz manages GoPro cameras through a sophisticated state machine with three primary lists representing different stages in the camera lifecycle:

#### 1. **Discovered Cameras** (`DiscoveredCamera`)
- **Purpose**: Registry of all reachable GoPro cameras detected by the system
- **Status**: `STATUS_DISCOVERED`
- **Characteristics**: 
  - Entry point for all cameras into the system
  - Contains basic device information (MAC address, signal strength, etc.)
  - Automatically populated by the discovery scanner
  - No user management or pairing has occurred

#### 2. **Camera Pool** (`ManagedCamera`)  
- **Purpose**: All cameras that are actively managed and paired by the user
- **Status Range**: `STATUS_MANAGED`, `STATUS_PAIRING`, `STATUS_PAIRED_REACHABLE`, `STATUS_PAIRED_UNREACHABLE`
- **Characteristics**:
  - Represents cameras the system is responsible for managing
  - Includes both reachable and temporarily unreachable devices
  - Maintains pairing information and connection history
  - Persisted across application restarts

#### 3. **Sync Queue** (`SyncQueueEntry`)
- **Purpose**: Queue of cameras waiting for video synchronization
- **Status Range**: `STATUS_SYNC_PENDING`, `STATUS_SYNCING`
- **Characteristics**:
  - Subset of the Camera Pool
  - Prioritized queue for video download operations
  - Tracks sync progress and retry attempts
  - Automatically managed based on camera availability

### Camera Status State Machine

```
┌─────────────────┐    Add to Pool    ┌─────────────────┐
│ STATUS_DISCOVERED│ ──────────────────►│ STATUS_MANAGED  │
└─────────────────┘                    └─────────────────┘
                                                │
                                                │ Start Pairing
                                                ▼
                                       ┌─────────────────┐
                                       │ STATUS_PAIRING  │
                                       └─────────────────┘
                                                │
                                ┌───────────────┼───────────────┐
                         Success│               │               │Failure
                                ▼               ▼               ▼
                ┌─────────────────────┐     ┌─────────────────┐   ┌─────────────────┐
                │STATUS_PAIRED_       │     │ STATUS_ERROR    │   │ Retry Pairing   │
                │     REACHABLE       │     └─────────────────┘   └─────────────────┘
                └─────────────────────┘              │                     │
                           │                         │                     │
           Connection Lost │                         │ User Action         │
                           ▼                         ▼                     │
                ┌─────────────────────┐     ┌─────────────────┐            │
                │STATUS_PAIRED_       │     │ Remove/Reset    │◄───────────┘
                │   UNREACHABLE       │     └─────────────────┘
                └─────────────────────┘              
                           │                         
            Reconnected    │                         
                           ▼                         
                ┌─────────────────────┐              
                │ STATUS_SYNC_PENDING │              
                └─────────────────────┘              
                           │                         
                Start Sync │                         
                           ▼                         
                ┌─────────────────────┐              
                │ STATUS_SYNCING      │              
                └─────────────────────┘
```

### Operational Workflows

#### Device Discovery Flow
```
BLE Scanner → Discovered Camera → User Selection → Camera Pool → Pairing Process
```

#### Video Synchronization Flow  
```
Camera Pool → Sync Queue → Download Coordination → Storage Management → Completion
```

#### Error Recovery Flow
```
Error Detection → Status Update → Retry Logic → User Notification → Recovery Action
```

### 🎯 Service Operations

The Dropz API provides comprehensive operations across multiple domains:

#### Camera Management Operations
```protobuf
// Device Discovery
rpc GetDiscoveredCameras(GetDiscoveredCamerasRequest) returns (GetDiscoveredCamerasResponse);
rpc WatchDiscoveredCameras(WatchDiscoveredCamerasRequest) returns (stream WatchDiscoveredCamerasResponse);

// Camera Pool Management  
rpc GetManagedCameras(GetManagedCamerasRequest) returns (GetManagedCamerasResponse);
rpc AddCameraToPool(AddCameraToPoolRequest) returns (AddCameraToPoolResponse);
rpc RemoveCameraFromPool(RemoveCameraFromPoolRequest) returns (RemoveCameraFromPoolResponse);

// Pairing Operations
rpc PairCamera(PairCameraRequest) returns (PairCameraResponse);
rpc UnpairCamera(UnpairCameraRequest) returns (UnpairCameraResponse);
```

#### Video Synchronization Operations
```protobuf
// Sync Queue Management
rpc GetSyncQueue(GetSyncQueueRequest) returns (GetSyncQueueResponse);
rpc WatchSyncQueue(WatchSyncQueueRequest) returns (stream WatchSyncQueueResponse);
rpc AddCameraToSyncQueue(AddCameraToSyncQueueRequest) returns (AddCameraToSyncQueueResponse);

// Video Operations
rpc GetVideoFiles(GetVideoFilesRequest) returns (GetVideoFilesResponse);
rpc DownloadVideo(DownloadVideoRequest) returns (stream DownloadVideoResponse);
```

#### Configuration & System Operations
```protobuf
// Configuration Management
rpc GetConfig(GetConfigRequest) returns (GetConfigResponse);
rpc UpdateConfig(UpdateConfigRequest) returns (UpdateConfigResponse);

// System Monitoring
rpc GetSystemStatus(GetSystemStatusRequest) returns (GetSystemStatusResponse);
rpc WatchLogs(WatchLogsRequest) returns (stream WatchLogsResponse);
```

#### Group Management Operations
```protobuf
// Camera Grouping
rpc CreateGroup(CreateGroupRequest) returns (CreateGroupResponse);
rpc GetGroups(GetGroupsRequest) returns (GetGroupsResponse);
rpc AddCameraToGroup(AddCameraToGroupRequest) returns (AddCameraToGroupResponse);
rpc SyncGroup(SyncGroupRequest) returns (SyncGroupResponse);
```
- Content synchronization
- Group management
- Video access
- Configuration and settings
- Logging

## Data Structure Overview

## 📊 Data Structure Overview

### Core Message Types

#### Camera Types
```protobuf
// Base camera information
message Camera {
  string id = 1;                    // Unique camera identifier (MAC address)
  string name = 2;                  // User-friendly camera name
  CameraMetadata metadata = 3;      // Technical specifications
  google.protobuf.Timestamp last_seen = 4;  // Last successful connection
}

// Technical camera details
message CameraMetadata {
  string model = 1;                 // GoPro model (Hero11, Hero12, etc.)
  string firmware_version = 2;      // Firmware version string
  string serial_number = 3;         // Device serial number
  int32 battery_level = 4;          // Battery percentage (0-100)
  string wifi_ssid = 5;            // WiFi network name
  string wifi_password = 6;        // WiFi network password
}

// Discovery-specific camera information
message DiscoveredCamera {
  Camera camera = 1;               // Base camera information
  int32 signal_strength = 2;       // BLE signal strength (RSSI)
  google.protobuf.Timestamp discovered_at = 3;  // Discovery timestamp
  CameraStatus status = 4;         // Current camera status
}

// Management-specific camera information  
message ManagedCamera {
  Camera camera = 1;               // Base camera information
  google.protobuf.Timestamp added_at = 2;       // When added to pool
  google.protobuf.Timestamp last_sync = 3;      // Last successful sync
  CameraStatus status = 4;         // Current management status
  repeated string group_ids = 5;   // Associated group memberships
}
```

#### Video and Media Types
```protobuf
// Video file metadata
message VideoFile {
  string id = 1;                   // Unique file identifier
  string camera_id = 2;            // Source camera ID
  string filename = 3;             // Original filename on camera
  int64 size_bytes = 4;           // File size in bytes
  google.protobuf.Timestamp created_at = 5;     // Creation timestamp
  google.protobuf.Timestamp downloaded_at = 6;  // Download timestamp
  string local_path = 7;          // Local storage path
  VideoMetadata metadata = 8;      // Video-specific metadata
}

// Video technical details
message VideoMetadata {
  int32 duration_seconds = 1;      // Video duration
  string resolution = 2;           // Video resolution (1920x1080, etc.)
  int32 framerate = 3;            // Frames per second
  string codec = 4;               // Video codec (H.264, H.265, etc.)
  int64 bitrate = 5;              // Video bitrate
}
```

#### Configuration Types
```protobuf
// System configuration
message Config {
  ScannerConfig scanner = 1;       // Device discovery settings
  SyncConfig sync = 2;            // Video synchronization settings
  StorageConfig storage = 3;       // Storage management settings
  LoggingConfig logging = 4;       // Logging configuration
  ServerConfig server = 5;         // gRPC server settings
}

// Scanner-specific configuration
message ScannerConfig {
  google.protobuf.Duration interval = 1;        // Scan frequency
  google.protobuf.Duration timeout = 2;         // Connection timeout
  bool auto_add_discovered = 3;    // Automatically add new cameras
}
```

#### Logging Types
```protobuf
// Log entry structure
message LogEntry {
  google.protobuf.Timestamp timestamp = 1;      // Log timestamp
  LogLevel level = 2;             // Severity level
  string component = 3;           // Source component
  string message = 4;             // Log message
  string camera_id = 5;           // Related camera (optional)
  map<string, string> metadata = 6;  // Additional context
}

// Log severity levels
enum LogLevel {
  LOG_LEVEL_UNSPECIFIED = 0;
  LOG_LEVEL_DEBUG = 1;
  LOG_LEVEL_INFO = 2;
  LOG_LEVEL_WARN = 3;
  LOG_LEVEL_ERROR = 4;
}
```

### Real-time Update Streams

The protocol supports efficient real-time updates through streaming RPCs:

#### Camera Discovery Updates
```protobuf
message WatchDiscoveredCamerasResponse {
  repeated DiscoveredCamera cameras = 1;    // Current camera list
  int64 change_counter = 2;                // Version for change detection
  google.protobuf.Timestamp timestamp = 3; // Update timestamp
}
```

#### Sync Progress Updates
```protobuf
message WatchSyncQueueResponse {
  repeated SyncQueueEntry entries = 1;     // Current sync queue
  SyncProgress progress = 2;               // Overall progress
  int64 change_counter = 3;               // Version tracking
}

message SyncProgress {
  int32 total_cameras = 1;                // Total cameras in queue
  int32 completed_cameras = 2;            // Successfully synced cameras
  int32 failed_cameras = 3;               // Failed sync attempts
  int64 total_bytes = 4;                  // Total data to download
  int64 downloaded_bytes = 5;             // Data downloaded so far
}
```

### Implementation Patterns

#### Change Detection
All list-based responses include a `change_counter` field for efficient change detection:
```protobuf
message GetManagedCamerasResponse {
  repeated ManagedCamera cameras = 1;
  int64 change_counter = 2;                // Increment on any list change
}
```

#### Error Handling
Consistent error reporting across all operations:
```protobuf
message AddCameraToPoolResponse {
  bool success = 1;
  string error_message = 2;               // Human-readable error
  ErrorCode error_code = 3;               // Machine-readable error code
}
```

#### Pagination Support
Large datasets support pagination for performance:
```protobuf
message GetVideoFilesRequest {
  string camera_id = 1;
  int32 page_size = 2;                    // Number of results per page
  string page_token = 3;                  // Pagination token
}
```

## 🔧 Implementation Notes

### Design Principles
- **Type Safety**: All timestamps use `google.protobuf.Timestamp` for standardization
- **Efficiency**: Change counters enable efficient change detection without full data transfers
- **Consistency**: Camera IDs serve as consistent references across all message types
- **Scalability**: Streaming RPCs provide real-time updates without polling overhead
- **Reliability**: Comprehensive error codes and retry mechanisms for robust communication

### Best Practices
- Always include change counters in list responses for client-side caching
- Use streaming RPCs for real-time data that changes frequently
- Implement proper error handling with both human and machine-readable error information
- Design messages with forward compatibility in mind
- Use consistent naming conventions across all proto files

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
