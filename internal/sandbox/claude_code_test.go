package sandbox

import (
	"errors"
	"os"
	"path/filepath"
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
	for name, content := range map[string]string{
		"cloud-init.yaml":    string(cloudInitTemplate),
		"prepare-machine.sh": string(prepare),
	} {
		for _, line := range []string{claudeInstallLine, claudeLinkLine} {
			if !strings.Contains(content, line) {
				t.Errorf("%s does not contain %q", name, line)
			}
		}
		if strings.Contains(content, "npm install -g @anthropic-ai/claude-code") {
			t.Errorf("%s installs Claude Code from npm", name)
		}
	}
}

// A deploy reads the Claude Code version before it restarts the agent, and
// installs the stable release when the version is older than the minimum or
// not found. When the install does not reach the minimum, the deploy stops
// before the restart, so the agent that runs keeps its turns.
func TestDeployKeepsClaudeCodeAtTheMinimum(t *testing.T) {
	const (
		read    = "read"
		install = "install"
		restart = "restart"
	)
	for name, tc := range map[string]struct {
		versions []string // what each version read prints; "" fails the read
		want     []string
		wantErr  string
	}{
		"current":      {versions: []string{"2.1.301"}, want: []string{read, restart}},
		"old":          {versions: []string{"2.1.197", claudecode.MinVersion}, want: []string{read, install, read, restart}},
		"not found":    {versions: []string{"", claudecode.MinVersion}, want: []string{read, install, read, restart}},
		"still old":    {versions: []string{"2.1.197", "2.1.197"}, want: []string{read, install, read}, wantErr: "Claude Code is 2.1.197 after the install, and the minimum is " + claudecode.MinVersion},
		"still absent": {versions: []string{"", ""}, want: []string{read, install, read}, wantErr: "Claude Code is not found after the install"},
	} {
		t.Run(name, func(t *testing.T) {
			var got []string
			versions := tc.versions
			origSCP, origSSH := scpExec, sshExec
			scpExec = func([]string) ([]byte, error) { return nil, nil }
			sshExec = func(args []string) ([]byte, error) {
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
