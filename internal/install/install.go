// Package install writes thin shim subagent files into host coding
// assistants (Claude Code, OpenCode, Codex, ...). Shims contain only host
// frontmatter and an instruction to fetch the real prompt via
// `spekk prompt <agent>`, so installed agents always match the binary.
package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
}

// target describes where shims go and how their frontmatter is rendered
// for one host tool.
type target struct {
	globalDir   func(home string) string
	projectDir  string // empty means --project is unsupported
	fileExt     string // defaults to ".md"
	frontmatter func(agent string) string
}

var targets = map[string]target{
	"claude-code": {
		globalDir:  func(home string) string { return filepath.Join(home, ".claude", "agents") },
		projectDir: filepath.Join(".claude", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
	},
	"opencode": {
		globalDir:  func(home string) string { return filepath.Join(home, ".config", "opencode", "agents") },
		projectDir: filepath.Join(".opencode", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\ndescription: %q\nmode: subagent\n---\n", descriptions[agent])
		},
	},
	"codex": {
		globalDir:  func(home string) string { return filepath.Join(home, ".codex", "prompts") },
		projectDir: "",
		frontmatter: func(agent string) string {
			return ""
		},
	},
	"copilot": {
		globalDir:  func(home string) string { return filepath.Join(home, ".copilot", "agents") },
		projectDir: filepath.Join(".github", "agents"),
		fileExt:    ".agent.md",
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
	},
	"cursor": {
		globalDir:  func(home string) string { return filepath.Join(home, ".cursor", "agents") },
		projectDir: filepath.Join(".cursor", "agents"),
		frontmatter: func(agent string) string {
			return fmt.Sprintf("---\nname: spekk-%s\ndescription: %q\n---\n", agent, descriptions[agent])
		},
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
briefly, mention that `+"`spekk`"+` can set one up, and otherwise stand down.
`, display, agent, agent)
}

// Install writes shim files for all spekk agents into the host tool's agent
// directory and returns the written paths.
func Install(opts Options) ([]string, error) {
	name := opts.Target
	if canonical, ok := targetAliases[name]; ok {
		name = canonical
	}
	t, ok := targets[name]
	if !ok {
		return nil, fmt.Errorf("unknown target %q: valid targets are %s\nFor other tools, use \"spekk prompt <agent>\" directly — see \"spekk install --help\"", opts.Target, strings.Join(ValidTargets(), ", "))
	}

	var dir string
	if opts.Project {
		if t.projectDir == "" {
			return nil, fmt.Errorf("target %q does not support --project installs; omit --project to install globally", name)
		}
		cwd := opts.Cwd
		if cwd == "" {
			var err error
			if cwd, err = os.Getwd(); err != nil {
				return nil, fmt.Errorf("determining working directory: %w", err)
			}
		}
		dir = filepath.Join(cwd, t.projectDir)
	} else {
		home := opts.HomeDir
		if home == "" {
			var err error
			if home, err = os.UserHomeDir(); err != nil {
				return nil, fmt.Errorf("determining home directory: %w", err)
			}
		}
		dir = t.globalDir(home)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating %s: %w", dir, err)
	}

	ext := t.fileExt
	if ext == "" {
		ext = ".md"
	}

	var written []string
	for _, agent := range agents {
		path := filepath.Join(dir, "spekk-"+agent+ext)
		content := t.frontmatter(agent) + shimBody(agent)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", path, err)
		}
		written = append(written, path)
	}
	return written, nil
}
