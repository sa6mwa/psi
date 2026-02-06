//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package ttygrp

import "syscall"

const (
	ioctlGetPgrp = syscall.TIOCGPGRP
	ioctlSetPgrp = syscall.TIOCSPGRP
)
