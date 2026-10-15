---
id: sandbox-installs-native-claude-code
parent: sandbox-claude-code-version
created: 2026-10-15T21:30:00Z
priority: 1
status: not_started
branch: fix/claude-code-version
depends-on: claude-code-minimum-version-defined-once
---

# Every Provisioning Path Installs the Native Claude Code as `agent`

## Success Criteria

- `internal/sandbox/cloud-init.yaml` (DigitalOcean and Google Cloud), the AWS CloudFormation template (its UserData is generated from `cloud-init.yaml`), and `scripts/prepare-machine.sh` install Claude Code as the `agent` user with `su - agent -c`. The line downloads the installer to a file with `curl -fsSL -o`, and runs that file with `bash ... stable` only when the download succeeds. A failed download makes the line fail with the curl exit code, and the installer does not run. The `stable` argument also makes the installer write `"autoUpdatesChannel": "stable"` to `~agent/.claude/settings.json`, so the self-updater stays on `stable`.
- Each path links `/usr/local/bin/claude` to `/home/agent/.local/bin/claude`, and only after the install line succeeds, so a failed install leaves no link to a missing file. The `spekk-agent` service has the systemd default `PATH`, so it finds this link before an npm copy in `/usr/bin`.
- No path runs `npm install -g @anthropic-ai/claude-code`.
- The install line and the link line are each defined once in Go, and the deploy upgrade (`deploy-upgrades-outdated-claude-code`) uses the same lines. A test fails when `cloud-init.yaml` or `prepare-machine.sh` does not contain them, or contains the npm install. A test runs the install command with a `curl` that fails, and the command fails with that exit code, and the installer does not run.
- `TestAWSUserDataMatchesCloudInit` passes, so the CloudFormation template carries the same lines.

**Tests:** `internal/sandbox/claude_code_test.go` (`TestProvisioningInstallsNativeClaudeCode`), `internal/sandbox/aws_cloudformation_test.go` (`TestAWSUserDataMatchesCloudInit`)
