package sandbox

import (
	"encoding/base64"
	"os/exec"
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
