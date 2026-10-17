package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestMain gives each test a model credential, as a sandbox that can run a
// turn has. A test of the sign-in state takes it away with t.Setenv.
func TestMain(m *testing.M) {
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Exit(m.Run())
}

// noClaudeCredential clears each source of a model credential.
func noClaudeCredential(t *testing.T) (home string) {
	t.Helper()
	for _, v := range claudeAuthEnvVars {
		t.Setenv(v, "")
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home = t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeCredentials(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeSignedIn(t *testing.T) {
	cases := map[string]struct {
		set  func(t *testing.T, home string)
		want bool
	}{
		"nothing": {func(t *testing.T, home string) {}, false},
		"sign-in file in HOME": {func(t *testing.T, home string) {
			writeCredentials(t, filepath.Join(home, ".claude"))
		}, true},
		"sign-in file in CLAUDE_CONFIG_DIR": {func(t *testing.T, home string) {
			dir := filepath.Join(home, "config")
			t.Setenv("CLAUDE_CONFIG_DIR", dir)
			writeCredentials(t, dir)
		}, true},
		// Claude Code reads only CLAUDE_CONFIG_DIR when it is set.
		"sign-in file in HOME, CLAUDE_CONFIG_DIR elsewhere": {func(t *testing.T, home string) {
			writeCredentials(t, filepath.Join(home, ".claude"))
			t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "config"))
		}, false},
	}
	for _, v := range claudeAuthEnvVars {
		cases[v] = struct {
			set  func(t *testing.T, home string)
			want bool
		}{func(t *testing.T, home string) { t.Setenv(v, "1") }, true}
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			home := noClaudeCredential(t)
			tc.set(t, home)
			if got := claudeSignedIn(); got != tc.want {
				t.Errorf("claudeSignedIn() = %v, want %v", got, tc.want)
			}
		})
	}
}

// A turn on a sandbox with no model credential is refused before claude
// starts, with an error that tells the person to sign in.
func TestTurnRefusedWithoutClaudeSignIn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake claude is a shell script")
	}
	noClaudeCredential(t)
	dir := t.TempDir()
	ran := filepath.Join(dir, "ran")
	script := "#!/bin/sh\n" + claudeVersionAnswer + "touch '" + ran + "'\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake claude: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	host := newTestControlHost(t)
	client := NewAgentClient(Config{Token: "t", Host: host.host(), Workspace: t.TempDir()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	dispatchTurn(t, host.nextConn(t))
	got := host.awaitFrame(t, MessageTypeError, 20*time.Second)
	if got.frame["error"] != signInRequiredDetail || got.frame["agent_session_id"] != "agent-session-1" {
		t.Errorf("error frame = %v", got.frame)
	}
	if _, err := os.Stat(ran); err == nil {
		t.Error("claude ran a turn with no model credential")
	}
}
