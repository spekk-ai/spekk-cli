package sandbox

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/claudecode"
)

// Every path that provisions a sandbox installs the native Claude Code the
// way a deploy does. The npm package gave 2.1.197, which kills a background
// task seconds after the turn's result. The AWS template is generated from
// cloud-init.yaml, and TestAWSUserDataMatchesCloudInit covers it.
func TestProvisioningInstallsNativeClaudeCode(t *testing.T) {
	prepare, err := os.ReadFile("../../scripts/prepare-machine.sh")
	if err != nil {
		t.Fatal(err)
	}
	// cloud-init runs each line even when one before it fails, so the link
	// follows the install with &&. prepare-machine.sh runs with set -e.
	for name, tc := range map[string]struct {
		content string
		lines   []string
	}{
		"cloud-init.yaml":    {string(cloudInitTemplate), []string{claudeInstallLine + " && " + claudeLinkLine}},
		"prepare-machine.sh": {string(prepare), []string{"set -euo pipefail", claudeInstallLine, claudeLinkLine}},
	} {
		for _, line := range tc.lines {
			if !strings.Contains(tc.content, line) {
				t.Errorf("%s does not contain %q", name, line)
			}
		}
		if strings.Contains(tc.content, "npm install -g @anthropic-ai/claude-code") {
			t.Errorf("%s installs Claude Code from npm", name)
		}
	}
}

// A failed download of the installer fails the install, and no part of the
// installer runs. In a pipe to bash, a failed download runs an empty script,
// which exits 0, and the link then points to a file that is not there.
func TestClaudeInstallFailsOnFailedDownload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake curl is a shell script")
	}
	dir := t.TempDir()
	ran := filepath.Join(dir, "installer-ran")
	// The fake curl writes part of a script, and then fails as curl -f does
	// on an HTTP error.
	curl := "#!/bin/sh\nout=/dev/stdout\nwhile [ $# -gt 0 ]; do [ \"$1\" = -o ] && out=$2; shift; done\necho \"touch '" + ran + "'\" > \"$out\"\nexit 22\n"
	if err := os.WriteFile(filepath.Join(dir, "curl"), []byte(curl), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", claudeInstallScript)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 22 {
		t.Errorf("install with a failed download: err = %v, want exit status 22", err)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("the installer ran after a failed download")
	}
}

// A deploy reads the Claude Code version before it restarts the agent, and
// installs the stable release when the version is older than the minimum or
// cannot be read. When the install leaves a version below the minimum, the
// deploy stops before the restart, so the agent that runs keeps its turns.
// When the version still cannot be read, the deploy warns and restarts: a
// later Claude Code can print its version in a new form.
func TestDeployKeepsClaudeCodeAtTheMinimum(t *testing.T) {
	const (
		read    = "read"
		install = "install"
		restart = "restart"
	)
	for name, tc := range map[string]struct {
		versions    []string // what each version read prints; "" fails the read
		installFail bool
		want        []string
		wantErr     string
	}{
		"current":       {versions: []string{"2.1.301"}, want: []string{read, restart}},
		"old":           {versions: []string{"2.1.197", claudecode.MinVersion}, want: []string{read, install, read, restart}},
		"not found":     {versions: []string{"", claudecode.MinVersion}, want: []string{read, install, read, restart}},
		"still old":     {versions: []string{"2.1.197", "2.1.197"}, want: []string{read, install, read}, wantErr: "Claude Code is 2.1.197 after the install, and the minimum is " + claudecode.MinVersion},
		"unreadable":    {versions: []string{"", ""}, want: []string{read, install, read, restart}},
		"install fails": {versions: []string{"2.1.197"}, installFail: true, want: []string{read, install}, wantErr: "installing Claude Code: exit status 22"},
	} {
		t.Run(name, func(t *testing.T) {
			var got []string
			versions := tc.versions
			origSCP, origSSH := scpExec, sshExec
			scpExec = func([]string) ([]byte, error) { return nil, nil }
			sshExec = func(args []string, stdin string) ([]byte, error) {
				cmd := args[len(args)-1]
				switch {
				case strings.Contains(cmd, claudeVersionLine):
					got = append(got, read)
					v := versions[0]
					versions = versions[1:]
					if v == "" {
						return []byte("-bash: line 1: claude: command not found\n"), errors.New("exit status 127")
					}
					return []byte(v + " (Claude Code)\n"), nil
				case strings.Contains(cmd, claudeInstallLine) && strings.Contains(cmd, claudeLinkLine):
					got = append(got, install)
					if tc.installFail {
						return []byte("curl: (22) The requested URL returned error: 503\n"), errors.New("exit status 22")
					}
				case strings.Contains(cmd, "/releases/download/"):
					// The machine downloads the release builds.
				case strings.Contains(cmd, "/usr/local/bin/codex --version"):
					return []byte("codex-cli 0.163.0\n"), nil
				case strings.Contains(cmd, "systemctl restart spekk-agent"):
					got = append(got, restart)
				default:
					t.Errorf("unexpected command %q", cmd)
				}
				return nil, nil
			}
			t.Cleanup(func() { scpExec, sshExec = origSCP, origSSH })

			artifacts := &releaseArtifacts{BinaryPath: filepath.Join(t.TempDir(), "agent-client")}
			err := deployAgent("9.9.9.9", "", "sb", "root", artifacts)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("deploy: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("deploy error = %v, want it to contain %q", err, tc.wantErr)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("commands = %v, want %v", got, tc.want)
			}
		})
	}
}

// The deploy reads the version with the agent service's user and environment
// file, so a PATH in agent.env applies to the check as it does to the
// service.
func TestClaudeVersionLineMatchesAgentUnit(t *testing.T) {
	for _, key := range []string{"User=", "EnvironmentFile="} {
		var value string
		for _, line := range strings.Split(spekkAgentUnit, "\n") {
			if v, ok := strings.CutPrefix(line, key); ok {
				value = v
			}
		}
		if value == "" {
			t.Fatalf("spekkAgentUnit has no %s line", key)
		}
		if want := "-p " + key + value + " "; !strings.Contains(claudeVersionLine, want) {
			t.Errorf("claudeVersionLine %q does not contain %q", claudeVersionLine, want)
		}
	}
}
