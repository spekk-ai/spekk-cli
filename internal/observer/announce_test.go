package observer

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

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

func obsContent(slug, severity string) string {
	return `---
slug: ` + slug + `
type: code_spec_misalignment
severity: ` + severity + `
status: open
created: 2026-07-26T12:00:00Z
affected:
  - internal/parser/parser.go
---

# Finding ` + slug + `

## Issue Description

The parser drops draft assertions. The spec requires them to be listed. This is confirmed drift.

## Evidence

internal/parser/parser.go line 42.
`
}

// newAnnounceRepos builds a bare origin and a working clone (with a minimal
// specs tree on main), chdirs into the clone, and returns (cloneDir,
// originDir).
func newAnnounceRepos(t *testing.T) (string, string) {
	t.Helper()
	origin := filepath.Join(t.TempDir(), "origin.git")
	gitT(t, t.TempDir(), "init", "-q", "--bare", "-b", "main", origin)

	clone := t.TempDir()
	gitT(t, clone, "init", "-q", "-b", "main")
	gitT(t, clone, "config", "user.email", "test@example.com")
	gitT(t, clone, "config", "user.name", "Test")
	gitT(t, clone, "config", "commit.gpgsign", "false")
	gitT(t, clone, "remote", "add", "origin", origin)

	specDir := filepath.Join(clone, "specs", "spec-a")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "spec-a.md"), []byte(`---
id: spec-a
created: 2026-01-01T00:00:00Z
priority: 1
---
# Spec A
`), 0o644); err != nil {
		t.Fatal(err)
	}
	gitT(t, clone, "add", ".")
	gitT(t, clone, "commit", "-q", "-m", "init specs")
	gitT(t, clone, "push", "-q", "-u", "origin", "main")
	chdirT(t, clone)
	return clone, origin
}

// addObserverBranch commits an observation on observer/<slug>, optionally
// pushes it to origin, and returns to main.
func addObserverBranch(t *testing.T, dir, slug, severity string, push bool) {
	t.Helper()
	gitT(t, dir, "checkout", "-q", "-b", "observer/"+slug, "main")
	path := filepath.Join(dir, "observations", slug+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(obsContent(slug, severity)), 0o644); err != nil {
		t.Fatal(err)
	}
	gitT(t, dir, "add", ".")
	gitT(t, dir, "commit", "-q", "-m", "observer: add "+slug)
	if push {
		gitT(t, dir, "push", "-q", "-u", "origin", "observer/"+slug)
	}
	gitT(t, dir, "checkout", "-q", "main")
}

// fixedNow pins the announce clock.
var fixedNow = time.Date(2026, 7, 27, 9, 0, 0, 0, time.UTC)

func announceOpts(t *testing.T, dir, spool string) AnnounceOptions {
	t.Helper()
	return AnnounceOptions{
		Dir: dir,
		Getenv: func(key string) string {
			if key == conversation.SpoolEnvVar {
				return spool
			}
			return ""
		},
		Now: func() time.Time { return fixedNow },
	}
}

// readSpoolRequest reads the single finalized request file in spool.
func readSpoolRequest(t *testing.T, spool string) conversation.Request {
	t.Helper()
	entries, err := os.ReadDir(spool)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), conversation.RequestFileExt) {
			files = append(files, e.Name())
		}
	}
	if len(files) != 1 {
		t.Fatalf("expected exactly 1 request file, got %v", files)
	}
	data, err := os.ReadFile(filepath.Join(spool, files[0]))
	if err != nil {
		t.Fatal(err)
	}
	var req conversation.Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("bad request JSON: %v", err)
	}
	return req
}

