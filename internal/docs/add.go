package docs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// PointerSize is the size above which a converted file stays out of git: its
// path goes in raw/.gitignore, and the stub and derived text keep the
// document searchable on every clone.
const PointerSize = 5 << 20

// Summary length bounds, in characters after whitespace is collapsed.
const (
	MinSummary = 40
	MaxSummary = 600
)

// AddOptions are the flags of `spekk docs add`. An empty field keeps the
// value the stub already has.
type AddOptions struct {
	Summary      string
	Type         string
	Tags         []string
	Source       string
	AllowSecrets bool
	// Now is the date for a new stub's created field (tests set it).
	Now time.Time
}

// Add files each path (a file or a folder) into the corpus and writes its
// stub. It reports each file on out and returns the number that failed. It
// never overwrites a file in raw/ that has other content.
func Add(root string, paths []string, opts AddOptions, out io.Writer) int {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	if opts.Summary != "" {
		opts.Summary = strings.Join(strings.Fields(opts.Summary), " ")
		if n := utf8.RuneCountInString(opts.Summary); n < MinSummary || n > MaxSummary {
			fmt.Fprintf(out, "error: --summary has %d characters; it must have %d to %d (1-3 sentences that let an agent decide from the summary alone whether to read the document).\n", n, MinSummary, MaxSummary)
			return len(paths)
		}
	}
	failed := 0
	for _, arg := range paths {
		jobs, skipped, err := addJobs(root, arg)
		if err != nil {
			fmt.Fprintf(out, "error: %s: %v\n", arg, err)
			failed++
			continue
		}
		for _, s := range skipped {
			fmt.Fprintf(out, "skipped %s: unsupported format\n", s)
		}
		for _, j := range jobs {
			if err := addOne(root, j, opts, out); err != nil {
				fmt.Fprintf(out, "error: %s: %v\n", j.display, err)
				failed++
			}
		}
	}
	return failed
}

// addJob is one file to add: src is where it is now, dst is its path under
// raw/ (relative to root, with forward slashes).
type addJob struct {
	src, dst, display, source string
}

