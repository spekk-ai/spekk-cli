package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket/wsjson"
)

const testAttachmentID = "00000000-0000-4000-8000-000000000001"
const otherAttachmentID = "00000000-0000-4000-8000-000000000002"

func testAttachmentConfig(t *testing.T, handler http.HandlerFunc) Config {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return Config{Host: strings.TrimPrefix(server.URL, "http://"), Token: "synthetic-token", Workspace: t.TempDir()}
}

func requireNoPartialAttachments(t *testing.T, workspace string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(workspace, ".attachments"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("partial output remains: %v", entries)
	}
}

func TestAttachmentBytesArePrivateAndUnique(t *testing.T) {
	text := []byte("synthetic-unique-text-amber-481\n")
	image := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 1, 2, 3}
	bodies := map[string][]byte{testAttachmentID: text, otherAttachmentID: image}
	cfg := testAttachmentConfig(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-token" {
			t.Error("missing scoped authorization")
		}
		id := strings.Split(r.URL.Path, "/")[4]
		if body, ok := bodies[id]; ok {
			_, _ = w.Write(body)
		} else {
			http.NotFound(w, r)
		}
	})
	attachments := []Attachment{{ID: testAttachmentID, Filename: "same.txt"}, {ID: otherAttachmentID, Filename: "same.txt"}}
	seen := make(map[string]bool)
	for range 2 {
		paths, err := downloadAttachments(context.Background(), cfg, attachments)
		if err != nil {
			t.Fatal(err)
		}
		for index, path := range paths {
			if seen[path] || !strings.HasPrefix(path, filepath.Join(cfg.Workspace, ".attachments")+string(os.PathSeparator)) {
				t.Fatalf("unsafe or reused path: %s", path)
			}
			seen[path] = true
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, bodies[attachments[index].ID]) {
				t.Fatalf("downloaded bytes differ: %v", err)
			}
			for _, entry := range []struct {
				path string
				mode os.FileMode
			}{{path, 0600}, {filepath.Dir(path), 0700}} {
				info, err := os.Stat(entry.path)
				if err != nil || info.Mode().Perm() != entry.mode {
					t.Fatalf("private mode is wrong for %s: %v", entry.path, err)
				}
			}
		}
	}
}

func TestAttachmentBatchFailureRemovesEarlierFiles(t *testing.T) {
	for _, kind := range []string{"status", "redirect", "oversize", "truncated", "empty"} {
		t.Run(kind, func(t *testing.T) {
			cfg := testAttachmentConfig(t, func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, testAttachmentID) {
					_, _ = w.Write([]byte("first file"))
					return
				}
				switch kind {
				case "status":
					w.WriteHeader(http.StatusForbidden)
				case "redirect":
					http.Redirect(w, r, "/unexpected", http.StatusFound)
				case "oversize":
					w.Header().Set("Content-Length", fmt.Sprint(maxAttachmentBytes+1))
				case "truncated":
					w.Header().Set("Content-Length", "100")
					_, _ = w.Write([]byte("short"))
				case "empty":
					w.WriteHeader(http.StatusOK)
				}
			})
			paths, err := downloadAttachments(context.Background(), cfg, []Attachment{
				{ID: testAttachmentID, Filename: "first.txt"},
				{ID: otherAttachmentID, Filename: "second.txt"},
			})
			if err == nil || len(paths) != 0 {
				t.Fatalf("failed batch returned paths: %v, %v", paths, err)
			}
			requireNoPartialAttachments(t, cfg.Workspace)
		})
	}
}

func TestAttachmentDownloadBoundsStreamingBytesAndContext(t *testing.T) {
	const budget = 8
	cfg := testAttachmentConfig(t, func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush()
		_, _ = w.Write(bytes.Repeat([]byte("x"), budget+1))
	})
	target := "http://" + cfg.Host + "/attachment"
	for _, canceled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if canceled {
			cancel()
		}
		_, err := downloadAttachment(ctx, http.DefaultClient, target, cfg.Token, filepath.Join(t.TempDir(), "file"), budget)
		cancel()
		if err == nil {
			t.Fatal("unbounded or canceled download was accepted")
		}
	}
}

