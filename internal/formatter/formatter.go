// Package formatter provides human-readable and machine-readable output
// formats for spekk list: table (default), TSV, and CSV.
package formatter

import (
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

	// Build all rows (header + data) as string slices.
	all := make([][]string, 0, len(rows)+1)
	all = append(all, columns(hdr, true, opts))
	for _, r := range rows {
		all = append(all, columns(r, false, opts))
	}

	numCols := len(all[0])

	// Compute column widths.
	widths := make([]int, numCols)
	for _, row := range all {
		for ci, cell := range row {
			if len(cell) > widths[ci] {
				widths[ci] = len(cell)
			}
		}
	}

	// Render rows.
	var sb strings.Builder
	for ri, row := range all {
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

// FormatTSV returns tab-separated output. The header is lowercase. No padding.
// The returned string ends with a newline (caller should use fmt.Print, not
// fmt.Println, to avoid a duplicate blank line).
func FormatTSV(rows []Row, opts Options) string {
	if len(rows) == 0 {
		return ""
	}

	hdr := headerRow()

	var sb strings.Builder

	writeRow := func(cells []string, lowercase bool) {
		for i, c := range cells {
			if i > 0 {
				sb.WriteByte('\t')
			}
			if lowercase {
				c = strings.ToLower(c)
			}
			// TSV has no quoting mechanism; replace control chars with spaces.
			sb.WriteString(sanitizeTSV(c))
		}
		sb.WriteByte('\n')
	}

	writeRow(columns(hdr, true, opts), true)
	for _, r := range rows {
		writeRow(columns(r, false, opts), false)
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

	writeCSVRow(columns(hdr, true, opts), true)
	for _, r := range rows {
		writeCSVRow(columns(r, false, opts), false)
	}

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
