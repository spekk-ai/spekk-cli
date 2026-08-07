package index_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

// makeSpecs creates a minimal specs directory for testing.
// It returns the directory path.
func makeSpecs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	// spec-a/spec-a.md
	specDir := filepath.Join(specsDir, "spec-a")
	assertDir := filepath.Join(specDir, "assertions")
	must(t, os.MkdirAll(assertDir, 0o755))
	must(t, os.WriteFile(filepath.Join(specDir, "spec-a.md"), []byte(`---
id: spec-a
created: 2026-01-01T00:00:00Z
priority: 1
---
# Spec A
`), 0o644))
	must(t, os.WriteFile(filepath.Join(assertDir, "assert-one.md"), []byte(`---
id: assert-one
parent: spec-a
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
---
# Assert One
`), 0o644))
	must(t, os.WriteFile(filepath.Join(assertDir, "assert-two.md"), []byte(`---
id: assert-two
parent: spec-a
created: 2026-01-01T01:00:00Z
priority: 1
status: not_started
depends-on: assert-one
---
# Assert Two
`), 0o644))
	return specsDir
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func openDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("cannot open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestBuildIndex verifies DB creation and table population.
func TestBuildIndex(t *testing.T) {
	specsDir := makeSpecs(t)
	repoRoot := filepath.Dir(specsDir)
	dbPath := index.DBPath(repoRoot)

	stats, err := index.BuildIndex(specsDir, dbPath, false)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	if stats.Specs != 1 {
		t.Errorf("expected 1 spec, got %d", stats.Specs)
	}
	if stats.Assertions != 2 {
		t.Errorf("expected 2 assertions, got %d", stats.Assertions)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("index.db not created: %v", err)
	}

	db := openDB(t, dbPath)
	if n := countRows(t, db, "specs"); n != 1 {
		t.Errorf("specs table: expected 1 row, got %d", n)
	}
	if n := countRows(t, db, "assertions"); n != 2 {
		t.Errorf("assertions table: expected 2 rows, got %d", n)
	}
	// One depends_on edge (assert-two depends on assert-one).
	if n := countRows(t, db, "depends_on"); n != 1 {
		t.Errorf("depends_on table: expected 1 row, got %d", n)
	}

	// Verify parent_id is set correctly.
	var parentID string
	if err := db.QueryRow(`SELECT parent_id FROM assertions WHERE id = 'assert-one'`).Scan(&parentID); err != nil {
		t.Fatalf("query parent_id: %v", err)
	}
	if parentID != "spec-a" {
		t.Errorf("expected parent_id 'spec-a', got %q", parentID)
	}
}

// TestBuildIndexIdempotent verifies running twice gives the same result.
func TestBuildIndexIdempotent(t *testing.T) {
	specsDir := makeSpecs(t)
	repoRoot := filepath.Dir(specsDir)
	dbPath := index.DBPath(repoRoot)

	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("first BuildIndex: %v", err)
	}
	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("second BuildIndex: %v", err)
	}

	db := openDB(t, dbPath)
	if n := countRows(t, db, "specs"); n != 1 {
		t.Errorf("after second run: specs expected 1 row, got %d", n)
	}
	if n := countRows(t, db, "assertions"); n != 2 {
		t.Errorf("after second run: assertions expected 2 rows, got %d", n)
	}
}

// TestBuildIndexForce verifies --force drops and recreates tables.
func TestBuildIndexForce(t *testing.T) {
	specsDir := makeSpecs(t)
	repoRoot := filepath.Dir(specsDir)
	dbPath := index.DBPath(repoRoot)

	// Build once, then force rebuild.
	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("initial BuildIndex: %v", err)
	}
	if _, err := index.BuildIndex(specsDir, dbPath, true); err != nil {
		t.Fatalf("force BuildIndex: %v", err)
	}

	db := openDB(t, dbPath)
	if n := countRows(t, db, "specs"); n != 1 {
		t.Errorf("after force: specs expected 1, got %d", n)
	}
	if n := countRows(t, db, "assertions"); n != 2 {
		t.Errorf("after force: assertions expected 2, got %d", n)
	}
}

