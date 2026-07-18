# pic2avif Usage Examples

pic2avif converts images to AVIF using `avifenc`, and preserves EXIF metadata
(via `exiftool`) and filesystem timestamps on the converted file.

## Basic Conversions

### Convert a single image
```bash
./pic2avif photo.jpg
# Output: photo.avif (next to the original)
```

### Convert an entire folder (into a new sibling folder)
```bash
./pic2avif /home/user/photos
# Creates /home/user/photos_avif and puts all converted files there.
# The original /home/user/photos folder is left untouched, so old and
# new files never get mixed together.
```

### Convert multiple specific files
```bash
./pic2avif img1.jpg img2.png img3.webp
```

### Mix files and folders
```bash
./pic2avif photo.jpg /vacation-photos /work/screenshots image.png
# photo.jpg and image.png convert next to themselves;
# /vacation-photos and /work/screenshots each get their own "_avif" sibling folder.
```

### Convert everything into one specific output folder
```bash
./pic2avif --output-dir=/home/user/converted photo.jpg /home/user/photos more.png
# Every converted file (loose files AND everything found in the folder)
# is written flat into /home/user/converted, regardless of where it came from.
```

## Advanced Usage

### High concurrency for large batches
```bash
./pic2avif --concurrency=16 /massive-photo-library
```

### Auto-overwrite existing output files
```bash
./pic2avif --overwrite-existing=true /photos
```

### Dump everything from several folders into one output folder
```bash
./pic2avif --output-dir=/exports vacation-2024 vacation-2025 screenshots
# All converted files from all three folders land flat in /exports
```

## Platform-Specific Examples

### Windows
```cmd
REM Single file
pic2avif.exe C:\Photos\vacation.jpg

REM Folder with spaces in name -> creates "C:\My Documents\Photos_avif"
pic2avif.exe "C:\My Documents\Photos"

REM Multiple folders, each gets its own "_avif" sibling folder
pic2avif.exe C:\Photos D:\Backup\Images

REM Everything into one folder, with options
pic2avif.exe --output-dir=C:\Converted --concurrency=8 C:\Photos
```

### macOS
```bash
# Convert photos in Pictures folder -> creates ~/Pictures_avif
./pic2avif ~/Pictures

# Convert a batch of PNGs on the Desktop
./pic2avif ~/Desktop/*.png
```

### Linux
```bash
# Convert all matching images found via find
find ~ -type f \( -iname "*.jpg" -o -iname "*.png" \) -print0 | \
  xargs -0 ./pic2avif --output-dir=~/converted

# Convert with automatic overwrite
./pic2avif --overwrite-existing=true /var/www/images

# Use all CPU cores (this is also the default)
./pic2avif --concurrency=$(nproc) /media/photos
```

## Metadata & Timestamp Preservation

Every converted file gets:
- **EXIF/metadata** copied from the original via `exiftool -TagsFromFile ... -all:all`.
- **Modification time** copied on all platforms via `os.Chtimes`.
- **Creation ("birth") time**:
  - **Windows** - fully preserved via the `SetFileTime` API.
  - **macOS** - fully preserved via `setattrlist` (requires the binary to have
    been built on macOS, since this uses cgo).
  - **Linux** - not preserved. Most Linux filesystems have no portable
    userspace syscall to *set* a file's birth time, even though some can
    store one, so only the modification time is carried over.

## Special Scenarios

### Converting animated GIFs
```bash
./pic2avif animation.gif
# Output: animation.avif (animated, if avifenc/libavif support is available)
```

### Batch processing with selective overwrite
```bash
# Program will ask for each existing file
./pic2avif /photos
# Respond with:
# y - overwrite this file
# n - skip this file
# a - overwrite all remaining
# i - ignore all remaining
```

## Integration Examples

### Using in a shell script
```bash
#!/bin/bash
# Convert all JPGs in a directory tree

find /photos -name "*.jpg" -type f | while read file; do
    echo "Converting: $file"
    ./pic2avif "$file"
done
```

### Using with GNU Parallel
```bash
# Convert multiple folders in parallel
parallel ./pic2avif ::: folder1 folder2 folder3 folder4
```

### GitHub Actions workflow
```yaml
name: Optimize Images
on: [push]
jobs:
  optimize:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Install libavif tools and ExifTool
        run: |
          sudo apt-get update
          sudo apt-get install -y libavif-bin libimage-exiftool-perl
      - name: Install pic2avif
        run: |
          wget https://github.com/user/pic2avif/releases/latest/download/pic2avif-linux-amd64
          chmod +x pic2avif-linux-amd64
      - name: Convert images
        run: ./pic2avif-linux-amd64 --overwrite-existing=true ./images
```

## Troubleshooting Examples

### Check if dependencies are installed
```bash
# Check avifenc
avifenc --version

# Check ExifTool
exiftool -ver
```

### Compare file sizes
```bash
# Before
ls -lh photo.jpg

# After
ls -lh photo.avif
```

## Common Workflows

### Photography Workflow
```bash
# Import photos from camera
cp /Volumes/CAMERA/DCIM/*.JPG ~/Photos/import/

# Convert to AVIF for archival -> creates ~/Photos/import_avif
./pic2avif ~/Photos/import/

# Verify EXIF data preserved
exiftool ~/Photos/import_avif/IMG_1234.avif
```

### Web Development Workflow
```bash
# Optimize images for a website into one output folder
./pic2avif --output-dir=/var/www/site/images-avif --concurrency=8 /var/www/site/images-src/
```

### Archival Workflow
```bash
# Convert old photos to AVIF with metadata, into a dedicated archive folder
./pic2avif --output-dir=/archive/photos-avif /archive/old-photos/

# Verify timestamps preserved
stat /archive/old-photos/photo.jpg /archive/photos-avif/photo.avif
```
