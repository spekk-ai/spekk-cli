package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/parser"
)

// makeTmpSpecs creates a temp directory with one minimal spec + assertion and
// returns the path to the specs directory (suitable for passing as specsDir to
// execList or ParseAllSpecs).
func makeTmpSpecs(t *testing.T) string {
	t.Helper()
	specsDir := t.TempDir()
	specDir := filepath.Join(specsDir, "my-spec")
	assertionsDir := filepath.Join(specDir, "assertions")
	if err := os.MkdirAll(assertionsDir, 0o755); err != nil {
		t.Fatalf("makeTmpSpecs: create assertions dir: %v", err)
	}
	specContent := `---
id: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
---
# My Spec
`
	if err := os.WriteFile(filepath.Join(specDir, "my-spec.md"), []byte(specContent), 0o644); err != nil {
		t.Fatalf("makeTmpSpecs: write spec: %v", err)
	}
	assertionContent := `---
id: my-assertion
parent: my-spec
created: 2026-01-01T00:00:00Z
priority: 1
status: not_started
---
# My Assertion
`
	if err := os.WriteFile(filepath.Join(assertionsDir, "my-assertion.md"), []byte(assertionContent), 0o644); err != nil {
		t.Fatalf("makeTmpSpecs: write assertion: %v", err)
	}
	return specsDir
}

// --- Mutual exclusion tests ---

func TestExecList_MutualExclusion_JSONandTSV(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--json", "--tsv"}, &stdout, &stderr, t.TempDir())
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in stderr, got: %q", stderr.String())
	}
}

func TestExecList_MutualExclusion_JSONandCSV(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--json", "--csv"}, &stdout, &stderr, t.TempDir())
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in stderr, got: %q", stderr.String())
	}
}

func TestExecList_MutualExclusion_TSVandCSV(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--tsv", "--csv"}, &stdout, &stderr, t.TempDir())
	if code == 0 {
		t.Error("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in stderr, got: %q", stderr.String())
	}
}

// --- Invalid status ---

func TestExecList_InvalidStatus(t *testing.T) {
	specsDir := makeTmpSpecs(t)
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--status", "bogus"}, &stdout, &stderr, specsDir)
	if code == 0 {
		t.Error("expected non-zero exit code for invalid status")
	}
	if !strings.Contains(stdout.String(), "bogus") {
		t.Errorf("expected stdout to contain 'bogus', got: %q", stdout.String())
	}
}

func TestExecList_PriorityFilter(t *testing.T) {
	specsDir := makeTmpSpecs(t)
	for id, priority := range map[string]int{"done-high": 1, "done-medium": 2} {
		content := fmt.Sprintf("---\nid: %s\nparent: my-spec\ncreated: 2026-01-01T00:00:00Z\npriority: %d\nstatus: done\n---\n# Example\n", id, priority)
		if err := os.WriteFile(filepath.Join(specsDir, "my-spec", "assertions", id+".md"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		name      string
		args      []string
		want      []string
		emptyText string
	}{
		{"unfiltered", nil, []string{"done-high", "done-medium", "my-assertion"}, ""},
		{"high-table", []string{"--priority", "1"}, []string{"done-high", "my-assertion"}, ""},
		{"medium-tsv", []string{"--priority", "2", "--tsv"}, []string{"done-medium"}, ""},
		{"combined-json", []string{"--priority", "1", "--status", "done", "--json"}, []string{"done-high"}, ""},
		{"combined-csv", []string{"--priority", "1", "--status", "done", "--csv", "--assertions-only", "--long"}, []string{"done-high"}, ""},
		{"zero-json", []string{"--priority", "0", "--json"}, nil, ""},
		{"zero-table", []string{"--priority", "0"}, nil, "No assertions match priority 0.\n"},
		{"empty-status-table", []string{"--status", "draft"}, nil, "No assertions match status 'draft'.\n"},
		{"combined-empty-table", []string{"--priority", "2", "--status", "not_started"}, nil, "No assertions match status 'not_started' and priority 2.\n"},
		{"empty-status-json", []string{"--status", "draft", "--json"}, nil, ""},
		{"combined-empty-json", []string{"--priority", "2", "--status", "not_started", "--json"}, nil, ""},
		{"outside-range-tsv", []string{"--priority", "4", "--tsv"}, nil, "id\tstatus\tpri\tparent\ttitle\n"},
		{"combined-empty-csv", []string{"--priority", "2", "--status", "not_started", "--csv"}, nil, "id,status,pri,parent,title\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := execList(tc.args, &stdout, &stderr, specsDir); code != 0 {
				t.Fatalf("exit %d: %s", code, stderr.String())
			}
			for _, id := range []string{"done-high", "done-medium", "my-assertion"} {
				if strings.Contains(stdout.String(), id) != slices.Contains(tc.want, id) {
					t.Errorf("wrong selection for %s: %s", id, stdout.String())
				}
			}
			if slices.Contains(tc.args, "--json") {
				var result parser.AssertionsFlatOutput
				if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.Assertions == nil || len(result.Assertions) != len(tc.want) {
					t.Fatalf("invalid assertion list: %s (%v)", stdout.String(), err)
				}
			}
			if tc.emptyText != "" && stdout.String() != tc.emptyText {
				t.Fatalf("empty output = %q, want %q", stdout.String(), tc.emptyText)
			}
		})
	}
}