// TestIsStale verifies the staleness check logic using synthetic mtimes.
func TestIsStale(t *testing.T) {
	specsDir := makeSpecs(t)
	repoRoot := filepath.Dir(specsDir)
	dbPath := index.DBPath(repoRoot)

	// DB absent → stale.
	stale, err := index.IsStale(specsDir, dbPath)
	if err != nil {
		t.Fatalf("IsStale (absent): %v", err)
	}
	if !stale {
		t.Error("expected stale=true when DB is absent")
	}

	// Build the index so DB exists.
	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	base := time.Now()
	specFile := filepath.Join(specsDir, "spec-a", "spec-a.md")

	// Set spec file mtime to base-10s and DB mtime to base (DB is newer → not stale).
	specOld := base.Add(-10 * time.Second)
	dbNew := base
	if err := os.Chtimes(specFile, specOld, specOld); err != nil {
		t.Fatalf("Chtimes specFile: %v", err)
	}
	if err := os.Chtimes(dbPath, dbNew, dbNew); err != nil {
		t.Fatalf("Chtimes dbPath: %v", err)
	}

	stale, err = index.IsStale(specsDir, dbPath)
	if err != nil {
		t.Fatalf("IsStale (fresh): %v", err)
	}
	if stale {
		t.Error("expected stale=false when DB is newer than all spec files")
	}

	// Now touch spec file to base+10s (newer than DB) → stale.
	specNew := base.Add(10 * time.Second)
	if err := os.Chtimes(specFile, specNew, specNew); err != nil {
		t.Fatalf("Chtimes specFile (make newer): %v", err)
	}

	stale, err = index.IsStale(specsDir, dbPath)
	if err != nil {
		t.Fatalf("IsStale (stale): %v", err)
	}
	if !stale {
		t.Error("expected stale=true when a spec file is newer than the DB")
	}
}

