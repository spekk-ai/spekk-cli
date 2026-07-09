package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty/v2"
	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// gitStageAndCommit stages all changes and commits with the given message.
// Returns true if a commit was created, false if nothing to commit.
func gitStageAndCommit(message string) (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	if strings.TrimSpace(string(out)) == "" {
		return false, nil
	}

	if err := exec.Command("git", "add", ".").Run(); err != nil {
		return false, fmt.Errorf("git add failed: %w", err)
	}

	if err := exec.Command("git", "commit", "-m", message).Run(); err != nil {
		return false, fmt.Errorf("git commit failed: %w", err)
	}

	return true, nil
}

// gitStageSpecsAndCommit stages spec-related changes and commits.
func gitStageSpecsAndCommit(message string) (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	status := string(out)
	if strings.TrimSpace(status) == "" {
		return false, nil
	}

	if !strings.Contains(status, "specs/") {
		return false, nil
	}

	if err := exec.Command("git", "add", "specs/").Run(); err != nil {
		return false, fmt.Errorf("git add failed: %w", err)
	}

	if err := exec.Command("git", "commit", "-m", message).Run(); err != nil {
		return false, fmt.Errorf("git commit failed: %w", err)
	}

	return true, nil
}

// completionMessage returns the appropriate message for the given assertion count.
func completionMessage(count int64) string {
	if count == 0 {
		return "No assertions to work on."
	}
	return fmt.Sprintf("Builder loop complete. %d assertions completed.", count)
}

// LoopFlags defines the flag set for the loop builder CLI.
var LoopFlags = cli.FlagSet{
	"watch":       {Names: []string{"--watch", "-w"}, Type: cli.BoolFlag},
	"idleTimeout": {Names: []string{"--idle-timeout"}, Type: cli.StringFlag},
}

