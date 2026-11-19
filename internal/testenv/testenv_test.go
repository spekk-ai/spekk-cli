package testenv

import (
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
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

// A test that reaches a host outside loopback fails at once and names the
// opt-in variable, and a loopback server still works.
func TestBlockRemoteHostsRefusesAnythingOutsideLoopback(t *testing.T) {
	t.Setenv(OnlineEnv, "")
	origTransport, origResolver := http.DefaultTransport, net.DefaultResolver
	t.Cleanup(func() { http.DefaultTransport, net.DefaultResolver = origTransport, origResolver })
	BlockRemoteHosts()

	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer local.Close()
	if resp, err := http.Get(local.URL); err != nil {
		t.Fatalf("loopback request failed: %v", err)
	} else {
		resp.Body.Close()
	}

	start := time.Now()
	for _, url := range []string{"http://192.0.2.1/", "https://github.com/"} {
		_, err := http.Get(url)
		if err == nil || !strings.Contains(err.Error(), OnlineEnv) {
			t.Errorf("GET %s: error = %v, want a refusal that names %s", url, err, OnlineEnv)
		}
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("refusal took %v, want it immediate", time.Since(start))
	}
}
