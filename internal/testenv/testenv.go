// Package testenv holds helpers that keep tests away from the developer's
// own machine state: the real config directory and the global git config.
// Only tests import it.
package testenv

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// Snapshot records every file under dir with its size and modification time.
// A missing dir gives an empty snapshot.
func Snapshot(dir string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out[path] = info.ModTime().Format(time.RFC3339Nano) + " " + strconv.FormatInt(info.Size(), 10)
		return nil
	})
	return out
}

// GuardDir snapshots dir now and returns a function that reports whether the
// files under it changed since. Call it before a test run changes any
// environment variable, so dir is the developer's real directory.
func GuardDir(dir string) (changed func() []string) {
	before := Snapshot(dir)
	return func() []string {
		after := Snapshot(dir)
		var diff []string
		for path, v := range after {
			if before[path] != v {
				diff = append(diff, path)
			}
		}
		for path := range before {
			if _, ok := after[path]; !ok {
				diff = append(diff, path)
			}
		}
		return diff
	}
}

// gitIsolation is the environment that makes git ignore the developer's
// setup. GIT_CONFIG_GLOBAL (set by IsolateGit) replaces both ~/.gitconfig and
// $XDG_CONFIG_HOME/git/config; GIT_CONFIG_SYSTEM replaces the system file.
// user.useConfigOnly stops git from inventing an identity out of the login
// name and the hostname, which is what a macOS machine does when no identity
// is configured: a repository the test leaves without user.name and
// user.email then really has none, on every OS.
func gitIsolation(emptyConfig string) map[string]string {
	return map[string]string{
		"GIT_CONFIG_SYSTEM":   emptyConfig,
		"GIT_CONFIG_NOSYSTEM": "1",
		"GIT_CONFIG_COUNT":    "1",
		"GIT_CONFIG_KEY_0":    "user.useConfigOnly",
		"GIT_CONFIG_VALUE_0":  "true",
	}
}

var gitIdentityVars = []string{
	"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_AUTHOR_DATE",
	"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "GIT_COMMITTER_DATE",
	"EMAIL", "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE",
}

// IsolateGit makes every git subprocess of this test ignore the developer's
// global and system git config. It changes only the test's environment, never
// production code, which keeps reading the user's real config.
func IsolateGit(t testing.TB) {
	t.Helper()
	empty := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", empty)
	for k, v := range gitIsolation(empty) {
		t.Setenv(k, v)
	}
	for _, k := range gitIdentityVars {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

// Main is the body of TestMain for a package whose tests run git or reach the
// app store. It isolates git for the whole package run, then runs the tests
// and fails the run when any file under the real user config directory's
// spekk directory changed.
func Main(m *testing.M) int {
	dir, err := os.MkdirTemp("", "spekk-testenv-")
	if err != nil {
		os.Stderr.WriteString("testenv: " + err.Error() + "\n")
		return 1
	}
	defer os.RemoveAll(dir)
	empty := filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		os.Stderr.WriteString("testenv: " + err.Error() + "\n")
		return 1
	}
	changed := func() []string { return nil }
	if cfg, err := os.UserConfigDir(); err == nil {
		changed = GuardDir(filepath.Join(cfg, "spekk"))
	}
	// HOME and XDG_CONFIG_HOME move to an empty directory instead of
	// GIT_CONFIG_GLOBAL, because tests that give themselves a HOME write
	// their own global config there, and GIT_CONFIG_GLOBAL would override it.
	// A test that sets its own HOME still wins.
	os.Setenv("HOME", dir)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	for k, v := range gitIsolation(empty) {
		os.Setenv(k, v)
	}
	for _, k := range append([]string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_PARAMETERS"}, gitIdentityVars...) {
		os.Unsetenv(k)
	}
	code := m.Run()
	if diff := changed(); len(diff) > 0 {
		os.Stderr.WriteString("tests changed files under the real user config directory:\n")
		for _, p := range diff {
			os.Stderr.WriteString("  " + p + "\n")
		}
		return 1
	}
	return code
}
