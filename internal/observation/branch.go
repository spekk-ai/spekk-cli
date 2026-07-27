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

// SlugFromBranch extracts the slug from an observation branch name. ok is
// false when the name is not under the observer/ namespace.
func SlugFromBranch(branch string) (slug string, ok bool) {
	slug, ok = strings.CutPrefix(branch, BranchPrefix)
	if !ok || slug == "" {
		return "", false
	}
	return slug, true
}
