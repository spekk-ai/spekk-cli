package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

// fakeEnv returns a getenv func backed by an in-memory map, so tests never
// need to mutate real process environment variables.
func fakeEnv(values map[string]string) func(string) string {
	return func(name string) string {
		return values[name]
	}
}

func TestConversationDirectDeliveryAndPartialFailure(t *testing.T) {
	resumeFile := filepath.Join(t.TempDir(), "claude-session-id")
	if err := os.WriteFile(resumeFile, []byte("claude-example"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(t.TempDir(), "schema.sql")
	second := filepath.Join(t.TempDir(), "api.yaml")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte("made-up file"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		if r.Header.Get("Authorization") != "Bearer test-agent-token" {
			t.Error("missing agent authorization")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
		}
		if r.FormValue("session_id") != "test-session" || r.FormValue("claude_session_id") != "claude-example" || r.FormValue("title") != "Report" {
			t.Errorf("wrong fields: %v", r.Form)
		}
		want := 0
		if posts != 1 {
			want = 2
		}
		if got := len(r.MultipartForm.File["file"]); got != want {
			t.Errorf("files = %d, want %d", got, want)
		}
		if posts == 1 {
			fmt.Fprint(w, `{"message_url":"https://app.slack.com/archives/C_HOME/p123456","file_ids":[]}`)
		} else if posts == 3 {
			w.WriteHeader(502)
			fmt.Fprint(w, `{"detail":"The Slack bot needs files:write.","message_url":"https://app.slack.com/archives/C_HOME/p123456","text_posted":true}`)
		} else {
			fmt.Fprint(w, `{"message_url":"https://app.slack.com/archives/C_HOME/p123456","file_ids":["F1","F2"]}`)
		}
	}))
	defer server.Close()
	env := fakeEnv(map[string]string{"SPEKK_HOST": server.URL, "SPEKK_AGENT_TOKEN": "test-agent-token", "SPEKK_AGENT_SESSION_ID": "test-session", "SPEKK_CONVERSATION_DELIVERY": "1", conversation.SessionFileEnvVar: resumeFile})
	var out, errOut bytes.Buffer
	if code := execConversationOpen([]string{"--title", "Report", "--body", "Ready"}, &out, &errOut, env); code != 0 || !strings.Contains(out.String(), "Message: https://") || strings.Contains(out.String(), "File:") {
		t.Fatalf("plain delivery: code %d, output %q, error %q", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := execConversationOpen([]string{"--title", "Report", "--body", "Ready", "--attach", first, "--attach", second}, &out, &errOut, env); code != 0 || !strings.Contains(out.String(), "File: F2") {
		t.Fatalf("file delivery: code %d, output %q, error %q", code, out.String(), errOut.String())
	}
	errOut.Reset()
	if code := execConversationOpen([]string{"--title", "Report", "--body", "Ready", "--attach", first, "--attach", second}, &out, &errOut, env); code == 0 || !strings.Contains(errOut.String(), "text posted at") || !strings.Contains(errOut.String(), "files:write") {
		t.Fatalf("partial delivery: code %d, error %q", code, errOut.String())
	}
	if posts != 3 || strings.Contains(out.String()+errOut.String(), "test-agent-token") {
		t.Fatalf("unexpected requests or token leak: %d", posts)
	}
}

func TestConversationReplyCapabilityAndRequest(t *testing.T) {
	resumeFile := filepath.Join(t.TempDir(), "claude-session-id")
	if err := os.WriteFile(resumeFile, []byte("claude-example"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(file, []byte("report"), 0o600); err != nil {
		t.Fatal(err)
	}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		if r.URL.Path != "/api/agents/conversations/" || r.Header.Get("Authorization") != "Bearer test-agent-token" {
			t.Errorf("wrong request path or authorization")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("session_id") != "test-session" || r.FormValue("turn_id") != "test-turn" || r.FormValue("claude_session_id") != "claude-example" || r.FormValue("delivery") != "current_thread" || r.FormValue("body") != "Ready" {
			t.Errorf("wrong reply fields: %v", r.Form)
		}
		if r.FormValue("channel") != "" || r.FormValue("thread_ts") != "" || r.FormValue("title") != "" {
			t.Errorf("reply supplied a destination or title: %v", r.Form)
		}
		if got := len(r.MultipartForm.File["file"]); got != 1 {
			t.Errorf("files = %d, want 1", got)
		}
		if posts == 2 {
			w.WriteHeader(502)
			fmt.Fprint(w, `{"detail":"The Slack bot needs files:write.","message_url":"https://app.slack.com/archives/C_THREAD/p123999","text_posted":true}`)
			return
		}
		fmt.Fprint(w, `{"message_url":"https://app.slack.com/archives/C_THREAD/p123999","file_ids":["F_REPORT"]}`)
	}))
	defer server.Close()
	values := map[string]string{"SPEKK_HOST": server.URL, "SPEKK_AGENT_TOKEN": "test-agent-token", "SPEKK_AGENT_SESSION_ID": "test-session", "SPEKK_AGENT_TURN_ID": "test-turn", conversation.SessionFileEnvVar: resumeFile}
	env := fakeEnv(values)
	var out, errOut bytes.Buffer
	if code := execConversationReply([]string{"--body", "Ready", "--attach", file}, &out, &errOut, env); code == 0 || !strings.Contains(errOut.String(), "does not support current Slack thread delivery") || posts != 0 {
		t.Fatalf("capability gate: code %d, error %q, posts %d", code, errOut.String(), posts)
	}
	values[currentThreadDeliveryEnvVar] = "1"
	delete(values, "SPEKK_AGENT_TURN_ID")
	errOut.Reset()
	if code := execConversationReply([]string{"--body", "Ready", "--attach", file}, &out, &errOut, env); code == 0 || !strings.Contains(errOut.String(), "turn ID is not ready") || posts != 0 {
		t.Fatalf("missing turn: code %d, error %q, posts %d", code, errOut.String(), posts)
	}
	values["SPEKK_AGENT_TURN_ID"] = "test-turn"
	errOut.Reset()
	if code := execConversationReply([]string{"--body", "Ready", "--attach", file}, &out, &errOut, env); code != 0 || !strings.Contains(out.String(), "File: F_REPORT") {
		t.Fatalf("reply: code %d, output %q, error %q", code, out.String(), errOut.String())
	}
	errOut.Reset()
	if code := execConversationReply([]string{"--body", "Ready", "--attach", file}, &out, &errOut, env); code == 0 || !strings.Contains(errOut.String(), "text posted at") || posts != 2 {
		t.Fatalf("partial reply: code %d, error %q, posts %d", code, errOut.String(), posts)
	}
}

func TestConversationCurrentSessionWithoutDeliveryCapability(t *testing.T) {
	spoolDir := t.TempDir()
	resumeFile := filepath.Join(t.TempDir(), "claude-session-id")
	if err := os.WriteFile(resumeFile, []byte("claude-example"), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(file, []byte("report"), 0o600); err != nil {
		t.Fatal(err)
	}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		http.NotFound(w, r)
	}))
	defer server.Close()
	env := fakeEnv(map[string]string{
		"SPEKK_HOST": server.URL, "SPEKK_AGENT_TOKEN": "test-agent-token", "SPEKK_AGENT_SESSION_ID": "test-session", conversationDeliveryEnvVar: "0",
		conversation.SpoolEnvVar: spoolDir, conversation.SessionFileEnvVar: resumeFile,
	})
	var out, errOut bytes.Buffer
	if code := execConversationOpen([]string{"--title", "Report", "--body", "Ready"}, &out, &errOut, env); code != 0 {
		t.Fatalf("plain message: code %d, error %q", code, errOut.String())
	}
	files := requestFiles(t, spoolDir)
	if len(files) != 1 {
		t.Fatalf("queued files = %v", files)
	}
	data, err := os.ReadFile(filepath.Join(spoolDir, files[0]))
	if err != nil {
		t.Fatal(err)
	}
	var req conversation.Request
	if err := json.Unmarshal(data, &req); err != nil || req.Title != "Report" || req.Body != "Ready" {
		t.Fatalf("queued request = %+v, error %v", req, err)
	}
	errOut.Reset()
	if code := execConversationOpen([]string{"--title", "Report", "--body", "Ready", "--attach", file}, &out, &errOut, env); code == 0 || !strings.Contains(errOut.String(), "app does not support conversation file delivery") {
		t.Fatalf("attachment: code %d, error %q", code, errOut.String())
	}
	if posts != 0 || len(requestFiles(t, spoolDir)) != 1 {
		t.Fatalf("HTTP posts = %d, queued files = %v", posts, requestFiles(t, spoolDir))
	}
}

