package index_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/docs"
	"github.com/spekk-ai/spekk-cli/internal/index"
)

// buildCorpus writes in-place docs under root/docs and builds the index.
func buildCorpus(t *testing.T, files map[string]string) string {
	t.Helper()
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	for rel, content := range files {
		writeFile(t, root, rel, content)
	}
	dbPath := filepath.Join(root, ".spekk", "index.db")
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	return dbPath
}

func searchIDs(t *testing.T, dbPath string, opts index.SearchOptions) []string {
	t.Helper()
	rs, err := index.Search(dbPath, opts)
	must(t, err)
	var ids []string
	for _, r := range rs {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestSearchMatchesIdentifiers(t *testing.T) {
	dbPath := buildCorpus(t, map[string]string{
		"docs/camel.md":  "# Camel\n\nThe parser entry point is parseAllSpecs.\n",
		"docs/snake.md":  "# Snake\n\nThe file index_db lives here.\n",
		"docs/dotted.md": "# Dotted\n\nSet foo.bar to true.\n",
		"docs/other.md":  "# Other\n\nThe db index of bar and foo, in another order.\n",
	})
	cases := map[string]string{
		"parseAllSpecs": "docs-camel",
		"parse specs":   "docs-camel",
		"index_db":      "docs-snake",
		"foo.bar":       "docs-dotted",
	}
	for q, want := range cases {
		ids := searchIDs(t, dbPath, index.SearchOptions{Query: q})
		if len(ids) == 0 || ids[0] != want {
			t.Errorf("search %q = %v, want %s first", q, ids, want)
		}
		if q == "index_db" || q == "foo.bar" {
			for _, id := range ids {
				if id == "docs-other" {
					t.Errorf("search %q matched the parts in another order (docs-other)", q)
				}
			}
		}
	}
}

func TestSearchFilters(t *testing.T) {
	dbPath := buildCorpus(t, map[string]string{
		"docs/old.md":  "---\ntitle: Old\ntype: manual\ntags: [pump]\ncreated: 2025-01-02\n---\n\nThe valve torque is 40 Nm.\n",
		"docs/new.md":  "---\ntitle: New\ntype: notes\ntags: [pump, field]\ncreated: 2026-05-01\n---\n\nThe valve was replaced.\n",
		"docs/bare.md": "# Bare\n\nA valve with no frontmatter.\n",
	})
	cases := []struct {
		opts index.SearchOptions
		want string
	}{
		{index.SearchOptions{Query: "valve", Type: "manual"}, "[docs-old]"},
		{index.SearchOptions{Query: "valve", Tag: "field"}, "[docs-new]"},
		{index.SearchOptions{Query: "valve", Since: "2026-01-01"}, "[docs-new]"},
		{index.SearchOptions{Query: "valve", Kinds: []string{"assertion"}}, "[]"},
	}
	for _, tc := range cases {
		if got := fmt.Sprint(searchIDs(t, dbPath, tc.opts)); got != tc.want {
			t.Errorf("search %+v = %s, want %s", tc.opts, got, tc.want)
		}
	}
}

func TestSearchOutputCap(t *testing.T) {
	var rs []index.SearchResult
	for i := 0; i < 20; i++ {
		rs = append(rs, index.SearchResult{
			Kind: "raw", ID: fmt.Sprintf("doc-%d", i), Title: strings.Repeat("T", 200), Path: "raw/x.pdf",
			Summary: strings.Repeat("s", 300), Lines: []docs.Line{{N: 1, Text: strings.Repeat("e", 160)}, {N: 2, Text: strings.Repeat("e", 160)}}, State: "ok", Also: []string{"1-2"},
		})
	}
	text := index.RenderSearchText("q", rs, 0)
	if len(text) > index.SearchCap {
		t.Errorf("text output is %d bytes, over the %d cap", len(text), index.SearchCap)
	}
	if !strings.Contains(text, "more results left out by the 4000-byte output cap") || !strings.Contains(text, index.UntrustedNotice) {
		t.Errorf("text output does not name the omitted results and the notice:\n%s", text)
	}

	out := index.RenderSearchJSON("q", rs, 0)
	if len(out) > index.SearchCap {
		t.Errorf("JSON output is %d bytes, over the %d cap", len(out), index.SearchCap)
	}
	var obj struct {
		Notice  string            `json:"notice"`
		Results []json.RawMessage `json:"results"`
		Omitted int               `json:"omitted"`
	}
	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		t.Fatalf("capped JSON does not parse: %v", err)
	}
	if len(obj.Results) == 0 || len(obj.Results)+obj.Omitted != 20 || obj.Notice == "" {
		t.Errorf("JSON has %d results, %d omitted, notice %q; want results+omitted = 20 and a notice", len(obj.Results), obj.Omitted, obj.Notice)
	}
}

func TestSearchCleansDocumentText(t *testing.T) {
	dbPath := buildCorpus(t, map[string]string{
		"docs/evil.md": "---\ntitle: \"Evil\x1b[31m title\"\n---\n\nThe quokka line one\n1. fake result line\x1b[2J\n",
	})
	rs, err := index.Search(dbPath, index.SearchOptions{Query: "quokka"})
	must(t, err)
	out := index.RenderSearchText("quokka", rs, 0)
	if strings.ContainsRune(out, 0x1b) {
		t.Errorf("escape character in output: %q", out)
	}
	if strings.Contains(out, "\n1. fake") {
		t.Errorf("document text forged an output line: %q", out)
	}
}

// A regression found in the walkthrough: FTS5 bm25() normalizes by the length
// of the whole row, so a summary on each passage made a three-line footer the
// best passage of a document whose summary matched.
func TestSearchShowsATextPassageNotAShortFooter(t *testing.T) {
	var long strings.Builder
	long.WriteString("---\nsummary: \"JSON Patch: add, remove, and test operations.\"\n---\n\n# Operations\n\n")
	for i := 0; i < 30; i++ {
		long.WriteString("Filler text about documents and members.\n")
	}
	long.WriteString("The test operation compares the value at the target location.\n\n# Authors\n\nPage 18\n")
	dbPath := buildCorpus(t, map[string]string{"docs/patch.md": long.String()})

	rs, err := index.Search(dbPath, index.SearchOptions{Query: "test operation"})
	must(t, err)
	if len(rs) == 0 || rs[0].ID != "docs-patch" {
		t.Fatalf("results = %+v, want docs-patch first", rs)
	}
	if rs[0].StartLine != 5 || !hasLine(rs[0].Lines, 37, "compares the value") {
		t.Errorf("best passage = lines %d-%d %+v, want the Operations section (line 5) with the match on line 37", rs[0].StartLine, rs[0].EndLine, rs[0].Lines)
	}
}

func hasLine(lines []docs.Line, n int, text string) bool {
	for _, l := range lines {
		if l.N == n && strings.Contains(l.Text, text) {
			return true
		}
	}
	return false
}