func TestAnnounceSuccessDeliversAndMarks(t *testing.T) {
	clone, origin := newAnnounceRepos(t)
	addObserverBranch(t, clone, "finding-a", "high", true)
	spool := t.TempDir()

	code := Announce(announceOpts(t, clone, spool))
	if code != 0 {
		t.Fatalf("announce exit %d", code)
	}

	// The conversation request has the fixed message shape.
	req := readSpoolRequest(t, spool)
	if req.Title != "Finding finding-a" {
		t.Fatalf("title: %q", req.Title)
	}
	if req.Severity != conversation.SeverityCritical {
		t.Fatalf("severity: %q", req.Severity)
	}
	for _, want := range []string{
		"The parser drops draft assertions.",
		"Evidence: internal/parser/parser.go",
		"Proposed fix in PR: observer/finding-a — merge to accept, close to dismiss. Reply here to discuss.",
		"Severity: high",
	} {
		if !strings.Contains(req.Body, want) {
			t.Fatalf("body missing %q:\n%s", want, req.Body)
		}
	}

	// The flip commit is on origin: exactly one commit on top, exactly one
	// file changed, and the frontmatter carries announced.
	content := gitT(t, origin, "show", "refs/heads/observer/finding-a:observations/finding-a.md")
	if !strings.Contains(content, "announced: 2026-07-27T09:00:00Z") {
		t.Fatalf("origin frontmatter lacks announced:\n%s", content)
	}
	msg := gitT(t, origin, "log", "-1", "--format=%s", "refs/heads/observer/finding-a")
	if msg != "observer: mark finding-a announced" {
		t.Fatalf("flip commit message: %q", msg)
	}
	changed := gitT(t, origin, "show", "--name-only", "--format=", "refs/heads/observer/finding-a")
	if strings.TrimSpace(changed) != "observations/finding-a.md" {
		t.Fatalf("flip commit must change only the observation file, got %q", changed)
	}

	// The local branch fast-forwarded to the flip commit.
	localTip := gitT(t, clone, "rev-parse", "refs/heads/observer/finding-a")
	originTip := gitT(t, origin, "rev-parse", "refs/heads/observer/finding-a")
	if localTip != originTip {
		t.Fatalf("local branch %s != origin %s", localTip, originTip)
	}

	// Success writes no failure log.
	if _, err := os.Stat(filepath.Join(clone, ".spekk", "observer-conversation.log")); !os.IsNotExist(err) {
		t.Fatal("successful announce must not write the failure log")
	}

	// Idempotent retry: a second run finds nothing to announce and stays
	// silent (exit 0, no new request, no log).
	code = Announce(announceOpts(t, clone, spool))
	if code != 0 {
		t.Fatalf("second announce exit %d", code)
	}
	_ = readSpoolRequest(t, spool) // still exactly one request file
}

func TestAnnounceBatchesEligibleFindingsIntoOneMessage(t *testing.T) {
	clone, origin := newAnnounceRepos(t)
	addObserverBranch(t, clone, "medium-first", "medium", true)
	addObserverBranch(t, clone, "high-later", "high", true)
	// Low never announces, whatever the queue looks like.
	addObserverBranch(t, clone, "low-noise", "low", true)
	spool := t.TempDir()

	if code := Announce(announceOpts(t, clone, spool)); code != 0 {
		t.Fatal("announce failed")
	}

	// ONE request carries both findings, high first, low absent.
	req := readSpoolRequest(t, spool)
	if req.Title != "Observer: 2 findings (1 high, 1 medium)" {
		t.Fatalf("title: %q", req.Title)
	}
	if req.Severity != conversation.SeverityCritical {
		t.Fatalf("batch severity must follow the highest, got %q", req.Severity)
	}
	highIdx := strings.Index(req.Body, "Finding high-later")
	mediumIdx := strings.Index(req.Body, "Finding medium-first")
	if highIdx < 0 || mediumIdx < 0 || highIdx > mediumIdx {
		t.Fatalf("body must list high before medium:\n%s", req.Body)
	}
	if strings.Contains(req.Body, "low-noise") {
		t.Fatalf("low must never appear:\n%s", req.Body)
	}
	if got := strings.Count(req.Body, "Reply here to discuss."); got != 1 {
		t.Fatalf("footer must appear once, got %d:\n%s", got, req.Body)
	}

	// Both announced branches got the flip on origin.
	for _, slug := range []string{"high-later", "medium-first"} {
		content := gitT(t, origin, "show", "refs/heads/observer/"+slug+":observations/"+slug+".md")
		if !strings.Contains(content, "announced: 2026-07-27T09:00:00Z") {
			t.Fatalf("%s lacks the announced flip:\n%s", slug, content)
		}
	}
}

