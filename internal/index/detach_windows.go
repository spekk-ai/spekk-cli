//go:build windows

package index

import "os/exec"

// detach does nothing on Windows: a started process already outlives the
// process that started it.
func detach(cmd *exec.Cmd) {}
