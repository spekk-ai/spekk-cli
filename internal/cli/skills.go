// Package cli provides CLI support utilities for the spekk CLI, including
// skill resolution with layered directory discovery.
package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// legacyAliases maps agent names to their legacy alias maps.
// Keys are alias names (e.g. "meeting"), values are full skill filename stems.
var legacyAliases = map[string]map[string]string{
	"coach": {
		"meeting":    "meeting-notes-to-specs-skill",
		"coordinate": "coordinator-skill",
	},
	"builder": {},
}

// packageSkillDirSuffixes lists the package-level skill directory suffixes
// to try for each agent, relative to InstallDir/specs/.
// The first existing directory wins.
var packageSkillDirSuffixes = map[string][]string{
	"coach":   {"-skills-system"},
	"builder": {"-skills"},
}

// Skill represents a resolved skill with its metadata and file path.
type Skill struct {
	// ID is the frontmatter id field, if present.
	ID string
	// Name is the filename stem (without .md extension).
	Name string
	// File is the absolute path to the skill markdown file.
	File string
	// Source is the directory the skill was found in.
	Source string
}

// SkillResolver discovers and loads skill files using layered resolution:
// local (.spekk/skills/{agent}) -> global (~/.spekk/skills/{agent}) -> package (specs/).
type SkillResolver struct {
	// InstallDir is the root of the spekk installation (where specs/ lives).
	InstallDir string
	// WorkDir is the current working directory (where .spekk/ lives).
	WorkDir string
	// HomeDir is the user's home directory (where ~/.spekk/ lives).
	HomeDir string
}

// skillDirs returns the layered skill directories for an agent in resolution
// order (first match wins): local, global, package.
func (sr *SkillResolver) skillDirs(agent string) []string {
	dirs := []string{
		filepath.Join(sr.WorkDir, ".spekk", "skills", agent),
		filepath.Join(sr.HomeDir, ".spekk", "skills", agent),
	}

	// Package directories: try each suffix for the agent.
	suffixes, ok := packageSkillDirSuffixes[agent]
	if ok {
		for _, suffix := range suffixes {
			dirs = append(dirs, filepath.Join(sr.InstallDir, "specs", agent+suffix))
		}
	} else {
		// Fallback: try {agent}-skills-system then {agent}-skills.
		dirs = append(dirs,
			filepath.Join(sr.InstallDir, "specs", agent+"-skills-system"),
			filepath.Join(sr.InstallDir, "specs", agent+"-skills"),
		)
	}

	return dirs
}

// parseFrontmatterID extracts the "id" field from a markdown file's YAML
// frontmatter. Returns empty string if no frontmatter or no id field.
func parseFrontmatterID(content string) string {
	// Normalize CRLF.
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Must start with "---".
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}

	// Find closing "---".
	rest := content[4:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return ""
	}

	fmBlock := rest[:idx]

	// Scan for id: line.
	scanner := bufio.NewScanner(strings.NewReader(fmBlock))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "id:") {
			return strings.TrimSpace(line[3:])
		}
	}

	return ""
}

// listMDFiles returns the names of .md files in a directory.
// Returns nil if the directory doesn't exist or can't be read.
func listMDFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, e.Name())
		}
	}
	return files
}

// ResolveSkill resolves a skill by subcommand name for an agent.
//
// Resolution order:
//  1. Apply legacy alias (if any) to get the resolved name.
//  2. For each directory layer (local, global, package):
//     a. Direct filename match: {resolvedName}.md
//     b. If alias was applied, try original {subcommand}.md
//     c. Scan all .md files for frontmatter id match.
//
// Returns nil if no skill is found.
func (sr *SkillResolver) ResolveSkill(agent, name string) (*Skill, error) {
	if name == "" {
		return nil, nil
	}

	// Resolve legacy alias.
	aliases := sr.ListAliases(agent)
	resolvedName := name
	if alias, ok := aliases[name]; ok {
		resolvedName = alias
	}

	dirs := sr.skillDirs(agent)

	for _, dir := range dirs {
		// Check if directory exists.
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		// Direct filename match: {resolvedName}.md
		directPath := filepath.Join(dir, resolvedName+".md")
		if _, err := os.Stat(directPath); err == nil {
			content, err := os.ReadFile(directPath)
			if err != nil {
				return nil, err
			}
			return &Skill{
				ID:     parseFrontmatterID(string(content)),
				Name:   resolvedName,
				File:   directPath,
				Source: dir,
			}, nil
		}

		// If alias was applied, also try original subcommand name.
		if name != resolvedName {
			aliasPath := filepath.Join(dir, name+".md")
			if _, err := os.Stat(aliasPath); err == nil {
				content, err := os.ReadFile(aliasPath)
				if err != nil {
					return nil, err
				}
				return &Skill{
					ID:     parseFrontmatterID(string(content)),
					Name:   name,
					File:   aliasPath,
					Source: dir,
				}, nil
			}
		}

		// Scan directory for frontmatter id match.
		files := listMDFiles(dir)
		for _, file := range files {
			filePath := filepath.Join(dir, file)
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			fmID := parseFrontmatterID(string(content))
			if fmID == resolvedName || fmID == name {
				stem := strings.TrimSuffix(file, ".md")
				return &Skill{
					ID:     fmID,
					Name:   stem,
					File:   filePath,
					Source: dir,
				}, nil
			}
		}
	}

	return nil, nil
}

// ListSkills returns all available skills for an agent, with deduplication.
// Local skills shadow global skills, which shadow package skills.
// Skills are identified by filename stem (without .md).
func (sr *SkillResolver) ListSkills(agent string) ([]Skill, error) {
	seen := make(map[string]bool)
	var skills []Skill

	dirs := sr.skillDirs(agent)

	for _, dir := range dirs {
		files := listMDFiles(dir)
		if files == nil {
			continue
		}

		for _, file := range files {
			stem := strings.TrimSuffix(file, ".md")
			if seen[stem] {
				continue
			}
			seen[stem] = true
			skills = append(skills, Skill{
				Name:   stem,
				File:   filepath.Join(dir, file),
				Source: dir,
			})
		}
	}

	return skills, nil
}

// ListAliases returns the legacy alias map for an agent.
// Keys are alias names (e.g. "meeting"), values are full skill filename stems.
func (sr *SkillResolver) ListAliases(agent string) map[string]string {
	aliases, ok := legacyAliases[agent]
	if !ok {
		return map[string]string{}
	}
	// Return a copy to prevent mutation.
	result := make(map[string]string, len(aliases))
	for k, v := range aliases {
		result[k] = v
	}
	return result
}
