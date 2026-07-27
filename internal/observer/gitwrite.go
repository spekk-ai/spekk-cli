package observer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// This file is the write-side git surface of the announce flow. It cannot
// route through internal/crossbranch's read-only chokepoint (fetch, push,
// and the plumbing that builds the flip commit are writes by design), so it
// execs git directly — but only through the small set of helpers below, and
// never by touching the working tree or the checked-out branch: the flip
// commit is built with plumbing (hash-object, read-tree into a temporary
// index, write-tree, commit-tree) and delivered with a targeted push.

// gitRunner executes git in a fixed repository directory.
type gitRunner struct {
	dir string
}

// run executes git with args and returns trimmed stdout. extraEnv entries
// are appended to the inherited environment.
func (g gitRunner) run(extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.Output()
	if err != nil {
		detail := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			detail = ": " + strings.TrimSpace(string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git %s: %w%s", strings.Join(args, " "), err, detail)
	}
	return strings.TrimSpace(string(out)), nil
}

// fetch runs the flow's single remote read: `git fetch origin`.
func (g gitRunner) fetch() error {
	_, err := g.run(nil, "fetch", "--quiet", "origin")
	return err
}

// originRef returns the remote-tracking ref for branch on origin, or ok
// false when origin does not carry the branch. Announce eligibility requires
// origin visibility: unpushed local observer branches are skipped — pushing
// the branch is the scan's job, not announce's.
func (g gitRunner) originRef(branch string) (string, bool, error) {
	ref := "refs/remotes/origin/" + branch
	out, err := g.run(nil, "for-each-ref", "--format=%(refname)", ref)
	if err != nil {
		return "", false, err
	}
	return ref, out != "", nil
}

// fileAt returns the contents of path at rev.
func (g gitRunner) fileAt(rev, path string) (string, error) {
	return g.run(nil, "show", rev+":"+path)
}

// commitFileChange builds a commit on top of parentRev that changes exactly
// one file to content, without touching the working tree or the index of the
// checked-out branch. It returns the new commit's sha.
func (g gitRunner) commitFileChange(parentRev, path, content, message string) (string, error) {
	parent, err := g.run(nil, "rev-parse", "--verify", parentRev+"^{commit}")
	if err != nil {
		return "", err
	}

	// Store the new blob.
	hashCmd := exec.Command("git", "hash-object", "-w", "--stdin")
	hashCmd.Dir = g.dir
	hashCmd.Stdin = strings.NewReader(content)
	blobOut, err := hashCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git hash-object: %w", err)
	}
	blob := strings.TrimSpace(string(blobOut))

	// Build the tree in a temporary index so the real index stays untouched.
	tmpIndex, err := os.CreateTemp("", "spekk-observer-index-*")
	if err != nil {
		return "", fmt.Errorf("cannot create temporary git index: %w", err)
	}
	tmpIndexPath := tmpIndex.Name()
	tmpIndex.Close()
	os.Remove(tmpIndexPath) // read-tree wants to create it itself
	defer os.Remove(tmpIndexPath)
	indexEnv := []string{"GIT_INDEX_FILE=" + tmpIndexPath}

	if _, err := g.run(indexEnv, "read-tree", parent); err != nil {
		return "", err
	}
	if _, err := g.run(indexEnv, "update-index", "--add", "--cacheinfo", "100644,"+blob+","+path); err != nil {
		return "", err
	}
	tree, err := g.run(indexEnv, "write-tree")
	if err != nil {
		return "", err
	}

	commit, err := g.run(nil, "commit-tree", tree, "-p", parent, "-m", message)
	if err != nil {
		return "", err
	}
	return commit, nil
}

// pushCommit pushes a commit sha to branch on origin. The push is a plain
// (non-force) update: it succeeds only as a fast-forward, so a concurrent
// update of the branch rejects the push instead of being overwritten — the
// flip then retries on the next run.
func (g gitRunner) pushCommit(commit, branch string) error {
	_, err := g.run(nil, "push", "--quiet", "origin", commit+":refs/heads/"+branch)
	return err
}

// fastForwardLocal moves the local branch ref to commit when the local
// branch exists and still points at parent. Failure is not an error: the
// local ref simply lags until the next fetch.
func (g gitRunner) fastForwardLocal(branch, commit, parent string) {
	local := "refs/heads/" + branch
	out, err := g.run(nil, "for-each-ref", "--format=%(objectname)", local)
	if err != nil || out != parent {
		return
	}
	_, _ = g.run(nil, "update-ref", local, commit, parent)
}

// repoRoot returns the toplevel of the repository containing dir.
func repoRoot(dir string) (string, error) {
	g := gitRunner{dir: dir}
	out, err := g.run(nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: %w", err)
	}
	return filepath.Clean(out), nil
}
