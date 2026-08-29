// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package host

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var globalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

// memoryStatusEx mirrors the Windows MEMORYSTATUSEX structure. WALDO only
// consumes TotalPhysical, but the complete layout is required by the API.
type memoryStatusEx struct {
	Length                   uint32
	MemoryLoad               uint32
	TotalPhysical            uint64
	AvailablePhysical        uint64
	TotalPageFile            uint64
	AvailablePageFile        uint64
	TotalVirtual             uint64
	AvailableVirtual         uint64
	AvailableExtendedVirtual uint64
}

func physicalMemory() (uint64, error) {
	status := memoryStatusEx{}
	status.Length = uint32(unsafe.Sizeof(status))
	result, _, callErr := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if result == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return 0, fmt.Errorf("GlobalMemoryStatusEx: %w", callErr)
		}
		return 0, fmt.Errorf("GlobalMemoryStatusEx failed")
	}
	if status.TotalPhysical == 0 {
		return 0, fmt.Errorf("GlobalMemoryStatusEx returned zero physical memory")
	}
	return status.TotalPhysical, nil
}
