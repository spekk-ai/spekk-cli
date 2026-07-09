package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// LoopConfig holds configuration for RunBuilderLoop.
type LoopConfig struct {
	Watch       bool
	IdleTimeout int      // seconds; 0 disables idle timeout
	Skills      []string // post-build skill names to run after all assertions complete
	InstallDir  string   // spekk install directory (needed for skill resolution)
}

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

// formatLoopComplete returns the appropriate completion message based on assertion count.
func formatLoopComplete(completed int64) string {
	if completed == 0 {
		return "No assertions to work on."
	}
	return fmt.Sprintf("Builder loop complete. %d assertions completed.", completed)
}

// logLoopComplete prints the loop completion message.
func logLoopComplete(completed int64) {
	colorLog(colorGreen, formatLoopComplete(completed))
}

// RunBuilderLoop runs the continuous builder loop.
// cfg.Watch: poll every 5s when all assertions are complete instead of exiting.
// cfg.IdleTimeout: kill stuck builders after this many seconds of no output (0=disabled, default 120).
func RunBuilderLoop(installDir string, cfg LoopConfig) {
	if cfg.Watch {
		colorLog(colorCyan, "Starting Builder Loop (watch mode)...")
	} else {
		colorLog(colorCyan, "Starting Builder Loop...")
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	colorLog(colorBlue, "This will continuously get next assertions and implement them.")
	colorLog(colorYellow, "Press Ctrl+C to exit gracefully.")

	var completed int64

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logLoopComplete(atomic.LoadInt64(&completed))
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
			if cfg.Watch {
				colorLog(colorBlue, "All assertions complete. Watching for new work (every 5s)...")
				time.Sleep(5 * time.Second)
				continue
			}
			c := atomic.LoadInt64(&completed)
			logLoopComplete(c)
			if len(cfg.Skills) > 0 && c > 0 {
				runPostBuildSkills(cfg)
			}
			return
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

		// Launch builder agent
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

		claudeArgs := []string{"--dangerously-skip-permissions", message}
		var success bool
		var launchErr error

		if cfg.IdleTimeout > 0 {
			var timedOut bool
			success, timedOut, launchErr = launchClaudeWithIdleTimeout(
				claudeArgs, nil,
				time.Duration(cfg.IdleTimeout)*time.Second,
			)
			if timedOut {
				colorLog(colorYellow, "Builder was idle too long, moving on...")
			}
		} else {
			success, launchErr = launchClaude(claudeArgs, nil)
		}

		if launchErr != nil {
			colorLog(colorRed, "Builder agent failed: "+launchErr.Error())
			os.Exit(1)
		}

		if success {
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
}

// runPostBuildSkills runs skill names sequentially after the builder loop completes.
func runPostBuildSkills(cfg LoopConfig) {
	colorLog(colorCyan, "\n--- Post-Build Skills ---")

	results := make([]bool, len(cfg.Skills))
	succeeded := 0

	for i, skill := range cfg.Skills {
		// Show checklist with current skill in progress
		for j, s := range cfg.Skills {
			if j < i {
				if results[j] {
					colorLog(colorGreen, fmt.Sprintf("[x] %s", s))
				} else {
					colorLog(colorRed, fmt.Sprintf("[!] %s", s))
				}
			} else if j == i {
				colorLog(colorYellow, fmt.Sprintf("[ ] %s (running...)", s))
			} else {
				colorLog(colorBlue, fmt.Sprintf("[ ] %s", s))
			}
		}

		opts := LaunchOptions{
			Agent:      "builder",
			InstallDir: cfg.InstallDir,
		}
		message, err := BuildActivationMessage(opts)
		if err != nil {
			colorLog(colorRed, fmt.Sprintf("Skill %s: failed to build message: %s", skill, err))
			continue
		}

		skillMsg, err := BuildSkillMessage(cfg.InstallDir, "builder", skill, nil)
		if err != nil {
			colorLog(colorRed, fmt.Sprintf("Skill %s: %s", skill, err))
			continue
		}
		if skillMsg == "" {
			colorLog(colorRed, fmt.Sprintf("Skill %s: not found", skill))
			continue
		}
		message += skillMsg

		claudeArgs := []string{"--dangerously-skip-permissions", message}
		var success bool
		var launchErr error

		if cfg.IdleTimeout > 0 {
			var timedOut bool
			success, timedOut, launchErr = launchClaudeWithIdleTimeout(
				claudeArgs, nil,
				time.Duration(cfg.IdleTimeout)*time.Second,
			)
			if timedOut {
				colorLog(colorYellow, fmt.Sprintf("Skill %s: timed out (idle too long)", skill))
				continue
			}
		} else {
			success, launchErr = launchClaude(claudeArgs, nil)
		}

		if launchErr != nil {
			colorLog(colorRed, fmt.Sprintf("Skill %s: failed: %s", skill, launchErr))
			continue
		}

		if success {
			results[i] = true
			succeeded++
			colorLog(colorGreen, fmt.Sprintf("Skill %s: completed", skill))
		} else {
			colorLog(colorRed, fmt.Sprintf("Skill %s: exited with error", skill))
		}
	}

	// Final checklist
	for i, s := range cfg.Skills {
		if results[i] {
			colorLog(colorGreen, fmt.Sprintf("[x] %s", s))
		} else {
			colorLog(colorRed, fmt.Sprintf("[!] %s", s))
		}
	}

	colorLog(colorCyan, fmt.Sprintf("Post-build skills: %d/%d completed", succeeded, len(cfg.Skills)))
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
