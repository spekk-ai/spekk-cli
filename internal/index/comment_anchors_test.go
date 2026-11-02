package index_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

func TestCommentAnchorsIndexAndRefreshAtNewHead(t *testing.T) {
	root, specsDir, dbPath := newIndexRepo(t)
	assertionsDir := filepath.Join(specsDir, "spec-a", "assertions")
	must(t, os.MkdirAll(assertionsDir, 0o755))
	must(t, os.WriteFile(filepath.Join(assertionsDir, "known-id.md"), []byte(`---
id: known-id
parent: spec-a
created: 2026-01-01T00:00:00Z
priority: 1
status: done
anchors: [source.py]
---
# Known ID
`), 0o644))
	gitRun(t, root, "add", "specs")
	gitRun(t, root, "commit", "-q", "-m", "Add property")
	source := filepath.Join(root, "source.py")
	must(t, os.WriteFile(source, []byte("# spekk: known-id\n# spekk: missing-id\n"), 0o644))
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	db := openDB(t, dbPath)
	rows := func() []string {
		return queryStrings(t, db, `SELECT assertion_id || ':' || path || ':' || COALESCE(CAST(line AS TEXT), 'NULL') || ':' || source FROM anchors ORDER BY source`)
	}
	if got, want := rows(), []string{"known-id:source.py:1:comment", "known-id:source.py:NULL:frontmatter"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("initial anchors = %v, want %v", got, want)
	}
	if got := queryStrings(t, db, `SELECT value FROM index_meta WHERE key = 'anchor_head'`); !reflect.DeepEqual(got, []string{gitRun(t, root, "rev-parse", "HEAD")}) {
		t.Fatalf("stored HEAD = %v", got)
	}
	must(t, os.WriteFile(source, []byte("# spekk: known-id\n# spekk: known-id\n"), 0o644))
	gitRun(t, root, "add", "source.py")
	gitRun(t, root, "commit", "-q", "-m", "Change source")
	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	must(t, err)
	if !rebuilt {
		t.Fatal("a new HEAD did not rebuild the index")
	}
	if got, want := rows(), []string{"known-id:source.py:1:comment", "known-id:source.py:2:comment", "known-id:source.py:NULL:frontmatter"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("anchors after HEAD change = %v, want %v", got, want)
	}
}
