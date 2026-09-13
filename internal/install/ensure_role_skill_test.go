package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// EnsureRoleSkill installs exactly the one coach/builder skill an interactive
// session is governed by — and nothing else. It must not write the observer
// agent shim, the dev-loop skill, or the other role, because the interactive
// launch path adds only what it needs.
func TestEnsureRoleSkill_WritesOnlyTheRoleSkill(t *testing.T) {
	home := t.TempDir()

	res, err := EnsureRoleSkill(Options{Target: "opencode", HomeDir: home}, "coach")
	if err != nil {
		t.Fatal(err)
	}

	skill := filepath.Join(home, ".config", "opencode", "skills", "spekk-coach", "SKILL.md")
	if len(res.Written) != 1 || res.Written[0] != skill {
		t.Fatalf("EnsureRoleSkill wrote %v, want only the coach skill %s", res.Written, skill)
	}
	body := string(mustRead(t, skill))
	if !strings.Contains(body, "`spekk prompt coach`") {
		t.Errorf("coach skill body must run spekk prompt coach: %q", body)
	}

	// No observer agent shim, no dev-loop skill, no builder skill were written.
	for _, absent := range []string{
		filepath.Join(home, ".config", "opencode", "agents", "spekk-observer.md"),
		filepath.Join(home, ".config", "opencode", "skills", "spekk-dev-loop", "SKILL.md"),
		filepath.Join(home, ".config", "opencode", "skills", "spekk-builder", "SKILL.md"),
	} {
		if _, err := os.Stat(absent); !os.IsNotExist(err) {
			t.Errorf("EnsureRoleSkill should not have written %s", absent)
		}
	}
}

// The hermes interactive launch preloads the skill by name (`chat -s
// spekk-<role>`), so EnsureRoleSkill must place it at the exact path hermes
// discovers a local skill under — ~/.hermes/skills/spekk-<role>/SKILL.md — with
// its frontmatter kept (hermes is a native-skill host that names skills by their
// `name:` field). If this path drifts from the -s argument the session opens
// governed by nothing.
func TestEnsureRoleSkill_HermesSkillPathMatchesPreload(t *testing.T) {
	home := t.TempDir()

	res, err := EnsureRoleSkill(Options{Target: "hermes", HomeDir: home}, "builder")
	if err != nil {
		t.Fatal(err)
	}

	skill := filepath.Join(home, ".hermes", "skills", "spekk-builder", "SKILL.md")
	if len(res.Written) != 1 || res.Written[0] != skill {
		t.Fatalf("EnsureRoleSkill wrote %v, want only the hermes builder skill %s", res.Written, skill)
	}
	body := string(mustRead(t, skill))
	if !strings.Contains(body, "name: spekk-builder") {
		t.Errorf("hermes skill must keep its frontmatter name so `chat -s spekk-builder` resolves it: %q", body)
	}
	if !strings.Contains(body, "`spekk prompt builder`") {
		t.Errorf("builder skill body must run spekk prompt builder: %q", body)
	}
}

// The install is idempotent: a second call over a pristine, up-to-date skill
// rewrites nothing. This is what lets the interactive launcher call it before
// every session without churn.
func TestEnsureRoleSkill_Idempotent(t *testing.T) {
	home := t.TempDir()
	opts := Options{Target: "opencode", HomeDir: home}

	if _, err := EnsureRoleSkill(opts, "builder"); err != nil {
		t.Fatal(err)
	}
	res, err := EnsureRoleSkill(opts, "builder")
	if err != nil {
		t.Fatalf("second EnsureRoleSkill should succeed: %v", err)
	}
	if len(res.Written) != 0 {
		t.Errorf("re-ensure not idempotent: rewrote %v", res.Written)
	}
}

// codex keeps every skill and the observer shim in one prompts/ directory.
// EnsureRoleSkill must add/refresh only the one role file there and leave the
// sibling spekk files alone — the guard that it uses writeDesired, which does
// not prune, rather than reconcile, which would delete every undesired file in
// the directory.
func TestEnsureRoleSkill_DoesNotPruneSharedDir(t *testing.T) {
	home := t.TempDir()

	// A full codex install seeds the shared prompts/ dir with every spekk file.
	if _, err := Install(Options{Target: "codex", HomeDir: home, SkillFS: fakeSkillFS()}); err != nil {
		t.Fatal(err)
	}
	prompts := filepath.Join(home, ".codex", "prompts")
	siblings := []string{"spekk-builder.md", "spekk-dev-loop.md", "spekk-observer.md"}
	for _, s := range siblings {
		if _, err := os.Stat(filepath.Join(prompts, s)); err != nil {
			t.Fatalf("setup: expected %s from full install: %v", s, err)
		}
	}

	// Ensuring just the coach skill must not remove its siblings.
	if _, err := EnsureRoleSkill(Options{Target: "codex", HomeDir: home}, "coach"); err != nil {
		t.Fatal(err)
	}
	for _, s := range siblings {
		if _, err := os.Stat(filepath.Join(prompts, s)); err != nil {
			t.Errorf("EnsureRoleSkill pruned sibling %s from the shared prompts dir: %v", s, err)
		}
	}
}

func TestEnsureRoleSkill_RejectsUnknownRole(t *testing.T) {
	if _, err := EnsureRoleSkill(Options{Target: "opencode", HomeDir: t.TempDir()}, "observer"); err == nil {
		t.Fatal("EnsureRoleSkill should reject a role other than coach/builder")
	}
}
