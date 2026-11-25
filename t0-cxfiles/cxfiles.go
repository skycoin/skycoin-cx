// Package cxfiles holds the open files of the file library by i32 handle,
// which the old loop and the vm both use (decision D119). It uses only the
// standard library.
package cxfiles

import (
	"os"
)

// VIOLATION: state outside State. Owner: t0-cxfiles; moves to OsFileMng in phase 1.
var fileHandles []*os.File

// VIOLATION: state outside State. Owner: t0-cxfiles; moves to OsFileMng in phase 1.
var fileHandlesFree []int32

// FileHandleOf returns the open file of handle, nil for a handle that is not
// open.
func FileHandleOf(handle int32) *os.File {
	if handle >= 0 && handle < int32(len(fileHandles)) && fileHandles[handle] != nil {
		return fileHandles[handle]
	}
	return nil
}

// FileHandleTake gives file the lowest freed handle, or a new one.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func FileHandleTake(file *os.File) int32 {
	var handle int32
	if count := len(fileHandlesFree); count > 0 {
		handle = fileHandlesFree[count-1]
		fileHandlesFree = fileHandlesFree[:count-1]
	} else {
		handle = int32(len(fileHandles))
		fileHandles = append(fileHandles, nil)
	}
	fileHandles[handle] = file
	return handle
}

// FileHandleClose closes the file of handle and frees the handle. It
// returns false for a handle that is not open or a failed close.
// VIOLATION VMALLOC-018: a system library call allocates in Go and the OS, #407.
func FileHandleClose(handle int32) bool {
	file := FileHandleOf(handle)
	if file == nil {
		return false
	}
	closed := file.Close() == nil
	fileHandles[handle] = nil
	fileHandlesFree = append(fileHandlesFree, handle)
	return closed
}

// FileHandleSync commits the writes of the file of handle to stable storage
// and returns false for a handle that is not open or a failed sync.
func FileHandleSync(handle int32) bool {
	file := FileHandleOf(handle)
	return file != nil && file.Sync() == nil
}

// FileHandleSeek moves the file of handle to offset from whence and returns
// the new offset, -1 for a handle that is not open or a failed seek.
func FileHandleSeek(handle int32, offset int64, whence int32) int64 {
	file := FileHandleOf(handle)
	if file == nil {
		return -1
	}
	position, err := file.Seek(offset, int(whence))
	if err != nil {
		return -1
	}
	return position
}