func TestAttachmentInputRefusal(t *testing.T) {
	cfg := testAttachmentConfig(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid attachment reached HTTP")
	})
	for _, attachment := range []Attachment{
		{ID: "../escape", Filename: "notes.txt"},
		{ID: testAttachmentID, Filename: "../notes.txt"},
		{ID: testAttachmentID, Filename: "..\\notes.txt"},
		{ID: testAttachmentID, Filename: "notes\n.txt"},
		{ID: testAttachmentID, Filename: ""},
	} {
		if _, err := downloadAttachments(context.Background(), cfg, []Attachment{attachment}); err == nil {
			t.Fatalf("accepted invalid attachment: %#v", attachment)
		}
	}
	if _, err := downloadAttachments(context.Background(), cfg, make([]Attachment, maxAttachmentCount+1)); err == nil {
		t.Fatal("accepted too many attachments")
	}
	requireNoPartialAttachments(t, cfg.Workspace)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(cfg.Workspace, ".attachments")); err != nil {
		t.Fatal(err)
	}
	if _, err := downloadAttachments(context.Background(), cfg, []Attachment{{ID: testAttachmentID, Filename: "notes.txt"}}); err == nil {
		t.Fatal("accepted a symlink attachment root")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) > 0 {
		t.Fatalf("symlink target changed: %v", err)
	}
}

func TestAttachmentHostAndLongName(t *testing.T) {
	for _, host := range []string{"example.test/path", "user@example.test", "example.test?token=x", "example.test#part"} {
		if _, err := attachmentBaseURL(host); err == nil {
			t.Fatalf("accepted unsafe host %q", host)
		}
	}
	for host, scheme := range map[string]string{"example.test": "https", "localhost:8000": "http", "127.0.0.1:8000": "http", "localhost.example.test": "https"} {
		parsed, err := attachmentBaseURL(host)
		if err != nil || parsed.Scheme != scheme {
			t.Fatalf("wrong scheme for %q: %v", host, err)
		}
	}
	name, err := safeAttachmentName(strings.Repeat("x", maxAttachmentNameBytes) + ".png")
	if err != nil || len(name) > maxAttachmentNameBytes || !strings.HasSuffix(name, ".png") {
		t.Fatalf("long name failed: %q, %v", name, err)
	}
}

