package install

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaults_AreOfficialRegistry(t *testing.T) {
	t.Setenv("SPEKK_SKILLS_RAW_BASE", "")
	t.Setenv("SPEKK_SKILLS_API_BASE", "")

	if got := RawBase(); got != "https://raw.githubusercontent.com/spekk-ai/spekk-skills/main" {
		t.Errorf("RawBase default = %q", got)
	}
	if got := APIBase(); got != "https://api.github.com/repos/spekk-ai/spekk-skills/contents" {
		t.Errorf("APIBase default = %q", got)
	}
}

func TestEnvOverride_TrimsTrailingSlash(t *testing.T) {
	t.Setenv("SPEKK_SKILLS_RAW_BASE", "https://mirror.example.com/skills/")
	t.Setenv("SPEKK_SKILLS_API_BASE", "https://api.example.com/contents///")

	if got := RawBase(); got != "https://mirror.example.com/skills" {
		t.Errorf("RawBase override = %q", got)
	}
	if got := APIBase(); got != "https://api.example.com/contents" {
		t.Errorf("APIBase override = %q", got)
	}
}

func TestSkillURL_BuildsAgentSkillPath(t *testing.T) {
	t.Setenv("SPEKK_SKILLS_RAW_BASE", "https://mirror.example.com/skills")
	want := "https://mirror.example.com/skills/coach/foo.md"
	if got := SkillURL("coach", "foo"); got != want {
		t.Errorf("SkillURL = %q, want %q", got, want)
	}
}

func TestFetch_WritesBodyVerbatim(t *testing.T) {
	body := "---\nid: foo\n---\n# Hello\n\nBody content with\ttabs and \r\n CRLFs.\n"
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		if r.URL.Path != "/coach/foo.md" {
			http.Error(w, "wrong path", 500)
			return
		}
		w.Write([]byte(body))
	}))
	defer srv.Close()
	t.Setenv("SPEKK_SKILLS_RAW_BASE", srv.URL)

	got, err := FetchSkill("coach", "foo")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(got) != body {
		t.Errorf("body mismatch: got %q want %q", got, body)
	}
	if gotUA != "spekk-cli" {
		t.Errorf("User-Agent = %q, want spekk-cli", gotUA)
	}
}

func TestFetch_404IncludesURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", 404)
	}))
	defer srv.Close()
	t.Setenv("SPEKK_SKILLS_RAW_BASE", srv.URL)

	_, err := FetchSkill("coach", "missing")
	if err == nil {
		t.Fatal("expected error on 404")
	}
	msg := err.Error()
	if !strings.Contains(strings.ToLower(msg), "not found") {
		t.Errorf("error should say 'not found', got: %s", msg)
	}
	wantURL := srv.URL + "/coach/missing.md"
	if !strings.Contains(msg, wantURL) {
		t.Errorf("error should name URL %q, got: %s", wantURL, msg)
	}
}

func TestFetch_OtherNon2xxIncludesCodeAndURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", 503)
	}))
	defer srv.Close()
	t.Setenv("SPEKK_SKILLS_RAW_BASE", srv.URL)

	_, err := FetchSkill("builder", "foo")
	if err == nil {
		t.Fatal("expected error on 503")
	}
	msg := err.Error()
	if !strings.Contains(msg, "503") {
		t.Errorf("error should include status code 503, got: %s", msg)
	}
	if !strings.Contains(msg, srv.URL+"/builder/foo.md") {
		t.Errorf("error should include URL, got: %s", msg)
	}
}

func TestFetchURL_TimeoutIs30Seconds(t *testing.T) {
	if defaultHTTPClient.Timeout.Seconds() != 30 {
		t.Errorf("HTTP client timeout = %v, want 30s", defaultHTTPClient.Timeout)
	}
}
