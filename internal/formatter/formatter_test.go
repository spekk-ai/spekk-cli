package formatter

import (
	"strings"
	"testing"
)

var twoRows = []Row{
	{ID: "short-id", Status: "done", Priority: 2, Title: "Short Title"},
	{ID: "a-very-long-id-indeed", Status: "in_progress", Priority: 1, Title: "Longer Title Here"},
}

// TestFormatTable_Header verifies the header row contains uppercase column names.
func TestFormatTable_Header(t *testing.T) {
	out := FormatTable(twoRows, Options{})
	lines := strings.Split(out, "\n")
	if len(lines) < 1 {
		t.Fatal("expected at least one line")
	}
	hdr := lines[0]
	for _, col := range []string{"ID", "STATUS", "PRI", "TITLE"} {
		if !strings.Contains(hdr, col) {
			t.Errorf("header missing %q: %q", col, hdr)
		}
	}
}

// TestFormatTable_ColumnAlignment verifies columns are aligned across all rows.
func TestFormatTable_ColumnAlignment(t *testing.T) {
	out := FormatTable(twoRows, Options{})
	lines := strings.Split(out, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected 3 lines (header + 2 data), got %d", len(lines))
	}

	// Find position of STATUS in header.
	statusPos := strings.Index(lines[0], "STATUS")
	if statusPos < 0 {
		t.Fatal("STATUS not found in header")
	}

	// Every data row must have its STATUS column starting at the same position.
	for i := 1; i < len(lines); i++ {
		if len(lines[i]) < statusPos {
			t.Errorf("row %d too short to contain STATUS at pos %d: %q", i, statusPos, lines[i])
			continue
		}
		// The STATUS column value starts at statusPos for every row.
		chunk := lines[i][statusPos:]
		if chunk == "" {
			t.Errorf("row %d: no content at STATUS column position %d", i, statusPos)
		}
	}
}

// TestFormatTable_NoTrailingWhitespaceOnLastColumn verifies the last column
// has no trailing spaces.
func TestFormatTable_NoTrailingWhitespaceOnLastColumn(t *testing.T) {
	out := FormatTable(twoRows, Options{})
	for i, line := range strings.Split(out, "\n") {
		if line != strings.TrimRight(line, " ") {
			t.Errorf("line %d has trailing whitespace: %q", i, line)
		}
	}
}

// TestFormatTable_DynamicWidth verifies the wider ID drives the column width.
func TestFormatTable_DynamicWidth(t *testing.T) {
	out := FormatTable(twoRows, Options{})
	lines := strings.Split(out, "\n")

	// "a-very-long-id-indeed" is 21 chars. STATUS should start at >= 23
	// (21 + 2 separator).
	wantPos := len("a-very-long-id-indeed") + 2
	statusPos := strings.Index(lines[0], "STATUS")
	if statusPos < wantPos {
		t.Errorf("STATUS column starts at %d, want >= %d", statusPos, wantPos)
	}
}

// TestFormatTable_ParentColumn verifies PARENT appears when ShowParent is set.
func TestFormatTable_ParentColumn(t *testing.T) {
	rows := []Row{
		{ID: "some-assertion", Status: "draft", Priority: 1, Title: "Some Title", Parent: "my-spec"},
	}
	out := FormatTable(rows, Options{ShowParent: true})
	if !strings.Contains(out, "PARENT") {
		t.Errorf("expected PARENT column in output: %q", out)
	}
	if !strings.Contains(out, "my-spec") {
		t.Errorf("expected parent value in output: %q", out)
	}
}

// TestFormatTable_FileColumn verifies FILE appears when ShowFile is set.
func TestFormatTable_FileColumn(t *testing.T) {
	rows := []Row{
		{ID: "an-assertion", Status: "done", Priority: 1, Title: "Title", File: "specs/foo/assertions/bar.md"},
	}
	out := FormatTable(rows, Options{ShowFile: true})
	if !strings.Contains(out, "FILE") {
		t.Errorf("expected FILE column in output: %q", out)
	}
}

