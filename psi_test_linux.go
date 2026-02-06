//go:build linux

package psi

import "syscall"

const ioctlSetCtty = syscall.TIOCSCTTY

func ioctlSetInt(fd int, req uintptr, value int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(value))
	if errno != 0 {
		return errno
	}
	return nil
}
