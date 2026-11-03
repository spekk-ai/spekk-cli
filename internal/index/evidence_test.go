package index_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

func TestEvidenceIndexAndHeadRefresh(t *testing.T) {
	root, specsDir, dbPath := newIndexRepo(t)
	assertionDir := filepath.Join(specsDir, "spec-a", "assertions")
	must(t, os.MkdirAll(assertionDir, 0o755))
	must(t, os.WriteFile(filepath.Join(assertionDir, "assert-a.md"), []byte("---\nid: assert-a\nparent: spec-a\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n---\n# Check\n"), 0o644))
	source := filepath.Join(root, "checks_test.go")
	must(t, os.WriteFile(source, []byte("// spekk: assert-a evidence=property\n// later doc line\nfunc TestCheck(t *testing.T) {}\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "test_checks.py"), []byte("@spekk('assert-a', evidence='contract')\ndef test_check(): pass\n"), 0o644))
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	db := openDB(t, dbPath)
	rows := func() []string {
		return queryStrings(t, db, `SELECT assertion_id || ':' || level || ':' || language || ':' || file || ':' || line || ':' || test FROM evidence ORDER BY file, line`)
	}
	if got, want := rows(), []string{"assert-a:property:go:checks_test.go:1:TestCheck", "assert-a:contract:python:test_checks.py:1:test_check"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("initial evidence = %v, want %v", got, want)
	}
	if countRows(t, db, "anchors") != 0 {
		t.Fatal("evidence marker added an anchor")
	}
	must(t, os.WriteFile(source, []byte("// spekk: assert-a\nfunc TestCheck(t *testing.T) {}\n"), 0o644))
	gitRun(t, root, "add", "checks_test.go")
	gitRun(t, root, "commit", "-q", "-m", "Change check")
	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	must(t, err)
	if !rebuilt {
		t.Fatal("new HEAD did not refresh evidence")
	}
	if got, want := rows(), []string{"assert-a:example:go:checks_test.go:1:TestCheck", "assert-a:contract:python:test_checks.py:1:test_check"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("refreshed evidence = %v, want %v", got, want)
	}
}
