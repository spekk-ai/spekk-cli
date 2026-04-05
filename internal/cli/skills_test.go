package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDirs creates a temp directory structure for skill resolution tests.
// Returns the base temp dir (to defer cleanup), and the workDir, homeDir, installDir.
func setupTestDirs(t *testing.T) (workDir, homeDir, installDir string) {
	t.Helper()
	base := t.TempDir()
	workDir = filepath.Join(base, "project")
	homeDir = filepath.Join(base, "home")
	installDir = filepath.Join(base, "install")
	return workDir, homeDir, installDir
}

// writeFile creates a file at the given path, creating parent directories as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newResolver(workDir, homeDir, installDir string) *SkillResolver {
	return &SkillResolver{
		WorkDir:    workDir,
		HomeDir:    homeDir,
		InstallDir: installDir,
	}
}

// ---------------------------------------------------------------------------
// ResolveSkill tests
// ---------------------------------------------------------------------------

func TestResolveSkill_DirectFilenameMatch(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// Create a package-level skill.
	skillPath := filepath.Join(installDir, "specs", "coach-skills-system", "my-skill.md")
	writeFile(t, skillPath, "---\nid: my-skill\n---\n# My Skill\n")

	skill, err := sr.ResolveSkill("coach", "my-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill to be found, got nil")
	}
	if skill.Name != "my-skill" {
		t.Errorf("expected name 'my-skill', got %q", skill.Name)
	}
	if skill.File != skillPath {
		t.Errorf("expected file %q, got %q", skillPath, skill.File)
	}
}

func TestResolveSkill_LocalShadowsGlobal(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	localPath := filepath.Join(workDir, ".spekk", "skills", "coach", "my-skill.md")
	globalPath := filepath.Join(homeDir, ".spekk", "skills", "coach", "my-skill.md")

	writeFile(t, localPath, "# Local skill\n")
	writeFile(t, globalPath, "# Global skill\n")

	skill, err := sr.ResolveSkill("coach", "my-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.File != localPath {
		t.Errorf("expected local file %q, got %q (should shadow global)", localPath, skill.File)
	}
}

func TestResolveSkill_GlobalShadowsPackage(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	globalPath := filepath.Join(homeDir, ".spekk", "skills", "coach", "my-skill.md")
	pkgPath := filepath.Join(installDir, "specs", "coach-skills-system", "my-skill.md")

	writeFile(t, globalPath, "# Global skill\n")
	writeFile(t, pkgPath, "# Package skill\n")

	skill, err := sr.ResolveSkill("coach", "my-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.File != globalPath {
		t.Errorf("expected global file %q, got %q (should shadow package)", globalPath, skill.File)
	}
}

