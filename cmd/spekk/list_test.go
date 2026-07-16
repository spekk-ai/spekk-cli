package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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

// --- Format-aware empty ---

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