// TestEnsureGitignored verifies that .spekk/index.db is added to .gitignore.
func TestEnsureGitignored(t *testing.T) {
	dir := t.TempDir()

	// Case 1: no .gitignore exists.
	if err := index.EnsureGitignored(dir); err != nil {
		t.Fatalf("EnsureGitignored: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(content) != ".spekk/index.db\n" {
		t.Errorf("unexpected .gitignore content: %q", string(content))
	}

	// Case 2: already present — no duplicate.
	if err := index.EnsureGitignored(dir); err != nil {
		t.Fatalf("second EnsureGitignored: %v", err)
	}
	content2, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if string(content2) != ".spekk/index.db\n" {
		t.Errorf("duplicate entry added: %q", string(content2))
	}
}

// setUserVersion stamps an arbitrary user_version onto an existing index, to
// simulate a database built by a different schema version.
func setUserVersion(t *testing.T, dbPath string, v int) {
	t.Helper()
	db := openDB(t, dbPath)
	defer db.Close()
	if _, err := db.Exec("PRAGMA user_version = " + strconv.Itoa(v)); err != nil {
		t.Fatalf("set user_version: %v", err)
	}
}

func getUserVersion(t *testing.T, dbPath string) int {
	t.Helper()
	db := openDB(t, dbPath)
	defer db.Close()
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	return v
}

// TestEnsureFreshBuildsWhenAbsent: no db yet → EnsureFresh builds it.
func TestEnsureFreshBuildsWhenAbsent(t *testing.T) {
	specsDir := makeSpecs(t)
	dbPath := index.DBPath(filepath.Dir(specsDir))

	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if !rebuilt {
		t.Errorf("expected rebuilt=true for absent index")
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("index not created: %v", err)
	}
}

// TestEnsureFreshNoRebuildWhenFresh: freshly built + current schema → no rebuild.
func TestEnsureFreshNoRebuildWhenFresh(t *testing.T) {
	specsDir := makeSpecs(t)
	dbPath := index.DBPath(filepath.Dir(specsDir))
	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if rebuilt {
		t.Errorf("expected no rebuild for a fresh, current-schema index")
	}
}

// TestEnsureFreshRebuildsOnStaleSpecs: a spec modified after the build → rebuild.
func TestEnsureFreshRebuildsOnStaleSpecs(t *testing.T) {
	specsDir := makeSpecs(t)
	dbPath := index.DBPath(filepath.Dir(specsDir))
	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	// Make a spec newer than the db.
	future := time.Now().Add(2 * time.Hour)
	specFile := filepath.Join(specsDir, "spec-a", "spec-a.md")
	if err := os.Chtimes(specFile, future, future); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if !rebuilt {
		t.Errorf("expected rebuild when specs are newer than the index")
	}
}

// TestEnsureFreshRebuildsOnSchemaMismatch: a db stamped with a foreign schema
// version is force-rebuilt, and the version is reset to the current one. This is
// the migration path that makes `spekk update` self-healing.
func TestEnsureFreshRebuildsOnSchemaMismatch(t *testing.T) {
	specsDir := makeSpecs(t)
	dbPath := index.DBPath(filepath.Dir(specsDir))
	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	// Simulate an index built by a different schema version.
	setUserVersion(t, dbPath, 999)

	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if !rebuilt {
		t.Errorf("expected rebuild on schema-version mismatch")
	}
	if v := getUserVersion(t, dbPath); v == 999 {
		t.Errorf("schema version was not reset (still 999); rebuild did not re-stamp")
	}
	// Data is still queryable after the forced rebuild.
	db := openDB(t, dbPath)
	defer db.Close()
	if n := countRows(t, db, "assertions"); n != 2 {
		t.Errorf("assertions after rebuild: expected 2, got %d", n)
	}
}

// makeSpecsWithCustomFields creates a specs directory whose spec and
// assertion carry custom frontmatter fields in the three multi-value
// spellings (comma scalar, flow sequence, block list).
func makeSpecsWithCustomFields(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	specDir := filepath.Join(specsDir, "spec-a")
	assertDir := filepath.Join(specDir, "assertions")
	must(t, os.MkdirAll(assertDir, 0o755))
	must(t, os.WriteFile(filepath.Join(specDir, "spec-a.md"), []byte(`---
id: spec-a
created: 2026-01-01T00:00:00Z
priority: 1
workflows: w1-note-and-claim, w2-claim-reimbursement
---
# Spec A
`), 0o644))
	must(t, os.WriteFile(filepath.Join(assertDir, "assert-one.md"), []byte(`---
id: assert-one
parent: spec-a
created: 2026-01-01T00:00:00Z
priority: 1
status: done
workflows: w1-note-and-claim
tags: [infrastructure, hipaa]
reviewers:
- alice
- bob
---
# Assert One
`), 0o644))
	return specsDir
}

func TestBuildIndexFrontmatterFields(t *testing.T) {
	specsDir := makeSpecsWithCustomFields(t)
	dbPath := filepath.Join(filepath.Dir(specsDir), ".spekk", "index.db")

	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex failed: %v", err)
	}

	db := openDB(t, dbPath)

	type row struct{ ownerType, ownerID, key, value string }
	want := []row{
		{"assertion", "assert-one", "reviewers", "alice"},
		{"assertion", "assert-one", "reviewers", "bob"},
		{"assertion", "assert-one", "tags", "hipaa"},
		{"assertion", "assert-one", "tags", "infrastructure"},
		{"assertion", "assert-one", "workflows", "w1-note-and-claim"},
		{"spec", "spec-a", "workflows", "w1-note-and-claim"},
		{"spec", "spec-a", "workflows", "w2-claim-reimbursement"},
	}

	rows, err := db.Query(`SELECT owner_type, owner_id, key, value FROM frontmatter_fields ORDER BY owner_type, owner_id, key, value`)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	var got []row
	for rows.Next() {
		var r row
		must(t, rows.Scan(&r.ownerType, &r.ownerID, &r.key, &r.value))
		got = append(got, r)
	}
	must(t, rows.Err())

	if len(got) != len(want) {
		t.Fatalf("expected %d rows, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: expected %v, got %v", i, want[i], got[i])
		}
	}
}

func TestBuildIndexFrontmatterFieldsJoin(t *testing.T) {
	// The report from issue #165: percent-complete per workflow via one
	// SQL join.
	specsDir := makeSpecsWithCustomFields(t)
	dbPath := filepath.Join(filepath.Dir(specsDir), ".spekk", "index.db")

	if _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex failed: %v", err)
	}

	db := openDB(t, dbPath)
	var total, done int
	err := db.QueryRow(`SELECT COUNT(*), SUM(a.status = 'done')
		FROM assertions a
		JOIN frontmatter_fields f
		  ON f.owner_type = 'assertion' AND f.owner_id = a.id AND f.key = 'workflows'
		WHERE f.value = 'w1-note-and-claim'`).Scan(&total, &done)
	if err != nil {
		t.Fatalf("join query failed: %v", err)
	}
	if total != 1 || done != 1 {
		t.Errorf("expected total=1 done=1, got total=%d done=%d", total, done)
	}
}
