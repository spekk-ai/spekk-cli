package sandbox

import (
	"strings"
	"testing"
)

// Verify ManualProvider satisfies the Provider interface at compile time.
var _ Provider = (*ManualProvider)(nil)

func TestManualProviderCreate(t *testing.T) {
	p := &ManualProvider{}

	t.Run("succeeds with ip and ssh_key", func(t *testing.T) {
		result, err := p.Create("test", map[string]string{
			"ip":      "192.168.1.100",
			"ssh_key": "/home/user/.ssh/id_ed25519",
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.IP != "192.168.1.100" {
			t.Errorf("IP = %q, want 192.168.1.100", result.IP)
		}
		if result.SSHKeyPath != "/home/user/.ssh/id_ed25519" {
			t.Errorf("SSHKeyPath = %q", result.SSHKeyPath)
		}
		if result.InstanceID != "" {
			t.Errorf("InstanceID should be empty for manual, got %q", result.InstanceID)
		}
		if result.Provider != "manual" {
			t.Errorf("Provider = %q, want manual", result.Provider)
		}
	})

	t.Run("errors without ip", func(t *testing.T) {
		_, err := p.Create("test", map[string]string{"ssh_key": "/tmp/key"})
		if err == nil {
			t.Fatal("expected error for missing --ip")
		}
		if !strings.Contains(err.Error(), "--ip") {
			t.Errorf("error %q should mention --ip", err)
		}
	})

	t.Run("errors without ssh_key", func(t *testing.T) {
		_, err := p.Create("test", map[string]string{"ip": "1.2.3.4"})
		if err == nil {
			t.Fatal("expected error for missing --ssh-key")
		}
		if !strings.Contains(err.Error(), "--ssh-key") {
			t.Errorf("error %q should mention --ssh-key", err)
		}
	})
}

func TestManualProviderDestroy(t *testing.T) {
	p := &ManualProvider{}
	// Destroy is a no-op — should never error.
	if err := p.Destroy(""); err != nil {
		t.Errorf("Destroy should be no-op, got: %v", err)
	}
}

func TestManualProviderStatus(t *testing.T) {
	p := &ManualProvider{}
	status, err := p.Status("")
	if err != nil {
		t.Fatal(err)
	}
	if status != "manual" {
		t.Errorf("Status = %q, want manual", status)
	}
}

func TestProvisionScriptIsIdempotent(t *testing.T) {
	script := provisionScript("ssh-ed25519 AAAA_TEST_KEY user@host")

	// Script uses idempotent patterns.
	checks := []struct {
		desc    string
		pattern string
	}{
		{"checks agent user existence", "id -u agent"},
		{"uses apt-get install -y", "apt-get install -y"},
		{"checks docker before installing", "command -v docker"},
		{"checks node before installing", "command -v node"},
		{"checks gh before installing", "command -v gh"},
		{"checks claude before installing", "command -v claude"},
		{"checks UFW rules before appending", "BEGIN UFW AND DOCKER"},
		{"checks authorized_keys before adding", "grep -qF"},
		{"touches provisioned marker", "touch /opt/spekk/.provisioned"},
	}
	for _, c := range checks {
		if !strings.Contains(script, c.pattern) {
			t.Errorf("provisioning script should %s (missing %q)", c.desc, c.pattern)
		}
	}
}

func TestProvisionScriptContainsSSHKey(t *testing.T) {
	key := "ssh-ed25519 AAAAC3NzaC1 test@example"
	script := provisionScript(key)
	if !strings.Contains(script, key) {
		t.Error("provisioning script should contain the SSH public key")
	}
}

func TestProvisionScriptCreatesRequiredInfra(t *testing.T) {
	script := provisionScript("ssh-ed25519 AAAA test@host")

	required := []string{
		"useradd",         // creates agent user
		"docker",          // installs Docker
		"ufw",             // configures firewall
		"nodejs",          // installs Node.js
		"gh",              // installs GitHub CLI
		"claude-code",     // installs Claude Code
		"/opt/spekk",      // creates spekk dir
		"/etc/spekk",      // creates config dir
		"/var/log/spekk",  // creates log dir
		"/workspace",      // creates workspace dir
		"fail2ban",        // configures fail2ban
		".provisioned",    // marks complete
	}
	for _, r := range required {
		if !strings.Contains(script, r) {
			t.Errorf("provisioning script should reference %q", r)
		}
	}
}
