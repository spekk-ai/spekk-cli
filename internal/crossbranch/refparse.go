package crossbranch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// ErrFileAbsent reports that a path does not exist at a given git ref. It is a
// normal, expected outcome (the spec/assertion simply isn't on that branch) and
// is deliberately distinct from a real failure such as an unknown ref or a git
// error. Callers compare against it with errors.Is to tell "absent" apart from
// "broken".
var ErrFileAbsent = errors.New("crossbranch: file absent at ref")

// FileAtRef returns the contents of path as it exists at the given git ref,
// without ever touching the working tree or index. ref may be a branch name,
// remote-tracking ref, commit, or a written tree object from a merge preview.
//
// Presence is probed with `git ls-tree <ref> -- <path>` rather than by reacting
// to `git show`'s exit status: against a valid ref, ls-tree lists the path when
// present and emits nothing (exit 0) when absent, which cleanly separates the
// "not on this branch" case from a genuine error like an unknown ref. When the
// path is absent FileAtRef returns "", ErrFileAbsent; an invalid ref or other
// git failure is returned as a wrapped error.
func FileAtRef(ref, path string) (string, error) {
	// `--` disambiguates the operand as a pathspec so refs/paths that look like
	// flags or each other can't be confused.
	listing, err := Run("ls-tree", ref, "--", path)
	if err != nil {
		// A failure here means the ref itself could not be resolved (or git is
		// otherwise unhappy); it is a real error, not an absent file.
		return "", fmt.Errorf("crossbranch: probing %s:%s: %w", ref, path, err)
	}
	if listing == "" {
		return "", fmt.Errorf("%s:%s: %w", ref, path, ErrFileAbsent)
	}

	// The path exists at this ref; read its blob contents.
	content, err := Run("show", ref+":"+path)
	if err != nil {
		return "", fmt.Errorf("crossbranch: reading %s:%s: %w", ref, path, err)
	}
	return content, nil
}

// SpecAtRef reads a spec parent file at the given ref and parses it with the
// existing parser. path is also used as the parsed spec's file path (for
// diagnostics). If the file is absent at ref the error wraps ErrFileAbsent so
// callers can distinguish "this spec isn't on that branch" from a parse or git
// failure; malformed content surfaces as the parser's own error, contained to
// this file/ref.
func SpecAtRef(ref, path string) (*parser.Spec, error) {
	content, err := FileAtRef(ref, path)
	if err != nil {
		return nil, err
	}
	spec, err := parser.ParseSpecContent(path, content)
	if err != nil {
		return nil, fmt.Errorf("crossbranch: parsing spec %s:%s: %w", ref, path, err)
	}
	return spec, nil
}

// AssertionAtRef reads an assertion file at the given ref and parses it with the
// existing parser. path is also used as the parsed assertion's file path (for
// diagnostics). If the file is absent at ref the error wraps ErrFileAbsent;
// malformed content surfaces as the parser's own error, contained to this
// file/ref.
func AssertionAtRef(ref, path string) (*parser.Assertion, error) {
	content, err := FileAtRef(ref, path)
	if err != nil {
		return nil, err
	}
	a, err := parser.ParseAssertionContent(path, content)
	if err != nil {
		return nil, fmt.Errorf("crossbranch: parsing assertion %s:%s: %w", ref, path, err)
	}
	return a, nil
}

// FilesAtRef lists the blob paths that exist under dir at the given git ref,
// without touching the working tree. Paths are returned repo-root-relative
// (as `git ls-tree -r --name-only <ref> -- <dir>` prints them), in git's
// stable sort order. A dir that does not exist at the ref yields an empty
// slice and a nil error — like ErrFileAbsent, an absent directory is a
// normal outcome, but for a listing the natural encoding is emptiness. An
// invalid ref or other git failure is returned as a wrapped error.
func FilesAtRef(ref, dir string) ([]string, error) {
	out, err := Run("ls-tree", "-r", "--name-only", ref, "--", dir)
	if err != nil {
		return nil, fmt.Errorf("crossbranch: listing %s:%s: %w", ref, dir, err)
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	paths := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			paths = append(paths, l)
		}
	}
	return paths, nil
}
