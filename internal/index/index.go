// Package index builds and queries a SQLite index of the spec tree.
// The index is stored at .spekk/index.db at the repo root (adjacent to specs/).
// It is a derived artifact — the Markdown files remain the source of truth.
package index

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

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
`

const dropTables = `
DROP TABLE IF EXISTS depends_on;
DROP TABLE IF EXISTS assertions;
DROP TABLE IF EXISTS specs;
`

// DBPath returns the canonical path for the index database given the repo root.
func DBPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".spekk", "index.db")
}

// BuildIndex creates or updates the SQLite index at dbPath from specsDir.
// If force is true all tables are dropped and recreated from scratch.
// Returns (specCount, assertionCount, err).
func BuildIndex(specsDir, dbPath string, force bool) (int, int, error) {
	// Ensure the .spekk directory exists.
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return 0, 0, fmt.Errorf("cannot create .spekk directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot open index.db: %w", err)
	}
	defer db.Close()

	if force {
		if _, err = db.Exec(dropTables); err != nil {
			return 0, 0, fmt.Errorf("cannot drop tables: %w", err)
		}
	}

	if _, err = db.Exec(createSchema); err != nil {
		return 0, 0, fmt.Errorf("cannot create schema: %w", err)
	}

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot parse specs: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Clear existing rows for idempotency (delete-all + re-insert).
	for _, table := range []string{"depends_on", "assertions", "specs"} {
		if _, err = tx.Exec("DELETE FROM " + table); err != nil {
			return 0, 0, fmt.Errorf("cannot clear %s: %w", table, err)
		}
	}

	// Insert specs.
	for _, s := range result.Specs {
		if _, err = tx.Exec(
			`INSERT INTO specs (id, title, status, priority, branch, file) VALUES (?, ?, ?, ?, ?, ?)`,
			s.ID, s.Title, s.Status, s.Priority, s.Branch, s.File,
		); err != nil {
			return 0, 0, fmt.Errorf("cannot insert spec %q: %w", s.ID, err)
		}
	}

	// Insert assertions and depends_on edges.
	for _, a := range result.Assertions {
		if _, err = tx.Exec(
			`INSERT INTO assertions (id, parent_id, title, status, priority, branch, file) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			a.ID, a.Parent, a.Title, a.Status, a.Priority, a.Branch, a.File,
		); err != nil {
			return 0, 0, fmt.Errorf("cannot insert assertion %q: %w", a.ID, err)
		}

		if a.DependsOn != "" {
			if _, err = tx.Exec(
				`INSERT INTO depends_on (assertion_id, depends_on_id) VALUES (?, ?)`,
				a.ID, a.DependsOn,
			); err != nil {
				return 0, 0, fmt.Errorf("cannot insert depends_on edge for %q: %w", a.ID, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("cannot commit transaction: %w", err)
	}

	return len(result.Specs), len(result.Assertions), nil
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
