package observation

import "strings"

// BranchPrefix is the namespace for observation branches. Each finding is
// born on exactly one branch named BranchPrefix + slug, carrying the
// observation file and its proposed remedy as two separate commits.
const BranchPrefix = "observer/"

// BranchGlob matches every observation branch in branch-discovery filters.
const BranchGlob = BranchPrefix + "*"

// BranchName returns the branch that carries the observation with the given
// slug.
func BranchName(slug string) string {
	return BranchPrefix + slug
}

// BranchFromRef reduces a fully-qualified ref to its logical branch name:
// refs/heads/observer/x and refs/remotes/origin/observer/x both give
// observer/x. A ref is read from git, so this is a fact about where a file
// was found — never a name rebuilt from the file's contents.
func BranchFromRef(ref string) string {
	name := strings.TrimPrefix(ref, "refs/heads/")
	if rest, ok := strings.CutPrefix(name, "refs/remotes/"); ok {
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			name = rest[i+1:]
		}
	}
	return name
}

// SlugFromBranch extracts the slug from an observation branch name. ok is
// false when the name is not under the observer/ namespace.
func SlugFromBranch(branch string) (slug string, ok bool) {
	slug, ok = strings.CutPrefix(branch, BranchPrefix)
	if !ok || slug == "" {
		return "", false
	}
	return slug, true
}
