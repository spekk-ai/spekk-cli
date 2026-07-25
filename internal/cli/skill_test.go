package cli

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func setupSkillDirs(t *testing.T, agent string) (home, cwd, install string) {
	t.Helper()
	home = t.TempDir()
	cwd = t.TempDir()
	install = t.TempDir()
	return
}

func writeSkillFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644)
}

func newSkillResolver(globalConfigDir, cwd, install string) *SkillResolver {
	return &SkillResolver{GlobalConfigDir: globalConfigDir, Cwd: cwd, InstallDir: install}
}

func TestResolveSkill_DirectFilenameMatch(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "coach")
	localDir := filepath.Join(cwd, ".spekk", "skills", "coach")
	writeSkillFile(t, localDir, "my-skill.md", "# My Skill")

	r := newSkillResolver(home, cwd, install)
	skill := r.ResolveSkill("coach", "my-skill")

	if skill == nil {
		t.Fatal("expected skill to be found")
	}
	if skill.Name != "my-skill" {
		t.Errorf("expected name my-skill, got %s", skill.Name)
	}
	if skill.Content != "# My Skill" {
		t.Errorf("unexpected content: %s", skill.Content)
	}
	if skill.Source != localDir {
		t.Errorf("expected source %s, got %s", localDir, skill.Source)
	}
}

func TestResolveSkill_LegacyAlias(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "coach")
	pkgDir := filepath.Join(install, "specs", "coach-skills-system")
	writeSkillFile(t, pkgDir, "meeting-notes-to-specs-skill.md", "# Meeting Skill")

	r := newSkillResolver(home, cwd, install)
	skill := r.ResolveSkill("coach", "meeting")

	if skill == nil {
		t.Fatal("expected skill via legacy alias")
	}
	if skill.Name != "meeting-notes-to-specs-skill" {
		t.Errorf("expected resolved name, got %s", skill.Name)
	}
}

func TestResolveSkill_FrontmatterIDMatch(t *testing.T) {
	globalConfigDir, cwd, install := setupSkillDirs(t, "builder")
	globalDir := filepath.Join(globalConfigDir, "skills", "builder")
	writeSkillFile(t, globalDir, "some-file.md", "---\nid: custom-id\n---\n\n# Custom")

	r := newSkillResolver(globalConfigDir, cwd, install)
	skill := r.ResolveSkill("builder", "custom-id")

	if skill == nil {
		t.Fatal("expected skill via frontmatter id")
	}
	if skill.Name != "some-file" {
		t.Errorf("expected name some-file, got %s", skill.Name)
	}
}

func TestResolveSkill_LocalShadowsGlobal(t *testing.T) {
	globalConfigDir, cwd, install := setupSkillDirs(t, "coach")

	globalDir := filepath.Join(globalConfigDir, "skills", "coach")
	writeSkillFile(t, globalDir, "my-skill.md", "# Global Version")

	localDir := filepath.Join(cwd, ".spekk", "skills", "coach")
	writeSkillFile(t, localDir, "my-skill.md", "# Local Version")

	r := newSkillResolver(globalConfigDir, cwd, install)
	skill := r.ResolveSkill("coach", "my-skill")

	if skill == nil {
		t.Fatal("expected skill")
	}
	if skill.Content != "# Local Version" {
		t.Error("local should shadow global")
	}
	if skill.Source != localDir {
		t.Errorf("expected local source, got %s", skill.Source)
	}
}

func TestResolveSkill_PackageFallback(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "coach")
	pkgDir := filepath.Join(install, "specs", "coach-skills-system")
	writeSkillFile(t, pkgDir, "pkg-skill.md", "# Package Skill")

	r := newSkillResolver(home, cwd, install)
	skill := r.ResolveSkill("coach", "pkg-skill")

	if skill == nil {
		t.Fatal("expected package skill")
	}
	if skill.Source != pkgDir {
		t.Errorf("expected package source, got %s", skill.Source)
	}
}

func TestResolveSkill_EmptySubcommand(t *testing.T) {
	r := newSkillResolver(t.TempDir(), t.TempDir(), t.TempDir())
	if r.ResolveSkill("coach", "") != nil {
		t.Error("expected nil for empty subcommand")
	}
}

func TestResolveSkill_NotFound(t *testing.T) {
	r := newSkillResolver(t.TempDir(), t.TempDir(), t.TempDir())
	if r.ResolveSkill("coach", "nonexistent") != nil {
		t.Error("expected nil for missing skill")
	}
}

