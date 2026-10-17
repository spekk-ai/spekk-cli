package index_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

// writeFile writes content to root/rel and creates its directories.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	must(t, os.MkdirAll(filepath.Dir(p), 0o755))
	must(t, os.WriteFile(p, []byte(content), 0o644))
}

func TestBuildIndexHoldsSpecBodies(t *testing.T) {
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	writeFile(t, root, "specs/spec-a/assertions/assert-three.md", `---
id: assert-three
parent: spec-a
created: 2026-01-01T02:00:00Z
priority: 1
status: not_started
---
# Assert Three

## Success criteria

- The importer rejects a zanzibar header.
`)
	dbPath := filepath.Join(root, ".spekk", "index.db")
	stats, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	if stats.Passages == 0 {
		t.Fatalf("stats.Passages = 0, want passages for spec bodies")
	}

	db := openDB(t, dbPath)
	if n := countRows(t, db, "documents"); n != 4 {
		t.Errorf("documents rows = %d, want 4 (1 spec, 3 assertions)", n)
	}
	var kind, id string
	var start, end int
	err = db.QueryRow(`SELECT kind, doc_id, start_line, end_line FROM passages WHERE passages MATCH 'zanzibar'`).Scan(&kind, &id, &start, &end)
	must(t, err)
	// Line numbers are those of the file: 7 frontmatter lines, then the
	// title on line 8, which has no body and joins the next section, and
	// the bullet on line 12.
	if kind != "assertion" || id != "assert-three" || start != 8 || end != 12 {
		t.Errorf("match = %s %s %d-%d, want assertion assert-three 8-12", kind, id, start, end)
	}
}
