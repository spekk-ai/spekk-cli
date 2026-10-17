package main

import (
	"os"
	"path/filepath"
)

// The states of the model credential of the agent, which each heartbeat
// reports in its claude_auth field (specs/sandbox-machine-mode/).
const (
	claudeAuthReady          = "ready"
	claudeAuthSignInRequired = "sign_in_required"
)

// claudeAuthEnvVars each give Claude Code a model credential, or select a
// cloud provider that has its own.
var claudeAuthEnvVars = []string{
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"CLAUDE_CODE_OAUTH_TOKEN",
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_VERTEX",
	"CLAUDE_CODE_USE_FOUNDRY",
}

// signInRequiredDetail ends a turn on a sandbox with no model credential. It
// says what to do, because the person who sent the turn is the person who
// must sign in.
const signInRequiredDetail = "Claude Code is not signed in on this sandbox. Sign in on the sandbox with `claude` as the agent user, then send the message again."

// claudeAuthState reports whether Claude Code has a model credential: a
// variable of the environment, or the file that Claude Code's own login
// writes. A sandbox launched with no model credential has neither until the
// person signs in. It reads them again on each call, so a sign-in takes
// effect with no restart. A sign-in file that is not valid counts as one:
// Claude Code reports that itself, better than a guess here.
func claudeAuthState() string {
	for _, v := range claudeAuthEnvVars {
		if os.Getenv(v) != "" {
			return claudeAuthReady
		}
	}
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return claudeAuthSignInRequired
		}
		dir = filepath.Join(home, ".claude")
	}
	if _, err := os.Stat(filepath.Join(dir, ".credentials.json")); err == nil {
		return claudeAuthReady
	}
	return claudeAuthSignInRequired
}
