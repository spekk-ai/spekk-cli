package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

func TestLocalDestination_BuildsCwdRelativePath(t *testing.T) {
	cwd := t.TempDir()
	got, err := Destination(cwd, "/home/u", ScopeLocal, "coach", "meeting-notes")
	if err != nil {
		t.Fatalf("Destination: %v", err)
	}
	want := filepath.Join(cwd, ".spekk", "skills", "coach", "meeting-notes.md")
	if got != want {
		t.Errorf("path: got %q want %q", got, want)
	}
}

func TestWriteSkill_CreatesDirsAndFileWithExpectedModes(t *testing.T) {
	cwd := t.TempDir()
	dest, err := Destination(cwd, "/home/u", ScopeLocal, "coach", "meeting-notes")
	if err != nil {
		t.Fatalf("Destination: %v", err)
	}

	body := []byte("---\nid: meeting-notes\n---\n# hello\n")
	if err := WriteSkill(dest, body, false); err != nil {
		t.Fatalf("WriteSkill: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("body mismatch: got %q want %q", got, body)
	}

	fi, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if got := fi.Mode().Perm(); got != 0644 {
		t.Errorf("file mode = %o, want 0644", got)
	}

	di, err := os.Stat(filepath.Dir(dest))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := di.Mode().Perm(); got != 0755 {
		t.Errorf("dir mode = %o, want 0755", got)
	}
}

func TestWriteSkill_CreatesNestedDirsWhenMissing(t *testing.T) {
	cwd := t.TempDir()
	// Confirm .spekk doesn't exist yet — the writer must create the whole tree.
	if _, err := os.Stat(filepath.Join(cwd, ".spekk")); !os.IsNotExist(err) {
		t.Fatalf("precondition: .spekk should not exist, got err=%v", err)
	}

	dest, _ := Destination(cwd, "/home/u", ScopeLocal, "builder", "my-skill")
	if err := WriteSkill(dest, []byte("body"), false); err != nil {
		t.Fatalf("WriteSkill: %v", err)
	}

	// Whole tree should now exist.
	for _, p := range []string{
		filepath.Join(cwd, ".spekk"),
		filepath.Join(cwd, ".spekk", "skills"),
		filepath.Join(cwd, ".spekk", "skills", "builder"),
	} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
		if !fi.IsDir() {
			t.Errorf("%s is not a directory", p)
		}
	}
}

func TestPerformInstall_GlobalScopeWritesUnderHomeAndIsResolvable(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	body := []byte("---\nid: my-skill\n---\n# global\n")

	out, err := PerformInstall(InstallRequest{
		Cwd:      cwd,
		HomeDir:  home,
		Scope:    ScopeGlobal,
		Agent:    "builder",
		Skill:    "my-skill",
		FetchFn:  func(agent, skill string) ([]byte, error) { return body, nil },
		FetchURL: func(url string) ([]byte, error) { return nil, nil },
	})
	if err != nil {
		t.Fatalf("PerformInstall: %v", err)
	}

	wantPath := filepath.Join(home, ".spekk", "skills", "builder", "my-skill.md")
	if !strings.Contains(out, wantPath) {
		t.Errorf("output should include absolute home-resolved path %q, got: %q", wantPath, out)
	}

	got, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("file content mismatch: got %q want %q", got, body)
	}

	di, err := os.Stat(filepath.Join(home, ".spekk", "skills", "builder"))
	if err != nil {
		t.Fatalf("stat agent dir: %v", err)
	}
	if mode := di.Mode().Perm(); mode != 0755 {
		t.Errorf("agent dir mode = %o, want 0755", mode)
	}

	// The success criteria require the installed skill to be discoverable
	// via SkillResolver.ResolveSkill without any resolver changes.
	resolver := &cli.SkillResolver{HomeDir: home, Cwd: cwd}
	resolved := resolver.ResolveSkill("builder", "my-skill")
	if resolved == nil {
		t.Fatalf("SkillResolver.ResolveSkill returned nil for globally-installed skill")
	}
	if resolved.Content != string(body) {
		t.Errorf("resolved content mismatch: got %q want %q", resolved.Content, body)
	}
}

func TestPerformInstall_RefusesWhenDestExistsWithoutForceAndSkipsFetch(t *testing.T) {
	cwd := t.TempDir()
	dest, _ := Destination(cwd, "/home/u", ScopeLocal, "coach", "meeting-notes")
	original := []byte("local customizations — do not clobber\n")
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(dest, original, 0644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	fetchCalls := 0
	urlCalls := 0
	_, err := PerformInstall(InstallRequest{
		Cwd:     cwd,
		HomeDir: "/home/u",
		Scope:   ScopeLocal,
		Agent:   "coach",
		Skill:   "meeting-notes",
		Force:   false,
		FetchFn: func(agent, skill string) ([]byte, error) {
			fetchCalls++
			return []byte("remote"), nil
		},
		FetchURL: func(url string) ([]byte, error) {
			urlCalls++
			return []byte("remote"), nil
		},
	})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, dest) {
		t.Errorf("error should name absolute conflicting path %q, got: %s", dest, msg)
	}
	if !strings.Contains(msg, "--force") {
		t.Errorf("error should mention --force, got: %s", msg)
	}
	if fetchCalls != 0 || urlCalls != 0 {
		t.Errorf("no HTTP fetch should occur on conflict; got fetchCalls=%d urlCalls=%d", fetchCalls, urlCalls)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("existing file was modified on conflict: got %q want %q", got, original)
	}
}

