//go:build windows
// +build windows

package main

import (
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// copyCreationTime copies the creation time on Windows using SetFileTime API
func copyCreationTime(inputPath, outputPath string, inputInfo os.FileInfo) error {
	// Get creation time from input file
	inputStat := inputInfo.Sys().(*syscall.Win32FileAttributeData)
	creationTime := time.Unix(0, inputStat.CreationTime.Nanoseconds())

	// Open output file
	outputPathUTF16, err := syscall.UTF16PtrFromString(outputPath)
	if err != nil {
		return err
	}

	handle, err := windows.CreateFile(
		outputPathUTF16,
		windows.FILE_WRITE_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)

	// Convert to Windows FILETIME
	creationFileTime := windows.NsecToFiletime(creationTime.UnixNano())
	modTimeFileTime := windows.NsecToFiletime(inputInfo.ModTime().UnixNano())

	// Set both creation time and modification time
	err = windows.SetFileTime(handle, &creationFileTime, nil, &modTimeFileTime)
	return err
}
