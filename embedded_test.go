package spekk

import (
	"io/fs"
	"os"
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

// TestEmbeddedFS_ObserverPruneSkill verifies the prune observer skill ships
// with the binary via the embedded FS, and that its frontmatter/body declare
// the id, description, and required sections per the assertion.
func TestEmbeddedFS_ObserverPruneSkill(t *testing.T) {
	const path = "specs/observer-skills/prune-skill.md"

	data, err := fs.ReadFile(EmbeddedFS, path)
	if err != nil {
		t.Fatalf("expected %s in embedded FS: %v", path, err)
	}

	content := string(data)
	for _, want := range []string{
		"id: prune",
		"description:",
		"## Triggers",
		"## Workflow",
		"## Output Format",
		"## Validation",
		"## Examples",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("embedded prune skill missing %q", want)
		}
	}
}

// TestPruneCandidateType_RegisteredInBothContractDocs verifies the
// prune_candidate observation type is registered in both places the
// Observation Output Contract's allowed-type list lives, per the
// prune-candidate-type-registered assertion. Read from disk (not the
// embedded FS) since observer-skill-discovery.md is a spec doc, not a
// packaged skill, and is never embedded.
func TestPruneCandidateType_RegisteredInBothContractDocs(t *testing.T) {
	for _, path := range []string{
		"specs/observer-agent/observer.prompt.md",
		"specs/observer-skill-discovery/observer-skill-discovery.md",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if !strings.Contains(string(data), "prune_candidate") {
			t.Errorf("%s: expected allowed-type list to include \"prune_candidate\"", path)
		}
	}
}

// TestEmbeddedFS_CoachPropertyTestsSkill verifies the property-tests coach
// skill ships with the binary via the embedded FS, and that it declares the
// id and the sections the assertion requires.
func TestEmbeddedFS_CoachPropertyTestsSkill(t *testing.T) {
	const path = "specs/coach-skills-system/property-tests-skill.md"

	data, err := fs.ReadFile(EmbeddedFS, path)
	if err != nil {
		t.Fatalf("expected %s in embedded FS: %v", path, err)
	}

	content := string(data)
	for _, want := range []string{
		"id: property-tests",
		"## Triggers",
		"## Workflow",
		"## Validation",
		"value gate",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("embedded property-tests skill missing %q", want)
		}
	}
}
