package sandbox

import (
	"fmt"
	"os"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/claudecode"
)

// The lines that install Claude Code on a sandbox. cloud-init.yaml and
// scripts/prepare-machine.sh carry the same lines, and
// TestProvisioningInstallsNativeClaudeCode keeps them in step.
const (
	// claudeInstallScript installs the stable release of the native Claude
	// Code in agent's home. It downloads the installer to a file first: in a
	// pipe to bash, a failed download runs an empty script, which exits 0.
	// The `stable` argument also writes "autoUpdatesChannel": "stable" to
	// ~/.claude/settings.json, so the self-updater stays on stable.
	claudeInstallScript = `curl -fsSL -o claude-install.sh https://claude.ai/install.sh && bash claude-install.sh stable && rm claude-install.sh`

	// claudeInstallLine runs claudeInstallScript as agent, so agent owns
	// Claude Code and it can update itself. It runs as root.
	claudeInstallLine = `su - agent -c '` + claudeInstallScript + `'`

	// claudeLinkLine puts that binary on the spekk-agent service's PATH. The
	// systemd default PATH has /usr/local/bin before /usr/bin, where an npm
	// copy of Claude Code lives.
	claudeLinkLine = "ln -sf /home/agent/.local/bin/claude /usr/local/bin/claude"

	// claudeVersionLine prints the version of the claude that the
	// spekk-agent service runs. It runs in a transient unit with the
	// service's User= and EnvironmentFile=, so it has the service's PATH: the
	// systemd default, or a PATH that agent.env sets.
	// TestClaudeVersionLineMatchesAgentUnit keeps it in step with
	// spekkAgentUnit.
	claudeVersionLine = `systemd-run --quiet --pipe --wait --collect -p User=agent -p EnvironmentFile=/etc/spekk/agent.env /bin/sh -c 'exec claude --version'`
)

// ensureClaudeCode leaves Claude Code on the sandbox at or above
// claudecode.MinVersion, and installs the stable release when it is older or
// its version cannot be read. A deploy runs it before it restarts the agent.
// Only a version that was read and is below the minimum after the install
// stops the deploy. An unreadable version gives a warning: a later Claude
// Code can print its version in a new form, and the client also runs turns
// when it cannot read the version. It stops no claude process: the
// installer writes a new version file and moves the link, and a process
// that runs keeps its old file.
func ensureClaudeCode(ip, keyPath, name, user string) error {
	version, err := sandboxClaudeVersion(ip, keyPath, name, user)
	if err == nil && !claudecode.BelowMinimum(version) {
		fmt.Fprintf(os.Stderr, "Claude Code %s (minimum %s).\n", version, claudecode.MinVersion)
		return nil
	}
	found := version
	if err != nil {
		found = "version not read (" + err.Error() + ")"
	}
	fmt.Fprintf(os.Stderr, "Claude Code %s, minimum %s. Installing the stable release...\n", found, claudecode.MinVersion)

	script := "set -e\n" + claudeInstallLine + "\n" + claudeLinkLine
	if out, err := runSSHCombined(ip, keyPath, name, user, privilegedScript(user, script)); err != nil {
		return fmt.Errorf("installing Claude Code: %s\n%s", err, out)
	}

	version, err = sandboxClaudeVersion(ip, keyPath, name, user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: cannot read the Claude Code version after the install: %v. The deploy continues.\n", err)
		return nil
	}
	if claudecode.BelowMinimum(version) {
		return fmt.Errorf("Claude Code is %s after the install, and the minimum is %s", version, claudecode.MinVersion)
	}
	fmt.Fprintf(os.Stderr, "Claude Code %s (minimum %s).\n", version, claudecode.MinVersion)
	return nil
}

// sandboxClaudeVersion reads the version of the claude that the spekk-agent
// service runs.
func sandboxClaudeVersion(ip, keyPath, name, user string) (string, error) {
	out, err := runSSHCombined(ip, keyPath, name, user, privilegedScript(user, claudeVersionLine))
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, strings.TrimSpace(out))
	}
	return claudecode.ParseVersion(out)
}
