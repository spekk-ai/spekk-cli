package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const testRepositoryID = "11111111-1111-4111-8111-111111111111"

func gitForRepositoryTest(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", args[0], err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestRepositoryAccessUsesAgentAuthAndChecksAnswer(t *testing.T) {
	var calls int
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/agents/agent-1/repository-access/" || r.Header.Get("Authorization") != "Bearer agent-secret" {
			t.Errorf("access request has wrong method, path, or auth")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["agent_session_id"] != "session-1" || body["repository_id"] != testRepositoryID || body["ref"] != "main" {
			t.Errorf("access request has wrong selection: %v", body)
		}
		w.WriteHeader(status)
		fmt.Fprint(w, `{"repository":"acme/fiction","ref":"main","token":"secret-token"}`)
	}))
	defer server.Close()
	cfg := Config{Host: strings.TrimPrefix(server.URL, "http://"), Token: "agent-secret", AgentID: "agent-1"}
	selection := RepositoryRequest{ID: testRepositoryID, Ref: "main"}
	if _, err := requestRepositoryAccess(context.Background(), cfg, "session-1", selection); err != nil {
		t.Fatal(err)
	}
	status = http.StatusForbidden
	if _, err := requestRepositoryAccess(context.Background(), cfg, "session-1", selection); err == nil || !strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("refused access = %v", err)
	}
	if calls != 2 {
		t.Fatalf("access calls = %d, want 2", calls)
	}
	selection.Ref = "-bad"
	if _, err := requestRepositoryAccess(context.Background(), cfg, "session-1", selection); err == nil || calls != 2 {
		t.Fatalf("bad ref reached server: %v", err)
	}
}

