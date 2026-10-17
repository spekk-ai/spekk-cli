package docs

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// converter is an external tool that turns one format into text. The binary
// converts the formats that the Go standard library (or x/net/html) reads,
// and calls a tool on PATH only where no small pure-Go reader exists.
type converter struct {
	tool string
	args func(file string) []string
	pkg  string
	apt  string
	brew string
}

var converters = map[string]converter{
	".pdf": {
		tool: "pdftotext",
		args: func(f string) []string { return []string{"-enc", "UTF-8", f, "-"} },
		pkg:  "poppler", apt: "poppler-utils", brew: "poppler",
	},
	".docx": {
		tool: "pandoc",
		args: func(f string) []string { return []string{"--to", "gfm", "--wrap", "none", f} },
		pkg:  "pandoc", apt: "pandoc", brew: "pandoc",
	},
}

// plainExts are the formats indexed from the raw file with no conversion.
var plainExts = map[string]bool{".md": true, ".markdown": true, ".txt": true}

// builtinExts are converted in the binary.
var builtinExts = map[string]bool{".html": true, ".htm": true, ".xlsx": true, ".eml": true}

// Supported reports whether `docs add` reads a file of this format.
func Supported(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	_, ext2 := converters[ext]
	return plainExts[ext] || builtinExts[ext] || ext2
}

// IsPlain reports whether a format is indexed from the raw file itself.
func IsPlain(p string) bool {
	return plainExts[strings.ToLower(path.Ext(p))]
}

// ErrNoText marks a conversion that gave no text, such as a scanned PDF.
var ErrNoText = errors.New("the converted text is empty (a scanned document needs OCR, which spekk does not do)")

// Extract returns the plaintext of a file. For a plain format it is the file
// itself.
func Extract(file string) (string, error) {
	ext := strings.ToLower(filepath.Ext(file))
	var text string
	var err error
	switch {
	case plainExts[ext]:
		var raw []byte
		raw, err = os.ReadFile(file)
		text = string(raw)
	case ext == ".html" || ext == ".htm":
		var raw []byte
		if raw, err = os.ReadFile(file); err == nil {
			text = HTMLToText(string(raw))
		}
	case ext == ".xlsx":
		text, err = xlsxToText(file)
	case ext == ".eml":
		text, err = emlToText(file)
	default:
		c, ok := converters[ext]
		if !ok {
			return "", fmt.Errorf("unsupported format %q", ext)
		}
		text, err = c.run(file)
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" && !plainExts[ext] {
		return "", ErrNoText
	}
	return text, nil
}

