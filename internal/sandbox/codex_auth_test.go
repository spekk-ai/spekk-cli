package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexCachePreflightRejectsInvalidLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"auth_mode":"chatgpt","tokens":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readCodexAuth(path); err == nil {
		t.Fatal("ChatGPT cache without refresh token accepted")
	}
}

func TestCodexCachePreflightRejectsUnsafeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(`{"auth_mode":"chatgpt","tokens":{"refresh_token":"test"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readCodexAuth(path); err == nil {
		t.Fatal("readable cache accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readCodexAuth(link); err == nil {
		t.Fatal("symlink cache accepted")
	}
}
