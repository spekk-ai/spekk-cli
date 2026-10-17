package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
		Provider:  "digitalocean",
		DropletID: 12345,
		SSHKeyID:  99,
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
	if got.Provider != "digitalocean" {
		t.Errorf("expected provider digitalocean, got %s", got.Provider)
	}
	if got.DropletID != 12345 || got.SSHKeyID != 99 {
		t.Errorf("expected droplet 12345 / key 99, got %d / %d", got.DropletID, got.SSHKeyID)
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

// A newer CLI can write fields that this one does not know. Every save
// rewrites the whole file, so each rewrite must keep them (issue #269).
func TestSaveKeepsUnknownFields(t *testing.T) {
	path := useTempStore(t)
	const file = `{
  "new-box": {
    "provider": "none",
    "ip": "10.0.0.2",
    "createdAt": "2026-10-02T00:00:00Z",
    "status": "active",
    "futureFlag": true,
    "network": {"vpc": "dev", "tags": ["a", "b"]}
  },
  "old-box": {"provider": "none", "ip": "10.0.0.1", "status": "active"}
}`
	if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
		t.Fatal(err)
	}

	meta, err := GetSandbox("new-box")
	if err != nil {
		t.Fatal(err)
	}
	meta.Status = "stopped"
	if err := SaveSandbox("new-box", meta); err != nil {
		t.Fatal(err)
	}
	if err := SaveSandbox("third-box", &SandboxMeta{IP: "10.0.0.3"}); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSandbox("old-box"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	rec := got["new-box"]
	if rec["status"] != "stopped" {
		t.Errorf("status = %v, want the new value stopped", rec["status"])
	}
	if rec["futureFlag"] != true {
		t.Errorf("futureFlag = %v, want true", rec["futureFlag"])
	}
	wantNetwork := map[string]any{"vpc": "dev", "tags": []any{"a", "b"}}
	if !reflect.DeepEqual(rec["network"], wantNetwork) {
		t.Errorf("network = %v, want %v", rec["network"], wantNetwork)
	}
}
