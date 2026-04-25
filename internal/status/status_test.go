package status

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestShow_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	os.Mkdir(specsDir, 0o755)

	output := captureStdout(t, func() {
		Show(specsDir)
	})

	if !strings.Contains(output, "No specifications found") {
		t.Errorf("expected empty message, got: %s", output)
	}
	if !strings.Contains(output, "specs/{spec-name}") {
		t.Errorf("expected getting-started hint, got: %s", output)
	}
}

func TestShow_WithSpecs(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	writeFile(t, filepath.Join(specsDir, "my-spec", "my-spec.md"), `---
id: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# My Feature
`)
	writeFile(t, filepath.Join(specsDir, "my-spec", "assertions", "a1.md"), `---
id: a1
parent: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: done
---

# First assertion
`)
	writeFile(t, filepath.Join(specsDir, "my-spec", "assertions", "a2.md"), `---
id: a2
parent: my-spec
created: 2026-01-02T00:00:00Z
priority: 2
status: not_started
---

# Second assertion
`)

	output := captureStdout(t, func() {
		Show(specsDir)
	})

	// Spec header with completion ratio.
	if !strings.Contains(output, "My Feature (1/2 assertions complete)") {
		t.Errorf("expected completion ratio, got: %s", output)
	}
	// Status icons present.
	if !strings.Contains(output, "✅") {
		t.Errorf("expected done icon")
	}
	if !strings.Contains(output, "📋") {
		t.Errorf("expected not_started icon")
	}
	// Overall stats.
	if !strings.Contains(output, "Total: 2 assertions") {
		t.Errorf("expected total count")
	}
	if !strings.Contains(output, "Completion: 50%") {
		t.Errorf("expected completion percentage")
	}
	// Next priority item.
	if !strings.Contains(output, "Next Priority Item") {
		t.Errorf("expected next priority section")
	}
	if !strings.Contains(output, "Second assertion") {
		t.Errorf("expected next item title")
	}
}

func TestShow_AllComplete(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	writeFile(t, filepath.Join(specsDir, "done-spec", "done-spec.md"), `---
id: done-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# Done Spec
`)
	writeFile(t, filepath.Join(specsDir, "done-spec", "assertions", "done-a.md"), `---
id: done-a
parent: done-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: done
---

# Done assertion
`)

	output := captureStdout(t, func() {
		Show(specsDir)
	})

	if !strings.Contains(output, "All specifications complete") {
		t.Errorf("expected all complete message, got: %s", output)
	}
	if !strings.Contains(output, "Completion: 100%") {
		t.Errorf("expected 100%% completion")
	}
}