func TestRepositoryCredentialIsScopedToApprovedPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"repository":"acme/fiction","ref":"main","token":"secret-token"}`)
	}))
	defer server.Close()
	useAgentEnvFile(t, server.URL)
	t.Setenv("SPEKK_REPOSITORY_SESSION_ID", "session-1")
	t.Setenv("SPEKK_REPOSITORY_ID", testRepositoryID)
	t.Setenv("SPEKK_REPOSITORY_REF", "main")
	for _, path := range []string{"acme/other.git", "acme/fiction.git"} {
		var out, errOut strings.Builder
		code := runGitCredential([]string{"get"}, strings.NewReader("protocol=https\nhost=github.com\npath="+path+"\n\n"), &out, &errOut)
		if path == "acme/other.git" && (code == 0 || out.Len() != 0 || strings.Contains(errOut.String(), "secret-token")) {
			t.Fatalf("other repository got credential: code=%d output=%q error=%q", code, out.String(), errOut.String())
		}
		if path == "acme/fiction.git" && (code != 0 || out.String() != "username=x-access-token\npassword=secret-token\n") {
			t.Fatalf("approved repository did not get credential: code=%d output=%q error=%q", code, out.String(), errOut.String())
		}
	}
}

func TestPrepareRepositoryClonesFetchesAndProtectsLocalChanges(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitForRepositoryTest(t, home, "init", "-q", "--bare", bare)
	gitForRepositoryTest(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	work := t.TempDir()
	gitForRepositoryTest(t, work, "init", "-q", "-b", "main")
	gitForRepositoryTest(t, work, "config", "user.email", "test@example.com")
	gitForRepositoryTest(t, work, "config", "user.name", "Test User")
	gitForRepositoryTest(t, work, "remote", "add", "origin", bare)
	writeCommit := func(content string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(work, "file.txt"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		gitForRepositoryTest(t, work, "add", "file.txt")
		gitForRepositoryTest(t, work, "commit", "-qm", "Update file")
		gitForRepositoryTest(t, work, "push", "-q", "origin", "main")
		return gitForRepositoryTest(t, work, "rev-parse", "HEAD")
	}
	first := writeCommit("first")
	gitForRepositoryTest(t, home, "config", "--global", "url.file://"+bare+".insteadOf", "https://github.com/acme/fiction.git")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"repository":"acme/fiction","ref":"main","token":"secret-token"}`)
	}))
	defer server.Close()
	cfg := Config{Host: strings.TrimPrefix(server.URL, "http://"), Token: "agent-secret", AgentID: "agent-1", Workspace: t.TempDir()}
	selection := RepositoryRequest{ID: testRepositoryID, Ref: "main"}
	dir, err := prepareRepository(context.Background(), cfg, "session-1", selection)
	if err != nil {
		t.Fatal(err)
	}
	if got := gitForRepositoryTest(t, dir, "rev-parse", "HEAD"); got != first {
		t.Fatalf("first checkout = %s, want %s", got, first)
	}
	second := writeCommit("second")
	if _, err := prepareRepository(context.Background(), cfg, "session-1", selection); err != nil {
		t.Fatal(err)
	}
	if got := gitForRepositoryTest(t, dir, "rev-parse", "HEAD"); got != second {
		t.Fatalf("fetched checkout = %s, want %s", got, second)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("local work"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareRepository(context.Background(), cfg, "session-1", selection); err == nil || !strings.Contains(err.Error(), "local changes") {
		t.Fatalf("dirty checkout was not refused: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil || string(data) != "local work" {
		t.Fatalf("local changes were lost: %v, %q", err, data)
	}
}

func TestRepositoryChatStartsClaudeInSelectedCheckout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ANTHROPIC_API_KEY", "test-model-key")
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitForRepositoryTest(t, home, "init", "-q", "--bare", bare)
	gitForRepositoryTest(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	work := t.TempDir()
	gitForRepositoryTest(t, work, "init", "-q", "-b", "main")
	gitForRepositoryTest(t, work, "config", "user.email", "test@example.com")
	gitForRepositoryTest(t, work, "config", "user.name", "Test User")
	gitForRepositoryTest(t, work, "remote", "add", "origin", bare)
	if err := os.WriteFile(filepath.Join(work, "file.txt"), []byte("ready"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForRepositoryTest(t, work, "add", "file.txt")
	gitForRepositoryTest(t, work, "commit", "-qm", "Prepare fixture")
	gitForRepositoryTest(t, work, "push", "-q", "origin", "main")
	gitForRepositoryTest(t, home, "config", "--global", "url.file://"+bare+".insteadOf", "https://github.com/acme/fiction.git")

	bin := t.TempDir()
	claude := "#!/bin/sh\n" + claudeVersionAnswer + "cat > /dev/null\nprintf '{\"type\":\"result\",\"session_id\":\"sess-1\",\"result\":\"%s\"}\\n' \"$(pwd)\"\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(claude), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	accepted := make(chan *websocket.Conn, 1)
	frames := make(chan map[string]any, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/agents/agent-1/repository-access/" {
			if r.Header.Get("Authorization") != "Bearer agent-secret" {
				t.Error("repository access request lacks agent auth")
			}
			fmt.Fprint(w, `{"repository":"acme/fiction","ref":"main","token":"secret-token"}`)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		accepted <- conn
		for {
			var frame map[string]any
			if err := wsjson.Read(r.Context(), conn, &frame); err != nil {
				return
			}
			frames <- frame
		}
	}))
	defer server.Close()
	workspace := t.TempDir()
	cfg := Config{Host: "localhost:" + strings.TrimPrefix(server.URL, "http://127.0.0.1:"), Token: "agent-secret", AgentID: "agent-1", Workspace: workspace}
	client := NewAgentClient(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)
	var conn *websocket.Conn
	select {
	case conn = <-accepted:
	case <-time.After(10 * time.Second):
		t.Fatal("agent client did not connect")
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	if err := wsjson.Write(ctx, conn, Message{Type: MessageTypeMessage, Text: "Inspect the spec", AgentSessionID: "session-1", Repository: &RepositoryRequest{ID: testRepositoryID, Ref: "main"}}); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(workspace, "repositories", testRepositoryID)
	deadline := time.After(10 * time.Second)
	for {
		select {
		case frame := <-frames:
			if frame["type"] == MessageTypeError {
				t.Fatalf("chat turn failed: %v", frame["error"])
			}
			if frame["type"] == MessageTypeResult {
				if frame["output"] != want {
					t.Fatalf("Claude started in %v, want %s", frame["output"], want)
				}
				return
			}
		case <-deadline:
			t.Fatal("chat turn did not finish")
		}
	}
}
