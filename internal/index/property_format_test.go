package index_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/index"
	"github.com/spekk-ai/spekk-cli/internal/parser"
	"github.com/spekk-ai/spekk-cli/internal/validate"
)

func TestV1TreeDefaults(t *testing.T) {
	specsDir := makeSpecs(t)
	gitRun(t, filepath.Dir(specsDir), "init", "-q")
	result, err := validate.Run(specsDir)
	must(t, err)
	if !result.Passed() || len(result.Warnings) != 0 || result.SpecCount != 1 || result.AssertionCount != 2 {
		t.Fatalf("v1 validation = %#v", result)
	}
	parsed, err := parser.ParseAllSpecs(specsDir)
	must(t, err)
	for _, a := range parsed.Assertions {
		if a.Kind != "example" || len(a.Anchors) != 0 || len(a.Sections) != 0 {
			t.Fatalf("v1 property = %#v", a)
		}
	}
	dbPath := index.DBPath(filepath.Dir(specsDir))
	_, err = index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	db := openDB(t, dbPath)
	if got := queryStrings(t, db, `SELECT kind FROM assertions ORDER BY id`); !reflect.DeepEqual(got, []string{"example", "example"}) {
		t.Fatalf("v1 kinds = %v", got)
	}
	if countRows(t, db, "anchors") != 0 || countRows(t, db, "property_sections") != 0 {
		t.Fatal("v1 assertions added anchors or sections")
	}
}

func TestPropertyIndexRebuildsV1Schema(t *testing.T) {
	specsDir := makeSpecs(t)
	dbPath := index.DBPath(filepath.Dir(specsDir))
	must(t, os.MkdirAll(filepath.Dir(dbPath), 0o755))
	db := openDB(t, dbPath)
	_, err := db.Exec(`CREATE TABLE assertions (id TEXT PRIMARY KEY, parent_id TEXT, title TEXT, status TEXT, priority INTEGER, branch TEXT, file TEXT)`)
	must(t, err)
	_, err = db.Exec(`PRAGMA user_version = 9`)
	must(t, err)
	_, err = index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	if got := queryStrings(t, db, `SELECT kind FROM assertions ORDER BY id`); !reflect.DeepEqual(got, []string{"example", "example"}) {
		t.Fatalf("rebuilt kinds = %v", got)
	}
}

func TestPropertyKindIndexKeepsCustomField(t *testing.T) {
	specsDir := makeSpecs(t)
	path := filepath.Join(specsDir, "spec-a", "assertions", "assert-one.md")
	raw, err := os.ReadFile(path)
	must(t, err)
	must(t, os.WriteFile(path, []byte(strings.Replace(string(raw), "---\n# Assert One\n", "kind: invariant\n---\n# Assert One\n", 1)), 0o644))
	dbPath := index.DBPath(filepath.Dir(specsDir))
	_, err = index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	db := openDB(t, dbPath)
	got := queryStrings(t, db, `SELECT id || ':' || kind FROM assertions ORDER BY id`)
	want := []string{"assert-one:invariant", "assert-two:example"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	got = queryStrings(t, db, `SELECT key || ':' || value FROM frontmatter_fields WHERE owner_id = 'assert-one'`)
	if !reflect.DeepEqual(got, []string{"kind:invariant"}) {
		t.Fatalf("custom kind field = %v", got)
	}
}

func TestFrontmatterAnchorsIndex(t *testing.T) {
	specsDir := makeSpecs(t)
	path := filepath.Join(specsDir, "spec-a", "assertions", "assert-one.md")
	raw, err := os.ReadFile(path)
	must(t, err)
	must(t, os.WriteFile(path, []byte(strings.Replace(string(raw), "---\n# Assert One\n", "anchors: [a.py, b.py]\n---\n# Assert One\n", 1)), 0o644))
	dbPath := index.DBPath(filepath.Dir(specsDir))
	_, err = index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	db := openDB(t, dbPath)
	got := queryStrings(t, db, `SELECT assertion_id || ':' || path || ':' || source || ':' || COALESCE(CAST(line AS TEXT), 'NULL') FROM anchors ORDER BY path`)
	want := []string{"assert-one:a.py:frontmatter:NULL", "assert-one:b.py:frontmatter:NULL"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("anchors = %v, want %v", got, want)
	}
	got = queryStrings(t, db, `SELECT value FROM frontmatter_fields WHERE owner_id = 'assert-one' AND key = 'anchors' ORDER BY value`)
	if !reflect.DeepEqual(got, []string{"a.py", "b.py"}) {
		t.Fatalf("custom anchor fields = %v", got)
	}
}

func TestPropertySectionsIndex(t *testing.T) {
	specsDir := makeSpecs(t)
	path := filepath.Join(specsDir, "spec-a", "assertions", "assert-one.md")
	raw, err := os.ReadFile(path)
	must(t, err)
	must(t, os.WriteFile(path, append(raw, []byte("\n## Decisions\n\nCommit only.\n")...), 0o644))
	dbPath := index.DBPath(filepath.Dir(specsDir))
	_, err = index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	db := openDB(t, dbPath)
	got := queryStrings(t, db, `SELECT assertion_id || ':' || section || ':' || text FROM property_sections`)
	if !reflect.DeepEqual(got, []string{"assert-one:decisions:Commit only."}) {
		t.Fatalf("sections = %v", got)
	}
}
