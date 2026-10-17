// Package index builds and queries a SQLite index of the spec tree and of
// the cross-branch observation set.
// The index is stored at .spekk/index.db at the repo root (adjacent to specs/).
// It is a derived artifact — the Markdown files remain the source of truth.
//
// The invariant: every table in .spekk/index.db is either rebuildable from
// plaintext in the repo/branch set, or safe to lose. SQLite is strictly a
// derived, ephemeral layer — deleting .spekk/index.db loses nothing, and a
// rebuild reproduces it from the working tree and the visible git refs. No
// lifecycle state exists only here. Prompts and agents get SELECT-only
// access via `spekk query` (read-only connection); all writes happen through
// Go code paths in this package. The index may cache, accelerate, and join;
// it may never remember.
package index

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/spekk-ai/spekk-cli/internal/observation"
	"github.com/spekk-ai/spekk-cli/internal/parser"
)

const createSchema = `
CREATE TABLE IF NOT EXISTS specs (
  id       TEXT PRIMARY KEY,
  title    TEXT,
  status   TEXT,
  priority INTEGER,
  branch   TEXT,
  file     TEXT
);
CREATE TABLE IF NOT EXISTS assertions (
  id        TEXT PRIMARY KEY,
  parent_id TEXT REFERENCES specs(id),
  title     TEXT,
  status    TEXT,
  priority  INTEGER,
  branch    TEXT,
  file      TEXT
);
CREATE TABLE IF NOT EXISTS depends_on (
  assertion_id  TEXT REFERENCES assertions(id),
  depends_on_id TEXT REFERENCES assertions(id)
);
-- Custom frontmatter fields (keys outside the known set of the owner's own
-- format) on specs, assertions, and observations. Multi-value spellings
-- ([a, b] / a, b) are split into one row per value, so a query can filter
-- and aggregate on any custom tag. owner_id is the spec or assertion id, or
-- the observation slug: an observation is keyed by (slug, ref) elsewhere,
-- but a custom key is not a lifecycle field, so the rows are merged across
-- refs and do not depend on how many branches carry the slug.
CREATE TABLE IF NOT EXISTS frontmatter_fields (
  owner_type TEXT,   -- 'spec' | 'assertion' | 'observation'
  owner_id   TEXT,
  key        TEXT,
  value      TEXT
);
-- Observation tables mirror the observation frontmatter schema
-- (specs/observation-lifecycle/): one row per (slug, ref) pair read from the
-- visible observer/* branches plus main. announced is NULL when the
-- frontmatter field is absent — the announce eligibility test is
-- "announced IS NULL", so absence must stay distinguishable from any value.
CREATE TABLE IF NOT EXISTS observations (
  slug      TEXT,
  ref       TEXT,
  type      TEXT,
  severity  TEXT,
  status    TEXT,
  created   TEXT,
  announced TEXT,
  pr        TEXT,
  title     TEXT,
  file      TEXT,
  PRIMARY KEY (slug, ref)
);
-- One evidence row per affected: path, keyed to the same (slug, ref).
CREATE TABLE IF NOT EXISTS observation_files (
  slug TEXT,
  ref  TEXT,
  path TEXT
);
-- Internal bookkeeping (derived, like everything else): the fingerprint of
-- the observation-relevant refs the last build saw, so freshness checks can
-- detect fetched, created, or deleted observer branches.
CREATE TABLE IF NOT EXISTS index_meta (
  key   TEXT PRIMARY KEY,
  value TEXT
);
-- Full-text corpus (specs/source-document-index/): one row per spec,
-- assertion, in-place doc, and raw source document. path is the document
-- file; text_path is the plaintext file that the passage line numbers refer
-- to (for a converted raw document, its derived text in formatted/). state
-- is 'ok' for every kind but raw (see internal/docs).
CREATE TABLE IF NOT EXISTS documents (
  kind      TEXT,   -- 'spec' | 'assertion' | 'doc' | 'raw'
  id        TEXT,
  path      TEXT,
  text_path TEXT,
  title     TEXT,
  type      TEXT,
  created   TEXT,
  summary   TEXT,
  sha256    TEXT,
  state     TEXT,
  -- The rowids of the document's passages, so that a re-index deletes them
  -- by rowid: the kind and doc_id columns of an FTS5 table have no index.
  passage_first INTEGER,
  passage_last  INTEGER,
  PRIMARY KEY (kind, id)
);
CREATE TABLE IF NOT EXISTS document_tags (
  kind   TEXT,
  doc_id TEXT,
  tag    TEXT
);
-- The manifest of the document corpus (internal/index/fresh.go): one row per
-- file that the document part reads, so the dirty check at query time is a
-- stat walk, and only the documents of changed files are indexed again.
CREATE TABLE IF NOT EXISTS doc_files (
  path     TEXT PRIMARY KEY,
  size     INTEGER,
  mtime_ns INTEGER,
  sha256   TEXT
);
-- One row per passage (a heading section, or a 40-line window of a long
-- one). A doc or raw document also has one header row (is_header = 1) that
-- holds its title, summary, and tags, so a match on them ranks the document.
-- The summary is not repeated on each passage: FTS5 bm25() normalizes by the
-- length of the whole row, so a repeated summary made the shortest passage of
-- a matching document rank first. terms holds the camelCase parts of
-- identifiers (internal/docs.IdentifierTerms).
CREATE VIRTUAL TABLE IF NOT EXISTS passages USING fts5(
  heading, body, terms,
  kind UNINDEXED, doc_id UNINDEXED, start_line UNINDEXED, end_line UNINDEXED, is_header UNINDEXED,
  tokenize = 'porter unicode61 remove_diacritics 2'
);
`

