//go:build !windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// LaunchHeadless spawns `claude -p` (print/non-interactive mode) with the given
// message. It does not inherit stdin and does not forward signals — suitable for
// cron jobs where no TTY is present.
//
// claudePath is the absolute path to the claude binary; if empty, "claude" is
// used (relies on PATH).
//
// lockFile is an optional project-scoped lock file path. When non-empty,
// LaunchHeadless acquires an exclusive non-blocking flock on the file before
// launching Claude. If the lock cannot be acquired (another instance is already
// running), it exits 0 silently. The lock is released automatically when the
// process exits; the fd is kept alive for the lifetime of the call.
func LaunchHeadless(claudePath, lockFile, message string) error {
	if claudePath == "" {
		claudePath = "claude"
	}

	var lockF *os.File
	if lockFile != "" {
		f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0o644)
		if err != nil {
			return fmt.Errorf("opening lock file %s: %w", lockFile, err)
		}
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			// Another instance is already running — exit 0 silently.
			os.Exit(0)
		}
		lockF = f
	}

	cmd := exec.Command(claudePath, "-p", "--dangerously-skip-permissions", message)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			fmt.Fprintln(os.Stderr, "Error: Claude Code CLI not found. Please install Claude Code first.")
			fmt.Fprintln(os.Stderr, "Visit: https://claude.ai/code for installation instructions.")
			os.Exit(1)
		}
		return fmt.Errorf("Error launching Claude Code headless: %w", err)
	}

	err := cmd.Wait()
	runtime.KeepAlive(lockF) // keep fd open until Claude exits
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}
