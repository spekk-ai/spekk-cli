//go:build linux || darwin

package sandbox

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexCachePreflightAcceptsPrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	want := []byte(`{"auth_mode":"chatgpt","tokens":{"refresh_token":"test"}}`)
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readCodexAuth(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("private ChatGPT cache was not read intact")
	}
}
