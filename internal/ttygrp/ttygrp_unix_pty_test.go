//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || illumos) && !android

package ttygrp

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"

	"github.com/creack/pty"
)

const ptyHelperEnv = "TTYGRP_PTY_HELPER"

func TestForegroundPgrpPTY(t *testing.T) {
	if os.Getenv(ptyHelperEnv) == "1" {
		runPtyHelper()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestForegroundPgrpPTY")
	cmd.Env = append(os.Environ(), ptyHelperEnv+"=1")
	out, err := cmd.CombinedOutput()
	if string(out) == "skip\n" {
		t.Skip("pty not supported")
	}
	if err != nil {
		t.Fatalf("helper failed: %v (output=%q)", err, string(out))
	}
}

func runPtyHelper() {
	signal.Ignore(syscall.SIGHUP)
	master, slave, err := pty.Open()
	if err != nil {
		_, _ = os.Stdout.WriteString("skip\n")
		os.Exit(0)
	}
	defer master.Close()
	defer slave.Close()

	if _, err := syscall.Setsid(); err != nil && err != syscall.EPERM {
		fatalf("setsid: %v", err)
	}
	if err := ioctlSetInt(int(slave.Fd()), syscall.TIOCSCTTY, 0); err != nil {
		fatalf("tiocsctty: %v", err)
	}

	selfPGID := syscall.Getpgrp()
	if err := SetForegroundPgrp(int(slave.Fd()), selfPGID); err != nil {
		fatalf("set foreground (self): %v", err)
	}

	cmd := exec.Command("sh", "-c", "sleep 2")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	if err := cmd.Start(); err != nil {
		fatalf("start child: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	if err := SetForegroundPgrp(int(slave.Fd()), cmd.Process.Pid); err != nil {
		fatalf("set foreground (child): %v", err)
	}
	got, err := GetForegroundPgrp(int(slave.Fd()))
	if err != nil {
		fatalf("get foreground: %v", err)
	}
	if got != cmd.Process.Pid {
		fatalf("foreground pgid = %d, want %d", got, cmd.Process.Pid)
	}
	os.Exit(0)
}

func ioctlSetInt(fd int, req uintptr, value int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(value))
	if errno != 0 {
		return errno
	}
	return nil
}

func fatalf(format string, args ...any) {
	_, _ = os.Stderr.WriteString(fmt.Sprintf(format+"\n", args...))
	os.Exit(2)
}
