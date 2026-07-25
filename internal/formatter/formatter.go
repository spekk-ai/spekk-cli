// Package formatter provides human-readable and machine-readable output
// formats for spekk list: table (default), TSV, and CSV.
package formatter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Row is a single record to be formatted. Parent and File are optional columns
// controlled by the calling site.
type Row struct {
	ID       string
	Status   string
	Priority int
	Title    string
	Parent   string // set when --assertions-only
	File     string // set when --long
}

// Options controls which optional columns are included.
type Options struct {
	ShowParent bool // include PARENT column (--assertions-only)
	ShowFile   bool // include FILE column (--long)
}

// columns returns the ordered list of column values for a row plus the header.
// header must be true only for the synthetic header row.
func columns(r Row, header bool, opts Options) []string {
	pri := strconv.Itoa(r.Priority)
	if header {
		pri = "PRI"
	}
	cols := []string{r.ID, r.Status, pri}
	if opts.ShowParent {
		cols = append(cols, r.Parent)
	}
	cols = append(cols, r.Title)
	if opts.ShowFile {
		cols = append(cols, r.File)
	}
	return cols
}

// headerRow returns a Row whose fields are the column header strings.
func headerRow() Row {
	return Row{
		ID:     "ID",
		Status: "STATUS",
		// Priority handled specially in columns()
		Title:  "TITLE",
		Parent: "PARENT",
		File:   "FILE",
	}
}

// FormatTable returns a space-padded table string. The first row is the header.
// Column widths are derived from the widest value in each column (including
// header). The last column has no trailing padding.
func FormatTable(rows []Row, opts Options) string {
	if len(rows) == 0 {
		return ""
	}

	hdr := headerRow()

	// Build all rows (header + data) as string slices, then render.
	all := make([][]string, 0, len(rows)+1)
	all = append(all, columns(hdr, true, opts))
	for _, r := range rows {
		all = append(all, columns(r, false, opts))
	}

	return RenderTable(all)
}

// RenderTable renders a matrix of string cells as a space-padded table. The
// first row is the header (rendering is uniform; the header is only special
// visually). Column widths are the widest value in each column; the last column
// has no trailing padding. There is no trailing newline. Empty input yields "".
//
// This is the column-generic core shared by FormatTable (fixed spec/assertion
// columns) and spekk query (arbitrary SELECT columns).
func RenderTable(matrix [][]string) string {
	if len(matrix) == 0 {
		return ""
	}

	numCols := len(matrix[0])

	// Compute column widths.
	widths := make([]int, numCols)
	for _, row := range matrix {
		for ci, cell := range row {
			if ci < numCols && len(cell) > widths[ci] {
				widths[ci] = len(cell)
			}
		}
	}

	// Render rows.
	var sb strings.Builder
	for ri, row := range matrix {
		if ri > 0 {
			sb.WriteByte('\n')
		}
		for ci, cell := range row {
			if ci < numCols-1 {
				// Left-align, pad to column width, then add 2-space separator.
				sb.WriteString(fmt.Sprintf("%-*s  ", widths[ci], cell))
			} else {
				// Last column: no trailing padding.
				sb.WriteString(cell)
			}
		}
	}

	return sb.String()
}

// FormatTSVHeader returns the TSV header row only (lowercase, tab-separated,
// newline-terminated). Useful when the data set is empty but callers still
// expect a well-formed header.
func FormatTSVHeader(opts Options) string {
	hdr := headerRow()
	cells := columns(hdr, true, opts)
	var sb strings.Builder
	for i, c := range cells {
		if i > 0 {
			sb.WriteByte('\t')
		}
		sb.WriteString(strings.ToLower(c))
	}
	sb.WriteByte('\n')
	return sb.String()
}

// FormatCSVHeader returns the CSV header row only (lowercase, RFC 4180
// CRLF-terminated). Useful when the data set is empty but callers still
// expect a well-formed header.
func FormatCSVHeader(opts Options) string {
	hdr := headerRow()
	cells := columns(hdr, true, opts)
	var sb strings.Builder
	for i, c := range cells {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strings.ToLower(c))
	}
	sb.WriteString("\r\n")
	return sb.String()
}

