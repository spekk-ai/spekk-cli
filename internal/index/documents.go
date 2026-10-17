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
	raw, err := docs.ReadRegular(file)
	if err != nil {
		// A raw document whose text is absent keeps its row, so status and
		// search can name it, and has no passages.
		raw = nil
	}
	return insertText(tx, d, raw)
}

// insertText is insertDocument with the text already read. A nil raw is an
// absent text.
func insertText(tx *sql.Tx, d docs.Document, raw []byte) (int, error) {
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
	if raw == nil {
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

// indexSpecDocuments brings the documents rows and the passages of the specs
// and assertions in line with result. texts holds the bytes of each spec file
// that the parse read and sums their sha256, so the passages and the sha256
// of a document come from the text of its rows. A document whose file has
// the same path and sha256 as its row keeps its passages, so a spec edit
// indexes again only the documents of the changed file. It returns the
// number of spec and assertion passages in the index.
func indexSpecDocuments(tx *sql.Tx, result *parser.ParseResult, texts map[string][]byte, sums map[string]string) (int, error) {
	type indexed struct {
		path, sha   string
		first, last sql.NullInt64
	}
	old := map[[2]string]indexed{}
	rows, err := tx.Query(`SELECT kind, id, path, sha256, passage_first, passage_last FROM documents WHERE kind IN ('spec', 'assertion')`)
	if err != nil {
		return 0, fmt.Errorf("cannot read the spec documents: %w", err)
	}
	for rows.Next() {
		var kind, id string
		var r indexed
		if err := rows.Scan(&kind, &id, &r.path, &r.sha, &r.first, &r.last); err != nil {
			rows.Close()
			return 0, fmt.Errorf("cannot read the spec documents: %w", err)
		}
		old[[2]string{kind, id}] = r
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("cannot read the spec documents: %w", err)
	}

	var current []docs.Document
	for _, s := range result.Specs {
		current = append(current, docs.Document{Kind: docs.KindSpec, ID: s.ID, Path: s.File, TextPath: s.File, Title: s.Title, Summary: s.Title, Created: s.Created, State: docs.StateOK})
	}
	for _, a := range result.Assertions {
		current = append(current, docs.Document{Kind: docs.KindAssertion, ID: a.ID, Path: a.File, TextPath: a.File, Title: a.Title, Summary: a.Title, Created: a.Created, State: docs.StateOK})
	}
	total := 0
	for _, d := range current {
		raw := texts[d.Path]
		d.SHA256 = sums[d.Path]
		key := [2]string{d.Kind, d.ID}
		r, ok := old[key]
		delete(old, key)
		if ok && r.path == d.Path && r.sha == d.SHA256 && d.SHA256 != "" {
			if r.first.Valid && r.last.Valid {
				total += int(r.last.Int64 - r.first.Int64 + 1)
			}
			continue
		}
		if ok {
			if err := deleteDocument(tx, d.Kind, d.ID); err != nil {
				return 0, err
			}
		}
		n, err := insertText(tx, d, raw)
		if err != nil {
			return 0, err
		}
		total += n
	}
	for key := range old {
		if err := deleteDocument(tx, key[0], key[1]); err != nil {
			return 0, err
		}
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
