package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	_ "embed"

	"pkt.systems/emrun"
	"pkt.systems/psi"
	"pkt.systems/pslog"

	"golang.org/x/term"
)

//go:embed embedded-binary
var embedded []byte

func main() {
	psi.Run(submain)
}

func submain(ctx context.Context) int {
	l := pslog.New(os.Stdout).With("app", "example")
	if isInteractive(os.Args) {
		return runInteractive(ctx, l)
	}
	l.Debug("Starting embedded executable")
	if err := emrun.RunIO(context.Background(), nil, os.Stdout, embedded); err != nil {
		l.Error("Embedded executable failed", "error", err)
		return 1
	}
	l.Debug("Done")
	return 0
}

func isInteractive(args []string) bool {
	if len(args) < 2 {
		return false
	}
	return strings.EqualFold(args[1], "interactive")
}

func runInteractive(ctx context.Context, l pslog.Logger) int {
	if err := promptLineCtx(ctx, os.Stdin, os.Stdout, "Username: "); err != nil {
		l.Error("Interactive read failed", "error", err)
		return 1
	}
	if err := promptPasswordCtx(ctx, os.Stdin, os.Stdout, "Password: "); err != nil {
		l.Error("Interactive read failed", "error", err)
		return 1
	}
	select {
	case <-ctx.Done():
		l.Info("Interactive session cancelled")
		return 99
	default:
	}
	fmt.Fprintln(os.Stdout, "Interactive input complete.")
	return 0
}

func promptLineCtx(ctx context.Context, r io.Reader, w io.Writer, prompt string) error {
	if _, err := fmt.Fprint(w, prompt); err != nil {
		return err
	}
	type result struct {
		err error
	}
	ch := make(chan result, 1)
	go func() {
		buf := make([]byte, 4096)
		n, err := r.Read(buf)
		if err != nil {
			ch <- result{err: err}
			return
		}
		if n == 0 {
			ch <- result{err: io.EOF}
			return
		}
		ch <- result{}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-ch:
		return res.err
	}
}

func promptPasswordCtx(ctx context.Context, r *os.File, w io.Writer, prompt string) error {
	if _, err := fmt.Fprint(w, prompt); err != nil {
		return err
	}
	type result struct {
		err error
	}
	ch := make(chan result, 1)
	go func() {
		fd := int(r.Fd())
		if term.IsTerminal(fd) {
			_, err := term.ReadPassword(fd)
			if _, werr := fmt.Fprintln(w); werr != nil && err == nil {
				err = werr
			}
			ch <- result{err: err}
			return
		}
		buf := make([]byte, 4096)
		n, err := r.Read(buf)
		if err != nil {
			ch <- result{err: err}
			return
		}
		if n == 0 {
			ch <- result{err: io.EOF}
			return
		}
		ch <- result{}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-ch:
		return res.err
	}
}
