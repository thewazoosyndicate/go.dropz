# Dropz Electron Frontend

[![Electron](https://img.shields.io/badge/electron-%5E25.0.0-blue.svg)](https://electronjs.org/)
[![Node.js](https://img.shields.io/badge/node-%3E%3D16-green.svg)](https://nodejs.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](../LICENSE)

A modern, cross-platform desktop application providing an intuitive graphical interface for managing GoPro camera fleets. Built with Electron and designed for professional-grade deployments.

## 🚀 Key Features

### Device Management
- **Real-time Discovery**: Live updates of available GoPro cameras
- **Visual Status Indicators**: Clear status displays for all managed devices
- **Bulk Operations**: Manage multiple cameras simultaneously
- **Device Grouping**: Organize cameras into logical groups for easier management

### Video Operations
- **Sync Queue Management**: Visual queue for pending video downloads
- **Progress Monitoring**: Real-time download progress and status
- **Storage Management**: Configure and monitor storage locations
- **Video Preview**: Quick preview of downloaded content

### System Integration
- **Complete Offline Mode**: No internet connection required
- **Native gRPC Communication**: Direct communication with Go backend
- **Cross-platform Support**: Runs on Linux, Windows, and macOS
- **System Notifications**: Desktop notifications for important events

### User Experience
- **Modern UI**: Clean, responsive interface built with modern web technologies
- **Dark/Light Themes**: Adaptive theming for user preference
- **Keyboard Shortcuts**: Efficient keyboard navigation and shortcuts
- **Comprehensive Logging**: Real-time log viewer with filtering capabilities

## 📦 Dependencies & Architecture

### Technology Stack

**Core Framework:**
- **Electron 25+**: Cross-platform desktop application framework
- **Node.js 16+**: JavaScript runtime environment
- **HTML5/CSS3/ES6+**: Modern web technologies for UI

**Communication Layer:**
- **@grpc/grpc-js 1.13+**: Native gRPC client for Node.js
- **@grpc/proto-loader**: Protocol Buffer loader for JavaScript
- **google-protobuf**: Google's Protocol Buffer runtime for JavaScript

**UI & Assets:**
- **Font Awesome 6.4.0**: Complete icon library (bundled locally for offline use)
- **Custom CSS**: Responsive and modern styling
- **SVG Graphics**: Scalable vector graphics for high-quality icons

**Development Tools:**
- **grpc-tools**: Protocol Buffer compilation tools for JavaScript
- **electron-builder**: Application packaging and distribution
- **electron-log**: Enhanced logging capabilities

### Local Dependencies

All dependencies are bundled locally to ensure complete offline functionality:

- **Font Awesome 6.4.0**: Located in `src/libs/fontawesome/`
  - Complete CSS framework with all icon styles
  - Web fonts for cross-platform compatibility
  - No external CDN dependencies

### Generated Code
- **Protocol Buffer Stubs**: Auto-generated JavaScript files in `src/proto/`
  - Message definitions (`*_pb.js`)
  - Service definitions (`*_grpc_pb.js`)
  - Synchronized with backend API contracts

## 🛠️ Development

### Prerequisites

- **Node.js**: Version 16 or higher
- **npm**: Version 8 or higher (included with Node.js)
- **Go**: Version 1.20+ (for backend development)
- **Protocol Buffers**: Compiler (protoc) for generating JavaScript stubs

### Quick Start

1. **Install Dependencies**
   ```bash
   # From the frontend directory
   npm install
   ```

2. **Start Development Environment**
   ```bash
   # Option 1: Start with automatic backend rebuild
   npm run dev
   
   # Option 2: Start with dependency installation + backend rebuild
   npm run dev-with-deps
   
   # Option 3: Start frontend only (backend must be running separately)
   npm start
   ```

3. **Manual Backend Setup** (if using Option 3)
   ```bash
   # In a separate terminal, from project root
   ./bin/build.sh --install-deps
   ./bin/dropz
   ```

### Development Scripts

| Script | Description | Use Case |
|--------|-------------|----------|
| `npm start` | Start Electron app only | Frontend development with existing backend |
| `npm run dev` | Build backend + start frontend | Full-stack development |
| `npm run dev-with-deps` | Install deps + build backend + start frontend | Initial setup or dependency updates |
| `npm run build` | Build production packages | Creating distributable packages |
| `npm test` | Run test suite | Development testing |
| `npm run test-build` | Test build process | Validate build configuration |

### Development Workflow

```bash
# 1. Initial setup
cd frontend
npm install

# 2. Start development mode
npm run dev-with-deps

# 3. Make changes to frontend code
# Files auto-reload in Electron

# 4. Test changes
# Use the application to verify functionality

# 5. Build for production (optional)
npm run build
```

### 📁 Project Structure

```
frontend/
├── src/                      # Source code directory
│   ├── main.js              # Main Electron process
│   │                        #   - Application lifecycle management
│   │                        #   - Window creation and management  
│   │                        #   - Backend process coordination
│   │                        #   - Inter-process communication setup
│   │
│   ├── renderer.js          # Renderer process (UI logic)
│   │                        #   - DOM manipulation and event handling
│   │                        #   - gRPC client communication
│   │                        #   - Real-time data updates
│   │                        #   - User interaction processing
│   │
│   ├── preload.js           # Secure context bridge
│   │                        #   - Safe API exposure to renderer
│   │                        #   - Node.js/Electron API gateway
│   │                        #   - Security boundary management
│   │
│   ├── index.html           # Main application UI
│   │                        #   - HTML structure and layout
│   │                        #   - Component definitions
│   │                        #   - UI element organization
│   │
│   ├── styles.css           # Application styling
│   │                        #   - Modern CSS with flexbox/grid
│   │                        #   - Responsive design rules
│   │                        #   - Theme and color definitions
│   │
│   ├── grpc-utils.js        # gRPC client utilities
│   │                        #   - Client initialization and management
│   │                        #   - Connection handling and retry logic
│   │                        #   - Service method wrappers
│   │
│   ├── fileLogger.js        # File logging system
│   │                        #   - Log file management
│   │                        #   - Log rotation and cleanup
│   │                        #   - Structured logging utilities
│   │
│   ├── protobuf-utils.js    # Protocol Buffer utilities
│   │                        #   - Message serialization helpers
│   │                        #   - Type conversion utilities
│   │                        #   - Validation functions
│   │
│   ├── proto/               # Generated Protocol Buffer files
│   │   ├── *_pb.js         #   - Message class definitions
│   │   └── *_grpc_pb.js    #   - Service client definitions
│   │
│   ├── icons/               # UI icon assets
│   │   ├── gopro.svg       #   - GoPro camera icon
│   │   └── plus.svg        #   - Add/create action icon
│   │
│   ├── imgs/                # Image assets and graphics
│   │   ├── 3_dropz.svg     #   - Application logo variations
│   │   └── logo.svg        #   - Main brand logo
│   │
│   └── libs/                # Bundled third-party libraries
│       └── fontawesome/    #   - Local Font Awesome installation
│           ├── css/        #   - Icon CSS framework
│           └── webfonts/   #   - Icon font files
│
├── resources/               # Build and distribution assets
│   ├── icons/              #   - Platform-specific application icons
│   ├── icon.png           #   - Main application icon (512x512)
│   ├── icon.ico           #   - Windows icon format
│   ├── icon.icns          #   - macOS icon format
│   └── svg/               #   - Source SVG files for icon generation
│
├── package.json            # npm configuration and dependencies
├── test-build.sh          # Build validation script
├── test-grpc.sh           # gRPC connectivity test script  
├── test-standard-grpc.sh  # Standard gRPC test script
└── README.md              # This documentation file
```

### Architecture Overview

The frontend follows Electron's multi-process architecture:

- **Main Process**: Manages application lifecycle, creates renderer processes, handles system integration
- **Renderer Process**: Runs the web-based UI, handles user interactions, communicates with backend via gRPC
- **Preload Script**: Provides secure bridge between main and renderer processes

### Communication Flow

```
┌─────────────────┐    IPC Messages    ┌──────────────────┐
│   Main Process  │ ◄─────────────────► │ Renderer Process │
│                 │                     │                  │
│ • Lifecycle     │                     │ • UI Logic       │
│ • Window Mgmt   │                     │ • Event Handling │
│ • System APIs   │                     │ • Data Display   │
└─────────────────┘                     └──────────────────┘
         │                                       │
         │                                       │ gRPC Calls
         │ Backend Process                       │
         │ Management                            ▼
         ▼                               ┌──────────────────┐
┌─────────────────┐                     │   Go Backend     │
│  Backend Binary │                     │                  │
│                 │                     │ • Device Mgmt    │
│ • dropz process │                     │ • Video Sync     │
│ • Auto-restart  │                     │ • Configuration  │
└─────────────────┘                     └──────────────────┘
```

## 🔧 Configuration & Customization

### Application Settings

The frontend application stores its configuration in:
- **User Preferences**: Stored in Electron's `userData` directory
- **Backend Configuration**: Managed through gRPC API calls
- **UI State**: Local storage for window positions, themes, and view preferences

### Customization Options

- **Theme Selection**: Choose between light and dark themes
- **Window Layout**: Customizable panel sizes and positions  
- **Notification Settings**: Configure desktop notifications
- **Log Levels**: Adjust verbosity of displayed logs
- **Auto-updates**: Control automatic refresh intervals

### Environment Variables

```bash
# Development mode
ELECTRON_IS_DEV=1 npm start

# Debug mode with additional logging
DEBUG=dropz:* npm start

# Custom backend address
BACKEND_ADDRESS=192.168.1.100:50051 npm start
```

## 🧪 Testing & Validation

### Testing Scripts

```bash
# Test gRPC connectivity
./test-standard-grpc.sh

# Test build process
./test-build.sh

# Full integration test
npm test
```

### Manual Testing Checklist

- [ ] Application starts without errors
- [ ] gRPC connection to backend establishes
- [ ] Device discovery panel shows devices
- [ ] Configuration changes persist
- [ ] Log viewer displays backend logs
- [ ] Build process completes successfully

### Debugging

```bash
# Start with debug console open
npm start -- --enable-logging --log-level=debug

# View Electron debug information
DEBUG=electron:* npm start
```

## 🤝 Contributing

### Frontend Development Guidelines

1. **Code Style**: 
   - Use modern ES6+ JavaScript
   - Follow consistent indentation (2 spaces)
   - Use meaningful variable names
   - Add comments for complex logic

2. **UI/UX Principles**:
   - Maintain responsive design
   - Ensure accessibility compliance
   - Test across different screen sizes
   - Follow platform-specific design patterns

3. **gRPC Integration**:
   - Always handle connection errors gracefully
   - Implement proper retry logic
   - Use streaming for real-time updates
   - Validate message types and fields

### Pull Request Process

1. Fork the repository and create a feature branch
2. Make your changes with appropriate tests
3. Update documentation as needed
4. Test the build process: `npm run test-build`
5. Submit a pull request with clear description

## 📚 Additional Resources

- **[Electron Documentation](https://electronjs.org/docs)** - Complete Electron framework guide
- **[gRPC Node.js](https://grpc.io/docs/languages/node/)** - gRPC client documentation
- **[Protocol Buffers](https://developers.google.com/protocol-buffers)** - Message format documentation
- **[Font Awesome](https://fontawesome.com/docs)** - Icon library documentation

---

For backend documentation, see the [main project README](../README.md).

For Protocol Buffer specifications, see the [proto directory README](../proto/README.md).
