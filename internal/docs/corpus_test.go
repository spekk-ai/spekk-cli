package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestID(t *testing.T) {
	for in, want := range map[string]string{
		"docs/advanced/index.md":     "docs-advanced-index",
		"README.md":                  "readme",
		"Vendor Notes/API v2.md":     "vendor-notes-api-v2",
		"papers/RFC_9110 (2022).pdf": "papers-rfc-9110-2022",
	} {
		if got := ID(in); got != want {
			t.Errorf("ID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInPlaceGlobs(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"README.md", "notes.txt", "sub/README.md", "docs/a.md", "docs/deep/b.md", "docs/deep/c.html",
		"docs/drafts/d.md", "adr/0001-use-sqlite.md", "adr/x/0002.md", "specs/s/s.md", "raw/r.md",
	} {
		write(t, root, rel, "# T\n\nBody.\n")
	}
	list := func() string {
		files, err := InPlaceFiles(root)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(files, " ")
	}
	// Defaults: docs/** text files, and *.md at the root only.
	if got, want := list(), "README.md docs/a.md docs/deep/b.md docs/drafts/d.md"; got != want {
		t.Errorf("default globs: got %q, want %q", got, want)
	}

	write(t, root, ".spekk/docs-paths", "# comment\n**/README.md\nadr/*.md\ndocs/**\n!docs/drafts/**\n!**/deep/*.md\nspecs/**\n")
	if got, want := list(), "README.md adr/0001-use-sqlite.md docs/a.md sub/README.md"; got != want {
		t.Errorf("custom globs: got %q, want %q", got, want)
	}
}

func TestScanRawStates(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
	add := func(rel, content, summary string) {
		write(t, root, rel, content)
		var out strings.Builder
		if n := Add(root, []string{filepath.Join(root, rel)}, AddOptions{Summary: summary, Now: now}, &out); n != 0 {
			t.Fatalf("add %s failed: %s", rel, out.String())
		}
	}
	summary := "A summary that is long enough to pass the forty character rule."
	add("raw/ok.md", "# OK\n\nok\n", summary)
	add("raw/nosum.md", "# No summary\n\nx\n", "")
	add("raw/stale.md", "# Stale\n\nold\n", "")
	write(t, root, "raw/stale.md", "# Stale\n\nnew\n") // edited after add: stale wins over no_summary
	add("raw/gone.md", "# Gone\n\nx\n", summary)
	os.Remove(filepath.Join(root, "raw/gone.md"))
	add("raw/big.md", "# Big\n\nx\n", summary)
	write(t, root, "raw/.gitignore", "/big.md\n")
	os.Remove(filepath.Join(root, "raw/big.md"))
	write(t, root, "raw/dropped.md", "# Dropped in by hand\n")

	c, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, d := range c.Docs {
		if d.Kind == KindRaw {
			got[d.ID] = d.State
		}
	}
	want := map[string]string{"ok": StateOK, "nosum": StateNoSummary, "stale": StateStale, "gone": StateMissingRaw, "big": StatePointer}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("states = %v, want %v", got, want)
	}
	if fmt.Sprint(c.NoStub) != "[raw/dropped.md]" {
		t.Errorf("NoStub = %v, want [raw/dropped.md]", c.NoStub)
	}
}

func TestCorpusStaysInsideTheRoot(t *testing.T) {
	outside := t.TempDir()
	write(t, outside, "secret.txt", "outside the repository")
	root := t.TempDir()
	write(t, root, "docs/ok.md", "# Ok\n")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "docs", "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "docs", "ok.md"), filepath.Join(root, "docs", "inner.md")); err != nil {
		t.Fatal(err)
	}
	write(t, root, "raw/r.txt", "a raw file")
	for name, fields := range map[string]string{
		"up":       "path: raw/r.txt\ntext: ../" + filepath.Base(outside) + "/secret.txt",
		"absolute": "path: " + filepath.Join(outside, "secret.txt"),
		"sideways": "path: docs/ok.md",
	} {
		write(t, root, "formatted/"+name+".md", "---\ntitle: x\n"+fields+"\n---\n")
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "raw", "via-link.txt")); err != nil {
		t.Fatal(err)
	}
	write(t, root, "formatted/via-link.md", "---\ntitle: x\npath: raw/via-link.txt\n---\n")

	c, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, d := range c.Docs {
		ids = append(ids, d.ID)
	}
	if strings.Join(ids, ",") != "docs-inner,docs-ok" {
		t.Errorf("indexed %v, want only docs-inner and docs-ok", ids)
	}
	if len(c.Warnings) != 4 {
		t.Errorf("warnings = %q, want one for each of the 4 stubs that leave the corpus", c.Warnings)
	}
}

func TestFirstParagraphSkipsAMetadataBlock(t *testing.T) {
	for text, want := range map[string]string{
		"# PEP 9\n\nPEP: 9 Title: Sample Author: A. Person\nStatus: Final\n\n# Abstract\n\nThe real summary.\n": "The real summary.",
		"# Notes\n\nNote: one field is a sentence, not metadata.\n":                                             "Note: one field is a sentence, not metadata.",
	} {
		if got := firstParagraph(text); got != want {
			t.Errorf("firstParagraph(%q) = %q, want %q", text, got, want)
		}
	}
}
