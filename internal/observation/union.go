package observation

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/crossbranch"
)

// Union is the cross-branch union of observations: every observation
// readable from a visible observer/* branch (local or remote-tracking) plus
// main. It is the reference set for scan-time dedup and for rendered views.
//
// The union is built via internal/crossbranch ref reads only — no branch is
// ever checked out, no forge API is ever consulted, and nothing produced by
// the current run participates. Dedup therefore never compares an artifact
// against itself: the reference set is committed observations on
// branches/main, never a digest or summary produced by the same pass.
type Union struct {
	// Observations holds one entry per (observation file, ref) pair, in
	// deterministic (ref, file) order.
	Observations []*Observation
	// Warnings names files that were skipped because they failed validation
	// (including the evidence gate). Invalid files are never silently
	// treated as valid observations.
	Warnings []string
}

// LoadUnion reads observations from every visible observer/* branch plus
// main, via git object reads. Branch discovery follows the same rules as
// internal/crossbranch (local heads + remote-tracking refs, deduplicated by
// logical name with the local head preferred), including the current branch.
// It performs no remote operations: keeping remote-tracking refs current is
// the caller's job (`git fetch`).
func LoadUnion() (*Union, error) {
	refs, err := unionRefs()
	if err != nil {
		return nil, err
	}

	u := &Union{}
	for _, ref := range refs {
		files, err := crossbranch.FilesAtRef(ref, Dir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			// Lifecycle observations live directly under observations/ as
			// <slug>.md. Subdirectories (observations/<skill-name>/...) hold
			// skill-specific advisory outputs with their own contracts and
			// are not part of the lifecycle union.
			rest, ok := strings.CutPrefix(file, Dir+"/")
			if !ok || strings.Contains(rest, "/") || !strings.HasSuffix(rest, ".md") {
				continue
			}
			content, err := crossbranch.FileAtRef(ref, file)
			if err != nil {
				return nil, err
			}
			o, err := Parse(file, content)
			if err != nil {
				u.Warnings = append(u.Warnings, fmt.Sprintf("skipped %s at %s: %v", file, ref, err))
				continue
			}
			o.Ref = ref
			u.Observations = append(u.Observations, o)
		}
	}
	return u, nil
}

// unionRefs returns the fully-qualified refs the union reads from: every
// visible observer/* branch plus main (falling back to master), in a stable
// order with main first.
func unionRefs() ([]string, error) {
	var refs []string
	if mainRef, err := MainRef(); err == nil {
		refs = append(refs, mainRef)
	}

	branches, err := crossbranch.DiscoverAllBranches(BranchGlob)
	if err != nil {
		return nil, err
	}
	for _, b := range branches {
		refs = append(refs, b.Ref)
	}
	return refs, nil
}

// MainRef resolves the ref to read main from: the local main branch when it
// exists, otherwise the remote-tracking one; master is the fallback name.
// It returns an error when no main branch is visible at all.
func MainRef() (string, error) {
	for _, name := range []string{"main", "master"} {
		branches, err := crossbranch.DiscoverAllBranches(name)
		if err != nil {
			return "", err
		}
		if len(branches) > 0 {
			return branches[0].Ref, nil
		}
	}
	return "", fmt.Errorf("observation: no main (or master) branch visible")
}

// isMainRef reports whether ref names the main (or master) branch, locally
// or on a remote.
func isMainRef(ref string) bool {
	name := ref
	name = strings.TrimPrefix(name, "refs/heads/")
	if rest, ok := strings.CutPrefix(name, "refs/remotes/"); ok {
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			name = rest[i+1:]
		}
	}
	return name == "main" || name == "master"
}

// OnMain reports whether an observation with the given slug exists on main.
// Presence on main means the finding is resolved history (or dismissed): the
// backstop rule is "observation present on main ⇒ effectively not open",
// regardless of a lagging frontmatter status.
func (u *Union) OnMain(slug string) bool {
	for _, o := range u.Observations {
		if o.Slug == slug && isMainRef(o.Ref) {
			return true
		}
	}
	return false
}

