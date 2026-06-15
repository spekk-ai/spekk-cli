package crossbranch

import (
	"fmt"
	"sort"
	"strings"
)

// State is the cross-branch classification of a single spec/assertion file for
// one comparison branch, all keyed to the current branch ("ours"). Exactly one
// state applies to a given (file, branch) pair; an unchanged pair produces no
// FileState at all (see Classify).
type State string

const (
	// StateIncomingAdd: the file exists on the other branch but not on ours —
	// a foreign spec/assertion that a merge would introduce.
	StateIncomingAdd State = "incoming_add"

	// StateIncomingMod: the file was modified on the other branch and is
	// unchanged on ours relative to the merge-base, so it would merge cleanly.
	StateIncomingMod State = "incoming_mod"

	// StateConflict: the file was modified on BOTH ours and the other branch in
	// a way that git merge-tree reports as a conflicted path. In degraded mode
	// (git < 2.38) a both-sides-modified file is reported with this state but
	// with FileState.Degraded set, meaning the conflict is a *candidate* that
	// could not be confirmed.
	StateConflict State = "conflict"

	// StateIncomingDel: the file exists on ours but was deleted on the other
	// branch.
	StateIncomingDel State = "incoming_del"
)

// FileState is one (file, branch) contribution the next wave
// (cross-branch-data-model) rolls up into a union view. A file unchanged
// between ours and a branch yields no FileState for that pair.
type FileState struct {
	// Path is the repo-relative path of the spec/assertion file, slash-form,
	// always under "specs/" (e.g. "specs/foo/assertions/bar.md").
	Path string

	// Branch is the display name of the contributing comparison branch (the
	// logical Branch.Name).
	Branch string

	// State is the cross-branch classification for this (file, branch) pair.
	State State

	// Degraded is true only for a StateConflict that could not be confirmed by
	// merge-tree because the installed git is too old (< 2.38). It means
	// "both sides modified this file; treat as a *potential* conflict." The
	// renderer should label such entries as unconfirmed rather than asserting a
	// true conflict. It is always false for the other three states.
	Degraded bool

	// OldStatus / NewStatus capture assertion status drift for a clean
	// StateIncomingMod on an assertion file: OldStatus is the assertion's
	// status: on ours, NewStatus is its status: on the other branch. They are
	// populated only when both sides parse as assertions and the status
	// actually differs (e.g. "not_started" -> "done"); otherwise both are "".
	// This lets the renderer highlight status drift without re-parsing.
	OldStatus string
	NewStatus string
}

// Classify discovers the comparison branches (via DiscoverBranches with the
// given name filter) and classifies every changed spec/assertion file under
// specs/ for each branch against the current branch ("ours"), returning a flat
// list of (path, branch, state[, drift]) contributions.
//
// For each branch it:
//   - finds the merge-base with HEAD (git merge-base),
//   - diffs base->ours and base->theirs over specs/ (git diff --name-status),
//     restricted to spec/assertion .md files,
//   - pairs the per-file change on each side to classify add / modify / delete,
//   - confirms both-sides-modified candidates as true conflicts with
//     merge-tree when the git version supports it; otherwise marks them as
//     degraded (unconfirmed) conflicts.
//
// Files unchanged on both sides contribute nothing. The result is sorted by
// (Path, Branch) for deterministic output. An empty comparison set (no other
// branches, detached HEAD, ...) yields an empty slice and a nil error.
func Classify(filter string) ([]FileState, error) {
	branches, err := DiscoverBranches(filter)
	if err != nil {
		return nil, err
	}

	mergeTreeOK, err := SupportsMergeTree()
	if err != nil {
		return nil, err
	}

	var result []FileState
	for _, b := range branches {
		states, cerr := classifyBranch(b, mergeTreeOK)
		if cerr != nil {
			return nil, cerr
		}
		result = append(result, states...)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Branch < result[j].Branch
	})
	return result, nil
}

