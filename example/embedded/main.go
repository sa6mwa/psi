package main

import (
	"os"
	"time"

	"pkt.systems/pslog"
)

func main() {
	l := pslog.New(os.Stdout).With("app", "embedded-binary")
	l.Info("Sleeping for 10 seconds")
	time.Sleep(10 * time.Second)
	l.Info("Done")
}