func TestConversationRejectsLargeAttachmentBeforePost(t *testing.T) {
	resumeFile := filepath.Join(t.TempDir(), "claude-session-id")
	if err := os.WriteFile(resumeFile, []byte("claude-example"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "large.bin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(10<<20 + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	var out, errOut bytes.Buffer
	code := execConversationOpen([]string{"--title", "Report", "--body", "Ready", "--attach", path}, &out, &errOut, fakeEnv(map[string]string{"SPEKK_AGENT_SESSION_ID": "session", conversationDeliveryEnvVar: "1", conversation.SessionFileEnvVar: resumeFile}))
	if code == 0 || !strings.Contains(errOut.String(), "10 MB") {
		t.Fatalf("code %d, error %q", code, errOut.String())
	}
}

// requestFiles returns the names of all non-directory entries in dir.
func requestFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// TestExecConversationOpen_HappyPath verifies a valid invocation writes
// exactly one well-formed request file, readable by the shared contract
// struct, containing no session_id key, and leaves no stray temp files
// behind (atomicity: temp file is renamed away, not left in place).
func TestExecConversationOpen_HappyPath(t *testing.T) {
	spoolDir := t.TempDir()
	env := fakeEnv(map[string]string{conversation.SpoolEnvVar: spoolDir})

	var stdout, stderr bytes.Buffer
	code := execConversationOpen(
		[]string{"--title", "Need input", "--body", "Should we proceed?", "--severity", "warning"},
		&stdout, &stderr, env,
	)

	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %q", code, stderr.String())
	}

	files := requestFiles(t, spoolDir)
	if len(files) != 1 {
		t.Fatalf("expected exactly one request file, got %d: %v", len(files), files)
	}
	if strings.HasSuffix(files[0], ".tmp") {
		t.Errorf("expected final file, not a stray temp file: %q", files[0])
	}

	data, err := os.ReadFile(filepath.Join(spoolDir, files[0]))
	if err != nil {
		t.Fatalf("read request file: %v", err)
	}

	var req conversation.Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("request file does not decode as conversation.Request: %v", err)
	}
	if req.Title != "Need input" || req.Body != "Should we proceed?" || req.Severity != conversation.SeverityWarning {
		t.Errorf("unexpected request contents: %+v", req)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("request file is not valid JSON: %v", err)
	}
	if _, ok := raw["session_id"]; ok {
		t.Errorf("request file must not contain session_id, got: %s", data)
	}
	if len(raw) != 3 {
		t.Errorf("expected exactly 3 keys (title, body, severity), got: %s", data)
	}
}

