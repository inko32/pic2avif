//go:build darwin
// +build darwin

package main

/*
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/attr.h>
#include <unistd.h>

// set_birthtime sets a file's creation ("birth") time via setattrlist(2),
// the only way to write ATTR_CMN_CRTIME from userspace on macOS.
static int set_birthtime(const char *path, long sec, long nsec) {
	struct attrlist attrList;
	struct timespec ts;

	memset(&attrList, 0, sizeof(attrList));
	attrList.bitmapcount = ATTR_BIT_MAP_COUNT;
	attrList.commonattr  = ATTR_CMN_CRTIME;

	ts.tv_sec  = sec;
	ts.tv_nsec = nsec;

	return setattrlist(path, &attrList, &ts, sizeof(ts), 0);
}
*/
import "C"

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// copyCreationTime sets the output file's creation time to match the input
// file's, using the macOS-only setattrlist syscall.
//
// NOTE: this file uses cgo, so it must be built on macOS (or with a
// working darwin cgo cross-toolchain, e.g. osxcross). A plain
// `GOOS=darwin CGO_ENABLED=0 go build` from another OS will fail here -
// see build.sh for details.
func copyCreationTime(inputPath, outputPath string, inputInfo os.FileInfo) error {
	stat, ok := inputInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("could not read source file stat_t")
	}

	cPath := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cPath))

	if ret := C.set_birthtime(cPath, C.long(stat.Birthtimespec.Sec), C.long(stat.Birthtimespec.Nsec)); ret != 0 {
		return fmt.Errorf("setattrlist failed (errno-style return %d)", int(ret))
	}

	return nil
}