// classifyBranch classifies all changed spec/assertion files for one comparison
// branch against HEAD.
func classifyBranch(b Branch, mergeTreeOK bool) ([]FileState, error) {
	base, err := Run("merge-base", "HEAD", b.Rev)
	if err != nil {
		return nil, fmt.Errorf("crossbranch: merge-base HEAD %s: %w", b.Rev, err)
	}

	ours, err := changedFiles(base, "HEAD")
	if err != nil {
		return nil, err
	}
	theirs, err := changedFiles(base, b.Rev)
	if err != nil {
		return nil, err
	}

	// Union of all files touched on either side (only files changed on theirs
	// can ever contribute an incoming state, but we need ours' change to
	// classify modify-vs-conflict).
	confirmedConflicts := map[string]bool{}
	conflictsResolved := false

	var states []FileState
	for path, their := range theirs {
		our, ourChanged := ours[path]

		switch their {
		case changeAdd:
			// Added on theirs. If also present/added on ours it would be a
			// both-added candidate; classify as conflict candidate only when
			// ours also added (same path). Otherwise it is a clean incoming
			// addition.
			if ourChanged && our == changeAdd {
				st, deg, rerr := resolveConflict(b, path, mergeTreeOK, &confirmedConflicts, &conflictsResolved)
				if rerr != nil {
					return nil, rerr
				}
				if st == StateConflict {
					states = append(states, FileState{Path: path, Branch: b.Name, State: StateConflict, Degraded: deg})
				} else {
					states = append(states, FileState{Path: path, Branch: b.Name, State: StateIncomingAdd})
				}
				continue
			}
			states = append(states, FileState{Path: path, Branch: b.Name, State: StateIncomingAdd})

		case changeDelete:
			// Deleted on theirs. If ours also changed it (modify/delete) the
			// merge is not trivially clean; let merge-tree adjudicate. A plain
			// deletion (ours untouched) is an incoming deletion.
			if ourChanged && our != changeDelete {
				st, deg, rerr := resolveConflict(b, path, mergeTreeOK, &confirmedConflicts, &conflictsResolved)
				if rerr != nil {
					return nil, rerr
				}
				if st == StateConflict {
					states = append(states, FileState{Path: path, Branch: b.Name, State: StateConflict, Degraded: deg})
					continue
				}
			}
			states = append(states, FileState{Path: path, Branch: b.Name, State: StateIncomingDel})

		case changeModify:
			if !ourChanged {
				// Clean incoming modification. Capture status drift for
				// assertion files where the status: field changed.
				fs := FileState{Path: path, Branch: b.Name, State: StateIncomingMod}
				if old, neu, ok := statusDrift(b, path); ok {
					fs.OldStatus = old
					fs.NewStatus = neu
				}
				states = append(states, fs)
				continue
			}
			// Both sides changed -> conflict candidate; confirm with merge-tree.
			st, deg, rerr := resolveConflict(b, path, mergeTreeOK, &confirmedConflicts, &conflictsResolved)
			if rerr != nil {
				return nil, rerr
			}
			if st == StateConflict {
				states = append(states, FileState{Path: path, Branch: b.Name, State: StateConflict, Degraded: deg})
			} else {
				// merge-tree merged this file cleanly despite both touching it.
				fs := FileState{Path: path, Branch: b.Name, State: StateIncomingMod}
				if old, neu, ok := statusDrift(b, path); ok {
					fs.OldStatus = old
					fs.NewStatus = neu
				}
				states = append(states, fs)
			}
		}
	}

	return states, nil
}

// resolveConflict decides whether path is a true conflict for branch b.
//
// In supported mode it runs merge-tree once per branch (memoized via the
// resolved flag and confirmed set) and reports whether path is among the
// conflicted paths. A both-changed file merge-tree merges cleanly is NOT a
// conflict (the caller treats it as a clean incoming-modification).
//
// In degraded mode (git < 2.38) merge-tree's real-merge cannot be trusted, so
// every both-changed candidate is reported as a conflict with Degraded=true:
// the state is surfaced as a *potential* conflict the renderer must label as
// unconfirmed.
func resolveConflict(b Branch, path string, mergeTreeOK bool, confirmed *map[string]bool, resolved *bool) (State, bool, error) {
	if !mergeTreeOK {
		// Degraded: cannot confirm. Surface as an unconfirmed conflict.
		return StateConflict, true, nil
	}
	if !*resolved {
		set, err := mergeTreeConflicts(b.Rev)
		if err != nil {
			return "", false, err
		}
		*confirmed = set
		*resolved = true
	}
	if (*confirmed)[path] {
		return StateConflict, false, nil
	}
	// merge-tree merged it cleanly.
	return StateIncomingMod, false, nil
}

type changeKind int

const (
	changeAdd changeKind = iota
	changeModify
	changeDelete
)

