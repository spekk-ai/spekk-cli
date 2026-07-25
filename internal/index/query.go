package index

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/spekk-ai/spekk-cli/internal/formatter"
)

// QueryResult holds the column names and rows returned by a SELECT query.
type QueryResult struct {
	Columns []string
	Rows    [][]string
}

// ErrDBNotFound is returned by RunQuery when the index database file does not exist.
var ErrDBNotFound = fmt.Errorf("index.db not found: run 'spekk index' to build the index first")

// RunQuery validates that sql is a SELECT statement, executes it against dbPath,
// and returns the result. Returns an error if the statement is not SELECT, the
// database file does not exist, or the query fails.
func RunQuery(dbPath, sqlStr string) (*QueryResult, error) {
	if !isSelectStatement(sqlStr) {
		return nil, fmt.Errorf("only SELECT statements are permitted; got: %s", firstWord(sqlStr))
	}

	// Check that the database file actually exists before opening.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, ErrDBNotFound
	}

	// Open read-only. This is the real enforcement of "queries never mutate":
	// the isSelectStatement prefix check is only a fast, friendly early error and
	// is bypassable (e.g. "WITH x AS (...) DELETE ..."), but a read-only
	// connection rejects any write regardless of how the SQL is phrased.
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("cannot open index.db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("cannot read columns: %w", err)
	}

	var result QueryResult
	result.Columns = cols

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			if v == nil {
				row[i] = ""
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &result, nil
}

// isSelectStatement reports whether the SQL string is a SELECT statement.
// It trims leading whitespace and comments, then checks if the first keyword is SELECT.
func isSelectStatement(sqlStr string) bool {
	word := strings.ToUpper(firstWord(sqlStr))
	return word == "SELECT" || word == "WITH" // CTEs start with WITH
}

// firstWord returns the first whitespace-delimited token from s.
func firstWord(s string) string {
	s = strings.TrimSpace(s)
	end := strings.IndexAny(s, " \t\n\r(")
	if end == -1 {
		return s
	}
	return s[:end]
}

// The FormatQuery* functions render a QueryResult through the shared
// internal/formatter renderers, so query and `spekk list` share one definition
// of each output format (correct RFC 4180 CSV, sanitized TSV, valid JSON). Query
// uses arbitrary SELECT columns, so it calls the column-generic Render* entry
// points rather than the Row-typed ones.

// FormatQueryTable renders a QueryResult as a space-padded table with an
// uppercased header row.
func FormatQueryTable(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return ""
	}
	matrix := make([][]string, 0, len(r.Rows)+1)
	hdr := make([]string, len(r.Columns))
	for i, c := range r.Columns {
		hdr[i] = strings.ToUpper(c)
	}
	matrix = append(matrix, hdr)
	matrix = append(matrix, r.Rows...)
	return formatter.RenderTable(matrix)
}

// FormatQueryTSV renders a QueryResult as tab-separated values with a lowercase
// header. Cells are sanitized (no tab/CR/LF corruption).
func FormatQueryTSV(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return ""
	}
	return strings.TrimRight(formatter.RenderTSV(r.Columns, r.Rows), "\n")
}

// FormatQueryCSV renders a QueryResult as RFC 4180 CSV with a lowercase header.
func FormatQueryCSV(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return ""
	}
	return strings.TrimRight(formatter.RenderCSV(r.Columns, r.Rows), "\r\n")
}

// FormatQueryJSON renders a QueryResult as a JSON array of objects, keys in
// column order, via encoding/json.
func FormatQueryJSON(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return "[]"
	}
	return formatter.RenderJSON(r.Columns, r.Rows)
}