func TestAnnounceCapsBatchAtThree(t *testing.T) {
	clone, _ := newAnnounceRepos(t)
	for _, slug := range []string{"cap-a", "cap-b", "cap-c", "cap-d"} {
		addObserverBranch(t, clone, slug, "medium", true)
	}
	spool := t.TempDir()

	var out strings.Builder
	opts := announceOpts(t, clone, spool)
	opts.Stdout = &out
	if code := Announce(opts); code != 0 {
		t.Fatal("announce failed")
	}
	req := readSpoolRequest(t, spool)
	if req.Title != "Observer: 3 findings (3 medium)" {
		t.Fatalf("title: %q", req.Title)
	}
	if strings.Contains(req.Body, "cap-d") {
		t.Fatalf("the fourth finding must wait:\n%s", req.Body)
	}
	if !strings.Contains(out.String(), "1 more findings wait for the next run") {
		t.Fatalf("stdout must report the deferred count, got %q", out.String())
	}
}

func TestAnnounceSkipsUnpushedLocalBranches(t *testing.T) {
	clone, _ := newAnnounceRepos(t)
	addObserverBranch(t, clone, "local-only", "high", false)
	spool := t.TempDir()

	var out strings.Builder
	opts := announceOpts(t, clone, spool)
	opts.Stdout = &out
	if code := Announce(opts); code != 0 {
		t.Fatalf("skip must not fail the run")
	}
	if !strings.Contains(out.String(), "nothing to announce") {
		t.Fatalf("expected nothing to announce, got %q", out.String())
	}
	if entries, _ := os.ReadDir(spool); len(entries) != 0 {
		t.Fatal("no request may be written for a local-only branch")
	}
}

func TestAnnounceFailsLoudlyWithoutSpool(t *testing.T) {
	clone, origin := newAnnounceRepos(t)
	addObserverBranch(t, clone, "finding-a", "high", true)

	var errOut strings.Builder
	opts := announceOpts(t, clone, "") // spool env unset
	opts.Stderr = &errOut
	code := Announce(opts)
	if code == 0 {
		t.Fatal("announce without spool must exit non-zero")
	}
	if !strings.Contains(errOut.String(), conversation.SpoolEnvVar) {
		t.Fatalf("error must name the env var: %q", errOut.String())
	}

	// The failure left a log line naming the slug, and no frontmatter flip.
	data, err := os.ReadFile(filepath.Join(clone, ".spekk", "observer-conversation.log"))
	if err != nil {
		t.Fatalf("failure log missing: %v", err)
	}
	line := string(data)
	if !strings.Contains(line, "slug=finding-a") || !strings.Contains(line, "2026-07-27T09:00:00Z") {
		t.Fatalf("log line incomplete: %q", line)
	}
	content := gitT(t, origin, "show", "refs/heads/observer/finding-a:observations/finding-a.md")
	if strings.Contains(content, "announced:") {
		t.Fatal("failed announce must not flip the frontmatter")
	}

	// The log is appended to, never truncated, and gets gitignored.
	code = Announce(opts)
	if code == 0 {
		t.Fatal("second failing run must exit non-zero")
	}
	data2, _ := os.ReadFile(filepath.Join(clone, ".spekk", "observer-conversation.log"))
	if len(strings.Split(strings.TrimSpace(string(data2)), "\n")) != 2 {
		t.Fatalf("expected 2 appended log lines, got %q", string(data2))
	}
	gitignore, err := os.ReadFile(filepath.Join(clone, ".gitignore"))
	if err != nil || !strings.Contains(string(gitignore), LogFileName) {
		t.Fatalf(".gitignore must cover the log: %v %q", err, string(gitignore))
	}
}

