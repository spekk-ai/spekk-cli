package sandbox

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Verify ManualProvider satisfies the Provider interface at compile time.
var _ Provider = (*ManualProvider)(nil)

func TestManualProviderCreate(t *testing.T) {
	p := &ManualProvider{}

	t.Run("records the machine it was given", func(t *testing.T) {
		meta := &SandboxMeta{}
		opts := CreateOptions{Name: "test", IP: "192.168.1.100", SSHKey: "/home/user/.ssh/id_ed25519"}
		if err := p.Create("test", opts, meta); err != nil {
			t.Fatal(err)
		}
		if meta.IP != opts.IP || meta.SSHKeyPath != opts.SSHKey {
			t.Errorf("meta = %+v, want the given ip and key", meta)
		}
		if meta.DropletID != 0 {
			t.Errorf("a manual machine has no droplet: %+v", meta)
		}
	})

	t.Run("errors name the missing flag", func(t *testing.T) {
		for flag, opts := range map[string]CreateOptions{
			"--ip":      {SSHKey: "/tmp/key"},
			"--ssh-key": {IP: "1.2.3.4"},
		} {
			err := p.Create("test", opts, &SandboxMeta{})
			if err == nil {
				t.Fatalf("expected an error when %s is missing", flag)
			}
			if !strings.Contains(err.Error(), flag) {
				t.Errorf("error %q should name %s", err, flag)
			}
		}
	})
}

// The provisioning script has to be valid shell. Its content is not asserted
// line by line: that pins wording rather than behavior, and the real check is
// running it on a machine.
func TestProvisionScriptIsValidShell(t *testing.T) {
	// A key whose comment holds a command substitution. Carried as text it
	// would run as root; carried base64-encoded it cannot.
	key := "ssh-ed25519 AAAAC3Nza $(touch /tmp/spekk-should-not-exist) test@host"
	script := provisionScript(key)

	if strings.Contains(script, "$(touch") {
		t.Error("the key reached the script as shell text, so its comment can run as root")
	}
	if !strings.Contains(script, base64.StdEncoding.EncodeToString([]byte(key))) {
		t.Error("the encoded key must reach the script")
	}
	if _, err := runShellCheck(t, script); err != nil {
		t.Errorf("generated script is not valid shell: %v", err)
	}

	// The script must still hand bash the original key, byte for byte.
	out, err := exec.Command("bash", "-c", grepLine(script, "PUB_KEY=")+"\nprintf %s \"$PUB_KEY\"").Output()
	if err != nil {
		t.Fatalf("decoding the key in bash failed: %v", err)
	}
	if string(out) != key {
		t.Errorf("key round trip: got %q, want %q", out, key)
	}
}

// grepLine returns the first line of script with the given prefix.
func grepLine(script, prefix string) string {
	for _, line := range strings.Split(script, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

// Destroy on a manual sandbox must never delete the operator's own key,
// and must not remove the local record while the agent may still run.
func TestManualDestroy(t *testing.T) {
	// The operator's key lives outside spekk's directories.
	operatorKey := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "id_ed25519")
		for _, p := range []string{path, path + ".pub"} {
			if err := os.WriteFile(p, []byte("secret"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return path
	}

	t.Run("keeps the operator's key", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		key := operatorKey(t)
		if err := SaveSandbox("box", &SandboxMeta{Provider: "manual", IP: "1.2.3.4", SSHKeyPath: key}); err != nil {
			t.Fatal(err)
		}
		orig := stopAgent
		stopAgent = func(meta *SandboxMeta, name string) error { return nil }
		t.Cleanup(func() { stopAgent = orig })

		if err := Destroy(&ManualProvider{}, "box", true); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(key); err != nil {
			t.Errorf("destroy deleted the operator's private key: %v", err)
		}
		if _, err := os.Stat(key + ".pub"); err != nil {
			t.Errorf("destroy deleted the operator's public key: %v", err)
		}
	})

	t.Run("keeps the record when the agent cannot be stopped", func(t *testing.T) {
		isolateConfig(t)
		useTempStore(t)
		if err := SaveSandbox("box", &SandboxMeta{Provider: "manual", IP: "1.2.3.4", SSHKeyPath: operatorKey(t)}); err != nil {
			t.Fatal(err)
		}
		orig := stopAgent
		stopAgent = func(meta *SandboxMeta, name string) error { return errors.New("host unreachable") }
		t.Cleanup(func() { stopAgent = orig })

		answerPrompt(t, "y\n")
		if err := Destroy(&ManualProvider{}, "box", false); err == nil {
			t.Fatal("destroy must fail when the agent cannot be stopped")
		}
		if got, _ := GetSandbox("box"); got == nil {
			t.Error("the record must survive, or an agent keeps running with nothing pointing at it")
		}
	})
}

// answerPrompt feeds one answer to the confirmation prompt Destroy reads
// from stdin.
func answerPrompt(t *testing.T, answer string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(answer); err != nil {
		t.Fatal(err)
	}
	w.Close()
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; r.Close() })
}
