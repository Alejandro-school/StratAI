//go:build windows

package main

import (
	"os"
	"syscall"
)

func pathIsReparsePoint(path string, _ os.FileInfo) (bool, error) {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attributes, err := syscall.GetFileAttributes(pathPointer)
	if err != nil {
		return false, err
	}
	return attributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}
