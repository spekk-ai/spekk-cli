package install

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
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

// TestScanOwned_SkipsASymlinkToStampedContent: the prune half must not follow a
// link another tool made, even when the far end is a spekk file. Without the
// guard the scan would claim the link and remove it.
func TestScanOwned_SkipsASymlinkToStampedContent(t *testing.T) {
	dir := t.TempDir()
	far := filepath.Join(t.TempDir(), "far.md")
	if err := os.WriteFile(far, StampContent([]byte("spekk body")), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "spekk-coach.md")
	if err := os.Symlink(far, link); err != nil {
		t.Fatal(err)
	}

	owned, err := scanOwned([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 0 {
		t.Fatalf("a symlink must not be owned, got %+v", owned)
	}

	// The prune half must therefore leave both the link and its far end alone.
	if _, err := reconcile(map[string][]byte{}, []string{dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Errorf("the link was removed: %v", err)
	}
	if _, err := os.Stat(far); err != nil {
		t.Errorf("the far end was removed: %v", err)
	}
}

// TestScanOwned_SkipsAFileItCannotRead: two of these directories hold the user's
// own prompts beside spekk's files. One file another program locked down must
// not disable the whole scan.
func TestScanOwned_SkipsAFileItCannotRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file, so the case cannot be built")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "spekk-coach.md"), StampContent([]byte("body")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "someone-elses-note.md"), []byte("private"), 0o000); err != nil {
		t.Fatal(err)
	}

	owned, err := scanOwned([]string{dir})
	if err != nil {
		t.Fatalf("one unreadable file stopped the scan: %v", err)
	}
	if len(owned) != 1 {
		t.Fatalf("owned = %+v, want the one stamped file", owned)
	}
}

// TestIsBackupName_CoversBothForms: backupFile writes "<name>.bak" and falls
// back to "<name>.bak.<n>". The scan must skip both, or an install backs its own
// backup up again.
func TestIsBackupName_CoversBothForms(t *testing.T) {
	for _, name := range []string{"SKILL.md.bak", "SKILL.md.bak.1", "SKILL.md.bak.27"} {
		if !isBackupName(name) {
			t.Errorf("isBackupName(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"SKILL.md", "spekk-coach.md"} {
		if isBackupName(name) {
			t.Errorf("isBackupName(%q) = true, want false", name)
		}
	}
}

// TestBackupFile_NeverOverwritesAnEarlierBackup: a second, different version
// gets its own file. Overwriting the first one would destroy what the user had.
func TestBackupFile_NeverOverwritesAnEarlierBackup(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte("version one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := backupFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if first != p+".bak" {
		t.Fatalf("first backup = %q, want %q", first, p+".bak")
	}

	if err := os.WriteFile(p, []byte("version two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := backupFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if second != p+".bak.1" {
		t.Fatalf("second backup = %q, want %q", second, p+".bak.1")
	}
	if got := string(mustRead(t, first)); got != "version one\n" {
		t.Errorf("the first backup was overwritten: %q", got)
	}
}

// TestBackupFile_KeepsOneCopyOfTheSameContent: the same bytes must not pile up.
// A file the prune half backs up and leaves in place is seen on every install.
func TestBackupFile_KeepsOneCopyOfTheSameContent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte("one version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		bak, err := backupFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if bak != p+".bak" {
			t.Fatalf("call %d wrote %q, want the one backup %q", i, bak, p+".bak")
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("the directory holds %d files, want the file and one backup", len(entries))
	}
}

// TestBackupFile_KeepsTheMode: a private file must not become readable by way of
// its backup.
func TestBackupFile_KeepsTheMode(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte("private\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bak, err := backupFile(p)
	if err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(bak)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("the backup is mode %v, want 0600", fi.Mode().Perm())
	}
}

// TestReconcile_PruneHalfDoesNotGrowBackups: an edited file at a path the layout
// no longer writes is backed up and left in place. Every later install sees it
// again, and must not add a second copy of the same bytes.
func TestReconcile_PruneHalfDoesNotGrowBackups(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "spekk-old.md")
	edited := bytes.Replace(StampContent([]byte("original")), []byte("original"), []byte("the user's edit"), 1)
	if err := os.WriteFile(p, edited, 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 4; i++ {
		res, err := reconcile(map[string][]byte{}, []string{dir})
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		if len(res.Warnings) != 1 {
			t.Errorf("run %d gave %d warnings, want one every run", i, len(res.Warnings))
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 2 {
			var names []string
			for _, e := range entries {
				names = append(names, e.Name())
			}
			t.Fatalf("after run %d the directory holds %v, want the file and one backup", i, names)
		}
	}
}

// TestReconcile_WarningNamesTheBackupItWrote: the warning sends the user to
// their replaced content, so it must name the file that holds it.
func TestReconcile_WarningNamesTheBackupItWrote(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "spekk-coach.md")
	desired := map[string][]byte{p: []byte("what spekk installs")}

	if err := os.WriteFile(p, []byte("the user's first version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := reconcile(desired, []string{dir}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("the user's second version\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := reconcile(desired, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("warnings = %v, want one", res.Warnings)
	}
	if !strings.Contains(res.Warnings[0], p+".bak.1") {
		t.Errorf("the warning must name %s.bak.1, got: %s", p, res.Warnings[0])
	}
	if got := string(mustRead(t, p+".bak.1")); got != "the user's second version\n" {
		t.Errorf("the named backup holds %q", got)
	}
}

// TestCheckStale_ReportsEachPathOnceThroughASymlink: os.Getwd gives the spelling
// the shell used, so a working directory that reaches home through a link names
// the same .claude directory twice. One file must still be one report.
func TestCheckStale_ReportsEachPathOnceThroughASymlink(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link-to-home")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}

	skillFS := fakeSkillFS()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	if err := os.WriteFile(p, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stale, err := CheckStale(home, link, skillFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 {
		t.Fatalf("stale = %+v, want one entry", stale)
	}
	if stale[0].Project {
		t.Errorf("the global install command should win the tie, got %q", stale[0].Remedy())
	}
}

// TestInstalledTargets_NamesEachScopeOnce: the reminder after a self-update must
// not name one directory under two spellings.
func TestInstalledTargets_NamesEachScopeOnce(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link-to-home")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: fakeSkillFS()}); err != nil {
		t.Fatal(err)
	}

	cmds, err := InstalledTargets(home, link)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"spekk install --target claude-code"}
	if len(cmds) != 1 || cmds[0] != want[0] {
		t.Fatalf("cmds = %v, want %v", cmds, want)
	}
}

// TestInstalledTargets_SilentWithNoInstall: a user who never installed gets no
// reminder.
func TestInstalledTargets_SilentWithNoInstall(t *testing.T) {
	home := t.TempDir()
	cmds, err := InstalledTargets(home, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 0 {
		t.Errorf("cmds = %v, want none", cmds)
	}
}

// mkfifo makes a FIFO for a test, and skips the test where it cannot.
func mkfifo(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o644); err != nil {
		t.Skipf("cannot make a FIFO here: %v", err)
	}
}

// runsWithin fails the test when fn has not returned by d. A read of a FIFO
// never returns, so a guard that regresses must fail rather than hang the suite.
func runsWithin(t *testing.T, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not return within %s: it read something that never ends", what, d)
	}
}

// TestScanOwned_SkipsAFIFO: these directories hold other tools' files. A read of
// a FIFO would never return, so the scan must not open one.
func TestScanOwned_SkipsAFIFO(t *testing.T) {
	dir := t.TempDir()
	mkfifo(t, filepath.Join(dir, "spekk-coach.md"))
	if err := os.WriteFile(filepath.Join(dir, "spekk-builder.md"), StampContent([]byte("body")), 0o644); err != nil {
		t.Fatal(err)
	}

	var owned []OwnedFile
	var err error
	runsWithin(t, 5*time.Second, "scanOwned", func() {
		owned, err = scanOwned([]string{dir})
	})
	if err != nil {
		t.Fatalf("a FIFO stopped the scan: %v", err)
	}
	if len(owned) != 1 {
		t.Fatalf("owned = %+v, want the one stamped file", owned)
	}
}

// TestScanOwned_SkipsADirectoryItCannotRead: one tool's config directory closed
// to spekk must not stop the scan of every other directory.
func TestScanOwned_SkipsADirectoryItCannotRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every directory, so the case cannot be built")
	}
	base := t.TempDir()
	closed := filepath.Join(base, "closed")
	open := filepath.Join(base, "open")
	for _, d := range []string{closed, open} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(open, "spekk-coach.md"), StampContent([]byte("body")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(closed, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(closed, 0o755) })

	owned, err := scanOwned([]string{closed, open})
	if err != nil {
		t.Fatalf("one closed directory stopped the scan: %v", err)
	}
	if len(owned) != 1 {
		t.Fatalf("owned = %+v, want the one file in the open directory", owned)
	}
}

// TestBackupFile_SurvivesABackupItCannotCompare: a .bak spekk cannot read is a
// taken name, not a failure. An install must still finish.
func TestBackupFile_SurvivesABackupItCannotCompare(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file, so the case cannot be built")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte("the user's file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p+".bak", []byte("older\n"), 0o000); err != nil {
		t.Fatal(err)
	}

	bak, err := backupFile(p)
	if err != nil {
		t.Fatalf("an unreadable backup failed the whole call: %v", err)
	}
	if bak != p+".bak.1" {
		t.Fatalf("backup = %q, want the next free name %q", bak, p+".bak.1")
	}
	if got := string(mustRead(t, bak)); got != "the user's file\n" {
		t.Errorf("the backup holds %q", got)
	}
}

