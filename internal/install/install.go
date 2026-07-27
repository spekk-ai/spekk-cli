// Package install writes thin shim subagent files into host coding
// assistants (Claude Code, OpenCode, Codex, ...). Shims contain only host
// frontmatter and an instruction to fetch the real prompt via
// `spekk prompt <agent>`, so installed agents always match the binary.
package install

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// skillEmbedPath is the path inside the skill FS where the bundled
// spekk-dev-loop skill lives.
const skillEmbedPath = "specs/install-spekk-dev-loop-skill/spekk-dev-loop-skill.md"

// DefaultSkillFS is the FS used to read the bundled spekk-dev-loop skill.
// Set in main() to spekk.EmbeddedFS; may be overridden per-call via
// Options.SkillFS. internal/install must not depend on internal/cli, so this
// is a separate var from cli.DefaultEmbeddedSkillFS.
var DefaultSkillFS fs.FS

// descriptions scope host-tool auto-delegation. They deliberately mention
// the specs/ directory so the agents stay dormant in projects that haven't
// opted into spekk.
var descriptions = map[string]string{
	"coach": "Spec-driven development coach for spekk projects. Use when the user wants to " +
		"turn an idea, feature request, or bug report into a spec, or to refine and " +
		"prioritize specs and assertions, in a project containing a specs/ directory. " +
		"Stay dormant in projects without one.",
	"builder": "Spec-driven development builder for spekk projects. Use when the user asks " +
		"to implement assertions or make specs true in a project containing a specs/ " +
		"directory (works through `spekk next`). Stay dormant in projects without one.",
	"observer": "Spec-code drift observer for spekk projects. Use when the user wants to " +
		"audit whether the codebase still satisfies its specs in a project containing " +
		"a specs/ directory. Stay dormant in projects without one.",
}

// Options configures an install run.
type Options struct {
	Target  string // claude-code|claude|opencode|codex
	Project bool   // install into the project instead of globally
	HomeDir string // defaults to os.UserHomeDir()
	Cwd     string // defaults to os.Getwd()
	SkillFS fs.FS  // FS to read the bundled skill from; falls back to DefaultSkillFS
}

// target describes where the observer agent shim and the skills go for one host
// tool.
type target struct {
	globalDir   func(home string) string
	projectDir  string                     // empty means --project is unsupported
	fileExt     string                     // defaults to ".md"
	frontmatter func(agent string) string  // frontmatter for the observer agent shim

	// Skill destinations. Each function returns the path for a named skill
	// (spekk-coach, spekk-builder, or spekk-dev-loop) in one scope. A nil
	// function means this target writes no skill for that scope. strip removes
	// the YAML frontmatter for a command or prompt host, and keeps it for a
	// native-skill host.
	skillGlobalPath  func(home, name string) string
	skillProjectPath func(cwd, name string) string
	strip            bool
}

