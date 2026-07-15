package main

import (
	"fmt"
	"strings"
	"testing"
)

// TestRenderManagedReadmeBlock_Markers checks the fence markers are present
// verbatim, each on its own line.
func TestRenderManagedReadmeBlock_Markers(t *testing.T) {
	block := renderManagedReadmeBlock()

	for _, line := range []string{readmeManagedBeginMarker, readmeManagedEndMarker} {
		found := false
		for _, l := range strings.Split(block, "\n") {
			if l == line {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected a line exactly equal to %q in managed block, got:\n%s", line, block)
		}
	}
}

// TestRenderManagedReadmeBlock_FrontmatterFields asserts every spec and
// assertion frontmatter field documented by internal/parser is mentioned,
// along with every valid status value and the schema version line.
func TestRenderManagedReadmeBlock_FrontmatterFields(t *testing.T) {
	block := renderManagedReadmeBlock()

	fields := []string{"id", "created", "priority", "status", "branch", "parent", "depends-on", "locked-by"}
	for _, f := range fields {
		if !strings.Contains(block, f) {
			t.Errorf("expected managed block to mention frontmatter field %q", f)
		}
	}

	statuses := []string{"not_started", "in_progress", "done", "draft", "failed"}
	for _, s := range statuses {
		if !strings.Contains(block, s) {
			t.Errorf("expected managed block to mention status value %q", s)
		}
	}

	if !strings.Contains(block, fmt.Sprintf("spekk_schema_version: %d", specSchemaVersion)) {
		t.Errorf("expected managed block to contain the spekk_schema_version line")
	}
}

// TestRenderManagedReadmeBlock_Pure asserts the block is a pure function of
// specSchemaVersion: it contains no per-spec state, and rendering it twice
// yields byte-identical output.
func TestRenderManagedReadmeBlock_Pure(t *testing.T) {
	first := renderManagedReadmeBlock()
	second := renderManagedReadmeBlock()
	if first != second {
		t.Fatalf("expected renderManagedReadmeBlock to be pure, got two different renders:\n---\n%s\n---\n%s", first, second)
	}
}

// TestSpliceManagedReadmeBlock_PreservesOuterProse asserts that splicing a
// new managed block into a README replaces only the span from the begin
// marker line through the end marker line, leaving the surrounding human
// prose byte-for-byte untouched.
func TestSpliceManagedReadmeBlock_PreservesOuterProse(t *testing.T) {
	before := "# My Project\n\nSome human intro that spekk does not own.\n\n"
	after := "\n## Human Notes\n\nDon't touch this section.\n"
	oldBlock := readmeManagedBeginMarker + "\nold body\n" + readmeManagedEndMarker + "\n"
	content := before + oldBlock + after

	newBlock := renderManagedReadmeBlock()
	got, ok := spliceManagedReadmeBlock(content, newBlock)
	if !ok {
		t.Fatalf("expected a well-formed fence to be found in:\n%s", content)
	}

	want := before + newBlock + after
	if got != want {
		t.Fatalf("splice did not preserve outer prose / swap the managed body correctly\ngot:\n%q\nwant:\n%q", got, want)
	}
}

// TestReplaceManagedReadmeBlock_Idempotent is the headline idempotency
// guarantee: regenerating a README that already contains the current
// render, with no change to specSchemaVersion, must be a byte-exact no-op,
// and regenerating twice in a row must produce identical output both times.
func TestReplaceManagedReadmeBlock_Idempotent(t *testing.T) {
	before := "# Notes\n\nHuman prose before the fence.\n\n"
	after := "\nHuman prose after the fence.\n"
	content := before + renderManagedReadmeBlock() + after

	first, ok := replaceManagedReadmeBlock(content)
	if !ok {
		t.Fatalf("expected a well-formed fence to be found in:\n%s", content)
	}
	if first != content {
		t.Fatalf("expected regeneration with an unchanged schema version to be a no-op\ngot:\n%q\nwant:\n%q", first, content)
	}

	second, ok := replaceManagedReadmeBlock(first)
	if !ok {
		t.Fatalf("expected a well-formed fence to be found on the second pass")
	}
	if second != first {
		t.Fatalf("expected two consecutive regenerations to be byte-identical\nfirst:\n%q\nsecond:\n%q", first, second)
	}
}

// legacyStaticReadme is the historical static specs/README.md content
// written by spekk init before the managed-fence mechanism existed (see
// commit 2d3cfb5, which deleted this exact constant from main.go). It has
// no markers at all, so it is the canonical fixture for the
// no-fence/legacy-upgrade path.
const legacyStaticReadme = `# Specs

This directory is a work queue for AI agents, managed with
[spekk](https://github.com/spekk-ai/spekk-cli).

Each spec is a folder containing a markdown file that states what must be
true, plus an assertions/ folder breaking that down into small, testable
assertions:

    specs/
      my-feature/
        my-feature.md          # what must be true, and why
        assertions/
          first-assertion.md   # one small, verifiable step

Common commands:

    spekk coach      # draft and refine specs with the coach agent
    spekk builder    # implement the next ready assertion
    spekk next       # print the next ready assertion
    spekk status     # overview of all specs and assertions
`

// TestRegenerateReadmeContent_FenceStates is a table-driven test covering
// every fence state fence-state-handling must handle: the legacy/static
// no-fence upgrade, each enumerated corrupt arrangement, and the
// already-well-formed happy path (which must not grow a second region).
// For every case it asserts: exactly one well-formed region exists after
// one pass, the prose outside any marker survives, and a second pass is
// byte-identical to the first (convergence).
func TestRegenerateReadmeContent_FenceStates(t *testing.T) {
	humanBefore := "# My Project\n\nHuman intro that spekk does not own.\n"
	humanAfter := "## Human Notes\n\nDon't touch this section.\n"
	oldBody := "stale schema docs from a previous spekk version\n"

	cases := map[string]struct {
		content        string
		wantContains   []string // outside-marker prose that must survive
		wantNotContain []string // CLI-owned/disposable text that must NOT survive
	}{
		"legacy static README, no markers": {
			content:      legacyStaticReadme,
			wantContains: []string{"Each spec is a folder containing a markdown file"},
		},
		"only a begin marker (no outro)": {
			content: humanBefore + readmeManagedBeginMarker + "\n" + oldBody,
			// No end marker anywhere: the span the begin marker would
			// have enclosed runs to EOF, so the stale body trailing it is
			// disposable CLI-owned content, not recognizable human prose.
			wantContains:   []string{humanBefore},
			wantNotContain: []string{oldBody},
		},
		"only an end marker (no intro)": {
			content: oldBody + readmeManagedEndMarker + "\n" + humanAfter,
			// No begin marker anywhere: the span runs back to the start
			// of the file, so the stale body preceding it is disposable.
			wantContains:   []string{humanAfter},
			wantNotContain: []string{oldBody},
		},
		"duplicate begin markers": {
			content: humanBefore +
				readmeManagedBeginMarker + "\n" + oldBody +
				readmeManagedBeginMarker + "\n" + oldBody +
				readmeManagedEndMarker + "\n" + humanAfter,
			wantContains:   []string{humanBefore, humanAfter},
			wantNotContain: []string{oldBody},
		},
		"duplicate end markers": {
			content: humanBefore +
				readmeManagedBeginMarker + "\n" + oldBody +
				readmeManagedEndMarker + "\n" + oldBody +
				readmeManagedEndMarker + "\n" + humanAfter,
			wantContains:   []string{humanBefore, humanAfter},
			wantNotContain: []string{oldBody},
		},
		"end before begin": {
			content: humanBefore +
				readmeManagedEndMarker + "\n" + oldBody +
				readmeManagedBeginMarker + "\n" + humanAfter,
			wantContains:   []string{humanBefore, humanAfter},
			wantNotContain: []string{oldBody},
		},
		"already well-formed (no second region grown)": {
			content:      humanBefore + renderManagedReadmeBlock() + humanAfter,
			wantContains: []string{humanBefore, humanAfter},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			first, _ := regenerateReadmeContent(tc.content)

			if beginCount := strings.Count(first, readmeManagedBeginMarker); beginCount != 1 {
				t.Fatalf("expected exactly one begin marker after regeneration, got %d in:\n%q", beginCount, first)
			}
			if endCount := strings.Count(first, readmeManagedEndMarker); endCount != 1 {
				t.Fatalf("expected exactly one end marker after regeneration, got %d in:\n%q", endCount, first)
			}
			beginIdx := strings.Index(first, readmeManagedBeginMarker)
			endIdx := strings.Index(first, readmeManagedEndMarker)
			if beginIdx >= endIdx {
				t.Fatalf("expected begin marker before end marker in:\n%q", first)
			}
			if strings.HasSuffix(first, "\n\n") || !strings.HasSuffix(first, "\n") {
				t.Fatalf("expected result to end in exactly one trailing newline, got tail %q", first[max(0, len(first)-10):])
			}

			for _, want := range tc.wantContains {
				if !strings.Contains(first, want) {
					t.Errorf("expected outside-marker prose to survive, missing %q in:\n%q", want, first)
				}
			}
			for _, notWant := range tc.wantNotContain {
				if strings.Contains(first, notWant) {
					t.Errorf("expected CLI-owned/disposable text to be stripped, still found %q in:\n%q", notWant, first)
				}
			}

			second, _ := regenerateReadmeContent(first)
			if second != first {
				t.Fatalf("expected a second pass to be byte-identical (convergence)\nfirst:\n%q\nsecond:\n%q", first, second)
			}
		})
	}
}

