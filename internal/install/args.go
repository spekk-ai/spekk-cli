// Package install implements the `spekk install` command surface:
// fetching skill files into layered skill directories and listing
// what's available in the remote registry.
package install

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// Scope is the install destination.
type Scope int

const (
	// ScopeLocal writes to <cwd>/.spekk/skills/<agent>/.
	ScopeLocal Scope = iota
	// ScopeGlobal writes to ~/.spekk/skills/<agent>/.
	ScopeGlobal
)

// String returns a human-readable scope name.
func (s Scope) String() string {
	switch s {
	case ScopeGlobal:
		return "global"
	default:
		return "local"
	}
}

// ValidAgents are the agent names accepted by `spekk install`.
var ValidAgents = []string{"coach", "builder", "observer"}

// Options holds parsed `spekk install` arguments.
type Options struct {
	// Agent is the agent name (coach, builder, observer).
	Agent string
	// Skill is the positional skill name. Empty when --list is used or
	// when the caller intends to derive the name from a --source URL.
	Skill string
	// Scope is the install destination (default: ScopeLocal).
	Scope Scope
	// Source is an optional explicit URL to fetch the skill from.
	Source string
	// Force overwrites an existing file at the destination.
	Force bool
	// List, when set, requests the remote registry listing for the agent
	// instead of an install. The value is the agent name passed to --list.
	List string
	// Help requests usage output.
	Help bool
}

// installFlags is the FlagSet used by ParseArgs.
var installFlags = cli.FlagSet{
	"global": {Names: []string{"--global"}, Type: cli.BoolFlag},
	"local":  {Names: []string{"--local"}, Type: cli.BoolFlag},
	"source": {Names: []string{"--source"}, Type: cli.StringFlag},
	"force":  {Names: []string{"--force"}, Type: cli.BoolFlag},
	"list":   {Names: []string{"--list"}, Type: cli.StringFlag},
	"help":   {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

// flagNames is every flag token consumed by installFlags, used to skip
// flag tokens when extracting positional args.
var flagNames = map[string]bool{
	"--global": true,
	"--local":  true,
	"--source": true,
	"--force":  true,
	"--list":   true,
	"--help":   true,
	"-h":       true,
}

// ParseArgs parses the arguments passed to `spekk install`.
// Returns parsed Options on success, or an error describing the user-facing
// problem (e.g. unknown agent, mutually exclusive flags). The caller is
// expected to print the error and exit non-zero.
func ParseArgs(args []string) (*Options, error) {
	parsed := cli.ParseFlags(args, installFlags)

	opts := &Options{
		Source: parsed.String("source"),
		Force:  parsed.Bool("force"),
		List:   parsed.String("list"),
		Help:   parsed.Bool("help"),
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

	positionals := extractPositionals(args)

	// --list takes precedence over positional agent/skill — it's a
	// discovery mode that bypasses install.
	if opts.List != "" {
		if !isValidAgent(opts.List) {
			return nil, unknownAgentError(opts.List)
		}
		return opts, nil
	}

	if len(positionals) == 0 {
		return nil, errors.New("missing <agent> argument")
	}

	opts.Agent = positionals[0]
	if !isValidAgent(opts.Agent) {
		return nil, unknownAgentError(opts.Agent)
	}

	if len(positionals) > 1 {
		opts.Skill = positionals[1]
	}

	return opts, nil
}

// extractPositionals returns the non-flag arguments, skipping any value
// that immediately follows a string-typed flag.
func extractPositionals(args []string) []string {
	stringFlags := map[string]bool{
		"--source": true,
		"--list":   true,
	}
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if stringFlags[a] {
			i++ // skip value
			continue
		}
		if flagNames[a] {
			continue
		}
		if strings.HasPrefix(a, "-") {
			// Unknown flag — skip to stay consistent with ParseFlags.
			continue
		}
		positionals = append(positionals, a)
	}
	return positionals
}

func isValidAgent(name string) bool {
	for _, a := range ValidAgents {
		if a == name {
			return true
		}
	}
	return false
}

func unknownAgentError(name string) error {
	return fmt.Errorf("unknown agent %q (valid agents: %s)", name, strings.Join(ValidAgents, ", "))
}

// UsageText is the help text printed for `spekk install --help`.
const UsageText = `spekk install - Install a skill for an agent

USAGE:
  spekk install <agent> <skill> [OPTIONS]
  spekk install --list <agent>

ARGUMENTS:
  <agent>   One of: coach, builder, observer
  <skill>   Name of the skill to install (omit when using --list)

OPTIONS:
  --global         Install to ~/.spekk/skills/<agent>/ (user-wide)
  --local          Install to <cwd>/.spekk/skills/<agent>/ (default)
  --source <URL>   Fetch the skill from an arbitrary http(s) URL
  --force          Overwrite an existing file at the destination
  --list <agent>   List skills available in the remote registry for <agent>
  --help, -h       Show this help message

EXAMPLES:
  spekk install coach meeting-notes
  spekk install builder my-skill --global
  spekk install coach foo --source https://example.com/foo.md
  spekk install --list coach
`
