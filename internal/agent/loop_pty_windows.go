//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// terminateChild stops the builder child on Ctrl+C. On Windows the child shares
// the console (no PTY/session), so a plain Kill is sufficient.
func terminateChild(p *os.Process) {
	if p == nil {
		return
	}
	_ = p.Kill()
}

// launchClaudeWithPTY on Windows runs Claude with inherited stdio.
// Idle timeout is not supported on Windows (requires PTY for activity tracking).
// The started process is registered in holder (if non-nil) so the caller's
// signal handler can stop it on Ctrl+C.
func launchClaudeWithPTY(claudeArgs []string, idleTimeout time.Duration, holder *processHolder) (bool, bool, error) {
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

	if holder != nil {
		holder.set(cmd.Process)
		defer holder.set(nil)
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