func TestResolveSkill_LegacyAlias(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// "meeting" is an alias for "meeting-notes-to-specs-skill"
	skillPath := filepath.Join(installDir, "specs", "coach-skills-system", "meeting-notes-to-specs-skill.md")
	writeFile(t, skillPath, "---\nid: meeting-notes-to-specs-skill\n---\n# Meeting notes\n")

	skill, err := sr.ResolveSkill("coach", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Name != "meeting-notes-to-specs-skill" {
		t.Errorf("expected name 'meeting-notes-to-specs-skill', got %q", skill.Name)
	}
}

func TestResolveSkill_LegacyAliasCoordinate(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// "coordinate" is an alias for "coordinator-skill"
	skillPath := filepath.Join(installDir, "specs", "coach-skills-system", "coordinator-skill.md")
	writeFile(t, skillPath, "# Coordinator\n")

	skill, err := sr.ResolveSkill("coach", "coordinate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Name != "coordinator-skill" {
		t.Errorf("expected name 'coordinator-skill', got %q", skill.Name)
	}
}

func TestResolveSkill_AliasOriginalNameFallback(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// The resolved alias name doesn't exist, but the original subcommand name does.
	skillPath := filepath.Join(installDir, "specs", "coach-skills-system", "meeting.md")
	writeFile(t, skillPath, "# Meeting shorthand\n")

	skill, err := sr.ResolveSkill("coach", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Name != "meeting" {
		t.Errorf("expected name 'meeting', got %q", skill.Name)
	}
}

func TestResolveSkill_FrontmatterIDMatch(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// File has a different name but matching frontmatter id.
	skillPath := filepath.Join(installDir, "specs", "coach-skills-system", "some-long-name.md")
	writeFile(t, skillPath, "---\nid: short-id\n---\n# Skill content\n")

	skill, err := sr.ResolveSkill("coach", "short-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Name != "some-long-name" {
		t.Errorf("expected name 'some-long-name', got %q", skill.Name)
	}
	if skill.ID != "short-id" {
		t.Errorf("expected ID 'short-id', got %q", skill.ID)
	}
}

func TestResolveSkill_FrontmatterIDMatchOriginalName(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// Legacy alias: "meeting" -> "meeting-notes-to-specs-skill"
	// File has frontmatter id matching the original subcommand name "meeting".
	skillPath := filepath.Join(installDir, "specs", "coach-skills-system", "different-file.md")
	writeFile(t, skillPath, "---\nid: meeting\n---\n# Meeting by frontmatter\n")

	skill, err := sr.ResolveSkill("coach", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Name != "different-file" {
		t.Errorf("expected name 'different-file', got %q", skill.Name)
	}
}

func TestResolveSkill_NotFound(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	skill, err := sr.ResolveSkill("coach", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill != nil {
		t.Errorf("expected nil skill, got %+v", skill)
	}
}

func TestResolveSkill_EmptyName(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	skill, err := sr.ResolveSkill("coach", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill != nil {
		t.Errorf("expected nil for empty name, got %+v", skill)
	}
}

func TestResolveSkill_BuilderAgent(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	skillPath := filepath.Join(installDir, "specs", "builder-skills", "my-builder-skill.md")
	writeFile(t, skillPath, "# Builder skill\n")

	skill, err := sr.ResolveSkill("builder", "my-builder-skill")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Name != "my-builder-skill" {
		t.Errorf("expected name 'my-builder-skill', got %q", skill.Name)
	}
}

func TestResolveSkill_MissingDirsDoNotError(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// None of the directories exist at all.
	skill, err := sr.ResolveSkill("coach", "anything")
	if err != nil {
		t.Fatalf("unexpected error for missing dirs: %v", err)
	}
	if skill != nil {
		t.Errorf("expected nil, got %+v", skill)
	}
}

// ---------------------------------------------------------------------------
// ListSkills tests
// ---------------------------------------------------------------------------

func TestListSkills_MergesLayers(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	writeFile(t, filepath.Join(workDir, ".spekk", "skills", "coach", "local-skill.md"), "# Local\n")
	writeFile(t, filepath.Join(homeDir, ".spekk", "skills", "coach", "global-skill.md"), "# Global\n")
	writeFile(t, filepath.Join(installDir, "specs", "coach-skills-system", "pkg-skill.md"), "# Package\n")

	skills, err := sr.ListSkills("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d: %+v", len(skills), skills)
	}

	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	for _, expected := range []string{"local-skill", "global-skill", "pkg-skill"} {
		if !names[expected] {
			t.Errorf("expected skill %q not found in %v", expected, names)
		}
	}
}

func TestListSkills_Deduplication(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// Same skill name exists in all three layers.
	writeFile(t, filepath.Join(workDir, ".spekk", "skills", "coach", "shared.md"), "# Local version\n")
	writeFile(t, filepath.Join(homeDir, ".spekk", "skills", "coach", "shared.md"), "# Global version\n")
	writeFile(t, filepath.Join(installDir, "specs", "coach-skills-system", "shared.md"), "# Package version\n")

	skills, err := sr.ListSkills("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill after dedup, got %d: %+v", len(skills), skills)
	}
	if skills[0].Name != "shared" {
		t.Errorf("expected name 'shared', got %q", skills[0].Name)
	}

	// Should be the local version (highest priority).
	localDir := filepath.Join(workDir, ".spekk", "skills", "coach")
	if skills[0].Source != localDir {
		t.Errorf("expected local source %q, got %q", localDir, skills[0].Source)
	}
}

func TestListSkills_EmptyWhenNoDirs(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	skills, err := sr.ListSkills("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestListSkills_IgnoresNonMDFiles(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	dir := filepath.Join(workDir, ".spekk", "skills", "coach")
	writeFile(t, filepath.Join(dir, "skill.md"), "# Skill\n")
	writeFile(t, filepath.Join(dir, "readme.txt"), "Not a skill\n")
	writeFile(t, filepath.Join(dir, "config.json"), "{}")

	skills, err := sr.ListSkills("coach")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "skill" {
		t.Errorf("expected name 'skill', got %q", skills[0].Name)
	}
}

// ---------------------------------------------------------------------------
// ListAliases tests
// ---------------------------------------------------------------------------

func TestListAliases_Coach(t *testing.T) {
	sr := &SkillResolver{}
	aliases := sr.ListAliases("coach")

	expected := map[string]string{
		"meeting":    "meeting-notes-to-specs-skill",
		"coordinate": "coordinator-skill",
	}

	if len(aliases) != len(expected) {
		t.Fatalf("expected %d aliases, got %d: %v", len(expected), len(aliases), aliases)
	}
	for k, v := range expected {
		if aliases[k] != v {
			t.Errorf("alias %q: expected %q, got %q", k, v, aliases[k])
		}
	}
}

func TestListAliases_Builder(t *testing.T) {
	sr := &SkillResolver{}
	aliases := sr.ListAliases("builder")
	if len(aliases) != 0 {
		t.Errorf("expected empty aliases for builder, got %v", aliases)
	}
}

func TestListAliases_UnknownAgent(t *testing.T) {
	sr := &SkillResolver{}
	aliases := sr.ListAliases("unknown-agent")
	if aliases == nil {
		t.Error("expected non-nil map for unknown agent")
	}
	if len(aliases) != 0 {
		t.Errorf("expected empty aliases for unknown agent, got %v", aliases)
	}
}

func TestListAliases_ReturnsCopy(t *testing.T) {
	sr := &SkillResolver{}
	aliases1 := sr.ListAliases("coach")
	aliases1["test"] = "mutation"

	aliases2 := sr.ListAliases("coach")
	if _, ok := aliases2["test"]; ok {
		t.Error("mutation of returned alias map should not affect subsequent calls")
	}
}

// ---------------------------------------------------------------------------
// parseFrontmatterID tests
// ---------------------------------------------------------------------------

func TestParseFrontmatterID_Valid(t *testing.T) {
	content := "---\nid: my-skill\npriority: 1\n---\n# Title\n"
	id := parseFrontmatterID(content)
	if id != "my-skill" {
		t.Errorf("expected 'my-skill', got %q", id)
	}
}

func TestParseFrontmatterID_NoFrontmatter(t *testing.T) {
	content := "# Just a title\nSome content\n"
	id := parseFrontmatterID(content)
	if id != "" {
		t.Errorf("expected empty, got %q", id)
	}
}

func TestParseFrontmatterID_NoIDField(t *testing.T) {
	content := "---\npriority: 1\nstatus: done\n---\n# Title\n"
	id := parseFrontmatterID(content)
	if id != "" {
		t.Errorf("expected empty for no id field, got %q", id)
	}
}

func TestParseFrontmatterID_CRLF(t *testing.T) {
	content := "---\r\nid: crlf-skill\r\npriority: 1\r\n---\r\n# Title\r\n"
	id := parseFrontmatterID(content)
	if id != "crlf-skill" {
		t.Errorf("expected 'crlf-skill', got %q", id)
	}
}

func TestParseFrontmatterID_ExtraSpaces(t *testing.T) {
	content := "---\nid:   spaced-id   \n---\n# Title\n"
	id := parseFrontmatterID(content)
	if id != "spaced-id" {
		t.Errorf("expected 'spaced-id', got %q", id)
	}
}

// ---------------------------------------------------------------------------
// skillDirs tests
// ---------------------------------------------------------------------------

func TestSkillDirs_CoachOrder(t *testing.T) {
	sr := &SkillResolver{
		WorkDir:    "/project",
		HomeDir:    "/home/user",
		InstallDir: "/install",
	}

	dirs := sr.skillDirs("coach")
	expected := []string{
		filepath.Join("/project", ".spekk", "skills", "coach"),
		filepath.Join("/home/user", ".spekk", "skills", "coach"),
		filepath.Join("/install", "specs", "coach-skills-system"),
	}

	if len(dirs) != len(expected) {
		t.Fatalf("expected %d dirs, got %d: %v", len(expected), len(dirs), dirs)
	}
	for i, exp := range expected {
		if dirs[i] != exp {
			t.Errorf("dir[%d]: expected %q, got %q", i, exp, dirs[i])
		}
	}
}

func TestSkillDirs_BuilderOrder(t *testing.T) {
	sr := &SkillResolver{
		WorkDir:    "/project",
		HomeDir:    "/home/user",
		InstallDir: "/install",
	}

	dirs := sr.skillDirs("builder")
	expected := []string{
		filepath.Join("/project", ".spekk", "skills", "builder"),
		filepath.Join("/home/user", ".spekk", "skills", "builder"),
		filepath.Join("/install", "specs", "builder-skills"),
	}

	if len(dirs) != len(expected) {
		t.Fatalf("expected %d dirs, got %d: %v", len(expected), len(dirs), dirs)
	}
	for i, exp := range expected {
		if dirs[i] != exp {
			t.Errorf("dir[%d]: expected %q, got %q", i, exp, dirs[i])
		}
	}
}

func TestSkillDirs_UnknownAgentFallback(t *testing.T) {
	sr := &SkillResolver{
		WorkDir:    "/project",
		HomeDir:    "/home/user",
		InstallDir: "/install",
	}

	dirs := sr.skillDirs("observer")
	if len(dirs) != 4 {
		t.Fatalf("expected 4 dirs for unknown agent, got %d: %v", len(dirs), dirs)
	}
	// Should include fallback suffixes.
	if dirs[2] != filepath.Join("/install", "specs", "observer-skills-system") {
		t.Errorf("expected fallback -skills-system, got %q", dirs[2])
	}
	if dirs[3] != filepath.Join("/install", "specs", "observer-skills") {
		t.Errorf("expected fallback -skills, got %q", dirs[3])
	}
}

// ---------------------------------------------------------------------------
// Integration-style tests
// ---------------------------------------------------------------------------

func TestResolveSkill_FullLayerPriority(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	localDir := filepath.Join(workDir, ".spekk", "skills", "coach")
	globalDir := filepath.Join(homeDir, ".spekk", "skills", "coach")
	pkgDir := filepath.Join(installDir, "specs", "coach-skills-system")

	writeFile(t, filepath.Join(localDir, "skill-a.md"), "# Local A\n")
	writeFile(t, filepath.Join(globalDir, "skill-a.md"), "# Global A\n")
	writeFile(t, filepath.Join(pkgDir, "skill-a.md"), "# Package A\n")

	// Should resolve to local.
	skill, err := sr.ResolveSkill("coach", "skill-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill, got nil")
	}
	if skill.Source != localDir {
		t.Errorf("expected source %q, got %q", localDir, skill.Source)
	}

	// Remove local, should fall through to global.
	os.Remove(filepath.Join(localDir, "skill-a.md"))
	skill, err = sr.ResolveSkill("coach", "skill-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill after removing local, got nil")
	}
	if skill.Source != globalDir {
		t.Errorf("expected source %q after removing local, got %q", globalDir, skill.Source)
	}

	// Remove global, should fall through to package.
	os.Remove(filepath.Join(globalDir, "skill-a.md"))
	skill, err = sr.ResolveSkill("coach", "skill-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill after removing global, got nil")
	}
	if skill.Source != pkgDir {
		t.Errorf("expected source %q after removing global, got %q", pkgDir, skill.Source)
	}
}

func TestResolveSkill_AliasWithFrontmatterFallback(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// The alias resolves to "meeting-notes-to-specs-skill" but neither that
	// nor "meeting.md" exists. However, a file with matching frontmatter id does.
	skillPath := filepath.Join(installDir, "specs", "coach-skills-system", "notes-skill.md")
	writeFile(t, skillPath, "---\nid: meeting-notes-to-specs-skill\n---\n# Notes\n")

	skill, err := sr.ResolveSkill("coach", "meeting")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill == nil {
		t.Fatal("expected skill via frontmatter id, got nil")
	}
	if skill.Name != "notes-skill" {
		t.Errorf("expected name 'notes-skill', got %q", skill.Name)
	}
	if skill.ID != "meeting-notes-to-specs-skill" {
		t.Errorf("expected id 'meeting-notes-to-specs-skill', got %q", skill.ID)
	}
}

func TestListSkills_MultipleFilesInOneLayer(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	dir := filepath.Join(installDir, "specs", "builder-skills")
	writeFile(t, filepath.Join(dir, "alpha.md"), "# Alpha\n")
	writeFile(t, filepath.Join(dir, "beta.md"), "# Beta\n")
	writeFile(t, filepath.Join(dir, "gamma.md"), "# Gamma\n")

	skills, err := sr.ListSkills("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(skills))
	}
}

func TestListSkills_DeduplicatesAcrossLayers(t *testing.T) {
	workDir, homeDir, installDir := setupTestDirs(t)
	sr := newResolver(workDir, homeDir, installDir)

	// "shared" in local and package; "only-pkg" in package only.
	writeFile(t, filepath.Join(workDir, ".spekk", "skills", "builder", "shared.md"), "# Local\n")
	writeFile(t, filepath.Join(installDir, "specs", "builder-skills", "shared.md"), "# Package\n")
	writeFile(t, filepath.Join(installDir, "specs", "builder-skills", "only-pkg.md"), "# Only in package\n")

	skills, err := sr.ListSkills("builder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d: %+v", len(skills), skills)
	}

	names := make(map[string]string)
	for _, s := range skills {
		names[s.Name] = s.Source
	}
	localDir := filepath.Join(workDir, ".spekk", "skills", "builder")
	if names["shared"] != localDir {
		t.Errorf("shared should come from local %q, got %q", localDir, names["shared"])
	}
}
