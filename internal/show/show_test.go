package show

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/crossbranch"
	"github.com/spekk-ai/spekk-cli/internal/parser"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// chdir changes the process working directory to dir for the test's duration,
// restoring it via t.Cleanup. Used instead of t.Chdir (a go1.24 API) to keep the
// module's declared go1.23 toolchain floor.
func chdir(t *testing.T, dir string) {
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

func TestBuildShowData(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	writeFile(t, filepath.Join(specsDir, "my-spec", "my-spec.md"), `---
id: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# My Spec`)

	writeFile(t, filepath.Join(specsDir, "my-spec", "assertions", "my-assertion.md"), `---
id: my-assertion
parent: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
branch: feature/test
---

# My Assertion`)

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatal(err)
	}

	data := buildShowData(specsDir, result)

	if data.ProjectName != filepath.Base(dir) {
		t.Errorf("expected project name %s, got %s", filepath.Base(dir), data.ProjectName)
	}
	if len(data.Specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(data.Specs))
	}
	if data.Specs[0].ID != "my-spec" {
		t.Errorf("expected spec id my-spec, got %s", data.Specs[0].ID)
	}
	if len(data.Assertions) != 1 {
		t.Fatalf("expected 1 assertion, got %d", len(data.Assertions))
	}
	if data.Assertions[0].Branch != "feature/test" {
		t.Errorf("expected branch feature/test, got %s", data.Assertions[0].Branch)
	}
}

func TestTemplateContainsPlaceholder(t *testing.T) {
	if !strings.Contains(templateHTML, "/*__SPEKK_DATA__*/") {
		t.Error("template should contain the data placeholder")
	}
}

func TestTemplateSanitizesMarkdown(t *testing.T) {
	// DOMPurify must be loaded
	if !strings.Contains(templateHTML, "dompurify") {
		t.Error("template must include DOMPurify library")
	}

	// Every marked.parse() call must be wrapped in DOMPurify.sanitize()
	if strings.Count(templateHTML, "marked.parse(") != strings.Count(templateHTML, "DOMPurify.sanitize(marked.parse(") {
		t.Error("all marked.parse() calls must be wrapped with DOMPurify.sanitize()")
	}
}

// git runs a raw git command in dir, failing the test on error. Used only to
// build fixture repos for cross-branch tests.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func assertionMD(id, status, body string) string {
	return "---\nid: " + id + "\nparent: demo\ncreated: 2026-01-01T00:00:00Z\n" +
		"priority: 1\nstatus: " + status + "\n---\n\n# " + id + "\n\n" + body + "\n"
}

// findContribution returns the contribution from a branch in a list, or fails.
func findContribution(t *testing.T, list []crossBranchContribution, branch string) crossBranchContribution {
	t.Helper()
	for _, c := range list {
		if c.Branch == branch {
			return c
		}
	}
	t.Fatalf("no contribution from branch %q in %+v", branch, list)
	return crossBranchContribution{}
}

