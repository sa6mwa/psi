//go:build solaris || illumos

package ttygrp

// Values from golang.org/x/sys/unix.
const (
	ioctlGetPgrp = 0x7414
	ioctlSetPgrp = 0x7415
)
