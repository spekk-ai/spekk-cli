package crossbranch

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Branch is one comparison target for cross-branch mode: a branch (local or
// remote-tracking) to merge-preview against the current branch ("ours").
//
// The classifier consumes these to run an in-memory three-way merge per
// branch, e.g. Run("merge-tree", "--write-tree", "HEAD", b.Rev). Rev is a
// fully-qualified ref name so it resolves unambiguously regardless of whether
// the logical branch came from refs/heads or refs/remotes.
type Branch struct {
	// Name is the logical, human-facing branch name with the ref namespace
	// stripped: "refs/heads/feature" and "refs/remotes/origin/feature" both
	// reduce to "feature". This is what the name filter matches against and
	// what should be displayed.
	Name string

	// Ref is the fully-qualified ref the branch was discovered as, e.g.
	// "refs/heads/feature" or "refs/remotes/origin/feature". Useful for
	// display when a branch only exists on the remote.
	Ref string

	// Rev is the revision to hand to git for the three-way merge (the value
	// that goes in the <branch> slot of `git merge-tree HEAD <branch>`). It is
	// the fully-qualified ref name, which git resolves directly.
	Rev string

	// Remote reports whether this logical branch was resolved from a
	// remote-tracking ref (refs/remotes/*) rather than a local head. A local
	// head is always preferred when both exist, so this is true only for
	// branches that exist solely on a remote.
	Remote bool
}

// DiscoverBranches returns the deduplicated union of local branches and
// remote-tracking refs to compare the current branch ("ours") against.
//
// Behavior:
//
//   - The set is the union of local heads (refs/heads/*) and remote-tracking
//     refs (refs/remotes/*), enumerated via `git for-each-ref` through the
//     read-only chokepoint (Run). `git branch` is deliberately avoided.
//   - The current branch (HEAD) is excluded.
//   - Deduplication is by logical name: a local "foo" and a remote-tracking
//     "origin/foo" collapse to a single Branch. When both exist the local head
//     wins (it is the ref a developer is actually working with); the remote
//     ref is kept only when there is no matching local head.
//   - "origin/HEAD" (and any "<remote>/HEAD" symbolic ref) is skipped — it is
//     an alias, not a real branch.
//   - filter, when non-empty, is a glob matched against the logical Name using
//     filepath.Match semantics ('*' matches any run of non-separator chars,
//     '?' a single char, '[...]' a class). Only matching branches are kept; an
//     empty filter keeps everything. A malformed pattern returns an error.
//
// Detached HEAD or a repo with no other branches yields an empty slice and a
// nil error, so callers can render normally with nothing to compare.
//
// The result is sorted by Name for stable, deterministic output.
func DiscoverBranches(filter string) ([]Branch, error) {
	// Validate the glob up front so a bad pattern fails fast rather than
	// silently dropping everything. filepath.Match only reports ErrBadPattern
	// when the pattern is actually exercised, so probe it once.
	if filter != "" {
		if _, err := filepath.Match(filter, ""); err != nil {
			return nil, fmt.Errorf("crossbranch: invalid branch filter %q: %w", filter, err)
		}
	}

	current, err := currentBranch()
	if err != nil {
		return nil, err
	}

	out, err := Run("for-each-ref", "--format=%(refname)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}

	// Collect by logical name. localSeen records names that have a local head
	// so a later remote-tracking ref for the same name does not overwrite it.
	byName := map[string]Branch{}
	localSeen := map[string]bool{}

	for _, ref := range strings.Split(out, "\n") {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}

		name, remote, ok := logicalName(ref)
		if !ok {
			continue // e.g. origin/HEAD, or an unrecognized namespace
		}

		// Exclude "ours". When HEAD is detached, current is "" and matches
		// nothing, so every branch stays in the set.
		if name == current {
			continue
		}

		b := Branch{Name: name, Ref: ref, Rev: ref, Remote: remote}

		if remote {
			// Only keep a remote-tracking branch if no local head claimed the
			// name and it is the first remote we have seen for it.
			if localSeen[name] {
				continue
			}
			if _, exists := byName[name]; exists {
				continue
			}
			byName[name] = b
			continue
		}

		// Local head: always wins over any remote-tracking ref.
		byName[name] = b
		localSeen[name] = true
	}

	branches := make([]Branch, 0, len(byName))
	for _, b := range byName {
		if filter != "" {
			matched, _ := filepath.Match(filter, b.Name)
			if !matched {
				continue
			}
		}
		branches = append(branches, b)
	}

	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})

	return branches, nil
}

// currentBranch returns the logical name of the checked-out branch, or "" when
// HEAD is detached (rev-parse reports "HEAD" in that case).
func currentBranch() (string, error) {
	out, err := Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", nil // detached HEAD: no branch to exclude
	}
	return out, nil
}

// logicalName reduces a fully-qualified ref to its logical branch name and
// reports whether it is a remote-tracking ref. The boolean ok is false for refs
// that should not participate in the comparison set: any "<remote>/HEAD"
// symbolic alias, and refs outside the heads/remotes namespaces.
//
//   - refs/heads/foo            -> ("foo", false, true)
//   - refs/remotes/origin/foo   -> ("foo", true,  true)
//   - refs/remotes/origin/HEAD  -> ("", false, false)  (skipped)
func logicalName(ref string) (name string, remote bool, ok bool) {
	switch {
	case strings.HasPrefix(ref, "refs/heads/"):
		return strings.TrimPrefix(ref, "refs/heads/"), false, true
	case strings.HasPrefix(ref, "refs/remotes/"):
		rest := strings.TrimPrefix(ref, "refs/remotes/")
		// rest is "<remote>/<branch...>". Drop the first path segment (the
		// remote name) to get the logical branch name.
		slash := strings.IndexByte(rest, '/')
		if slash < 0 {
			return "", false, false
		}
		branch := rest[slash+1:]
		if branch == "" || branch == "HEAD" {
			return "", false, false // skip origin/HEAD-style aliases
		}
		return branch, true, true
	default:
		return "", false, false
	}
}
