@echo off
REM Build script for pic2avif - AVIF-only image converter (avifenc + exiftool wrapper) - Windows

echo Building pic2avif...

if not exist build mkdir build

echo Downloading dependencies...
go mod download

echo Building for current platform (windows/amd64)...
set GOOS=windows
set GOARCH=amd64
go build -o build\pic2avif.exe

echo.
echo Build complete! Binary is in the 'build' directory:
dir build

echo.
echo To use, add the build directory to your PATH or copy pic2avif.exe to a directory in your PATH
echo.
echo Note: cross-compiling a macOS binary with real creation-time support
echo requires building ON macOS (timestamp_darwin.go uses cgo), so it is not
echo produced from this Windows build script.
pause
