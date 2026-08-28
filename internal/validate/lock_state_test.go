package validate

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// lockCheck runs checkLockState over one assertion and returns its findings.
func lockCheck(status, lockedBy string) *Result {
	result := &Result{}
	checkLockState(parser.Assertion{
		File:     "specs/demo/assertions/a.md",
		Status:   status,
		LockedBy: lockedBy,
	}, result)
	return result
}

func lockAgedBy(d time.Duration) string {
	return fmt.Sprintf("builder-host-1-%d", time.Now().Add(-d).Unix())
}

// The old rule made a stale lock legal and therefore unreportable: it flagged a
// missing lock and a lock on a done assertion, but never one months old. A
// value whose tail is no timestamp is what a coach invented to satisfy that
// rule, and it cannot be told from a lock a crashed builder left.
func TestStaleLockIsWarned(t *testing.T) {
	for _, lockedBy := range []string{lockAgedBy(3 * time.Hour), "coach-invented-value"} {
		result := lockCheck("in_progress", lockedBy)
		if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "stale") {
			t.Errorf("lock %q must be reported stale, got: %v", lockedBy, result.Warnings)
		}
		if !result.Passed() {
			t.Errorf("a stale lock must not fail the tree, got: %v", result.Failures)
		}
	}
}

// A fresh lock means a builder holds the assertion now. An empty one is the
// legal unlocked state, not a stale lock, although IsLockStale("") is true.
func TestHeldAndUnlockedAssertionsAreSilent(t *testing.T) {
	for _, lockedBy := range []string{lockAgedBy(time.Minute), ""} {
		result := lockCheck("in_progress", lockedBy)
		if len(result.Warnings) != 0 {
			t.Errorf("lock %q must be silent, got: %v", lockedBy, result.Warnings)
		}
	}
}

// The reverse rule stays a failure: nothing legitimate leaves a lock behind on
// a settled assertion.
func TestLockOnASettledAssertionStillFails(t *testing.T) {
	for _, status := range []string{"done", "failed", "not_started", "draft"} {
		result := lockCheck(status, lockAgedBy(time.Minute))
		if result.Passed() {
			t.Errorf("status %s carrying a lock must fail", status)
		}
	}
}
