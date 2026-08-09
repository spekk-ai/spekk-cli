package dontflag

import (
	"strings"
	"testing"
	"time"
)

const sampleFile = `# .spekk/dont-flag.yaml
- match: "internal/legacy/**"
  reason: "Legacy package scheduled for deletion in Q4; drift is expected."
  by: "william"
  until: 2026-12-31
- match: "parser-drops-*"
  reason: "Known parser looseness, accepted; see ADR-014."
  by: "william"
`

func TestParseValidFile(t *testing.T) {
	entries, err := Parse(sampleFile)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Match != "internal/legacy/**" || entries[0].Until != "2026-12-31" {
		t.Fatalf("entry 0: %+v", entries[0])
	}
	if entries[1].By != "william" || entries[1].Until != "" {
		t.Fatalf("entry 1: %+v", entries[1])
	}
}

func TestParseRejectsMalformedEntries(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantErr string
	}{
		{"missing reason", "- match: \"x/**\"\n  by: \"w\"\n", "reason"},
		{"missing by", "- match: \"x/**\"\n  reason: \"r\"\n", "'by'"},
		{"missing match", "- reason: \"r\"\n  by: \"w\"\n", "match"},
		{"bad until", "- match: \"x\"\n  reason: \"r\"\n  by: \"w\"\n  until: soon\n", "until"},
		{"unknown key", "- match: \"x\"\n  reason: \"r\"\n  by: \"w\"\n  untill: 2026-01-01\n", "unknown field"},
		{"stray line", "reason: \"r\"\n", "outside any entry"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.content)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error mentioning %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestEntryMatching(t *testing.T) {
	cases := []struct {
		name     string
		match    string
		slug     string
		affected []string
		want     bool
	}{
		{"path glob deep", "internal/legacy/**", "some-slug", []string{"internal/legacy/a/b.go"}, true},
		{"path glob direct child", "internal/legacy/**", "some-slug", []string{"internal/legacy/x.go"}, true},
		{"path glob miss", "internal/legacy/**", "some-slug", []string{"internal/parser/x.go"}, false},
		{"slug pattern", "parser-drops-*", "parser-drops-draft-status", []string{"internal/other.go"}, true},
		{"slug pattern miss", "parser-drops-*", "index-skips-drafts", []string{"internal/other.go"}, false},
		{"single star stays in segment", "internal/*", "s", []string{"internal/a/b.go"}, false},
		{"single star same segment", "internal/*", "s", []string{"internal/a.go"}, true},
		{"exact path", "go.mod", "s", []string{"go.mod"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := Entry{Match: tc.match, Reason: "r", By: "w"}
			if got := e.Matches(tc.slug, tc.affected); got != tc.want {
				t.Fatalf("Matches(%q, %v) with %q = %v, want %v", tc.slug, tc.affected, tc.match, got, tc.want)
			}
		})
	}
}

func TestUntilExpiresEndOfDayUTC(t *testing.T) {
	e := Entry{Match: "x", Reason: "r", By: "w", Until: "2026-12-31"}

	during := time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC)
	if !e.ActiveAt(during) {
		t.Fatal("entry must stay active through the whole until day (end-of-day UTC)")
	}
	after := time.Date(2027, 1, 1, 0, 0, 1, 0, time.UTC)
	if e.ActiveAt(after) {
		t.Fatal("entry must expire after the until day ends (UTC)")
	}

	permanent := Entry{Match: "x", Reason: "r", By: "w"}
	if !permanent.ActiveAt(after) {
		t.Fatal("entry without until is permanent")
	}
}

func TestSuppressed(t *testing.T) {
	entries, err := Parse(sampleFile)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)

	if e := Suppressed(entries, "legacy-drift", []string{"internal/legacy/old.go"}, now); e == nil || e.By != "william" {
		t.Fatalf("path-glob suppression missed: %+v", e)
	}
	if e := Suppressed(entries, "parser-drops-draft", []string{"internal/parser/p.go"}, now); e == nil {
		t.Fatal("slug-pattern suppression missed")
	}
	if e := Suppressed(entries, "fresh-drift", []string{"internal/index/index.go"}, now); e != nil {
		t.Fatalf("unexpected suppression: %+v", e)
	}
	// An expired entry suppresses nothing: the next scan may re-flag.
	expired := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	if e := Suppressed(entries, "legacy-drift", []string{"internal/legacy/old.go"}, expired); e != nil {
		t.Fatalf("expired entry must not suppress: %+v", e)
	}
	// The slug-pattern entry has no until: still active.
	if e := Suppressed(entries, "parser-drops-draft", nil, expired); e == nil {
		t.Fatal("permanent entry must outlive the dated one")
	}
}

