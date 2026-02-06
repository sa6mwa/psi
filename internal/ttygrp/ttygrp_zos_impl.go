//go:build zos

package ttygrp

import "unsafe"

func getForegroundPgrp(fd int) (int, error) {
	var pgid int
	if errno := ioctl(uintptr(fd), uintptr(ioctlGetPgrp), uintptr(unsafe.Pointer(&pgid))); errno != 0 {
		return 0, errno
	}
	return pgid, nil
}

func setForegroundPgrp(fd int, pgid int) error {
	if errno := ioctl(uintptr(fd), uintptr(ioctlSetPgrp), uintptr(unsafe.Pointer(&pgid))); errno != 0 {
		return errno
	}
	return nil
}
