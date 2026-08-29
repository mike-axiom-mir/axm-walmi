// Copyright (c) 2026 OpenWALDO Project contributors
// Copyright (c) 2026 CtrlIQ, Inc.
// Copyright (c) 2026 Gregory M. Kurtzer
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package host

import (
	"os"

	"golang.org/x/sys/windows"
)

func filesystemCapacity() (uint64, uint64, error) {
	directory, err := os.Getwd()
	if err != nil {
		return 0, 0, err
	}
	path, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return 0, 0, err
	}
	var available, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(path, &available, &total, &free); err != nil {
		return 0, 0, err
	}
	return total, available, nil
}