// TestCrossBranchFolding builds a temp git repo whose specs differ across
// branches, runs buildShowData + applyCrossBranch with cross-branch mode on, and
// asserts the per-item contributions, the incoming_add synthesis, and the
// spec-level rollup all appear.
func TestCrossBranchFolding(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")

	specsDir := filepath.Join(dir, "specs")
	specFile := filepath.Join(specsDir, "demo", "demo.md")
	modFile := filepath.Join(specsDir, "demo", "assertions", "clean-mod.md")
	delFile := filepath.Join(specsDir, "demo", "assertions", "to-delete.md")
	addFile := filepath.Join(specsDir, "demo", "assertions", "foreign.md")

	writeFile(t, specFile, "---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# demo\n")
	writeFile(t, modFile, assertionMD("clean-mod", "not_started", "original body"))
	writeFile(t, delFile, assertionMD("to-delete", "not_started", "doomed"))
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "base")

	// Branch "other" diverges: incoming add, status-drift modification, deletion.
	git(t, dir, "checkout", "-q", "-b", "other")
	writeFile(t, addFile, assertionMD("foreign", "draft", "only on branch"))
	writeFile(t, modFile, assertionMD("clean-mod", "done", "original body"))
	os.Remove(delFile)
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "branch work")

	// Back to ours; restore working tree to main's content.
	git(t, dir, "checkout", "-q", "main")
	chdir(t, dir)

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	data := buildShowData(specsDir, result)
	if err := applyCrossBranch(&data, ""); err != nil {
		t.Fatal(err)
	}

	if !data.CrossBranch {
		t.Error("expected CrossBranch mode flag to be true")
	}

	// Incoming addition: foreign assertion has no local file, so it must be
	// synthesized and carry an incoming_add contribution.
	var foreign *showAssertion
	for i := range data.Assertions {
		if data.Assertions[i].ID == "foreign" {
			foreign = &data.Assertions[i]
		}
	}
	if foreign == nil {
		t.Fatal("foreign assertion (incoming_add) was not synthesized into the data")
	}
	if foreign.Parent != "demo" {
		t.Errorf("synthesized foreign parent = %q, want demo", foreign.Parent)
	}
	if c := findContribution(t, foreign.CrossBranch, "other"); c.State != "incoming_add" {
		t.Errorf("foreign state = %q, want incoming_add", c.State)
	}

	// Clean modification with status drift not_started -> done.
	var mod *showAssertion
	for i := range data.Assertions {
		if data.Assertions[i].ID == "clean-mod" {
			mod = &data.Assertions[i]
		}
	}
	if mod == nil {
		t.Fatal("clean-mod assertion missing")
	}
	c := findContribution(t, mod.CrossBranch, "other")
	if c.State != "incoming_mod" {
		t.Errorf("clean-mod state = %q, want incoming_mod", c.State)
	}
	if c.OldStatus != "not_started" || c.NewStatus != "done" {
		t.Errorf("clean-mod drift = %q->%q, want not_started->done", c.OldStatus, c.NewStatus)
	}

	// Incoming deletion on the local to-delete assertion.
	var del *showAssertion
	for i := range data.Assertions {
		if data.Assertions[i].ID == "to-delete" {
			del = &data.Assertions[i]
		}
	}
	if del == nil {
		t.Fatal("to-delete assertion missing")
	}
	if c := findContribution(t, del.CrossBranch, "other"); c.State != "incoming_del" {
		t.Errorf("to-delete state = %q, want incoming_del", c.State)
	}

	// Rollup: precedence incoming_del > incoming_add > incoming_mod => the demo
	// spec's headline is incoming_del.
	var demo *showSpec
	for i := range data.Specs {
		if data.Specs[i].ID == "demo" {
			demo = &data.Specs[i]
		}
	}
	if demo == nil {
		t.Fatal("demo spec missing")
	}
	if demo.CrossBranchSummary != "incoming_del" {
		t.Errorf("demo rollup = %q, want incoming_del", demo.CrossBranchSummary)
	}

	// Branch metadata records the compared branch.
	if len(data.Branches) != 1 || data.Branches[0] != "other" {
		t.Errorf("Branches = %v, want [other]", data.Branches)
	}
}

