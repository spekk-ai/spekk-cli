package index_test

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

// gitRun runs a git command in dir, failing the test on error.
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// chdirTest switches the working directory for the test and restores it.
func chdirTest(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

const observationTemplate = `---
slug: SLUG
type: code_spec_misalignment
severity: SEVERITY
status: STATUS
created: 2026-07-26T12:00:00Z
ANNOUNCEDaffected:
  - internal/parser/parser.go
  - specs/spec-a/spec-a.md
---

# Finding SLUG

Evidence body.
`

func observationContent(slug, severity, status, announced string) string {
	s := strings.NewReplacer(
		"SLUG", slug, "SEVERITY", severity, "STATUS", status,
	).Replace(observationTemplate)
	if announced != "" {
		announced = "announced: " + announced + "\n"
	}
	return strings.Replace(s, "ANNOUNCED", announced, 1)
}

// newIndexRepo creates a temp git repo on main containing a minimal specs
// tree, chdirs into it, and returns (repoRoot, specsDir, dbPath).
func newIndexRepo(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test")
	gitRun(t, dir, "config", "commit.gpgsign", "false")

	specDir := filepath.Join(dir, "specs", "spec-a")
	must(t, os.MkdirAll(specDir, 0o755))
	must(t, os.WriteFile(filepath.Join(specDir, "spec-a.md"), []byte(`---
id: spec-a
created: 2026-01-01T00:00:00Z
priority: 1
---
# Spec A
`), 0o644))
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "init specs")
	chdirTest(t, dir)
	return dir, filepath.Join(dir, "specs"), index.DBPath(dir)
}

// addObserverBranch commits an observation on observer/<slug> and returns to
// main.
func addObserverBranch(t *testing.T, dir, slug, severity, status, announced string) {
	t.Helper()
	gitRun(t, dir, "checkout", "-q", "-b", "observer/"+slug, "main")
	path := filepath.Join(dir, "observations", slug+".md")
	must(t, os.MkdirAll(filepath.Dir(path), 0o755))
	must(t, os.WriteFile(path, []byte(observationContent(slug, severity, status, announced)), 0o644))
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "observer: add "+slug)
	gitRun(t, dir, "checkout", "-q", "main")
}

// queryStrings runs a single-column query and returns the values, with NULL
// rendered as "<nil>".
func queryStrings(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v sql.NullString
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		if v.Valid {
			out = append(out, v.String)
		} else {
			out = append(out, "<nil>")
		}
	}
	return out
}

func TestObservationTablesPopulated(t *testing.T) {
	dir, specsDir, dbPath := newIndexRepo(t)
	addObserverBranch(t, dir, "finding-a", "high", "open", "")
	addObserverBranch(t, dir, "finding-b", "medium", "open", "2026-07-26T13:05:00Z")

	stats, err := index.BuildIndex(specsDir, dbPath, false)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if stats.Observations != 2 {
		t.Fatalf("expected 2 observations indexed, got %d", stats.Observations)
	}

	db := openDB(t, dbPath)

	// One row per (slug, ref); announced is SQL NULL when absent.
	got := queryStrings(t, db, "SELECT announced FROM observations WHERE slug = 'finding-a'")
	if len(got) != 1 || got[0] != "<nil>" {
		t.Fatalf("finding-a announced must be NULL, got %v", got)
	}
	got = queryStrings(t, db, "SELECT announced FROM observations WHERE slug = 'finding-b'")
	if len(got) != 1 || got[0] != "2026-07-26T13:05:00Z" {
		t.Fatalf("finding-b announced: %v", got)
	}
	// The eligibility predicate works directly in SQL.
	got = queryStrings(t, db, "SELECT slug FROM observations WHERE announced IS NULL")
	if len(got) != 1 || got[0] != "finding-a" {
		t.Fatalf("announced IS NULL must match only finding-a, got %v", got)
	}

	// Evidence rows are joinable on (slug, ref).
	got = queryStrings(t, db, `SELECT of.path FROM observation_files of
		JOIN observations o ON o.slug = of.slug AND o.ref = of.ref
		WHERE o.slug = 'finding-a' ORDER BY of.path`)
	if len(got) != 2 || got[0] != "internal/parser/parser.go" {
		t.Fatalf("evidence rows: %v", got)
	}
}

