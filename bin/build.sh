#!/bin/bash

# Exit on error
set -e

echo "Building Dropz Go binary..."

# Move to the project root directory (assuming this script is in bin/)
cd "$(dirname "$0")/.."

# Build the Go binary
go build -o bin/dropz cmd/dropz/main.go

echo "Go binary built successfully at bin/dropz"

# Check if we should also install npm dependencies
if [ "$1" == "--install-deps" ]; then
  echo "Installing Electron dependencies..."
  cd frontend
  npm install
  echo "Dependencies installed successfully"
fi

echo "Done!" 