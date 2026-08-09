// Package dontflag reads and applies .spekk/dont-flag.yaml, the
// human-gated suppression file (specs/observer-dont-flag/).
//
// The file is committed on main and changes via reviewed PRs, so every
// suppression carries a match pattern, a stated reason, and a named author.
// It is consulted at scan time, before an observation is born: matching
// drift produces nothing — no observation file, no observer/<slug> branch,
// no index row, no announcement. The observer itself never writes this
// file; it only reads it, and there is no flag, environment variable, or
// prompt instruction that suppresses drift while bypassing it.
package dontflag

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/crossbranch"
	"github.com/spekk-ai/spekk-cli/internal/observation"
)

// FilePath is the repo-root-relative location of the suppression file. It
// lives in .spekk/ alongside — but unlike — the gitignored index.db: this
// file is tracked, because suppressions are repo state.
const FilePath = ".spekk/dont-flag.yaml"

// untilLayout is the date-only format of the optional expiry field.
const untilLayout = "2006-01-02"

// Entry is one suppression. The schema is exactly: match (required; a path
// glob matched against a would-be observation's affected paths, or a slug
// pattern matched against its slug), reason (required), by (required), and
// until (optional date; absent means permanent).
type Entry struct {
	Match  string
	Reason string
	By     string
	Until  string // YYYY-MM-DD; "" = permanent
}

// ActiveAt reports whether the entry still suppresses at time now. An
// `until` date is interpreted as end-of-day UTC: the entry suppresses
// through the whole named day and expires at the following UTC midnight.
// An expired entry suppresses nothing.
func (e Entry) ActiveAt(now time.Time) bool {
	if e.Until == "" {
		return true
	}
	day, err := time.ParseInLocation(untilLayout, e.Until, time.UTC)
	if err != nil {
		return false // Parse validates this; a bad date never suppresses
	}
	return now.UTC().Before(day.AddDate(0, 0, 1))
}

// Matches reports whether the entry matches a would-be observation: its
// pattern against the slug the observation would be given, or against any
// of its evidence paths. The match target for path globs is the drift's
// evidence (what would become `affected`), not every file the scan read.
// A path matches if the pattern matches it as written OR after
// observation.NormalizePath. Candidate paths come from an agent, so the same
// file arrives spelled several ways, and a `:42` suffix or a `./` prefix must
// not defeat a suppression. Suppression is the only gate here that dedup
// cannot back up: suppressed drift never becomes an observation, so nothing
// lands on a branch to cover it next time.
//
// The pattern itself is NEVER rewritten, and that is the whole design. It is
// a glob, not a path, so a path function has no business cleaning it:
// path.Clean resolves `..`, which turns the scoped-looking `docs/../**` into
// `**` and suppresses the entire repository; the location-suffix strip turns
// `**:1` into `**` for the same result. Both read as narrow to a reviewer,
// and this file is trusted precisely because a person read it.
//
// Trying the pattern against both spellings gets what normalizing was meant
// to get, and cannot do either of those things. It also removes any need for
// the normalizer to be idempotent — comparing a normalized pattern with a
// normalized path was only ever sound if it were.
//
// The slug is not a path, so it is compared as written.
func (e Entry) Matches(slug string, affected []string) bool {
	if globMatch(e.Match, slug) {
		return true
	}
	for _, p := range affected {
		if globMatch(e.Match, p) || globMatch(e.Match, observation.NormalizePath(p)) {
			return true
		}
	}
	return false
}

// Suppressed returns the first active entry matching the would-be
// observation, or nil when nothing suppresses it.
func Suppressed(entries []Entry, slug string, affected []string, now time.Time) *Entry {
	for i := range entries {
		if entries[i].ActiveAt(now) && entries[i].Matches(slug, affected) {
			return &entries[i]
		}
	}
	return nil
}

// LoadFromMain reads the suppression file as it exists on main (falling
// back to master), via a git object read — the working tree copy is not
// consulted, so a suppression takes effect only once it is committed on the
// main branch. A missing file (or no visible main branch) means "no
// suppressions" and is not an error; a malformed file is an error, because
// silently treating a broken safety file as empty would cause a re-flag
// flood.
func LoadFromMain() ([]Entry, error) {
	mainRef, err := observation.MainRef()
	if err != nil {
		return nil, nil // no main branch visible → nothing to suppress
	}
	content, err := crossbranch.FileAtRef(mainRef, FilePath)
	if err != nil {
		if errors.Is(err, crossbranch.ErrFileAbsent) {
			return nil, nil
		}
		return nil, err
	}
	return Parse(content)
}

