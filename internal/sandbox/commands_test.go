package sandbox

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildInjectScript_NoShellInterpolation(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantSafe bool // the generated script must not contain raw credential values
	}{
		{
			name: "heredoc breakout in AWS secret key",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIA1234",
				"AWS_SECRET_ACCESS_KEY": "secret\nENVEOF\nmalicious_command\nENVEOF",
				"AWS_DEFAULT_REGION":    "us-east-1",
				"GITHUB_TOKEN":          "ghp_normal",
			},
			wantSafe: true,
		},
		{
			name: "shell injection in GITHUB_TOKEN",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIA1234",
				"AWS_SECRET_ACCESS_KEY": "secret",
				"AWS_DEFAULT_REGION":    "us-east-1",
				"GITHUB_TOKEN":          `"; rm -rf /; echo "`,
			},
			wantSafe: true,
		},
		{
			name: "backtick injection",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIA1234",
				"AWS_SECRET_ACCESS_KEY": "`whoami`",
				"AWS_DEFAULT_REGION":    "us-east-1",
				"GITHUB_TOKEN":          "$(id)",
			},
			wantSafe: true,
		},
		{
			name: "single quote injection",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIA1234",
				"AWS_SECRET_ACCESS_KEY": "it's a secret",
				"AWS_DEFAULT_REGION":    "us-east-1",
				"GITHUB_TOKEN":          "ghp_normal",
			},
			wantSafe: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envContent := buildEnvContent(tt.envVars, "test-sandbox", "agent-token-123", "https://spekk.example.com")
			script := buildInjectScript(envContent)

			// The script should NOT contain any raw credential values —
			// they should only appear base64-encoded.
			for key, val := range tt.envVars {
				if val == "" {
					continue
				}
				if strings.Contains(script, val) && !isBase64Safe(val) {
					t.Errorf("script contains raw value of %s: %q", key, val)
				}
			}

			// The script should contain a base64-encoded payload
			if !strings.Contains(script, "base64 -d") {
				t.Error("script does not use base64 decoding")
			}

			// Verify the base64 payload decodes to the correct env content
			parts := strings.SplitN(script, "'", 3)
			if len(parts) < 3 {
				t.Fatal("unexpected script format")
			}
			encoded := parts[1]
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatalf("base64 decode failed: %v", err)
			}
			if string(decoded) != envContent {
				t.Errorf("decoded content mismatch:\ngot:  %q\nwant: %q", string(decoded), envContent)
			}
		})
	}
}

func TestBuildGitCredentialScript_NoShellInterpolation(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "shell injection with quotes and semicolons",
			token: `"; rm -rf /; echo "`,
		},
		{
			name:  "backtick command substitution",
			token: "`rm -rf /`",
		},
		{
			name:  "dollar command substitution",
			token: "$(cat /etc/passwd)",
		},
		{
			name:  "newline injection",
			token: "ghp_tok\n; malicious_command",
		},
		{
			name:  "single quote breakout",
			token: "ghp_tok' ; rm -rf / ; echo '",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := buildGitCredentialScript(tt.token)

			// The raw token must NOT appear in the script
			if strings.Contains(script, tt.token) {
				t.Errorf("script contains raw token value: %q", tt.token)
			}

			// Must use base64 decoding
			if !strings.Contains(script, "base64 -d") {
				t.Error("script does not use base64 decoding")
			}

			// Verify the base64 payload decodes to the correct token
			parts := strings.SplitN(script, "'", 3)
			if len(parts) < 3 {
				t.Fatal("unexpected script format")
			}
			encoded := parts[1]
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				t.Fatalf("base64 decode failed: %v", err)
			}
			if string(decoded) != tt.token {
				t.Errorf("decoded token mismatch:\ngot:  %q\nwant: %q", string(decoded), tt.token)
			}
		})
	}
}