func TestListSkills_Deduplication(t *testing.T) {
	globalConfigDir, cwd, install := setupSkillDirs(t, "coach")

	localDir := filepath.Join(cwd, ".spekk", "skills", "coach")
	writeSkillFile(t, localDir, "shared.md", "# Local")
	writeSkillFile(t, localDir, "local-only.md", "# Local Only")

	globalDir := filepath.Join(globalConfigDir, "skills", "coach")
	writeSkillFile(t, globalDir, "shared.md", "# Global")
	writeSkillFile(t, globalDir, "global-only.md", "# Global Only")

	r := newSkillResolver(globalConfigDir, cwd, install)
	skills := r.ListSkills("coach")

	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}

	if !names["shared"] || !names["local-only"] || !names["global-only"] {
		t.Errorf("expected all three skills, got: %v", skills)
	}

	// Verify "shared" comes from local (first wins)
	for _, s := range skills {
		if s.Name == "shared" && s.Source != localDir {
			t.Errorf("shared should come from local dir, got %s", s.Source)
		}
	}
}

func TestListSkills_SkipsSpecDoc(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "coach")

	// The package skill dir contains the spec doc named after the
	// directory itself (e.g. coach-skills-system/coach-skills-system.md);
	// it must not be listed as a skill.
	pkgDir := filepath.Join(install, "specs", "coach-skills-system")
	writeSkillFile(t, pkgDir, "coach-skills-system.md", "# Spec doc, not a skill")
	writeSkillFile(t, pkgDir, "real-skill.md", "# Real Skill")

	r := newSkillResolver(home, cwd, install)
	skills := r.ListSkills("coach")

	for _, s := range skills {
		if s.Name == "coach-skills-system" {
			t.Errorf("spec doc should not be listed as a skill: %v", skills)
		}
	}
	found := false
	for _, s := range skills {
		if s.Name == "real-skill" {
			found = true
		}
	}
	if !found {
		t.Errorf("real-skill should be listed, got: %v", skills)
	}
}

func TestListSkills_EmptyDirs(t *testing.T) {
	r := newSkillResolver(t.TempDir(), t.TempDir(), t.TempDir())
	skills := r.ListSkills("coach")
	if len(skills) != 0 {
		t.Errorf("expected empty list, got %d", len(skills))
	}
}

func TestListAliases_Coach(t *testing.T) {
	r := newSkillResolver(t.TempDir(), t.TempDir(), t.TempDir())
	aliases := r.ListAliases("coach")

	if aliases["meeting"] != "meeting-notes-to-specs-skill" {
		t.Errorf("expected meeting alias, got %v", aliases)
	}
	if aliases["coordinate"] != "coordinator-skill" {
		t.Errorf("expected coordinate alias, got %v", aliases)
	}
}

func TestListAliases_UnknownAgent(t *testing.T) {
	r := newSkillResolver(t.TempDir(), t.TempDir(), t.TempDir())
	aliases := r.ListAliases("unknown")
	if len(aliases) != 0 {
		t.Errorf("expected empty aliases for unknown agent, got %v", aliases)
	}
}

func TestResolveSkill_CoordinateAlias(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "coach")
	pkgDir := filepath.Join(install, "specs", "coach-skills-system")
	writeSkillFile(t, pkgDir, "coordinator-skill.md", "# Coordinator")

	r := newSkillResolver(home, cwd, install)
	skill := r.ResolveSkill("coach", "coordinate")

	if skill == nil {
		t.Fatal("expected skill via coordinate alias")
	}
	if skill.Name != "coordinator-skill" {
		t.Errorf("expected coordinator-skill, got %s", skill.Name)
	}
}

func TestResolveSkill_EmbeddedFallback(t *testing.T) {
	efs := fstest.MapFS{
		"specs/coach-skills-system/coordinator-skill.md": &fstest.MapFile{
			Data: []byte("# Embedded Coordinator"),
		},
	}

	r := &SkillResolver{
		GlobalConfigDir: t.TempDir(),
		Cwd:             t.TempDir(),
		InstallDir:      t.TempDir(),
		EmbeddedFS:      efs,
	}

	skill := r.ResolveSkill("coach", "coordinator-skill")
	if skill == nil {
		t.Fatal("expected embedded skill")
	}
	if skill.Content != "# Embedded Coordinator" {
		t.Errorf("unexpected content: %s", skill.Content)
	}
	if skill.Source != "(embedded)" {
		t.Errorf("expected source (embedded), got %s", skill.Source)
	}
}