// NormalizePath reduces an affected path to the form paths are compared in:
// repository-root-relative, no location suffix.
//
// Dedup compares these strings exactly, and the strings are written by an
// agent rather than produced by the code — so the same file arrives spelled
// several ways. A run that names `internal/parser/parser.go:42` where an
// earlier run named `internal/parser/parser.go` is describing the same file,
// but the raw strings differ, so nothing covers it and the finding is filed
// again. Normalizing removes the three spellings that carry no meaning:
//
//	./internal/parser/parser.go     leading ./
//	internal/parser/parser.go:42    a line, or line:column
//	internal/parser/parser.go/      a trailing slash
//
// A directory is deliberately NOT reduced to the files under it. Prefix
// containment would let one directory-level finding swallow every
// file-level finding beneath it — a false negative that hides real drift,
// which is worse than a duplicate. Naming files rather than directories is
// the prompt's job, not this function's.
func NormalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// A trailing :line or :line:column. Only digits qualify, so a path that
	// genuinely contains a colon is left alone.
	for {
		i := strings.LastIndex(p, ":")
		if i <= 0 || i == len(p)-1 || !allDigits(p[i+1:]) {
			break
		}
		p = p[:i]
	}
	if p == "" {
		return ""
	}
	// path.Clean settles the rest: ./ prefixes however many, duplicate
	// slashes, a trailing slash, and any . or .. segment.
	return path.Clean(p)
}

// allDigits reports whether s is one or more ASCII digits.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Covers reports whether the observation covers a candidate finding: same
// type and at least one overlapping affected path, compared after
// NormalizePath. This is the scan-time dedup test — a covered finding must
// not produce a new observation or a new branch, whatever the covering
// branch's PR state (parked branches participate exactly like pending ones).
func (o *Observation) Covers(typ string, affected []string) bool {
	if o.Type != typ {
		return false
	}
	existing := make(map[string]bool, len(o.Affected))
	for _, e := range o.Affected {
		existing[NormalizePath(e)] = true
	}
	for _, candidate := range affected {
		if existing[NormalizePath(candidate)] {
			return true
		}
	}
	return false
}

// FindCovering returns the first observation in the union that covers the
// candidate finding, or nil when the drift is not yet covered.
//
// Observations on observer/* branches always participate. Observations on
// main participate only while not superseded by recurrence semantics:
// resolved drift that recurs is new drift, so a row on main suppresses a
// re-flag only when the same slug is still carried by a visible observer
// branch — history alone does not. Open/parked findings are what dedup
// protects, not history.
func (u *Union) FindCovering(typ string, affected []string) *Observation {
	// Branch-carried observations first: they are the live state machine.
	for _, o := range u.Observations {
		if !isMainRef(o.Ref) && o.Covers(typ, affected) {
			return o
		}
	}
	return nil
}

// DigestCap is the maximum number of entries in the rendered digest view.
const DigestCap = 5

// Digest computes the rendered digest view over the union: observations with
// status open across the visible branch union, ranked by severity (high >
// medium > low, oldest first within a severity), capped at DigestCap.
//
// The digest is a query, never a committed artifact: observations/DIGEST.md
// is abolished, and no part of the observer workflow writes a digest file.
// A slug present on main is excluded (effectively not open — the backstop),
// and each slug appears at most once even when visible on several refs.
func (u *Union) Digest() []*Observation {
	seen := map[string]bool{}
	var open []*Observation
	for _, o := range u.Observations {
		if o.Status != StatusOpen || isMainRef(o.Ref) {
			continue
		}
		if u.OnMain(o.Slug) || seen[o.Slug] {
			continue
		}
		seen[o.Slug] = true
		open = append(open, o)
	}
	SortForDigest(open)
	if len(open) > DigestCap {
		open = open[:DigestCap]
	}
	return open
}

// RefsFingerprint returns a stable fingerprint of the observation-relevant
// refs — every visible observer/* branch (local and remote-tracking) plus
// main/master — and their tip object names. The index's freshness gate
// compares fingerprints across runs: a fetch that changes, adds, or deletes
// an observer branch changes the fingerprint and triggers a rebuild. Outside
// a git repository the fingerprint is "" (and stays "", so nothing churns).
func RefsFingerprint() (string, error) {
	if crossbranch.RepoRoot() == "" {
		return "", nil
	}
	out, err := crossbranch.Run("for-each-ref", "--format=%(refname) %(objectname)", "refs/heads", "refs/remotes")
	if err != nil {
		return "", err
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ref := strings.SplitN(line, " ", 2)[0]
		if refIsObservationRelevant(ref) {
			lines = append(lines, line)
		}
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n"), nil
}

// refIsObservationRelevant reports whether a fully-qualified ref belongs to
// the observation union: an observer/* branch or main/master, local or
// remote-tracking.
func refIsObservationRelevant(ref string) bool {
	name := ""
	if rest, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
		name = rest
	} else if rest, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			name = rest[i+1:]
		}
	}
	if name == "" {
		return false
	}
	return name == "main" || name == "master" || strings.HasPrefix(name, BranchPrefix)
}
