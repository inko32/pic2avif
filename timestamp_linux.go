//go:build linux
// +build linux

package main

import "os"

// copyCreationTime is a no-op on Linux. Linux filesystems have no portable
// userspace syscall to SET a file's birth ("creation") time — statx(2) can
// read stx_btime on ext4/xfs/btrfs, but there is nothing symmetrical to
// write it back with. Modification time is already handled by os.Chtimes
// before this is called, so there's nothing further to do here.
func copyCreationTime(inputPath, outputPath string, inputInfo os.FileInfo) error {
	return nil
}