func TestAttachmentFailureRefusesTheTurnBeforeModelStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake CLI is a shell script")
	}
	marker := filepath.Join(t.TempDir(), "model-started")
	dir := t.TempDir()
	script := "#!/bin/sh\n" + claudeVersionAnswer + "touch '" + marker + "'\ncat >/dev/null\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	host := newTestControlHost(t)
	workspace := t.TempDir()
	client := NewAgentClient(Config{Token: "synthetic-token", Host: host.host(), Workspace: workspace})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)
	conn := host.nextConn(t)
	err := wsjson.Write(ctx, conn, map[string]any{
		"type":             MessageTypeMessage,
		"text":             "Use the required file",
		"agent_session_id": "attachment-session",
		"attachments":      []Attachment{{ID: testAttachmentID, Filename: "required.txt"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := host.awaitFrame(t, MessageTypeError, 20*time.Second)
	reason, _ := got.frame["error"].(string)
	if !strings.HasPrefix(reason, "Attachment input failed:") {
		t.Fatalf("wrong refusal: %s", reason)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("model started after attachment failure: %v", err)
	}
	requireNoPartialAttachments(t, workspace)
}

func TestAttachmentInvocationKeepsFilesWithTheirMessage(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the test harness is a shell script")
	}
	workspace := t.TempDir()
	dir := t.TempDir()
	script := "#!/bin/sh\n" + claudeVersionAnswer + `set -eu
cat > prompt
sed -n 's/^  - //p' prompt | while IFS= read -r file; do cat "$file"; done > harness-input
echo '{"type":"result","session_id":"harness-session","result":"files read"}'
`
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", t.TempDir())

	text := []byte("synthetic text amber 481\n")
	image := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 1, 2, 3}
	host := newTestControlHost(t)
	websocketHandler := host.srv.Config.Handler
	host.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/agents/attachments/") {
			websocketHandler.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer synthetic-token" || r.URL.RawQuery != "" {
			t.Error("download request did not protect its credential")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/agents/attachments/" + testAttachmentID + "/download/":
			_, _ = w.Write(text)
		case "/api/agents/attachments/" + otherAttachmentID + "/download/":
			_, _ = w.Write(image)
		default:
			http.NotFound(w, r)
		}
	})
	client := NewAgentClient(Config{Token: "synthetic-token", Host: host.host(), Workspace: workspace})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)
	conn := host.nextConn(t)
	var earlierPaths []string
	for turn, attachments := range [][]Attachment{
		{{ID: testAttachmentID, Filename: "same name.txt"}, {ID: otherAttachmentID, Filename: "same name.txt"}},
		{{ID: otherAttachmentID, Filename: "same name.txt"}},
	} {
		session := fmt.Sprintf("attachment-session-%d", turn)
		message := fmt.Sprintf("Read the files for message %d", turn)
		if err := wsjson.Write(ctx, conn, Message{
			Type: MessageTypeMessage, Text: message, AgentSessionID: session, Attachments: attachments,
		}); err != nil {
			t.Fatal(err)
		}
		result := host.awaitFrame(t, MessageTypeResult, 20*time.Second)
		if result.frame["agent_session_id"] != session || result.frame["output"] != "files read" {
			t.Fatalf("result belongs to the wrong message: %v", result.frame)
		}
		prompt, err := os.ReadFile(filepath.Join(workspace, "prompt"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(prompt), message+"\n") || strings.Contains(string(prompt), "synthetic-token") {
			t.Fatalf("harness prompt lost the message or contains a credential: %q", prompt)
		}
		for _, path := range earlierPaths {
			if strings.Contains(string(prompt), path) {
				t.Fatal("a previous message attachment reached the harness")
			}
		}
		var paths []string
		for _, line := range strings.Split(string(prompt), "\n") {
			if path, ok := strings.CutPrefix(line, "  - "); ok {
				if !strings.HasPrefix(path, filepath.Join(workspace, ".attachments")+string(os.PathSeparator)) {
					t.Fatalf("harness received a path outside the workspace: %q", path)
				}
				paths = append(paths, path)
			}
		}
		if len(paths) != len(attachments) {
			t.Fatalf("harness received %d paths for %d attachments", len(paths), len(attachments))
		}
		got, err := os.ReadFile(filepath.Join(workspace, "harness-input"))
		want := image
		if turn == 0 {
			want = append(append([]byte{}, text...), image...)
		}
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("harness did not read the message attachment bytes: %v", err)
		}
		earlierPaths = paths
	}
}

