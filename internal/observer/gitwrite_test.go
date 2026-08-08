package observer

import (
	"os"
	"strings"
	"testing"
)

// envOf turns the slice commitIdentityEnv returns into a map, so a test can
// assert on one variable without depending on the order of the four.
func envOf(entries []string) map[string]string {
	m := map[string]string{}
	for _, e := range entries {
		k, v, _ := strings.Cut(e, "=")
		m[k] = v
	}
	return m
}

// A sandbox carries no git identity, so commit-tree has nothing to author
// with and announce fails with "Author identity unknown". The flip commit
// then never lands, and an unmarked finding is announced again for ever.
func TestCommitIdentityIsSuppliedWhenTheEnvironmentHasNone(t *testing.T) {
	env := envOf(commitIdentityEnv(func(string) string { return "" }))

	for _, key := range []string{
		"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL",
		"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL",
	} {
		if env[key] == "" {
			t.Errorf("%s was not supplied; commit-tree would have no identity", key)
		}
	}
	if env["GIT_AUTHOR_NAME"] != defaultCommitName {
		t.Errorf("GIT_AUTHOR_NAME = %q, want %q", env["GIT_AUTHOR_NAME"], defaultCommitName)
	}
	if env["GIT_AUTHOR_EMAIL"] != defaultCommitEmail {
		t.Errorf("GIT_AUTHOR_EMAIL = %q, want %q", env["GIT_AUTHOR_EMAIL"], defaultCommitEmail)
	}
}

// An operator who sets an identity has said what they want. This is a
// fallback, never an override.
func TestCommitIdentityKeepsWhatTheEnvironmentAlreadySets(t *testing.T) {
	set := map[string]string{
		"GIT_AUTHOR_NAME":     "ada",
		"GIT_AUTHOR_EMAIL":    "ada@example.com",
		"GIT_COMMITTER_NAME":  "ada",
		"GIT_COMMITTER_EMAIL": "ada@example.com",
	}
	entries := commitIdentityEnv(func(k string) string { return set[k] })

	if len(entries) != 0 {
		t.Errorf("a fully configured environment was overridden with %v", entries)
	}
}

// A half-configured environment is completed rather than ignored: each of the
// four is considered on its own, so one missing variable still fails the
// commit if it is left unset.
func TestCommitIdentityFillsOnlyTheMissingVariables(t *testing.T) {
	set := map[string]string{"GIT_AUTHOR_NAME": "ada"}
	env := envOf(commitIdentityEnv(func(k string) string { return set[k] }))

	if _, filled := env["GIT_AUTHOR_NAME"]; filled {
		t.Error("GIT_AUTHOR_NAME was already set and must not be replaced")
	}
	if env["GIT_AUTHOR_EMAIL"] != defaultCommitEmail {
		t.Errorf("GIT_AUTHOR_EMAIL = %q, want the fallback %q", env["GIT_AUTHOR_EMAIL"], defaultCommitEmail)
	}
	if env["GIT_COMMITTER_NAME"] != defaultCommitName {
		t.Errorf("GIT_COMMITTER_NAME = %q, want the fallback %q", env["GIT_COMMITTER_NAME"], defaultCommitName)
	}
}

// The end-to-end case the suite could not see. Every other announce test
// builds its clone with `git config user.email`, so the fixture supplied what
// production does not: measured on 2026-08-08, no spekk sandbox carries a git
// identity at all. Announce therefore failed only in production, with
// "Author identity unknown", and the finding it had just delivered to chat
// stayed unmarked -- so the next run announced it again.
//
// The identity is stripped after the fixture commits are built, which is the
// production shape exactly: a repo with history, and nothing configured to
// write more.
func TestAnnounceMarksTheFindingWhenTheRepoHasNoIdentity(t *testing.T) {
	clone, origin := newAnnounceRepos(t)
	addObserverBranch(t, clone, "finding-a", "high", true)

	gitT(t, clone, "config", "--unset", "user.email")
	gitT(t, clone, "config", "--unset", "user.name")
	for _, key := range []string{
		"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL",
		"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	if code := Announce(announceOpts(t, clone, t.TempDir())); code != 0 {
		t.Fatalf("announce exit %d with no git identity configured", code)
	}

	content := gitT(t, origin, "show", "refs/heads/observer/finding-a:observations/finding-a.md")
	if !strings.Contains(content, "announced:") {
		t.Fatalf("the flip commit did not land, so the finding announces again:\n%s", content)
	}
	author := gitT(t, origin, "log", "-1", "--format=%an <%ae>", "refs/heads/observer/finding-a")
	if author != defaultCommitName+" <"+defaultCommitEmail+">" {
		t.Fatalf("flip commit author = %q, want the supplied fallback", author)
	}
}
