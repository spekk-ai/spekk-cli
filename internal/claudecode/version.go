// Package claudecode holds the oldest Claude Code version that a sandbox
// runs, and the comparison against it.
package claudecode

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MinVersion is the oldest Claude Code that a sandbox runs. It is the oldest
// version that keeps a `claude -p` process alive while a background task
// runs, and reports again when the task ends. 2.1.291 and older kill the
// task seconds after the result and exit. The measurements are in
// specs/sandbox-claude-code-version/.
const MinVersion = "2.1.292"

// versionRe matches the version at the start of a line of `claude --version`
// output, such as "2.1.301 (Claude Code)".
var versionRe = regexp.MustCompile(`(?m)^(\d+)\.(\d+)\.(\d+)`)

// ParseVersion returns the version in the output of `claude --version`. Other
// lines, such as an ssh warning, are skipped.
func ParseVersion(out string) (string, error) {
	m := versionRe.FindString(out)
	if m == "" {
		return "", fmt.Errorf("no Claude Code version in %q", strings.TrimSpace(out))
	}
	return m, nil
}

// BelowMinimum reports whether version is older than MinVersion. version
// comes from ParseVersion.
func BelowMinimum(version string) bool {
	return compare(version, MinVersion) < 0
}

// compare compares two X.Y.Z versions part by part, as numbers.
func compare(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range pa {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
