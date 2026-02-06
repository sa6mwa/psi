//go:build plan9 || windows || js || wasm

package ttygrp

import "syscall"

func getForegroundPgrp(fd int) (int, error) {
	return 0, syscall.ENOSYS
}

func setForegroundPgrp(fd int, pgid int) error {
	return syscall.ENOSYS
}