func TestResolveSkill_EmbeddedAliasResolution(t *testing.T) {
	efs := fstest.MapFS{
		"specs/coach-skills-system/coordinator-skill.md": &fstest.MapFile{
			Data: []byte("# Embedded Coordinator"),
		},
	}

	r := &SkillResolver{
		GlobalConfigDir: t.TempDir(),
		Cwd:             t.TempDir(),
		InstallDir:      t.TempDir(),
		EmbeddedFS:      efs,
	}

	// "coordinate" is a legacy alias for "coordinator-skill"
	skill := r.ResolveSkill("coach", "coordinate")
	if skill == nil {
		t.Fatal("expected embedded skill via alias")
	}
	if skill.Name != "coordinator-skill" {
		t.Errorf("expected coordinator-skill, got %s", skill.Name)
	}
}

func TestResolveSkill_FilesystemShadowsEmbedded(t *testing.T) {
	efs := fstest.MapFS{
		"specs/coach-skills-system/coordinator-skill.md": &fstest.MapFile{
			Data: []byte("# Embedded Version"),
		},
	}

	globalConfigDir, cwd, install := setupSkillDirs(t, "coach")
	localDir := filepath.Join(cwd, ".spekk", "skills", "coach")
	writeSkillFile(t, localDir, "coordinator-skill.md", "# Local Version")

	r := &SkillResolver{
		GlobalConfigDir: globalConfigDir,
		Cwd:             cwd,
		InstallDir:      install,
		EmbeddedFS:      efs,
	}

	skill := r.ResolveSkill("coach", "coordinator-skill")
	if skill == nil {
		t.Fatal("expected skill")
	}
	if skill.Content != "# Local Version" {
		t.Error("filesystem should shadow embedded")
	}
	if skill.Source == "(embedded)" {
		t.Error("source should not be embedded when filesystem has the skill")
	}
}

func TestListSkills_IncludesEmbedded(t *testing.T) {
	efs := fstest.MapFS{
		"specs/coach-skills-system/coordinator-skill.md": &fstest.MapFile{
			Data: []byte("# Coordinator"),
		},
		"specs/coach-skills-system/meeting-notes-to-specs-skill.md": &fstest.MapFile{
			Data: []byte("# Meeting"),
		},
	}

	r := &SkillResolver{
		GlobalConfigDir: t.TempDir(),
		Cwd:             t.TempDir(),
		InstallDir:      t.TempDir(),
		EmbeddedFS:      efs,
	}

	skills := r.ListSkills("coach")
	names := map[string]bool{}
	for _, s := range skills {
		names[s.Name] = true
	}

	if !names["coordinator-skill"] {
		t.Error("expected coordinator-skill in list")
	}
	if !names["meeting-notes-to-specs-skill"] {
		t.Error("expected meeting-notes-to-specs-skill in list")
	}
}

func TestListSkills_FilesystemShadowsEmbedded(t *testing.T) {
	efs := fstest.MapFS{
		"specs/coach-skills-system/coordinator-skill.md": &fstest.MapFile{
			Data: []byte("# Embedded"),
		},
	}

	globalConfigDir, cwd, install := setupSkillDirs(t, "coach")
	localDir := filepath.Join(cwd, ".spekk", "skills", "coach")
	writeSkillFile(t, localDir, "coordinator-skill.md", "# Local")

	r := &SkillResolver{
		GlobalConfigDir: globalConfigDir,
		Cwd:             cwd,
		InstallDir:      install,
		EmbeddedFS:      efs,
	}

	skills := r.ListSkills("coach")
	for _, s := range skills {
		if s.Name == "coordinator-skill" {
			if s.Source == "(embedded)" {
				t.Error("filesystem should shadow embedded in ListSkills")
			}
			return
		}
	}
	t.Error("expected coordinator-skill in list")
}

func TestResolveSkill_DefaultEmbeddedSkillFS(t *testing.T) {
	efs := fstest.MapFS{
		"specs/coach-skills-system/coordinator-skill.md": &fstest.MapFile{
			Data: []byte("# Default Embedded"),
		},
	}

	// Set the package-level default
	old := DefaultEmbeddedSkillFS
	DefaultEmbeddedSkillFS = efs
	defer func() { DefaultEmbeddedSkillFS = old }()

	r := &SkillResolver{
		GlobalConfigDir: t.TempDir(),
		Cwd:             t.TempDir(),
		InstallDir:      t.TempDir(),
		// EmbeddedFS not set — should fall back to DefaultEmbeddedSkillFS
	}

	skill := r.ResolveSkill("coach", "coordinator-skill")
	if skill == nil {
		t.Fatal("expected skill from DefaultEmbeddedSkillFS")
	}
	if skill.Content != "# Default Embedded" {
		t.Errorf("unexpected content: %s", skill.Content)
	}
}

