package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/install"
)

// TestHelpTextListsInstallCommands ensures the top-level commands table
// surfaces the three skill-management entry points so users discover them
// from `spekk help` alone.
func TestHelpTextListsInstallCommands(t *testing.T) {
	for _, cmd := range []string{"install", "uninstall", "skills"} {
		if !strings.Contains(helpText, cmd) {
			t.Errorf("helpText missing command %q:\n%s", cmd, helpText)
		}
	}
}

// TestInstallHelpDocumentsArgsAndFlags pins the assertion's required
// argument/flag surface for `spekk install --help` so doc drift here breaks
// the build.
func TestInstallHelpDocumentsArgsAndFlags(t *testing.T) {
	for _, want := range []string{"<agent>", "<skill>", "--global", "--local", "--source", "--force", "--list"} {
		if !strings.Contains(install.UsageText, want) {
			t.Errorf("install UsageText missing %q", want)
		}
	}
}

// TestUninstallHelpDocumentsArgsAndFlags pins the argument/flag surface for
// `spekk uninstall --help`.
func TestUninstallHelpDocumentsArgsAndFlags(t *testing.T) {
	for _, want := range []string{"<agent>", "<skill>", "--global", "--local"} {
		if !strings.Contains(install.UninstallUsageText, want) {
			t.Errorf("UninstallUsageText missing %q", want)
		}
	}
}

// TestSkillsHelpDocumentsListSubcommand pins that `spekk skills --help`
// surfaces the `list` subcommand.
func TestSkillsHelpDocumentsListSubcommand(t *testing.T) {
	if !strings.Contains(install.SkillsUsageText, "list") {
		t.Errorf("SkillsUsageText missing 'list' subcommand:\n%s", install.SkillsUsageText)
	}
}

// TestReadmeDocumentsInstallSystem validates the "Installing Skills" section
// of README.md covers everything the install assertion requires: four
// representative invocations, the registry repo path, the env var overrides
// for self-hosted mirrors, and the --force overwrite rule.
func TestReadmeDocumentsInstallSystem(t *testing.T) {
	readme := readReadme(t)

	if !strings.Contains(readme, "## Installing Skills") {
		t.Fatal(`README is missing the "## Installing Skills" section`)
	}

	section := installingSkillsSection(readme)

	wantSubstrings := map[string]string{
		"registry install example":  "spekk install coach meeting-notes",
		"--global install example":  "--global",
		"--source install example":  "--source https://",
		"uninstall example":         "spekk uninstall",
		"registry repo path":        "github.com/spekk-ai/spekk-skills",
		"raw mirror env var":        "SPEKK_SKILLS_RAW_BASE",
		"api mirror env var":        "SPEKK_SKILLS_API_BASE",
		"--force overwrite mention": "--force",
	}
	for label, needle := range wantSubstrings {
		if !strings.Contains(section, needle) {
			t.Errorf("Installing Skills section missing %s (%q)", label, needle)
		}
	}
}

func readReadme(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	path := filepath.Join(root, "README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	return string(data)
}

// installingSkillsSection returns the text of the "Installing Skills" section
// (up to the next "## " heading) so assertions don't accidentally match
// content in other sections.
func installingSkillsSection(readme string) string {
	const header = "## Installing Skills"
	start := strings.Index(readme, header)
	if start < 0 {
		return ""
	}
	rest := readme[start+len(header):]
	next := strings.Index(rest, "\n## ")
	if next < 0 {
		return rest
	}
	return rest[:next]
}
