package install

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeContentsHandler serves a canned GitHub contents API payload.
func fakeContentsHandler(t *testing.T, agent, body string, status int) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/" + agent
		if r.URL.Path != wantPath {
			http.Error(w, "wrong path: "+r.URL.Path, 500)
			return
		}
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
}

func TestListURL_HitsAPIBaseForAgent(t *testing.T) {
	t.Setenv("SPEKK_SKILLS_API_BASE", "https://api.example.com/contents")
	want := "https://api.example.com/contents/coach"
	if got := ListURL("coach"); got != want {
		t.Errorf("ListURL = %q, want %q", got, want)
	}
}

func TestListRemoteSkills_FiltersMarkdownFilesAndAnnotatesInstalled(t *testing.T) {
	body := `[
		{"name": "meeting-notes.md", "type": "file"},
		{"name": "business-model-validator.md", "type": "file"},
		{"name": "README.txt", "type": "file"},
		{"name": "drafts", "type": "dir"},
		{"name": "nested.md", "type": "symlink"}
	]`
	srv := httptest.NewServer(fakeContentsHandler(t, "coach", body, 200))
	defer srv.Close()
	t.Setenv("SPEKK_SKILLS_API_BASE", srv.URL)

	cwd := t.TempDir()
	home := t.TempDir()
	// meeting-notes is installed locally; business-model-validator is global.
	mustWrite(t, filepath.Join(cwd, ".spekk", "skills", "coach", "meeting-notes.md"), "local body")
	mustWrite(t, filepath.Join(home, ".spekk", "skills", "coach", "business-model-validator.md"), "global body")

	skills, err := ListRemoteSkills("coach", cwd, home, FetchListRaw)
	if err != nil {
		t.Fatalf("ListRemoteSkills: %v", err)
	}

	if len(skills) != 2 {
		t.Fatalf("expected 2 .md file entries, got %d: %+v", len(skills), skills)
	}

	got := map[string]InstallStatus{}
	for _, s := range skills {
		got[s.Name] = s.Status
	}

	if got["meeting-notes"] != StatusLocal {
		t.Errorf("meeting-notes status = %v, want StatusLocal", got["meeting-notes"])
	}
	if got["business-model-validator"] != StatusGlobal {
		t.Errorf("business-model-validator status = %v, want StatusGlobal", got["business-model-validator"])
	}
	// .txt, dir, and symlink entries must be filtered out — also ensures we
	// never list a non-stem name like "README".
	for _, s := range skills {
		if strings.Contains(s.Name, ".") {
			t.Errorf("skill name should not include suffix, got %q", s.Name)
		}
		if s.Name == "README" || s.Name == "drafts" || s.Name == "nested" {
			t.Errorf("non-markdown-file entry leaked: %q", s.Name)
		}
	}
}

func TestListRemoteSkills_BothScopesAnnotation(t *testing.T) {
	body := `[{"name": "dual.md", "type": "file"}]`
	srv := httptest.NewServer(fakeContentsHandler(t, "builder", body, 200))
	defer srv.Close()
	t.Setenv("SPEKK_SKILLS_API_BASE", srv.URL)

	cwd := t.TempDir()
	home := t.TempDir()
	mustWrite(t, filepath.Join(cwd, ".spekk", "skills", "builder", "dual.md"), "local")
	mustWrite(t, filepath.Join(home, ".spekk", "skills", "builder", "dual.md"), "global")

	skills, err := ListRemoteSkills("builder", cwd, home, FetchListRaw)
	if err != nil {
		t.Fatalf("ListRemoteSkills: %v", err)
	}
	if len(skills) != 1 || skills[0].Status != StatusBoth {
		t.Fatalf("expected single StatusBoth entry, got %+v", skills)
	}
	if !strings.Contains(skills[0].Status.Annotation(), "local") ||
		!strings.Contains(skills[0].Status.Annotation(), "global") {
		t.Errorf("StatusBoth annotation should mention both scopes, got %q", skills[0].Status.Annotation())
	}
}

func TestListRemoteSkills_403MentionsRateLimit(t *testing.T) {
	srv := httptest.NewServer(fakeContentsHandler(t, "coach", `{"message":"API rate limit exceeded"}`, 403))
	defer srv.Close()
	t.Setenv("SPEKK_SKILLS_API_BASE", srv.URL)

	_, err := ListRemoteSkills("coach", t.TempDir(), t.TempDir(), FetchListRaw)
	if err == nil {
		t.Fatal("expected error on 403")
	}
	msg := err.Error()
	if !strings.Contains(msg, "60") || !strings.Contains(strings.ToLower(msg), "hour") {
		t.Errorf("error should mention the GitHub 60 req/hr limit, got: %s", msg)
	}
}

func TestListRemoteSkills_NetworkErrorIsWrapped(t *testing.T) {
	// Inject a fetcher that always fails — simulates DNS / TCP errors.
	fetch := func(url string) ([]byte, int, error) {
		return nil, 0, errors.New("dial tcp: lookup boom: no such host")
	}
	_, err := ListRemoteSkills("coach", t.TempDir(), t.TempDir(), fetch)
	if err == nil {
		t.Fatal("expected error when fetcher returns network failure")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("network error should propagate cause, got: %s", err)
	}
}

func TestListRemoteSkills_ParseErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(fakeContentsHandler(t, "coach", "not json", 200))
	defer srv.Close()
	t.Setenv("SPEKK_SKILLS_API_BASE", srv.URL)

	_, err := ListRemoteSkills("coach", t.TempDir(), t.TempDir(), FetchListRaw)
	if err == nil {
		t.Fatal("expected parse error on garbage body")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "parse") {
		t.Errorf("error should mention parse failure, got: %s", err)
	}
}

func TestListRemoteSkills_RejectsUnknownAgent(t *testing.T) {
	_, err := ListRemoteSkills("bogus", t.TempDir(), t.TempDir(), FetchListRaw)
	if err == nil {
		t.Fatal("expected unknown agent error")
	}
	for _, valid := range ValidAgents {
		if !strings.Contains(err.Error(), valid) {
			t.Errorf("error should list valid agent %q, got: %s", valid, err)
		}
	}
}

func TestFormatList_RendersNameAndAnnotation(t *testing.T) {
	out := FormatList("coach", []RemoteSkill{
		{Name: "meeting-notes", Status: StatusLocal},
		{Name: "fresh-one", Status: StatusNotInstalled},
	})
	for _, want := range []string{"meeting-notes", "installed (local)", "fresh-one", "not installed"} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatList output missing %q. got:\n%s", want, out)
		}
	}
	// No `.md` suffixes in the rendered output.
	if strings.Contains(out, ".md") {
		t.Errorf("FormatList should not include .md suffixes, got:\n%s", out)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
