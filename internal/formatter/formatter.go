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
func headerRow(opts Options) Row {
	r := Row{
		ID:     "ID",
		Status: "STATUS",
		// Priority handled specially in columns()
		Title:  "TITLE",
		Parent: "PARENT",
		File:   "FILE",
	}
	return r
}

// FormatTable returns a space-padded table string. The first row is the header.
// Column widths are derived from the widest value in each column (including
// header). The last column has no trailing padding.
func FormatTable(rows []Row, opts Options) string {
	if len(rows) == 0 {
		return ""
	}

	hdr := headerRow(opts)

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
func FormatTSV(rows []Row, opts Options) string {
	if len(rows) == 0 {
		return ""
	}

	hdr := headerRow(opts)

	var sb strings.Builder

	// Header row (lowercase).
	writeRow := func(cells []string, lowercase bool) {
		for i, c := range cells {
			if i > 0 {
				sb.WriteByte('\t')
			}
			if lowercase {
				sb.WriteString(strings.ToLower(c))
			} else {
				sb.WriteString(c)
			}
		}
		sb.WriteByte('\n')
	}

	writeRow(columns(hdr, true, opts), true)
	for _, r := range rows {
		writeRow(columns(r, false, opts), false)
	}

	// Trim trailing newline to match table behavior (caller uses fmt.Println).
	result := sb.String()
	return strings.TrimRight(result, "\n")
}

// FormatCSV returns RFC 4180 CSV output with a header row. Fields containing
// commas, double-quotes, or newlines are double-quoted; embedded double-quotes
// are doubled. Each row ends with CRLF.
func FormatCSV(rows []Row, opts Options) string {
	if len(rows) == 0 {
		return ""
	}

	hdr := headerRow(opts)

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

	// Trim the trailing CRLF so fmt.Println doesn't add a blank line.
	result := sb.String()
	return strings.TrimRight(result, "\r\n")
}

// csvField wraps a field in double-quotes when necessary (RFC 4180).
func csvField(s string) string {
	if strings.ContainsAny(s, ",\"\r\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
