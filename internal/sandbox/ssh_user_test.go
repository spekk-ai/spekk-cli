package sandbox

import (
	"encoding/base64"
	"strings"
	"testing"
)

// A machine spekk created is reached as root, so an empty SSHUser must read
// back as root; a machine the operator named may carry any login user.
func TestSSHUserDefaultsToRoot(t *testing.T) {
	if got := sshUser(&SandboxMeta{}); got != "root" {
		t.Errorf("empty SSHUser: got %q, want root", got)
	}
	if got := sshUser(&SandboxMeta{SSHUser: "ubuntu"}); got != "ubuntu" {
		t.Errorf("set SSHUser: got %q, want ubuntu", got)
	}
}

// A non-root login user runs privileged steps under sudo. sudoWrap must hand
// bash the exact script back, so a decode of what it pipes reproduces it.
func TestSudoWrapRunsTheScriptUnderSudo(t *testing.T) {
	script := "systemctl restart spekk-agent\nexit $rc"
	got := sudoWrap(script)

	if !strings.Contains(got, "| sudo bash") {
		t.Errorf("sudoWrap output does not pipe through sudo bash: %q", got)
	}

	// The wrapper is `echo '<b64>' | base64 -d | sudo bash`; pull the
	// base64 back out and confirm it decodes to the original script, so a
	// heredoc or nested quote in the script cannot corrupt the command.
	start := strings.Index(got, "'")
	end := strings.LastIndex(got, "'")
	if start < 0 || end <= start {
		t.Fatalf("no quoted payload in sudoWrap output: %q", got)
	}
	decoded, err := base64.StdEncoding.DecodeString(got[start+1 : end])
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	if string(decoded) != script {
		t.Errorf("decoded payload = %q, want %q", decoded, script)
	}
}
