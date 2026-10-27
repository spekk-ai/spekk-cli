package sandbox

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Both setup paths must leave Git without a persistent token store.
func TestGitCredentialSetupDoesNotStoreToken(t *testing.T) {
	script, err := os.ReadFile(setupCredentialsScript)
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{
		"sandbox create and provision": buildGitCredentialScript("example-token"),
		"credential setup script":      string(script),
	}
	store := regexp.MustCompile(`credential\.helper\s+["']?store\b`)
	write := regexp.MustCompile(`(?m)(?:>|tee\s+)\s*~?/?.*\.git-credentials`)
	for name, source := range paths {
		t.Run(name, func(t *testing.T) {
			if store.MatchString(source) {
				t.Error("Git credential store is configured")
			}
			if write.MatchString(source) {
				t.Error("the setup writes a Git credential file")
			}
		})
	}
}

// Run both credential paths through real Git. The shims replace only su and gh.
func TestGitCredentialSetupUsesWorkingHelperBeforeRemovingStore(t *testing.T) {
	setup, err := os.ReadFile(setupCredentialsScript)
	if err != nil {
		t.Fatal(err)
	}
	const blockStart = "    # Configure gh CLI for agent user"
	const blockEnd = "    rm -f /home/agent/.git-credentials"
	start := strings.Index(string(setup), blockStart)
	end := strings.Index(string(setup), blockEnd)
	if start < 0 || end < start {
		t.Fatal("credential setup block was not found")
	}
	setupBlock := "set -euo pipefail\n" + string(setup[start:end+len(blockEnd)])
	paths := map[string]string{
		"create and provision": buildGitCredentialScript("ghp_FAKELOGIN"),
		"setup script":         setupBlock,
	}
	for path, script := range paths {
		for _, state := range []string{"fresh", "legacy", "setup-git", "machine"} {
			t.Run(path+"/"+state, func(t *testing.T) {
				h := newGitCredentialHarness(t, state)
				h.runSetup(t, script, false)
				if got := h.fill(t); !strings.Contains(got, "password=ghp_FAKEGH\n") {
					t.Fatalf("Git did not use the gh credential: %q", got)
				}
				if events, err := os.ReadFile(h.events); err != nil || !strings.Contains(string(events), "credential\n") {
					t.Fatalf("setup did not check the helper before removal: events=%q, error=%v", events, err)
				}
				if _, err := os.Stat(h.stale); !os.IsNotExist(err) {
					t.Fatalf("stale credential file remains or cannot be checked: %v", err)
				}
			})
		}
		t.Run(path+"/helper-fails", func(t *testing.T) {
			h := newGitCredentialHarness(t, "legacy")
			h.runSetup(t, script, true)
			if _, err := os.Stat(h.stale); err != nil {
				t.Fatalf("failed setup removed the stale credential file: %v", err)
			}
		})
	}
}

type gitCredentialHarness struct {
	env    []string
	home   string
	stale  string
	events string
}

func newGitCredentialHarness(t *testing.T, state string) gitCredentialHarness {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	home := filepath.Join(dir, "agent")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	h := gitCredentialHarness{home: home, stale: filepath.Join(home, ".git-credentials"), events: filepath.Join(dir, "events")}
	for _, entry := range os.Environ() {
		key := strings.SplitN(entry, "=", 2)[0]
		if key == "HOME" || key == "PATH" || key == "GITHUB_TOKEN" || key == "GH_TOKEN" || key == "GH_CONFIG_DIR" || key == "XDG_CONFIG_HOME" || strings.HasPrefix(key, "GIT_CONFIG_") {
			continue
		}
		h.env = append(h.env, entry)
	}
	h.env = append(h.env, "HOME="+home, "PATH="+bin+":"+os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GITHUB_TOKEN=ghp_FAKELOGIN", "GH_TEST_HOME="+home, "GH_TEST_STALE="+h.stale, "GH_TEST_EVENTS="+h.events)
	writeCredentialShim(t, filepath.Join(bin, "su"), `#!/bin/sh
[ "$1" = - ] && [ "$2" = agent ] && [ "$3" = -c ] || exit 2
HOME="$GH_TEST_HOME" exec /bin/bash -c "$4"
`)
	writeCredentialShim(t, filepath.Join(bin, "gh"), `#!/bin/sh
case "$1 $2 $3" in
    "auth login --with-token") cat >/dev/null ;;
    "auth status ") exit 0 ;;
    "auth git-credential get")
        cat >/dev/null
        if [ "$GH_TEST_REQUIRE_STALE" = 1 ] && [ ! -e "$GH_TEST_STALE" ]; then exit 3; fi
        printf 'credential\n' >> "$GH_TEST_EVENTS"
        printf 'username=x-access-token\n'
        [ "$GH_TEST_NO_PASSWORD" = 1 ] || printf 'password=ghp_FAKEGH\n'
        ;;
    *) exit 2 ;;
esac
`)
	if state != "fresh" {
		if err := os.WriteFile(h.stale, []byte("https://x-access-token:ghp_FAKEOLD@github.com\n"), 0600); err != nil {
			t.Fatal(err)
		}
		h.env = append(h.env, "GH_TEST_REQUIRE_STALE=1")
	} else {
		h.env = append(h.env, "GH_TEST_REQUIRE_STALE=0")
	}
	h.env = append(h.env, "GH_TEST_NO_PASSWORD=0")
	switch state {
	case "fresh":
	case "legacy":
		h.git(t, "config", "--global", "credential.helper", "store")
	case "setup-git":
		h.git(t, "config", "--global", "--add", "credential.https://github.com.helper", "")
		h.git(t, "config", "--global", "--add", "credential.https://github.com.helper", "!/usr/bin/gh auth git-credential")
	case "machine":
		h.git(t, "config", "--global", "credential.https://github.com.helper", "/opt/spekk/agent-client git-credential")
	default:
		t.Fatal("unknown config state")
	}
	return h
}

func writeCredentialShim(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0700); err != nil {
		t.Fatal(err)
	}
}

func (h gitCredentialHarness) git(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = h.env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, out)
	}
}

func (h gitCredentialHarness) runSetup(t *testing.T, script string, noPassword bool) {
	t.Helper()
	script = strings.ReplaceAll(script, "/home/agent/.git-credentials", h.stale)
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = append([]string(nil), h.env...)
	if noPassword {
		cmd.Env = append(cmd.Env, "GH_TEST_NO_PASSWORD=1")
	}
	out, err := cmd.CombinedOutput()
	if noPassword {
		if err == nil || strings.Count(string(out), "spekk:") != 1 {
			t.Fatalf("failed helper must report one spekk error: error=%v output=%q", err, out)
		}
		return
	}
	if err != nil {
		t.Fatalf("credential setup failed: %v: %s", err, out)
	}
}

func (h gitCredentialHarness) fill(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "credential", "fill")
	cmd.Env = append(append([]string(nil), h.env...), "GH_TEST_REQUIRE_STALE=0")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\n\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		config, _ := os.ReadFile(filepath.Join(h.home, ".gitconfig"))
		events, _ := os.ReadFile(h.events)
		t.Fatalf("Git credential fill failed: %v: %s config=%q events=%q", err, out, config, events)
	}
	return string(out)
}
