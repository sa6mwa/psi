package ttygrp

// GetForegroundPgrp returns the foreground process group ID for the TTY.
func GetForegroundPgrp(fd int) (int, error) {
	return getForegroundPgrp(fd)
}

// SetForegroundPgrp sets the foreground process group ID for the TTY.
func SetForegroundPgrp(fd int, pgid int) error {
	return setForegroundPgrp(fd, pgid)
}
