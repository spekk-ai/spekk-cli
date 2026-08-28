// Package observation defines the observation file format and the
// branch-based observation lifecycle helpers
// (specs/observation-lifecycle/).
//
// An observation is a markdown file under observations/ whose YAML
// frontmatter carries the full lifecycle record. Observations are born on
// dedicated observer/<slug> branches — never committed to main by the
// observer — and the set of observer/* branches IS the state machine:
//
//	git fact                                  lifecycle state
//	----------------------------------------  -------------------------------
//	observer/<slug> visible (local or origin) announced / pending
//	branch merged to main                     resolved (status: resolved on
//	                                          main, remedy in the same merge)
//	branch kept, its PR closed                parked — still in the dedup
//	                                          union, never re-announced
//	branch deleted                            forgotten — the union forgets
//	                                          it; persistent drift is
//	                                          legitimately re-found
//
// Every lifecycle question is answerable from local git state after a
// `git fetch` — the only remote read any observer tooling performs. No code
// path consults a forge API (gh, GitHub REST/GraphQL, ...) for state: PR
// open/closed status is deliberately invisible, so parked and pending are
// distinguished by human convention, not by tooling. Dedup treats both
// identically and announce idempotency is keyed solely off the `announced:`
// frontmatter marker.
package observation

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// Dir is the repo-root-relative directory that holds observation files.
const Dir = "observations"

// The closed set of observation types.
const (
	TypeCodeSpecMisalignment = "code_spec_misalignment"
	TypeOutdatedSpecs        = "outdated_specs"
)

// The closed set of observation severities.
const (
	SeverityHigh   = "high"
	SeverityMedium = "medium"
	SeverityLow    = "low"
)

// The closed set of observation statuses.
const (
	StatusOpen      = "open"
	StatusResolved  = "resolved"
	StatusDismissed = "dismissed"
)

// Observation is one parsed observation file. Announced is the empty string
// when the frontmatter lacks the `announced:` field — absence is meaningful:
// it is the "not yet announced" marker, and no separate ledger exists.
type Observation struct {
	Slug      string   // kebab-case; matches the observer/<slug> branch name
	Type      string   // code_spec_misalignment | outdated_specs
	Severity  string   // high | medium | low
	Status    string   // open | resolved | dismissed
	Created   string   // ISO 8601 timestamp
	Announced string   // ISO 8601 timestamp; "" until a conversation opened
	PR        string   // optional PR URL
	Affected  []string // evidence paths; required, non-empty
	Title     string   // first H1 of the markdown body ("Untitled" if none)
	Body      string   // markdown body after the closing frontmatter delimiter
	File      string   // repo-root-relative path the observation was read from
	Ref       string   // git ref the observation was read from ("" = working tree)

	// Fields holds every frontmatter key outside knownKeys, with its values
	// already split into items. The index writes them to frontmatter_fields
	// under owner_type 'observation', so provenance an observation carries
	// beyond the lifecycle set (which skill found it, which run) is
	// reachable from spekk query rather than only from the file.
	Fields map[string][]string
}

// knownKeys are the frontmatter keys the lifecycle schema defines and Parse
// maps onto Observation. Every other key is a custom field, preserved on
// Fields. `affected` is known: it is the evidence gate and the dedup key,
// observation_files is its table, and a second copy of it under a custom
// name would invite a query to read the gate as a tag.
var knownKeys = map[string]bool{
	"slug":      true,
	"type":      true,
	"severity":  true,
	"status":    true,
	"created":   true,
	"announced": true,
	"pr":        true,
	"affected":  true,
}

