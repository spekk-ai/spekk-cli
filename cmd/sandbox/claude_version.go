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

// claudeCodeCheck is the result of checkClaudeCode.
type claudeCodeCheck int

const (
	claudeCodeCurrent    claudeCodeCheck = iota // at or above the minimum
	claudeCodeUnreadable                        // no version read: the turn runs
	claudeCodeOld                               // below the minimum: the turn is refused
)

// checkClaudeCode reads the version of the claude that a turn starts, and
// returns a status line for the log. Only a version that was read and is
// below claudecode.MinVersion gives claudeCodeOld. When no version can be
// read, the check fails open: a later Claude Code can print its version in
// a new form, and a refusal would then stop every turn on every sandbox. For
// claudeCodeOld, status is the refusal, written for the person who
// dispatched the turn. The control host keeps only the error text of an
// error frame, and a display can cut it at 300 characters, so status is
// short and says everything.
func checkClaudeCode() (status string, check claudeCodeCheck) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Sprintf("Claude Code is not on the PATH: %v.", err), claudeCodeUnreadable
	}
	ctx, cancel := context.WithTimeout(context.Background(), claudeVersionTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return fmt.Sprintf("Cannot read the Claude Code version: %s --version failed: %v.", path, err), claudeCodeUnreadable
	}
	version, err := claudecode.ParseVersion(string(out))
	if err != nil {
		return fmt.Sprintf("Cannot read the Claude Code version: %s --version printed no version that this client knows.", path), claudeCodeUnreadable
	}
	if claudecode.BelowMinimum(version) {
		return fmt.Sprintf("Claude Code %s at %s is older than %s, the minimum for this sandbox. %s", version, path, claudecode.MinVersion, claudeCodeRemedy), claudeCodeOld
	}
	return fmt.Sprintf("Claude Code %s at %s (minimum %s)", version, path, claudecode.MinVersion), claudeCodeCurrent
}
