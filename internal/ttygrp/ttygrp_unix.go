//go:build aix || android || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package ttygrp

import (
	"syscall"
	"unsafe"
)

func getForegroundPgrp(fd int) (int, error) {
	var pgid int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(ioctlGetPgrp), uintptr(unsafe.Pointer(&pgid)))
	if errno != 0 {
		return 0, errno
	}
	return pgid, nil
}

func setForegroundPgrp(fd int, pgid int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(ioctlSetPgrp), uintptr(unsafe.Pointer(&pgid)))
	if errno != 0 {
		return errno
	}
	return nil
}
