package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/docs"
)

// The document part of the index is refreshed at query time
// (specs/source-document-index/, "Freshness"). The dirty check stats the
// corpus files and compares them with the doc_files manifest; it reads no
// file content. A file whose size or mtime changed is hashed, and only a file
// whose content changed is indexed again, with only its documents.

// InlineLimit is the most changed document files a query indexes before it
// answers. Above it the query answers from the current index and a
// background `spekk index` does the work. Measured on the 518-PEP corpus on
// a 2-vCPU machine: a write costs about 110 ms (the commit and the WAL
// checkpoint), and each document about 12 ms more, so 10 files cost at most
// about 230 ms.
const InlineLimit = 10

// lockStale is the age after which a lock file is left from a process that
// did not finish, and is removed.
const lockStale = 10 * time.Minute

// LockPath is the lock file that a background refresh holds.
func LockPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".spekk", "index.lock")
}

// manifestRow is one row of doc_files.
type manifestRow struct {
	docs.FileStat
	sha string
}

// querier is a *sql.DB or a *sql.Tx.
type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// manifestHashKey is the index_meta key that holds statsHash of the
// manifest, so the check of a clean tree reads one row, not the manifest.
const manifestHashKey = "doc_files_hash"

// statsHash identifies a set of file stats.
func statsHash(stats map[string]docs.FileStat) string {
	paths := make([]string, 0, len(stats))
	for p := range stats {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", p, stats[p].Size, stats[p].MTime)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// storeManifestHash records the statsHash of the manifest rows in tx.
func storeManifestHash(tx *sql.Tx) error {
	m, err := readManifest(tx)
	if err != nil {
		return err
	}
	stats := make(map[string]docs.FileStat, len(m))
	for p, r := range m {
		stats[p] = r.FileStat
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO index_meta (key, value) VALUES (?, ?)`, manifestHashKey, statsHash(stats)); err != nil {
		return fmt.Errorf("cannot record the manifest hash: %w", err)
	}
	return nil
}

func readManifest(q querier) (map[string]manifestRow, error) {
	rows, err := q.Query(`SELECT path, size, mtime_ns, sha256 FROM doc_files`)
	if err != nil {
		return nil, fmt.Errorf("cannot read the document manifest: %w", err)
	}
	defer rows.Close()
	out := map[string]manifestRow{}
	for rows.Next() {
		var p string
		var r manifestRow
		if err := rows.Scan(&p, &r.Size, &r.MTime, &r.sha); err != nil {
			return nil, fmt.Errorf("cannot read the document manifest: %w", err)
		}
		out[p] = r
	}
	return out, rows.Err()
}

// docChanges is what the dirty check found: files that are new, gone, or
// have another size or mtime than the manifest says.
type docChanges struct {
	added, deleted, touched []string
	current                 map[string]docs.FileStat
	manifest                map[string]manifestRow
}

func (c docChanges) count() int { return len(c.added) + len(c.deleted) + len(c.touched) }

func diffCorpus(current map[string]docs.FileStat, manifest map[string]manifestRow) docChanges {
	c := docChanges{current: current, manifest: manifest}
	for p, st := range current {
		m, ok := manifest[p]
		switch {
		case !ok:
			c.added = append(c.added, p)
		case m.FileStat != st:
			c.touched = append(c.touched, p)
		}
	}
	for p := range manifest {
		if _, ok := current[p]; !ok {
			c.deleted = append(c.deleted, p)
		}
	}
	sort.Strings(c.added)
	sort.Strings(c.deleted)
	sort.Strings(c.touched)
	return c
}

// checkDocs runs the dirty check against the index at dbPath. When the
// stats match the stored manifest hash, the manifest is not read.
func checkDocs(root, dbPath string) (docChanges, error) {
	current, err := docs.CorpusStat(root)
	if err != nil {
		return docChanges{}, err
	}
	db, err := openRead(dbPath)
	if err != nil {
		return docChanges{}, err
	}
	defer db.Close()
	var stored string
	err = db.QueryRow(`SELECT value FROM index_meta WHERE key = ?`, manifestHashKey).Scan(&stored)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return docChanges{}, fmt.Errorf("cannot read the manifest hash: %w", err)
	}
	if stored == statsHash(current) {
		return docChanges{current: current}, nil
	}
	manifest, err := readManifest(db)
	if err != nil {
		return docChanges{}, err
	}
	return diffCorpus(current, manifest), nil
}

// hashBudget is the most bytes of touched files that a query hashes to
// count the real changes before it chooses the background refresh.
const hashBudget = 64 << 20

// contentChanges counts the changes of c that are real: an added or deleted
// file, or a touched file whose sha256 differs from the manifest. A touch
// with the same content (git checkout, touch) is not a change. Past
// hashBudget, a touched file counts as changed without a hash.
func contentChanges(root string, c docChanges) int {
	n := len(c.added) + len(c.deleted)
	budget := int64(hashBudget)
	for _, p := range c.touched {
		budget -= c.current[p].Size
		if budget < 0 {
			n++
			continue
		}
		sum, err := docs.FileSHA256(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil || sum != c.manifest[p].sha {
			n++
		}
	}
	return n
}

// refreshDocs brings the document part of the index up to date with the
// files on disk, inside tx. The check runs again inside the write
// transaction, so two processes that saw the same change index it once. It
// returns the number of documents indexed again and the warnings.
func refreshDocs(tx *sql.Tx, root string) (int, []string, error) {
	current, err := docs.CorpusStat(root)
	if err != nil {
		return 0, nil, err
	}
	manifest, err := readManifest(tx)
	if err != nil {
		return 0, nil, err
	}
	c := diffCorpus(current, manifest)
	if c.count() == 0 {
		// The hash can be behind the manifest when a file changed while
		// the last build read it.
		return 0, nil, storeManifestHash(tx)
	}

	// A touched file with the same content needs only a new manifest row.
	sums := map[string]string{}
	var changed []string
	for _, p := range c.touched {
		sum, err := docs.FileSHA256(filepath.Join(root, filepath.FromSlash(p)))
		if errors.Is(err, fs.ErrNotExist) {
			continue // gone since the stat: the next check sees it
		}
		if err != nil {
			return 0, nil, fmt.Errorf("cannot hash %s: %w", p, err)
		}
		sums[p] = sum
		if sum != c.manifest[p].sha {
			changed = append(changed, p)
		}
	}
	for _, p := range c.added {
		sum, err := docs.FileSHA256(filepath.Join(root, filepath.FromSlash(p)))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return 0, nil, fmt.Errorf("cannot hash %s: %w", p, err)
		}
		sums[p] = sum
		changed = append(changed, p)
	}
	changed = append(changed, c.deleted...)

	full := false
	for _, p := range changed {
		if p == docs.PathsFile || p == docs.RawIgnoreFile {
			full = true
		}
	}
	n := 0
	var warnings []string
	if full {
		if err := clearDocs(tx); err != nil {
			return 0, nil, err
		}
		docCount, _, w, err := indexCorpus(tx, root)
		if err != nil {
			return 0, nil, err
		}
		return docCount, w, nil
	}
	if len(changed) > 0 {
		n, warnings, err = reindexDocuments(tx, root, current, changed)
		if err != nil {
			return 0, nil, err
		}
	}

	for _, p := range c.deleted {
		if _, err := tx.Exec(`DELETE FROM doc_files WHERE path = ?`, p); err != nil {
			return 0, nil, fmt.Errorf("cannot update the document manifest: %w", err)
		}
	}
	for p, sum := range sums {
		st := c.current[p]
		if _, err := tx.Exec(`INSERT OR REPLACE INTO doc_files (path, size, mtime_ns, sha256) VALUES (?, ?, ?, ?)`,
			p, st.Size, st.MTime, sum); err != nil {
			return 0, nil, fmt.Errorf("cannot update the document manifest: %w", err)
		}
	}
	return n, warnings, storeManifestHash(tx)
}

// reindexDocuments deletes and inserts again the documents of the changed
// files: the in-place document of a path, and the raw document whose stub,
// raw file, or derived text is the path.
func reindexDocuments(tx *sql.Tx, root string, current map[string]docs.FileStat, changed []string) (int, []string, error) {
	// Two in-place paths can have the same id; the first in path order is
	// the document, as in docs.Scan.
	var inPlace []string
	for rel := range current {
		if !strings.HasPrefix(rel, docs.RawDir+"/") && !strings.HasPrefix(rel, docs.FormattedDir+"/") && rel != docs.PathsFile {
			inPlace = append(inPlace, rel)
		}
	}
	sort.Strings(inPlace)
	pathsOfID := map[string][]string{}
	for _, rel := range inPlace {
		id := docs.ID(rel)
		pathsOfID[id] = append(pathsOfID[id], rel)
	}

	docIDs, rawIDs := map[string]bool{}, map[string]bool{}
	for _, p := range changed {
		if strings.HasPrefix(p, docs.FormattedDir+"/") && path.Ext(p) == ".md" && !strings.Contains(strings.TrimPrefix(p, docs.FormattedDir+"/"), "/") {
			rawIDs[strings.TrimSuffix(path.Base(p), ".md")] = true
		}
		rows, err := tx.Query(`SELECT kind, id FROM documents WHERE kind IN ('doc', 'raw') AND (path = ? OR text_path = ?)`, p, p)
		if err != nil {
			return 0, nil, fmt.Errorf("cannot find the documents of %s: %w", p, err)
		}
		for rows.Next() {
			var kind, id string
			if err := rows.Scan(&kind, &id); err != nil {
				rows.Close()
				return 0, nil, err
			}
			if kind == docs.KindRaw {
				rawIDs[id] = true
			} else {
				docIDs[id] = true
			}
		}
		rows.Close()
		if !strings.HasPrefix(p, docs.RawDir+"/") && !strings.HasPrefix(p, docs.FormattedDir+"/") {
			docIDs[docs.ID(p)] = true
		}
	}

	n := 0
	var warnings []string
	for _, id := range sortedKeys(docIDs) {
		if err := deleteDocument(tx, docs.KindDoc, id); err != nil {
			return 0, nil, err
		}
		paths := pathsOfID[id]
		if len(paths) == 0 {
			continue
		}
		for _, other := range paths[1:] {
			warnings = append(warnings, fmt.Sprintf("%s and %s have the same document id %q; only %s is indexed", paths[0], other, id, paths[0]))
		}
		d, err := docs.ReadInPlace(root, paths[0])
		if err != nil {
			return 0, nil, err
		}
		if _, err := insertDocument(tx, d, filepath.Join(root, filepath.FromSlash(d.TextPath))); err != nil {
			return 0, nil, err
		}
		n++
	}
	ignored := docs.RawIgnored(root)
	for _, id := range sortedKeys(rawIDs) {
		if err := deleteDocument(tx, docs.KindRaw, id); err != nil {
			return 0, nil, err
		}
		name := id + ".md"
		if _, err := os.Stat(filepath.Join(root, docs.FormattedDir, name)); err != nil {
			continue
		}
		d, warning := docs.ReadRawStub(root, name, ignored)
		if warning != "" {
			warnings = append(warnings, warning)
			continue
		}
		if _, err := insertDocument(tx, d, filepath.Join(root, filepath.FromSlash(d.TextPath))); err != nil {
			return 0, nil, err
		}
		n++
	}
	return n, warnings, nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// deleteDocument deletes one document, its tags, and its passages.
func deleteDocument(tx *sql.Tx, kind, id string) error {
	var first, last sql.NullInt64
	err := tx.QueryRow(`SELECT passage_first, passage_last FROM documents WHERE kind = ? AND id = ?`, kind, id).Scan(&first, &last)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("cannot delete document %s %q: %w", kind, id, err)
	}
	if first.Valid && last.Valid {
		if _, err := tx.Exec(`DELETE FROM passages WHERE rowid BETWEEN ? AND ?`, first.Int64, last.Int64); err != nil {
			return fmt.Errorf("cannot delete document %s %q: %w", kind, id, err)
		}
	}
	for _, stmt := range []string{
		`DELETE FROM documents WHERE kind = ? AND id = ?`,
		`DELETE FROM document_tags WHERE kind = ? AND doc_id = ?`,
	} {
		if _, err := tx.Exec(stmt, kind, id); err != nil {
			return fmt.Errorf("cannot delete document %s %q: %w", kind, id, err)
		}
	}
	return nil
}

// clearDocs deletes every doc and raw row and the manifest.
func clearDocs(tx *sql.Tx) error {
	for _, stmt := range []string{
		"DELETE FROM doc_files",
		"DELETE FROM index_meta WHERE key = '" + manifestHashKey + "'",
		"DELETE FROM documents WHERE kind IN ('doc', 'raw')",
		"DELETE FROM document_tags WHERE kind IN ('doc', 'raw')",
		"DELETE FROM passages WHERE kind IN ('doc', 'raw')",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("cannot clear the index (%s): %w", stmt, err)
		}
	}
	return nil
}

// RefreshDocs applies every pending change to the document part, inline.
// It is what `spekk index` without --force and the background refresh run.
func RefreshDocs(specsDir, dbPath string) (int, []string, error) {
	db, err := openWrite(dbPath)
	if err != nil {
		return 0, nil, err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return 0, nil, fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	n, warnings, err := refreshDocs(tx, filepath.Dir(specsDir))
	if err != nil {
		return 0, nil, err
	}
	if err := tx.Commit(); err != nil {
		return 0, nil, fmt.Errorf("cannot commit transaction: %w", err)
	}
	return n, warnings, nil
}

// RefreshInBackground is `spekk index --background`: it takes the lock, or
// returns at once when another process holds it, and refreshes the index.
// The process has no terminal, so a failure is written to the error file,
// and the next query refreshes inline, where the error reaches a person.
func RefreshInBackground(specsDir, dbPath string) error {
	root := filepath.Dir(specsDir)
	release, ok := takeLock(LockPath(root))
	if !ok {
		return nil
	}
	defer release()
	_, err := Refresh(specsDir, dbPath, false)
	if err != nil {
		_ = os.WriteFile(errorPath(root), []byte(err.Error()+"\n"), 0o644)
	}
	return err
}

// errorPath is the file where a background refresh writes its failure.
func errorPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".spekk", "index.error")
}

// takeLock creates the lock file with O_EXCL. A lock older than lockStale
// is removed first. It reports false when another process holds the lock.
func takeLock(lock string) (func(), bool) {
	if info, err := os.Stat(lock); err == nil && time.Since(info.ModTime()) > lockStale {
		_ = os.Remove(lock)
	}
	f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, false
	}
	fmt.Fprintf(f, "%d\n", os.Getpid())
	f.Close()
	return func() { _ = os.Remove(lock) }, true
}

// lockHeld reports whether a background refresh holds the lock.
func lockHeld(lock string) bool {
	info, err := os.Stat(lock)
	return err == nil && time.Since(info.ModTime()) <= lockStale
}

// startBackground starts `spekk index --background` in the repository,
// detached from the query.
func startBackground(repoRoot string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "index", "--background")
	cmd.Dir = repoRoot
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
