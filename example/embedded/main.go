package main

import (
	"os"
	"time"

	"pkt.systems/logport/adapters/psl"
)

func main() {
	l := psl.New(os.Stdout).With("app", "embedded-binary")
	l.Info("Sleeping for 10 seconds")
	time.Sleep(10 * time.Second)
	l.Info("Done")
}