// TestBackupFile_SurvivesADirectoryInItsWay: same rule for a .bak that is a
// directory. os.ReadFile fails on one, and the install must not.
func TestBackupFile_SurvivesADirectoryInItsWay(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte("the user's file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(p+".bak", 0o755); err != nil {
		t.Fatal(err)
	}

	bak, err := backupFile(p)
	if err != nil {
		t.Fatalf("a directory in the way failed the whole call: %v", err)
	}
	if bak != p+".bak.1" {
		t.Fatalf("backup = %q, want %q", bak, p+".bak.1")
	}
}

// TestBackupFile_DoesNotWriteThroughASymlink: a .bak that is a link belongs to
// whoever made it. Writing through it would put the user's file somewhere they
// did not choose.
func TestBackupFile_DoesNotWriteThroughASymlink(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(p, []byte("the user's file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	far := filepath.Join(t.TempDir(), "far.md")
	if err := os.Symlink(far, p+".bak"); err != nil {
		t.Fatal(err)
	}

	bak, err := backupFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if bak != p+".bak.1" {
		t.Fatalf("backup = %q, want %q", bak, p+".bak.1")
	}
	if _, err := os.Stat(far); !os.IsNotExist(err) {
		t.Errorf("the backup was written through the link to %s", far)
	}
}

// TestReconcile_LeavesAPathItCannotRead: a FIFO at a desired path must not be
// read or written, and must not hang the install.
func TestReconcile_LeavesAPathItCannotRead(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "spekk-coach.md")
	mkfifo(t, p)
	desired := map[string][]byte{p: []byte("what spekk installs")}

	var res Result
	var err error
	runsWithin(t, 5*time.Second, "reconcile", func() {
		res, err = reconcile(desired, []string{dir})
	})
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}
	if len(res.Written) != 0 {
		t.Errorf("written = %v, want nothing", res.Written)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], p) {
		t.Fatalf("warnings = %v, want one that names %s", res.Warnings, p)
	}
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		t.Errorf("the FIFO was replaced: mode %v", fi.Mode())
	}
}

