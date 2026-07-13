package index

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "modernc.org/sqlite"
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

	db, err := sql.Open("sqlite", dbPath)
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

// FormatQueryTable renders a QueryResult as a space-padded table.
func FormatQueryTable(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return ""
	}

	// Build all rows: header + data.
	all := make([][]string, 0, len(r.Rows)+1)
	// Header: uppercase column names.
	hdr := make([]string, len(r.Columns))
	for i, c := range r.Columns {
		hdr[i] = strings.ToUpper(c)
	}
	all = append(all, hdr)
	all = append(all, r.Rows...)

	// Compute column widths.
	widths := make([]int, len(r.Columns))
	for _, row := range all {
		for ci, cell := range row {
			if len(cell) > widths[ci] {
				widths[ci] = len(cell)
			}
		}
	}

	var sb strings.Builder
	numCols := len(r.Columns)
	for ri, row := range all {
		if ri > 0 {
			sb.WriteByte('\n')
		}
		for ci, cell := range row {
			if ci < numCols-1 {
				sb.WriteString(fmt.Sprintf("%-*s  ", widths[ci], cell))
			} else {
				sb.WriteString(cell)
			}
		}
	}
	return sb.String()
}

// FormatQueryTSV renders a QueryResult as tab-separated values with a lowercase header.
func FormatQueryTSV(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return ""
	}
	var sb strings.Builder

	// Header (lowercase).
	for i, c := range r.Columns {
		if i > 0 {
			sb.WriteByte('\t')
		}
		sb.WriteString(strings.ToLower(c))
	}
	sb.WriteByte('\n')

	for _, row := range r.Rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteByte('\t')
			}
			sb.WriteString(cell)
		}
		sb.WriteByte('\n')
	}

	return strings.TrimRight(sb.String(), "\n")
}

// FormatQueryCSV renders a QueryResult as RFC 4180 CSV with a header row.
func FormatQueryCSV(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return ""
	}
	var sb strings.Builder

	writeCSVRow := func(cells []string, lowercase bool) {
		for i, c := range cells {
			if i > 0 {
				sb.WriteByte(',')
			}
			if lowercase {
				c = strings.ToLower(c)
			}
			sb.WriteString(csvQueryField(c))
		}
		sb.WriteString("\r\n")
	}

	writeCSVRow(r.Columns, true)
	for _, row := range r.Rows {
		writeCSVRow(row, false)
	}

	result := sb.String()
	return strings.TrimRight(result, "\r\n")
}

// FormatQueryJSON renders a QueryResult as a JSON array of objects.
func FormatQueryJSON(r *QueryResult) string {
	if r == nil || len(r.Columns) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[\n")
	for ri, row := range r.Rows {
		sb.WriteString("  {")
		for ci, col := range r.Columns {
			if ci > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q: %q", col, row[ci]))
		}
		sb.WriteString("}")
		if ri < len(r.Rows)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("]")
	return sb.String()
}

func csvQueryField(s string) string {
	if strings.ContainsAny(s, ",\"\r\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