var (
	slugPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	timestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`)

	validTypes      = map[string]bool{TypeCodeSpecMisalignment: true, TypeOutdatedSpecs: true}
	validSeverities = map[string]bool{SeverityHigh: true, SeverityMedium: true, SeverityLow: true}
	validStatuses   = map[string]bool{StatusOpen: true, StatusResolved: true, StatusDismissed: true}
)

// SeverityRank orders severities for ranking: high (0) before medium (1)
// before low (2). Unknown severities sort last.
func SeverityRank(severity string) int {
	switch severity {
	case SeverityHigh:
		return 0
	case SeverityMedium:
		return 1
	case SeverityLow:
		return 2
	default:
		return 3
	}
}

// Parse parses and validates one observation file. path is used in
// diagnostics and recorded as the observation's File.
//
// The frontmatter schema is closed for the known fields and open for growth:
// a field outside the documented set never breaks parsing. Such a field is
// preserved on Fields and indexed, so the format can grow without flag days.
// Validation enforces:
//
//   - slug: required, kebab-case
//   - type: one of code_spec_misalignment | outdated_specs
//   - severity: one of high | medium | low
//   - status: one of open | resolved | dismissed
//   - created: required ISO 8601 timestamp
//   - announced: absent means "not yet announced"; when the key is present
//     its value must be a non-empty ISO 8601 timestamp — a present-but-empty
//     value is invalid rather than either state
//   - affected: required, non-empty — the evidence gate: no evidence, no
//     observation
func Parse(path, content string) (*Observation, error) {
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("observation %s: %w", path, err)
	}

	// Custom fields come from the parser package rather than from the
	// scanner above, so a spec, an assertion, and an observation split a
	// multi-value key the same way and exclude a comment, a nested child,
	// and a block scalar the same way. One rule, not one copy of it each.
	//
	// Both scanners fail on the same two conditions — no opening `---`, no
	// closing one — so today this error cannot fire after the parse above
	// succeeded. It stays because that agreement is not enforced anywhere,
	// and a silent nil map would hide the day it ends.
	fields, err := parser.CustomFields(content, knownKeys)
	if err != nil {
		return nil, fmt.Errorf("observation %s: %w", path, err)
	}

	o := &Observation{
		Slug:      fm.scalars["slug"],
		Type:      fm.scalars["type"],
		Severity:  fm.scalars["severity"],
		Status:    fm.scalars["status"],
		Created:   fm.scalars["created"],
		Announced: fm.scalars["announced"],
		PR:        fm.scalars["pr"],
		Affected:  fm.lists["affected"],
		Title:     extractTitle(body),
		Body:      body,
		File:      path,
		Fields:    fields,
	}

	if o.Slug == "" {
		return nil, fmt.Errorf("observation %s: missing required field 'slug'", path)
	}
	if !slugPattern.MatchString(o.Slug) {
		return nil, fmt.Errorf("observation %s: field 'slug' must be kebab-case, got %q", path, o.Slug)
	}
	if !validTypes[o.Type] {
		return nil, fmt.Errorf("observation %s: field 'type' must be %s or %s, got %q",
			path, TypeCodeSpecMisalignment, TypeOutdatedSpecs, o.Type)
	}
	if !validSeverities[o.Severity] {
		return nil, fmt.Errorf("observation %s: field 'severity' must be high, medium, or low, got %q", path, o.Severity)
	}
	if !validStatuses[o.Status] {
		return nil, fmt.Errorf("observation %s: field 'status' must be open, resolved, or dismissed, got %q", path, o.Status)
	}
	if !timestampPattern.MatchString(o.Created) {
		return nil, fmt.Errorf("observation %s: field 'created' must be an ISO 8601 timestamp (YYYY-MM-DDTHH:MM:SSZ), got %q", path, o.Created)
	}
	// `announced` is a timestamp, not a boolean: its absence encodes
	// "unannounced". A present-but-empty value is neither state — reject it.
	if fm.present["announced"] && !timestampPattern.MatchString(o.Announced) {
		return nil, fmt.Errorf("observation %s: field 'announced' must be absent or an ISO 8601 timestamp, got %q", path, o.Announced)
	}
	// Evidence gate: no evidence, no observation.
	if len(o.Affected) == 0 {
		return nil, fmt.Errorf("observation %s: field 'affected' must list at least one evidence path (no evidence, no observation)", path)
	}

	return o, nil
}

// frontmatter holds one parsed YAML frontmatter block: scalar fields, list
// fields, and a presence map so callers can distinguish an absent key from a
// present-but-empty one.
type frontmatter struct {
	scalars map[string]string
	lists   map[string][]string
	present map[string]bool
}

// parseFrontmatter parses the leading `---` YAML block of an observation
// file. It supports scalar `key: value` fields and block lists
//
//	key:
//	  - item
//
// which is the subset the observation schema needs. It reads the known keys
// only. A key outside the known set is read by parser.CustomFields instead,
// off the same content, so an unknown key never breaks parsing here.
func parseFrontmatter(content string) (*frontmatter, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	if len(lines) == 0 || lines[0] != "---" {
		return nil, "", fmt.Errorf("file must start with --- YAML frontmatter delimiter")
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, "", fmt.Errorf("missing closing --- delimiter for YAML frontmatter")
	}

	fm := &frontmatter{
		scalars: map[string]string{},
		lists:   map[string][]string{},
		present: map[string]bool{},
	}

	currentList := "" // key of the list block being collected, if any
	for _, line := range lines[1:end] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Strip inline comments that start a token (` # ...`). A '#' inside a
		// value without a leading space is kept (e.g. URLs with fragments).
		if idx := strings.Index(trimmed, " #"); idx >= 0 {
			trimmed = strings.TrimSpace(trimmed[:idx])
			if trimmed == "" {
				continue
			}
		}

		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			if currentList == "" {
				continue // stray list item with no owning key; ignore
			}
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			item = unquote(item)
			if item != "" {
				fm.lists[currentList] = append(fm.lists[currentList], item)
			}
			continue
		}

		colon := strings.Index(trimmed, ":")
		if colon == -1 {
			continue // not a key line; ignore
		}
		key := strings.TrimSpace(trimmed[:colon])
		value := strings.TrimSpace(trimmed[colon+1:])
		fm.present[key] = true

		if value == "" {
			// Opens a block list (or is a present-but-empty scalar — the
			// presence map lets Parse tell the difference where it matters).
			currentList = key
			fm.scalars[key] = ""
			continue
		}
		currentList = ""
		fm.scalars[key] = unquote(value)
	}

	return fm, strings.Join(lines[end+1:], "\n"), nil
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

