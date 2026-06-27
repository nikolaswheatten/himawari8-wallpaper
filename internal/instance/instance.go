//go:build windows

package instance

import (
	"fmt"
	"syscall"
	"unsafe"
)

const mutexName = "Global\\Himawari8WallpaperUpdater"

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	createMutexW    = kernel32.NewProc("CreateMutexW")
	releaseMutex    = kernel32.NewProc("ReleaseMutex")
	closeHandle     = kernel32.NewProc("CloseHandle")
	getLastError    = kernel32.NewProc("GetLastError")
	errorAlreadyExists = 183
)

type Lock struct {
	handle syscall.Handle
}

func Acquire() (*Lock, error) {
	name, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		return nil, err
	}

	ret, _, callErr := createMutexW.Call(
		0,
		0,
		uintptr(unsafe.Pointer(name)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("CreateMutexW: %v", callErr)
	}

	lastErr, _, _ := getLastError.Call()
	if lastErr == uintptr(errorAlreadyExists) {
		closeHandle.Call(ret)
		return nil, fmt.Errorf("another instance is already running")
	}

	return &Lock{handle: syscall.Handle(ret)}, nil
}

func (l *Lock) Release() {
	if l == nil || l.handle == 0 {
		return
	}
	releaseMutex.Call(uintptr(l.handle))
	closeHandle.Call(uintptr(l.handle))
	l.handle = 0
}
