package install

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStamp_RoundTrip(t *testing.T) {
	cases := [][]byte{
		[]byte(""),
		[]byte("one line no newline"),
		[]byte("trailing newline\n"),
		[]byte("multi\nline\nbody\n"),
		[]byte("body that ends with a marker-like line <!-- not a stamp -->"),
	}
	for _, body := range cases {
		stamped := StampContent(body)
		got, hash, managed := ParseStamp(stamped)
		if !managed {
			t.Errorf("ParseStamp(StampContent(%q)) = managed false, want true", body)
			continue
		}
		if !bytes.Equal(got, body) {
			t.Errorf("round trip body = %q, want %q", got, body)
		}
		if hash != bodyHash(body) {
			t.Errorf("round trip hash = %q, want %q", hash, bodyHash(body))
		}
	}
}

func TestParseStamp_Unstamped(t *testing.T) {
	content := []byte("a plain user file\nwith no marker\n")
	got, _, managed := ParseStamp(content)
	if managed {
		t.Errorf("plain content reported as managed")
	}
	if !bytes.Equal(got, content) {
		t.Errorf("unstamped content changed: %q", got)
	}
}

func TestIsPristine(t *testing.T) {
	stamped := StampContent([]byte("hello"))
	if managed, pristine := isPristine(stamped); !managed || !pristine {
		t.Errorf("fresh stamp: managed=%v pristine=%v, want true true", managed, pristine)
	}
	// Change the body but keep the old stamp: no longer pristine.
	edited := bytes.Replace(stamped, []byte("hello"), []byte("HELLO"), 1)
	if managed, pristine := isPristine(edited); !managed || pristine {
		t.Errorf("edited stamp: managed=%v pristine=%v, want true false", managed, pristine)
	}
}

