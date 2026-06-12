package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	// Use temp dir as home
	tmpDir := t.TempDir()
	origFile := sandboxesFile
	sandboxesFile = func() string {
		return filepath.Join(tmpDir, ".spekk", "sandboxes.json")
	}
	defer func() { sandboxesFile = origFile }()

	// Initially empty
	sandboxes, err := LoadSandboxes()
	if err != nil {
		t.Fatal(err)
	}
	if len(sandboxes) != 0 {
		t.Fatalf("expected 0 sandboxes, got %d", len(sandboxes))
	}

	// Save a sandbox
	meta := &SandboxMeta{
		DropletID: 12345,
		IP:        "1.2.3.4",
		Region:    "nyc1",
		Size:      "s-1vcpu-1gb",
		CreatedAt: "2026-01-01T00:00:00Z",
		Status:    "active",
	}
	if err := SaveSandbox("test-sb", meta); err != nil {
		t.Fatal(err)
	}

	// Get it back
	got, err := GetSandbox("test-sb")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected sandbox, got nil")
	}
	if got.DropletID != 12345 {
		t.Errorf("expected droplet ID 12345, got %d", got.DropletID)
	}
	if got.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %s", got.IP)
	}

	// Get nonexistent
	missing, err := GetSandbox("nope")
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Error("expected nil for missing sandbox")
	}

	// Remove it
	if err := RemoveSandbox("test-sb"); err != nil {
		t.Fatal(err)
	}
	got2, _ := GetSandbox("test-sb")
	if got2 != nil {
		t.Error("expected nil after remove")
	}
}

func TestLoadSandboxesMissingFile(t *testing.T) {
	origFile := sandboxesFile
	sandboxesFile = func() string {
		return filepath.Join(t.TempDir(), "nonexistent", "sandboxes.json")
	}
	defer func() { sandboxesFile = origFile }()

	sandboxes, err := LoadSandboxes()
	if err != nil {
		t.Fatal(err)
	}
	if len(sandboxes) != 0 {
		t.Errorf("expected empty map, got %d entries", len(sandboxes))
	}
}

func TestRemapLegacyKeyPath(t *testing.T) {
	// Existing path is returned unchanged.
	tmp := t.TempDir()
	existing := filepath.Join(tmp, "key")
	os.WriteFile(existing, []byte("k"), 0o600)
	if got := remapLegacyKeyPath(existing); got != existing {
		t.Errorf("existing path should be unchanged, got %s", got)
	}

	// Empty path is returned unchanged.
	if got := remapLegacyKeyPath(""); got != "" {
		t.Errorf("empty path should be unchanged, got %s", got)
	}

	// A missing path outside ~/.spekk is returned unchanged.
	missing := filepath.Join(tmp, "nope")
	if got := remapLegacyKeyPath(missing); got != missing {
		t.Errorf("non-legacy missing path should be unchanged, got %s", got)
	}
}

func TestSandboxesFileWrittenCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	origFile := sandboxesFile
	sandboxesFile = func() string {
		return filepath.Join(tmpDir, ".spekk", "sandboxes.json")
	}
	defer func() { sandboxesFile = origFile }()

	SaveSandbox("my-sb", &SandboxMeta{DropletID: 1, IP: "10.0.0.1"})

	data, err := os.ReadFile(sandboxesFile())
	if err != nil {
		t.Fatal(err)
	}

	// Should be valid JSON ending with newline
	s := string(data)
	if s[len(s)-1] != '\n' {
		t.Error("expected trailing newline")
	}
}
