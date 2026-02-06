//go:build aix || android || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package ttygrp

import (
	"os"
	"testing"
)

func TestForegroundPgrp(t *testing.T) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("couldn't open /dev/tty: %v", err)
	}
	defer tty.Close()

	fd := int(tty.Fd())
	pgid, err := GetForegroundPgrp(fd)
	if err != nil {
		t.Fatalf("GetForegroundPgrp failed: %v", err)
	}
	if pgid == 0 {
		t.Fatalf("foreground process group is zero")
	}
	if err := SetForegroundPgrp(fd, pgid); err != nil {
		t.Fatalf("SetForegroundPgrp failed: %v", err)
	}
	again, err := GetForegroundPgrp(fd)
	if err != nil {
		t.Fatalf("GetForegroundPgrp (after) failed: %v", err)
	}
	if again != pgid {
		t.Fatalf("foreground process group changed: got %d, want %d", again, pgid)
	}
}
