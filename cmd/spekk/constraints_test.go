package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func constraintProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeConstraintFile(t, root, "specs/sample/sample.md", "---\nid: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Sample\n")
	for _, a := range []struct{ id, status, priority, anchors string }{
		{"alpha", "done", "2", "anchors: [code.go, other.go]\n"},
		{"beta", "not_started", "1", ""},
		{"gamma", "draft", "1", ""},
	} {
		content := "---\nid: " + a.id + "\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: " + a.priority + "\nstatus: " + a.status + "\n" + a.anchors + "---\n# " + a.id + " title\n"
		writeConstraintFile(t, root, "specs/sample/assertions/"+a.id+".md", content)
	}
	writeConstraintFile(t, root, "code.go", "package sample\n// spekk: alpha\n// spekk: beta\n// spekk: alpha\n")
	writeConstraintFile(t, root, "other.go", "package sample\n")
	writeConstraintFile(t, root, "checks_test.go", "package sample\nimport \"testing\"\n// spekk: beta evidence=example\nfunc TestBeta(t *testing.T) {}\n")
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.test")
	git("config", "user.name", "Test")
	git("add", ".")
	git("commit", "-q", "-m", "Add fixture")
	return root
}

func writeConstraintFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func constraintOutput(t *testing.T, specsDir string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := execConstraints(args, &stdout, &stderr, specsDir)
	return code, stdout.String(), stderr.String()
}

