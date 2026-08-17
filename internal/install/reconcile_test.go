package install

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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

// TestReconcile_LeavesASymlinkedPathAlone: a managed path that is a symlink has
// a second owner — a dotfiles manager, usually. The far end is not a path spekk
// owns, and only the user can say which tool should win, so spekk writes
// nothing, removes nothing, and says so.
func TestReconcile_LeavesASymlinkedPathAlone(t *testing.T) {
	dir := t.TempDir()
	elsewhere := filepath.Join(t.TempDir(), "dotfiles-copy.md")
	if err := os.WriteFile(elsewhere, []byte("the user's own file"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "a.md")
	if err := os.Symlink(elsewhere, p); err != nil {
		t.Skipf("this platform cannot make a symlink: %v", err)
	}

	res, err := reconcile(map[string][]byte{p: []byte("v2 body")}, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 0 || len(res.Removed) != 0 {
		t.Errorf("written=%v removed=%v, want neither", res.Written, res.Removed)
	}
	if got := string(mustRead(t, elsewhere)); got != "the user's own file" {
		t.Errorf("the far end of the link was written: %q", got)
	}
	fi, err := os.Lstat(p)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the link itself should still be there: %v", err)
	}
	if _, err := os.Stat(p + ".bak"); !os.IsNotExist(err) {
		t.Errorf("a path spekk did not write needs no backup")
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], elsewhere) {
		t.Errorf("warnings = %v, want one naming %s", res.Warnings, elsewhere)
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
	want := []StaleFile{
		{Path: old, Reason: StaleOldLayout, Target: "claude-code"},
		{Path: outdated, Reason: StaleOutOfDate, Target: "claude-code"},
	}
	sort.Slice(want, func(i, j int) bool { return want[i].Path < want[j].Path })
	if !reflect.DeepEqual(stale, want) {
		t.Fatalf("stale = %+v, want %+v", stale, want)
	}
	if got := stale[0].Remedy(); got != "run: spekk install --target claude-code" {
		t.Errorf("Remedy() = %q", got)
	}
}

// TestCheckStale_ReportsEachPathOnce: a user whose working directory is their
// home directory has the same .claude directory in both scopes. One file must
// not produce two lines with two contradictory fix commands.
func TestCheckStale_ReportsEachPathOnce(t *testing.T) {
	home := t.TempDir()
	skillFS := fakeSkillFS()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	if err := os.WriteFile(p, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, err := CheckStale(home, home, skillFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0].Path != p {
		t.Fatalf("stale = %+v, want one entry for %s", stale, p)
	}
	if stale[0].Project {
		t.Errorf("the global install command should win the tie, got %q", stale[0].Remedy())
	}
}

// TestCheckStale_ReportsASymlinkedPath: an install cannot fix a path a second
// tool owns, so the check must not advertise one. It names the link target and
// asks the user to choose an owner.
func TestCheckStale_ReportsASymlinkedPath(t *testing.T) {
	home := t.TempDir()
	skillFS := fakeSkillFS()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	elsewhere := filepath.Join(t.TempDir(), "dotfiles-copy.md")
	if err := os.WriteFile(elsewhere, []byte("an older skill"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, p); err != nil {
		t.Skipf("this platform cannot make a symlink: %v", err)
	}

	stale, err := CheckStale(home, "", skillFS)
	if err != nil {
		t.Fatal(err)
	}
	want := []StaleFile{{Path: p, Reason: StaleSymlink, Target: "claude-code", LinkTarget: elsewhere}}
	if !reflect.DeepEqual(stale, want) {
		t.Fatalf("stale = %+v, want %+v", stale, want)
	}
	if got := stale[0].Remedy(); got != "decide which tool owns this path" {
		t.Errorf("Remedy() = %q, want no install command", got)
	}
}

// TestReconcile_IgnoresItsOwnBackups: the backup of an edited managed file keeps
// the old stamp. If a scan counted it as owned, every install would back the
// backup up again and every check would report it as stale forever, with an
// install command that only makes more copies.
func TestReconcile_IgnoresItsOwnBackups(t *testing.T) {
	home := t.TempDir()
	skillFS := fakeSkillFS()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".claude", "skills", "spekk-dev-loop")
	p := filepath.Join(dir, "SKILL.md")
	// Edit the body and leave the trailing stamp marker in place, as a user
	// editing the prose of the skill would.
	cur := mustRead(t, p)
	body, _, _ := ParseStamp(cur)
	edited := bytes.Replace(cur, body, append([]byte("HAND EDIT\n"), body...), 1)
	if err := os.WriteFile(p, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	// The first install backs the edit up and updates the file; every install
	// after that is a no-op, so the directory stops growing.
	for i := 1; i <= 3; i++ {
		if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			names := []string{}
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("after install %d the skill directory holds %v, want SKILL.md and one backup", i, names)
		}
	}
	if !bytes.Equal(mustRead(t, p+".bak"), edited) {
		t.Errorf("the backup should hold the edit that was replaced")
	}
	stale, err := CheckStale(home, "", skillFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Errorf("a backup must not be reported as stale: %+v", stale)
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