func TestValidateSandboxName(t *testing.T) {
	valid := []string{"my-sandbox", "prod1", "a", "test-env-2", "abc123"}
	for _, name := range valid {
		if err := ValidateSandboxName(name); err != nil {
			t.Errorf("ValidateSandboxName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{
		"",
		"; rm -rf /",
		"$(whoami)",
		"name\nmalicious",
		"-starts-with-dash",
		"Has Spaces",
		"has`backtick",
		"UPPERCASE",
		"with;semicolon",
		"with'quote",
		`with"doublequote`,
		"with$dollar",
	}
	for _, name := range invalid {
		if err := ValidateSandboxName(name); err == nil {
			t.Errorf("ValidateSandboxName(%q) = nil, want error", name)
		}
	}
}

func TestSSHHostKeyOpts(t *testing.T) {
	tmpDir := t.TempDir()
	name := "test-host-key-sandbox"
	khDir := filepath.Join(tmpDir, "known_hosts")
	khFile := filepath.Join(khDir, name)

	// Override KnownHostsFile to use temp directory
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Verify the known_hosts path uses our temp dir
	got := KnownHostsFile(name)
	want := filepath.Join(tmpDir, ".spekk", "known_hosts", name)
	if got != want {
		t.Fatalf("KnownHostsFile(%q) = %q, want %q", name, got, want)
	}

	// First connection: no known_hosts file exists, should use accept-new
	opts := sshHostKeyOpts(name)
	joined := strings.Join(opts, " ")
	if !strings.Contains(joined, "StrictHostKeyChecking=accept-new") {
		t.Errorf("first connection should use accept-new, got: %v", opts)
	}
	if !strings.Contains(joined, "UserKnownHostsFile=") {
		t.Errorf("should specify UserKnownHostsFile, got: %v", opts)
	}
	if strings.Contains(joined, "/dev/null") {
		t.Errorf("should NOT use /dev/null, got: %v", opts)
	}

	// Simulate a known_hosts file being created (as SSH would after first connection)
	os.MkdirAll(khDir, 0o700)
	// Use the actual path from KnownHostsFile since it goes through ~/.spekk/
	actualKhFile := KnownHostsFile(name)
	os.MkdirAll(filepath.Dir(actualKhFile), 0o700)
	os.WriteFile(actualKhFile, []byte("192.168.1.1 ssh-ed25519 AAAA...\n"), 0o600)

	// Subsequent connection: known_hosts exists, should use strict checking
	opts = sshHostKeyOpts(name)
	joined = strings.Join(opts, " ")
	if !strings.Contains(joined, "StrictHostKeyChecking=yes") {
		t.Errorf("subsequent connection should use strict checking, got: %v", opts)
	}
	if strings.Contains(joined, "accept-new") {
		t.Errorf("subsequent connection should NOT use accept-new, got: %v", opts)
	}

	// sshArgs should include host key opts and not the old insecure flags
	sandbox := &SandboxMeta{IP: "192.168.1.1", SSHKeyPath: "/tmp/key"}
	args := sshArgs(sandbox, name)
	argsStr := strings.Join(args, " ")
	if strings.Contains(argsStr, "StrictHostKeyChecking=no") {
		t.Errorf("sshArgs should not use StrictHostKeyChecking=no, got: %v", args)
	}
	if strings.Contains(argsStr, "UserKnownHostsFile=/dev/null") {
		t.Errorf("sshArgs should not use UserKnownHostsFile=/dev/null, got: %v", args)
	}

	// Destroy cleanup: known_hosts file should be removable
	_ = khFile // suppress unused warning
	os.Remove(actualKhFile)
	if _, err := os.Stat(actualKhFile); !os.IsNotExist(err) {
		t.Errorf("known_hosts file should be removed after cleanup")
	}
}

// isBase64Safe checks if a value happens to be valid base64 characters only,
// meaning it could appear in the script as part of the encoded payload
// without being a shell injection risk.
func isBase64Safe(s string) bool {
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}
	return true
}