func TestConstraintsListsAnchoredProperties(t *testing.T) {
	root := constraintProject(t)
	specsDir := filepath.Join(root, "specs")
	code, table, diagnostic := constraintOutput(t, specsDir, filepath.Join(root, "code.go"))
	if code != 0 || diagnostic != "" {
		t.Fatalf("table: code %d, stderr %q", code, diagnostic)
	}
	lines := strings.Split(strings.TrimSpace(table), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], "ID") || !strings.Contains(lines[0], "STATUS") || !strings.Contains(lines[0], "TRUTH") || !strings.Contains(lines[0], "LINES") || !strings.Contains(lines[0], "TITLE") {
		t.Fatalf("table columns and row count: %q", table)
	}
	if !strings.Contains(lines[1], "beta") || !strings.Contains(lines[1], "not_started") || !strings.Contains(lines[1], "unchecked") || !strings.Contains(lines[1], "3") || !strings.Contains(lines[1], "beta title") {
		t.Fatalf("priority order or beta row: %q", table)
	}
	if !strings.Contains(lines[2], "alpha") || !strings.Contains(lines[2], "done") || !strings.Contains(lines[2], "2, 4") || !strings.Contains(lines[2], "alpha title") {
		t.Fatalf("deduplicated alpha row: %q", table)
	}
	code, output, diagnostic := constraintOutput(t, specsDir, filepath.Join(root, "code.go"), "--json")
	var rows []map[string]any
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(output), &rows) != nil || len(rows) != 2 {
		t.Fatalf("JSON: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	if len(rows[0]) != 7 || rows[0]["id"] != "beta" || rows[0]["status"] != "not_started" || rows[0]["truth"] != "unchecked" || rows[0]["level"] != "" || rows[0]["dirty"] != false || rows[0]["title"] != "beta title" || len(rows[0]["lines"].([]any)) != 1 {
		t.Fatalf("JSON beta row: %#v", rows[0])
	}
	if rows[1]["id"] != "alpha" || len(rows[1]["lines"].([]any)) != 2 {
		t.Fatalf("JSON alpha row: %#v", rows[1])
	}
	code, output, diagnostic = constraintOutput(t, specsDir, filepath.Join(root, "other.go"), "--json")
	var frontmatterRows []map[string]any
	if code != 0 || diagnostic != "" || json.Unmarshal([]byte(output), &frontmatterRows) != nil || len(frontmatterRows) != 1 || frontmatterRows[0]["id"] != "alpha" || len(frontmatterRows[0]["lines"].([]any)) != 0 {
		t.Fatalf("front matter row: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	code, output, diagnostic = constraintOutput(t, specsDir, filepath.Join(root, "checks_test.go"))
	if code != 0 || output != "" || diagnostic != "no properties anchor on checks_test.go\n" {
		t.Fatalf("test marker must not anchor: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	code, output, diagnostic = constraintOutput(t, specsDir, filepath.Join(root, "checks_test.go"), "--json")
	if code != 0 || output != "[]\n" || diagnostic != "" {
		t.Fatalf("empty JSON: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	code, output, diagnostic = constraintOutput(t, specsDir, filepath.Join(root, "future.go"))
	if code != 0 || output != "" || diagnostic != "no properties anchor on future.go\n" {
		t.Fatalf("missing file: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	code, _, diagnostic = constraintOutput(t, specsDir, filepath.Join(filepath.Dir(root), "outside.go"))
	if code != 1 || !strings.Contains(diagnostic, "outside project root") {
		t.Fatalf("outside path: code %d, stderr %q", code, diagnostic)
	}
	if err := os.Symlink(filepath.Dir(root), filepath.Join(root, "outside-link")); err == nil {
		code, _, diagnostic = constraintOutput(t, specsDir, filepath.Join(root, "outside-link", "future.go"))
		if code != 1 || !strings.Contains(diagnostic, "outside project root") {
			t.Fatalf("outside symlink path: code %d, stderr %q", code, diagnostic)
		}
	}
	if code, output, _ := constraintOutput(t, specsDir, "--help"); code != 0 || !strings.Contains(output, "spekk constraints <file>") || !strings.Contains(helpText, "constraints") {
		t.Fatalf("help: code %d, stdout %q", code, output)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	code, output, diagnostic = constraintOutput(t, specsDir, "code.go")
	if code != 0 || diagnostic != "" || output != table {
		t.Fatalf("relative path: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
}

func TestConstraintsReadsCurrentFile(t *testing.T) {
	root := constraintProject(t)
	specsDir := filepath.Join(root, "specs")
	code, output, _ := constraintOutput(t, specsDir, filepath.Join(root, "code.go"))
	if code != 0 || strings.Contains(output, "gamma") {
		t.Fatalf("before edit: code %d, stdout %q", code, output)
	}
	writeConstraintFile(t, root, "code.go", "package sample\n// spekk: alpha\n// spekk: beta\n// spekk: gamma\n")
	code, output, diagnostic := constraintOutput(t, specsDir, filepath.Join(root, "code.go"))
	if code != 0 || diagnostic != "" || strings.Index(output, "beta") >= strings.Index(output, "gamma") {
		t.Fatalf("same-priority order after add: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	writeConstraintFile(t, root, "code.go", "package sample\n// spekk: gamma\n// spekk: unknown-id\n")
	code, output, diagnostic = constraintOutput(t, specsDir, filepath.Join(root, "code.go"))
	if code != 0 || !strings.Contains(output, "gamma") || !strings.Contains(output, "2") || !strings.Contains(output, "alpha") || strings.Contains(output, "beta") || strings.Contains(output, "unknown-id") || !strings.Contains(diagnostic, "code.go:3") || !strings.Contains(diagnostic, "unknown-id") {
		t.Fatalf("after edit: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	writeConstraintFile(t, root, "code.go", "package sample\n")
	code, output, diagnostic = constraintOutput(t, specsDir, filepath.Join(root, "code.go"))
	if code != 0 || !strings.Contains(output, "alpha") || strings.Contains(output, "gamma") || strings.Contains(output, "beta") || diagnostic != "" {
		t.Fatalf("after removal: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
}

func TestConstraintsShowsRecordedTruth(t *testing.T) {
	specsDir := truthProject(t, true)
	root := filepath.Dir(specsDir)
	writeConstraintFile(t, root, "code.go", "package fixture\n// spekk: pass\n")
	code, table, diagnostic := constraintOutput(t, specsDir, filepath.Join(root, "code.go"))
	if code != 0 || !strings.Contains(diagnostic, "Results are from a run with uncommitted changes") || !strings.Contains(table, "pass") || !strings.Contains(table, "property · verified") {
		t.Fatalf("recorded truth: code %d, stdout %q, stderr %q", code, table, diagnostic)
	}
}
