package docs

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// Kinds of document. spec and assertion come from specs/ and are written by
// internal/index; this package finds doc and raw.
const (
	KindSpec      = "spec"
	KindAssertion = "assertion"
	KindDoc       = "doc"
	KindRaw       = "raw"
)

// Kinds lists every kind in a stable order.
var Kinds = []string{KindSpec, KindAssertion, KindDoc, KindRaw}

// States of a raw document. Every other kind is always StateOK.
const (
	StateOK         = "ok"
	StateStale      = "stale"
	StateNoSummary  = "no_summary"
	StateMissingRaw = "missing_raw"
	StatePointer    = "pointer"
	StateNoStub     = "no_stub"
)

// The fixed directories of the source corpus, relative to the repository root.
const (
	RawDir       = "raw"
	FormattedDir = "formatted"
	pathsFile    = ".spekk/docs-paths"
)

// Document is one indexed document of kind doc or raw. Path is the file the
// document is (for raw, the file in raw/); TextPath is the plaintext file the
// passages and line numbers come from. Both are relative to the repository
// root, with forward slashes.
type Document struct {
	Kind     string
	ID       string
	Path     string
	TextPath string
	StubPath string
	Title    string
	Type     string
	Created  string
	Summary  string
	Tags     []string
	SHA256   string
	State    string
}

// Corpus is what Scan found.
type Corpus struct {
	Docs []Document
	// NoStub lists files in raw/ of a supported format that no stub points
	// at. They are not indexed: a document enters the corpus through
	// `spekk docs add`, never by being dropped into a folder.
	NoStub []string
	// Warnings name documents that were skipped, such as a duplicate id.
	Warnings []string
}

// defaultIncludes are the in-place globs when .spekk/docs-paths is absent.
var defaultIncludes = []string{"docs/**/*.md", "docs/**/*.txt", "*.md"}

// neverInPlace are the top-level directories that never hold an in-place
// document: specs are indexed as specs, raw/ and formatted/ through stubs.
var neverInPlace = []string{"specs", RawDir, FormattedDir, ".spekk", ".git"}

// inPlaceExts are the formats an in-place document may have.
var inPlaceExts = map[string]bool{".md": true, ".markdown": true, ".txt": true}

// Scan finds every doc and raw document under root.
func Scan(root string) (*Corpus, error) {
	c := &Corpus{}
	seen := map[string]string{}
	add := func(d Document) {
		key := d.Kind + "\x00" + d.ID
		if prev, ok := seen[key]; ok {
			c.Warnings = append(c.Warnings, fmt.Sprintf("%s and %s have the same document id %q; only %s is indexed", prev, d.Path, d.ID, prev))
			return
		}
		seen[key] = d.Path
		c.Docs = append(c.Docs, d)
	}

	inPlace, err := inPlaceFiles(root)
	if err != nil {
		return nil, err
	}
	for _, rel := range inPlace {
		d, err := readInPlace(root, rel)
		if err != nil {
			return nil, err
		}
		add(d)
	}

	raws, noStub, err := scanRaw(root)
	if err != nil {
		return nil, err
	}
	for _, d := range raws {
		add(d)
	}
	c.NoStub = noStub
	return c, nil
}

// ID returns the document id for a path: kebab-case of the path without its
// extension. Letters are lowercased, and each run of other characters becomes
// one hyphen.
func ID(rel string) string {
	rel = strings.TrimSuffix(rel, path.Ext(rel))
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(rel) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
			continue
		}
		dash = true
	}
	return b.String()
}

// readInPlace reads one in-place document. Frontmatter is optional.
func readInPlace(root, rel string) (Document, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return Document{}, fmt.Errorf("cannot read %s: %w", rel, err)
	}
	text := string(raw)
	d := Document{Kind: KindDoc, ID: ID(rel), Path: rel, TextPath: rel, State: StateOK}
	if fields, err := parser.CustomFields(text, nil); err == nil {
		d.Title = scalar(fields["title"])
		d.Summary = scalar(fields["summary"])
		d.Type = scalar(fields["type"])
		d.Created = scalar(fields["created"])
		d.Tags = fields["tags"]
	}
	ps := Chunk(text)
	if d.Title == "" {
		for _, p := range ps {
			if p.Heading != "" {
				d.Title = strings.SplitN(p.Heading, " / ", 2)[0]
				break
			}
		}
	}
	if d.Title == "" {
		d.Title = path.Base(rel)
	}
	if d.Summary == "" {
		d.Summary = firstParagraph(text)
	}
	return d, nil
}

