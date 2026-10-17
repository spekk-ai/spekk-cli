//go:build !windows

package index

import (
	"os/exec"
	"syscall"
)

// detach starts cmd in its own session, so it outlives the query that
// started it and gets no signal meant for the query's terminal.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