// addJobs expands one argument into files. A file outside raw/ goes to
// raw/<name>; a folder to raw/<folder>/...; a file under raw/ stays.
func addJobs(root, arg string) ([]addJob, []string, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read: %w", err)
	}
	rawAbs := filepath.Join(root, RawDir)
	under := func(p string) (string, bool) {
		rel, err := filepath.Rel(rawAbs, p)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return "", false
		}
		return RawDir + "/" + filepath.ToSlash(rel), true
	}
	source := func(p string) string {
		// A path inside the repository is kept as written. For a file from
		// outside, only the name is kept, so a home directory path never
		// enters git.
		if rel, err := filepath.Rel(root, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
		return filepath.Base(p)
	}

	if !info.IsDir() {
		if !Supported(abs) {
			return nil, nil, fmt.Errorf("unsupported format %q (supported: .md .markdown .txt .html .htm .eml .xlsx .pdf .docx)", filepath.Ext(abs))
		}
		dst, in := under(abs)
		if !in {
			dst = RawDir + "/" + filepath.Base(abs)
		}
		return []addJob{{src: abs, dst: dst, display: arg, source: source(abs)}}, nil, nil
	}

	var jobs []addJob
	var skipped []string
	err = filepath.WalkDir(abs, func(p string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if de.IsDir() {
			if p != abs && (strings.HasPrefix(de.Name(), ".") || de.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if de.Name() == ".gitignore" {
			return nil
		}
		disp := filepath.Join(arg, mustRel(abs, p))
		if !Supported(p) {
			skipped = append(skipped, disp)
			return nil
		}
		dst, in := under(p)
		if !in {
			dst = RawDir + "/" + filepath.ToSlash(filepath.Join(filepath.Base(abs), mustRel(abs, p)))
		}
		jobs = append(jobs, addJob{src: p, dst: dst, display: disp, source: source(p)})
		return nil
	})
	return jobs, skipped, err
}

func mustRel(base, p string) string {
	rel, _ := filepath.Rel(base, p)
	return rel
}

// addOne adds one file. Nothing is written until every check passed.
func addOne(root string, j addJob, opts AddOptions, out io.Writer) error {
	text, err := Extract(j.src)
	if err != nil {
		return fmt.Errorf("cannot convert: %w", err)
	}
	if hits := ScanSecrets(text); len(hits) > 0 && !opts.AllowSecrets {
		var where []string
		for _, h := range hits {
			where = append(where, fmt.Sprintf("%s on line %d", h.Kind, h.Line))
		}
		return fmt.Errorf("it holds what looks like a secret (%s). Git history is permanent, so the file was not added. Remove the secret, or, after a person checks the file, add it with --allow-secrets", strings.Join(where, ", "))
	}

	sum, err := FileSHA256(j.src)
	if err != nil {
		return err
	}
	info, err := os.Stat(j.src)
	if err != nil {
		return err
	}
	dstAbs := filepath.Join(root, filepath.FromSlash(j.dst))
	if dstAbs != j.src {
		if have, err := FileSHA256(dstAbs); err == nil && have != sum {
			return fmt.Errorf("%s already exists with other content. Rename the file, or put it under %s/ yourself and add it from there", j.dst, RawDir)
		}
	}

	id := ID(strings.TrimPrefix(j.dst, RawDir+"/"))
	if id == "" {
		return fmt.Errorf("cannot make a document id from %q", j.dst)
	}
	stubRel := FormattedDir + "/" + id + ".md"
	stubAbs := filepath.Join(root, filepath.FromSlash(stubRel))
	stub, err := ReadStub(stubAbs)
	exists := err == nil
	if exists && stub.Path != j.dst {
		return fmt.Errorf("its id %q is already used by %s (stub %s). Rename one of the two files", id, stub.Path, stubRel)
	}

	changed := exists && stub.SHA256 != sum
	if !exists {
		stub = Stub{Created: opts.Now.Format("2006-01-02"), Title: titleOf(text, j.dst)}
	}
	stub.Path = j.dst
	stub.SHA256 = sum
	stub.Size = info.Size()
	if stub.Source == "" || opts.Source != "" {
		stub.Source = firstNonEmpty(opts.Source, j.source)
	}
	if opts.Type != "" {
		stub.Type = opts.Type
	}
	if opts.Tags != nil {
		stub.Tags = opts.Tags
	}
	cleared := false
	switch {
	case opts.Summary != "":
		stub.Summary = opts.Summary
	case changed && stub.Summary != "":
		stub.Summary = ""
		cleared = true
	}
	pointer := !IsPlain(j.dst) && info.Size() > PointerSize

	// Every check passed: write the raw copy, the derived text, the
	// pointer entry, and the stub.
	if dstAbs != j.src {
		if err := copyFile(j.src, dstAbs); err != nil {
			return err
		}
	}
	stub.Text = ""
	if !IsPlain(j.dst) {
		stub.Text = FormattedDir + "/" + id + ".txt"
		if err := writeFile(filepath.Join(root, filepath.FromSlash(stub.Text)), []byte(text)); err != nil {
			return err
		}
	}
	if pointer {
		if err := addPointer(root, j.dst); err != nil {
			return err
		}
	}
	if err := writeFile(stubAbs, []byte(stub.Format())); err != nil {
		return err
	}

	what := "added"
	switch {
	case changed:
		what = "updated (content changed)"
	case exists:
		what = "unchanged"
	}
	fmt.Fprintf(out, "%s %s as %s: stub %s", what, j.dst, id, stubRel)
	if stub.Text != "" {
		fmt.Fprintf(out, ", text %s", stub.Text)
	}
	fmt.Fprintln(out)
	if pointer {
		fmt.Fprintf(out, "  %s is larger than %d MB, so it stays local: it is listed in %s/.gitignore. The stub and the text are committed, so search works on every clone.\n", j.dst, PointerSize>>20, RawDir)
	}
	if cleared {
		fmt.Fprintln(out, "  The content changed, so the old summary was cleared.")
	}
	if strings.TrimSpace(stub.Summary) == "" {
		fmt.Fprintf(out, "  The stub has no summary. Read the document (spekk docs show %s), then run:\n  spekk docs add %s --summary \"<1-3 sentences, %d-%d characters>\"\n", id, j.dst, MinSummary, MaxSummary)
	}
	return nil
}

// titleOf is the first heading of the text, else the file name.
func titleOf(text, rel string) string {
	for _, p := range Chunk(text) {
		if p.Heading != "" {
			return strings.SplitN(p.Heading, " / ", 2)[0]
		}
	}
	return strings.TrimSuffix(path.Base(rel), path.Ext(rel))
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func writeFile(p string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeFile(dst, data)
}

// addPointer lists a raw file in raw/.gitignore once.
func addPointer(root, rel string) error {
	entry := "/" + strings.TrimPrefix(rel, RawDir+"/")
	p := filepath.Join(root, RawDir, ".gitignore")
	raw, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}
	s := string(raw)
	if s == "" {
		s = "# Source documents larger than 5 MB stay local (spekk docs add).\n"
	} else if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	return writeFile(p, []byte(s+entry+"\n"))
}
