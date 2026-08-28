package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gitT runs a git command in dir, failing the test on error.
func gitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// chdirT switches the working directory for the test and restores it.
func chdirT(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

// newObserverRepo builds a temp repo on main with one observer branch
// carrying a valid open observation, and chdirs into it.
func newObserverRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	gitT(t, dir, "config", "user.email", "test@example.com")
	gitT(t, dir, "config", "user.name", "Test")
	gitT(t, dir, "config", "commit.gpgsign", "false")
	gitT(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	writeObservationBranch(t, dir, "existing-finding", "high", "internal/parser/parser.go")
	chdirT(t, dir)
	return dir
}

// writeObservationBranch commits a minimal valid observation on
// observer/<slug> and returns to main.
func writeObservationBranch(t *testing.T, dir, slug, severity, affected string) {
	t.Helper()
	content := "---\n" +
		"slug: " + slug + "\n" +
		"type: code_spec_misalignment\n" +
		"severity: " + severity + "\n" +
		"status: open\n" +
		"created: 2026-07-26T12:00:00Z\n" +
		"affected:\n  - " + affected + "\n" +
		"---\n\n# Finding " + slug + "\n\nEvidence body.\n"
	gitT(t, dir, "checkout", "-q", "-b", "observer/"+slug, "main")
	path := filepath.Join(dir, "observations", slug+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitT(t, dir, "add", ".")
	gitT(t, dir, "commit", "-q", "-m", "observer: add "+slug)
	gitT(t, dir, "checkout", "-q", "main")
}

func TestScanCheckCoveredAndClear(t *testing.T) {
	newObserverRepo(t)
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	run := func(slug, affected string) scanCheckResult {
		t.Helper()
		var out, errOut bytes.Buffer
		code := execObserverScanCheck([]string{
			"--type", "code_spec_misalignment",
			"--slug", slug,
			"--affected", affected,
		}, &out, &errOut, now)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		var res scanCheckResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("bad JSON: %v", err)
		}
		return res
	}

	// The same finding is claimed already. The branch is read from the ref
	// the observation was found at, so it names a branch that exists.
	res := run("existing-finding", "internal/parser/parser.go")
	if res.Result != "covered" || res.Slug != "existing-finding" {
		t.Fatalf("want covered by existing-finding, got %+v", res)
	}
	if res.Ref != "refs/heads/observer/existing-finding" || res.Branch != "observer/existing-finding" {
		t.Fatalf("branch must come from the ref, got %+v", res)
	}

	// A different finding in a file the claim also names is still a finding.
	res = run("new-finding", "internal/parser/parser.go,internal/other.go")
	if res.Result != "clear" || res.Slug != "new-finding" || res.Branch != "observer/new-finding" {
		t.Fatalf("a shared path is not the same finding, got %+v", res)
	}

	// Missing flags fail fast.
	var out, errOut bytes.Buffer
	if code := execObserverScanCheck(nil, &out, &errOut, now); code == 0 {
		t.Fatal("missing flags must exit non-zero")
	}
}

func TestScanCheckSlugCollisionWithMain(t *testing.T) {
	dir := newObserverRepo(t)

	// Merge an observation with slug taken onto main (resolved history),
	// then delete its branch: re-found drift reuses the slug with a suffix.
	content := "---\nslug: taken\ntype: outdated_specs\nseverity: low\nstatus: resolved\n" +
		"created: 2026-07-01T12:00:00Z\naffected:\n  - specs/foo/foo.md\n---\n\n# Taken\n"
	if err := os.MkdirAll(filepath.Join(dir, "observations"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "observations", "taken.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitT(t, dir, "add", ".")
	gitT(t, dir, "commit", "-q", "-m", "resolve taken")

	var out, errOut bytes.Buffer
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	code := execObserverScanCheck([]string{
		"--type", "outdated_specs",
		"--slug", "taken",
		"--affected", "specs/bar/bar.md",
	}, &out, &errOut, now)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var res scanCheckResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if res.Result != "clear" || res.Slug != "taken-20260727" {
		t.Fatalf("want dated suffix for slug taken on main, got %+v", res)
	}
}

func TestObserverDigestOutput(t *testing.T) {
	dir := newObserverRepo(t)
	writeObservationBranch(t, dir, "medium-finding", "medium", "internal/agent/agent.go")
	writeObservationBranch(t, dir, "low-finding", "low", "internal/cli/cli.go")

	var out, errOut bytes.Buffer
	if code := execObserverDigest([]string{"--json"}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var entries []struct {
		Slug     string `json:"slug"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	want := []string{"existing-finding", "medium-finding", "low-finding"}
	if len(entries) != len(want) {
		t.Fatalf("want %d entries, got %+v", len(want), entries)
	}
	for i, slug := range want {
		if entries[i].Slug != slug {
			t.Fatalf("digest order: got %+v", entries)
		}
	}

	// Human view mentions the branch and the affected paths.
	out.Reset()
	if code := execObserverDigest(nil, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	text := out.String()
	if !strings.Contains(text, "observer/existing-finding") || !strings.Contains(text, "internal/parser/parser.go") {
		t.Fatalf("table output incomplete:\n%s", text)
	}
}

// writeDontFlagOnMain commits a dont-flag file on main.
func writeDontFlagOnMain(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".spekk", "dont-flag.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitT(t, dir, "add", ".spekk/dont-flag.yaml")
	gitT(t, dir, "commit", "-q", "-m", "add dont-flag entries")
}

func TestScanCheckSuppression(t *testing.T) {
	dir := newObserverRepo(t)
	writeDontFlagOnMain(t, dir, `- match: "internal/legacy/**"
  reason: "Legacy package; drift is expected."
  by: "william"
  until: 2026-12-31
- match: "noisy-*"
  reason: "Accepted; see ADR-014."
  by: "william"
`)
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	var out, errOut bytes.Buffer
	run := func(slug, affected string) scanCheckResult {
		t.Helper()
		out.Reset()
		code := execObserverScanCheck([]string{
			"--type", "code_spec_misalignment", "--slug", slug, "--affected", affected,
		}, &out, &errOut, now)
		if code != 0 {
			t.Fatalf("exit %d: %s", code, errOut.String())
		}
		var res scanCheckResult
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("bad JSON: %v", err)
		}
		return res
	}

	// Path-glob suppression.
	res := run("legacy-drift", "internal/legacy/old.go")
	if res.Result != "suppressed" || res.By != "william" || res.Reason == "" {
		t.Fatalf("want suppressed with reason and author, got %+v", res)
	}
	// Slug-pattern suppression.
	if res := run("noisy-finding", "internal/other.go"); res.Result != "suppressed" {
		t.Fatalf("want slug suppression, got %+v", res)
	}
	// Unmatched drift proceeds.
	if res := run("real-drift", "internal/other.go"); res.Result != "clear" {
		t.Fatalf("want clear, got %+v", res)
	}

	// After expiry the dated entry stops suppressing.
	expired := time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)
	out.Reset()
	if code := execObserverScanCheck([]string{
		"--type", "code_spec_misalignment", "--slug", "legacy-drift", "--affected", "internal/legacy/old.go",
	}, &out, &errOut, expired); code != 0 {
		t.Fatalf("exit: %s", errOut.String())
	}
	if !strings.Contains(out.String(), `"clear"`) {
		t.Fatalf("expired entry must not suppress, got %s", out.String())
	}

	// The committed-on-main copy governs: a working-tree edit alone changes
	// nothing.
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, ".spekk", "dont-flag.yaml"),
		[]byte("- match: \"**\"\n  reason: \"r\"\n  by: \"w\"\n"), 0o644))
	if res := run("real-drift", "internal/other.go"); res.Result != "clear" {
		t.Fatalf("uncommitted suppression must not apply, got %+v", res)
	}
	gitT(t, dir, "checkout", "-q", "--", ".spekk/dont-flag.yaml")
}

func TestScanCheckMalformedDontFlagFailsLoudly(t *testing.T) {
	dir := newObserverRepo(t)
	writeDontFlagOnMain(t, dir, "- match: \"x/**\"\n  by: \"w\"\n")

	var out, errOut bytes.Buffer
	code := execObserverScanCheck([]string{
		"--type", "code_spec_misalignment", "--slug", "s", "--affected", "a.go",
	}, &out, &errOut, time.Now())
	if code == 0 {
		t.Fatal("malformed dont-flag file must fail the check")
	}
	if !strings.Contains(errOut.String(), "reason") || !strings.Contains(errOut.String(), "entry 1") {
		t.Fatalf("error must name the offending entry: %q", errOut.String())
	}
}

// TestScanCheckDatedRecurrenceIsCovered pins the case a recurrence creates.
// A slug already on main files under a -YYYYMMDD name, and the next scan
// proposes the plain slug again. Comparing whole slugs missed that live
// claim, so the tool answered "clear" with a branch that already existed
// and could not be created.
func TestScanCheckDatedRecurrenceIsCovered(t *testing.T) {
	dir := newObserverRepo(t)
	now := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)

	// A finding resolved long ago on main, and the same drift claimed again
	// today under the dated slug the tool itself hands out.
	writeObservationOnMain(t, dir, "old-drift", "resolved")
	writeObservationBranch(t, dir, "old-drift-20260828", "high", "internal/parser/parser.go")

	res := runScanCheck(t, now, "old-drift", "internal/parser/parser.go")
	if res.Result != "covered" || res.Slug != "old-drift-20260828" {
		t.Fatalf("the dated claim covers its own plain slug, got %+v", res)
	}
	if res.Branch != "observer/old-drift-20260828" {
		t.Fatalf("branch must name the claim that exists, got %+v", res)
	}
}

// TestScanCheckCoveredAtRemoteRef pins the ref-stripping rule at the CLI
// boundary. A local ref cannot catch it: there BranchFromRef and the old
// slug-derived name produce the same string.
func TestScanCheckCoveredAtRemoteRef(t *testing.T) {
	dir := t.TempDir()
	origin := filepath.Join(dir, "origin")
	if err := os.MkdirAll(origin, 0o755); err != nil {
		t.Fatal(err)
	}
	gitT(t, origin, "init", "-q", "-b", "main")
	gitT(t, origin, "config", "user.email", "test@example.com")
	gitT(t, origin, "config", "user.name", "Test")
	gitT(t, origin, "config", "commit.gpgsign", "false")
	gitT(t, origin, "commit", "-q", "--allow-empty", "-m", "init")
	writeObservationBranch(t, origin, "remote-claim", "high", "internal/parser/parser.go")

	clone := filepath.Join(dir, "clone")
	gitT(t, dir, "clone", "-q", origin, clone)
	chdirT(t, clone)

	res := runScanCheck(t, time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
		"remote-claim", "internal/parser/parser.go")
	if res.Result != "covered" {
		t.Fatalf("a claim on a remote-tracking ref still covers, got %+v", res)
	}
	if res.Branch != "observer/remote-claim" {
		t.Fatalf("branch must be the logical name, got %q", res.Branch)
	}
	if res.Ref != "refs/remotes/origin/observer/remote-claim" {
		t.Fatalf("ref must be fully qualified, got %q", res.Ref)
	}
}

// TestScanCheckRejectsAMalformedSlug pins the gate. The slug is the dedup
// identity and the observation's own name, so one the format rejects would
// file an observation the union skips forever, with no failure anybody sees.
func TestScanCheckRejectsAMalformedSlug(t *testing.T) {
	newObserverRepo(t)
	var out, errOut bytes.Buffer
	code := execObserverScanCheck([]string{
		"--type", "code_spec_misalignment",
		"--slug", "New_Finding!",
		"--affected", "internal/parser/parser.go",
	}, &out, &errOut, time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC))
	if code == 0 {
		t.Fatalf("a malformed slug must fail, got exit 0 and %s", out.String())
	}
	if !strings.Contains(errOut.String(), "kebab-case") {
		t.Fatalf("the error must name the rule, got %q", errOut.String())
	}
}

// runScanCheck runs the command and decodes its JSON result.
func runScanCheck(t *testing.T, now time.Time, slug, affected string) scanCheckResult {
	t.Helper()
	var out, errOut bytes.Buffer
	code := execObserverScanCheck([]string{
		"--type", "code_spec_misalignment",
		"--slug", slug,
		"--affected", affected,
	}, &out, &errOut, now)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var res scanCheckResult
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	return res
}

// writeObservationOnMain commits an observation directly on main, which is
// how a resolved finding looks after its remedy merged.
func writeObservationOnMain(t *testing.T, dir, slug, status string) {
	t.Helper()
	content := "---\nslug: " + slug + "\ntype: code_spec_misalignment\n" +
		"severity: high\nstatus: " + status + "\ncreated: 2026-07-26T12:00:00Z\n" +
		"affected:\n  - internal/parser/parser.go\n---\n\n# Finding " + slug + "\n"
	gitT(t, dir, "checkout", "-q", "main")
	path := filepath.Join(dir, "observations", slug+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitT(t, dir, "add", ".")
	gitT(t, dir, "commit", "-q", "-m", "resolve "+slug)
}
