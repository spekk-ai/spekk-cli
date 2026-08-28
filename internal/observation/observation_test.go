package observation

import (
	"strings"
	"testing"
	"time"
)

// validObs builds a well-formed observation file, letting tests override
// single frontmatter lines.
func validObs(overrides map[string]string) string {
	fields := map[string]string{
		"slug":     "parser-drops-draft-status",
		"type":     "code_spec_misalignment",
		"severity": "high",
		"status":   "open",
		"created":  "2026-07-26T12:00:00Z",
	}
	for k, v := range overrides {
		fields[k] = v
	}
	var b strings.Builder
	b.WriteString("---\n")
	for _, k := range []string{"slug", "type", "severity", "status", "created", "announced", "pr"} {
		if v, ok := fields[k]; ok && v != "" {
			b.WriteString(k + ": " + v + "\n")
		} else if ok && v == "@empty" {
			b.WriteString(k + ":\n")
		}
	}
	if fields["affected"] != "@none" {
		b.WriteString("affected:\n  - internal/parser/parser.go\n  - specs/spec-validation/assertions/draft-excluded.md\n")
	}
	b.WriteString("---\n\n# Parser Drops Draft Status\n\nBody text.\n")
	return b.String()
}

func TestParseValid(t *testing.T) {
	o, err := Parse("observations/parser-drops-draft-status.md", validObs(nil))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if o.Slug != "parser-drops-draft-status" || o.Type != TypeCodeSpecMisalignment ||
		o.Severity != SeverityHigh || o.Status != StatusOpen {
		t.Fatalf("unexpected fields: %+v", o)
	}
	if o.Announced != "" {
		t.Fatalf("announced should be empty when absent, got %q", o.Announced)
	}
	if len(o.Affected) != 2 || o.Affected[0] != "internal/parser/parser.go" {
		t.Fatalf("affected not parsed: %v", o.Affected)
	}
	if o.Title != "Parser Drops Draft Status" {
		t.Fatalf("title: %q", o.Title)
	}
}

