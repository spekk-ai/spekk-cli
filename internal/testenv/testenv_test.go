package testenv

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A developer config that sets an identity, in all three places git reads:
// the global file, XDG, and the system file named by GIT_CONFIG_SYSTEM.
func TestIsolateGitHidesDeveloperIdentity(t *testing.T) {
	ident := "[user]\n\tname = Dev Person\n\temail = dev@example.com\n"
	home := t.TempDir()
	xdg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(xdg, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	system := filepath.Join(t.TempDir(), "system")
	for path, body := range map[string]string{
		filepath.Join(home, ".gitconfig"):   ident,
		filepath.Join(xdg, "git", "config"): ident,
		system:                              ident,
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", system)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "")
	os.Unsetenv("GIT_CONFIG_NOSYSTEM")

	git := func(dir string, args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	repo := t.TempDir()
	if out, err := git(repo, "init", "-q"); err != nil {
		t.Fatal(out)
	}
	if out, _ := git(repo, "config", "user.name"); out != "Dev Person" {
		t.Fatalf("fixture is wrong: the developer config gives no identity (%q)", out)
	}

	IsolateGit(t)

	if out, _ := git(repo, "config", "user.name"); out != "" {
		t.Errorf("user.name = %q after IsolateGit, want none", out)
	}
	if out, err := git(repo, "var", "GIT_AUTHOR_IDENT"); err == nil {
		t.Errorf("git still names an author: %s", out)
	}
}

// Scripts that tests write run under BSD (macOS) and GNU (Linux) userland.
// These flags mean different things there, or exist on only one side.
func TestTestScriptsAvoidFlagsThatDifferBetweenBSDAndGNU(t *testing.T) {
	banned := regexp.MustCompile(`\b(sed -i|readlink -f|date -d|grep -P|stat -c|sed -r)\b`)
	root := filepath.Join("..", "..")
	self, _ := filepath.Abs("testenv_test.go")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if abs, _ := filepath.Abs(path); abs == self {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			if m := banned.FindString(line); m != "" {
				t.Errorf("%s:%d uses %q, which differs between BSD and GNU", path, i+1, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
