#!/bin/bash

# Exit on error
set -e

echo "Testing Dropz Electron Build..."

# Check if AppImage exists
if [ -f "dist/Dropz-1.0.0.AppImage" ]; then
  echo "AppImage found: dist/Dropz-1.0.0.AppImage"
  echo "Making AppImage executable..."
  chmod +x dist/Dropz-1.0.0.AppImage
  
  echo "You can run the AppImage with: ./dist/Dropz-1.0.0.AppImage"
  echo "Build test completed successfully."
else
  echo "Error: AppImage not found. Build may have failed."
  exit 1
fi
