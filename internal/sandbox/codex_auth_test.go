package sandbox

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexChatGPTCacheUsesSSHStdin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	secret := `{"auth_mode":"chatgpt","tokens":{"refresh_token":"fixture-refresh-secret"}}`
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		t.Fatal(err)
	}
	original := sshExec
	t.Cleanup(func() { sshExec = original })
	sshExec = func(args []string, stdin string) ([]byte, error) {
		for _, arg := range args {
			if strings.Contains(arg, "fixture-refresh-secret") || strings.Contains(arg, base64.StdEncoding.EncodeToString([]byte(secret))) {
				t.Fatal("credential reached SSH argv")
			}
		}
		if !strings.Contains(stdin, base64.StdEncoding.EncodeToString([]byte(secret))) {
			t.Fatal("credential did not reach SSH stdin")
		}
		for _, part := range []string{"chown agent:agent", "chmod 0600", "/home/agent/.codex/auth.json"} {
			if !strings.Contains(stdin, part) {
				t.Errorf("private install script lacks %q", part)
			}
		}
		return nil, nil
	}
	if err := injectCodexAuth("127.0.0.1", "", "example", "root", path); err != nil {
		t.Fatal(err)
	}
}

func TestCodexCachePreflightRejectsInvalidLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"auth_mode":"chatgpt","tokens":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readCodexAuth(path); err == nil {
		t.Fatal("ChatGPT cache without refresh token accepted")
	}
}
