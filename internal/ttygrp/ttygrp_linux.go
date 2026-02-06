//go:build linux || android

package ttygrp

import "syscall"

const (
	ioctlGetPgrp = syscall.TIOCGPGRP
	ioctlSetPgrp = syscall.TIOCSPGRP
)
