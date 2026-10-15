package sandbox

import (
	"os"
	"strings"
	"testing"
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