func TestPerformInstall_ForceOverwritesForRegistryAndSource(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{name: "registry", source: ""},
		{name: "source", source: "https://example.com/foo.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cwd := t.TempDir()
			dest, _ := Destination(cwd, "/home/u", ScopeLocal, "coach", "meeting-notes")
			if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(dest, []byte("stale\n"), 0644); err != nil {
				t.Fatalf("seed: %v", err)
			}

			fresh := []byte("fresh content\n")
			_, err := PerformInstall(InstallRequest{
				Cwd:      cwd,
				HomeDir:  "/home/u",
				Scope:    ScopeLocal,
				Agent:    "coach",
				Skill:    "meeting-notes",
				Source:   tc.source,
				Force:    true,
				FetchFn:  func(agent, skill string) ([]byte, error) { return fresh, nil },
				FetchURL: func(url string) ([]byte, error) { return fresh, nil },
			})
			if err != nil {
				t.Fatalf("PerformInstall with --force: %v", err)
			}

			got, err := os.ReadFile(dest)
			if err != nil {
				t.Fatalf("read after force: %v", err)
			}
			if string(got) != string(fresh) {
				t.Errorf("force did not overwrite: got %q want %q", got, fresh)
			}
		})
	}
}

func TestRunInstall_DefaultsToLocalAndPrintsAbsolutePath(t *testing.T) {
	cwd := t.TempDir()
	body := []byte("---\nid: meeting-notes\n---\n# m\n")

	out, err := PerformInstall(InstallRequest{
		Cwd:      cwd,
		HomeDir:  t.TempDir(),
		Scope:    ScopeLocal,
		Agent:    "coach",
		Skill:    "meeting-notes",
		FetchFn:  func(agent, skill string) ([]byte, error) { return body, nil },
		FetchURL: func(url string) ([]byte, error) { return nil, nil },
	})
	if err != nil {
		t.Fatalf("PerformInstall: %v", err)
	}

	wantPath := filepath.Join(cwd, ".spekk", "skills", "coach", "meeting-notes.md")
	if !strings.Contains(out, wantPath) {
		t.Errorf("output should include absolute path %q, got: %q", wantPath, out)
	}
	if strings.Count(out, "\n") > 1 {
		t.Errorf("output should be a single line, got: %q", out)
	}

	got, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("file content mismatch: got %q want %q", got, body)
	}
}

func TestDestination_RejectsPathTraversalSkillNames(t *testing.T) {
	bad := []string{"../../../etc/evil", "..", ".", "a/b", `a\b`, "foo/../bar"}
	for _, skill := range bad {
		if _, err := Destination("/home/u/proj", "/home/u", ScopeLocal, "coach", skill); err == nil {
			t.Errorf("Destination(%q) should be rejected, got nil error", skill)
		}
	}
}

func TestPerformInstall_RefusesTraversalNameWithoutFetchingOrWriting(t *testing.T) {
	cwd := t.TempDir()
	// A file the install must never reach by escaping the skills dir.
	outside := filepath.Join(cwd, "victim.md")
	if err := os.WriteFile(outside, []byte("untouched\n"), 0644); err != nil {
		t.Fatalf("seed outside file: %v", err)
	}

	fetchCalls := 0
	urlCalls := 0
	_, err := PerformInstall(InstallRequest{
		Cwd:     cwd,
		HomeDir: "/home/u",
		Scope:   ScopeLocal,
		Agent:   "coach",
		Skill:   "../../victim", // would resolve to <cwd>/victim.md without the guard
		Force:   true,           // force must NOT bypass the name guard
		FetchFn: func(agent, skill string) ([]byte, error) {
			fetchCalls++
			return []byte("malicious"), nil
		},
		FetchURL: func(url string) ([]byte, error) {
			urlCalls++
			return []byte("malicious"), nil
		},
	})
	if err == nil {
		t.Fatal("expected traversal skill name to be rejected, got nil")
	}
	if fetchCalls != 0 || urlCalls != 0 {
		t.Errorf("no fetch should occur for an invalid name; got fetchCalls=%d urlCalls=%d", fetchCalls, urlCalls)
	}
	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatalf("outside file should still exist: %v", err)
	}
	if string(got) != "untouched\n" {
		t.Errorf("outside file was clobbered: got %q", got)
	}
}
