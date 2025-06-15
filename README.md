# Dropz - GoPro Fleet Management System

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/dropz/dropz)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-%3E%3D1.20-blue.svg)](https://golang.org/)
[![Electron](https://img.shields.io/badge/electron-%5E25.0.0-blue.svg)](https://electronjs.org/)

Dropz is a comprehensive fleet management system for GoPro cameras, designed to handle hundreds of devices simultaneously. It provides automated discovery, pairing, synchronization, and video download capabilities for professional GoPro deployments.

## 🚀 Key Features

### Camera Management
- **Automated Discovery**: Continuously scan for available GoPro cameras via Bluetooth Low Energy (BLE)
- **Bulk Operations**: Manage hundreds of GoPro cameras simultaneously
- **Smart Pairing**: Automated pairing process with resilient error handling
- **Status Monitoring**: Real-time status tracking for all connected devices

### Video Operations  
- **Automated Downloads**: Download videos automatically with configurable scheduling
- **Smart Sync**: Intelligent synchronization to avoid duplicate downloads
- **Date/Time Management**: Automatically set correct date and time on all cameras
- **Storage Management**: Configurable storage locations and cleanup policies

### System Reliability
- **Fault Tolerance**: Resilient to disconnections, timeouts, and network failures
- **Offline Operation**: Complete functionality without internet dependency
- **Error Recovery**: Automatic retry mechanisms for failed operations
- **Comprehensive Logging**: Detailed logging for troubleshooting and monitoring

### User Interface
- **Modern GUI**: Intuitive Electron-based desktop application
- **Real-time Updates**: Live status updates and progress monitoring
- **Configuration Management**: Easy-to-use settings and preferences
- **Cross-platform**: Built for Linux systems with Windows/macOS compatibility

## 🏗️ Architecture

Dropz employs a modern microservices architecture with clear separation of concerns:

### Components

1. **Go Backend Service** (`cmd/dropz/`)
   - Core BLE and WiFi management
   - Device discovery and pairing logic
   - Video download orchestration
   - gRPC API server
   - Configuration and logging systems

2. **Electron Frontend Application** (`frontend/`)
   - Cross-platform desktop GUI
   - Real-time device monitoring
   - User configuration interface
   - Log visualization and management

3. **Protocol Buffer Interface** (`proto/`)
   - Type-safe communication contracts
   - Auto-generated client/server code
   - Versioned API specifications

### Communication Architecture

```
┌─────────────────┐    gRPC/Protocol Buffers    ┌──────────────────┐
│  Electron GUI   │ ◄─────────────────────────► │   Go Backend     │
│                 │                              │                  │
│ • User Interface│                              │ • BLE Management │
│ • Configuration │                              │ • WiFi Control   │
│ • Log Viewer    │                              │ • Video Downloads│
│ • Status Monitor│                              │ • Device Pairing │
└─────────────────┘                              └──────────────────┘
                                                           │
                                                           ▼
                                                  ┌──────────────────┐
                                                  │   GoPro Devices  │
                                                  │                  │
                                                  │ • Camera Pool    │
                                                  │ • Video Storage  │
                                                  │ • Device Status  │
                                                  └──────────────────┘
```

### Technology Stack

**Backend:**
- **Go 1.20+**: High-performance, concurrent backend service
- **gRPC**: Fast, type-safe inter-service communication  
- **Protocol Buffers**: Efficient serialization and API contracts
- **TinyGo Bluetooth**: Low-level BLE device communication
- **GoNetworkManager**: WiFi connection management

**Frontend:**
- **Electron 25+**: Cross-platform desktop application framework
- **@grpc/grpc-js**: Native gRPC client for Node.js
- **Font Awesome 6.4**: Modern icon library (bundled locally)
- **HTML/CSS/JavaScript**: Standard web technologies for UI

**Communication:**
- **Standard gRPC**: Direct communication without proxy requirements
- **Protocol Buffers**: Strongly-typed message definitions
- **Streaming RPCs**: Real-time updates for device status and logs

## 📋 Requirements

### System Requirements
- **Operating System**: Linux (tested on Fedora 41, Ubuntu 20.04+)
- **Hardware**: Bluetooth Low Energy (BLE) capable adapter
- **Network**: WiFi adapter with NetworkManager support
- **Storage**: Adequate space for video downloads (recommend 100GB+ available)

### Software Dependencies
- **Go**: Version 1.20 or higher
- **Node.js**: Version 16+ (for frontend development)
- **npm**: Version 8+ (comes with Node.js)
- **Protocol Buffers**: Compiler (protoc) version 3.15+
- **System Libraries**:
  - `libbluetooth-dev` (Bluetooth development headers)
  - `network-manager` (WiFi management)
  - `systemd` (for service management)

### Development Tools (Optional)
- **Git**: For version control and updates
- **Make**: For using provided Makefile commands
- **VS Code**: Recommended IDE with Go and JavaScript extensions

## 🔧 Installation & Setup

### Quick Start

1. **Clone the Repository**
   ```bash
   git clone https://github.com/dropz/dropz.git
   cd dropz
   ```

2. **Install System Dependencies**
   
   **Fedora/RHEL/CentOS:**
   ```bash
   sudo dnf install golang protobuf-compiler nodejs npm bluetooth
   sudo dnf install libbluetooth-dev network-manager-dev
   ```
   
   **Ubuntu/Debian:**
   ```bash
   sudo apt update
   sudo apt install golang-go protobuf-compiler nodejs npm bluetooth
   sudo apt install libbluetooth-dev network-manager-dev
   ```

3. **Install Go Protocol Buffer Plugins**
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

4. **Build the Complete Application**
   ```bash
   # Generate Protocol Buffers and build backend
   make all
   
   # Install frontend dependencies and build
   cd frontend
   npm install
   ```

### Alternative Build Methods

#### Using the Build Script
```bash
# Build backend only
./bin/build.sh

# Build backend and install frontend dependencies
./bin/build.sh --install-deps
```

#### Manual Build Steps
```bash
# 1. Generate Protocol Buffer code
make proto

# 2. Build the Go backend
make build

# 3. Install and build frontend
cd frontend
npm install
npm run build  # For production builds
```

### Development Setup

For active development with automatic rebuilds:

```bash
# Terminal 1: Start the backend service
./bin/dropz

# Terminal 2: Start the frontend in development mode
cd frontend
npm run dev  # Rebuilds backend and starts frontend
```

## 🚀 Usage

### Starting the Application

#### Method 1: Integrated Start (Recommended)
```bash
# From the frontend directory - starts both backend and frontend
cd frontend
npm start
```

#### Method 2: Separate Processes
```bash
# Terminal 1: Start the backend service
./bin/dropz

# Terminal 2: Start the frontend application  
cd frontend
npm start
```

### Command Line Options

The backend service supports various configuration options:

```bash
./bin/dropz [OPTIONS]
```

**Available Options:**
- `--data-dir <path>`: Data storage directory (default: `./data`)
- `--video-dir <path>`: Video download directory (default: `./videos`) 
- `--log-dir <path>`: Log file directory (default: `./logs`)
- `--log-level <level>`: Logging verbosity - `debug`, `info`, `warn`, `error` (default: `info`)
- `--server-addr <address>`: gRPC server binding address (default: `127.0.0.1:50051`)
- `--config <path>`: Custom configuration file path
- `--version`: Display version information and exit
- `--help`: Display help message and exit

**Example Usage:**
```bash
# Start with custom directories and debug logging
./bin/dropz --video-dir /mnt/storage/gopro-videos \
           --log-level debug \
           --data-dir /var/lib/dropz

# Start with custom server address for remote access
./bin/dropz --server-addr 0.0.0.0:50051
```

### Using the GUI

1. **Device Discovery**: Cameras appear automatically in the "Discovered Devices" panel
2. **Device Management**: Add cameras to your managed pool by clicking "Add to Pool"
3. **Pairing**: New cameras will automatically begin the pairing process
4. **Synchronization**: Use the sync queue to download videos from paired cameras
5. **Configuration**: Access settings through the gear icon to customize behavior
6. **Monitoring**: View real-time logs and device status in the respective panels

### Testing & Validation

#### gRPC Connectivity Test
Verify communication between frontend and backend:

```bash
# Start the backend service
./bin/dropz

# In another terminal, test the gRPC connection
cd frontend
./test-standard-grpc.sh
```

#### Build Validation
Test that the application builds correctly:

```bash
# Test backend build
make test

# Test frontend build  
cd frontend
./test-build.sh
```

#### Development Workflow
```bash
# Development mode with auto-rebuild
cd frontend
npm run dev                # Rebuilds backend + starts frontend
npm run dev-with-deps     # Same as above + installs dependencies
```

## ⚙️ Configuration

Dropz provides flexible configuration options through multiple methods:

### Configuration Methods

1. **GUI Configuration** (Recommended)
   - Access settings through the frontend application
   - Real-time configuration updates
   - Validation and error checking
   - Persistent storage of preferences

2. **Configuration File**
   ```bash
   # Default location: ./data/config.json
   # Custom location: ./bin/dropz --config /path/to/config.json
   ```

3. **Command Line Arguments**
   - Override specific settings at runtime
   - Useful for deployment and automation

### Key Configuration Options

| Setting | Description | Default | Range/Options |
|---------|-------------|---------|---------------|
| **Scan Interval** | How often to scan for GoPro devices | 30s | 10s - 300s |
| **Connect Timeout** | Maximum time to wait for device connection | 30s | 10s - 120s |
| **Video Age Filter** | Download videos from last N days | 7 days | 1 - 365 days |
| **Time Sync** | Automatically set date/time on cameras | enabled | enabled/disabled |
| **Inactivity Timeout** | Reconnect to inactive devices after | 300s | 60s - 3600s |
| **Log Level** | Verbosity of log output | info | debug/info/warn/error |
| **Max Concurrent Downloads** | Simultaneous video downloads | 3 | 1 - 10 |
| **Storage Cleanup** | Auto-delete old videos | disabled | enabled/disabled |

### Example Configuration File

```json
{
  "scanner": {
    "interval": "30s",
    "timeout": "30s"
  },
  "sync": {
    "video_age_days": 7,
    "max_concurrent": 3,
    "enable_time_sync": true
  },
  "storage": {
    "video_dir": "/mnt/storage/gopro-videos",
    "data_dir": "/var/lib/dropz",
    "cleanup_enabled": false,
    "cleanup_age_days": 30
  },
  "logging": {
    "level": "info",
    "log_dir": "/var/log/dropz"
  },
  "server": {
    "address": "127.0.0.1:50051"
  }
}
```

## 🔧 Troubleshooting

### Common Issues and Solutions

#### Bluetooth/BLE Issues

**Problem**: Bluetooth adapter not detected or permission denied
```bash
# Check Bluetooth status
sudo systemctl status bluetooth
sudo bluetoothctl show

# Fix permissions
sudo usermod -a -G bluetooth $USER
# Log out and back in for group changes to take effect
```

**Problem**: Cannot discover GoPro devices
```bash
# Reset Bluetooth adapter
sudo systemctl restart bluetooth
sudo rfkill unblock bluetooth

# Check for interference
sudo hcitool lescan  # Should show discoverable BLE devices
```

**Problem**: Pairing fails repeatedly
- Ensure GoPro is in pairing mode (usually indicated by blinking blue light)
- Clear any existing pairings on the GoPro
- Try resetting the GoPro's connection settings
- Check logs for specific error messages: `tail -f logs/dropz.log`

#### WiFi Connection Issues

**Problem**: Cannot connect to GoPro WiFi
```bash
# Check NetworkManager status
sudo systemctl status NetworkManager
nmcli dev status

# Verify WiFi adapter capabilities
nmcli dev wifi list
```

**Problem**: WiFi connection drops frequently
- Check signal strength and interference
- Verify GoPro WiFi credentials in logs
- Ensure system has permission to manage WiFi connections
- Try manually connecting to GoPro WiFi first

#### Download and Sync Issues

**Problem**: Videos not downloading
- Check available storage space: `df -h`
- Verify GoPro has media to download
- Check network connectivity between devices
- Review sync queue status in the GUI

**Problem**: Partial or corrupted downloads
- Check storage device health: `sudo smartctl -a /dev/sdX`
- Verify sufficient bandwidth for multiple concurrent downloads
- Reduce max concurrent downloads in configuration

#### Performance Issues

**Problem**: High CPU or memory usage
```bash
# Monitor resource usage
top -p $(pgrep dropz)
htop

# Check for memory leaks
sudo valgrind --tool=memcheck ./bin/dropz
```

**Problem**: Slow device discovery
- Reduce scan interval in configuration
- Check for Bluetooth interference from other devices
- Verify Bluetooth adapter performance

#### Application Issues

**Problem**: Frontend cannot connect to backend
```bash
# Check if backend is running
ps aux | grep dropz

# Test gRPC connectivity
cd frontend && ./test-standard-grpc.sh

# Check port availability
sudo netstat -tlnp | grep 50051
```

**Problem**: Configuration changes not saving
- Check file permissions on data directory
- Verify sufficient disk space
- Review logs for write permission errors

### Debugging Steps

1. **Enable Debug Logging**
   ```bash
   ./bin/dropz --log-level debug
   ```

2. **Check System Resources**
   ```bash
   # Disk space
   df -h
   
   # Memory usage
   free -h
   
   # Bluetooth status
   sudo systemctl status bluetooth
   
   # NetworkManager status
   sudo systemctl status NetworkManager
   ```

3. **Review Log Files**
   ```bash
   # Application logs
   tail -f logs/dropz.log
   
   # System Bluetooth logs
   sudo journalctl -u bluetooth -f
   
   # NetworkManager logs
   sudo journalctl -u NetworkManager -f
   ```

4. **Test Components Individually**
   ```bash
   # Test BLE scanning
   sudo hcitool lescan
   
   # Test WiFi capabilities
   nmcli dev wifi list
   
   # Test gRPC communication
   cd frontend && ./test-standard-grpc.sh
   ```

### Getting Help

If you encounter issues not covered here:

1. **Check the Logs**: Enable debug logging and review the output
2. **GitHub Issues**: Search existing issues or create a new one
3. **System Requirements**: Verify all dependencies are properly installed
4. **Hardware Compatibility**: Ensure your Bluetooth adapter supports BLE

**When Reporting Issues:**
- Include relevant log excerpts
- Specify your Linux distribution and version
- List your hardware specifications (Bluetooth adapter, WiFi adapter)
- Describe the steps to reproduce the problem

## 📁 Project Structure

```
dropz/
├── cmd/dropz/                  # Application entry point
│   ├── main.go                # Main application with CLI setup
│   └── cli/                   # Command-line interface definitions
├── pkg/                       # Core application packages
│   ├── ble/                   # Bluetooth Low Energy management
│   ├── database/              # Data persistence layer
│   ├── logger/                # Logging utilities and configuration
│   ├── manager/               # Core business logic
│   │   ├── config/           # Configuration management
│   │   ├── discovery/        # Device discovery and scanning
│   │   ├── groups/           # Camera grouping functionality
│   │   ├── pairing/          # Device pairing orchestration
│   │   └── sync/             # Video synchronization logic
│   ├── protocol/             # Generated Protocol Buffer code
│   ├── queue/                # Task queue management
│   ├── server/               # gRPC server implementation
│   └── wifi/                 # WiFi connection management
├── proto/                    # Protocol Buffer definitions
│   ├── common.proto          # Shared types and enums
│   ├── gopro.proto           # Camera-related messages
│   ├── video.proto           # Video file operations
│   ├── logs.proto            # Logging messages
│   ├── config.proto          # Configuration messages
│   └── service.proto         # Main service definitions
├── frontend/                 # Electron desktop application
│   ├── src/                  # Frontend source code
│   │   ├── main.js           # Electron main process
│   │   ├── renderer.js       # UI logic and event handling
│   │   ├── preload.js        # Secure context bridge
│   │   ├── grpc-utils.js     # gRPC client utilities
│   │   ├── index.html        # Main UI structure
│   │   ├── styles.css        # Application styling
│   │   ├── proto/            # Generated JavaScript gRPC code
│   │   ├── icons/            # UI icons and graphics
│   │   └── libs/             # Bundled third-party libraries
│   ├── resources/            # Application assets and build resources
│   └── package.json          # Node.js dependencies and scripts
├── bin/                      # Built binaries and scripts
├── docs/                     # Documentation and API specifications
├── include/                  # Protocol Buffer include files
├── Makefile                  # Build automation
└── README.md                 # This file
```

### Key Directories Explained

- **`cmd/`**: Application entry points and CLI interface
- **`pkg/`**: Reusable Go packages organized by functionality
- **`proto/`**: API contract definitions using Protocol Buffers
- **`frontend/`**: Complete Electron desktop application
- **`bin/`**: Compiled binaries and build scripts
- **`docs/`**: Project documentation and API specifications

## 🏗️ Contributing

We welcome contributions to Dropz! Here's how you can help:

### Development Setup
1. Fork the repository and clone your fork
2. Follow the installation instructions above
3. Create a feature branch: `git checkout -b feature/your-feature-name`
4. Make your changes and test thoroughly
5. Submit a pull request with a clear description

### Code Style
- **Go**: Follow standard Go formatting (`go fmt`)
- **JavaScript**: Use consistent indentation and modern ES6+ syntax
- **Commits**: Use conventional commit messages
- **Documentation**: Update relevant documentation for new features

### Testing
- Run existing tests: `make test`
- Add tests for new functionality
- Test on actual GoPro hardware when possible
- Verify both backend and frontend integration

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **[GoPro Open API](https://gopro.github.io/OpenGoPro/)** - Comprehensive API specifications and documentation
- **[TinyGo Bluetooth](https://pkg.go.dev/tinygo.org/x/bluetooth)** - Excellent BLE connectivity library for Go
- **[GoNetworkManager](https://github.com/Wifx/gonetworkmanager)** - WiFi management integration for Linux
- **[Electron](https://electronjs.org/)** - Cross-platform desktop application framework
- **[Protocol Buffers](https://developers.google.com/protocol-buffers)** - Efficient serialization and API contracts
- **[Font Awesome](https://fontawesome.com/)** - Beautiful icons for the user interface

---

**Made with ❤️ for the GoPro community**

For support, feature requests, or bug reports, please visit our [GitHub Issues](https://github.com/dropz/dropz/issues) page.