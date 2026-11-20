package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactUploadAndDisabled(t *testing.T) {
	file := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(file, []byte("# Made-up report\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	posts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts++
		if r.Header.Get("Authorization") != "Bearer test-agent-token" {
			t.Error("agent token was not sent as a bearer header")
		}
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Error(err)
		}
		if r.FormValue("kind") != "markdown" || r.FormValue("title") != "report.md" || r.FormValue("access") != "" {
			t.Errorf("unexpected artifact fields: %v", r.Form)
		}
		switch posts {
		case 1:
			w.WriteHeader(201)
			fmt.Fprint(w, `{"app_url":"https://app.example.test/artifacts/one","view_url":"https://content.example.test/a/one/"}`)
		case 2:
			// An older app has no view_url.
			w.WriteHeader(201)
			fmt.Fprint(w, `{"app_url":"https://app.example.test/artifacts/one"}`)
		default:
			w.WriteHeader(404)
			fmt.Fprint(w, `{"detail":"Not found."}`)
		}
	}))
	defer server.Close()
	getenv := func(name string) string {
		if name == "SPEKK_HOST" {
			return server.URL
		}
		if name == "SPEKK_AGENT_TOKEN" {
			return "test-agent-token"
		}
		return ""
	}
	var out bytes.Buffer
	if err := execArtifact([]string{"upload", file}, &out, getenv); err != nil || out.String() != "https://content.example.test/a/one/\nApp page: https://app.example.test/artifacts/one\n" {
		t.Fatalf("upload: output %q, error %v", out.String(), err)
	}
	out.Reset()
	if err := execArtifact([]string{"upload", file}, &out, getenv); err != nil || out.String() != "https://app.example.test/artifacts/one\n" {
		t.Fatalf("older app: output %q, error %v", out.String(), err)
	}
	out.Reset()
	if err := execArtifact([]string{"upload", file}, &out, getenv); err == nil || !strings.Contains(err.Error(), "artifacts are off") {
		t.Fatalf("off response: %v", err)
	}
	if strings.Contains(out.String(), "test-agent-token") || posts != 3 {
		t.Fatalf("token leaked or wrong post count: %d, %q", posts, out.String())
	}
}

func TestTextArtifactInferenceAndOlderApp(t *testing.T) {
	dir := t.TempDir()
	var kinds []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Error(err)
		}
		kinds = append(kinds, r.FormValue("kind"))
		if r.FormValue("title") == "Older app" {
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"Choose the kind markdown, image, video, or verification."}`)
			return
		}
		w.WriteHeader(201)
		fmt.Fprint(w, `{"view_url":"https://content.example.test/a/one/","app_url":"https://app.example.test/artifacts/one"}`)
	}))
	defer server.Close()
	getenv := func(name string) string {
		switch name {
		case "SPEKK_HOST":
			return server.URL
		case "SPEKK_AGENT_TOKEN":
			return "test-agent-token"
		}
		return ""
	}
	textNames := []string{"file.txt", "file.log", "file.csv", "file.tsv", "file.json", "file.jsonl", "file.yaml", "file.yml", "file.toml", "file.ini", "file.sql", "file.py", "file.go", "file.js", "file.ts", "file.tsx", "file.jsx", "file.sh", "file.rb", "file.java", "file.kt", "file.rs", "file.c", "file.h", "file.cpp", "file.css", "file.graphql", "file.proto", "file.env.example", "file.diff", "file.patch", "file.custom"}
	for _, name := range append(textNames, "file.md", "file.png", "file.mp4") {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("key: value\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := execArtifact([]string{"upload", path}, &out, getenv); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out.String(), "https://content.example.test/a/one/\n") {
			t.Fatalf("direct link missing: %q", out.String())
		}
		expected := "text"
		if name == "file.md" {
			expected = "markdown"
		} else if name == "file.png" {
			expected = "image"
		} else if name == "file.mp4" {
			expected = "video"
		}
		if got := kinds[len(kinds)-1]; got != expected {
			t.Fatalf("%s inferred %s, want %s", name, got, expected)
		}
	}
	path := filepath.Join(dir, "query.sql")
	if err := os.WriteFile(path, []byte("SELECT 1;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := execArtifact([]string{"upload", path, "--kind", "text", "--title", "Older app"}, &out, getenv)
	if err == nil || !strings.Contains(err.Error(), "does not support text artifacts yet") || strings.Contains(err.Error(), "invalid response") {
		t.Fatalf("older app error: %v", err)
	}
	if kinds[len(kinds)-1] != "text" {
		t.Fatalf("explicit kind was %q", kinds[len(kinds)-1])
	}
	for _, content := range [][]byte{{0xff}, {'a', 0, 'b'}} {
		bad := filepath.Join(dir, "unknown.custom")
		if err := os.WriteFile(bad, content, 0o600); err != nil {
			t.Fatal(err)
		}
		err := execArtifact([]string{"upload", bad}, &out, getenv)
		if err == nil || !strings.Contains(err.Error(), "cannot infer the artifact kind") {
			t.Fatalf("binary inference: %v", err)
		}
	}
	active := filepath.Join(dir, "page.html")
	if err := os.WriteFile(active, []byte("<script>alert(1)</script>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := execArtifact([]string{"upload", active}, &out, getenv); err == nil || !strings.Contains(err.Error(), "active content") {
		t.Fatalf("active content inference: %v", err)
	}
}
