package install

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSourceSkill_SkillArgWinsOverURLBasename(t *testing.T) {
	got, err := ResolveSourceSkill("https://example.com/foo.md", "my-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "my-skill" {
		t.Errorf("skill: got %q, want my-skill", got)
	}
}

func TestResolveSourceSkill_DerivesFromBasenameWhenSkillOmitted(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://example.com/foo.md", "foo"},
		{"https://example.com/path/to/bar.md", "bar"},
		{"http://x.test/baz", "baz"}, // no .md suffix → use basename as-is
	}
	for _, tc := range cases {
		t.Run(tc.url, func(t *testing.T) {
			got, err := ResolveSourceSkill(tc.url, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveSourceSkill_RejectsUnusableBasenameAsksForExplicitSkill(t *testing.T) {
	cases := []string{
		"https://example.com/",
		"https://example.com",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			_, err := ResolveSourceSkill(raw, "")
			if err == nil {
				t.Fatalf("expected error for unusable basename in %q", raw)
			}
			msg := err.Error()
			if !strings.Contains(msg, "<skill>") {
				t.Errorf("error should ask for <skill> argument, got: %s", msg)
			}
		})
	}
}

func TestResolveSourceSkill_RejectsNonHTTPSchemes(t *testing.T) {
	cases := []string{
		"file:///etc/passwd",
		"ftp://example.com/foo.md",
		"example.com/foo.md", // no scheme
		"foo.md",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			_, err := ResolveSourceSkill(raw, "anything")
			if err == nil {
				t.Fatalf("expected error for non-http(s) URL %q", raw)
			}
		})
	}
}

func TestResolveSourceSkill_RejectsMalformedAndHostlessURLs(t *testing.T) {
	cases := []string{
		"https://",        // no host
		"http://",         // no host
		"://nohost",       // parse error
		"http://%zz",      // parse error
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			_, err := ResolveSourceSkill(raw, "x")
			if err == nil {
				t.Fatalf("expected error for malformed URL %q", raw)
			}
		})
	}
}

// TestPerformInstall_SourceFetchesFromURLAndWritesToSkillArg integrates
// ResolveSourceSkill with PerformInstall to lock in the headline behavior:
// `--source` bypasses the registry, uses the positional <skill> for the
// destination filename, and writes the URL body verbatim.
func TestPerformInstall_SourceFetchesFromURLAndWritesToSkillArg(t *testing.T) {
	cwd := t.TempDir()
	body := []byte("# fetched from arbitrary URL\n")

	const sourceURL = "https://example.com/foo.md"
	var fetchedURL string
	registryCalls := 0

	skill, err := ResolveSourceSkill(sourceURL, "my-skill")
	if err != nil {
		t.Fatalf("ResolveSourceSkill: %v", err)
	}

	out, err := PerformInstall(InstallRequest{
		Cwd:     cwd,
		HomeDir: t.TempDir(),
		Scope:   ScopeLocal,
		Agent:   "coach",
		Skill:   skill,
		Source:  sourceURL,
		FetchFn: func(agent, skill string) ([]byte, error) {
			registryCalls++
			return nil, nil
		},
		FetchURL: func(u string) ([]byte, error) {
			fetchedURL = u
			return body, nil
		},
	})
	if err != nil {
		t.Fatalf("PerformInstall: %v", err)
	}

	if registryCalls != 0 {
		t.Errorf("--source must bypass registry; got %d registry calls", registryCalls)
	}
	if fetchedURL != sourceURL {
		t.Errorf("fetched URL: got %q, want %q", fetchedURL, sourceURL)
	}

	// Filename must use <skill>, not the URL's basename ("foo").
	wantPath := filepath.Join(cwd, ".spekk", "skills", "coach", "my-skill.md")
	if !strings.Contains(out, wantPath) {
		t.Errorf("output should reference path %q, got: %q", wantPath, out)
	}
}
