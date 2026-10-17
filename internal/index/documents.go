package index

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/docs"
	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// insertDocument writes one documents row, its tags, and its passages from
// the plaintext file. It returns the number of passages.
func insertDocument(tx *sql.Tx, d docs.Document, file string) (int, error) {
	passages := 0
	if _, err := tx.Exec(
		`INSERT INTO documents (kind, id, path, text_path, title, type, created, summary, sha256, state)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.Kind, d.ID, d.Path, d.TextPath, d.Title, d.Type, d.Created, d.Summary, d.SHA256, d.State,
	); err != nil {
		return 0, fmt.Errorf("cannot insert document %s %q: %w", d.Kind, d.ID, err)
	}
	for _, tag := range d.Tags {
		if _, err := tx.Exec(`INSERT INTO document_tags (kind, doc_id, tag) VALUES (?, ?, ?)`, d.Kind, d.ID, tag); err != nil {
			return 0, fmt.Errorf("cannot insert tag of %q: %w", d.ID, err)
		}
	}
	raw, err := docs.ReadRegular(file)
	if err != nil {
		// A raw document whose text is absent keeps its row, so status and
		// search can name it, and has no passages.
		return 0, nil
	}
	ps := docs.Chunk(string(raw))
	var first, last int64
	add := func(heading, body string, start, end, header int) error {
		res, err := tx.Exec(
			`INSERT INTO passages (heading, body, terms, kind, doc_id, start_line, end_line, is_header)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			heading, body, docs.IdentifierTerms(heading+" "+body), d.Kind, d.ID, start, end, header,
		)
		if err != nil {
			return fmt.Errorf("cannot insert passage of %q: %w", d.ID, err)
		}
		if last, err = res.LastInsertId(); err != nil {
			return fmt.Errorf("cannot insert passage of %q: %w", d.ID, err)
		}
		if passages == 0 {
			first = last
		}
		passages++
		return nil
	}
	// A spec or an assertion has its title as the heading of its first
	// passage; another document gets a header row for its title, summary,
	// and tags, at the line range of its first passage.
	if d.Kind == docs.KindDoc || d.Kind == docs.KindRaw {
		start, end := 1, 1
		if len(ps) > 0 {
			start, end = ps[0].Start, ps[0].End
		}
		if err := add(d.Title, strings.Join(append([]string{d.Summary}, d.Tags...), " "), start, end, 1); err != nil {
			return 0, err
		}
	}
	for _, p := range ps {
		if err := add(p.Heading, p.Text, p.Start, p.End, 0); err != nil {
			return 0, err
		}
	}
	if passages > 0 {
		if _, err := tx.Exec(`UPDATE documents SET passage_first = ?, passage_last = ? WHERE kind = ? AND id = ?`,
			first, last, d.Kind, d.ID); err != nil {
			return 0, fmt.Errorf("cannot record the passages of %q: %w", d.ID, err)
		}
	}
	return passages, nil
}

// indexSpecDocuments writes a documents row and the passages of each spec and
// assertion body. It returns the number of passages.
func indexSpecDocuments(tx *sql.Tx, specsDir string, result *parser.ParseResult) (int, error) {
	// A spec's File is "specs/<dir>/<file>" whatever the directory is named,
	// so the file on disk is read relative to specsDir.
	specFile := func(rel string) string {
		return filepath.Join(specsDir, filepath.FromSlash(strings.TrimPrefix(rel, "specs/")))
	}
	total := 0
	for _, s := range result.Specs {
		d := docs.Document{Kind: docs.KindSpec, ID: s.ID, Path: s.File, TextPath: s.File, Title: s.Title, Summary: s.Title, Created: s.Created, State: docs.StateOK}
		n, err := insertDocument(tx, d, specFile(s.File))
		if err != nil {
			return 0, err
		}
		total += n
	}
	for _, a := range result.Assertions {
		d := docs.Document{Kind: docs.KindAssertion, ID: a.ID, Path: a.File, TextPath: a.File, Title: a.Title, Summary: a.Title, Created: a.Created, State: docs.StateOK}
		n, err := insertDocument(tx, d, specFile(a.File))
		if err != nil {
			return 0, err
		}
		total += n
	}
	return total, nil
}

// indexCorpus writes every doc and raw document and the manifest of the
// files they came from. It reads plaintext only, so the rebuild needs no
// converter. It returns the count of documents, the count of their passages,
// and the warnings of the corpus scan.
func indexCorpus(tx *sql.Tx, root string) (int, int, []string, error) {
	// The manifest is taken before the scan, so a file that changes during
	// the build makes the next check index it again.
	stats, err := docs.CorpusStat(root)
	if err != nil {
		return 0, 0, nil, err
	}
	for rel, st := range stats {
		sum, err := docs.FileSHA256(filepath.Join(root, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) {
			continue // gone since the stat: the next check sees it
		}
		if err != nil {
			return 0, 0, nil, fmt.Errorf("cannot hash %s: %w", rel, err)
		}
		if _, err := tx.Exec(`INSERT OR REPLACE INTO doc_files (path, size, mtime_ns, sha256) VALUES (?, ?, ?, ?)`,
			rel, st.Size, st.MTime, sum); err != nil {
			return 0, 0, nil, fmt.Errorf("cannot write the document manifest: %w", err)
		}
	}
	if err := storeManifestHash(tx); err != nil {
		return 0, 0, nil, err
	}
	corpus, err := docs.Scan(root)
	if err != nil {
		return 0, 0, nil, err
	}
	total := 0
	for _, d := range corpus.Docs {
		n, err := insertDocument(tx, d, filepath.Join(root, filepath.FromSlash(d.TextPath)))
		if err != nil {
			return 0, 0, nil, err
		}
		total += n
	}
	return len(corpus.Docs), total, corpus.Warnings, nil
}
