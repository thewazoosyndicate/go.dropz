# Dropz - GoPro Automated Video Downloader

Dropz is a robust application for automatically managing and downloading videos from multiple GoPro cameras. It uses Bluetooth Low Energy (BLE) to discover and connect to GoPro devices, set the date and time, enable WiFi, and download videos automatically.

## Features

- Continuously scan for available GoPro cameras
- Manage discovered GoPros (pair, connect, download)
- Set date and time on GoPro cameras
- Enable WiFi and download videos
- Manage hundreds of GoPro cameras simultaneously
- Resilient to disconnections, timeouts, and failures
- Configurable scan intervals, timeouts, and more
- Cross-platform support for Linux, macOS, and Windows
- Works completely offline with no internet dependency

## Architecture

Dropz consists of two main components:

1. **Go Backend**: Handles the core functionality (device scanning, connection, download)
2. **Electron Frontend**: Provides a user interface for managing GoPro devices

The two components communicate using standard gRPC with Protocol Buffers for efficient, type-safe communication. The frontend uses @grpc/grpc-js library to connect directly to the Go backend without the need for a proxy server.

### gRPC Implementation

The application has migrated from gRPC-web to standard gRPC for improved performance and simplicity:

- **Direct Communication**: Electron frontend communicates directly with the Go backend using native gRPC
- **No Proxy Required**: Eliminates the need for an Envoy proxy or other intermediary
- **Simplified Architecture**: Reduces complexity and potential points of failure
- **Better Performance**: Native gRPC offers better performance than gRPC-web

## Requirements

- Linux, macOS, or Windows system (tested on Fedora 41, Windows 11, and macOS 14)
- Go 1.20 or higher
- Protocol Buffers compiler (protoc)
- Node.js and npm (for the Electron frontend)
- Bluetooth capabilities
- NetworkManager for WiFi connections (or the platform equivalent)

## Building

### Backend

1. Install Go and Protocol Buffers:

```bash
# Install Go (if not already installed)
sudo dnf install golang

# Install protoc
sudo dnf install protobuf-compiler

# Install Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

2. Clone the repository:

```bash
git clone https://github.com/dropz/dropz.git
cd dropz
```

3. Generate Protocol Buffer code:

```bash
make proto
```

4. Build the backend:

```bash
make build
```

To build for a specific platform, set the `GOOS` environment variable:

```bash
# Windows build
GOOS=windows go build -o bin/dropz.exe cmd/dropz/main.go

# macOS build
GOOS=darwin go build -o bin/dropz cmd/dropz/main.go
```

### Frontend

The Electron frontend code is in a separate repository. Follow the instructions there to build and run it.

## Running

To run the backend service:

```bash
./bin/dropz
```

### Command-line Options

- `--data-dir`: Directory for storing data (default: "data")
- `--video-dir`: Directory for storing downloaded videos (default: "videos")
- `--log-dir`: Directory for storing logs (default: "logs")
- `--log-level`: Log level (debug, info, warn, error) (default: "info")
- `--server-addr`: gRPC server address (default: "127.0.0.1:50051")
- `--version`: Show version and exit

### Testing gRPC Connectivity

To test the gRPC connection between the frontend and backend:

```bash
# Start the backend first
./bin/dropz

# In a separate terminal, run the test script
cd frontend
./test-standard-grpc.sh
```

This script demonstrates how to connect to the backend using the standard gRPC client.

## Configuration

The application configuration is stored in a JSON file and can be modified through the frontend UI or by editing the file directly:

- Scan interval: How often to scan for GoPro devices
- Connect timeout: Maximum time to wait for connection
- Video age: How many days of videos to download
- Time setting: Enable/disable setting date and time on GoPros
- Inactivity timeout: Time before reconnecting to inactive devices
- Log level: Verbosity of log output

## Troubleshooting

### Bluetooth Issues

- Ensure Bluetooth is enabled on your system
- Check that you have the necessary permissions to use Bluetooth
- Verify that the GoPro is in pairing mode when trying to connect for the first time

### WiFi Issues

- Make sure NetworkManager is running and properly configured
- Check the GoPro's WiFi credentials in the logs if connection fails
- Ensure your system has permission to manage WiFi connections

### Download Issues

- Verify that the GoPro has media to download
- Check storage space on your system
- Look for connection timeout errors in the logs

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- [GoPro Open API](https://gopro.github.io/OpenGoPro/) for providing the API specifications
- [TinyGo Bluetooth](https://pkg.go.dev/tinygo.org/x/bluetooth) for BLE connectivity
- [GoNetworkManager](https://github.com/Wifx/gonetworkmanager) for WiFi management