// TestReconcile_PruneWarningNamesTheBackupItWrote: the prune half names its
// backup too, and its backup is not always <path>.bak.
func TestReconcile_PruneWarningNamesTheBackupItWrote(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "spekk-old.md")
	edited := bytes.Replace(StampContent([]byte("original")), []byte("original"), []byte("the user's edit"), 1)
	if err := os.WriteFile(p, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	// A backup of a different version is already there, so the prune backup
	// has to take the next name.
	if err := os.WriteFile(p+backupSuffix, []byte("an older version\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := reconcile(map[string][]byte{}, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("warnings = %v, want one", res.Warnings)
	}
	if !strings.Contains(res.Warnings[0], p+".bak.1") {
		t.Errorf("the warning must name %s.bak.1, got: %s", p, res.Warnings[0])
	}
	if !bytes.Equal(mustRead(t, p+".bak.1"), edited) {
		t.Errorf("the named backup does not hold the edit")
	}
}

// TestCheckStale_ReportsAPathItCannotRead: no install can settle a path that
// holds something spekk cannot read, so the report must say so and must not
// offer an install command that would not help.
func TestCheckStale_ReportsAPathItCannotRead(t *testing.T) {
	home := t.TempDir()
	skillFS := fakeSkillFS()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	mkfifo(t, p)

	var stale []StaleFile
	var err error
	runsWithin(t, 5*time.Second, "CheckStale", func() {
		stale, err = CheckStale(home, "", skillFS)
	})
	if err != nil {
		t.Fatalf("CheckStale failed: %v", err)
	}
	if len(stale) != 1 || stale[0].Path != p {
		t.Fatalf("stale = %+v, want one entry for %s", stale, p)
	}
	if stale[0].Reason != StaleUnreadable {
		t.Errorf("reason = %v, want StaleUnreadable", stale[0].Reason)
	}
	if strings.Contains(stale[0].Remedy(), "spekk install") {
		t.Errorf("an install cannot settle this, so no install command belongs here: %q", stale[0].Remedy())
	}
}

// TestBackupFile_StopsOnANameItCannotLookAt: a name too long stays too long, so
// counting up to the next name would never end. The call must return instead.
func TestBackupFile_StopsOnANameItCannotLookAt(t *testing.T) {
	dir := t.TempDir()
	// Adding ".bak" to this name goes past the limit for one path element.
	p := filepath.Join(dir, strings.Repeat("a", 250)+".md")
	if err := os.WriteFile(p, []byte("the user's file\n"), 0o644); err != nil {
		t.Skipf("cannot build the case here: %v", err)
	}
	if _, err := os.Lstat(p + backupSuffix); err == nil || os.IsNotExist(err) {
		t.Skip("this filesystem accepts the longer name, so the case does not arise")
	}

	done := make(chan error, 1)
	go func() {
		_, err := backupFile(p)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Errorf("backupFile reported success for a name it cannot write")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("backupFile did not return: it is counting names forever")
	}
}

// TestScanOwned_SkipsAFileWhereADirectoryBelongs: a regular file at a tool's
// config path makes every read under it fail with ENOTDIR. One such path must
// not stop the scan of every other tool.
func TestScanOwned_SkipsAFileWhereADirectoryBelongs(t *testing.T) {
	base := t.TempDir()
	notADir := filepath.Join(base, "claude")
	if err := os.WriteFile(notADir, []byte("a file, not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	open := filepath.Join(base, "opencode")
	if err := os.Mkdir(open, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(open, "spekk-coach.md"), StampContent([]byte("body")), 0o644); err != nil {
		t.Fatal(err)
	}

	owned, err := scanOwned([]string{filepath.Join(notADir, "agents"), open})
	if err != nil {
		t.Fatalf("a file where a directory belongs stopped the scan: %v", err)
	}
	if len(owned) != 1 {
		t.Fatalf("owned = %+v, want the one file in the real directory", owned)
	}
}

// TestInspect_TreatsAFileWhereADirectoryBelongsAsAbsent: the same rule for a
// single path. Nothing can be at a path whose parent is a file.
func TestInspect_TreatsAFileWhereADirectoryBelongsAsAbsent(t *testing.T) {
	base := t.TempDir()
	notADir := filepath.Join(base, "claude")
	if err := os.WriteFile(notADir, []byte("a file\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found, err := inspect(filepath.Join(notADir, "agents", "spekk-coach.md"))
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if found.State != pathAbsent {
		t.Errorf("state = %v, want pathAbsent", found.State)
	}
}

// TestCheckStale_ReportsEachPathOnceWhenADirectoryIsClosed: the key must resolve
// the deepest ancestor it can. A closed directory defeats a whole-path resolve,
// and the two scopes would then report one file under two spellings.
func TestCheckStale_ReportsEachPathOnceWhenADirectoryIsClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root searches every directory, so the case cannot be built")
	}
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link-to-home")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	skillFS := fakeSkillFS()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	// Close the directory ABOVE the one that holds the file. Only then does
	// resolving the file's own parent fail, which is what makes the key fall
	// back to the spelling it was given.
	closed := filepath.Join(home, ".claude", "skills")
	if err := os.Chmod(closed, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(closed, 0o755) })

	stale, err := CheckStale(home, link, skillFS)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) == 0 {
		t.Fatal("nothing was reported, so the case does not exercise the key")
	}
	seen := map[string]int{}
	for _, f := range stale {
		seen[filepath.Base(filepath.Dir(f.Path))+"/"+filepath.Base(f.Path)]++
	}
	for name, n := range seen {
		if n > 1 {
			t.Errorf("%s was reported %d times; one file is one report", name, n)
		}
	}
}

// TestCheckStale_ReportsAPathInsideAClosedDirectory: the parent directory, not
// the file, can be the one spekk cannot open. os.Lstat fails then, and the check
// must report the path rather than stop.
func TestCheckStale_ReportsAPathInsideAClosedDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every directory, so the case cannot be built")
	}
	home := t.TempDir()
	skillFS := fakeSkillFS()
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	closed := filepath.Join(home, ".claude", "skills", "spekk-dev-loop")
	if err := os.Chmod(closed, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(closed, 0o755) })

	stale, err := CheckStale(home, "", skillFS)
	if err != nil {
		t.Fatalf("one closed directory stopped the check: %v", err)
	}
	want := filepath.Join(closed, "SKILL.md")
	for _, f := range stale {
		if f.Path == want {
			if f.Reason != StaleUnreadable {
				t.Errorf("reason = %v, want StaleUnreadable", f.Reason)
			}
			return
		}
	}
	t.Errorf("stale = %+v, want an entry for %s", stale, want)
}

