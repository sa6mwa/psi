//go:build aix

package ttygrp

// Values from golang.org/x/sys/unix.
const (
	ioctlGetPgrp = 0x40047477
	ioctlSetPgrp = 0xffffffff80047476
)
