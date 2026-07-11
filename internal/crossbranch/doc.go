// Package crossbranch implements the read-only cross-branch / merge-preview
// engine behind `spekk show --cross-branch`.
//
// It compares the current branch ("ours") against every other local and
// remote-tracking branch to surface incoming additions, clean modifications,
// conflicts, and deletions to the spec corpus — a preview of what merging each
// branch would do, without ever performing a merge.
//
// The comparison is a three-way diff against a single merge-base (git merge-base
// HEAD <branch>) confirmed by git merge-tree. For criss-cross histories with
// multiple merge bases this approximates git's recursive merge rather than
// reproducing it exactly; it is a preview, not the merge itself, so the
// occasional divergence from a real merge on such histories is acceptable.
//
// The package is read-only with respect to git state. It shells out to git via
// os/exec, but every invocation is funneled through the single chokepoint in
// gitcmd.go, which permits only an allowlist of subcommands that do not mutate
// the working tree, the index, the current branch, or any refs (the allowlist
// gates subcommands; it does not exhaustively constrain every flag, so callers
// must not construct a read subcommand with a writing flag such as
// `diff --output`). All call sites in this package are fixed and pass no such
// flags, and the diff calls additionally pass --no-ext-diff to avoid invoking
// external programs while reading other branches' content.
package crossbranch