// TestCrossBranchRollupDirNotEqualID: when a spec's directory differs from its
// frontmatter id, a foreign incoming_add assertion (whose synthesized parent is
// derived from the directory) must still roll up into that spec's summary. The
// rollup keys by spec directory, not frontmatter id, so this holds.
func TestCrossBranchRollupDirNotEqualID(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")

	specsDir := filepath.Join(dir, "specs")
	// Directory is "user-auth" but the spec's frontmatter id is "authentication".
	writeFile(t, filepath.Join(specsDir, "user-auth", "user-auth.md"),
		"---\nid: authentication\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# Authentication\n")
	writeFile(t, filepath.Join(specsDir, "user-auth", "assertions", "existing.md"),
		"---\nid: existing\nparent: authentication\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n---\n\n# existing\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "base")

	// Branch adds a foreign assertion under the same directory.
	git(t, dir, "checkout", "-q", "-b", "other")
	writeFile(t, filepath.Join(specsDir, "user-auth", "assertions", "new.md"),
		"---\nid: new\nparent: authentication\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: draft\n---\n\n# new\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "foreign assertion")
	git(t, dir, "checkout", "-q", "main")
	chdir(t, dir)

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	data := buildShowData(specsDir, result)
	if err := applyCrossBranch(&data, ""); err != nil {
		t.Fatal(err)
	}

	var spec *showSpec
	for i := range data.Specs {
		if data.Specs[i].ID == "authentication" {
			spec = &data.Specs[i]
		}
	}
	if spec == nil {
		t.Fatal("authentication spec missing")
	}
	if spec.CrossBranchSummary != "incoming_add" {
		t.Errorf("rollup for dir!=id spec = %q, want incoming_add (foreign assertion must roll into the spec summary)", spec.CrossBranchSummary)
	}
}

// TestTemplateCrossBranchUI guards the cross-branch template wiring: the
// branch-selection dropdown, its localStorage persistence, the client-side
// re-render, the icon-only badge rendering, and the missing-status fallback. These
// are string-level checks so an accidental removal of the JS is caught.
func TestTemplateCrossBranchUI(t *testing.T) {
	markers := []string{
		"cb-branch-toggle",            // dropdown button
		"cb-branch-checkbox",          // per-branch checkboxes
		"cbApplyBranchSelection",      // client-side re-render on toggle
		"spekkCrossBranchDeselected:", // localStorage key (per project)
		"function cbFilteredContribs", // contribution filter
		"function cbSpecVisible",      // foreign-item hiding
		"function statusClass",        // empty-status fallback
		".status-unknown",             // neutral badge style
	}
	for _, m := range markers {
		if !strings.Contains(templateHTML, m) {
			t.Errorf("template.html missing cross-branch UI marker %q", m)
		}
	}
	// The inline state badge must be icon-only: cbBadgeHtml builds a titled span
	// with no inner label text.
	if !strings.Contains(templateHTML, `class="' + cls + '" title="' + escapeHtml(label) + '"></span>`) {
		t.Error("cbBadgeHtml should render an icon-only badge (title tooltip, empty body)")
	}
}

// TestForeignItemHasMetadataAndFlag verifies that a wholly foreign spec (parent +
// assertion added only on another branch) is synthesized with Foreign=true and
// real metadata: the assertion carries its parsed status, and the spec's status is
// derived from that assertion (not left blank, which previously produced an empty
// status badge).
func TestForeignItemHasMetadataAndFlag(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")

	specsDir := filepath.Join(dir, "specs")
	// A local spec so the explorer has something on ours.
	writeFile(t, filepath.Join(specsDir, "local", "local.md"),
		"---\nid: local\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# Local\n")
	writeFile(t, filepath.Join(specsDir, "local", "assertions", "a.md"),
		"---\nid: a\nparent: local\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n---\n\n# a\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "base")

	// A branch adds an entirely new spec with a done assertion.
	git(t, dir, "checkout", "-q", "-b", "other")
	writeFile(t, filepath.Join(specsDir, "shiny", "shiny.md"),
		"---\nid: shiny\ncreated: 2026-01-01T00:00:00Z\npriority: 2\n---\n\n# Shiny Feature\n")
	writeFile(t, filepath.Join(specsDir, "shiny", "assertions", "works.md"),
		"---\nid: works\nparent: shiny\ncreated: 2026-01-01T00:00:00Z\npriority: 3\nstatus: done\n---\n\n# It Works\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "foreign spec")
	git(t, dir, "checkout", "-q", "main")
	chdir(t, dir)

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	data := buildShowData(specsDir, result)
	if err := applyCrossBranch(&data, ""); err != nil {
		t.Fatal(err)
	}

	var fs *showSpec
	for i := range data.Specs {
		if data.Specs[i].ID == "shiny" {
			fs = &data.Specs[i]
		}
	}
	if fs == nil {
		t.Fatal("foreign spec 'shiny' was not synthesized")
	}
	if !fs.Foreign {
		t.Error("foreign spec must be flagged Foreign=true")
	}
	if fs.Title != "Shiny Feature" {
		t.Errorf("foreign spec title = %q, want parsed 'Shiny Feature'", fs.Title)
	}
	if fs.Priority != 2 {
		t.Errorf("foreign spec priority = %d, want 2 (parsed from ref)", fs.Priority)
	}
	// status derived from the single done assertion -> done (never empty).
	if fs.Status != "done" {
		t.Errorf("foreign spec status = %q, want done (derived from foreign assertions)", fs.Status)
	}

	var fa *showAssertion
	for i := range data.Assertions {
		if data.Assertions[i].ID == "works" {
			fa = &data.Assertions[i]
		}
	}
	if fa == nil {
		t.Fatal("foreign assertion 'works' was not synthesized")
	}
	if !fa.Foreign || fa.Status != "done" || fa.Priority != 3 {
		t.Errorf("foreign assertion = {Foreign:%v Status:%q Priority:%d}, want {true done 3}", fa.Foreign, fa.Status, fa.Priority)
	}
}

// TestSynthesizeForeignLinkage verifies a synthesized foreign spec and its
// synthesized foreign assertions link by spec-directory: the spec's id (from the
// directory, not the file basename) must equal the assertions' derived parent so
// the tree nests them together.
func TestSynthesizeForeignLinkage(t *testing.T) {
	const (
		specPath = "specs/foo/foo.md"
		aPath    = "specs/foo/assertions/bar.md"
	)
	if got := specDirName(specPath); got != "foo" {
		t.Errorf("specDirName(%q) = %q, want foo", specPath, got)
	}
	s := synthesizeSpec(specPath, nil)
	a := synthesizeAssertion(aPath, nil)
	if s.ID != a.Parent {
		t.Errorf("synthesized spec id %q != assertion parent %q (would orphan the assertion)", s.ID, a.Parent)
	}
	if s.ID != "foo" {
		t.Errorf("synthesized spec id = %q, want foo (the directory name)", s.ID)
	}
	if !s.Foreign || !a.Foreign {
		t.Errorf("synthesized items must be flagged Foreign: spec=%v assertion=%v", s.Foreign, a.Foreign)
	}

	// With metadata, real title/status/priority flow through.
	meta := &crossbranch.FileMeta{Title: "Real Title", Status: "done", Priority: 3, Content: "# body"}
	am := synthesizeAssertion(aPath, meta)
	if am.Title != "Real Title" || am.Status != "done" || am.Priority != 3 || am.Content != "# body" {
		t.Errorf("synthesizeAssertion with meta = %+v, want real title/status/priority/content", am)
	}
}

// TestCrossBranchOffUnchanged verifies the non-cross-branch path leaves the new
// fields empty so existing output is unaffected.
func TestCrossBranchOffUnchanged(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	writeFile(t, filepath.Join(specsDir, "demo", "demo.md"),
		"---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# demo\n")

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		t.Fatal(err)
	}
	data := buildShowData(specsDir, result)

	if data.CrossBranch || data.Degraded || data.Branches != nil {
		t.Error("cross-branch metadata must be zero-valued when mode is off")
	}
	for _, s := range data.Specs {
		if s.CrossBranch != nil || s.CrossBranchSummary != "" {
			t.Error("spec cross-branch fields must be empty when mode is off")
		}
	}
}

func TestRunWritesHTML(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")

	writeFile(t, filepath.Join(specsDir, "test-spec", "test-spec.md"), `---
id: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# Test Spec`)

	writeFile(t, filepath.Join(specsDir, "test-spec", "assertions", "test-a.md"), `---
id: test-a
parent: test-spec
created: 2026-01-01T00:00:00Z
priority: 1
---

# Test Assertion`)

	// Set CI to skip browser opening
	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")

	err := Run(specsDir, Options{})
	if err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, ".spekk", "index.html")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	html := string(content)
	if !strings.Contains(html, "test-spec") {
		t.Error("HTML should contain spec ID")
	}
	if !strings.Contains(html, "Test Assertion") {
		t.Error("HTML should contain assertion title")
	}
	if strings.Contains(html, "/*__SPEKK_DATA__*/") {
		t.Error("placeholder should be replaced with actual data")
	}
}