// extractTitle returns the first H1 heading of the markdown body, or
// "Untitled" when there is none.
func extractTitle(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return "Untitled"
}

// MarkAnnounced returns content with `announced: <ts>` set in the
// frontmatter. The line is inserted directly after the `created:` line so
// the flip is a one-line, deterministic diff; when the field is already
// present its value is replaced in place. ts must be an ISO 8601 timestamp.
func MarkAnnounced(content, ts string) (string, error) {
	if !timestampPattern.MatchString(ts) {
		return "", fmt.Errorf("announced timestamp must be ISO 8601 (YYYY-MM-DDTHH:MM:SSZ), got %q", ts)
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return "", fmt.Errorf("content has no frontmatter to mark")
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return "", fmt.Errorf("content has no closing frontmatter delimiter")
	}

	newLine := "announced: " + ts

	// Replace in place when present.
	for i := 1; i < end; i++ {
		key := strings.TrimSpace(strings.SplitN(lines[i], ":", 2)[0])
		if key == "announced" {
			lines[i] = newLine
			return strings.Join(lines, "\n"), nil
		}
	}

	// Insert after created:, else before the closing delimiter.
	insertAt := end
	for i := 1; i < end; i++ {
		key := strings.TrimSpace(strings.SplitN(lines[i], ":", 2)[0])
		if key == "created" {
			insertAt = i + 1
			break
		}
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, newLine)
	out = append(out, lines[insertAt:]...)
	return strings.Join(out, "\n"), nil
}

// ResolveSlug returns the slug a (possibly re-found) finding should use.
// The decision on slug reuse after "forgotten": reuse the plain slug; append
// a -YYYYMMDD suffix only when the plain slug collides with an observation
// that is already on main (whose slug is therefore taken by history).
func ResolveSlug(plain string, onMain func(slug string) bool, now time.Time) string {
	if onMain == nil || !onMain(plain) {
		return plain
	}
	return plain + "-" + now.UTC().Format("20060102")
}

// SortForDigest orders observations for ranked views: severity (high before
// medium before low), then oldest created first, then slug as the stable
// tie-break so identical timestamps still order deterministically.
func SortForDigest(obs []*Observation) {
	sort.SliceStable(obs, func(i, j int) bool {
		if a, b := SeverityRank(obs[i].Severity), SeverityRank(obs[j].Severity); a != b {
			return a < b
		}
		if obs[i].Created != obs[j].Created {
			return obs[i].Created < obs[j].Created
		}
		return obs[i].Slug < obs[j].Slug
	})
}