func (c converter) run(file string) (string, error) {
	if _, err := exec.LookPath(c.tool); err != nil {
		return "", fmt.Errorf("%s is not on PATH. Install %s (apt: %s, brew: %s), or convert the file to text and add the text file", c.tool, c.pkg, c.apt, c.brew)
	}
	var stderr bytes.Buffer
	cmd := exec.Command(c.tool, c.args(file)...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s failed: %v: %s", c.tool, err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// blockTags end a line of text.
var blockTags = map[string]bool{
	"p": true, "div": true, "br": true, "tr": true, "table": true,
	"ul": true, "ol": true, "dl": true, "dt": true, "dd": true, "pre": true,
	"blockquote": true, "section": true, "article": true, "header": true,
	"footer": true, "nav": true, "aside": true, "main": true, "hr": true,
	"figure": true, "figcaption": true, "form": true, "address": true,
}

// skipTags hold no document text.
var skipTags = map[string]bool{"script": true, "style": true, "noscript": true, "template": true, "svg": true}

var manyNewlines = regexp.MustCompile(`\n{3,}`)

// HTMLToText converts HTML to plaintext. Headings become `#` lines so that
// passages follow the sections, block elements end a line, script and style
// are dropped, and entities are decoded.
func HTMLToText(src string) string {
	z := html.NewTokenizer(strings.NewReader(src))
	var b strings.Builder
	skip, pre := 0, 0
	var title string
	inTitle := false
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			out := b.String()
			lines := strings.Split(out, "\n")
			for i, l := range lines {
				lines[i] = strings.TrimRight(l, " \t")
			}
			out = manyNewlines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
			out = strings.TrimSpace(out)
			if title != "" && !strings.HasPrefix(out, "# ") {
				out = "# " + title + "\n\n" + out
			}
			return out + "\n"
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			name, _ := z.TagName()
			tag := string(name)
			start := tt != html.EndTagToken
			switch {
			case skipTags[tag]:
				if tt == html.StartTagToken {
					skip++
				} else if tt == html.EndTagToken && skip > 0 {
					skip--
				}
			case tag == "title":
				inTitle = tt == html.StartTagToken
			case len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6':
				if start {
					b.WriteString("\n\n" + strings.Repeat("#", int(tag[1]-'0')) + " ")
				} else {
					b.WriteString("\n\n")
				}
			case tag == "li":
				if start {
					b.WriteString("\n- ")
				}
			case tag == "td" || tag == "th":
				if start {
					b.WriteString(" | ")
				}
			case blockTags[tag]:
				if tag == "pre" {
					if start {
						pre++
					} else if pre > 0 {
						pre--
					}
				}
				b.WriteString("\n")
				if tag == "p" || tag == "pre" || tag == "blockquote" || tag == "table" {
					b.WriteString("\n")
				}
			}
		case html.TextToken:
			if skip > 0 {
				continue
			}
			t := string(z.Text())
			if inTitle {
				title = strings.Join(strings.Fields(t), " ")
				continue
			}
			if pre > 0 {
				b.WriteString(t)
				continue
			}
			if f := strings.Join(strings.Fields(t), " "); f != "" {
				if strings.HasPrefix(t, " ") || strings.HasPrefix(t, "\n") {
					f = " " + f
				}
				if strings.HasSuffix(t, " ") || strings.HasSuffix(t, "\n") {
					f += " "
				}
				b.WriteString(f)
			}
		}
	}
}

// XLSX limits. A spreadsheet is someone else's file: a cell reference can
// name any column.
const (
	maxColumns = 16384 // column XFD, the last one Excel has
)

