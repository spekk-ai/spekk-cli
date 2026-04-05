package parser

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// parseFrontmatter
// ---------------------------------------------------------------------------

func TestParseFrontmatter_Basic(t *testing.T) {
	content := "---\nid: my-spec\npriority: 1\n---\n# Title\n\nBody text."
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.get("id") != "my-spec" {
		t.Errorf("expected id=my-spec, got %q", fm.get("id"))
	}
	if fm.get("priority") != "1" {
		t.Errorf("expected priority=1, got %q", fm.get("priority"))
	}
	if !strings.Contains(body, "# Title") {
		t.Errorf("expected body to contain heading, got %q", body)
	}
}

func TestParseFrontmatter_CRLFNormalization(t *testing.T) {
	content := "---\r\nid: crlf-test\r\npriority: 2\r\n---\r\n# Heading\r\n"
	fm, _, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.get("id") != "crlf-test" {
		t.Errorf("expected id=crlf-test, got %q", fm.get("id"))
	}
}

func TestParseFrontmatter_NoDelimiter(t *testing.T) {
	content := "# Just markdown\n\nNo frontmatter here."
	_, _, err := parseFrontmatter(content)
	if err == nil {
		t.Fatal("expected error for missing --- delimiter")
	}
}

func TestParseFrontmatter_MissingClosingDelimiter(t *testing.T) {
	content := "---\nid: no-close\npriority: 1\n"
	_, _, err := parseFrontmatter(content)
	if err == nil {
		t.Fatal("expected error for missing closing ---")
	}
}

func TestParseFrontmatter_QuotedValues(t *testing.T) {
	content := "---\nid: quoted\nbranch: \"feature/test\"\npriority: 1\n---\n"
	fm, _, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.get("branch") != "feature/test" {
		t.Errorf("expected branch=feature/test, got %q", fm.get("branch"))
	}
}

func TestParseFrontmatter_DependsOnField(t *testing.T) {
	content := "---\nid: dep-test\nparent: foo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\ndepends-on: other-assertion\n---\n# Title\n"
	fm, _, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.get("depends-on") != "other-assertion" {
		t.Errorf("expected depends-on=other-assertion, got %q", fm.get("depends-on"))
	}
}

// ---------------------------------------------------------------------------
// extractTitle
// ---------------------------------------------------------------------------

func TestExtractTitle_Found(t *testing.T) {
	content := "\n# My Spec Title\n\nSome body text."
	title := extractTitle(content)
	if title != "My Spec Title" {
		t.Errorf("expected 'My Spec Title', got %q", title)
	}
}

func TestExtractTitle_NotFound(t *testing.T) {
	content := "No heading here.\n\nJust paragraphs."
	title := extractTitle(content)
	if title != "Untitled" {
		t.Errorf("expected 'Untitled', got %q", title)
	}
}

func TestExtractTitle_FirstH1(t *testing.T) {
	content := "# First Heading\n\n## Second\n\n# Third"
	title := extractTitle(content)
	if title != "First Heading" {
		t.Errorf("expected 'First Heading', got %q", title)
	}
}

// ---------------------------------------------------------------------------
// hasFrontmatter
// ---------------------------------------------------------------------------

func TestHasFrontmatter_True(t *testing.T) {
	if !hasFrontmatter("---\nid: x\n---\n") {
		t.Error("expected hasFrontmatter=true")
	}
}

func TestHasFrontmatter_False(t *testing.T) {
	if hasFrontmatter("# Just markdown") {
		t.Error("expected hasFrontmatter=false")
	}
}

func TestHasFrontmatter_CRLF(t *testing.T) {
	if !hasFrontmatter("---\r\nid: x\r\n---\r\n") {
		t.Error("expected hasFrontmatter=true for CRLF content")
	}
}

// ---------------------------------------------------------------------------
// IsLockStale
// ---------------------------------------------------------------------------

