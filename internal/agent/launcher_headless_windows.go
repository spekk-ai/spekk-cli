//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
)

// LaunchHeadless spawns `claude -p` (print/non-interactive mode) with the given
// message. It does not inherit stdin and does not forward signals — suitable for
// background invocations where no TTY is present.
//
// On Windows the overlap guard (flock) is not available; the lockFile parameter
// is accepted for API compatibility but ignored.
func LaunchHeadless(claudePath, lockFile, message string) error {
	if claudePath == "" {
		claudePath = "claude"
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

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}
