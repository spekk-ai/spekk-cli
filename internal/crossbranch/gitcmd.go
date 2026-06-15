package crossbranch

import (
	"fmt"
	"os/exec"
	"strings"
)

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
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
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