func TestIsLockStale_EmptyString(t *testing.T) {
	if !IsLockStale("") {
		t.Error("expected empty lockedBy to be stale")
	}
}

func TestIsLockStale_InvalidFormat(t *testing.T) {
	if !IsLockStale("builder-hostname-notanumber") {
		t.Error("expected invalid format to be stale")
	}
}

func TestIsLockStale_FreshLock(t *testing.T) {
	// Lock created 60 seconds ago should NOT be stale.
	freshTimestamp := strconv.FormatInt(time.Now().Unix()-60, 10)
	lockedBy := "builder-hostname-12345-" + freshTimestamp
	if IsLockStale(lockedBy) {
		t.Errorf("expected fresh lock (60s old) to NOT be stale")
	}
}

func TestIsLockStale_StaleLock(t *testing.T) {
	// Lock created 3 hours ago should be stale.
	staleTimestamp := strconv.FormatInt(time.Now().Unix()-10800, 10)
	lockedBy := "builder-hostname-12345-" + staleTimestamp
	if !IsLockStale(lockedBy) {
		t.Errorf("expected old lock (3h) to be stale")
	}
}

// ---------------------------------------------------------------------------
// ParseAllSpecs — integration tests using temporary directories
// ---------------------------------------------------------------------------

// writeFile writes content to path, creating parent directories as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestParseAllSpecs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.Mkdir(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Specs) != 0 || len(result.Assertions) != 0 {
		t.Errorf("expected empty result, got %d specs, %d assertions", len(result.Specs), len(result.Assertions))
	}
}

func TestParseAllSpecs_NonExistentDir(t *testing.T) {
	result, err := ParseAllSpecs("/tmp/definitely-does-not-exist-spekk-test-12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestParseAllSpecs_SingleSpecWithAssertion(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	// Write spec file.
	specContent := `---
id: my-feature
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
---

# My Feature Spec

Description here.
`
	writeFile(t, filepath.Join(specsDir, "my-feature", "my-feature.md"), specContent)

	// Write assertion file.
	assertionContent := `---
id: my-feature-works
parent: my-feature
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
branch: feature/my-feature
---

# My feature works correctly

It should do things.
`
	writeFile(t, filepath.Join(specsDir, "my-feature", "assertions", "my-feature-works.md"), assertionContent)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(result.Specs))
	}
	if len(result.Assertions) != 1 {
		t.Fatalf("expected 1 assertion, got %d", len(result.Assertions))
	}

	spec := result.Specs[0]
	if spec.ID != "my-feature" {
		t.Errorf("expected spec id=my-feature, got %q", spec.ID)
	}
	if spec.Title != "My Feature Spec" {
		t.Errorf("expected spec title='My Feature Spec', got %q", spec.Title)
	}
	if spec.Priority != 1 {
		t.Errorf("expected priority=1, got %d", spec.Priority)
	}

	assertion := result.Assertions[0]
	if assertion.ID != "my-feature-works" {
		t.Errorf("expected assertion id=my-feature-works, got %q", assertion.ID)
	}
	if assertion.Parent != "my-feature" {
		t.Errorf("expected assertion parent=my-feature, got %q", assertion.Parent)
	}
	if assertion.Branch != "feature/my-feature" {
		t.Errorf("expected assertion branch=feature/my-feature, got %q", assertion.Branch)
	}
	if assertion.Title != "My feature works correctly" {
		t.Errorf("expected assertion title='My feature works correctly', got %q", assertion.Title)
	}
}

