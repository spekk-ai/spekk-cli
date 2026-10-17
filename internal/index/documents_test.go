package index_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/index"
)

// writeFile writes content to root/rel and creates its directories.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	must(t, os.MkdirAll(filepath.Dir(p), 0o755))
	must(t, os.WriteFile(p, []byte(content), 0o644))
}

func TestBuildIndexHoldsSpecBodies(t *testing.T) {
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	writeFile(t, root, "specs/spec-a/assertions/assert-three.md", `---
id: assert-three
parent: spec-a
created: 2026-01-01T02:00:00Z
priority: 1
status: not_started
---
# Assert Three

## Success criteria

- The importer rejects a zanzibar header.
`)
	dbPath := filepath.Join(root, ".spekk", "index.db")
	stats, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	if stats.Passages == 0 {
		t.Fatalf("stats.Passages = 0, want passages for spec bodies")
	}

	db := openDB(t, dbPath)
	if n := countRows(t, db, "documents"); n != 4 {
		t.Errorf("documents rows = %d, want 4 (1 spec, 3 assertions)", n)
	}
	var kind, id string
	var start, end int
	err = db.QueryRow(`SELECT kind, doc_id, start_line, end_line FROM passages WHERE passages MATCH 'zanzibar'`).Scan(&kind, &id, &start, &end)
	must(t, err)
	// Line numbers are those of the file: 7 frontmatter lines, then the
	// title on line 8, which has no body and joins the next section, and
	// the bullet on line 12.
	if kind != "assertion" || id != "assert-three" || start != 8 || end != 12 {
		t.Errorf("match = %s %s %d-%d, want assertion assert-three 8-12", kind, id, start, end)
	}
}

func TestEnsureFreshSeesDeletedDocument(t *testing.T) {
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	writeFile(t, root, "docs/guide.md", "# Guide\n\nThe wombat setting.\n")
	writeFile(t, root, "docs/other.md", "# Other\n\nText.\n")
	dbPath := filepath.Join(root, ".spekk", "index.db")
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)

	// A file that no glob selects, beside the root-level *.md documents,
	// does not make the index stale.
	writeFile(t, root, "build.log", "x")
	if stale, err := index.IsStale(specsDir, dbPath); err != nil || stale {
		t.Fatalf("IsStale after an unrelated file = %v, %v; want false", stale, err)
	}

	// A deletion changes no file that remains; the fingerprint still sees it.
	must(t, os.Remove(filepath.Join(root, "docs", "guide.md")))
	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	must(t, err)
	if !rebuilt {
		t.Fatal("EnsureFresh did not rebuild after a document was deleted")
	}
	rs, err := index.Search(dbPath, index.SearchOptions{Query: "wombat"})
	must(t, err)
	if len(rs) != 0 {
		t.Errorf("search still finds the deleted document: %+v", rs)
	}
}

// docRows returns the rows of the given kinds (an SQL list), documents and
// passages, in a stable order.
func docRows(t *testing.T, dbPath, kinds string) string {
	t.Helper()
	db := openDB(t, dbPath)
	var b strings.Builder
	for _, q := range []string{
		`SELECT kind, id, path, text_path, title, summary, sha256, state FROM documents WHERE kind IN (` + kinds + `) ORDER BY kind, id`,
		`SELECT kind, doc_id, start_line, end_line, is_header, heading, body FROM passages WHERE kind IN (` + kinds + `) ORDER BY kind, doc_id, start_line, is_header DESC`,
	} {
		rows, err := db.Query(q)
		must(t, err)
		cols, _ := rows.Columns()
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			must(t, rows.Scan(ptrs...))
			fmt.Fprintln(&b, vals...)
		}
		rows.Close()
	}
	return b.String()
}

func TestRefreshIndexesOnlyWhatChanged(t *testing.T) {
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	writeFile(t, root, "docs/a.md", "# A\n\nThe aardvark.\n")
	writeFile(t, root, "docs/b.md", "# B\n\nThe badger.\n")
	writeFile(t, root, "docs/c.md", "# C\n\nThe caracal.\n")
	writeFile(t, root, "raw/r.txt", "The raw file.\n")
	writeFile(t, root, "formatted/r.md", "---\ntitle: R\npath: raw/r.txt\ntext: formatted/r.txt\nsummary: A raw document about the rhea, long enough for a summary.\n---\n")
	writeFile(t, root, "formatted/r.txt", "The rhea.\n")
	dbPath := filepath.Join(root, ".spekk", "index.db")
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)

	refresh := func(want int, what string) {
		t.Helper()
		n, _, err := index.RefreshDocs(specsDir, dbPath)
		must(t, err)
		if n != want {
			t.Errorf("%s: re-indexed %d documents, want %d", what, n, want)
		}
	}
	later := time.Now().Add(time.Hour)
	must(t, os.Chtimes(filepath.Join(root, "docs", "b.md"), later, later))
	refresh(0, "a touch with the same content")
	refresh(0, "no change after the manifest was updated")

	writeFile(t, root, "docs/a.md", "# A\n\nThe axolotl.\n")
	refresh(1, "an edit")
	writeFile(t, root, "docs/d.md", "# D\n\nThe dingo.\n")
	refresh(1, "an add")
	must(t, os.Remove(filepath.Join(root, "docs", "c.md")))
	refresh(0, "a delete")
	writeFile(t, root, "formatted/r.txt", "The rhea, again.\n")
	refresh(1, "an edit of the derived text of a raw document")
	writeFile(t, root, "raw/r.txt", "The raw file, changed.\n")
	refresh(1, "an edit of a raw file (its state goes stale)")

	got := docRows(t, dbPath, "'doc', 'raw'")
	forced := filepath.Join(t.TempDir(), "forced.db")
	_, err = index.BuildIndex(specsDir, forced, true)
	must(t, err)
	if want := docRows(t, forced, "'doc', 'raw'"); got != want {
		t.Errorf("rows after the edits differ from a forced rebuild:\n%s\nwant:\n%s", got, want)
	}
}

