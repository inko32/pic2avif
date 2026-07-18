// pic2avif converts images to AVIF using avifenc, preserving EXIF metadata
// and filesystem timestamps (including macOS creation time).
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

// outputExt is the only output format this tool produces.
const outputExt = "avif"

// avifencArgs are the encoder parameters used for every conversion.
var avifencArgs = []string{
	"--range", "full", // full-range color, better for photos
	"--jobs", "all", // use all CPU cores per encode
	"-y", "444", // 4:4:4 chroma, no color subsampling
}

// imageExtensions lists the input formats we'll pick up from folders / accept as arguments.
var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".gif": true, ".bmp": true, ".tiff": true, ".tif": true,
	".heic": true, ".heif": true,
}

// Configuration holds all program settings.
type Configuration struct {
	Concurrency       int
	OverwriteExisting string // "true", "false", or "ask"
	OutputDir         string // if set, ALL output files go here (flat)
}

// ImageJob pairs an input file with its already-resolved output path.
type ImageJob struct {
	InputPath  string
	OutputPath string
}

var (
	config          Configuration
	interactiveLock sync.Mutex
	outputLock      sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
)

func main() {
	parseFlags()

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal, canceling all operations...")
		cancel()
	}()

	if flag.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}

	jobs := collectImageFiles(flag.Args())
	if len(jobs) == 0 {
		fmt.Println("No image files found to process")
		os.Exit(0)
	}

	fmt.Printf("Found %d image(s) to convert to AVIF\n", len(jobs))
	fmt.Printf("Concurrency: %d\n\n", config.Concurrency)

	semaphore := make(chan struct{}, config.Concurrency)
	for _, job := range jobs {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		semaphore <- struct{}{} // acquire

		go func(j ImageJob) {
			defer wg.Done()
			defer func() { <-semaphore }() // release

			if ctx.Err() != nil {
				return
			}

			processImage(j)
		}(job)
	}

	wg.Wait()
	fmt.Println("\nAll conversions completed")
}

// printUsage prints CLI help.
func printUsage() {
	fmt.Println("Usage: pic2avif [options] <file1> <folder1> ...")
	fmt.Println()
	fmt.Println("  Loose files are converted alongside themselves.")
	fmt.Println("  A folder argument (e.g. 'photos') is converted into a new sibling")
	fmt.Println("  folder (e.g. 'photos_avif') so originals are never touched or mixed in.")
	fmt.Println("  Use --output-dir to send everything to one folder instead.")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
}

// parseFlags parses command line flags and sets configuration.
func parseFlags() {
	flag.IntVar(&config.Concurrency, "concurrency", runtime.NumCPU(), "Number of concurrent conversions")
	flag.StringVar(&config.OverwriteExisting, "overwrite-existing", "ask", "Overwrite existing files: true, false, or ask")
	flag.StringVar(&config.OutputDir, "output-dir", "", "Write all converted files into this folder (flat). If omitted, each input folder gets its own '<folder>_avif' sibling folder, and loose files are converted alongside themselves")
	flag.Parse()
}

// collectImageFiles collects all image files from the given paths and resolves
// the output path for each one according to the following rules:
//
//   - If --output-dir is set, EVERY converted file (whether it came from a
//     loose file argument or from a folder argument) is written flat into
//     that single directory.
//   - Otherwise, a folder argument gets its own sibling output folder named
//     "<folder>_avif", so the original folder is never touched and the new
//     files never mix with the old ones.
//   - Otherwise, a loose file argument is converted alongside itself.
func collectImageFiles(paths []string) []ImageJob {
	var jobs []ImageJob

	// Global override directory (flat output for everything)
	var globalOutDir string
	if config.OutputDir != "" {
		globalOutDir = filepath.Clean(config.OutputDir)
		if err := os.MkdirAll(globalOutDir, 0755); err != nil {
			fmt.Printf("Error: cannot create output directory '%s': %v\n", globalOutDir, err)
			os.Exit(1)
		}
	}

	for _, path := range paths {
		path = filepath.Clean(path)

		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("Warning: cannot access '%s': %v\n", path, err)
			continue
		}

		if info.IsDir() {
			outDir := globalOutDir
			if outDir == "" {
				outDir = path + "_" + outputExt
			}
			if err := os.MkdirAll(outDir, 0755); err != nil {
				fmt.Printf("Warning: cannot create output directory '%s': %v\n", outDir, err)
				continue
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				fmt.Printf("Warning: cannot read directory '%s': %v\n", path, err)
				continue
			}

			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if imageExtensions[ext] {
					inputPath := filepath.Join(path, entry.Name())
					outputPath := filepath.Join(outDir, toAvifName(entry.Name()))
					jobs = append(jobs, ImageJob{InputPath: inputPath, OutputPath: outputPath})
				}
			}
		} else {
			ext := strings.ToLower(filepath.Ext(path))
			if !imageExtensions[ext] {
				fmt.Printf("Warning: '%s' is not a recognized image format\n", path)
				continue
			}

			outDir := globalOutDir
			if outDir == "" {
				outDir = filepath.Dir(path)
			}
			outputPath := filepath.Join(outDir, toAvifName(filepath.Base(path)))
			jobs = append(jobs, ImageJob{InputPath: path, OutputPath: outputPath})
		}
	}

	return jobs
}