// extractAllPositionalArgs returns all positional (non-flag) arguments
// from the args list, skipping known flags and their values.
func extractAllPositionalArgs(args []string, flags cli.FlagSet) []string {
	flagStrings := make(map[string]bool)
	stringFlags := make(map[string]bool)

	for _, def := range flags {
		for _, f := range def.Names {
			flagStrings[f] = true
			if def.Type == cli.StringFlag {
				stringFlags[f] = true
			}
		}
	}

	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if flagStrings[arg] {
			if stringFlags[arg] {
				i++ // skip value
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		positional = append(positional, arg)
	}
	return positional
}

// skillsSummary returns the post-build skills summary line.
func skillsSummary(succeeded, total int) string {
	return fmt.Sprintf("Post-build skills: %d/%d completed", succeeded, total)
}

// RunBuilderLoop runs the continuous builder loop.
func RunBuilderLoop(args []string, installDir string) {
	parsed := cli.ParseFlags(args, LoopFlags)
	watch := parsed.Bool("watch")
	skills := extractAllPositionalArgs(args, LoopFlags)

	idleTimeout := 120
	if s := parsed.String("idleTimeout"); s != "" {
		val, err := strconv.Atoi(s)
		if err != nil || val <= 0 {
			colorLog(colorRed, "Error: --idle-timeout must be a positive integer (seconds)")
			os.Exit(1)
		}
		idleTimeout = val
	}

	colorLog(colorCyan, "Starting Builder Loop...")
	colorLog(colorBlue, "This will continuously get next assertions and implement them.")
	if watch {
		colorLog(colorYellow, "Watch mode: will poll for new work after completion.")
	}
	if len(skills) > 0 {
		colorLog(colorBlue, fmt.Sprintf("Post-build skills: %s", strings.Join(skills, ", ")))
	}
	colorLog(colorBlue, fmt.Sprintf("Idle timeout: %ds", idleTimeout))
	colorLog(colorYellow, "Press Ctrl+C to exit gracefully.")

	var completed int64

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		count := atomic.LoadInt64(&completed)
		colorLog(colorGreen, "\n"+completionMessage(count))
		os.Exit(0)
	}()

	spekkBin := findSpekkBin()
	iteration := 0

	for {
		iteration++
		colorLog(colorBright, fmt.Sprintf("\n--- Builder Loop Iteration %d ---", iteration))

		// Get next assertion
		colorLog(colorBlue, "Getting next priority assertion...")

		result, err := getNextAssertionForLoop(spekkBin)
		if err != nil {
			colorLog(colorYellow, "Parser error (transient). Retrying in 5s...")
			colorLog(colorYellow, "   "+err.Error())
			time.Sleep(5 * time.Second)
			continue
		}

		if result.Type == "complete" {
			count := atomic.LoadInt64(&completed)
			colorLog(colorGreen, completionMessage(count))
			if watch {
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}

		if result.Type != "assertion" {
			colorLog(colorYellow, "Unexpected result type from parser. Retrying in 5s...")
			time.Sleep(5 * time.Second)
			continue
		}

		colorLog(colorGreen, fmt.Sprintf("Working on: %s (%s)", result.ID, result.Title))
		colorLog(colorBlue, fmt.Sprintf("   File: %s", result.File))
		colorLog(colorBlue, fmt.Sprintf("   Status: %s", result.Status))
		colorLog(colorBlue, fmt.Sprintf("   Priority: %d", result.Priority))

		// Launch builder agent with PTY and idle timeout
		colorLog(colorMagenta, "Launching Builder Agent...")

		opts := LaunchOptions{
			Agent:      "builder",
			InstallDir: installDir,
		}
		message, err := BuildActivationMessage(opts)
		if err != nil {
			colorLog(colorRed, fmt.Sprintf("Error building activation message: %s", err))
			os.Exit(1)
		}

		success, timedOut, launchErr := launchClaudeWithPTY(
			[]string{"--dangerously-skip-permissions", message},
			time.Duration(idleTimeout)*time.Second,
		)

		if launchErr != nil {
			colorLog(colorRed, "Builder agent failed: "+launchErr.Error())
			os.Exit(1)
		}

		if timedOut {
			colorLog(colorYellow, "Resetting assertion status to not_started...")
			if resetErr := resetAssertionStatus(result.File); resetErr != nil {
				colorLog(colorRed, "Failed to reset assertion status: "+resetErr.Error())
			}
		} else if success {
			atomic.AddInt64(&completed, 1)
			colorLog(colorGreen, "Builder agent completed work")
		} else {
			colorLog(colorRed, "Builder agent exited with error")
		}

		// Commit changes
		colorLog(colorBlue, "Committing changes...")
		commitMsg := fmt.Sprintf("Complete %s\n\n%s", result.ID, result.Title)
		committed, commitErr := gitStageAndCommit(commitMsg)
		if commitErr != nil {
			colorLog(colorRed, "Git operations failed: "+commitErr.Error())
		} else if committed {
			colorLog(colorGreen, "Changes committed successfully")
		} else {
			colorLog(colorYellow, "No changes to commit")
		}

		// Brief pause
		colorLog(colorBlue, "Preparing for next iteration...")
		time.Sleep(500 * time.Millisecond)
	}

	// Post-build skills pipeline
	count := atomic.LoadInt64(&completed)
	if len(skills) > 0 && count > 0 {
		runPostBuildSkills(skills, installDir, idleTimeout)
	}
}

// runPostBuildSkills launches each skill as a separate builder agent invocation
// after all assertions have been completed.
func runPostBuildSkills(skills []string, installDir string, idleTimeout int) {
	colorLog(colorCyan, "\n--- Post-Build Skills Pipeline ---")

	done := make(map[string]bool)
	succeeded := 0

	for _, skill := range skills {
		// Display current checklist state
		for _, s := range skills {
			if done[s] {
				colorLog(colorGreen, fmt.Sprintf("[x] %s", s))
			} else if s == skill {
				colorLog(colorYellow, fmt.Sprintf("[ ] %s", s))
			} else {
				colorLog(colorBlue, fmt.Sprintf("[ ] %s", s))
			}
		}

		// Build activation message with skill content
		opts := LaunchOptions{
			Agent:      "builder",
			InstallDir: installDir,
		}
		message, err := BuildActivationMessage(opts)
		if err != nil {
			colorLog(colorRed, fmt.Sprintf("Skill %s: failed to build message: %s", skill, err))
			done[skill] = true
			continue
		}

		skillMsg, err := BuildSkillMessage(installDir, "builder", skill, []string{skill})
		if err != nil {
			colorLog(colorRed, fmt.Sprintf("Skill %s: error: %s", skill, err))
			done[skill] = true
			continue
		}
		if skillMsg == "" {
			colorLog(colorYellow, fmt.Sprintf("Skill %q not found, skipping", skill))
			done[skill] = true
			continue
		}

		message += skillMsg

		success, timedOut, launchErr := launchClaudeWithPTY(
			[]string{"--dangerously-skip-permissions", message},
			time.Duration(idleTimeout)*time.Second,
		)

		done[skill] = true

		if launchErr != nil {
			colorLog(colorRed, fmt.Sprintf("Skill %s failed: %s", skill, launchErr))
		} else if timedOut {
			colorLog(colorYellow, fmt.Sprintf("Skill %s timed out", skill))
		} else if success {
			succeeded++
			colorLog(colorGreen, fmt.Sprintf("Skill %s completed", skill))
		} else {
			colorLog(colorRed, fmt.Sprintf("Skill %s exited with error", skill))
		}
	}

	colorLog(colorGreen, skillsSummary(succeeded, len(skills)))
}

// RunCoachLoop runs the continuous coach loop.
func RunCoachLoop(installDir string) {
	colorLog(colorCyan, "Starting Coach Loop...")
	colorLog(colorBlue, "This will launch the coach agent for interactive spec creation.")
	colorLog(colorYellow, "Press Ctrl+C to exit gracefully.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		colorLog(colorYellow, fmt.Sprintf("\nReceived %s. Exiting gracefully...", sig))
		os.Exit(0)
	}()

	session := 0

	for {
		session++
		colorLog(colorBright, fmt.Sprintf("\n--- Coach Loop Session %d ---", session))

		// Launch coach agent
		colorLog(colorMagenta, "Launching Coach Agent...")

		opts := LaunchOptions{
			Agent:      "coach",
			InstallDir: installDir,
		}
		message, err := BuildActivationMessage(opts)
		if err != nil {
			colorLog(colorRed, fmt.Sprintf("Error building activation message: %s", err))
			os.Exit(1)
		}

		success, launchErr := launchClaude(
			[]string{"--dangerously-skip-permissions", message},
			nil,
		)

		if launchErr != nil {
			colorLog(colorRed, "Coach agent failed: "+launchErr.Error())
			os.Exit(1)
		}

		if success {
			colorLog(colorGreen, "Coach session completed")
		} else {
			colorLog(colorRed, "Coach agent exited with error")
		}

		// Commit spec changes
		colorLog(colorBlue, "Checking for new specs to commit...")
		commitMsg := fmt.Sprintf("Add new specs from coach session %d", session)
		committed, commitErr := gitStageSpecsAndCommit(commitMsg)
		if commitErr != nil {
			colorLog(colorRed, "Git operations failed: "+commitErr.Error())
		} else if committed {
			colorLog(colorGreen, "New specs committed successfully")
		} else {
			colorLog(colorYellow, "No new specs to commit")
		}

		// Brief pause
		time.Sleep(500 * time.Millisecond)
	}
}

// getNextAssertionForLoop calls the parser without filters.
func getNextAssertionForLoop(spekkBin string) (*AssertionResult, error) {
	cmd := exec.Command(spekkBin, "next")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("parser command failed: %w", err)
	}

	var result AssertionResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("malformed JSON from parser: %w", err)
	}
	return &result, nil
}

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

	// Forward stdin to PTY
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

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

// resetAssertionStatus resets an assertion file's status from in_progress to
// not_started and removes the locked-by field. Used when a builder is killed
// by idle timeout so the next iteration can pick it up again.
func resetAssertionStatus(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading assertion file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "status:") && strings.Contains(trimmed, "in_progress") {
			line = strings.Replace(line, "in_progress", "not_started", 1)
		}
		if strings.HasPrefix(trimmed, "locked-by:") {
			continue
		}
		result = append(result, line)
	}

	return os.WriteFile(filePath, []byte(strings.Join(result, "\n")), 0o644)
}