// knownKeys is the closed set of entry fields. An unknown key is an error
// rather than being ignored: in a safety-relevant file, a typo like
// "untill" silently ignored would turn an intended-to-expire suppression
// into a permanent one.
var knownKeys = map[string]bool{"match": true, "reason": true, "by": true, "until": true}

// Parse parses the suppression file: a YAML list of flat entries. Errors
// name the offending entry (1-based position and its match pattern when one
// was read), so a broken file is fixable from the message alone.
func Parse(content string) ([]Entry, error) {
	var entries []Entry
	var current *Entry

	finish := func() error {
		if current == nil {
			return nil
		}
		if err := validate(*current, len(entries)+1); err != nil {
			return err
		}
		entries = append(entries, *current)
		current = nil
		return nil
	}

	for lineNo, raw := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip trailing comments introduced by " #".
		if i := strings.Index(line, " #"); i >= 0 {
			line = strings.TrimSpace(line[:i])
			if line == "" {
				continue
			}
		}

		isItem := strings.HasPrefix(line, "- ")
		if isItem {
			if err := finish(); err != nil {
				return nil, err
			}
			current = &Entry{}
			line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
		}
		if current == nil {
			return nil, fmt.Errorf("%s: line %d is outside any entry: %q", FilePath, lineNo+1, line)
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("%s: entry %d: line %d is not a key: value pair: %q", FilePath, len(entries)+1, lineNo+1, line)
		}
		key = strings.TrimSpace(key)
		value = unquote(strings.TrimSpace(value))
		if !knownKeys[key] {
			return nil, fmt.Errorf("%s: entry %d: unknown field %q (known: match, reason, by, until)", FilePath, len(entries)+1, key)
		}
		switch key {
		case "match":
			current.Match = value
		case "reason":
			current.Reason = value
		case "by":
			current.By = value
		case "until":
			current.Until = value
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}
	return entries, nil
}

// validate enforces the entry schema; pos is the 1-based entry position for
// error messages.
func validate(e Entry, pos int) error {
	name := e.Match
	if name == "" {
		name = fmt.Sprintf("entry %d", pos)
	} else {
		name = fmt.Sprintf("entry %d (match: %q)", pos, e.Match)
	}
	if e.Match == "" {
		return fmt.Errorf("%s: %s: missing required field 'match'", FilePath, name)
	}
	if e.Reason == "" {
		return fmt.Errorf("%s: %s: missing required field 'reason'", FilePath, name)
	}
	if e.By == "" {
		return fmt.Errorf("%s: %s: missing required field 'by'", FilePath, name)
	}
	if e.Until != "" {
		if _, err := time.ParseInLocation(untilLayout, e.Until, time.UTC); err != nil {
			return fmt.Errorf("%s: %s: field 'until' must be a YYYY-MM-DD date, got %q", FilePath, name, e.Until)
		}
	}
	return nil
}

// unquote strips one level of surrounding double or single quotes.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// globMatch matches a slash-separated glob pattern that supports "**" as a
// full segment (matching any number of path segments, including zero);
// within a segment, "*", "?", and "[...]" follow path.Match semantics. A
// malformed pattern matches nothing.
func globMatch(pattern, s string) bool {
	return segmentsMatch(strings.Split(pattern, "/"), strings.Split(s, "/"))
}

func segmentsMatch(pattern, parts []string) bool {
	if len(pattern) == 0 {
		return len(parts) == 0
	}
	if pattern[0] == "**" {
		// "**" absorbs zero or more leading segments.
		for skip := 0; skip <= len(parts); skip++ {
			if segmentsMatch(pattern[1:], parts[skip:]) {
				return true
			}
		}
		return false
	}
	if len(parts) == 0 {
		return false
	}
	ok, err := path.Match(pattern[0], parts[0])
	if err != nil || !ok {
		return false
	}
	return segmentsMatch(pattern[1:], parts[1:])
}
