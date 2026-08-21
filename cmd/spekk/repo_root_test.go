package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	spekk "github.com/spekk-ai/spekk-cli"
	"github.com/spekk-ai/spekk-cli/internal/install"
)

// gitRepoWithSubdir makes a repository with one subdirectory, chdirs into that
// subdirectory, and returns the repository root and the subdirectory.
func gitRepoWithSubdir(t *testing.T) (root, sub string) {
	t.Helper()
	root = chdirTemp(t)
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Skipf("cannot make a git repository here: %v: %s", err, out)
	}
	// git reports the resolved path, so resolve ours before comparing.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	sub = filepath.Join(root, "src", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	return root, sub
}

// TestRepoRoot_FromASubdirectory: a project is the repository. A command run
// three levels down must name the same project directory as one run at the top.
func TestRepoRoot_FromASubdirectory(t *testing.T) {
	root, sub := gitRepoWithSubdir(t)

	got := repoRoot()
	if got != root {
		t.Errorf("repoRoot() from %s = %s, want the repository root %s", sub, got, root)
	}
}

// TestRepoRoot_OutsideARepository: with no repository there is no root to find,
// so the working directory stands in for the project.
func TestRepoRoot_OutsideARepository(t *testing.T) {
	dir := chdirTemp(t)
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput(); err == nil {
		t.Skipf("the temp directory is inside a repository: %s", out)
	}
	resolved := dir
	if r, err := filepath.EvalSymlinks(dir); err == nil {
		resolved = r
	}

	if got := repoRoot(); got != resolved {
		t.Errorf("repoRoot() = %s, want the working directory %s", got, resolved)
	}
}

// TestFindSpecsDir_FromASubdirectory: specs/ lives at the repository root, and
// project-scope install paths must agree with it.
func TestFindSpecsDir_FromASubdirectory(t *testing.T) {
	root, _ := gitRepoWithSubdir(t)

	want := filepath.Join(root, "specs")
	if got := findSpecsDir(); got != want {
		t.Errorf("findSpecsDir() = %s, want %s", got, want)
	}
}

// TestTargetInstallOptions_ScopeIsTheRepository: the install a user runs from a
// subdirectory must write into the repository, not into the subdirectory.
func TestTargetInstallOptions_ScopeIsTheRepository(t *testing.T) {
	root, sub := gitRepoWithSubdir(t)

	opts := targetInstallOptions("claude-code", true)
	if opts.Cwd != root {
		t.Errorf("a project install from %s is scoped to %s, want the repository root %s", sub, opts.Cwd, root)
	}
}

// TestProjectInstall_FromASubdirectoryWritesToTheRepositoryRoot: the whole path,
// from the working directory to the files on disk. Without the repository root
// the install makes a second copy under the subdirectory, and the user has two
// installs and no warning.
func TestProjectInstall_FromASubdirectoryWritesToTheRepositoryRoot(t *testing.T) {
	root, sub := gitRepoWithSubdir(t)
	home := t.TempDir()

	opts := targetInstallOptions("claude-code", true)
	opts.HomeDir = home
	opts.SkillFS = spekk.EmbeddedFS
	if _, err := install.Install(opts); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(root, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("the project install did not write %s: %v", want, err)
	}
	if _, err := os.Stat(filepath.Join(sub, ".claude")); err == nil {
		t.Errorf("the install made a second copy under %s", sub)
	}
}

// TestReportStale_SeesTheProjectInstallFromASubdirectory: the report follows the
// same rule. A stale project file must not go silent because the user stood in a
// subdirectory of their own repository.
func TestReportStale_SeesTheProjectInstallFromASubdirectory(t *testing.T) {
	root, sub := gitRepoWithSubdir(t)
	home := t.TempDir()

	opts := targetInstallOptions("claude-code", true)
	opts.HomeDir = home
	opts.SkillFS = spekk.EmbeddedFS
	if _, err := install.Install(opts); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	if err := os.WriteFile(p, []byte("edited by hand\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	reportStale(&buf, home, repoRoot())
	if !strings.Contains(buf.String(), p) {
		t.Errorf("the stale project file was not reported from %s:\n%s", sub, buf.String())
	}
}

// TestScopeForInstall_ProjectIsTheRepository: the install and the report share
// this one answer, so neither can disagree about where the project is.
func TestScopeForInstall_ProjectIsTheRepository(t *testing.T) {
	root, sub := gitRepoWithSubdir(t)

	home, project := scopeForInstall()
	if project != root {
		t.Errorf("from %s the project is %s, want the repository root %s", sub, project, root)
	}
	wantHome, _ := os.UserHomeDir()
	if home != wantHome {
		t.Errorf("home = %s, want %s", home, wantHome)
	}
}
