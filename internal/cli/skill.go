package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/config"
	"github.com/spekk-ai/spekk-cli/internal/fsutil"
)

// legacyAliases maps agent → subcommand → skill filename stem.
var legacyAliases = map[string]map[string]string{
	"coach": {
		"meeting":        "meeting-notes-to-specs-skill",
		"coordinate":     "coordinator-skill",
		"validate":       "business-model-validator-skill",
		"property-tests": "property-tests-skill",
	},
	"builder": {
		"review": "review-skill",
	},
	"observer": {
		"coverage-gap": "coverage-gap-skill",
		"prune":        "prune-skill",
	},
}

// packageSkillDirNames maps agent → relative directory under installDir.
var packageSkillDirNames = map[string]string{
	"coach":    "specs/coach-skills-system",
	"builder":  "specs/builder-skills",
	"observer": "specs/observer-skills",
}

// DefaultEmbeddedSkillFS is the embedded filesystem containing built-in skill files.
// Set this from main() before any skill resolution occurs.
var DefaultEmbeddedSkillFS fs.FS

// SkillResolver discovers and loads skill files using layered resolution.
type SkillResolver struct {
	// GlobalConfigDir overrides the global config directory used for skill
	// resolution. When empty, config.GlobalConfigDir() is called (production).
	// Set this in tests to inject a temp directory without triggering migration.
	GlobalConfigDir string
	Cwd             string
	InstallDir      string
	EmbeddedFS      fs.FS // override for testing; falls back to DefaultEmbeddedSkillFS
}

// NewSkillResolver creates a resolver with default paths.
func NewSkillResolver(installDir string) *SkillResolver {
	cwd, _ := os.Getwd()
	return &SkillResolver{
		Cwd:        cwd,
		InstallDir: installDir,
	}
}

func (r *SkillResolver) globalDir() (string, error) {
	if r.GlobalConfigDir != "" {
		return r.GlobalConfigDir, nil
	}
	return config.GlobalConfigDir()
}

// Skill represents a resolved skill.
type Skill struct {
	Name    string
	Content string
	Source  string
}

// skillDirs returns the layered skill directories for an agent (local → global → package).
func (r *SkillResolver) skillDirs(agent string) []string {
	dirs := []string{filepath.Join(r.Cwd, ".spekk", "skills", agent)}
	if globalConfigDir, err := r.globalDir(); err == nil {
		dirs = append(dirs, filepath.Join(globalConfigDir, "skills", agent))
	}
	if relDir, ok := packageSkillDirNames[agent]; ok {
		dirs = append(dirs, filepath.Join(r.InstallDir, relDir))
	}
	return dirs
}

// embeddedFS returns the embedded filesystem to use, preferring the
// instance field over the package-level default.
func (r *SkillResolver) embeddedFS() fs.FS {
	if r.EmbeddedFS != nil {
		return r.EmbeddedFS
	}
	return DefaultEmbeddedSkillFS
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
		if !fsutil.DirExists(dir) {
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

	// Fallback: check embedded FS
	if skill := r.resolveFromEmbedded(agent, resolvedName, subcommand); skill != nil {
		return skill
	}

	return nil
}

// resolveFromEmbedded searches the embedded filesystem for a skill.
func (r *SkillResolver) resolveFromEmbedded(agent, resolvedName, subcommand string) *Skill {
	efs := r.embeddedFS()
	if efs == nil {
		return nil
	}
	relDir, ok := packageSkillDirNames[agent]
	if !ok {
		return nil
	}

	// Direct filename match
	path := relDir + "/" + resolvedName + ".md"
	if data, err := fs.ReadFile(efs, path); err == nil {
		return &Skill{Name: resolvedName, Content: string(data), Source: "(embedded)"}
	}

	// Try original subcommand name if alias resolved
	if subcommand != resolvedName {
		path = relDir + "/" + subcommand + ".md"
		if data, err := fs.ReadFile(efs, path); err == nil {
			return &Skill{Name: subcommand, Content: string(data), Source: "(embedded)"}
		}
	}

	// Scan embedded dir for frontmatter id match
	entries, err := fs.ReadDir(efs, relDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := fs.ReadFile(efs, relDir+"/"+entry.Name())
		if err != nil {
			continue
		}
		content := string(data)
		fmID := parseFrontmatterID(content)
		if fmID == resolvedName || fmID == subcommand {
			stem := strings.TrimSuffix(entry.Name(), ".md")
			return &Skill{Name: stem, Content: content, Source: "(embedded)"}
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
			// Skip the spec doc that shares its directory's name
			// (e.g. specs/coach-skills-system/coach-skills-system.md)
			if stem == filepath.Base(dir) {
				continue
			}
			if seen[stem] {
				continue
			}
			seen[stem] = true
			skills = append(skills, SkillEntry{Name: stem, Source: dir})
		}
	}

	// Fallback: merge skills from embedded FS (filesystem still shadows)
	efs := r.embeddedFS()
	if efs != nil {
		if relDir, ok := packageSkillDirNames[agent]; ok {
			entries, err := fs.ReadDir(efs, relDir)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
						continue
					}
					stem := strings.TrimSuffix(entry.Name(), ".md")
					if seen[stem] {
						continue
					}
					seen[stem] = true
					skills = append(skills, SkillEntry{Name: stem, Source: "(embedded)"})
				}
			}
		}
	}

	return skills
}
