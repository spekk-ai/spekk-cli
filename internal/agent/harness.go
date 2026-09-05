package agent

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Profile describes how to launch an agent under a particular harness (the CLI
// tool that actually runs the coach, builder, or observer). Every value that a
// launch site used to embed as a `"claude"` literal — the binary name, the
// launch argv, and the not-found guidance — lives here instead, so selecting a
// different harness is a matter of resolving a different profile.
//
// The three *Argv fields hold the tokens that precede the prompt in each launch
// mode; the prompt is always appended as the trailing positional. Modeling each
// mode as an explicit token list — rather than a set of shared flag slices —
// lets harnesses differ structurally: claude passes flags on the bare command,
// while opencode routes every launch through a `run` subcommand.
type Profile struct {
	// Name is the canonical harness name (e.g. "claude-code").
	Name string
	// Binary is the executable spawned at every launch site.
	Binary string
	// DisplayName is how the harness is named in user-facing messages
	// (e.g. "Claude Code").
	DisplayName string
	// InstallURL points at the harness's install instructions, shown when the
	// binary is missing.
	InstallURL string
	// InteractiveArgv are the tokens before the prompt positional when the
	// harness is launched interactively (coach, builder). The prompt seeds the
	// session as the first user message; the session stays interactive.
	InteractiveArgv []string
	// SystemPromptArgv are the tokens before the prompt when it seeds the
	// session as a system prompt rather than a first user message (the
	// interactive builder, which then waits for user input).
	SystemPromptArgv []string
	// HeadlessArgv are the tokens before the prompt in non-interactive
	// (no-TTY) mode, where the prompt is run as a one-off task with no human to
	// answer permission prompts.
	HeadlessArgv []string
}

// claudeCodeProfile is the built-in default: the argv and guidance that were
// hardcoded across the launch sites before harnesses became selectable.
//   - Interactive:   claude --dangerously-skip-permissions <prompt>
//   - System prompt: claude --dangerously-skip-permissions --system-prompt <prompt>
//   - Headless:      claude -p --dangerously-skip-permissions <prompt>
var claudeCodeProfile = Profile{
	Name:             "claude-code",
	Binary:           "claude",
	DisplayName:      "Claude Code",
	InstallURL:       "https://claude.ai/code",
	InteractiveArgv:  []string{"--dangerously-skip-permissions"},
	SystemPromptArgv: []string{"--dangerously-skip-permissions", "--system-prompt"},
	HeadlessArgv:     []string{"-p", "--dangerously-skip-permissions"},
}

// opencodeProfile launches the coach, builder, and observer through the
// opencode CLI. Its argv follows opencode's own conventions (verified against
// `opencode --help` / `opencode run --help`, v1.18.x) and is deliberately not a
// copy of the claude flags:
//
//   - Interactive: `opencode run -i <msg>` — the `run` subcommand carries the
//     agent prompt as a bare positional message and `-i` keeps the session
//     interactive. An earlier version passed the prompt with `--prompt` on the
//     bare `opencode` command; on binaries that do not define that flag the
//     whole prompt was silently dropped and opencode opened an empty TUI instead
//     of acting as the agent, so the launch now routes through `run`.
//   - Headless: `opencode run --auto <msg>` — `run` executes the message as a
//     one-off task and `--auto` auto-approves permissions not explicitly denied,
//     opencode's equivalent of claude's --dangerously-skip-permissions (there is
//     no human at a no-TTY cron run to answer a prompt).
//
// opencode has no separate system-prompt flag, so the interactive builder reuses
// the interactive `run -i` form and seeds the session with the prompt as its
// message. `--auto` is intentionally absent interactively: a human is present to
// answer permission prompts.
var opencodeProfile = Profile{
	Name:             "opencode",
	Binary:           "opencode",
	DisplayName:      "opencode",
	InstallURL:       "https://opencode.ai/docs/",
	InteractiveArgv:  []string{"run", "-i"},
	SystemPromptArgv: []string{"run", "-i"},
	HeadlessArgv:     []string{"run", "--auto"},
}

const defaultHarness = "claude-code"

// HarnessEnvVar is the environment variable that selects the harness when no
// --harness flag is given.
const HarnessEnvVar = "SPEKK_HARNESS"

// harnessProfiles maps canonical harness names to their profiles.
var harnessProfiles = map[string]Profile{
	claudeCodeProfile.Name: claudeCodeProfile,
	opencodeProfile.Name:   opencodeProfile,
}

// harnessAliases maps alternative names to their canonical harness name.
var harnessAliases = map[string]string{
	"claude": "claude-code",
}

// DefaultProfile returns the built-in default harness profile (claude-code).
func DefaultProfile() Profile {
	return claudeCodeProfile
}

// ResolveProfile returns the profile for a harness name. An empty name resolves
// to the default; aliases resolve to their canonical profile. An unknown name
// returns an error listing the valid names rather than a zero profile a caller
// might spawn.
func ResolveProfile(name string) (Profile, error) {
	if name == "" {
		name = defaultHarness
	}
	if canonical, ok := harnessAliases[name]; ok {
		name = canonical
	}
	p, ok := harnessProfiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown harness %q; valid harnesses are: %s", name, strings.Join(knownHarnessNames(), ", "))
	}
	return p, nil
}