// TestCheckStale_KeepsTwoManagedPathsThatPointAtOneFile: the key must not
// follow a symlink at the path itself. Each managed path is its own conflict,
// and each host tool needs its own answer.
func TestCheckStale_KeepsTwoManagedPathsThatPointAtOneFile(t *testing.T) {
	home := t.TempDir()
	skillFS := fakeSkillFS()
	for _, target := range []string{"claude-code", "opencode"} {
		if _, err := Install(Options{Target: target, HomeDir: home, SkillFS: skillFS}); err != nil {
			t.Fatal(err)
		}
	}

	far := filepath.Join(t.TempDir(), "one-file.md")
	if err := os.WriteFile(far, []byte("the dotfiles copy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var linked []string
	for _, rel := range [][]string{
		{".claude", "agents", "spekk-observer.md"},
		{".config", "opencode", "agents", "spekk-observer.md"},
	} {
		p := filepath.Join(home, filepath.Join(rel...))
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(far, p); err != nil {
			t.Fatal(err)
		}
		linked = append(linked, p)
	}

	stale, err := CheckStale(home, "", skillFS)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range stale {
		if f.Reason == StaleSymlink {
			got[f.Path] = true
		}
	}
	for _, p := range linked {
		if !got[p] {
			t.Errorf("%s was not reported; two paths that point at one file are two conflicts", p)
		}
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
