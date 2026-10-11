//go:build windows

// Package singleinstance stops a second copy of LoL Remote from running at
// the same time — without it, each copy would bind its own port, show its
// own tray icon, and confuse anyone with phone bookmarks to the old one.
package singleinstance

import (
	"syscall"
	"unsafe"
)

// Fixed, arbitrary name — just needs to not collide with another app's.
const mutexName = `Global\LoLRemote-7F3C2E7B-7E2E-4C2E-9C7F-9B1A2D3E4F5A`

const errorAlreadyExists = 183

// Acquire reports whether this process is the only running instance. The OS
// releases the underlying mutex automatically on process exit, so there is
// nothing to release explicitly.
func Acquire() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("CreateMutexW")
	namePtr, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		return true // fail open rather than block the user over an encoding error
	}
	handle, _, callErr := proc.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))
	if handle == 0 {
		return true // CreateMutex itself failed; fail open
	}
	errno, _ := callErr.(syscall.Errno)
	return errno != errorAlreadyExists
}