// dropAllTables drops every table in the database, enumerated from
// sqlite_master rather than a hardcoded list. A binary can only name the
// tables of its own generation, and a hardcoded list is exactly how a
// version-skewed force rebuild leaves a future table's stale rows in place
// (an old binary rebuilding a newer database refreshed every table it knew
// and silently kept the rest). Dropping whatever exists makes every future
// schema bump safe against this binary going stale.
func dropAllTables(db *sql.DB) error {
	// Virtual tables come first: dropping one drops its shadow tables, which
	// SQLite does not let a caller drop on their own first.
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY (sql LIKE 'CREATE VIRTUAL TABLE%') DESC, rowid`)
	if err != nil {
		return fmt.Errorf("cannot list tables: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return fmt.Errorf("cannot read table name: %w", err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("cannot list tables: %w", err)
	}

	for _, n := range names {
		quoted := `"` + strings.ReplaceAll(n, `"`, `""`) + `"`
		if _, err := db.Exec(`DROP TABLE IF EXISTS ` + quoted); err != nil {
			return fmt.Errorf("cannot drop table %q: %w", n, err)
		}
	}
	return nil
}

// schemaVersion is the current on-disk index schema. Bump it whenever the table
// shape changes. Because the index is a derived artifact, migration is never an
// ALTER — readers (via EnsureFresh) rebuild any database whose stored
// user_version differs from this value, so a spekk upgrade heals the index on
// first use.
//
// Version history: 1 = specs/assertions/depends_on; 2 adds the observation
// tables (observations, observation_files, index_meta); 3 adds
// frontmatter_fields; 4 adds observation rows to frontmatter_fields; 5 adds
// the full-text corpus (documents, document_tags, passages); 6 adds the
// document manifest (doc_files) in place of one corpus fingerprint. Version
// 4 changes no column — the bump is what makes an index that version 3 built
// rebuild, instead of answering a custom-key query from rows that predate
// the change. Bump on a change to the rows a build writes, not only on a
// change to the shape it writes them into.
const schemaVersion = 6

// observationRefsKey is the index_meta key holding the observation
// RefsFingerprint the last build saw.
const observationRefsKey = "observation_refs"

// busyTimeoutMS is how long a connection waits for another process's write.
// The index is in WAL mode, so a reader never waits for a writer; a writer
// waits for the other writer.
const busyTimeoutMS = 10000

// ReadDSN is the data source name of a read-only connection to the index.
func ReadDSN(dbPath string) string {
	return fmt.Sprintf("file:%s?mode=ro&_pragma=busy_timeout(%d)", dbPath, busyTimeoutMS)
}

// openRead opens a read-only connection to the index.
func openRead(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", ReadDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("cannot open index.db: %w", err)
	}
	return db, nil
}

// openWrite opens a connection that writes the index: WAL, so that readers
// are not blocked; synchronous NORMAL, so that a commit does not wait for a
// disk sync (the index is derived, and a lost last commit is rebuilt on the
// next check); and BEGIN IMMEDIATE, so that a transaction takes the write
// lock before it reads what it will change.
func openWrite(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_txlock=immediate", dbPath, busyTimeoutMS))
	if err != nil {
		return nil, fmt.Errorf("cannot open index.db: %w", err)
	}
	return db, nil
}

