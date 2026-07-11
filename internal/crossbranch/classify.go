package crossbranch

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/spekk-ai/spekk-cli/internal/parser"
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

	// Meta carries the parsed metadata of an incoming-addition ("foreign") file,
	// read from the branch where it was added so the renderer can synthesize a
	// complete item (real title/status/priority/content) instead of a blank
	// placeholder. It is populated only for StateIncomingAdd and is nil for every
	// other state and when the foreign file could not be parsed.
	Meta *FileMeta
}

// FileMeta is the parsed metadata of a foreign (incoming-addition) spec or
// assertion file, read from the branch that introduces it. It mirrors the subset
// of parser.Spec / parser.Assertion the explorer needs to render a synthesized
// item. For a spec parent file, Status is left empty: a spec's status is derived
// from its assertions, not stored, so the caller computes it from the foreign
// assertions instead.
type FileMeta struct {
	Title    string
	Status   string
	Branch   string
	Priority int
	Content  string
	// Parent is the frontmatter parent id for assertion files. When populated it
	// takes precedence over the path-derived parent in synthesizeAssertion, so
	// an assertion that lives under one spec directory but declares a different
	// parent in its frontmatter is linked correctly.
	Parent string
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
func Classify(filter string) ([]FileState, bool, error) {
	branches, err := DiscoverBranches(filter)
	if err != nil {
		return nil, false, err
	}

	mergeTreeOK, err := SupportsMergeTree()
	if err != nil {
		return nil, false, err
	}

	var result []FileState
	for _, b := range branches {
		states, cerr := classifyBranch(b, mergeTreeOK)
		if cerr != nil {
			return nil, false, cerr
		}
		result = append(result, states...)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Branch < result[j].Branch
	})
	return result, mergeTreeOK, nil
}

