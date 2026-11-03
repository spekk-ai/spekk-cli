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
	for _, want := range []string{
		"name: spekk-dev-loop",
		"spekk skill show builder review",
		"spekk builder review",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("embedded spekk-dev-loop skill missing %q", want)
		}
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

// TestEmbeddedFS_BuilderReviewSkill verifies the review builder skill ships
// with the binary via the embedded FS, and that its frontmatter and body
// carry the fields and the sections, in order, that the
// review-skill-markdown-exists assertion requires.
func TestEmbeddedFS_BuilderReviewSkill(t *testing.T) {
	const path = "specs/builder-skills/review-skill.md"

	data, err := fs.ReadFile(EmbeddedFS, path)
	if err != nil {
		t.Fatalf("expected %s in embedded FS: %v", path, err)
	}

	content := string(data)
	for _, want := range []string{"id: review", "description:"} {
		if !strings.Contains(content, want) {
			t.Errorf("embedded review skill missing %q", want)
		}
	}

	last := -1
	for _, heading := range []string{
		"## Triggers",
		"## Workflow",
		"## Output Format",
		"## Validation",
		"## Examples",
	} {
		idx := strings.Index(content, heading)
		if idx < 0 {
			t.Errorf("embedded review skill missing %q", heading)
			continue
		}
		if idx < last {
			t.Errorf("embedded review skill has %q out of order", heading)
		}
		last = idx
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

// checkPrompt reads one agent prompt from the embedded FS, the text that
// `spekk prompt <role>` prints when no override exists. It fails when the
// prompt uses "declarative" or misses a key text.
func checkPrompt(t *testing.T, role string, wants []string) {
	t.Helper()
	path := "specs/" + role + "-agent/" + role + ".prompt.md"
	data, err := fs.ReadFile(EmbeddedFS, path)
	if err != nil {
		t.Fatalf("expected %s in embedded FS: %v", path, err)
	}
	content := string(data)
	if strings.Contains(strings.ToLower(content), "declarativ") {
		t.Errorf("%s prompt uses \"declarative\"", role)
	}
	for _, want := range wants {
		if !strings.Contains(content, want) {
			t.Errorf("%s prompt missing %q", role, want)
		}
	}
}

// TestEmbeddedPrompt_CoachWritesProperties checks the key text of the
// coach-prompt-writes-properties assertion.
func TestEmbeddedPrompt_CoachWritesProperties(t *testing.T) {
	checkPrompt(t, "coach", []string{
		"steps do not compress, and they go stale first",
		"kind: invariant",
		"`invariant`, `ordering`, `response`, `authorization`, `example`",
		"## Rationale",
		"only the ledger writes `balance`",
		"status: not_started             # not_started | in_progress | done | failed | draft",
	})
}

// TestEmbeddedPrompt_BuilderBuildsEvidence checks the key text of the
// builder-prompt-builds-evidence assertion.
func TestEmbeddedPrompt_BuilderBuildsEvidence(t *testing.T) {
	checkPrompt(t, "builder", []string{
		"// spekk: balance-nonneg evidence=property",
		`@spekk("balance-nonneg", evidence="property")`,
		"spekk constraints <file>",
		"spekk verify",
		"enforcement point",
		"Spekk reads markers only in Go `_test.go` files and in pytest files (`test_*.py`, `*_test.py`). For a check in another language, keep the `**Tests:**` link and add no marker.",
	})
}

// TestEmbeddedPrompt_ObserverWatchesTruth checks the key text of the
// observer-prompt-watches-truth assertion.
func TestEmbeddedPrompt_ObserverWatchesTruth(t *testing.T) {
	checkPrompt(t, "observer", []string{
		"`failing` or `unconfirmed` is a drift candidate",
		"weak evidence",
		"enforcement point",
	})
}