// DBPath returns the canonical path for the index database given the repo root.
func DBPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".spekk", "index.db")
}

// Stats reports what a build indexed. Warnings name observation files that
// were skipped because they failed validation (including the evidence gate);
// such files are never silently indexed as valid.
type Stats struct {
	Specs        int
	Assertions   int
	Observations int
	// Documents counts kinds doc and raw; Passages counts every passage,
	// specs and assertions included.
	Documents int
	Passages  int
	Warnings  []string
}

// ErrSpecsUnparseable marks an index build that failed at the parse step and
// not at the database.
//
// The two failures need different corrections. A database failure applies to
// one checkout, and a person deletes .spekk/index.db to correct it. A
// spec-tree failure applies to all users: each spekk command builds the same
// index from the same committed files, so one malformed field stops all of
// them until a person edits that file.
//
// The previous message ("cannot parse specs") gave the cause but not the
// effect. An agent read that message, accepted the syntax error as the full
// problem, and proposed a second spelling that fails the same check.
var ErrSpecsUnparseable = errors.New("the spec tree does not parse, so the index cannot be rebuilt")

// specParseGuidance follows the parse error at each surface that a person
// reads. It gives the scope of the failure and names the command that reports
// each problem, and not only the first problem that the parser finds.
const specParseGuidance = `One malformed field fails the parse of the whole tree, not just its own file,
so no spekk command works until it is fixed — on every branch, for every user.
Run "spekk validate" to see every problem at once.`

// FormatError writes an index error for a person. A spec-tree failure gets
// the guidance text. Each other error passes through as written, so a
// database failure does not show as a spec problem.
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrSpecsUnparseable) {
		return err.Error() + "\n\n" + specParseGuidance
	}
	return err.Error()
}

// BuildIndex creates or updates the SQLite index at dbPath from specsDir,
// from the visible git refs (observations), and from the document corpus. If
// force is true all tables are dropped and recreated from scratch.
func BuildIndex(specsDir, dbPath string, force bool) (Stats, error) {
	return build(specsDir, dbPath, force, true, true)
}

