// Package psi (pkt.systems init) is a tiny PID1 wrapper for single-process
// containers. It provides sane PID1 behavior for scratch images without a
// dedicated init by re-execing a managed child process and handling the
// responsibilities PID1 typically lacks in a regular app.
//
// Behavior summary:
//
//   - If the process is not PID1, Run executes your submain directly.
//   - If the process is PID1, Run re-execs itself as a child process group and
//     the parent becomes a minimal init that:
//   - forwards signals to the child's process group
//   - reaps orphaned child processes
//   - enforces a forced shutdown timeout
//   - makes the child the foreground process group on the controlling TTY
//
// The child process receives a context that is cancelled on terminate-like
// signals (SIGTERM/SIGINT/SIGQUIT/SIGHUP). Your submain should return an exit
// code; psi will exit with the same code.
//
// Configuration:
//
//	PSI_STOP_TIMEOUT sets the duration between the first terminate-like signal
//	and a forced SIGKILL to the child's process group. Accepts Go duration
//	strings like "30s" or "1m15s". Bare numbers like "10" are treated as
//	seconds. Default is 30s.
//
// Example:
//
//	package main
//
//	import (
//	    "context"
//	    "os"
//
//	    "pkt.systems/psi"
//	)
//
//	func submain(ctx context.Context) int {
//	    // your old main logic here
//	    // ctx is cancelled on SIGTERM/SIGINT/SIGQUIT/SIGHUP
//	    return 0
//	}
//
//	func main() {
//	    psi.Run(submain)
//	}
package psi
