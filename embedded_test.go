package spekk

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEmbeddedFS_SpekkDevLoopSkill verifies the spekk-dev-loop skill ships
// with the binary via the embedded FS, and that its frontmatter declares
// name: spekk-dev-loop as required by the assertion.
func TestEmbeddedFS_SpekkDevLoopSkill(t *testing.T) {
	const path = "specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md"

	data, err := fs.ReadFile(EmbeddedFS, path)
	if err != nil {
		t.Fatalf("expected %s in embedded FS: %v", path, err)
	}

	content := string(data)
	if !strings.Contains(content, "name: spekk-dev-loop") {
		t.Errorf("embedded spekk-dev-loop skill missing \"name: spekk-dev-loop\" in frontmatter")
	}
}

// TestEmbeddedFS_ObserverCoverageGapSkill verifies the coverage-gap observer
// skill ships with the binary via the embedded FS, and that its frontmatter
// declares the id and description required by the assertion.
func TestEmbeddedFS_ObserverCoverageGapSkill(t *testing.T) {
	const path = "specs/observer-skills/coverage-gap-skill.md"

	data, err := fs.ReadFile(EmbeddedFS, path)
	if err != nil {
		t.Fatalf("expected %s in embedded FS: %v", path, err)
	}

	content := string(data)
	for _, want := range []string{
		"id: coverage-gap",
		"description:",
		"## Triggers",
		"## Workflow",
		"## Validation",
		"## Examples",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("embedded coverage-gap skill missing %q", want)
		}
	}
}