// firstParagraph returns the first paragraph of body text (not a heading, not
// frontmatter, not a code block), cut to 300 characters.
func firstParagraph(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var para []string
	fence := false
	for i := frontmatterEnd(lines); i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if fenceMarker(t) != "" {
			fence = !fence
			continue
		}
		_, atx := atxHeading(t)
		setext := i+1 < len(lines) && t != "" && isSetextUnderline(lines[i+1])
		skip := fence || atx || setext || isSetextUnderline(t) || strings.HasPrefix(t, "|") || strings.HasPrefix(t, "<")
		if t == "" || skip {
			if len(para) > 0 {
				break
			}
			if setext {
				i++
			}
			continue
		}
		para = append(para, t)
	}
	return Truncate(strings.Join(para, " "), 300)
}

// Truncate cuts s to at most n runes, with an ellipsis when it cut.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

// scalar joins the items of a custom field back into one string. The parser
// splits a bare scalar on commas, which is right for a tag list and wrong for
// a sentence, so a summary written without quotes comes back whole. A
// double-quoted value written by WriteStub has its two escapes undone.
func scalar(items []string) string {
	s := strings.Join(items, ", ")
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// scanRaw reads every stub in formatted/ and checks it against raw/.
func scanRaw(root string) ([]Document, []string, error) {
	entries, err := os.ReadDir(filepath.Join(root, FormattedDir))
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("cannot read %s/: %w", FormattedDir, err)
	}
	ignored := rawIgnored(root)
	referenced := map[string]bool{}
	var docs []Document
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".md" {
			continue
		}
		stubRel := FormattedDir + "/" + e.Name()
		stub, err := ReadStub(filepath.Join(root, FormattedDir, e.Name()))
		if err != nil {
			return nil, nil, fmt.Errorf("cannot read stub %s: %w", stubRel, err)
		}
		d := Document{
			Kind:     KindRaw,
			ID:       strings.TrimSuffix(e.Name(), ".md"),
			Path:     stub.Path,
			TextPath: stub.Text,
			StubPath: stubRel,
			Title:    stub.Title,
			Type:     stub.Type,
			Created:  stub.Created,
			Summary:  stub.Summary,
			Tags:     stub.Tags,
			SHA256:   stub.SHA256,
		}
		if d.TextPath == "" {
			d.TextPath = d.Path
		}
		referenced[d.Path] = true
		d.State = rawState(root, d, ignored[d.Path])
		docs = append(docs, d)
	}

	var noStub []string
	walkErr := filepath.WalkDir(filepath.Join(root, RawDir), func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if de.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if !referenced[rel] && Supported(rel) {
			noStub = append(noStub, rel)
		}
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return nil, nil, fmt.Errorf("cannot walk %s/: %w", RawDir, walkErr)
	}
	sort.Strings(noStub)
	return docs, noStub, nil
}

// rawState computes the state of a raw document. stale wins over no_summary:
// a summary of the old content is worse than none.
func rawState(root string, d Document, ignored bool) string {
	sum, err := FileSHA256(filepath.Join(root, filepath.FromSlash(d.Path)))
	switch {
	case err != nil && ignored:
		return StatePointer
	case err != nil:
		return StateMissingRaw
	case sum != d.SHA256:
		return StateStale
	case strings.TrimSpace(d.Summary) == "":
		return StateNoSummary
	}
	return StateOK
}

// FileSHA256 returns the hex sha256 of a file.
func FileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// rawIgnored reads raw/.gitignore: the pointer entries that `docs add`
// wrote, one repository-relative path per line ("/" + path under raw/).
func rawIgnored(root string) map[string]bool {
	out := map[string]bool{}
	f, err := os.Open(filepath.Join(root, RawDir, ".gitignore"))
	if err != nil {
		return out
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[RawDir+"/"+strings.TrimPrefix(line, "/")] = true
	}
	return out
}

// Globs returns the in-place include and exclude globs: the lines of
// .spekk/docs-paths, or the defaults when the file is absent.
func Globs(root string) (include, exclude []string, err error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(pathsFile)))
	if os.IsNotExist(err) {
		return defaultIncludes, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read %s: %w", pathsFile, err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "!") {
			exclude = append(exclude, strings.TrimPrefix(line, "!"))
			continue
		}
		include = append(include, line)
	}
	return include, exclude, nil
}

