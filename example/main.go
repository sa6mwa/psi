package main

import (
	"context"
	"os"

	_ "embed"

	"pkt.systems/emrun"
	"pkt.systems/logport/adapters/zerologger"
	"pkt.systems/psi"
)

//go:embed embedded-binary
var embedded []byte

func main() {
	psi.Run(submain)
}

func submain(ctx context.Context) int {
	l := zerologger.New(os.Stdout).With("app", "example")
	l.Debug("Starting embedded executable")
	if err := emrun.RunIO(context.Background(), nil, os.Stdout, embedded); err != nil {
		l.Error("Embedded executable failed", "error", err)
		return 1
	}
	l.Debug("Done")
	return 0
}
