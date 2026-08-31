//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// launchClaudeWithPTY on Windows runs Claude with inherited stdio.
// Idle timeout is not supported on Windows (requires PTY for activity tracking).
func launchClaudeWithPTY(claudeArgs []string, idleTimeout time.Duration) (bool, bool, error) {
	if idleTimeout > 0 {
		colorLog(colorYellow, "Idle timeout not supported on Windows (requires PTY). Running without timeout.")
	}

	cmd := exec.Command("claude", claudeArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			colorLog(colorRed, "Error: Claude Code CLI not found. Please install Claude Code first.")
			colorLog(colorBlue, "Visit: https://claude.ai/code for installation instructions.")
			os.Exit(1)
		}
		return false, false, fmt.Errorf("error launching Claude: %w", err)
	}

	waitErr := cmd.Wait()

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			colorLog(colorYellow, fmt.Sprintf("Claude Code exited with code %d", exitErr.ExitCode()))
			return false, false, nil
		}
		return false, false, waitErr
	}

	return true, false, nil
}
