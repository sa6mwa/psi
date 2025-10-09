package psi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	helperEnv     = "GO_WANT_HELPER_PROCESS"
	helperModeEnv = "GO_HELPER_MODE"
)

func TestRunNonPID1(t *testing.T) {
	cmd := helperCommand("run-nonpid")
	err := cmd.Run()
	if exit := exitStatus(err); exit != 42 {
		t.Fatalf("expected exit code 42, got %d (err=%v)", exit, err)
	}
}

func TestRunChildContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signals not reliable on Windows")
	}

	cmd := helperCommand("run-child", fmt.Sprintf("%s=%s", childEnvKey, childEnvVal))
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper: %v", err)
	}
	// Give runChild time to install the signal handler.
	time.Sleep(100 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to signal helper: %v", err)
	}
	err := cmd.Wait()
	if exit := exitStatus(err); exit != 99 {
		t.Fatalf("expected exit code 99 after cancellation, got %d (err=%v)", exit, err)
	}
}

func TestParseStopTimeoutDefault(t *testing.T) {
	t.Setenv(stopTimeoutEnv, "")
	def := 45 * time.Second
	if got := parseStopTimeout(def); got != def {
		t.Fatalf("expected default %s, got %s", def, got)
	}
}

func TestParseStopTimeoutDurationStrings(t *testing.T) {
	t.Setenv(stopTimeoutEnv, "1m15s")
	want := 75 * time.Second
	if got := parseStopTimeout(0); got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestParseStopTimeoutPlainDigits(t *testing.T) {
	t.Setenv(stopTimeoutEnv, "12")
	want := 12 * time.Second
	if got := parseStopTimeout(0); got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestParseStopTimeoutInvalidFallsBack(t *testing.T) {
	var buf bytes.Buffer
	origWriter := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(origWriter) })

	t.Setenv(stopTimeoutEnv, "bogus")
	def := 33 * time.Second
	if got := parseStopTimeout(def); got != def {
		t.Fatalf("expected fallback %s, got %s", def, got)
	}
	if !strings.Contains(buf.String(), "invalid PSI_STOP_TIMEOUT") {
		t.Fatalf("expected warning log, got %q", buf.String())
	}
}

func TestParseStopTimeoutNegativeFallsBack(t *testing.T) {
	t.Setenv(stopTimeoutEnv, "-2s")
	def := 12 * time.Second
	if got := parseStopTimeout(def); got != def {
		t.Fatalf("expected fallback %s, got %s", def, got)
	}
}

func TestIsAllDigits(t *testing.T) {
	cases := map[string]bool{
		"":      false,
		"0":     true,
		"12345": true,
		"12a":   false,
		" 12 ":  false,
	}
	for input, want := range cases {
		if got := isAllDigits(input); got != want {
			t.Fatalf("isAllDigits(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestIsTerminateSignal(t *testing.T) {
	if !isTerminateSignal(syscall.SIGTERM) {
		t.Fatal("SIGTERM should be terminate signal")
	}
	if isTerminateSignal(syscall.SIGUSR1) {
		t.Fatal("SIGUSR1 should not be terminate signal")
	}
}

func TestToSyscallSignal(t *testing.T) {
	if sig, ok := toSyscallSignal(syscall.SIGUSR2); !ok || sig != syscall.SIGUSR2 {
		t.Fatalf("expected SIGUSR2 roundtrip, got %v ok=%v", sig, ok)
	}
	fs := fakeSignal("sigterm")
	sig, ok := toSyscallSignal(fs)
	if !ok || sig != syscall.SIGTERM {
		t.Fatalf("expected SIGTERM from fake signal, got %v ok=%v", sig, ok)
	}
	if _, ok := toSyscallSignal(fakeSignal("unknown")); ok {
		t.Fatalf("unexpected success converting unknown signal")
	}
}

func TestKillTimerC(t *testing.T) {
	select {
	case <-killTimerC(nil):
		t.Fatal("killTimerC(nil) should not fire immediately")
	default:
	}
	timer := time.NewTimer(20 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-killTimerC(timer):
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timer channel did not fire")
	}
}

func TestEffectiveTimeout(t *testing.T) {
	t.Setenv(stopTimeoutEnv, "5s")
	if got := EffectiveTimeout(); got != 5*time.Second {
		t.Fatalf("expected 5s, got %s", got)
	}
}

func TestIsPID1(t *testing.T) {
	if os.Getpid() == 1 {
		t.Skip("test process is PID 1 unexpectedly")
	}
	if IsPID1() {
		t.Fatal("IsPID1 should be false for regular test process")
	}
}

func TestChildPIDEnv(t *testing.T) {
	t.Setenv(childEnvKey, childEnvVal)
	got, ok := ChildPIDEnv()
	if !ok || got != childEnvVal {
		t.Fatalf("ChildPIDEnv() = %q, %v; want %q, true", got, ok, childEnvVal)
	}
}

func TestReapUntilChildExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Wait4 not available on Windows")
	}
	otherPID, err := forkExecExit(0)
	if err != nil {
		t.Fatalf("failed to fork extra child: %v", err)
	}
	targetPID, err := forkExecExit(7)
	if err != nil {
		t.Fatalf("failed to fork target child: %v", err)
	}
	if code := reapUntilChildExit(targetPID); code != 7 {
		t.Fatalf("expected exit status 7, got %d", code)
	}
	// Ensure the extra child is also reaped to avoid leaks.
	var ws syscall.WaitStatus
	_, err = syscall.Wait4(otherPID, &ws, syscall.WNOHANG, nil)
	if err != nil && !errors.Is(err, syscall.ECHILD) {
		t.Fatalf("unexpected wait after reap: %v", err)
	}
}

func TestDrainZombiesNonBlock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Wait4 not available on Windows")
	}
	pid, err := forkExecExit(0)
	if err != nil {
		t.Fatalf("failed to fork child: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	drainZombiesNonBlock()
	var ws syscall.WaitStatus
	_, err = syscall.Wait4(pid, &ws, syscall.WNOHANG, nil)
	if err == nil {
		t.Fatalf("expected no child left to reap")
	}
	if !errors.Is(err, syscall.ECHILD) {
		t.Fatalf("expected ECHILD, got %v", err)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		return
	}
	mode := os.Getenv(helperModeEnv)
	switch mode {
	case "run-nonpid":
		Run(func(context.Context) int { return 42 })
	case "run-child":
		Run(func(ctx context.Context) int {
			select {
			case <-ctx.Done():
				return 99
			case <-time.After(2 * time.Second):
				return 23
			}
		})
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode %q\n", mode)
		os.Exit(3)
	}
	os.Exit(0)
}

type fakeSignal string

func (f fakeSignal) String() string { return string(f) }
func (fakeSignal) Signal()          {}

func helperCommand(mode string, extraEnv ...string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", mode)
	env := filterEnv(os.Environ(), childEnvKey)
	env = append(env, helperEnv+"=1", helperModeEnv+"="+mode)
	env = append(env, extraEnv...)
	cmd.Env = env
	return cmd
}

func exitStatus(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func filterEnv(env []string, key string) []string {
	prefix := key + "="
	out := env[:0]
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return append([]string(nil), out...)
}

func forkExecExit(code int) (int, error) {
	prog := "/bin/sh"
	args := []string{"sh", "-c", fmt.Sprintf("exit %d", code)}
	attr := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}
	return syscall.ForkExec(prog, args, attr)
}