func TestParseAllSpecs_MalformedAssertionSkipped(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	specContent := `---
id: alpha
created: 2026-01-01T00:00:00Z
priority: 1
---

# Alpha
`
	writeFile(t, filepath.Join(specsDir, "alpha", "alpha.md"), specContent)

	// Malformed: missing required 'parent' field.
	badAssertion := `---
id: bad-assertion
created: 2026-01-01T00:00:00Z
priority: 1
---

# Bad assertion
`
	writeFile(t, filepath.Join(specsDir, "alpha", "assertions", "bad.md"), badAssertion)

	// Valid assertion.
	goodAssertion := `---
id: good-assertion
parent: alpha
created: 2026-01-01T00:00:00Z
priority: 1
---

# Good assertion
`
	writeFile(t, filepath.Join(specsDir, "alpha", "assertions", "good.md"), goodAssertion)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the good assertion should be parsed.
	if len(result.Assertions) != 1 {
		t.Errorf("expected 1 assertion, got %d", len(result.Assertions))
	}
	if len(result.Assertions) > 0 && result.Assertions[0].ID != "good-assertion" {
		t.Errorf("expected good-assertion, got %q", result.Assertions[0].ID)
	}
}

func TestParseAllSpecs_NoFrontmatterFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	specContent := `---
id: beta
created: 2026-01-01T00:00:00Z
priority: 2
---

# Beta Spec
`
	writeFile(t, filepath.Join(specsDir, "beta", "beta.md"), specContent)

	// This file has no frontmatter — should be silently skipped.
	noFm := "# Just a readme\n\nNo frontmatter here."
	writeFile(t, filepath.Join(specsDir, "beta", "assertions", "readme.md"), noFm)

	// Valid assertion.
	goodAssertion := `---
id: beta-works
parent: beta
created: 2026-01-01T00:00:00Z
priority: 2
---

# Beta works
`
	writeFile(t, filepath.Join(specsDir, "beta", "assertions", "beta-works.md"), goodAssertion)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Assertions) != 1 {
		t.Errorf("expected 1 assertion, got %d", len(result.Assertions))
	}
}

func TestParseAllSpecs_CRLFNormalization(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	// Write file with CRLF line endings.
	crlfContent := "---\r\nid: crlf-spec\r\ncreated: 2026-01-01T00:00:00Z\r\npriority: 1\r\n---\r\n\r\n# CRLF Spec\r\n"
	writeFile(t, filepath.Join(specsDir, "crlf-spec", "crlf-spec.md"), crlfContent)

	crlfAssertion := "---\r\nid: crlf-assertion\r\nparent: crlf-spec\r\ncreated: 2026-01-01T00:00:00Z\r\npriority: 1\r\n---\r\n\r\n# CRLF Assertion\r\n"
	writeFile(t, filepath.Join(specsDir, "crlf-spec", "assertions", "crlf-assertion.md"), crlfAssertion)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(result.Specs))
	}
	if result.Specs[0].ID != "crlf-spec" {
		t.Errorf("expected id=crlf-spec, got %q", result.Specs[0].ID)
	}
	if result.Specs[0].Title != "CRLF Spec" {
		t.Errorf("expected title='CRLF Spec', got %q", result.Specs[0].Title)
	}
}

func TestParseAllSpecs_DependsOnExtracted(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	specContent := `---
id: dep-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# Dep Spec
`
	writeFile(t, filepath.Join(specsDir, "dep-spec", "dep-spec.md"), specContent)

	firstAssertion := `---
id: first-step
parent: dep-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: done
---

# First Step
`
	writeFile(t, filepath.Join(specsDir, "dep-spec", "assertions", "first-step.md"), firstAssertion)

	secondAssertion := `---
id: second-step
parent: dep-spec
created: 2026-01-02T00:00:00Z
priority: 1
depends-on: first-step
---

# Second Step
`
	writeFile(t, filepath.Join(specsDir, "dep-spec", "assertions", "second-step.md"), secondAssertion)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(result.Assertions))
	}

	var secondStep *Assertion
	for i := range result.Assertions {
		if result.Assertions[i].ID == "second-step" {
			secondStep = &result.Assertions[i]
			break
		}
	}

	if secondStep == nil {
		t.Fatal("second-step assertion not found")
	}
	if secondStep.DependsOn != "first-step" {
		t.Errorf("expected DependsOn=first-step, got %q", secondStep.DependsOn)
	}
}