func TestRepositoryAttachmentKeepsCheckoutCleanForNextTurn(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the test harness is a shell script")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ANTHROPIC_API_KEY", "synthetic-model-key")
	bare := filepath.Join(t.TempDir(), "remote.git")
	gitForRepositoryTest(t, home, "init", "-q", "--bare", bare)
	gitForRepositoryTest(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	work := t.TempDir()
	gitForRepositoryTest(t, work, "init", "-q", "-b", "main")
	gitForRepositoryTest(t, work, "config", "user.email", "test@example.com")
	gitForRepositoryTest(t, work, "config", "user.name", "Test User")
	gitForRepositoryTest(t, work, "remote", "add", "origin", bare)
	commit := func(content string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(work, "file.txt"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		gitForRepositoryTest(t, work, "add", "file.txt")
		gitForRepositoryTest(t, work, "commit", "-qm", "Update fixture")
		gitForRepositoryTest(t, work, "push", "-q", "origin", "main")
		return gitForRepositoryTest(t, work, "rev-parse", "HEAD")
	}
	firstCommit := commit("first")
	gitForRepositoryTest(t, home, "config", "--global", "url.file://"+bare+".insteadOf", "https://github.com/acme/fiction.git")

	capture := t.TempDir()
	t.Setenv("TEST_CAPTURE_DIR", capture)
	bin := t.TempDir()
	script := "#!/bin/sh\n" + claudeVersionAnswer + `set -eu
cat > "$TEST_CAPTURE_DIR/prompt"
sed -n 's/^  - //p' "$TEST_CAPTURE_DIR/prompt" | while IFS= read -r file; do cat "$file"; done > "$TEST_CAPTURE_DIR/input"
echo '{"type":"result","session_id":"fixture-session","result":"files read"}'
`
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	input := []byte("synthetic repository attachment\n")
	host := newTestControlHost(t)
	websocketHandler := host.srv.Config.Handler
	host.srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/agents/agent-1/repository-access/":
			if r.Header.Get("Authorization") != "Bearer synthetic-token" {
				t.Error("repository access lost agent authorization")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"repository":"acme/fiction","ref":"main","token":"fixture-token"}`))
		case "/api/agents/attachments/" + testAttachmentID + "/download/":
			if r.Header.Get("Authorization") != "Bearer synthetic-token" {
				t.Error("attachment download lost agent authorization")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write(input)
		default:
			websocketHandler.ServeHTTP(w, r)
		}
	})
	workspace := t.TempDir()
	cfg := Config{Token: "synthetic-token", Host: host.host(), AgentID: "agent-1", Workspace: workspace}
	client := NewAgentClient(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)
	conn := host.nextConn(t)
	selected := filepath.Join(workspace, "repositories", testRepositoryID)
	repository := &RepositoryRequest{ID: testRepositoryID, Ref: "main"}
	if err := wsjson.Write(ctx, conn, Message{Type: MessageTypeMessage, Text: "Read the file", AgentSessionID: "first-turn", Repository: repository, Attachments: []Attachment{{ID: testAttachmentID, Filename: "input.txt"}}}); err != nil {
		t.Fatal(err)
	}
	first := host.awaitFrame(t, MessageTypeResult, 20*time.Second)
	if first.frame["agent_session_id"] != "first-turn" || first.frame["output"] != "files read" {
		t.Fatalf("first turn failed: %v", first.frame)
	}
	prompt, err := os.ReadFile(filepath.Join(capture, "prompt"))
	if err != nil {
		t.Fatal(err)
	}
	var attachmentPath string
	for _, line := range strings.Split(string(prompt), "\n") {
		if path, ok := strings.CutPrefix(line, "  - "); ok {
			attachmentPath = path
		}
	}
	if !strings.HasPrefix(attachmentPath, filepath.Join(workspace, ".attachments")+string(os.PathSeparator)) {
		t.Fatalf("model got a path outside attachment storage: %q", attachmentPath)
	}
	got, err := os.ReadFile(filepath.Join(capture, "input"))
	if err != nil || !bytes.Equal(got, input) {
		t.Fatalf("model boundary could not read attachment: %v, %q", err, got)
	}
	if status := gitForRepositoryTest(t, selected, "status", "--porcelain"); status != "" {
		t.Fatalf("attachment made the checkout dirty: %q", status)
	}
	if got := gitForRepositoryTest(t, selected, "rev-parse", "HEAD"); got != firstCommit {
		t.Fatalf("first turn selected %s, want %s", got, firstCommit)
	}

	secondCommit := commit("second")
	if err := wsjson.Write(ctx, conn, Message{Type: MessageTypeMessage, Text: "Select the ref again", AgentSessionID: "second-turn", Repository: repository}); err != nil {
		t.Fatal(err)
	}
	second := host.awaitFrame(t, MessageTypeResult, 20*time.Second)
	if second.frame["agent_session_id"] != "second-turn" || second.frame["output"] != "files read" {
		t.Fatalf("second turn failed: %v", second.frame)
	}
	if got := gitForRepositoryTest(t, selected, "rev-parse", "HEAD"); got != secondCommit {
		t.Fatalf("later turn selected %s, want %s", got, secondCommit)
	}
	prompt, err = os.ReadFile(filepath.Join(capture, "prompt"))
	if err != nil || strings.Contains(string(prompt), attachmentPath) {
		t.Fatalf("earlier attachment reached later prompt: %v, %q", err, prompt)
	}
	if err := os.WriteFile(filepath.Join(selected, "user-note.txt"), []byte("keep this"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := wsjson.Write(ctx, conn, Message{Type: MessageTypeMessage, Text: "Select a dirty ref", AgentSessionID: "third-turn", Repository: repository}); err != nil {
		t.Fatal(err)
	}
	third := host.awaitFrame(t, MessageTypeError, 20*time.Second)
	reason, _ := third.frame["error"].(string)
	if third.frame["agent_session_id"] != "third-turn" || !strings.Contains(reason, "repository has local changes") {
		t.Fatalf("user change was not refused: %v", third.frame)
	}
	if got, err := os.ReadFile(filepath.Join(selected, "user-note.txt")); err != nil || string(got) != "keep this" {
		t.Fatalf("user change was lost: %v, %q", err, got)
	}
}
