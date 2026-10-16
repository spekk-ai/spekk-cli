package observer

import (
	"os"
	"path/filepath"
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

// detachFromAmbientGitIdentity makes the test see the git identity of a bare
// sandbox, whatever the machine running it is configured with. It empties the
// four identity variables and points git at an empty global config with no
// system config, so nothing outside the repository can name an author.
//
// Without this the developer's own `~/.gitconfig` supplies an identity, the
// fallback never engages, and the test passes for a reason production does
// not have.
func detachFromAmbientGitIdentity(t *testing.T) {
	t.Helper()

	empty := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(empty, []byte("[user]\n\tuseConfigOnly = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", empty)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	for _, key := range []string{
		"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL",
		"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL",
	} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}
}

// A sandbox carries no git identity, so commit-tree has nothing to author
// with and announce fails with "Author identity unknown". The flip commit
// then never lands, and an unmarked finding is announced again for ever.
func TestCommitIdentityIsSuppliedWhenGitCanNameNoAuthor(t *testing.T) {
	env := envOf(commitIdentityEnv(func(string) bool { return false }))

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

// This is a fallback, never an override. git resolves an identity from the
// environment and from config alike, so a name in either one stands -- which
// is what keeps a provisioned sandbox committing under its own name instead
// of the shared constant.
func TestCommitIdentityYieldsToAnIdentityGitAlreadyHas(t *testing.T) {
	entries := commitIdentityEnv(func(string) bool { return true })

	if len(entries) != 0 {
		t.Errorf("git could already name an author, but it was overridden with %v", entries)
	}
}

// Author and committer are asked separately, so an identity git can build for
// one role and not the other is completed rather than ignored.
func TestCommitIdentityFillsOnlyTheUnresolvedRole(t *testing.T) {
	env := envOf(commitIdentityEnv(func(ident string) bool {
		return ident == "GIT_AUTHOR_IDENT"
	}))

	if _, filled := env["GIT_AUTHOR_NAME"]; filled {
		t.Error("git could name the author already, so it must not be replaced")
	}
	if _, filled := env["GIT_AUTHOR_EMAIL"]; filled {
		t.Error("git could name the author already, so it must not be replaced")
	}
	if env["GIT_COMMITTER_NAME"] != defaultCommitName {
		t.Errorf("GIT_COMMITTER_NAME = %q, want the fallback %q", env["GIT_COMMITTER_NAME"], defaultCommitName)
	}
	if env["GIT_COMMITTER_EMAIL"] != defaultCommitEmail {
		t.Errorf("GIT_COMMITTER_EMAIL = %q, want the fallback %q", env["GIT_COMMITTER_EMAIL"], defaultCommitEmail)
	}
}

// resolvesIdent asks git the same question commit-tree asks, so the two must
// agree. Read against a repository with an identity and against one without.
func TestResolvesIdentAgreesWithWhatCommitTreeAccepts(t *testing.T) {
	detachFromAmbientGitIdentity(t)

	dir := t.TempDir()
	gitT(t, dir, "init", "-q", "-b", "main")
	g := gitRunner{dir: dir}

	for _, ident := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		if g.resolvesIdent(ident) {
			t.Errorf("%s reported resolvable in a repository with no identity", ident)
		}
	}

	gitT(t, dir, "config", "user.name", "Ada")
	gitT(t, dir, "config", "user.email", "ada@example.com")

	for _, ident := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		if !g.resolvesIdent(ident) {
			t.Errorf("%s reported unresolvable although git config names one", ident)
		}
	}
}

// The end-to-end case the suite could not see. Every other announce test
// builds its clone with `git config user.email`, so the fixture supplied what
// production does not: measured on 2026-08-08, no spekk sandbox carried a git
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
	detachFromAmbientGitIdentity(t)

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

// A provisioned sandbox names itself in `git config --global`, so its flip
// commits must carry that name. Reading only the environment would look like
// a fallback and act as an override, because GIT_AUTHOR_* outranks config:
// every box in the fleet would then commit under the same constant, and a
// commit could no longer say which one wrote it.
func TestAnnounceCommitsUnderTheIdentityTheRepoConfigures(t *testing.T) {
	clone, origin := newAnnounceRepos(t)
	addObserverBranch(t, clone, "finding-a", "high", true)
	detachFromAmbientGitIdentity(t)

	gitT(t, clone, "config", "user.name", "sandbox-alpha")
	gitT(t, clone, "config", "user.email", "sandbox-alpha@spekk.local")

	if code := Announce(announceOpts(t, clone, t.TempDir())); code != 0 {
		t.Fatalf("announce exit %d", code)
	}

	author := gitT(t, origin, "log", "-1", "--format=%an <%ae>", "refs/heads/observer/finding-a")
	if author != "sandbox-alpha <sandbox-alpha@spekk.local>" {
		t.Fatalf("flip commit author = %q, want the configured sandbox identity", author)
	}
}