// build rebuilds the spec part (specs, assertions, their passages, and the
// observations), the document part (doc and raw documents), or both. The
// parts go stale for different reasons, and a large document corpus must not
// be indexed again for each spec edit.
func build(specsDir, dbPath string, force, specsPart, docsPart bool) (Stats, error) {
	var stats Stats

	// Ensure the .spekk directory exists.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return stats, fmt.Errorf("cannot create .spekk directory: %w", err)
	}

	db, err := openWrite(dbPath)
	if err != nil {
		return stats, err
	}
	defer db.Close()

	if force {
		if err = dropAllTables(db); err != nil {
			return stats, fmt.Errorf("cannot drop tables: %w", err)
		}
	}

	if _, err = db.Exec(createSchema); err != nil {
		return stats, fmt.Errorf("cannot create schema: %w", err)
	}

	var result *parser.ParseResult
	if specsPart {
		result, err = parser.ParseAllSpecs(specsDir)
		if err != nil {
			return stats, fmt.Errorf("%w: %w", ErrSpecsUnparseable, err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return stats, fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Clear the rows of each part that is rebuilt (delete + re-insert).
	var clear []string
	if specsPart {
		clear = append(clear,
			"DELETE FROM depends_on", "DELETE FROM frontmatter_fields", "DELETE FROM assertions", "DELETE FROM specs",
			"DELETE FROM observation_files", "DELETE FROM observations",
			"DELETE FROM index_meta WHERE key = '"+observationRefsKey+"'",
			"DELETE FROM documents WHERE kind IN ('spec', 'assertion')",
			"DELETE FROM document_tags WHERE kind IN ('spec', 'assertion')",
			"DELETE FROM passages WHERE kind IN ('spec', 'assertion')")
	}
	for _, stmt := range clear {
		if _, err = tx.Exec(stmt); err != nil {
			return stats, fmt.Errorf("cannot clear the index (%s): %w", stmt, err)
		}
	}
	if docsPart {
		if err = clearDocs(tx); err != nil {
			return stats, err
		}
	}

	if specsPart {
		if err = insertSpecs(tx, result); err != nil {
			return stats, err
		}
		n, err := indexSpecDocuments(tx, specsDir, result)
		if err != nil {
			return stats, err
		}
		stats.Passages += n

		// Observation pass: read the cross-branch union from git refs.
		obsCount, warnings, err := indexObservations(tx)
		if err != nil {
			return stats, err
		}
		stats.Specs = len(result.Specs)
		stats.Assertions = len(result.Assertions)
		stats.Observations = obsCount
		stats.Warnings = append(stats.Warnings, warnings...)
	}

	if docsPart {
		docCount, passageCount, docWarnings, err := indexCorpus(tx, filepath.Dir(specsDir))
		if err != nil {
			return stats, err
		}
		stats.Documents = docCount
		stats.Passages += passageCount
		stats.Warnings = append(docWarnings, stats.Warnings...)
	}

	if err = tx.Commit(); err != nil {
		return stats, fmt.Errorf("cannot commit transaction: %w", err)
	}

	// Stamp the schema version so a future binary can detect and rebuild an
	// index built against an older schema. PRAGMA takes no bound parameters;
	// schemaVersion is a trusted integer constant.
	if _, err = db.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		return stats, fmt.Errorf("cannot set schema version: %w", err)
	}
	return stats, nil
}

// insertSpecs writes the spec, assertion, depends_on, and custom-field rows.
func insertSpecs(tx *sql.Tx, result *parser.ParseResult) error {
	for _, s := range result.Specs {
		if _, err := tx.Exec(
			`INSERT INTO specs (id, title, status, priority, branch, file) VALUES (?, ?, ?, ?, ?, ?)`,
			s.ID, s.Title, s.Status, s.Priority, s.Branch, s.File,
		); err != nil {
			return fmt.Errorf("cannot insert spec %q: %w", s.ID, err)
		}
		if err := insertFrontmatterFields(tx, "spec", s.ID, s.Fields); err != nil {
			return err
		}
	}

	for _, a := range result.Assertions {
		if _, err := tx.Exec(
			`INSERT INTO assertions (id, parent_id, title, status, priority, branch, file) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			a.ID, a.Parent, a.Title, a.Status, a.Priority, a.Branch, a.File,
		); err != nil {
			return fmt.Errorf("cannot insert assertion %q: %w", a.ID, err)
		}

		if a.DependsOn != "" {
			if _, err := tx.Exec(
				`INSERT INTO depends_on (assertion_id, depends_on_id) VALUES (?, ?)`,
				a.ID, a.DependsOn,
			); err != nil {
				return fmt.Errorf("cannot insert depends_on edge for %q: %w", a.ID, err)
			}
		}

		if err := insertFrontmatterFields(tx, "assertion", a.ID, a.Fields); err != nil {
			return err
		}
	}
	return nil
}

// insertFrontmatterFields writes one row per distinct (key, value) for an
// owner's custom frontmatter fields. The parser already split every
// multi-value spelling into items; a repeated value (`workflows: w1, w1`)
// inserts once, so COUNT-style report queries stay accurate.
func insertFrontmatterFields(tx *sql.Tx, ownerType, ownerID string, fields map[string][]string) error {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		seen := make(map[string]bool)
		for _, value := range fields[key] {
			if seen[value] {
				continue
			}
			seen[value] = true
			if _, err := tx.Exec(
				`INSERT INTO frontmatter_fields (owner_type, owner_id, key, value) VALUES (?, ?, ?, ?)`,
				ownerType, ownerID, key, value,
			); err != nil {
				return fmt.Errorf("cannot insert frontmatter field %q for %s %q: %w", key, ownerType, ownerID, err)
			}
		}
	}
	return nil
}

// indexObservations populates the observation tables inside the build
// transaction. It reads the observation union from the visible observer/*
// branches plus main via git object reads (internal/crossbranch) — never by
// checking branches out — and performs no remote operations: it indexes
// whatever refs are already visible, and keeping remote-tracking refs
// current is the caller's job (git fetch).
//
// Outside a git repository (RefsFingerprint returns "") the pass indexes
// nothing and records an empty fingerprint, so specs-only uses of the index
// keep working unchanged.
func indexObservations(tx *sql.Tx) (int, []string, error) {
	fingerprint, err := observation.RefsFingerprint()
	if err != nil {
		return 0, nil, fmt.Errorf("cannot fingerprint observation refs: %w", err)
	}

	count := 0
	var warnings []string
	if fingerprint != "" {
		u, err := observation.LoadUnion()
		if err != nil {
			return 0, nil, fmt.Errorf("cannot load observation union: %w", err)
		}
		warnings = u.Warnings

		// Custom fields per slug, collected here and written after the
		// loop, once per slug. See the frontmatter_fields comment above for
		// why the rows are merged across refs.
		fields := map[string]map[string][]string{}

		for _, o := range u.Observations {
			// announced must be SQL NULL when the frontmatter field is
			// absent, never the empty string.
			var announced any
			if o.Announced != "" {
				announced = o.Announced
			}
			var pr any
			if o.PR != "" {
				pr = o.PR
			}
			if _, err := tx.Exec(
				`INSERT INTO observations (slug, ref, type, severity, status, created, announced, pr, title, file)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				o.Slug, o.Ref, o.Type, o.Severity, o.Status, o.Created, announced, pr, o.Title, o.File,
			); err != nil {
				return 0, nil, fmt.Errorf("cannot insert observation %q at %q: %w", o.Slug, o.Ref, err)
			}
			for _, path := range o.Affected {
				if _, err := tx.Exec(
					`INSERT INTO observation_files (slug, ref, path) VALUES (?, ?, ?)`,
					o.Slug, o.Ref, path,
				); err != nil {
					return 0, nil, fmt.Errorf("cannot insert observation file for %q: %w", o.Slug, err)
				}
			}
			if len(o.Fields) > 0 && fields[o.Slug] == nil {
				fields[o.Slug] = map[string][]string{}
			}
			for key, values := range o.Fields {
				fields[o.Slug][key] = append(fields[o.Slug][key], values...)
			}
			count++
		}

		slugs := make([]string, 0, len(fields))
		for slug := range fields {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
		for _, slug := range slugs {
			if err := insertFrontmatterFields(tx, "observation", slug, fields[slug]); err != nil {
				return 0, nil, err
			}
		}
	}

	if _, err := tx.Exec(
		`INSERT INTO index_meta (key, value) VALUES (?, ?)`,
		observationRefsKey, fingerprint,
	); err != nil {
		return 0, nil, fmt.Errorf("cannot record observation refs fingerprint: %w", err)
	}
	return count, warnings, nil
}

// readSchemaVersion returns the user_version stamped into the index database.
// A database built by a version-unaware spekk (or any fresh SQLite file) reports
// 0, which never equals a real schemaVersion, so it is always treated as stale.
func readSchemaVersion(dbPath string) (int, error) {
	db, err := openRead(dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("cannot read schema version: %w", err)
	}
	return v, nil
}

// Freshness is what a refresh did before a read.
type Freshness struct {
	// Rebuilt is true when the refresh wrote the index.
	Rebuilt bool
	// Pending counts the document files that changed on disk and are not
	// in the index yet: a background refresh is indexing them, and the read
	// answers from the current index.
	Pending int
}

// EnsureFresh brings the index at dbPath up to date before a read, and
// reports whether it wrote the index. It rebuilds everything when the index
// is absent or stamped with another schema version, rebuilds the spec part
// when a file under specsDir is newer than the index or the observation refs
// changed (an observer branch was fetched, created, or deleted), and
// re-indexes the documents of changed corpus files. A large document change
// goes to a background refresh, so a read never waits for it (Refresh).
//
// This is the single freshness gate every index reader should call, so the
// "when do we rebuild" rule lives in exactly one place.
func EnsureFresh(specsDir, dbPath string) (bool, error) {
	f, err := Refresh(specsDir, dbPath, true)
	return f.Rebuilt, err
}

// EnsureFreshInline is EnsureFresh with no background refresh: every change
// is indexed before it returns.
func EnsureFreshInline(specsDir, dbPath string) (bool, error) {
	f, err := Refresh(specsDir, dbPath, false)
	return f.Rebuilt, err
}

// Refresh is EnsureFresh with the full result. With background true, a
// document change of more than InlineLimit files, or any change while a
// background refresh holds the lock, is left to the background refresh and
// counted in Pending. With no index at all, the build is inline: there is
// nothing to answer from.
func Refresh(specsDir, dbPath string, background bool) (Freshness, error) {
	var f Freshness
	if _, err := os.Stat(dbPath); err != nil {
		if !os.IsNotExist(err) {
			return f, fmt.Errorf("cannot stat index.db: %w", err)
		}
		if _, err := BuildIndex(specsDir, dbPath, false); err != nil {
			return f, err
		}
		f.Rebuilt = true
		return f, nil
	}

	// The database exists: a schema-version mismatch requires a force rebuild.
	v, err := readSchemaVersion(dbPath)
	if err != nil {
		return f, err
	}
	if v != schemaVersion {
		if _, err := BuildIndex(specsDir, dbPath, true); err != nil {
			return f, err
		}
		f.Rebuilt = true
		return f, nil
	}

	specsStale, err := specsNewer(specsDir, dbPath)
	if err != nil {
		return f, err
	}
	if !specsStale {
		specsStale, err = observationRefsChanged(dbPath)
		if err != nil {
			return f, err
		}
	}
	if specsStale {
		if _, err := build(specsDir, dbPath, false, true, false); err != nil {
			return f, err
		}
		f.Rebuilt = true
	}

	root := filepath.Dir(specsDir)
	c, err := checkDocs(root, dbPath)
	if err != nil {
		return f, err
	}
	if c.count() == 0 {
		return f, nil
	}
	if background {
		lock := LockPath(root)
		if lockHeld(lock) {
			f.Pending = c.count()
			return f, nil
		}
		// A background start that fails falls back to the inline refresh.
		if c.count() > InlineLimit && startBackground(root) == nil {
			f.Pending = c.count()
			return f, nil
		}
	}
	if _, _, err := RefreshDocs(specsDir, dbPath); err != nil {
		return f, err
	}
	f.Rebuilt = true
	return f, nil
}

// observationRefsChanged reports whether the observation-relevant refs
// differ from the fingerprint recorded at the last build.
func observationRefsChanged(dbPath string) (bool, error) {
	current, err := observation.RefsFingerprint()
	if err != nil {
		return false, fmt.Errorf("cannot fingerprint observation refs: %w", err)
	}

	db, err := openRead(dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var stored string
	err = db.QueryRow("SELECT value FROM index_meta WHERE key = ?", observationRefsKey).Scan(&stored)
	if err == sql.ErrNoRows {
		return true, nil // no fingerprint recorded → treat as changed
	}
	if err != nil {
		return false, fmt.Errorf("cannot read observation refs fingerprint: %w", err)
	}
	return stored != current, nil
}

// IsStale reports whether the index at dbPath is absent, older than a file
// or directory under specsDir (a directory counts, so a deleted spec file
// makes the index stale), or built from document files other than the ones
// on disk. Returns (true, nil) when the index should be rebuilt, (false, nil)
// when it is fresh, and (false, err) on unexpected I/O errors.
func IsStale(specsDir, dbPath string) (bool, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return true, nil // absent → stale
		}
		return false, fmt.Errorf("cannot stat index.db: %w", err)
	}
	stale, err := specsNewer(specsDir, dbPath)
	if err != nil || stale {
		return stale, err
	}
	c, err := checkDocs(filepath.Dir(specsDir), dbPath)
	return c.count() > 0, err
}

