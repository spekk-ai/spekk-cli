// Package observer implements the deterministic, Go-side observer steps —
// today the `spekk observer announce` flow (specs/observer-announce/).
//
// Announce is the direct fix for a production silent failure: announce
// eligibility is computed from declarative state (branches + frontmatter,
// via the SQLite index), the message shape lives in code rather than a
// prompt, and every failure leaves an exit code and a log line. The only
// remote operations in the whole flow are `git fetch` at the start and the
// final `git push` of the announced: flip — never a forge API call.
package observer

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/spekk-ai/spekk-cli/internal/observation"
)

// Candidate is one observation row considered for announcement, as loaded
// from the index.
type Candidate struct {
	Slug     string
	Ref      string
	Severity string
	Created  string
	PR       string
	Title    string
	File     string
	Body     string // loaded lazily from the ref, not from the index
	Affected []string
	OnMain   bool // an observation with this slug exists on main
}

// SelectCandidates filters and orders announce candidates. An observation is
// eligible iff all of:
//
//   - status: open (rows are pre-filtered by the SQL query)
//   - announced is absent
//   - severity is high or medium — low NEVER announces, regardless of age
//     or queue emptiness
//   - it lives on a visible observer/<slug> branch (not only on main)
//   - the evidence gate holds: at least one affected path
//   - no observation with the same slug is on main (present on main means
//     effectively not open — the backstop rule)
//
// Ordering: severity (high before medium), then oldest created first, then
// slug as the stable tie-break, so the same repo state always yields the
// same choice.
func SelectCandidates(rows []Candidate) []Candidate {
	seen := map[string]bool{}
	var out []Candidate
	for _, c := range rows {
		if c.OnMain || seen[c.Slug] {
			continue
		}
		if c.Severity != observation.SeverityHigh && c.Severity != observation.SeverityMedium {
			continue
		}
		if !isObserverRef(c.Ref) {
			continue
		}
		// Evidence gate, enforced in code even though parsing and indexing
		// already reject evidence-free observations: no affected paths, no
		// announcement.
		if len(c.Affected) == 0 {
			continue
		}
		seen[c.Slug] = true
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if a, b := observation.SeverityRank(out[i].Severity), observation.SeverityRank(out[j].Severity); a != b {
			return a < b
		}
		if out[i].Created != out[j].Created {
			return out[i].Created < out[j].Created
		}
		return out[i].Slug < out[j].Slug
	})
	return out
}

// isObserverRef reports whether a fully-qualified ref names an observer/*
// branch, local or remote-tracking.
func isObserverRef(ref string) bool {
	if rest, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
		return strings.HasPrefix(rest, observation.BranchPrefix)
	}
	if rest, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			return strings.HasPrefix(rest[i+1:], observation.BranchPrefix)
		}
	}
	return false
}

// loadCandidates reads the unannounced open observations from the index at
// dbPath, with their evidence paths and on-main marker attached. The query
// side is deliberately simple; eligibility and ordering live in
// SelectCandidates, one tested pure function.
func loadCandidates(dbPath string) ([]Candidate, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("cannot open index.db: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT slug, ref, severity, created, COALESCE(pr, ''), title, file
		FROM observations
		WHERE status = ? AND announced IS NULL`, observation.StatusOpen)
	if err != nil {
		return nil, fmt.Errorf("cannot query observations: %w", err)
	}
	defer rows.Close()

	var out []Candidate
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.Slug, &c.Ref, &c.Severity, &c.Created, &c.PR, &c.Title, &c.File); err != nil {
			return nil, fmt.Errorf("cannot scan observation row: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot read observation rows: %w", err)
	}

	mainSlugs := map[string]bool{}
	mainRows, err := db.Query(`SELECT DISTINCT slug FROM observations
		WHERE ref LIKE 'refs/heads/main' OR ref LIKE 'refs/heads/master'
		   OR ref LIKE 'refs/remotes/%/main' OR ref LIKE 'refs/remotes/%/master'`)
	if err != nil {
		return nil, fmt.Errorf("cannot query main observations: %w", err)
	}
	defer mainRows.Close()
	for mainRows.Next() {
		var slug string
		if err := mainRows.Scan(&slug); err != nil {
			return nil, fmt.Errorf("cannot scan main slug: %w", err)
		}
		mainSlugs[slug] = true
	}
	if err := mainRows.Err(); err != nil {
		return nil, fmt.Errorf("cannot read main slugs: %w", err)
	}

	for i := range out {
		out[i].OnMain = mainSlugs[out[i].Slug]
		paths, err := loadAffected(db, out[i].Slug, out[i].Ref)
		if err != nil {
			return nil, err
		}
		out[i].Affected = paths
	}
	return out, nil
}

// loadAffected returns the evidence paths for one (slug, ref) pair.
func loadAffected(db *sql.DB, slug, ref string) ([]string, error) {
	rows, err := db.Query(`SELECT path FROM observation_files WHERE slug = ? AND ref = ? ORDER BY path`, slug, ref)
	if err != nil {
		return nil, fmt.Errorf("cannot query observation files: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("cannot scan observation file row: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
