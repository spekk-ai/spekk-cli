package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// legacyAliases maps agent → subcommand → skill filename stem.
var legacyAliases = map[string]map[string]string{
	"coach": {
		"meeting":    "meeting-notes-to-specs-skill",
		"coordinate": "coordinator-skill",
		"validate":   "business-model-validator-skill",
	},
	"builder": {},
}

// packageSkillDirNames maps agent → relative directory under installDir.
var packageSkillDirNames = map[string]string{
	"coach":   "specs/coach-skills-system",
	"builder": "specs/builder-skills",
}

// SkillResolver discovers and loads skill files using layered resolution.
type SkillResolver struct {
	HomeDir    string
	Cwd        string
	InstallDir string
}

// NewSkillResolver creates a resolver with default paths.
func NewSkillResolver(installDir string) *SkillResolver {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	return &SkillResolver{
		HomeDir:    home,
		Cwd:        cwd,
		InstallDir: installDir,
	}
}

// Skill represents a resolved skill.
type Skill struct {
	Name    string
	Content string
	Source  string
}

// skillDirs returns the layered skill directories for an agent (local → global → package).
func (r *SkillResolver) skillDirs(agent string) []string {
	dirs := []string{
		filepath.Join(r.Cwd, ".spekk", "skills", agent),
		filepath.Join(r.HomeDir, ".spekk", "skills", agent),
	}
	if relDir, ok := packageSkillDirNames[agent]; ok {
		dirs = append(dirs, filepath.Join(r.InstallDir, relDir))
	}
	return dirs
}

// parseFrontmatterID extracts the `id` field from YAML frontmatter.
var frontmatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
var idFieldRe = regexp.MustCompile(`(?m)^id:\s*(.+)$`)

func parseFrontmatterID(content string) string {
	m := frontmatterRe.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	idm := idFieldRe.FindStringSubmatch(m[1])
	if idm == nil {
		return ""
	}
	return strings.TrimSpace(idm[1])
}

// ResolveSkill finds a skill by subcommand name for an agent.
// Returns nil if not found.
func (r *SkillResolver) ResolveSkill(agent, subcommand string) *Skill {
	if subcommand == "" {
		return nil
	}

	// Resolve legacy alias
	resolvedName := subcommand
	if aliases, ok := legacyAliases[agent]; ok {
		if mapped, ok := aliases[subcommand]; ok {
			resolvedName = mapped
		}
	}

	dirs := r.skillDirs(agent)

	for _, dir := range dirs {
		if !dirExists(dir) {
			continue
		}

		// Direct filename match: <resolvedName>.md
		directPath := filepath.Join(dir, resolvedName+".md")
		if content, ok := readIfExists(directPath); ok {
			return &Skill{Name: resolvedName, Content: content, Source: dir}
		}

		// If alias resolved, also try original subcommand name
		if subcommand != resolvedName {
			aliasPath := filepath.Join(dir, subcommand+".md")
			if content, ok := readIfExists(aliasPath); ok {
				return &Skill{Name: subcommand, Content: content, Source: dir}
			}
		}

		// Scan directory for frontmatter id match
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			filePath := filepath.Join(dir, entry.Name())
			content, ok := readIfExists(filePath)
			if !ok {
				continue
			}
			fmID := parseFrontmatterID(content)
			if fmID == resolvedName || fmID == subcommand {
				stem := strings.TrimSuffix(entry.Name(), ".md")
				return &Skill{Name: stem, Content: content, Source: dir}
			}
		}
	}

	return nil
}

// ListAliases returns the legacy alias map for an agent.
func (r *SkillResolver) ListAliases(agent string) map[string]string {
	if aliases, ok := legacyAliases[agent]; ok {
		return aliases
	}
	return map[string]string{}
}

// SkillEntry is a skill name and its source directory.
type SkillEntry struct {
	Name   string
	Source string
}

// ListSkills returns all available skills for an agent with deduplication
// (local shadows global shadows package).
func (r *SkillResolver) ListSkills(agent string) []SkillEntry {
	seen := make(map[string]bool)
	var skills []SkillEntry
	dirs := r.skillDirs(agent)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			stem := strings.TrimSuffix(entry.Name(), ".md")
			if seen[stem] {
				continue
			}
			seen[stem] = true
			skills = append(skills, SkillEntry{Name: stem, Source: dir})
		}
	}

	return skills
}

// dirExists checks if a path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