// specsNewer reports whether a file or directory under specsDir is newer than
// the index.
func specsNewer(specsDir, dbPath string) (bool, error) {
	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		return false, fmt.Errorf("cannot stat index.db: %w", err)
	}
	var newest time.Time
	walkErr := filepath.WalkDir(specsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil && info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	if walkErr != nil {
		return false, fmt.Errorf("cannot walk specs directory: %w", walkErr)
	}
	return newest.After(dbInfo.ModTime()), nil
}

// EnsureGitignored adds .spekk/index.db (or .spekk/) to .gitignore at repoRoot
// if not already present. Creates .gitignore if it doesn't exist.
func EnsureGitignored(repoRoot string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	entry := ".spekk/index.db"

	// Read existing .gitignore if present.
	var existing string
	raw, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read .gitignore: %w", err)
	}
	existing = string(raw)

	// Check if already covered (handles .spekk/ or .spekk/index.db).
	for _, line := range strings.Split(existing, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".spekk/" || trimmed == ".spekk" || trimmed == entry {
			return nil // already covered
		}
	}

	// Append the entry.
	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open .gitignore: %w", err)
	}
	defer f.Close()

	// Add a newline before the entry if the file is non-empty and doesn't end with one.
	if len(existing) > 0 && !strings.HasSuffix(existing, "\n") {
		if _, err = fmt.Fprintln(f); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(f, entry)
	return err
}
