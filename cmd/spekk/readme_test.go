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
