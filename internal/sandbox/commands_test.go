package sandbox

import (
	"encoding/base64"
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
