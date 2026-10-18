package main

import (
	"os"
	"path/filepath"
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

// claudeSignedIn reports whether Claude Code has a model credential: a
// variable of the environment, or the file that Claude Code's own login
// writes. A sandbox launched with no model credential has neither until the
// person signs in. It checks them again on each call, so a sign-in takes
// effect with no restart. The sign-in file counts when it is a regular file
// that the agent can open and that is not empty. The check opens the file and
// reads none of it: the agent never holds the credential of the person. A
// file that is not valid still counts: Claude Code reports that itself,
// better than a guess here. Each heartbeat reports it as claude_signed_in
// (specs/sandbox-machine-mode/, specs/sandbox-sign-in-terminal/).
func claudeSignedIn() bool {
	for _, v := range claudeAuthEnvVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		dir = filepath.Join(home, ".claude")
	}
	f, err := os.Open(filepath.Join(dir, ".credentials.json"))
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}