// Observer coverage for the observer-skill-discovery spec
// (assertion: skill-resolver-includes-observer).

func TestSkillDirs_ObserverLayers(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "observer")
	r := newSkillResolver(home, cwd, install)

	dirs := r.skillDirs("observer")
	want := []string{
		filepath.Join(cwd, ".spekk", "skills", "observer"),
		filepath.Join(home, "skills", "observer"),
		filepath.Join(install, "specs", "observer-skills"),
	}
	if len(dirs) != len(want) {
		t.Fatalf("expected %d dirs, got %d: %v", len(want), len(dirs), dirs)
	}
	for i := range want {
		if dirs[i] != want[i] {
			t.Errorf("dir %d: expected %s, got %s", i, want[i], dirs[i])
		}
	}
}

func TestResolveSkill_ObserverLocalShadowsPackage(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "observer")
	writeSkillFile(t, filepath.Join(install, "specs", "observer-skills"), "my-scan.md", "# Package Scan")
	writeSkillFile(t, filepath.Join(cwd, ".spekk", "skills", "observer"), "my-scan.md", "# Local Scan")

	r := newSkillResolver(home, cwd, install)
	skill := r.ResolveSkill("observer", "my-scan")
	if skill == nil {
		t.Fatal("expected observer skill to resolve")
	}
	if skill.Content != "# Local Scan" {
		t.Errorf("expected local layer to shadow package, got: %s", skill.Content)
	}

	skills := r.ListSkills("observer")
	if len(skills) != 1 {
		t.Fatalf("expected 1 deduped skill, got %d: %v", len(skills), skills)
	}
	if skills[0].Source != filepath.Join(cwd, ".spekk", "skills", "observer") {
		t.Errorf("expected local source, got %s", skills[0].Source)
	}
}

func TestListAliases_ObserverNeverNil(t *testing.T) {
	home, cwd, install := setupSkillDirs(t, "observer")
	r := newSkillResolver(home, cwd, install)

	aliases := r.ListAliases("observer")
	if aliases == nil {
		t.Fatal("ListAliases(observer) must not return nil")
	}
	if aliases["coverage-gap"] != "coverage-gap-skill" {
		t.Errorf("expected coverage-gap alias to map to coverage-gap-skill, got %q", aliases["coverage-gap"])
	}
	if aliases["prune"] != "prune-skill" {
		t.Errorf("expected prune alias to map to prune-skill, got %q", aliases["prune"])
	}
}

func TestResolveSkill_ObserverEmbeddedCoverageGapAlias(t *testing.T) {
	efs := fstest.MapFS{
		"specs/observer-skills/coverage-gap-skill.md": &fstest.MapFile{
			Data: []byte("---\nid: coverage-gap\n---\n# Coverage Gap"),
		},
	}
	home, cwd, install := setupSkillDirs(t, "observer")
	r := newSkillResolver(home, cwd, install)
	r.EmbeddedFS = efs

	for _, name := range []string{"coverage-gap", "coverage-gap-skill"} {
		skill := r.ResolveSkill("observer", name)
		if skill == nil {
			t.Fatalf("expected embedded observer skill to resolve via %q", name)
		}
		if skill.Source != "(embedded)" {
			t.Errorf("%s: expected source (embedded), got %s", name, skill.Source)
		}
	}
}

// TestResolveSkill_ObserverEmbeddedPruneAlias mirrors
// TestResolveSkill_ObserverEmbeddedCoverageGapAlias for the prune skill
// (assertion: prune-skill-discoverable).
func TestResolveSkill_ObserverEmbeddedPruneAlias(t *testing.T) {
	efs := fstest.MapFS{
		"specs/observer-skills/prune-skill.md": &fstest.MapFile{
			Data: []byte("---\nid: prune\n---\n# Prune"),
		},
	}
	home, cwd, install := setupSkillDirs(t, "observer")
	r := newSkillResolver(home, cwd, install)
	r.EmbeddedFS = efs

	for _, name := range []string{"prune", "prune-skill"} {
		skill := r.ResolveSkill("observer", name)
		if skill == nil {
			t.Fatalf("expected embedded observer skill to resolve via %q", name)
		}
		if skill.Source != "(embedded)" {
			t.Errorf("%s: expected source (embedded), got %s", name, skill.Source)
		}
	}

	skills := r.ListSkills("observer")
	found := false
	for _, s := range skills {
		if s.Name == "prune-skill" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ListSkills(observer) to include prune-skill, got %v", skills)
	}
}
