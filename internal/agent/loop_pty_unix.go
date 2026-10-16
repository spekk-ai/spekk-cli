//go:build !windows

package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty/v2"
)

const (
	// doneMarker is the completion signal the builder agent prints when it
	// finishes work (configured in ~/.config/spekk/builder.prompt.md).
	// When detected, the loop gives the process a short grace period to
	// exit cleanly before force-stopping it.
	doneMarker      = "[SPEKK_DONE]"
	doneMarkerGrace = 5 * time.Second
)

// sigkillAfter waits 5 seconds after SIGTERM, then sends SIGKILL if the
// process is still alive. Aborts if done is closed first.
func sigkillAfter(proc *os.Process, done <-chan struct{}) {
	select {
	case <-done:
		return
	case <-time.After(5 * time.Second):
		proc.Signal(syscall.SIGKILL)
	}
}

// terminateChild stops a builder child (started under its own session via the
// PTY) and everything it spawned. It signals the whole process group so no
// subprocess is orphaned, escalating from SIGTERM to SIGKILL after a grace
// period. Called from the loop's Ctrl+C handler before the process exits.
func terminateChild(p *os.Process) {
	if p == nil {
		return
	}
	// pty.Start puts the child in a new session, so its PGID equals its PID;
	// a negative PID targets the whole group.
	syscall.Kill(-p.Pid, syscall.SIGTERM)
	time.Sleep(2 * time.Second)
	syscall.Kill(-p.Pid, syscall.SIGKILL)
}

// markerAtLineStart reports whether marker appears at the start of a line in
// data — preceded by a newline, or at the very start of the stream. Requiring a
// line boundary avoids false completions when the marker string shows up inline
// within other builder output (e.g. echoed file contents).
func markerAtLineStart(data, marker []byte, atStreamStart bool) bool {
	from := 0
	for {
		rel := bytes.Index(data[from:], marker)
		if rel < 0 {
			return false
		}
		pos := from + rel
		if pos == 0 {
			if atStreamStart {
				return true
			}
		} else if data[pos-1] == '\n' {
			return true
		}
		from = pos + 1
	}
}

// launchClaudeWithPTY spawns claude inside a pseudo-terminal for idle timeout detection.
// Returns (success, timedOut, error). When timedOut is true, the process was killed
// due to inactivity and the caller should reset the assertion status. The started
// process is registered in holder (if non-nil) so the caller's signal handler can
// stop it on Ctrl+C.
func launchClaudeWithPTY(claudeArgs []string, idleTimeout time.Duration, holder *processHolder) (bool, bool, error) {
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

	if holder != nil {
		holder.set(cmd.Process)
		defer holder.set(nil)
	}

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

	var timeoutFired atomic.Bool
	var markerDetected atomic.Bool
	done := make(chan struct{})

	// Copy PTY output to stdout, tracking activity on each read.
	// Also detect the done marker so we can force-stop a hung process.
	// A rolling tail buffer handles the marker being split across reads.
	go func() {
		buf := make([]byte, 4096)
		markerSeen := false
		markerBytes := []byte(doneMarker)
		// Retain the last len(markerBytes) bytes so a marker — and the newline
		// that must precede it — can be detected even when split across reads.
		tail := make([]byte, 0, len(markerBytes))
		firstRead := true
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				lastActivity.Store(time.Now().UnixNano())
				os.Stdout.Write(buf[:n])

				if !markerSeen {
					// Check combined tail+current for a marker spanning reads.
					combined := make([]byte, 0, len(tail)+n)
					combined = append(combined, tail...)
					combined = append(combined, buf[:n]...)
					if markerAtLineStart(combined, markerBytes, firstRead) {
						markerSeen = true
						markerDetected.Store(true)
						go func() {
							select {
							case <-done:
								return
							case <-time.After(doneMarkerGrace):
								colorLog(colorYellow, "\nDone marker seen but process still running. Force-stopping...")
								cmd.Process.Signal(syscall.SIGTERM)
								sigkillAfter(cmd.Process, done)
							}
						}()
					}
					// Keep the last len(markerBytes) bytes as tail for next read.
					if len(combined) > len(markerBytes) {
						tail = append(tail[:0], combined[len(combined)-len(markerBytes):]...)
					} else {
						tail = append(tail[:0], combined...)
					}
				}
				firstRead = false
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
					sigkillAfter(cmd.Process, done)
					return
				}
			}
		}
	}()

	waitErr := cmd.Wait()
	close(done)

	// Done marker means the builder completed its work — treat as success
	// even if the process exited uncleanly (it was force-stopped).
	if markerDetected.Load() {
		return true, false, nil
	}

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
