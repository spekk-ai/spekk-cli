package index_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

// buildTestDB creates an index from the standard makeSpecs fixture and returns the dbPath.
func buildTestDB(t *testing.T) string {
	t.Helper()
	specsDir := makeSpecs(t)
	repoRoot := filepath.Dir(specsDir)
	dbPath := index.DBPath(repoRoot)
	if _, _, err := index.BuildIndex(specsDir, dbPath, false); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	return dbPath
}

// TestRunQuery_SelectReturnsRows verifies a valid SELECT returns rows and columns.
func TestRunQuery_SelectReturnsRows(t *testing.T) {
	dbPath := buildTestDB(t)

	result, err := index.RunQuery(dbPath, "SELECT id, status FROM assertions")
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if len(result.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(result.Columns))
	}
	if len(result.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(result.Rows))
	}
}

// TestRunQuery_EmptyResult verifies empty result set returns columns but no rows.
func TestRunQuery_EmptyResult(t *testing.T) {
	dbPath := buildTestDB(t)

	result, err := index.RunQuery(dbPath, "SELECT id FROM specs WHERE id = 'nonexistent'")
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if len(result.Columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(result.Columns))
	}
	if len(result.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(result.Rows))
	}
}

// TestRunQuery_RejectsInsert verifies INSERT is rejected.
func TestRunQuery_RejectsInsert(t *testing.T) {
	dbPath := buildTestDB(t)

	_, err := index.RunQuery(dbPath, "INSERT INTO specs (id) VALUES ('x')")
	if err == nil {
		t.Fatal("expected error for INSERT, got nil")
	}
	if !strings.Contains(err.Error(), "only SELECT") {
		t.Errorf("expected 'only SELECT' error, got: %v", err)
	}
}

// TestRunQuery_RejectsDrop verifies DROP is rejected.
func TestRunQuery_RejectsDrop(t *testing.T) {
	dbPath := buildTestDB(t)

	_, err := index.RunQuery(dbPath, "DROP TABLE specs")
	if err == nil {
		t.Fatal("expected error for DROP, got nil")
	}
	if !strings.Contains(err.Error(), "only SELECT") {
		t.Errorf("expected 'only SELECT' error, got: %v", err)
	}
}

// TestRunQuery_NoDBError verifies a clear error when DB doesn't exist.
func TestRunQuery_NoDBError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), ".spekk", "index.db")
	// Ensure the file does NOT exist.
	os.Remove(dbPath)

	_, err := index.RunQuery(dbPath, "SELECT id FROM specs")
	if err == nil {
		t.Fatal("expected error for missing DB, got nil")
	}
	if err != index.ErrDBNotFound {
		t.Errorf("expected ErrDBNotFound, got: %v", err)
	}
}

// TestFormatQueryTable verifies the table formatter.
func TestFormatQueryTable(t *testing.T) {
	r := &index.QueryResult{
		Columns: []string{"id", "status"},
		Rows: [][]string{
			{"foo", "done"},
			{"bar-baz", "not_started"},
		},
	}
	out := index.FormatQueryTable(r)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d: %q", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "ID") {
		t.Errorf("header should start with ID, got %q", lines[0])
	}
	// No trailing whitespace on last column.
	if strings.HasSuffix(lines[1], " ") {
		t.Errorf("trailing whitespace on data row: %q", lines[1])
	}
}

// TestFormatQueryTSV verifies the TSV formatter.
func TestFormatQueryTSV(t *testing.T) {
	r := &index.QueryResult{
		Columns: []string{"id", "status"},
		Rows:    [][]string{{"foo", "done"}},
	}
	out := index.FormatQueryTSV(r)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "id\tstatus" {
		t.Errorf("header: expected 'id\\tstatus', got %q", lines[0])
	}
}

// TestFormatQueryCSV verifies the CSV formatter.
func TestFormatQueryCSV(t *testing.T) {
	r := &index.QueryResult{
		Columns: []string{"id", "title"},
		Rows:    [][]string{{"foo", "Foo, Bar"}},
	}
	out := index.FormatQueryCSV(r)
	lines := strings.Split(out, "\r\n")
	if lines[0] != "id,title" {
		t.Errorf("CSV header: expected 'id,title', got %q", lines[0])
	}
	if lines[1] != `foo,"Foo, Bar"` {
		t.Errorf("CSV row: expected 'foo,\"Foo, Bar\"', got %q", lines[1])
	}
}

// TestRunQuery_ReadOnlyRejectsDisguisedWrite verifies that a write which slips
// past the first-keyword SELECT check (a leading WITH) is still rejected,
// because the connection is opened read-only. This is the real enforcement of
// "queries never mutate".
func TestRunQuery_ReadOnlyRejectsDisguisedWrite(t *testing.T) {
	dbPath := buildTestDB(t)

	// Sanity: two assertions in the fixture before the attempted write.
	before, err := index.RunQuery(dbPath, "SELECT COUNT(*) FROM assertions")
	if err != nil {
		t.Fatalf("count query: %v", err)
	}
	if before.Rows[0][0] != "2" {
		t.Fatalf("fixture precondition: expected 2 assertions, got %s", before.Rows[0][0])
	}

	// A DELETE disguised behind a CTE passes the first-keyword check (WITH) but
	// must fail on the read-only connection.
	if _, err := index.RunQuery(dbPath, "WITH x AS (SELECT 1) DELETE FROM assertions"); err == nil {
		t.Fatal("expected disguised write to be rejected, got nil error")
	}

	// The table must be untouched.
	after, err := index.RunQuery(dbPath, "SELECT COUNT(*) FROM assertions")
	if err != nil {
		t.Fatalf("count query after: %v", err)
	}
	if after.Rows[0][0] != "2" {
		t.Errorf("assertions were modified by a disguised write: now %s, want 2", after.Rows[0][0])
	}
}