func TestParseValidation(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[string]string
		wantErr   string
	}{
		{"announced present valid", map[string]string{"announced": "2026-07-26T13:05:00Z"}, ""},
		{"pr optional", map[string]string{"pr": "https://github.com/org/repo/pull/7"}, ""},
		{"missing slug", map[string]string{"slug": "@empty"}, "slug"},
		{"bad slug", map[string]string{"slug": "Not_Kebab"}, "kebab-case"},
		{"bad type", map[string]string{"type": "surprise"}, "type"},
		{"bad severity", map[string]string{"severity": "urgent"}, "severity"},
		{"bad status", map[string]string{"status": "parked"}, "status"},
		{"bad created", map[string]string{"created": "yesterday"}, "created"},
		// Present-but-empty announced is invalid, not either state.
		{"empty announced", map[string]string{"announced": "@empty"}, "announced"},
		{"bad announced", map[string]string{"announced": "true"}, "announced"},
		// Evidence gate: no affected paths, no observation.
		{"no evidence", map[string]string{"affected": "@none"}, "affected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("observations/x.md", validObs(tc.overrides))
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want error mentioning %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestParseCustomFields(t *testing.T) {
	extra := "status: open\n" +
		"skill: observer-prune\n" +
		"tags: [provenance, \"drift, recurring\"]\n" +
		"runs:\n  - 2026-01-01\n  - 2026-02-01\n" +
		"# comment: not-a-field\n" +
		"note: |\n"
	content := strings.Replace(validObs(nil), "status: open\n", extra, 1)
	o, err := Parse("observations/x.md", content)
	if err != nil {
		t.Fatalf("custom fields must not break parsing: %v", err)
	}
	if o.Status != StatusOpen {
		t.Fatalf("status: %q", o.Status)
	}

	want := map[string][]string{
		"skill": {"observer-prune"},
		"tags":  {"provenance", "drift, recurring"},
		"runs":  {"2026-01-01", "2026-02-01"},
	}
	if len(o.Fields) != len(want) {
		t.Fatalf("custom fields: got %v, want %v", o.Fields, want)
	}
	for key, values := range want {
		if strings.Join(o.Fields[key], "|") != strings.Join(values, "|") {
			t.Fatalf("field %q: got %v, want %v", key, o.Fields[key], values)
		}
	}
	// A lifecycle key is not a custom field; affected in particular stays
	// the evidence gate, and observation_files is its only table.
	for _, key := range []string{"slug", "type", "severity", "status", "created", "affected"} {
		if _, ok := o.Fields[key]; ok {
			t.Fatalf("known key %q must not appear as a custom field", key)
		}
	}
}

func TestMarkAnnounced(t *testing.T) {
	content := validObs(nil)
	marked, err := MarkAnnounced(content, "2026-07-27T09:00:00Z")
	if err != nil {
		t.Fatalf("MarkAnnounced: %v", err)
	}
	o, err := Parse("observations/x.md", marked)
	if err != nil {
		t.Fatalf("marked content must stay valid: %v", err)
	}
	if o.Announced != "2026-07-27T09:00:00Z" {
		t.Fatalf("announced: %q", o.Announced)
	}
	// The flip is exactly one added line.
	if len(strings.Split(marked, "\n")) != len(strings.Split(content, "\n"))+1 {
		t.Fatalf("expected a one-line insertion")
	}
	// Idempotent replace: marking again updates in place, adds nothing.
	again, err := MarkAnnounced(marked, "2026-07-27T10:00:00Z")
	if err != nil {
		t.Fatalf("MarkAnnounced again: %v", err)
	}
	if len(strings.Split(again, "\n")) != len(strings.Split(marked, "\n")) {
		t.Fatalf("second mark must replace, not insert")
	}
	if _, err := MarkAnnounced(content, "not-a-timestamp"); err == nil {
		t.Fatal("bad timestamp must be rejected")
	}
}

func TestBranchNameRoundTrip(t *testing.T) {
	if BranchName("my-slug") != "observer/my-slug" {
		t.Fatalf("BranchName: %q", BranchName("my-slug"))
	}
	slug, ok := SlugFromBranch("observer/my-slug")
	if !ok || slug != "my-slug" {
		t.Fatalf("SlugFromBranch: %q %v", slug, ok)
	}
	if _, ok := SlugFromBranch("feature/my-slug"); ok {
		t.Fatal("non-observer branch must not yield a slug")
	}
	if _, ok := SlugFromBranch("observer/"); ok {
		t.Fatal("empty slug must not be ok")
	}
	for ref, want := range map[string]string{
		"refs/heads/observer/my-slug":            "observer/my-slug",
		"refs/remotes/origin/observer/my-slug":   "observer/my-slug",
		"refs/remotes/upstream/observer/my-slug": "observer/my-slug",
		"refs/heads/main":                        "main",
	} {
		if got := BranchFromRef(ref); got != want {
			t.Errorf("BranchFromRef(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestResolveSlug(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	onMain := func(s string) bool { return s == "taken" }
	if got := ResolveSlug("fresh", onMain, now); got != "fresh" {
		t.Fatalf("fresh slug must be reused: %q", got)
	}
	if got := ResolveSlug("taken", onMain, now); got != "taken-20260727" {
		t.Fatalf("colliding slug must get a dated suffix: %q", got)
	}
}

func TestDigestSemantics(t *testing.T) {
	mk := func(slug, severity, status, created, ref string) *Observation {
		return &Observation{
			Slug: slug, Type: TypeCodeSpecMisalignment, Severity: severity,
			Status: status, Created: created, Affected: []string{"a.go"}, Ref: ref,
		}
	}
	u := &Union{Observations: []*Observation{
		mk("low-1", SeverityLow, StatusOpen, "2026-01-01T00:00:00Z", "refs/heads/observer/low-1"),
		mk("med-old", SeverityMedium, StatusOpen, "2026-01-01T00:00:00Z", "refs/heads/observer/med-old"),
		mk("med-new", SeverityMedium, StatusOpen, "2026-02-01T00:00:00Z", "refs/heads/observer/med-new"),
		mk("high-1", SeverityHigh, StatusOpen, "2026-03-01T00:00:00Z", "refs/heads/observer/high-1"),
		mk("dismissed", SeverityHigh, StatusDismissed, "2026-01-01T00:00:00Z", "refs/heads/observer/dismissed"),
		// Present on main: effectively not open, whatever the branch row says.
		mk("merged", SeverityHigh, StatusOpen, "2026-01-01T00:00:00Z", "refs/heads/observer/merged"),
		mk("merged", SeverityHigh, StatusResolved, "2026-01-01T00:00:00Z", "refs/heads/main"),
		mk("low-2", SeverityLow, StatusOpen, "2026-01-02T00:00:00Z", "refs/heads/observer/low-2"),
		// An inherited copy at another finding's branch is not a claim, so
		// the digest never shows a finding whose own branch is gone.
		mk("inherited", SeverityHigh, StatusOpen, "2026-01-01T00:00:00Z", "refs/heads/observer/low-2"),
		mk("high-2", SeverityHigh, StatusOpen, "2026-01-01T00:00:00Z", "refs/heads/observer/high-2"),
	}}

	got := u.Digest()
	want := []string{"high-2", "high-1", "med-old", "med-new", "low-1"}
	if len(got) != len(want) {
		t.Fatalf("digest length: got %d want %d", len(got), len(want))
	}
	for i, slug := range want {
		if got[i].Slug != slug {
			t.Fatalf("digest[%d]: got %q want %q (full: %v)", i, got[i].Slug, slug, slugs(got))
		}
	}
	if !u.OnMain("merged") || u.OnMain("high-1") {
		t.Fatal("OnMain misreports")
	}
}

func slugs(obs []*Observation) []string {
	out := make([]string, len(obs))
	for i, o := range obs {
		out[i] = o.Slug
	}
	return out
}
