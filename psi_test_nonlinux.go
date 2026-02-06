//go:build !linux

package psi

import "syscall"

const ioctlSetCtty = uintptr(0)

func ioctlSetInt(fd int, req uintptr, value int) error {
	_ = fd
	_ = req
	_ = value
	return syscall.ENOSYS
}
