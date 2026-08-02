# pic2avif

A small, fast, concurrent CLI tool that converts images to **AVIF** using
[`avifenc`](https://github.com/AOMediaCodec/libavif), while preserving each
file's EXIF metadata and filesystem timestamps.

Tested on Windows 11 x86_64, libavif 1.4.2, go1.26.5
requires avifenc

I wanted to use ffmpeg but it's parameters are too complicated. 

## Features

- Converts single files, multiple files, and whole folders in one command
- Runs conversions concurrently, with a configurable worker count
- Preserves EXIF/metadata via `exiftool`
- Preserves filesystem timestamps:
  - Modification time — all platforms
  - Creation time — Windows and macOS (see [Timestamp support](#timestamp-support) below)
- Folder-safe by default: converting a folder never mixes new AVIF files in
  with the originals
- Optional single output folder for combining input from multiple
  files/folders into one place
- Interactive, auto-yes, or auto-skip handling for existing output files
- Graceful shutdown on Ctrl-C (in-flight conversions finish or cancel cleanly)

## Note

On Windows the converted files may look wrong in color. Use chromium-based browser to compare instead of windows-builtin photo viewer which is buggy.

## Requirements

Two external tools should be installed and on your `PATH`:

- [`avifenc`](https://github.com/AOMediaCodec/libavif) (from libavif) — does the actual image conversion
- [`exiftool`](https://exiftool.org/) (optional)— copies metadata from the original file to the AVIF

Check both are available:

```bash
avifenc --version
exiftool -ver
```
exiftool is optional if the source files are jpg or something that avifenc could process automatically. 

## Installation

### Build from source

Requires [Go](https://go.dev/) 1.21+.

```bash
git clone <this-repo>
cd pic2avif
go mod download
go build -o pic2avif
```

Or use the provided build scripts:

```bash
./build.sh      # Linux / macOS
build.bat       # Windows
```

### Automated builds & releases (GitHub Actions)

This repo includes two workflows under `.github/workflows/`:

- **`ci.yml`** - runs `go vet` and `go build` on Linux, Windows, and macOS
  runners for every push to `main` and every pull request, so breakage is
  caught before you ever cut a release.
- **`release.yml`** - builds binaries for `linux/amd64`, `linux/arm64`,
  `windows/amd64`, `darwin/amd64`, and `darwin/arm64`, then publishes them
  as assets on a GitHub Release. macOS builds run natively on `macos-13`
  (Intel) and `macos-latest` (Apple Silicon) runners rather than being
  cross-compiled, since `timestamp_darwin.go` needs cgo.

To cut a release, tag a commit and push the tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The `release.yml` workflow picks up any tag matching `v*.*.*`, builds all
five platform binaries, and attaches them to a new GitHub Release
(auto-generating release notes from the commits/PRs since the last tag).

> **Note on macOS creation time:** `timestamp_darwin.go` uses cgo to call
> `setattrlist` so it can actually preserve creation ("birth") time on macOS.
> That means a macOS binary must be built *on macOS* (or with a working
> `osxcross` cgo cross-toolchain) — cross-compiling for `darwin` from Linux
> or Windows with plain `go build` will fail on that file. Linux and Windows
> binaries have no such restriction and cross-compile normally.

Then optionally install it somewhere on your `PATH`:

```bash
sudo cp build/pic2avif /usr/local/bin/
```

## Usage

```
pic2avif [options] <file1> <folder1> ...
```

You can freely mix individual files and folders in the same command.

- **Loose files** are converted alongside themselves (`photo.jpg` → `photo.avif`, same folder).
- **A folder argument** (e.g. `photos`) gets its own sibling output folder
  (`photos_avif`), so the original folder is never touched or mixed with
  the converted files.
- **`--output-dir`** overrides both of the above and sends every converted
  file — from loose files and from folders alike — flat into one folder
  you specify.

### Options

| Flag                  | Default | Description |
|------------------------|---------|-------------|
| `--concurrency`        | number of CPU cores | Number of images converted in parallel |
| `--overwrite-existing` | `ask`   | `true`, `false`, or `ask` before overwriting an existing output file |
| `--output-dir`         | *(none)* | Write all converted files into this one folder (flat) |

### Examples

Convert a single image:
```bash
pic2avif photo.jpg
```

Convert a whole folder (creates `photos_avif` next to `photos`):
```bash
pic2avif photos
```

Convert several folders and files into one output folder:
```bash
pic2avif --output-dir=/exports vacation-2024 vacation-2025 photo.jpg
```

Batch-convert with higher concurrency and no overwrite prompts:
```bash
pic2avif --concurrency=16 --overwrite-existing=true /massive-photo-library
```

See [EXAMPLES.md](./EXAMPLES.md) for more.

## Supported input formats

`.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.bmp`, `.tiff`, `.tif`, `.heic`, `.heif`

Output is always AVIF.

## Timestamp support

| Platform | Modification time | Creation time |
|----------|:---:|:---:|
| Windows  | ✅ | ✅ |
| macOS    | ✅ | ✅ (requires building on macOS — see note above) |
| Linux    | ✅ | ❌ — most Linux filesystems have no portable userspace syscall to *set* a birth time, even on filesystems that can store one |

## Error handling

If a conversion or metadata copy fails, pic2avif writes a `.log` file next
to the input image containing the tool's stderr output, and continues
processing the rest of the batch.

