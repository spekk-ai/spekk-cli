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
	name := BranchFromRef(ref)
	return name == "main" || name == "master"
}

// isLiveClaim reports whether an observation still speaks for its finding:
// somebody filed it, its branch is there, and the work is not finished.
// Both facts come from git, not from the frontmatter:
//
//   - The ref is the branch named after the observation. Every observer
//     branch is cut from origin/main, so every branch carries a copy of
//     every observation already merged. Such a copy sits at another
//     finding's branch and is not a claim on anything. Main is never
//     observer/<slug>, so this excludes main as well.
//   - The slug is not on main. Presence on main ends the claim, whatever a
//     lagging status field says, and it settles the branch that was merged
//     but not deleted: both facts are true at once, and main wins.
//
// The branch-local `status: resolved` is deliberately not part of this. The
// frontmatter is the record and the branch set is the state machine, so a
// status flipped while the remedy PR is still open must not end a claim
// that is live.
func (u *Union) isLiveClaim(o *Observation) bool {
	if BranchFromRef(o.Ref) != BranchName(o.Slug) {
		return false
	}
	return !u.OnMain(o.Slug)
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
// as written, but with no location suffix, no ./ prefix, no redundant
// slashes, and no surrounding whitespace. It does not make an absolute path
// repository-root-relative — writing root-relative paths is the prompt's
// job, like naming files rather than directories.
//
// Dont-flag suppression matches on these strings, and the strings are
// written by an agent rather than produced by the code — so the same file
// arrives spelled several ways. A run that names
// `internal/parser/parser.go:42` where a suppression entry named
// `internal/parser/parser.go` is describing the same file, but the raw
// strings differ, so the entry does not match. Normalizing removes the
// three spellings that carry no meaning:
//
//	./internal/parser/parser.go     leading ./
//	internal/parser/parser.go:42    a line, or line:column
//	internal/parser/parser.go/      a trailing slash
//
// A directory is deliberately NOT reduced to the files under it. Prefix
// containment would let one directory-level entry swallow every file-level
// finding beneath it — a false negative that hides real drift, which is
// worse than a duplicate. Naming files rather than directories is the
// prompt's job, not this function's.
//
// Scan-time dedup no longer reads these strings at all: it keys on the type
// and the slug (see Covers).
func NormalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	// A trailing :line or :line:column. Only digits qualify, and at most two
	// are removed, because that is all a location can be. A file whose name
	// genuinely ends in a colon and digits still loses that much -- rare, and
	// preferred to leaving every real location suffix in place -- but a name
	// like `a:1:2:3` is not eaten whole.
	for strips := 0; strips < 2; strips++ {
		i := strings.LastIndex(p, ":")
		if i <= 0 || i == len(p)-1 || !allDigits(p[i+1:]) {
			break
		}
		p = p[:i]
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

// Covers reports whether the observation is the same finding as a
// candidate: same type and same slug.
//
// A dated slug and its plain form are the same finding, so the comparison
// is on the base slug (see BaseSlug).
//
// The slug is what a finding is called; an affected path is only where it
// lives. Keying on an overlapping path made two unrelated findings in one
// file the same finding — and the affected list of a code_spec_misalignment
// names the code as well as the assertion, so the code file is exactly the
// half that unrelated findings share. That is a false negative, and this
// package prefers a visible duplicate to drift nobody hears about (see
// NormalizePath on directories). The type stays in the key because one file
// can carry drift of both types at once.
func (o *Observation) Covers(typ, slug string) bool {
	return o.Type == typ && BaseSlug(o.Slug) == BaseSlug(slug)
}

// BaseSlug strips a trailing -YYYYMMDD from a slug.
//
// ResolveSlug appends that suffix when a plain slug is already taken by
// history, so a recurrence files under a dated name. The suffix uniquifies
// the branch; it does not make the finding a different finding. Comparing
// whole slugs made a recurrence's own live claim invisible to the next
// scan, which then proposed a branch that already existed and could not be
// created.
func BaseSlug(slug string) string {
	i := strings.LastIndexByte(slug, '-')
	if i <= 0 || len(slug)-i-1 != 8 || !allDigits(slug[i+1:]) {
		return slug
	}
	return slug[:i]
}

// FindCovering returns the live claim in the union that is the candidate
// finding, or nil when nobody is claiming it.
//
// Only a live claim covers (see isLiveClaim): the observation on the branch
// named after it, whose slug has not reached main. A parked branch (its PR
// closed, the branch kept) claims exactly like a pending one, because the
// tooling never reads PR state. A covered finding must not produce a new
// observation or a new branch.
func (u *Union) FindCovering(typ, slug string) *Observation {
	for _, o := range u.Observations {
		if u.isLiveClaim(o) && o.Covers(typ, slug) {
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
// It shows the live claims that are open (see isLiveClaim), so the digest
// and scan-time dedup answer "which findings are live" the same way. Each
// slug appears at most once, and it is the copy on the branch named after
// it — the copies other branches inherited are not claims.
func (u *Union) Digest() []*Observation {
	seen := map[string]bool{}
	var open []*Observation
	for _, o := range u.Observations {
		if o.Status != StatusOpen || !u.isLiveClaim(o) {
			continue
		}
		if seen[o.Slug] {
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