// toAvifName swaps a bare filename's extension for ".avif".
func toAvifName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name)) + "." + outputExt
}

// processImage handles the complete conversion pipeline for a single image.
func processImage(job ImageJob) {
	inputPath := job.InputPath
	outputPath := job.OutputPath
	logSafe(fmt.Sprintf("Processing: %s", inputPath))

	if !shouldOverwrite(outputPath) {
		logSafe(fmt.Sprintf("Skipped: %s", inputPath))
		return
	}

	if err := convertToAvif(inputPath, outputPath); err != nil {
		logError(inputPath, fmt.Sprintf("avifenc conversion failed: %v", err))
		return
	}

	if err := copyExifMetadata(inputPath, outputPath); err != nil {
		logError(inputPath, fmt.Sprintf("EXIF copy failed: %v", err))
		// Continue anyway, file is already converted
	}

	if err := copyFileTimestamps(inputPath, outputPath); err != nil {
		logError(inputPath, fmt.Sprintf("Timestamp copy failed: %v", err))
		// Continue anyway, file is already converted
	}

	logSafe(fmt.Sprintf("Completed: %s -> %s", inputPath, outputPath))
}

// shouldOverwrite determines if an existing file should be overwritten.
func shouldOverwrite(outputPath string) bool {
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return true
	}

	switch config.OverwriteExisting {
	case "true":
		return true
	case "false":
		return false
	case "ask":
		// Interactive prompt with global lock
		interactiveLock.Lock()
		defer interactiveLock.Unlock()

		fmt.Printf("\nFile exists: %s\n", outputPath)
		fmt.Print("Overwrite? [Y]es / [N]o / [A]ll / [I]gnore all: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		case "a", "all":
			config.OverwriteExisting = "true"
			return true
		case "i", "ignore":
			config.OverwriteExisting = "false"
			return false
		default:
			return false
		}
	}

	return false
}

// convertToAvif executes avifenc to convert a single image to AVIF.
func convertToAvif(inputPath, outputPath string) error {
	avifencBin := "avifenc"
	if runtime.GOOS == "windows" {
		avifencBin = "avifenc.exe"
	}

	args := make([]string, len(avifencArgs))
	copy(args, avifencArgs)
	args = append(args, inputPath, outputPath)

	cmd := exec.CommandContext(ctx, avifencBin, args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	stderrBytes, _ := bufio.NewReader(stderr).ReadBytes(0)

	if err := cmd.Wait(); err != nil {
		writeLogFile(inputPath, string(stderrBytes))
		return err
	}

	return nil
}

// copyExifMetadata copies EXIF metadata using exiftool.
func copyExifMetadata(inputPath, outputPath string) error {
	exiftoolBin := "exiftool"
	if runtime.GOOS == "windows" {
		exiftoolBin = "exiftool.exe"
	}

	cmd := exec.CommandContext(ctx, exiftoolBin,
		"-TagsFromFile", inputPath,
		"-all:all",
		"-overwrite_original",
		outputPath,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	stderrBytes, _ := bufio.NewReader(stderr).ReadBytes(0)

	if err := cmd.Wait(); err != nil {
		writeLogFile(inputPath, string(stderrBytes))
		return err
	}

	return nil
}

// copyFileTimestamps copies filesystem timestamps from source to destination.
// Modification time is handled here for all platforms; creation ("birth")
// time is handled by the platform-specific copyCreationTime implementation
// (see timestamp_linux.go / timestamp_darwin.go / timestamp_windows.go).
func copyFileTimestamps(inputPath, outputPath string) error {
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return err
	}

	modTime := inputInfo.ModTime()

	if err := os.Chtimes(outputPath, modTime, modTime); err != nil {
		return err
	}

	return copyCreationTime(inputPath, outputPath, inputInfo)
}

// writeLogFile writes error log to a .log file next to the input image.
func writeLogFile(inputPath, errorContent string) {
	logPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".log"
	os.WriteFile(logPath, []byte(errorContent), 0644)
}

// logSafe prints a message with mutex protection for concurrent access.
func logSafe(message string) {
	outputLock.Lock()
	defer outputLock.Unlock()
	fmt.Println(message)
}

// logError logs an error message.
func logError(inputPath, message string) {
	logSafe(fmt.Sprintf("ERROR [%s]: %s", inputPath, message))
}
