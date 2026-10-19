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
		var request struct {
			Ref string `json:"ref"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		fmt.Fprintf(w, `{"repository":"acme/fiction","ref":%q,"token":"secret-token"}`, request.Ref)
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
	gitForRepositoryTest(t, dir, "checkout", "--", "file.txt")
	gitForRepositoryTest(t, work, "checkout", "-qb", "other")
	if err := os.WriteFile(filepath.Join(work, "secret.txt"), []byte("tracked"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForRepositoryTest(t, work, "add", "secret.txt")
	gitForRepositoryTest(t, work, "commit", "-qm", "Add tracked file")
	gitForRepositoryTest(t, work, "push", "-q", "origin", "other")
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "exclude"), []byte("secret.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte("personal"), 0o644); err != nil {
		t.Fatal(err)
	}
	selection.Ref = "other"
	if _, err := prepareRepository(context.Background(), cfg, "session-1", selection); err == nil {
		t.Fatal("checkout replaced an ignored local file")
	}
	data, err = os.ReadFile(filepath.Join(dir, "secret.txt"))
	if err != nil || string(data) != "personal" {
		t.Fatalf("ignored file was lost: %v, %q", err, data)
	}
}

func TestPrepareRepositoryRejectsParentWorktree(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitForRepositoryTest(t, home, "init", "-q", "--bare", bare)
	gitForRepositoryTest(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	remote := "https://github.com/acme/fiction.git"
	gitForRepositoryTest(t, home, "config", "--global", "url.file://"+bare+".insteadOf", remote)
	workspace := t.TempDir()
	gitForRepositoryTest(t, workspace, "init", "-q", "-b", "main")
	gitForRepositoryTest(t, workspace, "config", "user.email", "test@example.com")
	gitForRepositoryTest(t, workspace, "config", "user.name", "Test User")
	gitForRepositoryTest(t, workspace, "remote", "add", "origin", remote)
	if err := os.WriteFile(filepath.Join(workspace, ".gitignore"), []byte("repositories/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForRepositoryTest(t, workspace, "add", ".gitignore")
	gitForRepositoryTest(t, workspace, "commit", "-qm", "Ignore repository checkout")
	selected := filepath.Join(workspace, "repositories", testRepositoryID)
	if err := os.MkdirAll(selected, 0o700); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"repository":"acme/fiction","ref":"main","token":"secret-token"}`)
	}))
	defer server.Close()
	cfg := Config{Host: strings.TrimPrefix(server.URL, "http://"), Token: "agent-secret", AgentID: "agent-1", Workspace: workspace}
	if _, err := prepareRepository(context.Background(), cfg, "session-1", RepositoryRequest{ID: testRepositoryID, Ref: "main"}); err == nil || !strings.Contains(err.Error(), "worktree") {
		t.Fatalf("parent worktree was accepted: %v", err)
	}
	if got := gitForRepositoryTest(t, workspace, "rev-parse", "--abbrev-ref", "HEAD"); got != "main" {
		t.Fatalf("parent branch changed to %s", got)
	}
}

func TestRepositoryTurnsKeepCheckoutUntilClaudeEnds(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(work, "file.txt"), []byte("main"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForRepositoryTest(t, work, "add", "file.txt")
	gitForRepositoryTest(t, work, "commit", "-qm", "Add main file")
	gitForRepositoryTest(t, work, "push", "-q", "origin", "main")
	mainCommit := gitForRepositoryTest(t, work, "rev-parse", "HEAD")
	gitForRepositoryTest(t, work, "checkout", "-qb", "other")
	if err := os.WriteFile(filepath.Join(work, "file.txt"), []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitForRepositoryTest(t, work, "commit", "-qam", "Change other file")
	gitForRepositoryTest(t, work, "push", "-q", "origin", "other")
	otherCommit := gitForRepositoryTest(t, work, "rev-parse", "HEAD")
	gitForRepositoryTest(t, home, "config", "--global", "url.file://"+bare+".insteadOf", "https://github.com/acme/fiction.git")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Ref string `json:"ref"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		fmt.Fprintf(w, `{"repository":"acme/fiction","ref":%q,"token":"secret-token"}`, request.Ref)
	}))
	defer server.Close()
	workspace := t.TempDir()
	cfg := Config{Host: strings.TrimPrefix(server.URL, "http://"), Token: "agent-secret", AgentID: "agent-1", Workspace: workspace}
	started := filepath.Join(home, "started")
	release := filepath.Join(home, "release")
	bin := t.TempDir()
	claude := "#!/bin/sh\n" + claudeVersionAnswer + "cat > /dev/null\n" +
		"if [ \"$(cat file.txt 2>/dev/null)\" = main ]; then\n" +
		"  touch '" + started + "'\n" +
		"  while [ ! -e '" + release + "' ]; do sleep 0.05; done\n" +
		"fi\n" +
		"printf '{\"type\":\"result\",\"session_id\":\"sess-1\",\"result\":\"%s\"}\\n' \"$(pwd)\"\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(claude), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	host := newTestControlHost(t)
	conn, _, err := websocket.Dial(context.Background(), "ws://"+host.host(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	host.nextConn(t)
	conns := newConnHolder()
	conns.set(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		(&Worker{}).invoke(ctx, cfg, conns, Message{AgentSessionID: "session-1", Repository: &RepositoryRequest{ID: testRepositoryID, Ref: "main"}})
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first Claude turn did not start")
		}
		time.Sleep(20 * time.Millisecond)
	}
	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		(&Worker{}).invoke(ctx, cfg, conns, Message{AgentSessionID: "session-2", Repository: &RepositoryRequest{ID: testRepositoryID, Ref: "other"}})
	}()
	select {
	case <-secondDone:
		t.Fatal("second turn ended before the first turn")
	case <-time.After(500 * time.Millisecond):
	}
	selected := filepath.Join(workspace, "repositories", testRepositoryID)
	if got := gitForRepositoryTest(t, selected, "rev-parse", "HEAD"); got != mainCommit {
		t.Fatalf("checkout changed during first turn: %s", got)
	}
	if err := os.WriteFile(release, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstDone:
	case <-time.After(10 * time.Second):
		t.Fatal("first turn did not finish")
	}
	select {
	case <-secondDone:
	case <-time.After(10 * time.Second):
		t.Fatal("second turn did not finish")
	}
	if got := gitForRepositoryTest(t, selected, "rev-parse", "HEAD"); got != otherCommit {
		t.Fatalf("later turn selected %s, want %s", got, otherCommit)
	}
	for i := 0; i < 2; i++ {
		frame := host.awaitFrame(t, MessageTypeResult, 10*time.Second)
		if frame.frame["output"] != selected {
			t.Fatalf("Claude used %v, want %s", frame.frame["output"], selected)
		}
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
				if err := wsjson.Write(ctx, conn, Message{Type: MessageTypeMessage, Text: "Inspect the workspace", AgentSessionID: "session-2"}); err != nil {
					t.Fatal(err)
				}
				for {
					select {
					case frame := <-frames:
						if frame["type"] == MessageTypeError {
							t.Fatalf("generic chat turn failed: %v", frame["error"])
						}
						if frame["type"] == MessageTypeResult {
							if frame["output"] != workspace {
								t.Fatalf("generic chat used %v, want %s", frame["output"], workspace)
							}
							return
						}
					case <-deadline:
						t.Fatal("generic chat turn did not finish")
					}
				}
			}
		case <-deadline:
			t.Fatal("chat turn did not finish")
		}
	}
}
