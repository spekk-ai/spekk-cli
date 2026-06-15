// Package crossbranch implements the read-only cross-branch / merge-preview
// engine behind `spekk show --cross-branch`.
//
// It compares the current branch ("ours") against every other local and
// remote-tracking branch to surface incoming additions, clean modifications,
// conflicts, and deletions to the spec corpus — a preview of what merging each
// branch would do, without ever performing a merge.
//
// The whole package is strictly read-only. It shells out to git via os/exec,
// but every git invocation is funneled through the single chokepoint in
// gitcmd.go, which permits only an allowlist of non-mutating subcommands. As a
// result it is structurally impossible for this package to change the working
// tree, the index, the current branch, or any refs.
package crossbranch