var targets = map[string]target{
	"claude-code": {
		globalDir:  func(home string) string { return filepath.Join(home, ".claude", "agents") },
		projectDir: filepath.Join(".claude", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
		skillGlobalPath: func(home, name string) string {
			return filepath.Join(home, ".claude", "skills", name, "SKILL.md")
		},
		skillProjectPath: func(cwd, name string) string {
			return filepath.Join(cwd, ".claude", "skills", name, "SKILL.md")
		},
		strip: false,
	},
	"opencode": {
		globalDir:  func(home string) string { return filepath.Join(home, ".config", "opencode", "agents") },
		projectDir: filepath.Join(".opencode", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\ndescription: %q\nmode: subagent\n---\n", descriptions[agent])
		},
		skillGlobalPath: func(home, name string) string {
			return filepath.Join(home, ".config", "opencode", "skills", name, "SKILL.md")
		},
		skillProjectPath: func(cwd, name string) string {
			return filepath.Join(cwd, ".opencode", "skills", name, "SKILL.md")
		},
		strip: false,
	},
	"codex": {
		globalDir:  func(home string) string { return filepath.Join(home, ".codex", "prompts") },
		projectDir: "",
		frontmatter: func(agent string) string {
			return ""
		},
		skillGlobalPath: func(home, name string) string {
			return filepath.Join(home, ".codex", "prompts", name+".md")
		},
		// No skillProjectPath: codex has no --project support (projectDir is "").
		strip: true,
	},
	"copilot": {
		globalDir:  func(home string) string { return filepath.Join(home, ".copilot", "agents") },
		projectDir: filepath.Join(".github", "agents"),
		fileExt:    ".agent.md",
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
		// No skillGlobalPath: copilot has no standard global path for a personal
		// prompt file, so a global install writes no skill file.
		skillProjectPath: func(cwd, name string) string {
			return filepath.Join(cwd, ".github", "prompts", name+".prompt.md")
		},
		strip: true,
	},
	"cursor": {
		globalDir:  func(home string) string { return filepath.Join(home, ".cursor", "agents") },
		projectDir: filepath.Join(".cursor", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
		skillGlobalPath: func(home, name string) string {
			return filepath.Join(home, ".cursor", "commands", name+".md")
		},
		skillProjectPath: func(cwd, name string) string {
			return filepath.Join(cwd, ".cursor", "commands", name+".md")
		},
		strip: true,
	},
}

// skillNames lists the skills that spekk install writes. The coach and the
// builder are thin skills; the dev-loop is the bundled skill.
var skillNames = []string{"spekk-coach", "spekk-builder", "spekk-dev-loop"}

// targetAliases maps alternate names to canonical target keys.
var targetAliases = map[string]string{
	"claude": "claude-code",
}

// ValidTargets returns the sorted list of canonical target names.
func ValidTargets() []string {
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// shimBody returns the shared shim body for an agent.
func shimBody(agent string) string {
	display := strings.ToUpper(agent[:1]) + agent[1:]
	return fmt.Sprintf(`You are the spekk %s agent (%s).

Run `+"`spekk prompt %s`"+` with your shell tool and adopt its output as your
operating instructions for this session — treat those instructions as if
they were written here.

If the `+"`spekk`"+` command is not found, tell the user to install spekk
(https://github.com/spekk-ai/spekk-cli) and stop — do not attempt the
workflow without it.

If the project has no specs/ directory, it does not use spekk: say so
briefly, mention that `+"`spekk init`"+` can set one up, and otherwise stand
down.
`, display, agent, agent)
}

// agentDir returns the directory where this target's agent shims live for the
// given scope.
func (t target) agentDir(project bool, home, cwd string) string {
	if project {
		return filepath.Join(cwd, t.projectDir)
	}
	return t.globalDir(home)
}

// skillPath returns the destination of the named skill for the given scope, or
// "" when this target and scope write no skill file.
func (t target) skillPath(project bool, home, cwd, name string) string {
	if project {
		if t.skillProjectPath != nil {
			return t.skillProjectPath(cwd, name)
		}
		return ""
	}
	if t.skillGlobalPath != nil {
		return t.skillGlobalPath(home, name)
	}
	return ""
}

// ext returns the shim file extension for this target.
func (t target) ext() string {
	if t.fileExt != "" {
		return t.fileExt
	}
	return ".md"
}

// managedDirs returns every directory a scan must read to find the files this
// target owns for the given scope: the agent directory (for the observer shim,
// and for any old coach or builder shim to prune) and the directory of every
// skill.
func (t target) managedDirs(project bool, home, cwd string) []string {
	candidates := []string{t.agentDir(project, home, cwd)}
	for _, name := range skillNames {
		if sp := t.skillPath(project, home, cwd, name); sp != "" {
			candidates = append(candidates, filepath.Dir(sp))
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, d := range candidates {
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

// desiredPaths returns the destination paths this target writes for the given
// scope: the observer agent shim and the skills. It reads no files, so a caller
// that needs only paths (CheckStale) avoids the embedded skill.
func (t target) desiredPaths(project bool, home, cwd string) []string {
	paths := []string{t.observerShimPath(project, home, cwd)}
	for _, name := range skillNames {
		if sp := t.skillPath(project, home, cwd, name); sp != "" {
			paths = append(paths, sp)
		}
	}
	return paths
}

// observerShimPath returns the path of the observer agent shim for the given
// scope.
func (t target) observerShimPath(project bool, home, cwd string) string {
	return filepath.Join(t.agentDir(project, home, cwd), "spekk-observer"+t.ext())
}

// skillFrontmatter returns the YAML frontmatter for a thin role skill. A
// native-skill host uses the name and the description; a command host strips it.
func skillFrontmatter(role string) string {
	return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", role, descriptions[role])
}

// skillContent returns the body of a thin role skill: the frontmatter plus the
// shared shim body, with the frontmatter stripped for a command host.
func (t target) skillContent(role string) []byte {
	body := []byte(skillFrontmatter(role) + shimBody(role))
	if t.strip {
		body = stripFrontmatter(body)
	}
	return body
}

// desiredFiles returns the destination path -> unstamped body for every file
// this target writes for the given scope: the observer agent shim, the thin
// coach and builder skills, and the bundled dev-loop skill.
func (t target) desiredFiles(project bool, home, cwd string, skillFS fs.FS) (map[string][]byte, error) {
	desired := map[string][]byte{}

	// The observer stays an agent shim.
	desired[t.observerShimPath(project, home, cwd)] = []byte(t.frontmatter("observer") + shimBody("observer"))

	// The coach and the builder are thin skills.
	for _, role := range []string{"coach", "builder"} {
		if sp := t.skillPath(project, home, cwd, "spekk-"+role); sp != "" {
			desired[sp] = t.skillContent(role)
		}
	}

	// The dev-loop skill comes from the embedded FS.
	if sp := t.skillPath(project, home, cwd, "spekk-dev-loop"); sp != "" {
		if skillFS == nil {
			return nil, fmt.Errorf("no skill FS available; set install.DefaultSkillFS in main or provide Options.SkillFS")
		}
		data, err := fs.ReadFile(skillFS, skillEmbedPath)
		if err != nil {
			return nil, fmt.Errorf("reading embedded skill %s: %w", skillEmbedPath, err)
		}
		if t.strip {
			data = stripFrontmatter(data)
		}
		desired[sp] = data
	}
	return desired, nil
}

// legacyAgentShimPaths returns the old agent-shim paths for the coach and the
// builder — the roles that are now skills. The install removes these to migrate
// a user from the old layout.
func (t target) legacyAgentShimPaths(project bool, home, cwd string) []string {
	dir := t.agentDir(project, home, cwd)
	ext := t.ext()
	return []string{
		filepath.Join(dir, "spekk-coach"+ext),
		filepath.Join(dir, "spekk-builder"+ext),
	}
}

// Install reconciles the managed files for one target and scope to their desired
// final state. It writes the desired files (each stamped), removes owned files
// that are no longer desired, and never clobbers a file the user changed.
func Install(opts Options) (Result, error) {
	name := opts.Target
	if canonical, ok := targetAliases[name]; ok {
		name = canonical
	}
	t, ok := targets[name]
	if !ok {
		return Result{}, fmt.Errorf("unknown target %q: valid targets are %s\nFor other tools, use \"spekk prompt <agent>\" directly — see \"spekk install --help\"", opts.Target, strings.Join(ValidTargets(), ", "))
	}

	home := opts.HomeDir
	cwd := opts.Cwd
	if opts.Project {
		if t.projectDir == "" {
			return Result{}, fmt.Errorf("target %q does not support --project installs; omit --project to install globally", name)
		}
		if cwd == "" {
			var err error
			if cwd, err = os.Getwd(); err != nil {
				return Result{}, fmt.Errorf("determining working directory: %w", err)
			}
		}
	} else {
		if home == "" {
			var err error
			if home, err = os.UserHomeDir(); err != nil {
				return Result{}, fmt.Errorf("determining home directory: %w", err)
			}
		}
	}

	skillFS := opts.SkillFS
	if skillFS == nil {
		skillFS = DefaultSkillFS
	}
	desired, err := t.desiredFiles(opts.Project, home, cwd, skillFS)
	if err != nil {
		return Result{}, err
	}

	// Migrate an unstamped legacy coach or builder agent shim first. The
	// reconciler owns and prunes a stamped shim, but it does not own an
	// unstamped file from an older version. migrateLegacy removes that file, so
	// the reconcile then writes the new layout onto a clean state.
	res, err := migrateLegacy(t.legacyAgentShimPaths(opts.Project, home, cwd), desired)
	if err != nil {
		return res, err
	}
	rec, err := reconcile(desired, t.managedDirs(opts.Project, home, cwd))
	if err != nil {
		return res, err
	}
	res.Written = append(res.Written, rec.Written...)
	res.Removed = append(res.Removed, rec.Removed...)
	res.Warnings = append(res.Warnings, rec.Warnings...)
	return res, nil
}

// stripFrontmatter removes a leading YAML frontmatter block (the opening
// "---", the fields, the closing "---", and the following blank line) from
// content destined for command/prompt harnesses, which render the whole
// file as a prompt and don't understand (or forbid) YAML frontmatter.
// Content that does not begin with "---\n" is returned unchanged.
func stripFrontmatter(data []byte) []byte {
	const opening = "---\n"
	if !bytes.HasPrefix(data, []byte(opening)) {
		return data
	}
	rest := data[len(opening):]

	const closing = "\n---\n"
	idx := bytes.Index(rest, []byte(closing))
	if idx == -1 {
		// No closing delimiter found; don't guess, return unchanged.
		return data
	}
	body := rest[idx+len(closing):]
	// Drop the single blank line that follows the closing delimiter.
	body = bytes.TrimPrefix(body, []byte("\n"))
	return body
}