// TestRegenerateReadmeContent_LegacyUpgradeAppendsOneRegion asserts the
// no-fence upgrade path in detail: the entire legacy file is preserved
// byte-for-byte, and exactly one blank line separates it from a freshly
// appended, well-formed managed region.
func TestRegenerateReadmeContent_LegacyUpgradeAppendsOneRegion(t *testing.T) {
	updated, changed := regenerateReadmeContent(legacyStaticReadme)
	if !changed {
		t.Fatalf("expected upgrading a legacy README to report a change")
	}

	want := strings.TrimRight(legacyStaticReadme, "\n") + "\n\n" + renderManagedReadmeBlock()
	if updated != want {
		t.Fatalf("expected legacy content preserved + one appended region\ngot:\n%q\nwant:\n%q", updated, want)
	}
}

// TestSpliceManagedReadmeBlock_VersionChangePreservesOuterProse simulates a
// schema version bump (a different rendered managed block) and asserts the
// splice still replaces only the fenced span, preserving human prose outside
// it verbatim rather than corrupting the file.
func TestSpliceManagedReadmeBlock_VersionChangePreservesOuterProse(t *testing.T) {
	before := "# Notes\n\nHuman prose before the fence.\n\n"
	after := "\nHuman prose after the fence.\n"
	oldBlock := readmeManagedBeginMarker + "\nschema v1 body\n" + readmeManagedEndMarker + "\n"
	newBlock := readmeManagedBeginMarker + "\nschema v2 body\n" + readmeManagedEndMarker + "\n"
	content := before + oldBlock + after

	got, ok := spliceManagedReadmeBlock(content, newBlock)
	if !ok {
		t.Fatalf("expected a well-formed fence to be found in:\n%s", content)
	}

	want := before + newBlock + after
	if got != want {
		t.Fatalf("expected version-bumped block to replace old block while preserving outer prose\ngot:\n%q\nwant:\n%q", got, want)
	}
}