// ResolveHarness resolves the active harness profile using the selection
// precedence: the --harness flag value, then the SPEKK_HARNESS environment
// variable, then the built-in default (claude-code). An unknown name from
// either source fails fast with an identical error, because both are funneled
// through ResolveProfile with the same name string.
func ResolveHarness(flag string) (Profile, error) {
	name := flag
	if name == "" {
		name = os.Getenv(HarnessEnvVar)
	}
	return ResolveProfile(name)
}

// resolveHarnessOrExit resolves the harness from the given flag value (env and
// default fill in per ResolveHarness), printing the error and exiting on an
// unknown name rather than spawning a nonexistent binary.
func resolveHarnessOrExit(flag string) Profile {
	p, err := ResolveHarness(flag)
	if err != nil {
		colorLog(colorRed, "Error: "+err.Error())
		os.Exit(1)
	}
	return p
}

// knownHarnessNames returns the canonical harness names plus aliases, sorted,
// for use in error messages.
func knownHarnessNames() []string {
	names := make([]string, 0, len(harnessProfiles)+len(harnessAliases))
	for name := range harnessProfiles {
		names = append(names, name)
	}
	for alias := range harnessAliases {
		names = append(names, alias)
	}
	sort.Strings(names)
	return names
}

// argvWithPrompt appends the prompt as the trailing positional to a copy of the
// given mode prefix.
func argvWithPrompt(prefix []string, prompt string) []string {
	return append(append([]string{}, prefix...), prompt)
}

// InteractiveArgs returns the argv (excluding the binary) to launch the harness
// interactively with prompt as the initial message.
func (p Profile) InteractiveArgs(prompt string) []string {
	return argvWithPrompt(p.InteractiveArgv, prompt)
}

// SystemPromptArgs is like InteractiveArgs but passes the prompt as a system
// prompt (used by the interactive builder so the harness waits for user input).
func (p Profile) SystemPromptArgs(prompt string) []string {
	return argvWithPrompt(p.SystemPromptArgv, prompt)
}

// HeadlessArgs returns the argv (excluding the binary) to run the harness in
// non-interactive/headless mode with prompt as the trailing positional.
func (p Profile) HeadlessArgs(prompt string) []string {
	return argvWithPrompt(p.HeadlessArgv, prompt)
}

// notFoundLines returns the two guidance lines shown when the harness binary is
// missing: the "not found — install X" line and the install URL line.
func (p Profile) notFoundLines() (string, string) {
	return fmt.Sprintf("%s CLI not found. Please install %s first.", p.DisplayName, p.DisplayName),
		fmt.Sprintf("Visit: %s for installation instructions.", p.InstallURL)
}
