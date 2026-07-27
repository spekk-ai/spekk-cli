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

// agents lists the spekk agents installed as shims.
var agents = []string{"coach", "builder", "observer"}

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

// target describes where shims go and how their frontmatter is rendered
// for one host tool.
type target struct {
	globalDir   func(home string) string
	projectDir  string // empty means --project is unsupported
	fileExt     string // defaults to ".md"
	frontmatter func(agent string) string

	// Skill destination: where the bundled spekk-dev-loop skill/command
	// goes for this host tool. Either func may be nil/return "" to opt
	// that scope out of writing a dev-loop file. strip controls whether
	// the embedded skill's YAML frontmatter is removed before writing
	// (command/prompt harnesses) or written verbatim (native-skill
	// harnesses).
	globalPath  func(home string) string
	projectPath func(cwd string) string
	strip       bool
}

var targets = map[string]target{
	"claude-code": {
		globalDir:  func(home string) string { return filepath.Join(home, ".claude", "agents") },
		projectDir: filepath.Join(".claude", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
		globalPath: func(home string) string {
			return filepath.Join(home, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
		},
		projectPath: func(cwd string) string {
			return filepath.Join(cwd, ".claude", "skills", "spekk-dev-loop", "SKILL.md")
		},
		strip: false,
	},
	"opencode": {
		globalDir:  func(home string) string { return filepath.Join(home, ".config", "opencode", "agents") },
		projectDir: filepath.Join(".opencode", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\ndescription: %q\nmode: subagent\n---\n", descriptions[agent])
		},
		globalPath: func(home string) string {
			return filepath.Join(home, ".config", "opencode", "skills", "spekk-dev-loop", "SKILL.md")
		},
		projectPath: func(cwd string) string {
			return filepath.Join(cwd, ".opencode", "skills", "spekk-dev-loop", "SKILL.md")
		},
		strip: false,
	},
	"codex": {
		globalDir:  func(home string) string { return filepath.Join(home, ".codex", "prompts") },
		projectDir: "",
		frontmatter: func(agent string) string {
			return ""
		},
		globalPath: func(home string) string {
			return filepath.Join(home, ".codex", "prompts", "spekk-dev-loop.md")
		},
		// No projectPath: codex already has no --project support (projectDir
		// is "" above), so --project errors before skill logic is reached.
		strip: true,
	},
	"copilot": {
		globalDir:  func(home string) string { return filepath.Join(home, ".copilot", "agents") },
		projectDir: filepath.Join(".github", "agents"),
		fileExt:    ".agent.md",
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
		// No globalPath: copilot has no standard global filesystem path for
		// a personal prompt file, so global installs write no dev-loop file.
		projectPath: func(cwd string) string {
			return filepath.Join(cwd, ".github", "prompts", "spekk-dev-loop.prompt.md")
		},
		strip: true,
	},
	"cursor": {
		globalDir:  func(home string) string { return filepath.Join(home, ".cursor", "agents") },
		projectDir: filepath.Join(".cursor", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
		globalPath: func(home string) string {
			return filepath.Join(home, ".cursor", "commands", "spekk-dev-loop.md")
		},
		projectPath: func(cwd string) string {
			return filepath.Join(cwd, ".cursor", "commands", "spekk-dev-loop.md")
		},
		strip: true,
	},
}

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

// skillPath returns the destination of the bundled dev-loop skill for the given
// scope, or "" when this target+scope writes no skill file.
func (t target) skillPath(project bool, home, cwd string) string {
	if project {
		if t.projectPath != nil {
			return t.projectPath(cwd)
		}
		return ""
	}
	if t.globalPath != nil {
		return t.globalPath(home)
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
// target owns for the given scope.
func (t target) managedDirs(project bool, home, cwd string) []string {
	candidates := []string{t.agentDir(project, home, cwd)}
	if sp := t.skillPath(project, home, cwd); sp != "" {
		candidates = append(candidates, filepath.Dir(sp))
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
// scope. It reads no files, so a caller that needs only paths (CheckStale)
// avoids touching the embedded skill.
func (t target) desiredPaths(project bool, home, cwd string) []string {
	dir := t.agentDir(project, home, cwd)
	ext := t.ext()
	paths := make([]string, 0, len(agents)+1)
	for _, agent := range agents {
		paths = append(paths, filepath.Join(dir, "spekk-"+agent+ext))
	}
	if sp := t.skillPath(project, home, cwd); sp != "" {
		paths = append(paths, sp)
	}
	return paths
}

// desiredFiles returns the destination path -> unstamped body for every file
// this target writes for the given scope.
func (t target) desiredFiles(project bool, home, cwd string, skillFS fs.FS) (map[string][]byte, error) {
	dir := t.agentDir(project, home, cwd)
	ext := t.ext()
	desired := map[string][]byte{}
	for _, agent := range agents {
		path := filepath.Join(dir, "spekk-"+agent+ext)
		desired[path] = []byte(t.frontmatter(agent) + shimBody(agent))
	}
	if sp := t.skillPath(project, home, cwd); sp != "" {
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
	return reconcile(desired, t.managedDirs(opts.Project, home, cwd))
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
