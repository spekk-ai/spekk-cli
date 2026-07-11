package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/config"
)

// UninstallOptions holds parsed `spekk uninstall` arguments.
type UninstallOptions struct {
	Agent string
	Skill string
	Scope Scope
	Help  bool
}

var uninstallFlags = cli.FlagSet{
	"global": {Names: []string{"--global"}, Type: cli.BoolFlag},
	"local":  {Names: []string{"--local"}, Type: cli.BoolFlag},
	"help":   {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

var uninstallFlagNames = map[string]bool{
	"--global": true,
	"--local":  true,
	"--help":   true,
	"-h":       true,
}

// ParseUninstallArgs parses `spekk uninstall <agent> <skill> [--global|--local]`.
func ParseUninstallArgs(args []string) (*UninstallOptions, error) {
	parsed := cli.ParseFlags(args, uninstallFlags)

	opts := &UninstallOptions{
		Help: parsed.Bool("help"),
	}

	global := parsed.Bool("global")
	local := parsed.Bool("local")
	if global && local {
		return nil, errors.New("--global and --local are mutually exclusive")
	}
	if global {
		opts.Scope = ScopeGlobal
	} else {
		opts.Scope = ScopeLocal
	}

	if opts.Help {
		return opts, nil
	}

	positionals := extractUninstallPositionals(args)

	if len(positionals) == 0 {
		return nil, errors.New("missing <agent> argument")
	}
	opts.Agent = positionals[0]
	if !isValidAgent(opts.Agent) {
		return nil, unknownAgentError(opts.Agent)
	}

	if len(positionals) < 2 {
		return nil, errors.New("missing <skill> argument")
	}
	opts.Skill = positionals[1]

	return opts, nil
}

func extractUninstallPositionals(args []string) []string {
	var positionals []string
	for _, a := range args {
		if uninstallFlagNames[a] {
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		positionals = append(positionals, a)
	}
	return positionals
}

// UninstallUsageText is the help text printed for `spekk uninstall --help`.
const UninstallUsageText = `spekk uninstall - Remove an installed skill

USAGE:
  spekk uninstall <agent> <skill> [--global|--local]

ARGUMENTS:
  <agent>   One of: coach, builder, observer
  <skill>   Name of the skill to remove

OPTIONS:
  --global    Remove from ~/.spekk/skills/<agent>/ (user-wide)
  --local     Remove from <cwd>/.spekk/skills/<agent>/ (default)
  --help, -h  Show this help message

EXAMPLES:
  spekk uninstall coach meeting-notes
  spekk uninstall builder my-skill --global
`

// Uninstall deletes the skill file at the scope-resolved destination for
// (agent, skill). Returns the absolute path that was removed (for the
// confirmation message) on success, or an error.
//
// Errors:
//   - The file does not exist: returns a "not installed: <path>" error so
//     callers can exit non-zero and scripts can detect the case.
//   - The resolved destination escapes the scope's `.spekk/skills/<agent>/`
//     directory: returns an error and the filesystem is not touched. This
//     is a defense against a future caller passing an agent/skill name
//     containing path-traversal segments.
func Uninstall(cwd, home string, scope Scope, agent, skill string) (string, error) {
	dest, err := Destination(cwd, home, scope, agent, skill)
	if err != nil {
		return "", err
	}

	scopeRoot, err := scopeDir(cwd, home, scope, agent)
	if err != nil {
		return "", err
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("resolve dest path: %w", err)
	}
	absScope, err := filepath.Abs(scopeRoot)
	if err != nil {
		return "", fmt.Errorf("resolve scope path: %w", err)
	}

	if !strings.HasPrefix(absDest+string(os.PathSeparator), absScope+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing to uninstall outside scope directory %s", absScope)
	}

	if _, err := os.Stat(absDest); os.IsNotExist(err) {
		return "", fmt.Errorf("not installed: %s", absDest)
	} else if err != nil {
		return "", fmt.Errorf("stat %s: %w", absDest, err)
	}

	if err := os.Remove(absDest); err != nil {
		return "", fmt.Errorf("remove %s: %w", absDest, err)
	}
	return absDest, nil
}

// scopeDir returns the directory that bounds an uninstall for the given
// scope and agent — `<root>/.spekk/skills/<agent>/`. Uninstall must never
// delete files outside this directory.
func scopeDir(cwd, home string, scope Scope, agent string) (string, error) {
	if agent == "" {
		return "", fmt.Errorf("agent is required")
	}
	var root string
	switch scope {
	case ScopeGlobal:
		globalDir, err := config.GlobalConfigDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve --global scope: %w", err)
		}
		return filepath.Join(globalDir, "skills", agent), nil
	default:
		if cwd == "" {
			return "", fmt.Errorf("working directory is unknown; cannot resolve --local scope")
		}
		root = cwd
	}
	return filepath.Join(root, ".spekk", "skills", agent), nil
}
