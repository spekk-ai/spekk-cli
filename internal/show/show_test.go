package show

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildShowData(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	writeFile(t, filepath.Join(specsDir, "my-spec", "my-spec.md"), `---
id: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# My Spec`)

	writeFile(t, filepath.Join(specsDir, "my-spec", "assertions", "my-assertion.md"), `---
id: my-assertion
parent: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
branch: feature/test
---

# My Assertion`)

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatal(err)
	}

	data := buildShowData(specsDir, result)

	if data.ProjectName != filepath.Base(dir) {
		t.Errorf("expected project name %s, got %s", filepath.Base(dir), data.ProjectName)
	}
	if len(data.Specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(data.Specs))
	}
	if data.Specs[0].ID != "my-spec" {
		t.Errorf("expected spec id my-spec, got %s", data.Specs[0].ID)
	}
	if len(data.Assertions) != 1 {
		t.Fatalf("expected 1 assertion, got %d", len(data.Assertions))
	}
	if data.Assertions[0].Branch != "feature/test" {
		t.Errorf("expected branch feature/test, got %s", data.Assertions[0].Branch)
	}
}

func TestTemplateContainsPlaceholder(t *testing.T) {
	if !strings.Contains(templateHTML, "/*__SPEKK_DATA__*/") {
		t.Error("template should contain the data placeholder")
	}
}

func TestRunWritesHTML(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	writeFile(t, filepath.Join(specsDir, "test-spec", "test-spec.md"), `---
id: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# Test Spec`)

	writeFile(t, filepath.Join(specsDir, "test-spec", "assertions", "test-a.md"), `---
id: test-a
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# Test Assertion`)

	// Set CI to skip browser opening
	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")

	err := Run(specsDir)
	if err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, ".spekk", "index.html")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	html := string(content)
	if !strings.Contains(html, "test-spec") {
		t.Error("HTML should contain spec ID")
	}
	if !strings.Contains(html, "Test Assertion") {
		t.Error("HTML should contain assertion title")
	}
	if strings.Contains(html, "/*__SPEKK_DATA__*/") {
		t.Error("placeholder should be replaced with actual data")
	}
}
