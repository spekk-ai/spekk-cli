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
	// claudeInstallLine installs the stable release of the native Claude
	// Code as agent, so agent owns it and it can update itself. It runs as
	// root.
	claudeInstallLine = `su - agent -c 'curl -fsSL https://claude.ai/install.sh | bash -s stable'`

	// claudeLinkLine puts that binary on the spekk-agent service's PATH. The
	// systemd default PATH has /usr/local/bin before /usr/bin, where an npm
	// copy of Claude Code lives.
	claudeLinkLine = "ln -sf /home/agent/.local/bin/claude /usr/local/bin/claude"

	// claudeVersionLine prints the version of the claude that the
	// spekk-agent service runs: as agent, with the systemd default PATH.
	claudeVersionLine = `su - agent -c 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin claude --version'`
)

// ensureClaudeCode leaves Claude Code on the sandbox at or above
// claudecode.MinVersion, and installs the stable release when it is older or
// not found. A deploy runs it before it restarts the agent. It stops no
// claude process: the installer writes a new version file and moves the
// link, and a process that runs keeps its old file.
func ensureClaudeCode(ip, keyPath, name, user string) error {
	version, err := sandboxClaudeVersion(ip, keyPath, name, user)
	if err == nil && !claudecode.BelowMinimum(version) {
		fmt.Fprintf(os.Stderr, "Claude Code %s (minimum %s).\n", version, claudecode.MinVersion)
		return nil
	}
	found := version
	if err != nil {
		found = "not found (" + err.Error() + ")"
	}
	fmt.Fprintf(os.Stderr, "Claude Code %s, minimum %s. Installing the stable release...\n", found, claudecode.MinVersion)

	script := "set -e\n" + claudeInstallLine + "\n" + claudeLinkLine
	if out, err := runSSHCombined(ip, keyPath, name, user, privilegedScript(user, script)); err != nil {
		return fmt.Errorf("installing Claude Code: %s\n%s", err, out)
	}

	version, err = sandboxClaudeVersion(ip, keyPath, name, user)
	if err != nil {
		return fmt.Errorf("Claude Code is not found after the install: %w", err)
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