func TestSpecEditIndexesOnlyThatSpec(t *testing.T) {
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	// A document after the spec passages, so a passage written again gets
	// another rowid.
	writeFile(t, root, "docs/guide.md", "# Guide\n\nText.\n")
	dbPath := filepath.Join(root, ".spekk", "index.db")
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	passageFirst := func(id string) int64 {
		t.Helper()
		var first int64
		must(t, openDB(t, dbPath).QueryRow(`SELECT passage_first FROM documents WHERE kind = 'assertion' AND id = ?`, id).Scan(&first))
		return first
	}
	touchedBefore := passageFirst("assert-one")

	// A touch alone does not rebuild the spec part, and the next check
	// does not see it again.
	touched := filepath.Join(specsDir, "spec-a", "assertions", "assert-one.md")
	earlier := time.Now().Add(-time.Hour)
	must(t, os.Chtimes(dbPath, earlier, earlier))
	must(t, os.Chtimes(touched, earlier.Add(time.Minute), earlier.Add(time.Minute)))
	if rebuilt, err := index.EnsureFresh(specsDir, dbPath); err != nil || rebuilt {
		t.Errorf("EnsureFresh after a touch = %v, %v; want no rebuild", rebuilt, err)
	}
	if stale, err := index.IsStale(specsDir, dbPath); err != nil || stale {
		t.Errorf("IsStale after the touch was checked = %v, %v; want false", stale, err)
	}

	later := time.Now().Add(time.Hour)
	must(t, os.Chtimes(touched, later, later))
	edited := filepath.Join(specsDir, "spec-a", "assertions", "assert-two.md")
	raw, err := os.ReadFile(edited)
	must(t, err)
	must(t, os.WriteFile(edited, append(raw, "\nThe quokka rule.\n"...), 0o644))
	must(t, os.Chtimes(edited, later, later))
	rebuilt, err := index.EnsureFresh(specsDir, dbPath)
	must(t, err)
	if !rebuilt {
		t.Fatal("EnsureFresh did not rebuild after a spec edit")
	}

	if got := passageFirst("assert-one"); got != touchedBefore {
		t.Errorf("the touched assertion's passages were written again (first rowid %d, was %d)", got, touchedBefore)
	}
	rs, err := index.Search(dbPath, index.SearchOptions{Query: "quokka"})
	must(t, err)
	if len(rs) != 1 || rs[0].ID != "assert-two" {
		t.Errorf("search for the new text = %+v, want assert-two", rs)
	}

	must(t, os.WriteFile(edited, raw, 0o644))
	must(t, os.Chtimes(edited, later.Add(time.Hour), later.Add(time.Hour)))
	_, err = index.EnsureFresh(specsDir, dbPath)
	must(t, err)
	if rs, err := index.Search(dbPath, index.SearchOptions{Query: "quokka"}); err != nil || len(rs) != 0 {
		t.Errorf("search still finds the removed text: %+v, %v", rs, err)
	}
	forced := filepath.Join(t.TempDir(), "forced.db")
	_, err = index.BuildIndex(specsDir, forced, true)
	must(t, err)
	if got, want := docRows(t, dbPath, "'spec', 'assertion'"), docRows(t, forced, "'spec', 'assertion'"); got != want {
		t.Errorf("rows after the edits differ from a forced rebuild:\n%s\nwant:\n%s", got, want)
	}
}

func TestParallelRefreshesAllSucceed(t *testing.T) {
	specsDir := makeSpecs(t)
	root := filepath.Dir(specsDir)
	for i := range 20 {
		writeFile(t, root, fmt.Sprintf("docs/d%02d.md", i), fmt.Sprintf("# D%d\n\nText %d.\n", i, i))
	}
	dbPath := filepath.Join(root, ".spekk", "index.db")
	_, err := index.BuildIndex(specsDir, dbPath, false)
	must(t, err)
	writeFile(t, root, "docs/d03.md", "# D3\n\nThe numbat.\n")

	errs := make(chan error, 3)
	for range 3 {
		go func() {
			_, err := index.Refresh(specsDir, dbPath, false)
			errs <- err
		}()
	}
	for range 3 {
		if err := <-errs; err != nil {
			t.Errorf("a parallel refresh failed: %v", err)
		}
	}
	rs, err := index.Search(dbPath, index.SearchOptions{Query: "numbat"})
	must(t, err)
	db := openDB(t, dbPath)
	var n int
	must(t, db.QueryRow(`SELECT COUNT(*) FROM documents WHERE id = 'docs-d03'`).Scan(&n))
	if len(rs) != 1 || n != 1 {
		t.Errorf("after parallel refreshes: %d results, %d rows of docs-d03; want 1 and 1", len(rs), n)
	}
}
