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

func TestReconcile_DoesNotClobberEditedDesiredFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	// A managed file the user edited by hand: keep the stale stamp, change body.
	edited := bytes.Replace(StampContent([]byte("original")), []byte("original"), []byte("EDITED BY USER"), 1)
	if err := os.WriteFile(p, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	desired := map[string][]byte{p: []byte("original")}

	res, err := reconcile(desired, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 0 {
		t.Errorf("edited file was overwritten: written=%v", res.Written)
	}
	if len(res.Warnings) != 1 {
		t.Errorf("warnings = %v, want 1", res.Warnings)
	}
	if !bytes.Contains(mustRead(t, p), []byte("EDITED BY USER")) {
		t.Errorf("file body was changed")
	}
	if !bytes.Contains(mustRead(t, p+".bak"), []byte("EDITED BY USER")) {
		t.Errorf(".bak backup was not written")
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

func TestReconcile_UnstampedUserFileAtDesiredPathIsNotClobbered(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	if err := os.WriteFile(p, []byte("hand-written, no stamp\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := reconcile(map[string][]byte{p: []byte("desired")}, []string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Written) != 0 {
		t.Errorf("unstamped user file was overwritten")
	}
	if !bytes.Contains(mustRead(t, p), []byte("hand-written")) {
		t.Errorf("user file body changed")
	}
	if _, err := os.Stat(p + ".bak"); err != nil {
		t.Errorf(".bak backup not written for user file")
	}
}

func TestCheckStale(t *testing.T) {
	home := t.TempDir()
	skillFS := fakeSkillFS()

	// Install claude-code globally: this is the current desired layout.
	if _, err := Install(Options{Target: "claude-code", HomeDir: home, SkillFS: skillFS}); err != nil {
		t.Fatal(err)
	}
	// No stale files right after a clean install.
	stale, err := CheckStale(home, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("clean install reports stale files: %v", stale)
	}

	// Drop a stamped file that the current layout does not write (an old agent).
	old := filepath.Join(home, ".claude", "agents", "spekk-old.md")
	if err := os.WriteFile(old, StampContent([]byte("legacy")), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err = CheckStale(home, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || !strings.Contains(stale[0], old) {
		t.Fatalf("stale = %v, want one entry for %s", stale, old)
	}
	if !strings.Contains(stale[0], "spekk install --target claude-code") {
		t.Errorf("stale entry lacks the install command: %q", stale[0])
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
