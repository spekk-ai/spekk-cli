package main

import (
	"bytes"
	"encoding/json"
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

// TestExecConversationOpen_Help confirms --help short-circuits before any
// env or flag validation.
func TestExecConversationOpen_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := execConversationOpen([]string{"--help"}, &stdout, &stderr, fakeEnv(nil))
	if code != 0 {
		t.Fatalf("expected exit 0 for --help, got %d", code)
	}
	if !strings.Contains(stdout.String(), "spekk conversation open") {
		t.Errorf("expected usage text, got: %q", stdout.String())
	}
}
