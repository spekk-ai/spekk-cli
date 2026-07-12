//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

// launchClaudeWithPTY on Windows falls back to a simple exec without PTY support.
// Idle timeout is not supported on Windows.
func launchClaudeWithPTY(claudeArgs []string, idleTimeout time.Duration) (bool, bool, error) {
	cmd := exec.Command("claude", claudeArgs...)
	cmd.Stdin = os.Stdin
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

	// Idle timeout monitor (no PTY output tracking, uses wall-clock only)
	var timeoutFired atomic.Bool
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		start := time.Now()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if time.Since(start) >= idleTimeout {
					timeoutFired.Store(true)
					colorLog(colorYellow, fmt.Sprintf("\nBuilder idle for %ds. Force-stopping...", int(idleTimeout.Seconds())))
					cmd.Process.Kill()
					return
				}
			}
		}
	}()

	waitErr := cmd.Wait()
	close(done)

	if timeoutFired.Load() {
		return false, true, nil
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			colorLog(colorYellow, fmt.Sprintf("Claude Code exited with code %d", exitErr.ExitCode()))
			return false, false, nil
		}
		return false, false, waitErr
	}

	return true, false, nil
}