// globRegexp compiles a glob: `**` matches zero or more directories, `*` and
// `?` match inside one path segment.
func globRegexp(glob string) (*regexp.Regexp, error) {
	glob = strings.TrimPrefix(glob, "/")
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		switch {
		case strings.HasPrefix(glob[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(glob[i:], "**"):
			b.WriteString(".*")
			i++
		case glob[i] == '*':
			b.WriteString("[^/]*")
		case glob[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(glob[i])))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// globBase is the directory an in-place glob starts in, and how deep: Depth
// is the number of path segments below Dir a match can have, or -1 for any.
type globBase struct {
	Dir   string
	Depth int
}

// globRoot returns the fixed directory a glob starts in and its depth.
func globRoot(glob string) globBase {
	segs := strings.Split(strings.TrimPrefix(glob, "/"), "/")
	i := 0
	for i < len(segs)-1 && !strings.ContainsAny(segs[i], "*?[") {
		i++
	}
	r := globBase{Dir: strings.Join(segs[:i], "/"), Depth: len(segs) - i}
	for _, s := range segs[i:] {
		if strings.Contains(s, "**") {
			r.Depth = -1
		}
	}
	return r
}

// inPlaceFiles returns every file the globs select, in path order.
func inPlaceFiles(root string) ([]string, error) {
	include, exclude, err := Globs(root)
	if err != nil {
		return nil, err
	}
	match := func(globs []string) ([]*regexp.Regexp, error) {
		var out []*regexp.Regexp
		for _, g := range globs {
			re, err := globRegexp(g)
			if err != nil {
				return nil, fmt.Errorf("%s: bad glob %q: %w", pathsFile, g, err)
			}
			out = append(out, re)
		}
		return out, nil
	}
	inc, err := match(include)
	if err != nil {
		return nil, err
	}
	exc, err := match(exclude)
	if err != nil {
		return nil, err
	}

	found := map[string]bool{}
	for _, g := range include {
		wr := globRoot(g)
		err := walkRoot(root, wr, func(rel string, isDir bool) {
			if isDir || found[rel] || !inPlaceExts[strings.ToLower(path.Ext(rel))] {
				return
			}
			if anyMatch(inc, rel) && !anyMatch(exc, rel) {
				found[rel] = true
			}
		})
		if err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(found))
	for rel := range found {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
}

func anyMatch(res []*regexp.Regexp, s string) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// walkRoot calls fn for each file and directory under the base of one glob,
// to its depth, and skips the directories that never hold in-place documents.
func walkRoot(root string, wr globBase, fn func(rel string, isDir bool)) error {
	base := filepath.Join(root, filepath.FromSlash(wr.Dir))
	err := filepath.WalkDir(base, func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && p == base {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}
		if de.IsDir() && rel != "" {
			top := strings.SplitN(rel, "/", 2)[0]
			for _, n := range neverInPlace {
				if top == n {
					return filepath.SkipDir
				}
			}
			if de.Name() == "node_modules" || de.Name() == "vendor" {
				return filepath.SkipDir
			}
			if wr.Depth >= 0 {
				relToBase, _ := filepath.Rel(base, p)
				if strings.Count(filepath.ToSlash(relToBase), "/")+1 >= wr.Depth {
					fn(rel, true)
					return filepath.SkipDir
				}
			}
		}
		fn(rel, de.IsDir())
		return nil
	})
	if err == filepath.SkipDir {
		return nil
	}
	return err
}

// Fingerprint identifies the document corpus on disk: the path, size, and
// mtime of each in-place document, of each file under raw/ and formatted/,
// and of .spekk/docs-paths. It reads no file content. The index stores it and
// rebuilds the document part when it changes, so a new, edited, or deleted
// document is seen, and an unrelated file is not.
func Fingerprint(root string) (string, error) {
	files, err := inPlaceFiles(root)
	if err != nil {
		return "", err
	}
	for _, dir := range []string{RawDir, FormattedDir} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(p string, de fs.DirEntry, err error) error {
			if err != nil || de.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			files = append(files, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	files = append(files, pathsFile)
	sort.Strings(files)
	h := sha256.New()
	for _, rel := range files {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", rel, info.Size(), info.ModTime().UnixNano())
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