// TestExecConversationOpen_DefaultSeverity confirms an omitted --severity
// defaults to "info" rather than an empty/invalid value.
func TestExecConversationOpen_DefaultSeverity(t *testing.T) {
	spoolDir := t.TempDir()
	env := fakeEnv(map[string]string{conversation.SpoolEnvVar: spoolDir})

	var stdout, stderr bytes.Buffer
	code := execConversationOpen([]string{"--title", "T", "--body", "B"}, &stdout, &stderr, env)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %q", code, stderr.String())
	}

	files := requestFiles(t, spoolDir)
	if len(files) != 1 {
		t.Fatalf("expected exactly one request file, got %d", len(files))
	}
	data, _ := os.ReadFile(filepath.Join(spoolDir, files[0]))
	var req conversation.Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if req.Severity != conversation.DefaultSeverity {
		t.Errorf("expected default severity %q, got %q", conversation.DefaultSeverity, req.Severity)
	}
}

func TestExecConversationOpen_ChannelRequest(t *testing.T) {
	spoolDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := execConversationOpen(
		[]string{"--title", "T", "--body", "B", "--channel", "#project-room"},
		&stdout, &stderr, fakeEnv(map[string]string{conversation.SpoolEnvVar: spoolDir}),
	)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	files := requestFiles(t, spoolDir)
	if len(files) != 1 {
		t.Fatalf("want one request, got %v", files)
	}
	data, err := os.ReadFile(filepath.Join(spoolDir, files[0]))
	if err != nil {
		t.Fatal(err)
	}
	var req conversation.Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatal(err)
	}
	if req.Channel != "#project-room" {
		t.Fatalf("channel = %q", req.Channel)
	}
}

