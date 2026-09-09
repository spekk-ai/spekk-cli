package sandbox

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"slices"
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

// The login user is stored and then interpolated into an ssh argument on
// every later command, so a value ssh reads as an option has to be refused
// before anything is recorded.
func TestValidateSSHUserRejectsAnSSHOption(t *testing.T) {
	for _, user := range []string{"", "root", "ubuntu", "ec2-user", "_spekk", "Administrator"} {
		if err := validateSSHUser(user); err != nil {
			t.Errorf("validateSSHUser(%q) = %v, want a valid login name", user, err)
		}
	}
	for _, user := range []string{"-oProxyCommand=touch /tmp/pwned;", "-l root", "ubuntu@elsewhere", "root me", "-"} {
		if err := validateSSHUser(user); err == nil {
			t.Errorf("validateSSHUser(%q) = nil, want a rejection", user)
		}
	}
}

// The login user survives create, because destroy and deploy read it back
// from the store and would otherwise fall back to root on a machine that
// refuses root.
func TestCreateRecordsTheLoginUser(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	stubCreateEnv(t)

	key := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(key, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	origCheck := checkReady
	checkReady = func(meta *SandboxMeta, name string) error { return errors.New("not provisioned") }
	t.Cleanup(func() { checkReady = origCheck })

	if err := Create(nil, CreateOptions{Name: "borrowed", IP: "9.9.9.9", SSHKey: key, SSHUser: "ubuntu"}); err == nil {
		t.Fatal("expected the provisioned check to stop the create")
	}
	got, _ := GetSandbox("borrowed")
	if got == nil {
		t.Fatal("the machine was registered but nothing was recorded")
	}
	if got.SSHUser != "ubuntu" {
		t.Errorf("SSHUser = %q, want ubuntu", got.SSHUser)
	}
	if user := sshArgs(got, "borrowed"); !slices.Contains(user, "ubuntu@9.9.9.9") {
		t.Errorf("ssh destination = %v, want ubuntu@9.9.9.9", user)
	}
	if user := sshBatchArgs(got, "borrowed"); !slices.Contains(user, "ubuntu@9.9.9.9") {
		t.Errorf("batch ssh destination = %v, want ubuntu@9.9.9.9", user)
	}

	// A login user ssh would read as an option costs no metadata entry.
	if err := Create(nil, CreateOptions{Name: "hostile", IP: "9.9.9.9", SSHKey: key, SSHUser: "-oProxyCommand=id"}); err == nil {
		t.Fatal("expected an invalid --ssh-user to stop the create")
	}
	if got, _ := GetSandbox("hostile"); got != nil {
		t.Errorf("an invalid --ssh-user was recorded anyway: %+v", got)
	}
}

// Each privileged step builds its own remote command, so each is checked
// here. A call site that drops privilegedScript still compiles, and the unit
// tests above would still pass, but the step would then run unprivileged and
// fail on somebody's machine rather than in CI.
func TestEveryPrivilegedStepEscalatesForANonRootLogin(t *testing.T) {
	var sent []string
	origExec := sshExec
	sshExec = func(args []string) ([]byte, error) {
		sent = append(sent, args[len(args)-1])
		return nil, nil
	}
	t.Cleanup(func() { sshExec = origExec })

	steps := map[string]func(user string) error{
		"inject credentials": func(user string) error {
			return injectCredentials("9.9.9.9", "", "sb", user, "agent-token", AuthBedrock)
		},
		"git credentials": func(user string) error {
			return configureGitCredentials("9.9.9.9", "", "sb", user)
		},
		"teardown": func(user string) error {
			return stopAgentService(&SandboxMeta{IP: "9.9.9.9", SSHUser: user}, "sb")
		},
	}
	for name, step := range steps {
		t.Run(name, func(t *testing.T) {
			sent = nil
			if err := step("ubuntu"); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if len(sent) != 1 {
				t.Fatalf("%s sent %d commands, want 1", name, len(sent))
			}
			if !strings.Contains(sent[0], "| sudo bash") {
				t.Errorf("%s does not escalate for a non-root login: %q", name, sent[0])
			}

			// A root login must be untouched by any of this.
			sent = nil
			if err := step("root"); err != nil {
				t.Fatalf("%s as root: %v", name, err)
			}
			if strings.Contains(sent[0], "sudo") {
				t.Errorf("%s escalates for root, which already is root: %q", name, sent[0])
			}
		})
	}
}

// Deployment stages uploads in the login user's home and then installs them.
func TestDeployStagesAndInstallsForEachLogin(t *testing.T) {
	var scpArgs, sshCommands []string
	origSCP, origSSH := scpExec, sshExec
	scpExec = func(args []string) ([]byte, error) {
		scpArgs = append(scpArgs, args[len(args)-1])
		return nil, nil
	}
	sshExec = func(args []string) ([]byte, error) {
		sshCommands = append(sshCommands, args[len(args)-1])
		return nil, nil
	}
	t.Cleanup(func() { scpExec, sshExec = origSCP, origSSH })

	artifacts := &releaseArtifacts{BinaryPath: filepath.Join(t.TempDir(), "agent-client")}

	for _, user := range []string{"root", "ubuntu"} {
		t.Run(user, func(t *testing.T) {
			scpArgs, sshCommands = nil, nil
			if err := deployAgent("9.9.9.9", "", "sb", user, artifacts); err != nil {
				t.Fatalf("deploy: %v", err)
			}
			if len(scpArgs) != 1 || len(sshCommands) != 1 {
				t.Fatalf("expected one upload and installation, got %v and %v", scpArgs, sshCommands)
			}
			if want := user + "@9.9.9.9:" + stagedBinary; scpArgs[0] != want {
				t.Errorf("staged at %q, want %q", scpArgs[0], want)
			}
			if got := strings.Contains(sshCommands[0], "| sudo bash"); got != (user != "root") {
				t.Errorf("incorrect privilege level for %s: %s", user, sshCommands[0])
			}
			if !strings.Contains(sshCommands[0], `cd "$HOME"`) {
				t.Fatalf("installation must start in the upload directory: %s", sshCommands[0])
			}
		})
	}

	scpExec = func([]string) ([]byte, error) { return nil, errors.New("upload failed") }
	sshCommands = nil
	if err := deployAgent("9.9.9.9", "", "sb", "root", artifacts); err == nil || len(sshCommands) != 0 {
		t.Fatalf("upload failure must stop installation: error %v, commands %v", err, sshCommands)
	}
}
