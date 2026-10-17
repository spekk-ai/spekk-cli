package docs

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var testNow = time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)

const longSummary = `Vendor "Acme" manual, chapter 3: pump torque, valve limits, and wiring.`

func addFiles(t *testing.T, root string, opts AddOptions, paths ...string) (int, string) {
	t.Helper()
	var out strings.Builder
	opts.Now = testNow
	n := Add(root, paths, opts, &out)
	return n, out.String()
}

func TestStubRoundTrip(t *testing.T) {
	in := Stub{Title: `A "quoted", title`, Type: "manual", Created: "2026-10-17", Source: `C:\docs\x.pdf`, Path: "raw/x.pdf",
		Text: "formatted/x.txt", SHA256: "abc", Size: 42, Tags: []string{"pump", "field notes"}, Summary: longSummary + "\nSecond line."}
	p := filepath.Join(t.TempDir(), "x.md")
	if err := os.WriteFile(p, []byte(in.Format()), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadStub(p)
	if err != nil {
		t.Fatal(err)
	}
	in.Summary = longSummary + " Second line." // newlines become spaces
	if got.Title != in.Title || got.Source != in.Source || got.Summary != in.Summary || got.Size != 42 ||
		strings.Join(got.Tags, "|") != "pump|field notes" || got.Text != in.Text || got.Path != in.Path {
		t.Errorf("round trip:\n got %+v\nwant %+v", got, in)
	}
}

func TestAddSummaryBoundsAndReAdd(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(t.TempDir(), "Pump Manual.md")
	write(t, filepath.Dir(src), "Pump Manual.md", "# Pump Manual\n\nTorque is 40 Nm.\n")

	if n, out := addFiles(t, root, AddOptions{Summary: "too short"}, src); n == 0 || !strings.Contains(out, "it must have 40 to 600") {
		t.Fatalf("short summary was accepted: %s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "raw")); !os.IsNotExist(err) {
		t.Fatalf("a failed add wrote raw/")
	}

	n, out := addFiles(t, root, AddOptions{}, src)
	if n != 0 || !strings.Contains(out, `spekk docs add raw/Pump Manual.md --summary`) {
		t.Fatalf("add without summary: n=%d out=%s", n, out)
	}
	stubPath := filepath.Join(root, "formatted", "pump-manual.md")
	stub, err := ReadStub(stubPath)
	if err != nil || stub.Path != "raw/Pump Manual.md" || stub.Title != "Pump Manual" || stub.Source != "Pump Manual.md" || stub.Created != "2026-10-17" {
		t.Fatalf("stub = %+v, %v", stub, err)
	}

	// Unchanged content keeps the summary that a second add wrote.
	if n, out := addFiles(t, root, AddOptions{Summary: longSummary, Tags: []string{"pump"}}, filepath.Join(root, "raw", "Pump Manual.md")); n != 0 {
		t.Fatal(out)
	}
	if n, out := addFiles(t, root, AddOptions{}, filepath.Join(root, "raw", "Pump Manual.md")); n != 0 || strings.Contains(out, "no summary") {
		t.Fatalf("re-add of unchanged content: %s", out)
	}
	if stub, _ = ReadStub(stubPath); stub.Summary != longSummary || strings.Join(stub.Tags, ",") != "pump" {
		t.Fatalf("unchanged re-add lost fields: %+v", stub)
	}

	// Changed content clears the summary and keeps the tags.
	write(t, root, "raw/Pump Manual.md", "# Pump Manual\n\nTorque is 45 Nm.\n")
	if _, out := addFiles(t, root, AddOptions{}, filepath.Join(root, "raw", "Pump Manual.md")); !strings.Contains(out, "old summary was cleared") {
		t.Fatalf("changed re-add: %s", out)
	}
	if stub, _ = ReadStub(stubPath); stub.Summary != "" || strings.Join(stub.Tags, ",") != "pump" {
		t.Fatalf("changed re-add: %+v", stub)
	}

	// A different file with the same name never overwrites raw/.
	other := filepath.Join(t.TempDir(), "Pump Manual.md")
	write(t, filepath.Dir(other), "Pump Manual.md", "# Other\n")
	if n, out := addFiles(t, root, AddOptions{}, other); n != 1 || !strings.Contains(out, "already exists with other content") {
		t.Fatalf("overwrite: n=%d %s", n, out)
	}
}

func TestHTMLToText(t *testing.T) {
	got := HTMLToText(`<html><head><title>Guide</title><style>p{}</style><script>alert(1)</script></head>
<body><h1>Install</h1><p>Run &lt;make&gt; &amp; wait.</p><h2>Notes</h2><ul><li>one</li><li>two</li></ul></body></html>`)
	want := "# Install\n\nRun <make> & wait.\n\n## Notes\n\n- one\n- two\n"
	if got != want {
		t.Errorf("HTMLToText =\n%q\nwant\n%q", got, want)
	}
}

func writeZip(t *testing.T, p string, files map[string]string) {
	t.Helper()
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	f.Close()
}

func TestXLSXToText(t *testing.T) {
	p := filepath.Join(t.TempDir(), "parts.xlsx")
	writeZip(t, p, map[string]string{
		"xl/workbook.xml": `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="Parts" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>`,
		"xl/sharedStrings.xml":       `<sst><si><t>Part</t></si><si><t>Torque</t></si><si><r><t>Valve </t></r><r><t>V2</t></r></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet><sheetData>
<row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
<row r="2"><c r="A2" t="s"><v>2</v></c><c r="C2"><v>40.5</v></c></row>
<row r="3"><c r="B3" t="inlineStr"><is><t>inline</t></is></c></row>
</sheetData></worksheet>`,
	})
	got, err := Extract(p)
	want := "# Parts\n\nPart\tTorque\nValve V2\t\t40.5\n\tinline\n\n"
	if err != nil || got != want {
		t.Errorf("xlsx = %q, %v\nwant %q", got, err, want)
	}
}

func TestXLSXColumnBound(t *testing.T) {
	sheet := func(body string) map[string]string {
		return map[string]string{
			"xl/workbook.xml":          `<workbook><sheets><sheet name="S" sheetId="1"/></sheets></workbook>`,
			"xl/worksheets/sheet1.xml": `<worksheet><sheetData>` + body + `</sheetData></worksheet>`,
		}
	}
	dir := t.TempDir()
	col := filepath.Join(dir, "col.xlsx")
	writeZip(t, col, sheet(`<row r="1"><c r="ZZZZZZZZZZZZ1"><v>1</v></c></row>`))
	if _, err := Extract(col); err == nil || !strings.Contains(err.Error(), "past the last column") {
		t.Errorf("a column past XFD: err = %v", err)
	}
	last := filepath.Join(dir, "last.xlsx")
	writeZip(t, last, sheet(`<row r="1"><c r="XFD1"><v>1</v></c></row>`))
	if _, err := Extract(last); err != nil {
		t.Errorf("column XFD must be read: %v", err)
	}
}

func TestEmailToText(t *testing.T) {
	p := filepath.Join(t.TempDir(), "msg.eml")
	write(t, filepath.Dir(p), "msg.eml", strings.ReplaceAll(`From: Ana <ana@example.com>
To: ops@example.com
Date: Mon, 5 Oct 2026 10:00:00 +0000
Subject: =?UTF-8?Q?Pump_r=C3=A9sum=C3=A9?=
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="b1"

--b1
Content-Type: text/plain; charset=UTF-8
Content-Transfer-Encoding: quoted-printable

The torque limit is 40 Nm=2E
--b1
Content-Type: application/pdf; name="spec.pdf"
Content-Disposition: attachment; filename="spec.pdf"
Content-Transfer-Encoding: base64

JVBERi0=
--b1--
`, "\n", "\r\n"))
	got, err := Extract(p)
	want := "From: Ana <ana@example.com>\nTo: ops@example.com\nDate: Mon, 5 Oct 2026 10:00:00 +0000\nSubject: Pump résumé\n\nThe torque limit is 40 Nm.\n\nAttachments:\n- spec.pdf\n"
	if err != nil || got != want {
		t.Errorf("eml = %q, %v\nwant %q", got, err, want)
	}
}

func TestMissingConverterMessage(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	src := filepath.Join(root, "raw", "scan.pdf")
	write(t, root, "raw/scan.pdf", "%PDF-1.4")
	n, out := addFiles(t, root, AddOptions{}, src)
	if n != 1 || !strings.Contains(out, "pdftotext is not on PATH. Install poppler (apt: poppler-utils, brew: poppler)") {
		t.Errorf("n=%d out=%s", n, out)
	}
}

func TestSecretScan(t *testing.T) {
	text := strings.Join([]string{
		"plain line",
		"-----BEGIN OPENSSH PRIVATE KEY-----",
		"key AKIAABCDEFGHIJKLMNOP here",
		"token ghp_" + strings.Repeat("a", 36),
		"slack xoxb-1234567890-abc",
		"anthropic sk-ant-" + strings.Repeat("b", 24),
		"google AIza" + strings.Repeat("c", 35),
		"near miss AKIA1234 and ghp_short and sk-ant-x",
	}, "\n")
	hits := ScanSecrets(text)
	var got []string
	for _, h := range hits {
		got = append(got, h.Kind)
		if h.Line < 2 || h.Line > 7 {
			t.Errorf("hit on line %d: %+v", h.Line, h)
		}
	}
	if len(hits) != 6 {
		t.Errorf("hits = %v, want 6 (one for each pattern, no near miss)", got)
	}

	root := t.TempDir()
	write(t, root, "raw/creds.txt", "notes\nAKIAABCDEFGHIJKLMNOP\n")
	n, out := addFiles(t, root, AddOptions{}, filepath.Join(root, "raw", "creds.txt"))
	if n != 1 || !strings.Contains(out, "AWS access key id on line 2") || strings.Contains(out, "AKIAABCDEFGHIJKLMNOP") {
		t.Errorf("secret add: n=%d out=%s", n, out)
	}
	if _, err := os.Stat(filepath.Join(root, "formatted")); !os.IsNotExist(err) {
		t.Errorf("a refused file wrote a stub")
	}
	if n, out := addFiles(t, root, AddOptions{AllowSecrets: true}, filepath.Join(root, "raw", "creds.txt")); n != 0 {
		t.Errorf("--allow-secrets: %s", out)
	}
}

func TestPointerSizeRule(t *testing.T) {
	root := t.TempDir()
	page := "<p>" + strings.Repeat("x", 1000) + "</p>"
	html := strings.Repeat(page, PointerSize/len(page)+1)
	at := html[:PointerSize]
	write(t, root, "raw/at.html", at)
	write(t, root, "raw/over.html", at+"x")
	if n, out := addFiles(t, root, AddOptions{}, filepath.Join(root, "raw")); n != 0 {
		t.Fatal(out)
	}
	ignore, _ := os.ReadFile(filepath.Join(root, "raw", ".gitignore"))
	if !strings.Contains(string(ignore), "/over.html\n") || strings.Contains(string(ignore), "/at.html") {
		t.Errorf("raw/.gitignore = %q, want only /over.html", ignore)
	}
	if _, err := os.Stat(filepath.Join(root, "formatted", "over.txt")); err != nil {
		t.Errorf("a pointer must keep its derived text: %v", err)
	}
}