// changedFiles returns the spec/assertion .md files under specs/ that changed
// between from and to, mapped to their change kind. Renames and copies are
// decomposed by git into delete+add via --diff-filter defaults; we read the
// name-status form and treat A/C as add, M as modify, D as delete. Paths are
// repo-relative slash-form.
func changedFiles(from, to string) (map[string]changeKind, error) {
	// --no-renames keeps a rename as delete+add, which is exactly how we want
	// to classify it at the file level (the old path is an incoming deletion,
	// the new path an incoming addition) and avoids R<score> status parsing.
	out, err := Run("diff", "--name-status", "--no-renames", from, to, "--", "specs")
	if err != nil {
		return nil, fmt.Errorf("crossbranch: diff %s..%s: %w", from, to, err)
	}
	changes := map[string]changeKind{}
	if out == "" {
		return changes, nil
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// name-status lines are "<STATUS>\t<path>".
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) != 2 {
			continue
		}
		status, path := fields[0], strings.TrimSpace(fields[1])
		if !isSpecFile(path) {
			continue
		}
		switch status[0] {
		case 'A', 'C':
			changes[path] = changeAdd
		case 'M', 'T':
			changes[path] = changeModify
		case 'D':
			changes[path] = changeDelete
		default:
			// Unknown status (e.g. U for unmerged) — treat as modify so it is
			// surfaced rather than silently dropped.
			changes[path] = changeModify
		}
	}
	return changes, nil
}

// isSpecFile reports whether path is a spec parent or assertion markdown file
// under specs/. Both spec files (specs/<id>/<id>.md) and assertion files
// (specs/<id>/assertions/<name>.md) qualify.
func isSpecFile(path string) bool {
	return strings.HasPrefix(path, "specs/") && strings.HasSuffix(path, ".md")
}

// isAssertionFile reports whether path is an assertion file (lives under an
// assertions/ directory). Used to decide whether status drift is meaningful.
func isAssertionFile(path string) bool {
	return isSpecFile(path) && strings.Contains(path, "/assertions/")
}

// statusDrift returns (oldStatus, newStatus, true) when path is an assertion
// file whose status: differs between ours (HEAD) and the other branch. It
// reuses the existing parser via AssertionAtRef. Any parse/absence problem, a
// non-assertion path, or an unchanged status yields ok=false so the caller
// simply omits drift detail.
func statusDrift(b Branch, path string) (oldStatus, newStatus string, ok bool) {
	if !isAssertionFile(path) {
		return "", "", false
	}
	ourA, err := AssertionAtRef("HEAD", path)
	if err != nil {
		// Absent on ours (it's an incoming add, not a mod) or unparseable —
		// no drift to report here.
		return "", "", false
	}
	theirA, err := AssertionAtRef(b.Rev, path)
	if err != nil {
		return "", "", false
	}
	if ourA.Status == theirA.Status {
		return "", "", false
	}
	return ourA.Status, theirA.Status, true
}

// mergeTreeConflicts runs an in-memory three-way merge of HEAD with rev and
// returns the set of conflicted paths (repo-relative slash-form).
//
// It uses `git merge-tree --write-tree --name-only HEAD <rev>`, whose
// conflicted-file-info section is a plain list of paths (one per line) after
// the merged-tree OID on the first line. merge-tree is read-only — it writes
// only to the object store, never the working tree or index — and is on the
// crossbranch read-only allowlist.
//
// merge-tree signals "conflicts exist" with exit status 1 while still printing
// the conflict report to stdout, so this routes through RunReportingExit(1, ...)
// (the chokepoint variant that captures stdout on the expected exit code) rather
// than plain Run, which would discard the report on the nonzero exit.
func mergeTreeConflicts(rev string) (map[string]bool, error) {
	out, err := RunReportingExit(1, "merge-tree", "--write-tree", "--name-only", "HEAD", rev)
	if err != nil {
		return nil, fmt.Errorf("crossbranch: merge-tree HEAD %s: %w", rev, err)
	}
	return parseMergeTreeConflicts(out), nil
}

// parseMergeTreeConflicts extracts the conflicted paths from the stdout of
// `git merge-tree --write-tree --name-only`. The format is:
//
//	<merged-tree-oid>\n
//	<conflicted path>\n      (zero or more, one path per line)
//	\n                       (blank line)
//	<informational messages> (Auto-merging / CONFLICT (...) lines)
//
// On a clean merge only the OID line is present (and there are no conflicts).
// We read the lines after the OID up to the first blank line as the conflicted
// path list.
func parseMergeTreeConflicts(out string) map[string]bool {
	conflicts := map[string]bool{}
	out = strings.ReplaceAll(out, "\r\n", "\n")
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		return conflicts
	}
	// lines[0] is the merged-tree OID; the conflicted-file section runs from
	// line 1 until the blank line that precedes the informational messages.
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			break
		}
		conflicts[strings.TrimSpace(line)] = true
	}
	return conflicts
}
