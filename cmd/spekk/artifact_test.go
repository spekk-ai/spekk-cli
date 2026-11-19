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
		if posts == 1 {
			w.WriteHeader(201)
			fmt.Fprint(w, `{"app_url":"https://app.example.test/artifacts/one"}`)
		} else {
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
	if err := execArtifact([]string{"upload", file}, &out, getenv); err != nil || strings.TrimSpace(out.String()) != "https://app.example.test/artifacts/one" {
		t.Fatalf("upload: output %q, error %v", out.String(), err)
	}
	out.Reset()
	if err := execArtifact([]string{"upload", file}, &out, getenv); err == nil || !strings.Contains(err.Error(), "artifacts are off") {
		t.Fatalf("off response: %v", err)
	}
	if strings.Contains(out.String(), "test-agent-token") || posts != 2 {
		t.Fatalf("token leaked or wrong post count: %d, %q", posts, out.String())
	}
}
