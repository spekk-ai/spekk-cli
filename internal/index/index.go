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
`

// dropAllTables drops every table in the database, enumerated from
// sqlite_master rather than a hardcoded list. A binary can only name the
// tables of its own generation, and a hardcoded list is exactly how a
// version-skewed force rebuild leaves a future table's stale rows in place
// (an old binary rebuilding a newer database refreshed every table it knew
// and silently kept the rest). Dropping whatever exists makes every future
// schema bump safe against this binary going stale.
func dropAllTables(db *sql.DB) error {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
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
// frontmatter_fields; 4 adds observation rows to frontmatter_fields. Version
// 4 changes no column — the bump is what makes an index that version 3 built
// rebuild, instead of answering a custom-key query from rows that predate
// the change. Bump on a change to the rows a build writes, not only on a
// change to the shape it writes them into.
const schemaVersion = 4

// observationRefsKey is the index_meta key holding the observation
// RefsFingerprint the last build saw.
const observationRefsKey = "observation_refs"

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
	Warnings     []string
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

// BuildIndex creates or updates the SQLite index at dbPath from specsDir and
// from the visible git refs (observations). If force is true all tables are
// dropped and recreated from scratch.
func BuildIndex(specsDir, dbPath string, force bool) (Stats, error) {
	var stats Stats

	// Ensure the .spekk directory exists.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return stats, fmt.Errorf("cannot create .spekk directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return stats, fmt.Errorf("cannot open index.db: %w", err)
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

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		return stats, fmt.Errorf("%w: %w", ErrSpecsUnparseable, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return stats, fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Clear existing rows for idempotency (delete-all + re-insert).
	for _, table := range []string{"depends_on", "frontmatter_fields", "assertions", "specs", "observation_files", "observations", "index_meta"} {
		if _, err = tx.Exec("DELETE FROM " + table); err != nil {
			return stats, fmt.Errorf("cannot clear %s: %w", table, err)
		}
	}

	// Insert specs.
	for _, s := range result.Specs {
		if _, err = tx.Exec(
			`INSERT INTO specs (id, title, status, priority, branch, file) VALUES (?, ?, ?, ?, ?, ?)`,
			s.ID, s.Title, s.Status, s.Priority, s.Branch, s.File,
		); err != nil {
			return stats, fmt.Errorf("cannot insert spec %q: %w", s.ID, err)
		}
		if err = insertFrontmatterFields(tx, "spec", s.ID, s.Fields); err != nil {
			return stats, err
		}
	}

	// Insert assertions and depends_on edges.
	for _, a := range result.Assertions {
		if _, err = tx.Exec(
			`INSERT INTO assertions (id, parent_id, title, status, priority, branch, file) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			a.ID, a.Parent, a.Title, a.Status, a.Priority, a.Branch, a.File,
		); err != nil {
			return stats, fmt.Errorf("cannot insert assertion %q: %w", a.ID, err)
		}

		if a.DependsOn != "" {
			if _, err = tx.Exec(
				`INSERT INTO depends_on (assertion_id, depends_on_id) VALUES (?, ?)`,
				a.ID, a.DependsOn,
			); err != nil {
				return stats, fmt.Errorf("cannot insert depends_on edge for %q: %w", a.ID, err)
			}
		}

		if err = insertFrontmatterFields(tx, "assertion", a.ID, a.Fields); err != nil {
			return stats, err
		}
	}

	// Observation pass: read the cross-branch union from git refs.
	obsCount, warnings, err := indexObservations(tx)
	if err != nil {
		return stats, err
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

	stats.Specs = len(result.Specs)
	stats.Assertions = len(result.Assertions)
	stats.Observations = obsCount
	stats.Warnings = warnings
	return stats, nil
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
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, fmt.Errorf("cannot open index.db: %w", err)
	}
	defer db.Close()

	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return 0, fmt.Errorf("cannot read schema version: %w", err)
	}
	return v, nil
}

// EnsureFresh rebuilds the index at dbPath from specsDir when it is absent,
// older than the specs (mtime), stamped with a schema version other than the
// current one, or built against a different set of observation refs (an
// observer branch was fetched, created, or deleted since the last build). It
// reports whether a rebuild happened. A schema-version mismatch forces a
// drop-and-recreate so a changed schema never leaves old-shaped tables in
// place; a plain content-staleness rebuild does not need to force.
//
// This is the single freshness gate every index reader should call, so the
// "when do we rebuild" rule lives in exactly one place.
func EnsureFresh(specsDir, dbPath string) (bool, error) {
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			if _, buildErr := BuildIndex(specsDir, dbPath, false); buildErr != nil {
				return false, buildErr
			}
			return true, nil
		}
		return false, fmt.Errorf("cannot stat index.db: %w", err)
	}

	// The database exists: a schema-version mismatch requires a force rebuild.
	v, err := readSchemaVersion(dbPath)
	if err != nil {
		return false, err
	}
	if v != schemaVersion {
		if _, buildErr := BuildIndex(specsDir, dbPath, true); buildErr != nil {
			return false, buildErr
		}
		return true, nil
	}

	// Schema is current: rebuild if the specs are newer than the index, or
	// the observation refs changed since the last build.
	stale, err := IsStale(specsDir, dbPath)
	if err != nil {
		return false, err
	}
	if !stale {
		stale, err = observationRefsChanged(dbPath)
		if err != nil {
			return false, err
		}
	}
	if stale {
		if _, buildErr := BuildIndex(specsDir, dbPath, false); buildErr != nil {
			return false, buildErr
		}
		return true, nil
	}

	return false, nil
}

// observationRefsChanged reports whether the observation-relevant refs
// differ from the fingerprint recorded at the last build.
func observationRefsChanged(dbPath string) (bool, error) {
	current, err := observation.RefsFingerprint()
	if err != nil {
		return false, fmt.Errorf("cannot fingerprint observation refs: %w", err)
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return false, fmt.Errorf("cannot open index.db: %w", err)
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

// IsStale reports whether the index at dbPath is absent or older than the
// most recently modified file under specsDir. Returns (true, nil) when the
// index should be rebuilt, (false, nil) when it is fresh, and (false, err)
// on unexpected I/O errors.
func IsStale(specsDir, dbPath string) (bool, error) {
	dbInfo, err := os.Stat(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // absent → stale
		}
		return false, fmt.Errorf("cannot stat index.db: %w", err)
	}
	dbMtime := dbInfo.ModTime()

	// Walk specs dir to find the most recent mtime.
	var newestSpec time.Time
	walkErr := filepath.WalkDir(specsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(newestSpec) {
			newestSpec = info.ModTime()
		}
		return nil
	})
	if walkErr != nil {
		return false, fmt.Errorf("cannot walk specs directory: %w", walkErr)
	}

	return newestSpec.After(dbMtime), nil
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
