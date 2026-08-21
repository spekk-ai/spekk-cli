package validate

import (
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// assertion builds the only three fields checkBranchRefs reads.
func assertion(id, branch, status string) parser.Assertion {
	return parser.Assertion{ID: id, Branch: branch, Status: status}
}

// warn runs the check over one ref set and returns the warning lines.
func warn(refs []string, assertions ...parser.Assertion) []string {
	result := &Result{}
	checkBranchRefs(assertions, refs, result)
	return result.Warnings
}

// An assertion still in the queue whose branch names nothing is invisible to
// spekk next for ever. Whether a typo or a deletion put the value there, the
// assertion is stranded either way.
func TestStrandedAssertionIsReported(t *testing.T) {
	got := warn([]string{"main", "feat/retry"}, assertion("a", "feat/retryy", "not_started"))
	if len(got) != 1 || !strings.Contains(got[0], `"feat/retryy"`) {
		t.Fatalf("a queue-visible assertion on an absent branch must be reported, got: %v", got)
	}
}

// A done assertion on a deleted branch is the normal end state of merged work,
// and a draft one is out of the queue by choice. Neither is stranded.
func TestSettledAssertionIsNotReported(t *testing.T) {
	got := warn([]string{"main"},
		assertion("a", "feat/merged-and-deleted", "done"),
		assertion("b", "feat/merged-and-deleted", "draft"),
		assertion("c", "main", "not_started"),
	)
	if len(got) != 0 {
		t.Errorf("done, draft, and existing-branch assertions must be silent, got: %v", got)
	}
}

// The parser defaults an absent branch field to "main", so on a repository
// whose trunk is master every assertion without the field carries the same
// wrong value. Grouped, that is one line instead of one per assertion.
func TestOneLinePerDistinctValueWithACount(t *testing.T) {
	got := warn([]string{"master"},
		assertion("a", "main", "not_started"),
		assertion("b", "main", "in_progress"),
		assertion("c", "main", "failed"),
		assertion("d", "gone", "not_started"),
	)
	if len(got) != 2 {
		t.Fatalf("expected one line per distinct value, got: %v", got)
	}
	// Sorted by value, so "gone" precedes "main".
	if !strings.Contains(got[0], "(1 assertion not done)") {
		t.Errorf("a single assertion must read in the singular, got: %q", got[0])
	}
	if !strings.Contains(got[1], "(3 assertions not done)") {
		t.Errorf("three assertions must be counted and pluralized, got: %q", got[1])
	}
}

// No refs means git is absent or this tree is in no repository. Neither is a
// problem with the specs, so validate must stay quiet rather than report every
// branch value as missing.
func TestNoRefsReportsNothing(t *testing.T) {
	if got := warn(nil, assertion("a", "feat/anything", "not_started")); len(got) != 0 {
		t.Errorf("an empty ref set must report nothing, got: %v", got)
	}
}