func TestExecConversationOpen_EmptyChannelRejected(t *testing.T) {
	spoolDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := execConversationOpen(
		[]string{"--title", "T", "--body", "B", "--channel", " "},
		&stdout, &stderr, fakeEnv(map[string]string{conversation.SpoolEnvVar: spoolDir}),
	)
	if code == 0 || !strings.Contains(stderr.String(), "--channel") {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	if len(requestFiles(t, spoolDir)) != 0 {
		t.Fatal("invalid channel created a request")
	}
}

// TestExecConversationOpen_TwoInvocationsDoNotCollide verifies two quick
// back-to-back calls each produce their own file instead of one clobbering
// the other's filename.
func TestExecConversationOpen_TwoInvocationsDoNotCollide(t *testing.T) {
	spoolDir := t.TempDir()
	env := fakeEnv(map[string]string{conversation.SpoolEnvVar: spoolDir})
	var stdout, stderr bytes.Buffer

	for i := 0; i < 2; i++ {
		code := execConversationOpen([]string{"--title", "T", "--body", "B"}, &stdout, &stderr, env)
		if code != 0 {
			t.Fatalf("invocation %d: expected exit 0, got %d; stderr: %q", i, code, stderr.String())
		}
	}

	files := requestFiles(t, spoolDir)
	if len(files) != 2 {
		t.Fatalf("expected 2 distinct request files, got %d: %v", len(files), files)
	}
}

// TestExecConversationOpen_UnsetSpoolVar verifies an unset (or empty) spool
// env var fails clearly rather than writing anywhere.
func TestExecConversationOpen_UnsetSpoolVar(t *testing.T) {
	env := fakeEnv(map[string]string{})

	var stdout, stderr bytes.Buffer
	code := execConversationOpen([]string{"--title", "T", "--body", "B"}, &stdout, &stderr, env)

	if code == 0 {
		t.Fatal("expected non-zero exit when spool env var is unset")
	}
	if !strings.Contains(stderr.String(), conversation.SpoolEnvVar) {
		t.Errorf("expected error to name %s, got: %q", conversation.SpoolEnvVar, stderr.String())
	}
}

// TestExecConversationOpen_MissingFlags verifies missing --title and/or
// --body each fail with a message naming the missing flag.
func TestExecConversationOpen_MissingFlags(t *testing.T) {
	spoolDir := t.TempDir()
	env := fakeEnv(map[string]string{conversation.SpoolEnvVar: spoolDir})

	cases := map[string][]string{
		"missing title": {"--body", "B"},
		"missing body":  {"--title", "T"},
		"missing both":  {},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := execConversationOpen(args, &stdout, &stderr, env)
			if code == 0 {
				t.Fatalf("expected non-zero exit for %s", name)
			}
			if !strings.Contains(stderr.String(), "--title") && !strings.Contains(stderr.String(), "--body") {
				t.Errorf("expected error naming the missing flag, got: %q", stderr.String())
			}
		})
	}
}

// TestExecConversationOpen_InvalidSeverity verifies an out-of-range
// --severity is rejected with a message listing the valid values, rather
// than silently corrected.
func TestExecConversationOpen_InvalidSeverity(t *testing.T) {
	spoolDir := t.TempDir()
	env := fakeEnv(map[string]string{conversation.SpoolEnvVar: spoolDir})

	var stdout, stderr bytes.Buffer
	code := execConversationOpen([]string{"--title", "T", "--body", "B", "--severity", "urgent"}, &stdout, &stderr, env)

	if code == 0 {
		t.Fatal("expected non-zero exit for invalid severity")
	}
	for _, want := range []string{"info", "warning", "critical"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("expected error listing valid severities (%q missing), got: %q", want, stderr.String())
		}
	}
	if len(requestFiles(t, spoolDir)) != 0 {
		t.Error("expected no request file written for an invalid severity")
	}
}