// classifyBranch classifies all changed spec/assertion files for one comparison
// branch against HEAD.
func classifyBranch(b Branch, mergeTreeOK bool) ([]FileState, error) {
	// merge-base exits 1 with empty stdout when the histories share no common
	// ancestor (orphan branches, imported subtrees, --orphan docs branches).
	// Route through RunReportingExit so that exit-1-with-no-output is not treated
	// as a fatal error: such a branch has no three-way base to diff against, so we
	// skip it (contribute nothing) rather than aborting the whole cross-branch
	// render. A genuine failure (bad ref, exit != 1) still returns an error.
	base, err := RunReportingExit(1, "merge-base", "HEAD", b.Rev)
	if err != nil {
		return nil, fmt.Errorf("crossbranch: merge-base HEAD %s: %w", b.Rev, err)
	}
	if base == "" {
		return nil, nil
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
			// Added on theirs. If ours also added the same path it is a both-added
			// candidate: an add/add with differing content conflicts (git cannot
			// three-way merge with no base), which merge-tree reports. If it merges
			// cleanly the two sides added identical content, so the file is already
			// in sync with theirs and a merge would change nothing — contribute no
			// FileState, exactly like an unchanged file (it is NOT an incoming add,
			// since the file already exists on ours).
			if ourChanged && our == changeAdd {
				st, deg, rerr := resolveConflict(b, path, mergeTreeOK, &confirmedConflicts, &conflictsResolved)
				if rerr != nil {
					return nil, rerr
				}
				if st == StateConflict {
					states = append(states, FileState{Path: path, Branch: b.Name, State: StateConflict, Degraded: deg})
				}
				continue
			}
			states = append(states, FileState{Path: path, Branch: b.Name, State: StateIncomingAdd, Meta: foreignMeta(b.Rev, path)})

		case changeDelete:
			// Deleted on theirs. If ours also deleted it (both-deleted), the
			// file is simply gone on both sides — a non-event that contributes
			// nothing (both agree on the deletion).
			if our == changeDelete {
				continue
			}
			// If ours changed it (modify/delete) the merge is not trivially
			// clean; let merge-tree adjudicate. A plain deletion (ours
			// untouched) is an incoming deletion.
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
			// When ours DELETED the file while theirs modified it (a modify/delete
			// conflict), the file has no local entry — it is foreign — so parse its
			// metadata from the branch, which still has it, for synthesis. (When
			// both sides modified, the file exists on ours and is matched, so no
			// foreign metadata is needed.)
			var meta *FileMeta
			if our == changeDelete {
				meta = foreignMeta(b.Rev, path)
			}
			if st == StateConflict {
				states = append(states, FileState{Path: path, Branch: b.Name, State: StateConflict, Degraded: deg, Meta: meta})
			} else {
				// merge-tree merged this file cleanly despite both touching it.
				fs := FileState{Path: path, Branch: b.Name, State: StateIncomingMod, Meta: meta}
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
	// --no-ext-diff prevents any configured external diff driver from running
	// while comparing other branches' content (defense-in-depth; --name-status
	// would not invoke one anyway).
	out, err := Run("diff", "--name-status", "--no-renames", "--no-ext-diff", from, to, "--", "specs")
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
// under specs/, using the same structural layout the parser (ParseAllSpecs)
// recognizes. This deliberately mirrors the parser's convention rather than
// matching any *.md under specs/, so stray markdown (e.g. specs/README.md,
// specs/foo/NOTES.md, deeply nested files) is not classified as a phantom
// spec/assertion the parser would never surface.
func isSpecFile(path string) bool {
	return isSpecParentFile(path) || isAssertionFile(path)
}

// isSpecParentFile reports whether path is a spec parent file of the exact form
// specs/<dir>/<dir>.md — the file basename must equal its immediate directory,
// which is the layout ParseAllSpecs requires (specs/{spec-id}/{spec-id}.md).
func isSpecParentFile(path string) bool {
	if !strings.HasPrefix(path, "specs/") || !strings.HasSuffix(path, ".md") {
		return false
	}
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return false
	}
	return parts[2] == parts[1]+".md"
}

// isAssertionFile reports whether path is an assertion file of the exact form
// specs/<dir>/assertions/<name>.md (a direct child of an assertions/ directory),
// matching how the parser enumerates assertions. Used both to filter the diff and
// to decide whether status drift is meaningful.
func isAssertionFile(path string) bool {
	if !strings.HasPrefix(path, "specs/") || !strings.HasSuffix(path, ".md") {
		return false
	}
	parts := strings.Split(path, "/")
	return len(parts) == 4 && parts[2] == "assertions"
}

// statusDrift returns (oldStatus, newStatus, true) when path is an assertion
// file whose status: differs between ours (HEAD) and the other branch. Any
// parse/absence problem, a non-assertion path, or an unchanged status yields
// ok=false so the caller simply omits drift detail.
//
// statusDrift is only ever called for a clean incoming-modification, where the
// assertion is known to exist on both HEAD and b.Rev. It therefore reads each
// blob with a single `git show <ref>:<path>` rather than AssertionAtRef, whose
// extra ls-tree existence probe is redundant here — halving the git spawns per
// drifting assertion (2 instead of 4).
func statusDrift(b Branch, path string) (oldStatus, newStatus string, ok bool) {
	if !isAssertionFile(path) {
		return "", "", false
	}
	oldStatus, ok = assertionStatusAtRef("HEAD", path)
	if !ok {
		// Absent/unparseable on ours — no drift to report here.
		return "", "", false
	}
	newStatus, ok = assertionStatusAtRef(b.Rev, path)
	if !ok {
		return "", "", false
	}
	if oldStatus == newStatus {
		return "", "", false
	}
	return oldStatus, newStatus, true
}

// foreignMeta parses the metadata of an incoming-addition file at the branch ref
// that introduces it, so the renderer can synthesize a complete foreign item. It
// reuses the existing parser via AssertionAtRef / SpecAtRef (parse-from-ref). Any
// git/parse failure yields nil — the caller falls back to a path-derived
// placeholder rather than failing the whole classification.
func foreignMeta(rev, path string) *FileMeta {
	if isAssertionFile(path) {
		a, err := AssertionAtRef(rev, path)
		if err != nil {
			return nil
		}
		return &FileMeta{Title: a.Title, Status: a.Status, Branch: a.Branch, Priority: a.Priority, Content: a.Content, Parent: a.Parent}
	}
	if isSpecParentFile(path) {
		s, err := SpecAtRef(rev, path)
		if err != nil {
			return nil
		}
		// Spec status is derived from assertions, not stored on the parent; leave
		// it empty for the caller to compute from the foreign assertions.
		return &FileMeta{Title: s.Title, Branch: s.Branch, Priority: s.Priority, Content: s.Content}
	}
	return nil
}

// assertionStatusAtRef reads ref:path as an assertion and returns its status,
// with ok=false on any git/parse failure (treated as "no drift to report"). It
// reads the blob directly via the read-only chokepoint without a separate
// existence probe, since callers only invoke it for paths already known present.
func assertionStatusAtRef(ref, path string) (status string, ok bool) {
	content, err := Run("show", ref+":"+path)
	if err != nil {
		return "", false
	}
	a, err := parser.ParseAssertionContent(path, content)
	if err != nil {
		return "", false
	}
	return a.Status, true
}

// noWriteTreeOnce guards the one-time probe that detects whether the installed
// git supports `merge-tree --no-write-tree`. The flag was introduced alongside
// --write-tree in git 2.38; on older gits the flag is unknown and we fall back
// to --write-tree (which writes loose tree objects that git auto-GC eventually
// collects). A sync.Once keeps the probe at most once per process.
var (
	noWriteTreeOnce      sync.Once
	noWriteTreeSupported bool
)

// supportsNoWriteTree reports whether the installed git understands the
// `merge-tree --no-write-tree` flag. It probes by merging HEAD with itself
// (always a clean, zero-conflict merge) and treating a nil error as "supported"
// and any error as "not supported / flag unknown".
func supportsNoWriteTree() bool {
	noWriteTreeOnce.Do(func() {
		// Merging HEAD with HEAD always exits 0 (clean merge) — no conflict
		// output to worry about. A non-nil error here means git rejected the
		// flag (exit 128 "unknown option") rather than a real merge problem.
		_, err := Run("merge-tree", "--no-write-tree", "HEAD", "HEAD")
		noWriteTreeSupported = (err == nil)
	})
	return noWriteTreeSupported
}

// mergeTreeConflicts runs an in-memory three-way merge of HEAD with rev and
// returns the set of conflicted paths (repo-relative slash-form).
//
// It uses `git merge-tree --name-only HEAD <rev>`, whose conflicted-file-info
// section is a plain list of paths (one per line) after the merged-tree OID on
// the first line. When the installed git supports --no-write-tree (git >= 2.38),
// that flag is used to avoid writing loose tree objects to the object store.
// When it is not supported, --write-tree is used instead; the resulting loose
// objects are harmless — git auto-GC will eventually collect them.
//
// merge-tree signals "conflicts exist" with exit status 1 while still printing
// the conflict report to stdout, so this routes through RunReportingExit(1, ...)
// (the chokepoint variant that captures stdout on the expected exit code) rather
// than plain Run, which would discard the report on the nonzero exit.
func mergeTreeConflicts(rev string) (map[string]bool, error) {
	writeFlag := "--write-tree"
	if supportsNoWriteTree() {
		// --no-write-tree avoids writing loose tree objects to the object
		// store; supported on git >= 2.38.
		writeFlag = "--no-write-tree"
	}
	// Note: when writeFlag is --write-tree, merge-tree writes tree objects
	// which git auto-GC will eventually collect.
	out, err := RunReportingExit(1, "merge-tree", writeFlag, "--name-only", "HEAD", rev)
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
