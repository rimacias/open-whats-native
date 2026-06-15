#!/bin/bash
# Open-Whats Production Build Script
# This script uses fyne-cross (which relies on Docker) to cross-compile the app
# for macOS (.app/dmg), Windows (.exe), and Linux (tarball) from a single machine.

set -e

echo "Building production packages for Open-Whats..."

# Ensure fyne is installed
if ! command -v fyne &> /dev/null
then
    echo "Installing Fyne CLI..."
    go install fyne.io/fyne/v2/cmd/fyne@latest
fi

# Ensure fyne-cross is installed
if ! command -v fyne-cross &> /dev/null
then
    echo "Installing fyne-cross..."
    go install github.com/fyne-io/fyne-cross@latest
fi

# Ensure Docker is running (fyne-cross needs it)
if ! docker info > /dev/null 2>&1; then
  echo "Error: Docker does not seem to be running. Please start Docker to cross-compile."
  exit 1
fi

APP_ID="com.openwhats.native"
TARGET_DIR="./cmd/open-whats/"

echo "-----------------------------------"
echo "Creating Windows executable..."
fyne-cross windows -arch=amd64 -app-id=$APP_ID -dir=$TARGET_DIR
echo "Windows build complete!"

echo "-----------------------------------"
echo "Creating macOS application..."
fyne-cross darwin -arch=amd64,arm64 -app-id=$APP_ID -dir=$TARGET_DIR
echo "macOS build complete!"

echo "-----------------------------------"
echo "Creating Linux tarball..."
fyne-cross linux -arch=amd64 -app-id=$APP_ID -dir=$TARGET_DIR
echo "Linux build complete!"

echo "-----------------------------------"
echo "All builds completed successfully!"
echo "Check the 'fyne-cross/bin/' directory for your binaries."