func TestObservationSameSlugMultipleRefs(t *testing.T) {
	dir, specsDir, dbPath := newIndexRepo(t)
	addObserverBranch(t, dir, "merged-finding", "high", "open", "")

	// Merge the observation to main (resolved) while the branch stays.
	resolved := observationContent("merged-finding", "high", "resolved", "")
	must(t, os.MkdirAll(filepath.Join(dir, "observations"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "observations", "merged-finding.md"), []byte(resolved), 0o644))
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "merge merged-finding")

	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	db := openDB(t, dbPath)
	got := queryStrings(t, db, "SELECT ref FROM observations WHERE slug = 'merged-finding' ORDER BY ref")
	if len(got) != 2 {
		t.Fatalf("expected one row per ref, got %v", got)
	}
}

func TestObservationInvalidFileSkippedWithWarning(t *testing.T) {
	dir, specsDir, dbPath := newIndexRepo(t)
	// Evidence gate failure: affected list removed.
	gitRun(t, dir, "checkout", "-q", "-b", "observer/no-evidence", "main")
	bad := strings.Split(observationContent("no-evidence", "high", "open", ""), "affected:")[0] + "---\n\n# Bad\n"
	must(t, os.MkdirAll(filepath.Join(dir, "observations"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "observations", "no-evidence.md"), []byte(bad), 0o644))
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-q", "-m", "bad observation")
	gitRun(t, dir, "checkout", "-q", "main")

	stats, err := index.BuildIndex(specsDir, dbPath, false)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if stats.Observations != 0 {
		t.Fatalf("invalid observation must not be indexed, got %d rows", stats.Observations)
	}
	if len(stats.Warnings) != 1 || !strings.Contains(stats.Warnings[0], "no-evidence.md") {
		t.Fatalf("expected a warning naming the file, got %v", stats.Warnings)
	}
	db := openDB(t, dbPath)
	if n := countRows(t, db, "observations"); n != 0 {
		t.Fatalf("observations table must be empty, got %d", n)
	}
}

// TestObservationIndexRoundTrip demonstrates the derived-only invariant for
// the observation tables: deleting index.db and rebuilding reproduces
// equivalent contents from the repo and its visible refs.
func TestObservationIndexRoundTrip(t *testing.T) {
	dir, specsDir, dbPath := newIndexRepo(t)
	addObserverBranch(t, dir, "finding-a", "high", "open", "")
	addObserverBranch(t, dir, "finding-b", "low", "open", "2026-07-26T13:05:00Z")

	dump := func() []string {
		db := openDB(t, dbPath)
		rows := queryStrings(t, db, `SELECT slug || '|' || ref || '|' || type || '|' || severity || '|' ||
			status || '|' || created || '|' || COALESCE(announced,'∅') || '|' || file
			FROM observations ORDER BY slug, ref`)
		rows = append(rows, queryStrings(t, db, `SELECT slug || '|' || ref || '|' || path
			FROM observation_files ORDER BY slug, ref, path`)...)
		return rows
	}

	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	before := dump()
	if len(before) == 0 {
		t.Fatal("expected observation rows before the round trip")
	}

	must(t, os.Remove(dbPath))
	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("rebuild after delete: %v", err)
	}
	after := dump()

	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("round trip changed contents:\nbefore:\n%s\nafter:\n%s",
			strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
}

// TestEnsureFreshRebuildsOnNewObserverBranch verifies the staleness gate
// accounts for observation sources: a new observer branch (as a fetch would
// create) triggers a rebuild that picks up its observation.
func TestEnsureFreshRebuildsOnNewObserverBranch(t *testing.T) {
	dir, specsDir, dbPath := newIndexRepo(t)

	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if rebuilt, err := index.EnsureFresh(specsDir, dbPath); err != nil || rebuilt {
		t.Fatalf("fresh index must not rebuild (rebuilt=%v err=%v)", rebuilt, err)
	}

	addObserverBranch(t, dir, "new-finding", "high", "open", "")
	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if !rebuilt {
		t.Fatal("new observer branch must trigger a rebuild")
	}
	db := openDB(t, dbPath)
	got := queryStrings(t, db, "SELECT slug FROM observations")
	if len(got) != 1 || got[0] != "new-finding" {
		t.Fatalf("rebuild must pick up the new observation, got %v", got)
	}
}
