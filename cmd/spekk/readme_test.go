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
