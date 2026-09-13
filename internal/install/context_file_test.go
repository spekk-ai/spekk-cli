package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gemini and hermes must both be valid, listed targets so the auto-ensure step
// in the interactive launcher has something to call for them.
func TestValidTargets_IncludesHermesAndGemini(t *testing.T) {
	got := strings.Join(ValidTargets(), ",")
	for _, name := range []string{"hermes", "gemini"} {
		if !strings.Contains(got, name) {
			t.Errorf("ValidTargets should list %q, got %v", name, ValidTargets())
		}
	}
}

// A full `spekk install --target hermes` lands the coach and builder skills in
// hermes's own skills directory — the path `hermes chat -s spekk-<role>`
// discovers — with the frontmatter kept so the name resolves.
func TestInstall_HermesDestination(t *testing.T) {
	home := t.TempDir()
	if _, err := Install(Options{Target: "hermes", HomeDir: home, SkillFS: fakeSkillFS()}); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"coach", "builder"} {
		skill := filepath.Join(home, ".hermes", "skills", "spekk-"+role, "SKILL.md")
		body := string(mustRead(t, skill))
		if !strings.Contains(body, "name: spekk-"+role) {
			t.Errorf("hermes %s skill must keep its frontmatter name: %q", role, body)
		}
		if !strings.Contains(body, "`spekk prompt "+role+"`") {
			t.Errorf("hermes %s skill body must run spekk prompt %s", role, role)
		}
	}
}

// `spekk install --target gemini` writes the spekk section into gemini's
// auto-read context file — ~/.gemini/GEMINI.md globally, ./GEMINI.md in a project
// — carrying the coach and builder roles the activation message names. The write
// is idempotent.
func TestInstall_GeminiDestination(t *testing.T) {
	for _, tc := range []struct {
		name    string
		project bool
		rel     []string
	}{
		{"global", false, []string{".gemini", "GEMINI.md"}},
		{"project", true, []string{"GEMINI.md"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := t.TempDir()
			opts := Options{Target: "gemini"}
			if tc.project {
				opts.Project = true
				opts.Cwd = base
			} else {
				opts.HomeDir = base
			}

			res, err := Install(opts)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(append([]string{base}, tc.rel...)...)
			if len(res.Written) != 1 || res.Written[0] != path {
				t.Fatalf("wrote %v, want only %s", res.Written, path)
			}
			content := string(mustRead(t, path))
			if !strings.Contains(content, contextSectionBegin) || !strings.Contains(content, contextSectionEnd) {
				t.Errorf("GEMINI.md must carry the spekk section markers: %q", content)
			}
			for _, role := range []string{"coach", "builder"} {
				if !strings.Contains(content, "### spekk-"+role) {
					t.Errorf("GEMINI.md must define spekk-%s so the activation message resolves it", role)
				}
				if !strings.Contains(content, "`spekk prompt "+role+"`") {
					t.Errorf("spekk-%s section must run spekk prompt %s", role, role)
				}
			}

			// Idempotent: a re-install rewrites nothing.
			res2, err := Install(opts)
			if err != nil {
				t.Fatalf("re-install should succeed: %v", err)
			}
			if len(res2.Written) != 0 {
				t.Errorf("re-install not idempotent: rewrote %v", res2.Written)
			}
		})
	}
}

// The shared context file is the user's too: install must refresh only the
// spekk-owned section and leave every other line — before and after it — intact.
func TestInstall_GeminiPreservesUserContent(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".gemini", "GEMINI.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	before := "# My own gemini instructions\n\nAlways use tabs.\n"
	after := "\n## Trailing notes\n\nKeep the build green.\n"
	// Seed a stale spekk section between the user's own content.
	seed := before + contextSectionBegin + "\nSTALE spekk content\n" + contextSectionEnd + after
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Install(Options{Target: "gemini", HomeDir: home}); err != nil {
		t.Fatal(err)
	}
	content := string(mustRead(t, path))

	if !strings.HasPrefix(content, before) {
		t.Errorf("user content before the section was clobbered: %q", content)
	}
	if !strings.HasSuffix(content, after) {
		t.Errorf("user content after the section was clobbered: %q", content)
	}
	if strings.Contains(content, "STALE spekk content") {
		t.Errorf("the stale spekk section should have been replaced: %q", content)
	}
	if !strings.Contains(content, "`spekk prompt coach`") {
		t.Errorf("the refreshed section should carry the coach shim: %q", content)
	}
	// Exactly one spekk section — the replace happened in place, not by appending.
	if n := strings.Count(content, contextSectionBegin); n != 1 {
		t.Errorf("want exactly one spekk section, got %d", n)
	}
}

// EnsureRoleSkill on a context-file host writes the same complete section Install
// does (coach and builder both present), so switching between a coach and a
// builder interactive session never drops the other role from GEMINI.md.
func TestEnsureRoleSkill_GeminiWritesFullSection(t *testing.T) {
	home := t.TempDir()
	res, err := EnsureRoleSkill(Options{Target: "gemini", HomeDir: home}, "coach")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".gemini", "GEMINI.md")
	if len(res.Written) != 1 || res.Written[0] != path {
		t.Fatalf("wrote %v, want only %s", res.Written, path)
	}
	content := string(mustRead(t, path))
	for _, role := range []string{"coach", "builder"} {
		if !strings.Contains(content, "### spekk-"+role) {
			t.Errorf("EnsureRoleSkill(coach) must still write the %s section: %q", role, content)
		}
	}
}
