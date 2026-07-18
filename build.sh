#!/bin/bash
# Build script for pic2avif - AVIF-only image converter (avifenc + exiftool wrapper)

set -e

echo "Building pic2avif..."

mkdir -p build

echo "Downloading dependencies..."
go mod download

# Build for current platform.
#
# NOTE on macOS creation-time support: timestamp_darwin.go uses cgo
# (setattrlist) to actually preserve the file's creation date, which is not
# possible from pure Go. That means a real macOS binary can only be produced
# by building ON macOS (or with a working osxcross cgo cross-toolchain) with
# CGO_ENABLED=1 (the default when GOOS matches the host). Cross-compiling
# for darwin from Linux/Windows with plain `go build` will fail on that file.
echo "Building for current platform ($(go env GOOS)/$(go env GOARCH))..."
go build -o build/pic2avif

echo ""
echo "Build complete:"
ls -lh build/

echo ""
echo "To install for current platform, run:"
echo "  sudo cp build/pic2avif /usr/local/bin/"

echo ""
echo "To cross-compile for other platforms:"
echo "  Linux:   GOOS=linux   GOARCH=amd64 go build -o build/pic2avif-linux-amd64"
echo "  Windows: GOOS=windows GOARCH=amd64 go build -o build/pic2avif-windows-amd64.exe"
echo "  macOS:   must be built ON macOS (see note above) -- e.g."
echo "           GOOS=darwin GOARCH=arm64 go build -o build/pic2avif-macos-arm64"
