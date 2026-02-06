# psi

`psi` (**p**kt.**s**ystems **i**nit) is a tiny PID1 wrapper for single-process
containers. It provides sane PID1 behavior for `FROM scratch` images without a
dedicated init: signal forwarding to the child process group, zombie reaping,
and a configurable forced shutdown timeout.

## Why `psi`

When your process is PID1 inside a container, Linux changes signal and child
reaping semantics. Many applications aren’t written with PID1 behavior in mind,
and that can lead to stuck shutdowns or zombie processes. `psi` makes PID1 act
like a minimal init while keeping your app unchanged.

## Behavior

- **Non-PID1:** runs your `submain` directly
- **PID1:** re-execs itself as a child and:
  - forwards signals to the child process group
  - reaps zombies
  - enforces a forced shutdown timeout
  - makes the child the foreground process group on the controlling TTY, so
    interactive reads work correctly

## Usage

Wrap your existing `main` in a `submain`:

```go
package main

import (
    "context"
    "os"

    "pkt.systems/psi"
)

func submain(ctx context.Context) int {
    // Your old main logic.
    // ctx is cancelled on SIGTERM/SIGINT/SIGQUIT/SIGHUP.
    return 0
}

func main() {
    psi.Run(submain)
}
```

## Configuration

`PSI_STOP_TIMEOUT` controls how long PSI waits after the first terminate-like
signal before sending `SIGKILL` to the child’s process group.

Examples:

- `PSI_STOP_TIMEOUT=30s` (default)
- `PSI_STOP_TIMEOUT=1m15s`
- `PSI_STOP_TIMEOUT=10` (interpreted as seconds)

## Example container

The `example/` module builds a scratch container with `psi` as PID1.

Build and run:

```sh
make -C example build
make -C example run
```

Interactive demo (TTY foreground behavior):

```sh
make -C example run-interactive
```

Or directly:

```sh
podman run -ti --rm localhost/psi-example:latest interactive
```

## Notes

- `psi` forwards signals to the **child’s process group**. If your app spawns
  subprocesses, they will receive signals as expected.
- Interactive reads from a TTY are supported by setting the child’s process
  group as the foreground process group for the controlling TTY.