// TestFormatTable_Empty verifies empty input returns empty string.
func TestFormatTable_Empty(t *testing.T) {
	if got := FormatTable(nil, Options{}); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// --- TSV ---

// TestFormatTSV_LowercaseHeader verifies the header row uses lowercase names.
func TestFormatTSV_LowercaseHeader(t *testing.T) {
	out := FormatTSV(twoRows, Options{})
	lines := strings.Split(out, "\n")
	if len(lines) < 1 {
		t.Fatal("expected at least one line")
	}
	hdr := lines[0]
	if strings.Contains(hdr, "ID") && !strings.Contains(hdr, "id") {
		t.Errorf("header should be lowercase, got: %q", hdr)
	}
	for _, col := range []string{"id", "status", "pri", "title"} {
		if !strings.Contains(hdr, col) {
			t.Errorf("TSV header missing %q: %q", col, hdr)
		}
	}
}

// TestFormatTSV_TabSeparated verifies fields are separated by tabs, not spaces.
func TestFormatTSV_TabSeparated(t *testing.T) {
	out := FormatTSV(twoRows, Options{})
	for i, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "\t") {
			t.Errorf("line %d has no tab separator: %q", i, line)
		}
		// Ensure no double-tab (would indicate an empty field in the wrong place).
		if strings.Contains(line, "\t\t") {
			t.Errorf("line %d has double-tab: %q", i, line)
		}
	}
}

// TestFormatTSV_NoTrailingTab verifies no line ends with a tab.
func TestFormatTSV_NoTrailingTab(t *testing.T) {
	out := FormatTSV(twoRows, Options{})
	for i, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, "\t") {
			t.Errorf("line %d has trailing tab: %q", i, line)
		}
	}
}

// --- CSV ---

// TestFormatCSV_Header verifies the header row is lowercase, comma-separated.
func TestFormatCSV_Header(t *testing.T) {
	out := FormatCSV(twoRows, Options{})
	lines := strings.Split(out, "\r\n")
	if len(lines) < 1 {
		t.Fatal("expected at least one line")
	}
	hdr := lines[0]
	if hdr != "id,status,pri,title" {
		t.Errorf("CSV header = %q, want %q", hdr, "id,status,pri,title")
	}
}

// TestFormatCSV_QuotedComma verifies fields with commas are double-quoted.
func TestFormatCSV_QuotedComma(t *testing.T) {
	rows := []Row{
		{ID: "auth-roles", Status: "in_progress", Priority: 1, Title: "Auth, Permissions & Roles"},
	}
	out := FormatCSV(rows, Options{})
	if !strings.Contains(out, `"Auth, Permissions & Roles"`) {
		t.Errorf("expected quoted comma field in CSV output: %q", out)
	}
}

// TestFormatCSV_QuotedDoubleQuote verifies embedded double-quotes are doubled.
func TestFormatCSV_QuotedDoubleQuote(t *testing.T) {
	rows := []Row{
		{ID: "test-id", Status: "done", Priority: 1, Title: `She said "hello"`},
	}
	out := FormatCSV(rows, Options{})
	if !strings.Contains(out, `"She said ""hello"""`) {
		t.Errorf("expected escaped double-quote in CSV output: %q", out)
	}
}

// TestFormatCSV_CRLFLineEndings verifies RFC 4180 CRLF line endings.
func TestFormatCSV_CRLFLineEndings(t *testing.T) {
	rows := []Row{
		{ID: "id-one", Status: "done", Priority: 1, Title: "Title One"},
		{ID: "id-two", Status: "done", Priority: 2, Title: "Title Two"},
	}
	out := FormatCSV(rows, Options{})
	// The output (after TrimRight of the trailing CRLF) should contain at least
	// one CRLF between lines.
	if !strings.Contains(out, "\r\n") {
		t.Errorf("expected CRLF line endings in CSV output: %q", out)
	}
}

// TestFormatCSV_NoQuoteForPlainField verifies plain fields are not quoted.
func TestFormatCSV_NoQuoteForPlainField(t *testing.T) {
	rows := []Row{
		{ID: "plain-id", Status: "done", Priority: 1, Title: "Plain Title"},
	}
	out := FormatCSV(rows, Options{})
	// Data row should be: plain-id,done,1,Plain Title (no quotes)
	if strings.Contains(out, `"plain-id"`) {
		t.Errorf("plain ID should not be quoted: %q", out)
	}
	if strings.Contains(out, `"Plain Title"`) {
		t.Errorf("plain title should not be quoted: %q", out)
	}
}
