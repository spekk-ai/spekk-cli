package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/docs"
	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// indexDocuments writes the full-text corpus inside the build transaction:
// one documents row and its passages for each spec, assertion, in-place doc,
// and raw source document. It reads plaintext only, so the rebuild needs no
// converter. It returns the count of doc and raw documents, the count of all
// passages, and the warnings of the corpus scan.
func indexDocuments(tx *sql.Tx, specsDir string, result *parser.ParseResult) (int, int, []string, error) {
	root := filepath.Dir(specsDir)
	passages := 0

	insert := func(d docs.Document, file string) error {
		if _, err := tx.Exec(
			`INSERT INTO documents (kind, id, path, text_path, title, type, created, summary, sha256, state)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			d.Kind, d.ID, d.Path, d.TextPath, d.Title, d.Type, d.Created, d.Summary, d.SHA256, d.State,
		); err != nil {
			return fmt.Errorf("cannot insert document %s %q: %w", d.Kind, d.ID, err)
		}
		for _, tag := range d.Tags {
			if _, err := tx.Exec(`INSERT INTO document_tags (kind, doc_id, tag) VALUES (?, ?, ?)`, d.Kind, d.ID, tag); err != nil {
				return fmt.Errorf("cannot insert tag of %q: %w", d.ID, err)
			}
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			// A raw document whose text is absent keeps its row, so status
			// and search can name it, and has no passages.
			return nil
		}
		meta := strings.Join(append([]string{d.Title, d.Summary}, d.Tags...), " ")
		for _, p := range docs.Chunk(string(raw)) {
			if _, err := tx.Exec(
				`INSERT INTO passages (meta, heading, body, terms, kind, doc_id, start_line, end_line)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				meta, p.Heading, p.Text, docs.IdentifierTerms(p.Heading+" "+p.Text), d.Kind, d.ID, p.Start, p.End,
			); err != nil {
				return fmt.Errorf("cannot insert passage of %q: %w", d.ID, err)
			}
			passages++
		}
		return nil
	}

	// A spec's File is "specs/<dir>/<file>" whatever the directory is named,
	// so the file on disk is read relative to specsDir.
	specFile := func(rel string) string {
		return filepath.Join(specsDir, filepath.FromSlash(strings.TrimPrefix(rel, "specs/")))
	}
	for _, s := range result.Specs {
		d := docs.Document{Kind: docs.KindSpec, ID: s.ID, Path: s.File, TextPath: s.File, Title: s.Title, Summary: s.Title, Created: s.Created, State: docs.StateOK}
		if err := insert(d, specFile(s.File)); err != nil {
			return 0, 0, nil, err
		}
	}
	for _, a := range result.Assertions {
		d := docs.Document{Kind: docs.KindAssertion, ID: a.ID, Path: a.File, TextPath: a.File, Title: a.Title, Summary: a.Title, Created: a.Created, State: docs.StateOK}
		if err := insert(d, specFile(a.File)); err != nil {
			return 0, 0, nil, err
		}
	}

	corpus, err := docs.Scan(root)
	if err != nil {
		return 0, 0, nil, err
	}
	for _, d := range corpus.Docs {
		if err := insert(d, filepath.Join(root, filepath.FromSlash(d.TextPath))); err != nil {
			return 0, 0, nil, err
		}
	}
	return len(corpus.Docs), passages, corpus.Warnings, nil
}
