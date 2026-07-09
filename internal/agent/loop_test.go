package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