// A malformed command line prints on stderr, like the mutually exclusive
// format flags.
func TestExecList_MalformedPriorityArgs(t *testing.T) {
	for _, args := range [][]string{
		{"--priority"}, {"--priority", "--json"},
		{"--priority=1"}, {"--priority="}, {"--priority", ""},
		{"--priority", "1", "--cross-branch"},
		{"--priority", "1", "--priority"},
		{"--priority", "1", "--priority", "2"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := execList(args, &stdout, &stderr, t.TempDir()); code == 0 || !strings.Contains(stderr.String(), "--priority") || stdout.Len() != 0 {
				t.Fatalf("expected a command-line error on stderr, got exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
			}
		})
	}
}

// A bad filter value prints as JSON on stdout, the same as an invalid
// --status, so one caller reads both in one format.
func TestExecList_InvalidPriorityValue(t *testing.T) {
	for _, args := range [][]string{
		{"--priority", "text"}, {"--priority", "-1"},
		{"--priority", "999999999999999999999999999999"},
		{"--json", "--priority", "text"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := execList(args, &stdout, &stderr, t.TempDir())
			if code == 0 {
				t.Fatalf("expected a nonzero exit, got stdout %q, stderr %q", stdout.String(), stderr.String())
			}
			var out struct {
				Error   bool   `json:"error"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
				t.Fatalf("expected a JSON error on stdout, got %q (%v)", stdout.String(), err)
			}
			if !out.Error || !strings.Contains(out.Message, "--priority") {
				t.Fatalf("expected the message to name --priority, got %q", stdout.String())
			}
		})
	}
}

func TestExecList_MissingStatus(t *testing.T) {
	for _, args := range [][]string{
		{"--status"}, {"--status", ""}, {"--status", "--priority", "1"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := execList(args, &stdout, &stderr, t.TempDir()); code == 0 || !strings.Contains(stderr.String(), "--status") || stdout.Len() != 0 {
				t.Fatalf("expected status error without output, got exit %d, stdout %q, stderr %q", code, stdout.String(), stderr.String())
			}
		})
	}
}

// --- Format-aware empty ---

func TestExecList_EmptyJSON(t *testing.T) {
	emptyDir := t.TempDir()
	for _, dir := range []string{emptyDir, filepath.Join(emptyDir, "missing")} {
		var stdout, stderr bytes.Buffer
		if code := execList([]string{"--json"}, &stdout, &stderr, dir); code != 0 {
			t.Fatalf("exit %d: %s", code, stderr.String())
		}
		var result parser.AssertionsFlatOutput
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.Type != "assertions" || result.Assertions == nil || len(result.Assertions) != 0 {
			t.Fatalf("expected an empty flat list for %s, got %s (%v)", dir, stdout.String(), err)
		}
	}
}

func TestExecList_EmptyTSV(t *testing.T) {
	emptyDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--tsv"}, &stdout, &stderr, emptyDir)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	out := stdout.String()
	if strings.HasPrefix(out, `{"status"`) {
		t.Errorf("expected TSV header, got JSON: %q", out)
	}
	if !strings.HasPrefix(out, "id\t") {
		t.Errorf("expected TSV header starting with 'id\\t', got: %q", out)
	}
}

func TestExecList_EmptyCSV(t *testing.T) {
	emptyDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--csv"}, &stdout, &stderr, emptyDir)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %q", code, stderr.String())
	}
	out := stdout.String()
	if strings.HasPrefix(out, `{"status"`) {
		t.Errorf("expected CSV header, got JSON: %q", out)
	}
	if !strings.HasPrefix(out, "id,") {
		t.Errorf("expected CSV header starting with 'id,', got: %q", out)
	}
}

// --- Sort order consistency ---

func TestListRows_SortedByPriorityThenID(t *testing.T) {
	result := &parser.ParseResult{
		Specs: []parser.Spec{
			{ID: "spec-a"},
			{ID: "spec-b"},
		},
		Assertions: []parser.Assertion{
			{ID: "z-assertion", Parent: "spec-a", Priority: 2, Status: "not_started"},
			{ID: "a-assertion", Parent: "spec-b", Priority: 2, Status: "not_started"},
			{ID: "m-assertion", Parent: "spec-a", Priority: 1, Status: "not_started"},
		},
	}

	rows := listRows(result, true)

	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// m-assertion: priority 1 (lowest) → comes first
	if rows[0].ID != "m-assertion" {
		t.Errorf("row[0] should be m-assertion (priority 1), got %q", rows[0].ID)
	}
	// a-assertion and z-assertion both priority 2: alphabetical order
	if rows[1].ID != "a-assertion" {
		t.Errorf("row[1] should be a-assertion (priority 2, alpha first), got %q", rows[1].ID)
	}
	if rows[2].ID != "z-assertion" {
		t.Errorf("row[2] should be z-assertion (priority 2, alpha last), got %q", rows[2].ID)
	}
}

func TestListRows_MatchesFormatAssertionsFlat(t *testing.T) {
	result := &parser.ParseResult{
		Assertions: []parser.Assertion{
			{ID: "z-assertion", Parent: "spec-a", Priority: 2, Status: "not_started"},
			{ID: "a-assertion", Parent: "spec-b", Priority: 2, Status: "not_started"},
			{ID: "m-assertion", Parent: "spec-a", Priority: 1, Status: "not_started"},
		},
	}

	rows := listRows(result, true)

	data, err := parser.FormatAssertionsFlat(result)
	if err != nil {
		t.Fatalf("FormatAssertionsFlat: %v", err)
	}

	var flatOut struct {
		Assertions []struct {
			ID string `json:"id"`
		} `json:"assertions"`
	}
	if err := json.Unmarshal(data, &flatOut); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(rows) != len(flatOut.Assertions) {
		t.Fatalf("row count mismatch: listRows=%d, FormatAssertionsFlat=%d", len(rows), len(flatOut.Assertions))
	}
	for i := range rows {
		if rows[i].ID != flatOut.Assertions[i].ID {
			t.Errorf("position %d: listRows=%q, FormatAssertionsFlat=%q", i, rows[i].ID, flatOut.Assertions[i].ID)
		}
	}
}

// --- Cross-branch mode tests ---

// gitList runs a raw git command in dir for cross-branch fixtures.
func gitList(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// makeCrossBranchRepo creates a temp git repo where branch "feat/x" adds one
// assertion that main does not have, and chdirs into it (Classify shells out
// in the process cwd).
func makeCrossBranchRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	gitList(t, dir, "init", "-q", "-b", "main")
	gitList(t, dir, "config", "user.email", "test@example.com")
	gitList(t, dir, "config", "user.name", "Test")
	gitList(t, dir, "config", "commit.gpgsign", "false")

	specDir := filepath.Join(dir, "specs", "demo")
	if err := os.MkdirAll(filepath.Join(specDir, "assertions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "demo.md"),
		[]byte("---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitList(t, dir, "add", "-A")
	gitList(t, dir, "commit", "-qm", "base")

	gitList(t, dir, "checkout", "-q", "-b", "feat/x")
	if err := os.WriteFile(filepath.Join(specDir, "assertions", "foreign.md"),
		[]byte("---\nid: foreign\nparent: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\n---\n# Foreign\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitList(t, dir, "add", "-A")
	gitList(t, dir, "commit", "-qm", "add foreign")
	gitList(t, dir, "checkout", "-q", "main")

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestExecList_CrossBranchJSON(t *testing.T) {
	makeCrossBranchRepo(t)
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--cross-branch", "--json"}, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	var rows []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &rows); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, stdout.String())
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %v", len(rows), rows)
	}
	r := rows[0]
	if r["path"] != "specs/demo/assertions/foreign.md" || r["branch"] != "feat/x" || r["state"] != "incoming_add" {
		t.Errorf("unexpected row: %v", r)
	}
}

func TestExecList_CrossBranchFilterExcludes(t *testing.T) {
	makeCrossBranchRepo(t)
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--cross-branch", "--branch-filter", "release/*", "--json"}, &stdout, &stderr, "")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "[]" {
		t.Errorf("expected empty array for non-matching filter, got %s", stdout.String())
	}
}

func TestExecList_CrossBranchRejectsStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--cross-branch", "--status", "done"}, &stdout, &stderr, "")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--status") {
		t.Errorf("expected --status error, got %q", stderr.String())
	}
}

func TestExecList_CrossBranchRejectsSpecsDir(t *testing.T) {
	// Silently ignoring --specs-dir would report success for a scope the
	// command never used: the engine reads the git object store, not a
	// directory.
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--cross-branch", "--specs-dir", "/nonexistent"}, &stdout, &stderr, "")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--specs-dir does not apply") {
		t.Errorf("expected a --specs-dir error, got %q", stderr.String())
	}
}

func TestExecList_CrossBranchRejectsCallerSpecsDir(t *testing.T) {
	// The same guard for the plumbed-in directory, so a future caller cannot
	// reintroduce the silent substitution.
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--cross-branch"}, &stdout, &stderr, t.TempDir())
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--specs-dir does not apply") {
		t.Errorf("expected a --specs-dir error, got %q", stderr.String())
	}
}

func TestExecList_BranchFilterRequiresCrossBranch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execList([]string{"--branch-filter", "feat/*"}, &stdout, &stderr, makeTmpSpecs(t))
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--cross-branch") {
		t.Errorf("expected --cross-branch error, got %q", stderr.String())
	}
}
