package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/claudecode"
)

// claudeVersionTimeout caps `claude --version`, which takes about 20 ms.
const claudeVersionTimeout = 10 * time.Second

// claudeCodeRemedy ends each refusal. A deploy upgrades Claude Code before
// it restarts the agent (specs/sandbox-claude-code-version/).
const claudeCodeRemedy = "Run `spekk sandbox deploy` for this sandbox to install a current version."

// checkClaudeCode reads the version of the claude that a turn starts. ok is
// true when the version is at or above claudecode.MinVersion. Otherwise
// status is the reason to refuse the turn, written for the person who
// dispatched it. The control host keeps only the error text of an error
// frame, and a display can cut it at 300 characters, so status is short and
// says everything.
func checkClaudeCode() (status string, ok bool) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Sprintf("Claude Code is not installed on this sandbox: %v. %s", err, claudeCodeRemedy), false
	}
	ctx, cancel := context.WithTimeout(context.Background(), claudeVersionTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return fmt.Sprintf("Cannot read the Claude Code version: %s --version failed: %v. %s", path, err, claudeCodeRemedy), false
	}
	version, err := claudecode.ParseVersion(string(out))
	if err != nil {
		return fmt.Sprintf("Cannot read the Claude Code version: %s --version printed no version. %s", path, claudeCodeRemedy), false
	}
	if claudecode.BelowMinimum(version) {
		return fmt.Sprintf("Claude Code %s at %s is older than %s, the minimum for this sandbox. %s", version, path, claudecode.MinVersion, claudeCodeRemedy), false
	}
	return fmt.Sprintf("Claude Code %s at %s (minimum %s)", version, path, claudecode.MinVersion), true
}
