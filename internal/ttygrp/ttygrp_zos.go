//go:build zos

package ttygrp

// Values from golang.org/x/sys/unix for z/OS.
const (
	ioctlGetPgrp = 0x4004a777
	ioctlSetPgrp = 0x8004a776
)