func TestParseAllSpecs_DefaultStatusNotStarted(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	// Spec and assertion with no explicit status field.
	specContent := `---
id: no-status
created: 2026-01-01T00:00:00Z
priority: 1
---

# No Status Spec
`
	writeFile(t, filepath.Join(specsDir, "no-status", "no-status.md"), specContent)

	assertionContent := `---
id: no-status-assertion
parent: no-status
created: 2026-01-01T00:00:00Z
priority: 1
---

# No Status Assertion
`
	writeFile(t, filepath.Join(specsDir, "no-status", "assertions", "no-status-assertion.md"), assertionContent)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Assertions[0].Status != "not_started" {
		t.Errorf("expected default status=not_started, got %q", result.Assertions[0].Status)
	}
}

func TestParseAllSpecs_DefaultBranchMain(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	specContent := `---
id: no-branch
created: 2026-01-01T00:00:00Z
priority: 1
---

# No Branch Spec
`
	writeFile(t, filepath.Join(specsDir, "no-branch", "no-branch.md"), specContent)

	assertionContent := `---
id: no-branch-assertion
parent: no-branch
created: 2026-01-01T00:00:00Z
priority: 1
---

# No Branch Assertion
`
	writeFile(t, filepath.Join(specsDir, "no-branch", "assertions", "no-branch-assertion.md"), assertionContent)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Specs[0].Branch != "main" {
		t.Errorf("expected default branch=main for spec, got %q", result.Specs[0].Branch)
	}
	if result.Assertions[0].Branch != "main" {
		t.Errorf("expected default branch=main for assertion, got %q", result.Assertions[0].Branch)
	}
}

func TestParseAllSpecs_RelativeFilePaths(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	specContent := `---
id: path-check
created: 2026-01-01T00:00:00Z
priority: 1
---

# Path Check Spec
`
	writeFile(t, filepath.Join(specsDir, "path-check", "path-check.md"), specContent)

	assertionContent := `---
id: path-check-assertion
parent: path-check
created: 2026-01-01T00:00:00Z
priority: 1
---

# Path Check Assertion
`
	writeFile(t, filepath.Join(specsDir, "path-check", "assertions", "path-check-assertion.md"), assertionContent)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSpecFile := "specs/path-check/path-check.md"
	if result.Specs[0].File != expectedSpecFile {
		t.Errorf("expected spec file=%q, got %q", expectedSpecFile, result.Specs[0].File)
	}

	expectedAssertionFile := "specs/path-check/assertions/path-check-assertion.md"
	if result.Assertions[0].File != expectedAssertionFile {
		t.Errorf("expected assertion file=%q, got %q", expectedAssertionFile, result.Assertions[0].File)
	}
}

func TestParseAllSpecs_ContentPreserved(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	specContent := `---
id: content-test
created: 2026-01-01T00:00:00Z
priority: 1
---

# Content Test

Some detailed spec content that should be preserved.
`
	writeFile(t, filepath.Join(specsDir, "content-test", "content-test.md"), specContent)

	assertionContent := `---
id: content-assertion
parent: content-test
created: 2026-01-01T00:00:00Z
priority: 1
---

# Content Assertion

Detailed assertion body text.
`
	writeFile(t, filepath.Join(specsDir, "content-test", "assertions", "content-assertion.md"), assertionContent)

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.Specs[0].Content, "Some detailed spec content") {
		t.Error("spec content not preserved")
	}
	if !strings.Contains(result.Assertions[0].Content, "Detailed assertion body text") {
		t.Error("assertion content not preserved")
	}
}

// ---------------------------------------------------------------------------
// computeParentStatus
// ---------------------------------------------------------------------------

func TestComputeParentStatus_AllDone(t *testing.T) {
	assertions := []Assertion{
		{ID: "a1", Parent: "p1", Status: "done"},
		{ID: "a2", Parent: "p1", Status: "done"},
	}
	status := computeParentStatus("p1", assertions)
	if status != "done" {
		t.Errorf("expected done, got %q", status)
	}
}

