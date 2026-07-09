package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	// Create initial commit so HEAD exists
	f := filepath.Join(dir, "README.md")
	os.WriteFile(f, []byte("init"), 0o644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func TestGitStageAndCommit_WithChanges(t *testing.T) {
	dir := initGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create a new file
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0o644)

	committed, err := gitStageAndCommit("test commit")
	if err != nil {
		t.Fatal(err)
	}
	if !committed {
		t.Error("expected commit to be created")
	}

	// Verify commit exists
	out := runGit(t, dir, "log", "--oneline", "-1")
	if !strings.Contains(out, "test commit") {
		t.Errorf("expected commit message in log, got: %s", out)
	}
}

func TestGitStageAndCommit_NoChanges(t *testing.T) {
	dir := initGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	committed, err := gitStageAndCommit("should not commit")
	if err != nil {
		t.Fatal(err)
	}
	if committed {
		t.Error("expected no commit when there are no changes")
	}
}

func TestGitStageSpecsAndCommit_WithSpecs(t *testing.T) {
	dir := initGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create a spec file
	specsDir := filepath.Join(dir, "specs")
	os.MkdirAll(specsDir, 0o755)
	os.WriteFile(filepath.Join(specsDir, "test.md"), []byte("# Test spec"), 0o644)

	committed, err := gitStageSpecsAndCommit("add specs")
	if err != nil {
		t.Fatal(err)
	}
	if !committed {
		t.Error("expected commit to be created for spec changes")
	}
}

func TestGitStageSpecsAndCommit_NoSpecs(t *testing.T) {
	dir := initGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create a non-spec file
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("package main"), 0o644)

	committed, err := gitStageSpecsAndCommit("should not commit")
	if err != nil {
		t.Fatal(err)
	}
	if committed {
		t.Error("expected no commit when there are no spec changes")
	}
}

func TestGitStageSpecsAndCommit_NoChanges(t *testing.T) {
	dir := initGitRepo(t)
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	committed, err := gitStageSpecsAndCommit("should not commit")
	if err != nil {
		t.Fatal(err)
	}
	if committed {
		t.Error("expected no commit when there are no changes")
	}
}

func TestLoopFlags_WatchParsing(t *testing.T) {
	tests := []struct {
		args  []string
		watch bool
	}{
		{nil, false},
		{[]string{}, false},
		{[]string{"--watch"}, true},
		{[]string{"-w"}, true},
	}
	for _, tt := range tests {
		parsed := cli.ParseFlags(tt.args, LoopFlags)
		got := parsed.Bool("watch")
		if got != tt.watch {
			t.Errorf("LoopFlags(%v): watch = %v, want %v", tt.args, got, tt.watch)
		}
	}
}

func TestLoopFlags_IdleTimeoutParsing(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{"--idle-timeout", "60"}, "60"},
		{[]string{"--watch", "--idle-timeout", "300"}, "300"},
		{[]string{"--idle-timeout", "30", "-w"}, "30"},
	}
	for _, tt := range tests {
		parsed := cli.ParseFlags(tt.args, LoopFlags)
		got := parsed.String("idleTimeout")
		if got != tt.want {
			t.Errorf("LoopFlags(%v): idleTimeout = %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestResetAssertionStatus(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test-assertion.md")

	content := `---
id: test-assertion
parent: test-spec
created: 2026-01-20T16:00:00Z
priority: 1
status: in_progress
locked-by: builder-macbook-12345-1706210400
branch: feature/test
---

# Test Assertion

Some content here.
`
	os.WriteFile(filePath, []byte(content), 0o644)

	err := resetAssertionStatus(filePath)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filePath)
	result := string(data)

	if strings.Contains(result, "in_progress") {
		t.Error("expected status to be reset from in_progress")
	}
	if !strings.Contains(result, "status: not_started") {
		t.Error("expected status to be not_started")
	}
	if strings.Contains(result, "locked-by:") {
		t.Error("expected locked-by line to be removed")
	}
	if !strings.Contains(result, "# Test Assertion") {
		t.Error("expected body content to be preserved")
	}
}

func TestCompletionMessage(t *testing.T) {
	tests := []struct {
		count int64
		want  string
	}{
		{0, "No assertions to work on."},
		{1, "Builder loop complete. 1 assertions completed."},
		{5, "Builder loop complete. 5 assertions completed."},
	}
	for _, tt := range tests {
		got := completionMessage(tt.count)
		if got != tt.want {
			t.Errorf("completionMessage(%d) = %q, want %q", tt.count, got, tt.want)
		}
	}
}
