package sandbox

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
)