func TestComputeParentStatus_AnyFailed(t *testing.T) {
	assertions := []Assertion{
		{ID: "a1", Parent: "p1", Status: "done"},
		{ID: "a2", Parent: "p1", Status: "failed"},
	}
	status := computeParentStatus("p1", assertions)
	if status != "failed" {
		t.Errorf("expected failed, got %q", status)
	}
}

func TestComputeParentStatus_InProgress(t *testing.T) {
	assertions := []Assertion{
		{ID: "a1", Parent: "p1", Status: "done"},
		{ID: "a2", Parent: "p1", Status: "not_started"},
	}
	status := computeParentStatus("p1", assertions)
	if status != "in_progress" {
		t.Errorf("expected in_progress, got %q", status)
	}
}

func TestComputeParentStatus_NoChildren(t *testing.T) {
	status := computeParentStatus("p1", []Assertion{})
	if status != "not_started" {
		t.Errorf("expected not_started, got %q", status)
	}
}

func TestComputeParentStatus_AllDraft(t *testing.T) {
	assertions := []Assertion{
		{ID: "a1", Parent: "p1", Status: "draft"},
	}
	status := computeParentStatus("p1", assertions)
	if status != "not_started" {
		t.Errorf("expected not_started when all children are draft, got %q", status)
	}
}

// ---------------------------------------------------------------------------
// FindNextAssertion
// ---------------------------------------------------------------------------

func makeTestAssertion(id, parent, status, branch, created string, priority int) Assertion {
	return Assertion{
		ID:       id,
		Parent:   parent,
		Status:   status,
		Branch:   branch,
		Created:  created,
		Priority: priority,
	}
}

func TestFindNextAssertion_PriorityOrdering(t *testing.T) {
	assertions := []Assertion{
		makeTestAssertion("low-prio", "spec", "not_started", "main", "2026-01-01T00:00:00Z", 3),
		makeTestAssertion("high-prio", "spec", "not_started", "main", "2026-01-01T00:00:00Z", 1),
		makeTestAssertion("mid-prio", "spec", "not_started", "main", "2026-01-01T00:00:00Z", 2),
	}

	next := FindNextAssertion(assertions, nil, FindOptions{AllBranches: true})
	if next == nil {
		t.Fatal("expected an assertion, got nil")
	}
	if next.ID != "high-prio" {
		t.Errorf("expected high-prio, got %q", next.ID)
	}
}

func TestFindNextAssertion_SkipsDone(t *testing.T) {
	assertions := []Assertion{
		makeTestAssertion("done-assertion", "spec", "done", "main", "2026-01-01T00:00:00Z", 1),
		makeTestAssertion("pending", "spec", "not_started", "main", "2026-01-02T00:00:00Z", 1),
	}

	next := FindNextAssertion(assertions, nil, FindOptions{AllBranches: true})
	if next == nil {
		t.Fatal("expected an assertion")
	}
	if next.ID != "pending" {
		t.Errorf("expected pending, got %q", next.ID)
	}
}

func TestFindNextAssertion_BranchFilter(t *testing.T) {
	assertions := []Assertion{
		makeTestAssertion("other-branch", "spec", "not_started", "feature/other", "2026-01-01T00:00:00Z", 1),
		makeTestAssertion("my-branch", "spec", "not_started", "feature/mine", "2026-01-01T00:00:00Z", 1),
	}

	next := FindNextAssertion(assertions, nil, FindOptions{
		AllBranches:   false,
		CurrentBranch: "feature/mine",
	})
	if next == nil {
		t.Fatal("expected an assertion")
	}
	if next.ID != "my-branch" {
		t.Errorf("expected my-branch, got %q", next.ID)
	}
}

