package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	spekk "github.com/spekk-ai/spekk-cli"
	"github.com/spekk-ai/spekk-cli/internal/install"
)

// installInto puts a real claude-code install under a temp home, using the same
// embedded files the reporting path compares against.
func installInto(t *testing.T, home string) {
	t.Helper()
	if _, err := install.Install(install.Options{
		Target:  "claude-code",
		HomeDir: home,
		SkillFS: spekk.EmbeddedFS,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
}

// TestReportStale_SilentOnACleanInstall: the binary that wrote the files must
// never report them.
func TestReportStale_SilentOnACleanInstall(t *testing.T) {
	home := t.TempDir()
	installInto(t, home)

	var buf bytes.Buffer
	reportStale(&buf, home, "")
	if buf.Len() != 0 {
		t.Errorf("a clean install reported:\n%s", buf.String())
	}
}

// TestReportStale_SilentWithNoInstall: a user who never ran install sees
// nothing, whatever the binary holds.
func TestReportStale_SilentWithNoInstall(t *testing.T) {
	var buf bytes.Buffer
	reportStale(&buf, t.TempDir(), "")
	if buf.Len() != 0 {
		t.Errorf("an empty home reported:\n%s", buf.String())
	}
}

// TestReportStale_NamesTheFileAndTheFix: an edited file is reported as out of
// date, with the install command that clears it.
func TestReportStale_NamesTheFileAndTheFix(t *testing.T) {
	home := t.TempDir()
	installInto(t, home)
	p := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	if err := os.WriteFile(p, []byte("edited by hand\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	reportStale(&buf, home, "")
	out := buf.String()
	if !strings.Contains(out, p) {
		t.Errorf("the report does not name %s:\n%s", p, out)
	}
	if !strings.Contains(out, "is out of date") {
		t.Errorf("the report does not give the reason:\n%s", out)
	}
	if !strings.Contains(out, "spekk install --target claude-code") {
		t.Errorf("the report does not give the fix:\n%s", out)
	}
}

// TestReportStale_SymlinkAsksForAnOwnerAndShowsNoCommand: an install cannot
// settle a path a second tool owns, so the report must not suggest one.
func TestReportStale_SymlinkAsksForAnOwnerAndShowsNoCommand(t *testing.T) {
	home := t.TempDir()
	installInto(t, home)
	p := filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
	far := filepath.Join(t.TempDir(), "dotfiles-SKILL.md")
	if err := os.WriteFile(far, []byte("the other tool's file\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(far, p); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	reportStale(&buf, home, "")
	out := buf.String()
	if !strings.Contains(out, far) {
		t.Errorf("the report does not name the link target:\n%s", out)
	}
	if !strings.Contains(out, "decide which tool owns this path") {
		t.Errorf("the report does not ask for an owner:\n%s", out)
	}
	if strings.Contains(out, "spekk install --target") {
		t.Errorf("an install cannot fix a symlink, so no command belongs here:\n%s", out)
	}
}

// TestReportReinstall_NamesTheInstalledTargets: after a real self-update the
// check cannot help, so the reminder names the commands to run.
func TestReportReinstall_NamesTheInstalledTargets(t *testing.T) {
	home := t.TempDir()
	installInto(t, home)

	var buf bytes.Buffer
	reportReinstall(&buf, home, "")
	out := buf.String()
	if !strings.Contains(out, "spekk install --target claude-code") {
		t.Errorf("the reminder does not name the install command:\n%s", out)
	}
}

// TestReportReinstall_SilentWithNoInstall: nothing on disk, nothing to re-run.
func TestReportReinstall_SilentWithNoInstall(t *testing.T) {
	var buf bytes.Buffer
	reportReinstall(&buf, t.TempDir(), "")
	if buf.Len() != 0 {
		t.Errorf("an empty home produced a reminder:\n%s", buf.String())
	}
}

// TestWarnCheckFailed_SaysTheCheckDidNotRun: a check that cannot run must not
// read as "nothing is stale".
func TestWarnCheckFailed_SaysTheCheckDidNotRun(t *testing.T) {
	var buf bytes.Buffer
	warnCheckFailed(&buf, errors.New("scanning /x: permission denied"))
	out := buf.String()
	if !strings.Contains(out, "could not check the installed spekk files") {
		t.Errorf("the warning does not say the check failed: %q", out)
	}
	if !strings.Contains(out, "permission denied") {
		t.Errorf("the warning drops the reason: %q", out)
	}
}