func TestScanOwned(t *testing.T) {
	dir := t.TempDir()
	managed := filepath.Join(dir, "managed.md")
	if err := os.WriteFile(managed, StampContent([]byte("m")), 0o644); err != nil {
		t.Fatal(err)
	}
	user := filepath.Join(dir, "user.md")
	if err := os.WriteFile(user, []byte("a user file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "does-not-exist")

	owned, err := scanOwned([]string{dir, missing})
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 1 || owned[0].Path != managed {
		t.Fatalf("owned = %+v, want only %s", owned, managed)
	}
	if !owned[0].Pristine {
		t.Errorf("managed file should be pristine")
	}
}

func TestReconcile_WritesPrunesIdempotent(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.md")
	p2 := filepath.Join(dir, "b.md")
	desired := map[string][]byte{p1: []byte("hello A"), p2: []byte("hello B")}

	// A stray stamped file that is not desired must be pruned.
	stray := filepath.Join(dir, "stray.md")
	if err := os.WriteFile(stray, StampContent([]byte("old")), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := reconcile(desired, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 2 {
		t.Errorf("written = %v, want 2 files", res.Written)
	}
	if len(res.Removed) != 1 || res.Removed[0] != stray {
		t.Errorf("removed = %v, want [%s]", res.Removed, stray)
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("stray file was not pruned")
	}
	// Written files carry a stamp with the right body.
	got, _, managed := ParseStamp(mustRead(t, p1))
	if !managed || string(got) != "hello A" {
		t.Errorf("p1 body = %q managed=%v", got, managed)
	}

	// Second run: nothing changes.
	res2, err := reconcile(desired, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res2.Written) != 0 || len(res2.Removed) != 0 {
		t.Errorf("second run not idempotent: written=%v removed=%v", res2.Written, res2.Removed)
	}
}

// TestReconcile_UpdatesFileAtDesiredPath: a desired path belongs to spekk by its
// path, so the reconciler brings it to the current content whatever is there. It
// keeps what it replaced in a .bak, unless the file is a pristine managed file
// from another spekk version.
func TestReconcile_UpdatesFileAtDesiredPath(t *testing.T) {
	cases := []struct {
		name       string
		existing   []byte
		wantBackup bool
	}{
		{
			name:       "managed file the user edited by hand",
			existing:   bytes.Replace(StampContent([]byte("v1 body")), []byte("v1 body"), []byte("EDITED BY USER"), 1),
			wantBackup: true,
		},
		{
			// No stamp and none of the markers the old content sniff looked for.
			name:       "unstamped file of any content",
			existing:   []byte("my own agent, nothing to do with the tool\n"),
			wantBackup: true,
		},
		{
			name:       "pristine managed file from another version",
			existing:   StampContent([]byte("v1 body")),
			wantBackup: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "a.md")
			if err := os.WriteFile(p, tc.existing, 0o644); err != nil {
				t.Fatal(err)
			}

			res, err := reconcile(map[string][]byte{p: []byte("v2 body")}, []string{dir})
			if err != nil {
				t.Fatal(err)
			}
			if len(res.Written) != 1 || res.Written[0] != p {
				t.Errorf("written = %v, want [%s]", res.Written, p)
			}
			body, _, managed := ParseStamp(mustRead(t, p))
			if !managed || string(body) != "v2 body" {
				t.Errorf("file = managed %v body %q, want the current stamped content", managed, body)
			}
			bak, err := os.ReadFile(p + ".bak")
			switch {
			case tc.wantBackup && err != nil:
				t.Errorf(".bak backup not written: %v", err)
			case tc.wantBackup && !bytes.Equal(bak, tc.existing):
				t.Errorf(".bak = %q, want the replaced file %q", bak, tc.existing)
			case !tc.wantBackup && err == nil:
				t.Errorf("a pristine managed file should need no backup")
			}
			wantWarnings := 0
			if tc.wantBackup {
				wantWarnings = 1
			}
			if len(res.Warnings) != wantWarnings {
				t.Errorf("warnings = %v, want %d", res.Warnings, wantWarnings)
			}
		})
	}
}

func TestReconcile_DoesNotPruneEditedFile(t *testing.T) {
	dir := t.TempDir()
	stray := filepath.Join(dir, "stray.md")
	edited := bytes.Replace(StampContent([]byte("orig")), []byte("orig"), []byte("USER TOUCHED"), 1)
	if err := os.WriteFile(stray, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := reconcile(map[string][]byte{}, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Removed) != 0 {
		t.Errorf("edited stray was pruned: %v", res.Removed)
	}
	if _, err := os.Stat(stray); err != nil {
		t.Errorf("edited stray should still exist: %v", err)
	}
	if _, err := os.Stat(stray + ".bak"); err != nil {
		t.Errorf(".bak backup for stray not written")
	}
	if len(res.Warnings) != 1 {
		t.Errorf("warnings = %v, want 1", res.Warnings)
	}
}

func TestCheckStale(t *testing.T) {
	home := t.TempDir()
	skillFS := fakeSkillFS()

	// Install claude-code globally: this is the current desired layout.
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	// No stale files right after a clean install. A target the user never
	// installed (every one but claude-code) stays silent too.
	stale, err := CheckStale(home, "", skillFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("clean install reports stale files: %v", stale)
	}

	// Drop a stamped file that the current layout does not write (an old agent),
	// and edit an installed file so it no longer matches this binary.
	old := filepath.Join(home, ".claude", "agents", "spekk-old.md")
	if err := os.WriteFile(old, StampContent([]byte("legacy")), 0o644); err != nil {
		t.Fatal(err)
	}
	outdated := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	if err := os.WriteFile(outdated, []byte("an older dev-loop skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, err = CheckStale(home, "", skillFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 2 {
		t.Fatalf("stale = %v, want one old-layout entry and one out-of-date entry", stale)
	}
	assertStale(t, stale, old, "is from an old layout")
	assertStale(t, stale, outdated, "is out of date")
}

// assertStale fails unless exactly one stale line names path, says why, and
// shows the install command that fixes it.
func assertStale(t *testing.T, stale []string, path, reason string) {
	t.Helper()
	var got string
	for _, s := range stale {
		if strings.Contains(s, path) {
			if got != "" {
				t.Fatalf("%s reported more than once: %v", path, stale)
			}
			got = s
		}
	}
	if got == "" {
		t.Fatalf("no stale entry for %s: %v", path, stale)
	}
	if !strings.Contains(got, reason) {
		t.Errorf("stale entry %q does not say %q", got, reason)
	}
	if !strings.Contains(got, "spekk install --target claude-code") {
		t.Errorf("stale entry lacks the install command: %q", got)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