// A suppression is an explicit human decision, so it must not be defeated by
// the spelling of the path a scan happens to report. This is stricter than
// dedup: suppressed drift never becomes an observation, so nothing lands on a
// branch to cover it next time -- a missed suppression files the observation
// its author asked never to see again, on every run.
func TestSuppressionMatchesHoweverThePathIsSpelled(t *testing.T) {
	entries := []Entry{{Match: "internal/foo.go", Reason: "accepted", By: "someone"}}
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	for _, spelling := range []string{
		"internal/foo.go",
		"internal/foo.go:42",
		"internal/foo.go:42:7",
		"./internal/foo.go",
		"internal//foo.go",
	} {
		if Suppressed(entries, "some-slug", []string{spelling}, now) == nil {
			t.Errorf("%q was not suppressed; the observation would be filed anyway", spelling)
		}
	}

	if Suppressed(entries, "some-slug", []string{"internal/bar.go"}, now) != nil {
		t.Error("an unrelated file must not be suppressed")
	}
}

// Glob suppressions keep working over normalized paths.
func TestSuppressionGlobStillMatchesAfterNormalization(t *testing.T) {
	entries := []Entry{{Match: "internal/**", Reason: "legacy", By: "someone"}}
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	if Suppressed(entries, "s", []string{"./internal/deep/x.go:9"}, now) == nil {
		t.Error("glob suppression must survive a location suffix and a ./ prefix")
	}
}

// The contract, stated exactly. A pattern written the ordinary way matches
// every spelling of the path. A pattern written an odd way still matches its
// own spelling -- it does not go dead, which is what normalizing only the
// candidate did to it. Two differently-spelled patterns are NOT made
// equivalent: that would mean rewriting the pattern, and rewriting a glob
// with a path function is what let `docs/../**` suppress everything.
func TestSuppressionPathSpellingContract(t *testing.T) {
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	ordinary := []Entry{{Match: "internal/foo.go", Reason: "r", By: "b"}}
	for _, spelling := range []string{
		"internal/foo.go",
		"./internal/foo.go",
		"internal/foo.go:42",
		"internal/foo.go:42:7",
		"internal//foo.go",
	} {
		if Suppressed(ordinary, "s", []string{spelling}, now) == nil {
			t.Errorf("an ordinary pattern did not suppress %q", spelling)
		}
	}

	// An odd pattern still matches what it literally names. Before, a
	// normalized candidate could not match it at all.
	odd := []Entry{{Match: "./internal/foo.go", Reason: "r", By: "b"}}
	if Suppressed(odd, "s", []string{"./internal/foo.go"}, now) == nil {
		t.Error("an oddly spelled pattern went dead against its own spelling")
	}

	// Nothing here widens onto a different file.
	if Suppressed(ordinary, "s", []string{"internal/foobar.go"}, now) != nil {
		t.Error("an unrelated file was suppressed")
	}
}

// The pattern is never rewritten. It is a glob, not a path, so cleaning it
// as a path is unsound: path.Clean resolves `..`, and the location-suffix
// strip removes a trailing `:digits`. Both turn a pattern that reads as
// narrow into one that matches everything. This file is trusted because a
// person read it, so a pattern must mean what it says.
func TestSuppressionNeverWidensThePattern(t *testing.T) {
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	unrelated := []string{"cmd/spekk/main.go", "secrets/key.pem", "anything/at/all.go"}
	for _, pattern := range []string{
		"docs/../**",             // path.Clean would make this **
		"internal/../secrets/**", // path.Clean would retarget this
		"**:1",                   // the suffix strip would make this **
		"*:80",                   // and this *
	} {
		entries := []Entry{{Match: pattern, Reason: "r", By: "b"}}
		for _, p := range unrelated {
			if e := Suppressed(entries, "s", []string{p}, now); e != nil {
				t.Errorf("pattern %q suppressed the unrelated path %q", pattern, p)
			}
		}
	}
}

// Normalization runs on the path only, so it does not have to be idempotent.
// Comparing a normalized pattern against a normalized path would have needed
// that, and the bounded location strip does not provide it.
func TestSuppressionDoesNotDependOnIdempotentNormalization(t *testing.T) {
	now := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	entries := []Entry{{Match: "a:1", Reason: "r", By: "b"}}

	for _, spelling := range []string{"a:1", "a:1:2:3"} {
		if Suppressed(entries, "s", []string{spelling}, now) == nil {
			t.Errorf("pattern %q did not suppress %q", "a:1", spelling)
		}
	}
}