func TestSelectCandidatesRules(t *testing.T) {
	mk := func(slug, severity, created, ref string, affected []string, onMain bool) Candidate {
		return Candidate{Slug: slug, Severity: severity, Created: created, Ref: ref, Affected: affected, OnMain: onMain}
	}
	branchRef := func(slug string) string { return "refs/remotes/origin/observer/" + slug }
	rows := []Candidate{
		mk("low-old", "low", "2026-01-01T00:00:00Z", branchRef("low-old"), []string{"a.go"}, false),
		mk("medium-old", "medium", "2026-01-01T00:00:00Z", branchRef("medium-old"), []string{"a.go"}, false),
		mk("high-new", "high", "2026-03-01T00:00:00Z", branchRef("high-new"), []string{"a.go"}, false),
		mk("high-old", "high", "2026-01-01T00:00:00Z", branchRef("high-old"), []string{"a.go"}, false),
		mk("no-evidence", "high", "2026-01-01T00:00:00Z", branchRef("no-evidence"), nil, false),
		mk("on-main", "high", "2026-01-01T00:00:00Z", branchRef("on-main"), []string{"a.go"}, true),
		mk("not-observer", "high", "2026-01-01T00:00:00Z", "refs/heads/feature/x", []string{"a.go"}, false),
		// Tie on severity+created breaks by slug.
		mk("high-old-b", "high", "2026-01-01T00:00:00Z", branchRef("high-old-b"), []string{"a.go"}, false),
	}

	got := SelectCandidates(rows)
	want := []string{"high-old", "high-old-b", "high-new", "medium-old"}
	if len(got) != len(want) {
		t.Fatalf("got %d candidates, want %d: %+v", len(got), len(want), got)
	}
	for i, slug := range want {
		if got[i].Slug != slug {
			t.Fatalf("candidate[%d]: got %q want %q", i, got[i].Slug, slug)
		}
	}
}

func TestComposeRequestWithPRURL(t *testing.T) {
	req := composeBatch([]Candidate{{
		Slug: "x", Severity: "medium", Title: "T",
		PR:       "https://github.com/org/repo/pull/7",
		Affected: []string{"a.go"},
		Body:     "## Issue Description\n\nOne. Two. Three. Four.\n",
	}})
	if req.Title != "T" {
		t.Fatalf("a single finding keeps its own title, got %q", req.Title)
	}
	if !strings.Contains(req.Body, "Proposed fix in PR: https://github.com/org/repo/pull/7 — merge to accept, close to dismiss. Reply here to discuss.") {
		t.Fatalf("pointer line wrong:\n%s", req.Body)
	}
	if strings.Contains(req.Body, "Four.") {
		t.Fatalf("summary must cap at 3 sentences:\n%s", req.Body)
	}
	if req.Severity != conversation.SeverityWarning {
		t.Fatalf("medium must map to warning, got %q", req.Severity)
	}
	if !strings.Contains(req.Body, "Severity: medium") {
		t.Fatalf("severity warning missing:\n%s", req.Body)
	}
}

func TestComposeBatchMultiple(t *testing.T) {
	req := composeBatch([]Candidate{
		{Slug: "a", Severity: "high", Title: "A", Affected: []string{"a.go"},
			Body: "## Issue Description\n\nDrift A.\n"},
		{Slug: "b", Severity: "medium", Title: "B",
			PR:       "https://github.com/org/repo/pull/9",
			Affected: []string{"b.go"},
			Body:     "## Issue Description\n\nDrift B.\n"},
	})
	if req.Title != "Observer: 2 findings (1 high, 1 medium)" {
		t.Fatalf("title: %q", req.Title)
	}
	for _, want := range []string{
		"1. *A* (high)",
		"Proposed fix in PR: observer/a — merge to accept, close to dismiss.",
		"2. *B* (medium)",
		"Proposed fix in PR: https://github.com/org/repo/pull/9 — merge to accept, close to dismiss.",
		"Severity: high",
	} {
		if !strings.Contains(req.Body, want) {
			t.Fatalf("body missing %q:\n%s", want, req.Body)
		}
	}
	if got := strings.Count(req.Body, "Reply here to discuss."); got != 1 {
		t.Fatalf("footer must appear once, got %d", got)
	}
}
