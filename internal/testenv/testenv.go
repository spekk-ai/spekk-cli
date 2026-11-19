// Package testenv holds helpers that keep tests away from the developer's
// own machine state: the real config directory and the global git config.
// Only tests import it.
package testenv

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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
	// An editor that hosts the terminal points git at a prompt program. A
	// test that makes git ask for a password would then wait for a person
	// to answer a dialog instead of failing.
	"GIT_ASKPASS", "SSH_ASKPASS", "VSCODE_GIT_ASKPASS_NODE", "VSCODE_GIT_ASKPASS_MAIN",
	"VSCODE_GIT_ASKPASS_EXTRA_ARGS", "VSCODE_GIT_IPC_HANDLE",
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

// OnlineEnv names the variable that lets a test reach a real remote service.
// A test that can only run against one skips unless it is set to 1.
const OnlineEnv = "SPEKK_TEST_ONLINE"

// RequireOnline skips the test unless OnlineEnv is set.
func RequireOnline(t testing.TB) {
	t.Helper()
	if os.Getenv(OnlineEnv) != "1" {
		t.Skip("needs a real remote service; set " + OnlineEnv + "=1 to run it")
	}
}

// BlockRemoteHosts makes the default HTTP transport and the default resolver
// refuse anything outside loopback, at once and with a message that names the
// host, instead of waiting for DNS or a connect timeout. Hostnames other than
// localhost fail to resolve, and a dial to a non-loopback address fails.
// Tests that build their own client with their own transport are not covered.
func BlockRemoteHosts() {
	if os.Getenv(OnlineEnv) == "1" {
		return
	}
	refuse := func(addr string) error {
		return fmt.Errorf("test tried to reach %s; tests may only use loopback (set %s=1 for a test that needs the real service)", addr, OnlineEnv)
	}
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, refuse("a DNS server")
		},
	}
	guard := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if host != "localhost" {
			if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
				return nil, refuse(addr)
			}
		}
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	if t, ok := http.DefaultTransport.(*http.Transport); ok {
		t = t.Clone()
		t.DialContext = guard
		t.Proxy = nil
		http.DefaultTransport = t
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
	keepGoEnv()
	os.Setenv("HOME", dir)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	for k, v := range gitIsolation(empty) {
		os.Setenv(k, v)
	}
	for _, k := range append([]string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_PARAMETERS"}, gitIdentityVars...) {
		os.Unsetenv(k)
	}
	os.Setenv("GIT_TERMINAL_PROMPT", "0")
	BlockRemoteHosts()
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

// keepGoEnv pins where the go command keeps its caches before HOME moves. A
// test that runs `go build` would otherwise start from an empty module cache
// and fetch every dependency from the network.
func keepGoEnv() {
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	out, err := exec.Command(goBin, "env", "GOCACHE", "GOMODCACHE", "GOPATH", "GOFLAGS").Output()
	if err != nil {
		return
	}
	lines := strings.Split(string(out), "\n")
	for i, name := range []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOFLAGS"} {
		if i < len(lines) && lines[i] != "" && os.Getenv(name) == "" {
			os.Setenv(name, lines[i])
		}
	}
}
