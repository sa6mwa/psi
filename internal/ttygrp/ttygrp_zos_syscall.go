//go:build zos

package ttygrp

import "syscall"

//go:linkname ioctl syscall.ioctl
func ioctl(fd, req, arg uintptr) syscall.Errno
