//go:build !windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty/v2"
)

// launchClaudeWithPTY spawns claude inside a pseudo-terminal for idle timeout detection.
// Returns (success, timedOut, error). When timedOut is true, the process was killed
// due to inactivity and the caller should reset the assertion status.
func launchClaudeWithPTY(claudeArgs []string, idleTimeout time.Duration) (bool, bool, error) {
	cmd := exec.Command("claude", claudeArgs...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		if isNotFound(err) {
			colorLog(colorRed, "Error: Claude Code CLI not found. Please install Claude Code first.")
			colorLog(colorBlue, "Visit: https://claude.ai/code for installation instructions.")
			os.Exit(1)
		}
		return false, false, fmt.Errorf("error launching Claude with PTY: %w", err)
	}
	defer ptmx.Close()

	// Propagate terminal size from parent terminal to PTY
	_ = pty.InheritSize(os.Stdin, ptmx)
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	go func() {
		for range sigwinch {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	defer func() {
		signal.Stop(sigwinch)
		close(sigwinch)
	}()

	// Track last output activity (unix nanoseconds)
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())

	// Copy PTY output to stdout, tracking activity on each read
	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				lastActivity.Store(time.Now().UnixNano())
				os.Stdout.Write(buf[:n])
			}
			if readErr != nil {
				return
			}
		}
	}()

	// No stdin forwarding — the loop builder runs autonomously with
	// --dangerously-skip-permissions so Claude doesn't need user input.
	// Forwarding stdin here causes the goroutine to block between iterations,
	// wedging the loop and requiring Ctrl+C to continue.

	// Idle timeout monitor
	var timeoutFired atomic.Bool
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				last := time.Unix(0, lastActivity.Load())
				if time.Since(last) >= idleTimeout {
					timeoutFired.Store(true)
					colorLog(colorYellow, fmt.Sprintf("\nBuilder idle for %ds. Force-stopping...", int(idleTimeout.Seconds())))
					cmd.Process.Signal(syscall.SIGTERM)
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
