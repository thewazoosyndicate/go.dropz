# Dropz Electron Frontend

This is the Electron frontend for the Dropz application. It provides a graphical user interface for managing the Dropz Go service.

## Features

- Run and manage the Dropz Go service
- Monitor the service status
- View logs in real-time
- Manage discovered GoPro devices
- Restart the service when needed
- Complete offline functionality with no internet connection required

## Dependencies

The frontend includes all required dependencies locally:

- **Font Awesome 6.4.0** - Local copy for icons (located in `src/libs/fontawesome`)
- **Electron** - For cross-platform desktop application
- **gRPC-Web** - For communication with the Go backend

## Development

### Prerequisites

- Node.js (v14+)
- npm (v6+)
- Go (1.16+)

### Setup

1. Build the Go binary:

   ```bash
   # From the project root
   ./bin/build.sh --install-deps
   ```

   This will:

   - Build the Go binary and place it in the `bin` directory
   - Install the npm dependencies for the Electron app

2. Start the Electron app:

   ```bash
   # From the project root
   cd frontend
   npm start
   ```

3. Start the Electron app in development mode (rebuilds Go binary):

   ```bash
   # From the frontend directory
   npm run dev
   ```

   This will:
   - Navigate to the project root (`cd ..`)
   - Build the Go binary (`bin/build.sh`)
   - Navigate back to the frontend directory (`cd frontend`)
   - Start the Electron application (`npm start`)

   There is also `npm run dev-with-deps` which additionally installs npm dependencies for the Electron app before starting.

### Project Structure

```text
frontend/
├── src/                    # Source files
│   ├── main.js             # Main Electron process: Handles window creation, application lifecycle, and communication with the Go backend.
│   ├── index.html          # UI HTML: The main HTML structure for the application's user interface.
│   ├── renderer.js         # Renderer process: Manages the UI logic, event handling, and interacts with the main process.
│   ├── preload.js          # Preload script: Safely exposes Node.js and Electron APIs to the renderer process.
│   ├── styles.css          # UI styles: Contains all CSS rules for styling the application's appearance.
│   ├── grpc-utils.js       # gRPC client utilities: Provides helper functions for initializing and managing gRPC clients to communicate with the Go backend.
│   ├── fileLogger.js       # File logging utilities: Implements logging functionality to write application logs to files.
│   ├── icons/              # Icon assets: Contains SVG and other image files used for icons within the application.
│   ├── imgs/               # Image assets: Stores images and logos used in the UI.
│   └── proto/              # Generated Protocol Buffer files: Contains JavaScript files generated from .proto definitions for gRPC messages and services.
│       ├── *_pb.js         # Protocol Buffer message definitions
│       └── *_grpc_pb.js    # gRPC service definitions
├── package.json            # npm configuration
└── README.md               # This file
```

### gRPC Communication

The frontend uses standard gRPC with the @grpc/grpc-js library to communicate with the Go backend:

- Generated JavaScript stubs are located in the `src/proto/` directory
- `grpc-utils.js` provides client initialization and management functionality
- The communication is direct without any proxy server or gRPC-web adapters
- Service and message classes can be imported directly from the generated files

## Building for Production

To build a distributable package:

```bash
# From the frontend directory
npm run build
```

This will:

1. Build the Go binary
2. Package the Electron app with the Go binary
3. Create distributable packages in the `frontend/dist` directory

## Building for Distribution

The application can be built for distribution on multiple platforms:

### Build Process

1. All platform-specific icons are generated from the source SVG file in `resources/svg/icon.svg`
2. The build process creates:
   - AppImage for Linux
   - ICNS for macOS
   - ICO for Windows

### Icon Structure

- `resources/icon.png` - Main icon (512x512)
- `resources/icon-square.png` - Square icon for Linux (512x512)
- `resources/icon.ico` - Windows icon file
- `resources/icon.icns` - macOS icon file
- `resources/icons/hicolor/` - Linux hicolor theme directory structure
- `resources/icons.iconset/` - macOS iconset directory
