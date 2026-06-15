package crossbranch

import (
	"fmt"
	"strconv"
	"strings"
)

// mergeTreeFloor is the minimum git version that supports the real-merge mode
// of `git merge-tree --write-tree` (introduced in git 2.38). At or above this
// version, cross-branch mode can detect true merge conflicts; below it, the
// feature degrades to classification-only (add/modify/delete) with no honest
// conflict confirmation.
const (
	mergeTreeFloorMajor = 2
	mergeTreeFloorMinor = 38
)

// GitVersion detects the installed git version by parsing the output of
// `git --version`. It returns the major and minor components.
//
// Errors propagate from the underlying exec (git missing / not runnable) or
// from an unparseable version string. Callers that want graceful degradation
// rather than a hard error should prefer SupportsMergeTree, which treats any
// detection failure as "conflict detection unavailable".
func GitVersion() (major, minor int, err error) {
	out, err := Run("--version")
	if err != nil {
		return 0, 0, err
	}
	return parseGitVersion(out)
}

// parseGitVersion extracts the major and minor version numbers from a
// `git --version` output line. It is a pure function (no exec) so the parsing
// and comparison logic can be unit-tested directly with injected strings.
//
// It tolerates vendor/distribution suffixes, e.g.:
//
//	"git version 2.43.0"                  -> 2, 43
//	"git version 2.39.3 (Apple Git-145)"  -> 2, 39
//	"git version 2.30.windows.1"          -> 2, 30
//
// The patch component and anything after it are ignored. If no numeric
// "major.minor" token can be found, an error is returned so the caller can
// decide to degrade.
func parseGitVersion(s string) (major, minor int, err error) {
	fields := strings.Fields(s)
	for _, f := range fields {
		// Find the first field that begins "<digits>.<digits>". This skips the
		// "git"/"version" words and any leading non-numeric tokens, and ignores
		// trailing suffixes like ".0", ".windows.1", or "-rc0".
		maj, min, ok := splitMajorMinor(f)
		if ok {
			return maj, min, nil
		}
	}
	return 0, 0, fmt.Errorf("crossbranch: could not parse git version from %q", s)
}

// splitMajorMinor parses a single token of the form "MAJOR.MINOR[.…]" or
// "MAJOR.MINOR[suffix]" into its leading major and minor integers. It returns
// ok=false if the token does not start with two dot-separated numeric runs.
func splitMajorMinor(token string) (major, minor int, ok bool) {
	dot := strings.IndexByte(token, '.')
	if dot <= 0 {
		return 0, 0, false
	}
	maj, err := strconv.Atoi(token[:dot])
	if err != nil {
		return 0, 0, false
	}
	rest := token[dot+1:]
	// The minor component runs up to the next '.', or a non-digit suffix
	// (e.g. "30.windows.1" -> minor "30"; "38-rc0" handled by trimming).
	minorRun := leadingDigits(rest)
	if minorRun == "" {
		return 0, 0, false
	}
	min, err := strconv.Atoi(minorRun)
	if err != nil {
		return 0, 0, false
	}
	return maj, min, true
}

// leadingDigits returns the maximal prefix of s consisting of ASCII digits.
func leadingDigits(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i]
}

// atLeastMergeTree reports whether the given major/minor satisfies the
// git >= 2.38 floor required for `merge-tree --write-tree` real-merge mode.
func atLeastMergeTree(major, minor int) bool {
	if major != mergeTreeFloorMajor {
		return major > mergeTreeFloorMajor
	}
	return minor >= mergeTreeFloorMinor
}

// SupportsMergeTree reports whether the installed git is new enough
// (>= 2.38) to use `git merge-tree --write-tree` for honest conflict
// detection.
//
// It is designed for graceful degradation: it returns (false, nil) — not an
// error — when git is present but too old. A non-nil error is returned only
// when git itself could not be run at all (e.g. not installed). When git runs
// but its version string cannot be parsed, it conservatively reports
// (false, nil) so the caller degrades to classification-only rather than
// failing outright (per the assertion's "unparseable -> degrade, not error"
// requirement).
//
// The classify wave should branch on this: true => confirm conflicts via
// merge-tree; false => classification-only mode and surface the limitation to
// the user.
func SupportsMergeTree() (bool, error) {
	out, err := Run("--version")
	if err != nil {
		// git could not be run at all — this is a genuine error, not merely an
		// old git. Report it so the caller can decide (cross-branch mode needs
		// git regardless of merge-tree support).
		return false, err
	}
	major, minor, perr := parseGitVersion(out)
	if perr != nil {
		// Present but unparseable version => degrade, do not error.
		return false, nil
	}
	return atLeastMergeTree(major, minor), nil
}