// xlsxToText writes one `# <sheet>` heading for each sheet, then one line for
// each row with the cell values joined by a tab.
func xlsxToText(file string) (string, error) {
	zr, err := zip.OpenReader(file)
	if err != nil {
		return "", fmt.Errorf("not a readable XLSX file: %w", err)
	}
	defer zr.Close()
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	errMissing := errors.New("missing")
	readXML := func(name string, v any) error {
		f, ok := files[name]
		if !ok {
			return fmt.Errorf("%s is %w", name, errMissing)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return xml.NewDecoder(rc).Decode(v)
	}

	var wb struct {
		Sheets []struct {
			Name string `xml:"name,attr"`
			RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"sheets>sheet"`
	}
	if err := readXML("xl/workbook.xml", &wb); err != nil {
		return "", fmt.Errorf("not a readable XLSX file: %w", err)
	}
	var rels struct {
		Rel []struct {
			ID     string `xml:"Id,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	// A workbook may have no relationships part or no shared strings; a part
	// that is there and does not parse is an error.
	if err := readXML("xl/_rels/workbook.xml.rels", &rels); err != nil && !errors.Is(err, errMissing) {
		return "", fmt.Errorf("not a readable XLSX file: %w", err)
	}
	targets := map[string]string{}
	for _, r := range rels.Rel {
		t := strings.TrimPrefix(r.Target, "/")
		if !strings.HasPrefix(t, "xl/") {
			t = "xl/" + t
		}
		targets[r.ID] = t
	}
	type richText struct {
		T string `xml:"t"`
		R []struct {
			T string `xml:"t"`
		} `xml:"r"`
	}
	text := func(r richText) string {
		s := r.T
		for _, run := range r.R {
			s += run.T
		}
		return s
	}
	var sst struct {
		SI []richText `xml:"si"`
	}
	if err := readXML("xl/sharedStrings.xml", &sst); err != nil && !errors.Is(err, errMissing) {
		return "", fmt.Errorf("not a readable XLSX file: %w", err)
	}

	var b strings.Builder
	for i, s := range wb.Sheets {
		target := targets[s.RID]
		if target == "" {
			target = fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1)
		}
		var ws struct {
			Rows []struct {
				Cells []struct {
					Ref    string   `xml:"r,attr"`
					Type   string   `xml:"t,attr"`
					V      string   `xml:"v"`
					Inline richText `xml:"is"`
				} `xml:"c"`
			} `xml:"sheetData>row"`
		}
		if err := readXML(target, &ws); err != nil {
			return "", fmt.Errorf("cannot read sheet %q: %w", s.Name, err)
		}
		fmt.Fprintf(&b, "# %s\n\n", s.Name)
		for _, row := range ws.Rows {
			var cells []string
			for _, c := range row.Cells {
				v := c.V
				switch c.Type {
				case "s":
					if n, err := strconv.Atoi(c.V); err == nil && n >= 0 && n < len(sst.SI) {
						v = text(sst.SI[n])
					}
				case "inlineStr":
					v = text(c.Inline)
				}
				// A sparse row skips empty cells; the cell reference keeps
				// each value in its own column.
				col := columnIndex(c.Ref)
				if col >= maxColumns {
					return "", fmt.Errorf("sheet %q: cell reference %q is past the last column (XFD)", s.Name, c.Ref)
				}
				if col > len(cells) {
					cells = append(cells, make([]string, col-len(cells))...)
				}
				cells = append(cells, strings.Join(strings.Fields(v), " "))
			}
			b.WriteString(strings.TrimRight(strings.Join(cells, "\t"), "\t") + "\n")
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

// columnIndex returns the 0-based column of a cell reference such as "C7".
// A reference with more letters than any real column returns maxColumns, so
// the sum cannot overflow.
func columnIndex(ref string) int {
	n := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			break
		}
		n = n*26 + int(r-'A'+1)
		if n > maxColumns {
			return maxColumns
		}
	}
	return n - 1
}

// emlToText writes the From, To, Date, and Subject headers, then the
// text/plain part (or the text/html part as text), then the names of the
// attachments.
func emlToText(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()
	msg, err := mail.ReadMessage(f)
	if err != nil {
		return "", fmt.Errorf("not a readable email: %w", err)
	}
	dec := new(mime.WordDecoder)
	var b strings.Builder
	for _, h := range []string{"From", "To", "Date", "Subject"} {
		v := msg.Header.Get(h)
		if d, err := dec.DecodeHeader(v); err == nil {
			v = d
		}
		if v != "" {
			fmt.Fprintf(&b, "%s: %s\n", h, v)
		}
	}
	b.WriteString("\n")

	var plain, htmlPart string
	var attachments []string
	var walk func(h map[string][]string, body io.Reader) error
	walk = func(h map[string][]string, body io.Reader) error {
		header := mail.Header(h)
		ct := header.Get("Content-Type")
		if ct == "" {
			ct = "text/plain"
		}
		mt, params, _ := mime.ParseMediaType(ct)
		if strings.HasPrefix(mt, "multipart/") {
			mr := multipart.NewReader(body, params["boundary"])
			for {
				p, err := mr.NextRawPart()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}
				if err := walk(p.Header, p); err != nil {
					return err
				}
			}
		}
		disp, dparams, _ := mime.ParseMediaType(header.Get("Content-Disposition"))
		name := dparams["filename"]
		if name == "" {
			name = params["name"]
		}
		if disp == "attachment" || (name != "" && !strings.HasPrefix(mt, "text/")) {
			if name == "" {
				name = "(unnamed " + mt + ")"
			}
			attachments = append(attachments, name)
			return nil
		}
		var r io.Reader = body
		switch strings.ToLower(header.Get("Content-Transfer-Encoding")) {
		case "quoted-printable":
			r = quotedprintable.NewReader(body)
		case "base64":
			r = base64.NewDecoder(base64.StdEncoding, body)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		switch {
		case mt == "text/plain" && plain == "":
			plain = string(data)
		case mt == "text/html" && htmlPart == "":
			htmlPart = string(data)
		}
		return nil
	}
	if err := walk(msg.Header, msg.Body); err != nil {
		return "", fmt.Errorf("cannot read the email body: %w", err)
	}
	switch {
	case plain != "":
		b.WriteString(strings.ReplaceAll(plain, "\r\n", "\n"))
	case htmlPart != "":
		b.WriteString(HTMLToText(htmlPart))
	}
	if len(attachments) > 0 {
		b.WriteString("\n\nAttachments:\n")
		for _, a := range attachments {
			b.WriteString("- " + a + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}
