// psi (pkt.systems init) is a tiny PID1 wrapper for single-process containers.
// It runs your application's "submain" and, when running as PID 1, provides
// proper signal forwarding (to the child's process group), zombie reaping, and
// a configurable forced-shutdown timeout via PSI_STOP_TIMEOUT (default 30s).
//
// Usage:
//
//	func submain(ctx context.Context) int { /* your old main */ }
//	func main() { psi.Run(submain) }
//
// Build statically for scratch images:
//
//	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"
package psi

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const childEnvKey = "PSI_CHILD"
const childEnvVal = "1"

const stopTimeoutEnv = "PSI_STOP_TIMEOUT"
const defaultStopTimeout = 30 * time.Second

// SubMain is your application's entrypoint (old main), returning an exit code.
// The provided context is cancelled when a termination signal is received.
type SubMain func(ctx context.Context) int

// Run wraps submain with PID1 responsibilities when needed. If PID != 1 and
// PSI_CHILD not set: runs submain directly (nice for local dev). If PID == 1
// and PSI_CHILD not set: forks/execs itself; parent becomes init, child runs
// submain. If PSI_CHILD == "1": executes submain path (child).
func Run(submain SubMain) {
	if os.Getenv(childEnvKey) == childEnvVal {
		runChild(submain)
		// runChild never returns.
		return
	}
	if os.Getpid() != 1 {
		code := submain(context.Background())
		os.Exit(code)
	}
	runAsInit()
	// runAsInit never returns.
}

func runChild(submain SubMain) {
	// Child path: set up graceful cancellation on termination signals.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	termCh := make(chan os.Signal, 8)
	signal.Notify(termCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	go func() {
		for range termCh {
			// Cancel once; repeated signals are fine.
			cancel()
		}
	}()
	code := submain(ctx)
	os.Exit(code)
}

func runAsInit() {
	// Re-exec this binary as the managed child running submain.
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", childEnvKey, childEnvVal))
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Put child in its own process group so signals can be forwarded to the whole tree.
		Setpgid: true,
	}
	if err := cmd.Start(); err != nil {
		log.Fatalf("psi: failed to start child: %v", err)
	}
	childPID := cmd.Process.Pid
	// Channel that yields the child's exit code once reaped.
	done := make(chan int, 1)
	go func() {
		done <- reapUntilChildExit(childPID)
	}()
	// Signal forwarding and shutdown policy.
	allSig := make(chan os.Signal, 64)
	// Subscribe to all signals we can catch; SIGKILL/SIGSTOP cannot be caught.
	signal.Notify(allSig)
	// Parse stop timeout once.
	stopTimeout := parseStopTimeout(defaultStopTimeout)
	// Start the kill timer on the first terminate-like signal.
	var startOnce sync.Once
	var killTimer *time.Timer
	startKillTimer := func() {
		if killTimer == nil {
			killTimer = time.NewTimer(stopTimeout)
		} else {
			if !killTimer.Stop() {
				select {
				case <-killTimer.C:
				default:
				}
			}
			killTimer.Reset(stopTimeout)
		}
	}
	// Supervisor loop: wait on signals, child exit, or forced kill timer.
	for {
		select {
		case code := <-done:
			// Child exited; small grace to reap stragglers, then exit with its code.
			time.Sleep(50 * time.Millisecond)
			drainZombiesNonBlock()
			os.Exit(code)
		case s := <-allSig:
			// Never handle SIGCHLD here (we reap in reapUntilChildExit).
			if s == syscall.SIGCHLD {
				continue
			}
			// Forward everything we can to the child's process group.
			if sig, ok := toSyscallSignal(s); ok {
				_ = syscall.Kill(-childPID, sig)
			}
			// On first terminate-like signal, start the forced-kill countdown.
			if isTerminateSignal(s) {
				startOnce.Do(func() {
					startKillTimer()
				})
			}
		case <-killTimerC(killTimer):
			// Forced shutdown: SIGKILL the child's process group.
			_ = syscall.Kill(-childPID, syscall.SIGKILL)
			// Wait for reap loop to deliver child's exit code.
			code := <-done
			os.Exit(code)
		}
	}
}

// reapUntilChildExit reaps children until the managed child exits,
// returning the managed child's exit code (shell-style).
func reapUntilChildExit(childPID int) int {
	for {
		var ws syscall.WaitStatus
		var ru syscall.Rusage
		pid, err := syscall.Wait4(-1, &ws, 0, &ru)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			if err == syscall.ECHILD {
				// No children left; assume success if we somehow missed it.
				return 0
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if pid == childPID {
			if ws.Exited() {
				return ws.ExitStatus()
			}
			if ws.Signaled() {
				return 128 + int(ws.Signal())
			}
			return 1
		}
		// Reaped some other orphan; keep looping.
	}
}

// drainZombiesNonBlock performs a single non-blocking reap pass.
func drainZombiesNonBlock() {
	for {
		var ws syscall.WaitStatus
		pid, err := syscall.Wait4(-1, &ws, syscall.WNOHANG, nil)
		if err != nil || pid <= 0 {
			return
		}
	}
}

// parseStopTimeout reads PSI_STOP_TIMEOUT, accepts Go time.Duration strings.
// Falls back to default on empty or invalid values.
// Examples: "30s", "1m15s", "2h"; bare numbers like "30" are treated as seconds.
func parseStopTimeout(def time.Duration) time.Duration {
	val := strings.TrimSpace(os.Getenv(stopTimeoutEnv))
	if val == "" {
		return def
	}
	// Allow plain seconds like "30" as a convenience => "30s".
	if isAllDigits(val) {
		val = val + "s"
	}
	d, err := time.ParseDuration(val)
	if err != nil || d < 0 {
		log.Printf("psi: invalid %s=%q; using default %s", stopTimeoutEnv, val, def)
		return def
	}
	return d
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isTerminateSignal(s os.Signal) bool {
	switch s {
	case syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP:
		return true
	default:
		return false
	}
}

func toSyscallSignal(s os.Signal) (syscall.Signal, bool) {
	if sig, ok := s.(syscall.Signal); ok {
		return sig, true
	}
	switch strings.ToUpper(s.String()) {
	case "SIGTERM":
		return syscall.SIGTERM, true
	case "SIGINT":
		return syscall.SIGINT, true
	case "SIGQUIT":
		return syscall.SIGQUIT, true
	case "SIGHUP":
		return syscall.SIGHUP, true
	case "SIGUSR1":
		return syscall.SIGUSR1, true
	case "SIGUSR2":
		return syscall.SIGUSR2, true
	default:
		return 0, false
	}
}

// killTimerC safely returns the channel for a possibly-nil timer.
// If the timer is nil (not started yet), return a channel that never fires.
func killTimerC(t *time.Timer) <-chan time.Time {
	if t == nil {
		never := make(chan time.Time)
		return never
	}
	return t.C
}

// Optional helpers for testing / diagnostics...

// EffectiveTimeout exposes the parsed PSI_STOP_TIMEOUT (or default). Useful in
// unit tests.
func EffectiveTimeout() time.Duration {
	return parseStopTimeout(defaultStopTimeout)
}

// IsPID1 reports whether the current process is PID 1.
func IsPID1() bool { return os.Getpid() == 1 }

// ChildPIDEnv returns the PSI_CHILD env var as seen by the current process.
func ChildPIDEnv() (string, bool) {
	v, ok := os.LookupEnv(childEnvKey)
	return v, ok
}