// FormatTSV returns tab-separated output. The header is lowercase. No padding.
// The returned string ends with a newline (caller should use fmt.Print, not
// fmt.Println, to avoid a duplicate blank line).
func FormatTSV(rows []Row, opts Options) string {
	if len(rows) == 0 {
		return ""
	}

	hdr := headerRow()
	data := make([][]string, len(rows))
	for i, r := range rows {
		data[i] = columns(r, false, opts)
	}
	return RenderTSV(columns(hdr, true, opts), data)
}

// RenderTSV renders headers + rows as tab-separated values. The header row is
// lowercased; every cell is sanitized (TSV has no quoting, so tab/CR/LF become
// spaces to preserve column alignment). The result ends with a newline. This is
// the column-generic core shared by FormatTSV and spekk query.
func RenderTSV(headers []string, rows [][]string) string {
	var sb strings.Builder

	writeRow := func(cells []string, lowercase bool) {
		for i, c := range cells {
			if i > 0 {
				sb.WriteByte('\t')
			}
			if lowercase {
				c = strings.ToLower(c)
			}
			sb.WriteString(sanitizeTSV(c))
		}
		sb.WriteByte('\n')
	}

	writeRow(headers, true)
	for _, r := range rows {
		writeRow(r, false)
	}

	return sb.String()
}

// FormatCSV returns RFC 4180 CSV output with a header row. Fields containing
// commas, double-quotes, or newlines are double-quoted; embedded double-quotes
// are doubled. Each row (including the last) ends with CRLF per RFC 4180.
// The returned string ends with CRLF (caller should use fmt.Print, not
// fmt.Println, to avoid appending a bare LF after the final CRLF).
func FormatCSV(rows []Row, opts Options) string {
	if len(rows) == 0 {
		return ""
	}

	hdr := headerRow()
	data := make([][]string, len(rows))
	for i, r := range rows {
		data[i] = columns(r, false, opts)
	}
	return RenderCSV(columns(hdr, true, opts), data)
}

// RenderCSV renders headers + rows as RFC 4180 CSV. The header row is
// lowercased; fields containing commas, quotes, or newlines are double-quoted
// with embedded quotes doubled; every row ends with CRLF. This is the
// column-generic core shared by FormatCSV and spekk query.
func RenderCSV(headers []string, rows [][]string) string {
	var sb strings.Builder

	writeCSVRow := func(cells []string, lowercase bool) {
		for i, c := range cells {
			if i > 0 {
				sb.WriteByte(',')
			}
			if lowercase {
				c = strings.ToLower(c)
			}
			sb.WriteString(csvField(c))
		}
		sb.WriteString("\r\n")
	}

	writeCSVRow(headers, true)
	for _, r := range rows {
		writeCSVRow(r, false)
	}

	return sb.String()
}

// RenderJSON renders headers + rows as a JSON array of objects, one object per
// row, with keys in SELECT/column order. Keys and values are encoded with
// encoding/json (correct escaping); all values are strings. Empty input yields
// "[]". Shared by spekk query's --json output.
func RenderJSON(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[\n")
	for ri, row := range rows {
		sb.WriteString("  {")
		for ci, h := range headers {
			if ci > 0 {
				sb.WriteString(", ")
			}
			key, _ := json.Marshal(h)
			val := ""
			if ci < len(row) {
				val = row[ci]
			}
			value, _ := json.Marshal(val)
			sb.Write(key)
			sb.WriteString(": ")
			sb.Write(value)
		}
		sb.WriteString("}")
		if ri < len(rows)-1 {
			sb.WriteByte(',')
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("]")
	return sb.String()
}

// csvField wraps a field in double-quotes when necessary (RFC 4180).
func csvField(s string) string {
	if strings.ContainsAny(s, ",\"\r\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// sanitizeTSV replaces tab, CR, and LF characters with a single space.
// TSV has no quoting mechanism, so embedded control characters cannot be
// represented faithfully; replacing them preserves column alignment.
func sanitizeTSV(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