func TestFindNextAssertion_DependencyNotSatisfied(t *testing.T) {
	assertions := []Assertion{
		{ID: "dep", Parent: "spec", Status: "not_started", Priority: 1, Created: "2026-01-01T00:00:00Z"},
		{ID: "dependent", Parent: "spec", Status: "not_started", Priority: 1, Created: "2026-01-01T00:00:00Z", DependsOn: "dep"},
	}

	next := FindNextAssertion(assertions, nil, FindOptions{AllBranches: true})
	if next == nil {
		t.Fatal("expected an assertion")
	}
	// Should return dep (the prerequisite), not dependent.
	if next.ID != "dep" {
		t.Errorf("expected dep (unblocked assertion), got %q", next.ID)
	}
}

func TestFindNextAssertion_DependencySatisfied(t *testing.T) {
	assertions := []Assertion{
		{ID: "dep", Parent: "spec", Status: "done", Priority: 1, Created: "2026-01-01T00:00:00Z"},
		{ID: "dependent", Parent: "spec", Status: "not_started", Priority: 1, Created: "2026-01-02T00:00:00Z", DependsOn: "dep"},
	}

	next := FindNextAssertion(assertions, nil, FindOptions{AllBranches: true})
	if next == nil {
		t.Fatal("expected an assertion")
	}
	if next.ID != "dependent" {
		t.Errorf("expected dependent, got %q", next.ID)
	}
}

func TestFindNextAssertion_ByAssertionID(t *testing.T) {
	assertions := []Assertion{
		makeTestAssertion("specific", "spec", "not_started", "main", "2026-01-01T00:00:00Z", 3),
		makeTestAssertion("other", "spec", "not_started", "main", "2026-01-01T00:00:00Z", 1),
	}

	next := FindNextAssertion(assertions, nil, FindOptions{AssertionID: "specific", AllBranches: true})
	if next == nil {
		t.Fatal("expected an assertion")
	}
	if next.ID != "specific" {
		t.Errorf("expected specific, got %q", next.ID)
	}
}

func TestFindNextAssertion_AllNone(t *testing.T) {
	assertions := []Assertion{
		makeTestAssertion("done1", "spec", "done", "main", "2026-01-01T00:00:00Z", 1),
	}

	next := FindNextAssertion(assertions, nil, FindOptions{AllBranches: true})
	if next != nil {
		t.Errorf("expected nil, got %v", next)
	}
}

// ---------------------------------------------------------------------------
// Integration test: parse project's own specs directory
// ---------------------------------------------------------------------------

func TestParseAllSpecs_ProjectSpecs(t *testing.T) {
	// Walk up from internal/parser/ to find the project root specs/ dir.
	// The test binary runs from the package directory.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	// Traverse up until we find a specs/ directory (max 5 levels).
	specsDir := ""
	dir := cwd
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "specs")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			specsDir = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if specsDir == "" {
		t.Skip("specs/ directory not found — skipping project integration test")
	}

	result, err := ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatalf("ParseAllSpecs on project specs: %v", err)
	}

	if len(result.Specs) == 0 {
		t.Error("expected at least one spec group from project specs/")
	}
	if len(result.Assertions) == 0 {
		t.Error("expected at least one assertion from project specs/")
	}

	t.Logf("Parsed %d specs and %d assertions from project specs/", len(result.Specs), len(result.Assertions))

	// Verify all assertions have non-empty required fields.
	for _, a := range result.Assertions {
		if a.ID == "" {
			t.Errorf("assertion with empty ID found in file %s", a.File)
		}
		if a.Parent == "" {
			t.Errorf("assertion %q has empty parent", a.ID)
		}
		if a.Title == "" {
			t.Errorf("assertion %q has empty title", a.ID)
		}
	}

	// Verify all specs have non-empty required fields.
	for _, s := range result.Specs {
		if s.ID == "" {
			t.Errorf("spec with empty ID found in file %s", s.File)
		}
		if s.Title == "" {
			t.Errorf("spec %q has empty title", s.ID)
		}
	}
}
