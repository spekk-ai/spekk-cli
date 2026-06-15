package crossbranch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// rootCache memoizes the repository toplevel for a given working directory so
// repoRoot does not shell out on every git call. Keyed by cwd (not a single
// process-wide value) so that test binaries operating on several temp repos in
// one process each resolve their own root correctly.
var rootCache sync.Map // cwd string -> root string

// repoRoot returns the absolute toplevel of the git repository containing the
// current working directory, or "" if it cannot be determined (e.g. not in a
// repo). Every git command in this package runs with its working directory set
// to this root, so pathspecs like "specs" and "ls-tree -- <path>" resolve
// against the repo root regardless of which subdirectory the user invoked spekk
// from. It execs git directly (not via Run) to avoid recursing through the
// chokepoint, and is read-only (rev-parse).
func repoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if v, ok := rootCache.Load(cwd); ok {
		return v.(string)
	}
	root := ""
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		root = strings.TrimSpace(string(out))
	}
	rootCache.Store(cwd, root)
	return root
}

// readOnlySubcommands is the allowlist of git subcommands this package may run.
// Every command here either reports information or writes only to the object
// store (merge-tree) — none of them can mutate the working tree, the index, the
// current branch, or any refs. Routing every git call through Run with this
// allowlist makes the read-only guarantee structural: a mutating subcommand
// (checkout, switch, merge, reset, stash, add, commit, ...) cannot reach exec.
var readOnlySubcommands = map[string]bool{
	"--version":    true,
	"rev-parse":    true,
	"for-each-ref": true,
	"branch":       true, // list/read only — see guarded flags below
	"merge-base":   true,
	"diff":         true,
	"diff-tree":    true,
	"show":         true,
	"ls-tree":      true,
	"cat-file":     true,
	"merge-tree":   true,
}

// branchMutatingFlags are flags that turn `git branch` from a read-only list
// into a ref mutation (delete, rename, force, copy). They are rejected so that
// only the listing forms of `git branch` are reachable.
var branchMutatingFlags = map[string]bool{
	"-d":                 true,
	"-D":                 true,
	"--delete":           true,
	"-m":                 true,
	"-M":                 true,
	"--move":             true,
	"-c":                 true,
	"-C":                 true,
	"--copy":             true,
	"-f":                 true,
	"--force":            true,
	"--edit-description": true,
	"--set-upstream-to":  true,
	"-u":                 true,
	"--unset-upstream":   true,
}

// Run executes a read-only git command and returns its trimmed stdout.
//
// It is the single git exec chokepoint for the crossbranch package: all sibling
// files MUST route their git calls through it rather than calling os/exec
// directly. The first argument is the git subcommand and is validated against
// readOnlySubcommands; anything not on the allowlist (or a mutating form of an
// allowlisted command, such as `branch -D`) returns an error without executing.
//
// Usage:
//
//	out, err := crossbranch.Run("rev-parse", "HEAD")
//	out, err := crossbranch.Run("merge-tree", "--write-tree", "HEAD", branch)
func Run(args ...string) (string, error) {
	if err := guard(args); err != nil {
		return "", err
	}
	out, err := gitCmd(args).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// gitCmd builds the *exec.Cmd for an already-guarded git invocation, applying the
// package-wide execution policy: run at the repository toplevel (so cwd-relative
// pathspecs resolve against the repo root) and force core.quotePath=false so that
// non-ASCII spec paths appear verbatim — not octal-escaped and double-quoted — in
// diff/ls-tree/merge-tree output, keeping path matching consistent across calls.
// The config is injected via environment (git >= 2.31) rather than a global -c
// flag so the guard's "args[0] is the subcommand" model stays intact.
func gitCmd(args []string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	if root := repoRoot(); root != "" {
		cmd.Dir = root
	}
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=core.quotePath",
		"GIT_CONFIG_VALUE_0=false",
	)
	return cmd
}

// RunReportingExit behaves like Run but tolerates a single expected nonzero exit
// code, returning the command's stdout in that case instead of discarding it.
//
// It exists for `git merge-tree`, which signals "conflicts found" with exit
// status 1 while still printing the full conflict report to stdout. Run relies on
// (*exec.Cmd).Output, which drops stdout on any nonzero exit, so the report would
// be lost. This keeps merge-tree on the same single chokepoint: the identical
// read-only guard applies, so it cannot broaden which git commands are reachable;
// only output handling differs. Exit codes other than okExit are real errors.
//
//	out, err := crossbranch.RunReportingExit(1, "merge-tree", "--write-tree", "--name-only", "HEAD", rev)
func RunReportingExit(okExit int, args ...string) (string, error) {
	if err := guard(args); err != nil {
		return "", err
	}
	out, err := gitCmd(args).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != okExit {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
	}
	return strings.TrimSpace(string(out)), nil
}

// guard validates that args describe a read-only git invocation. It is exported
// behavior of Run; kept separate so it can be reasoned about and tested in
// isolation.
func guard(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("crossbranch: refusing to run git with no subcommand")
	}
	sub := args[0]
	if !readOnlySubcommands[sub] {
		return fmt.Errorf("crossbranch: git subcommand %q is not on the read-only allowlist", sub)
	}
	if sub == "branch" {
		// `git branch` is read-only only in its listing forms. Reject any
		// mutating flag (delete/rename/force/copy/upstream) and any non-flag
		// operand, since `git branch <name> [<start>]` *creates* a ref.
		sawSeparator := false
		for _, a := range args[1:] {
			if a == "--" {
				sawSeparator = true
				continue
			}
			if !sawSeparator && strings.HasPrefix(a, "-") {
				if branchMutatingFlags[a] {
					return fmt.Errorf("crossbranch: git branch flag %q is not read-only", a)
				}
				continue
			}
			// A positional operand (branch name) means a create/reset form.
			return fmt.Errorf("crossbranch: git branch with operand %q is not read-only (it would create or move a ref)", a)
		}
	}
	return nil
}
