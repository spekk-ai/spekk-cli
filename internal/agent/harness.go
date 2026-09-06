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
	// InstallTarget selects how an interactive coach/builder session is
	// governed. An empty value means the harness can take an inline system
	// prompt (claude-code): the full agent prompt seeds the session and no skill
	// is installed. A non-empty value is the `spekk install --target` name for a
	// harness that executes any message it is handed — it cannot take an inline
	// system prompt, so spekk instead ensures the harness's spekk skill is
	// installed and opens a skill-governed session seeded only with a short
	// activation. See InteractiveLaunch.
	InstallTarget string
	// SkillPreload marks a skill-delivery harness that receives its role skill
	// as a named preload (hermes: `chat -s spekk-<role>`) rather than as an
	// activation-message positional (opencode: `run -i <activation>`). Some CLIs
	// — hermes's interactive `chat` among them — reject a bare positional
	// message, so the only way to seed the session is to name the already
	// installed skill. When set, the interactive skill-governed launch appends
	// the skill name spekk-<role> to InteractiveArgv (whose trailing token is the
	// preload flag) in place of the activation message. See InteractiveLaunch.
	SkillPreload bool
}

// InteractivePlan describes how to open an interactive coach/builder session.
type InteractivePlan struct {
	// Argv are the tokens (excluding the binary) to spawn.
	Argv []string
	// InstallTarget, when non-empty, is the `spekk install --target` name whose
	// spekk skill must be installed before the session launches. Empty means the
	// session is governed by an inline system prompt and installs nothing.
	InstallTarget string
}

// InteractiveLaunch resolves how to open an interactive coach/builder session.
//
// A harness with no install target (claude-code) accepts an inline system
// prompt: it launches with inlineArgv — the exact argv the launch site has
// always used to seed the full agent prompt — and installs nothing.
//
// Every other harness executes any message it receives, so seeding the full
// prompt inline makes it auto-run the prompt as a task (opencode starts a build;
// hermes answers once and exits). Such a harness instead reports an install
// target so the caller can ensure its spekk skill is installed, and launches a
// skill-governed interactive session seeded only with the short activation — the
// full prompt body is never passed as a message argument.
//
// role is "coach" or "builder"; it names the installed skill (spekk-<role>) for
// a SkillPreload harness, which seeds the session by naming the skill rather than
// by passing the activation message (its interactive subcommand takes no
// positional). For every other skill-delivery harness the activation message is
// the seed, exactly as before.
func (p Profile) InteractiveLaunch(role, activation string, inlineArgv []string) InteractivePlan {
	if p.InstallTarget == "" {
		return InteractivePlan{Argv: inlineArgv}
	}
	seed := activation
	if p.SkillPreload {
		seed = "spekk-" + role
	}
	return InteractivePlan{Argv: p.InteractiveArgs(seed), InstallTarget: p.InstallTarget}
}

// SkillActivationMessage is the short message that seeds a skill-governed
// interactive session: it names the role's spekk skill so the harness loads and
// follows it, and it carries none of the full prompt body. agent is "coach" or
// "builder".
func SkillActivationMessage(agent string) string {
	return fmt.Sprintf("Load and follow your `spekk-%s` skill for this session, then wait for my instructions.", agent)
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
	InstallTarget:    "opencode",
}

// hermesProfile launches the coach, builder, and observer through the Hermes
// Agent CLI (Nous Research). Its argv follows hermes's own conventions —
// verified against the installed `hermes --help` / `hermes chat --help`
// (v0.18.x) — and is deliberately not a copy of the claude/opencode flags:
//
//   - Interactive: `hermes chat -s spekk-<role>` — `chat` is hermes's
//     interactive subcommand ("Interactive chat with the agent") and
//     `-s/--skills` preloads a named, already-installed skill, so the session is
//     governed by the spekk role skill and waits for the user's input. `chat`
//     takes no positional message and its only prompt-carrying flag
//     (`-q/--query`) is non-interactive, so the skill is named rather than a
//     message passed — see SkillPreload and InteractiveLaunch. `-z/--oneshot` is
//     deliberately NOT used interactively: it "sends a single prompt and prints
//     ONLY the final response", i.e. one-shot, exiting instead of waiting.
//   - Headless: `hermes --yolo --cli -z <prompt>` — `-z/--oneshot` runs the
//     single prompt non-interactively (its whole purpose), `--cli` selects the
//     classic (non-TUI) interface, and `--yolo` "bypasses all dangerous command
//     approval prompts", hermes's equivalent of claude's
//     --dangerously-skip-permissions for a no-TTY cron run with no human to
//     confirm.
//
// hermes has no separate system-prompt flag, so the interactive builder reuses
// the interactive `chat -s` form. `--yolo` is intentionally absent from the
// interactive/system-prompt modes: a human is present to answer approval prompts
// there, exactly as opencode omits `--auto`.
var hermesProfile = Profile{
	Name:             "hermes",
	Binary:           "hermes",
	DisplayName:      "Hermes",
	InstallURL:       "https://hermes-agent.nousresearch.com/docs/",
	InteractiveArgv:  []string{"chat", "-s"},
	SystemPromptArgv: []string{"chat", "-s"},
	HeadlessArgv:     []string{"--yolo", "--cli", "-z"},
	InstallTarget:    "hermes",
	SkillPreload:     true,
}

// codexProfile launches the coach, builder, and observer through the codex CLI
// (OpenAI Codex). Its argv follows codex's own conventions — verified against
// the installed `codex --help` / `codex exec --help` (codex-cli v0.153.x) — and
// is deliberately not a copy of the claude/opencode flags:
//
//   - Interactive: bare `codex <prompt>` — codex with no subcommand forwards to
//     the interactive TUI and takes the prompt as its trailing positional
//     ("Optional user prompt to start the session"), so it seeds the prompt and
//     waits for input. There is no interactive flag to add; "bare" means bare.
//   - Headless: `codex exec --dangerously-bypass-approvals-and-sandbox <prompt>`
//     — the `exec` subcommand runs Codex non-interactively with the prompt as a
//     one-off task, and `--dangerously-bypass-approvals-and-sandbox` skips all
//     confirmation prompts, codex's equivalent of claude's
//     --dangerously-skip-permissions for a no-TTY cron run with no human to
//     confirm.
//
// codex has no separate system-prompt flag, so the interactive builder reuses
// the bare interactive form and seeds the session with the prompt. The
// permission-skip flag is intentionally absent from the interactive/system-prompt
// modes: a human is present to answer approval prompts there, exactly as
// opencode omits `--auto` and hermes omits `--yolo`.
var codexProfile = Profile{
	Name:             "codex",
	Binary:           "codex",
	DisplayName:      "Codex",
	InstallURL:       "https://github.com/openai/codex",
	InteractiveArgv:  []string{},
	SystemPromptArgv: []string{},
	HeadlessArgv:     []string{"exec", "--dangerously-bypass-approvals-and-sandbox"},
	InstallTarget:    "codex",
}

// geminiProfile launches the coach, builder, and observer through the Gemini CLI
// (Google). Its argv follows gemini's own conventions — verified against the
// installed `gemini --help` (v0.58.x) — and is deliberately not a copy of the
// claude/opencode flags:
//
//   - Interactive: `gemini -i <prompt>` — `-i/--prompt-interactive` "Execute the
//     provided prompt and continue in interactive mode", so gemini seeds the
//     prompt and stays interactive. The bare `query` positional also starts
//     interactive, but `-i` is the flag whose stated purpose is exactly "carry a
//     prompt and keep the session interactive", so the profile uses it.
//   - Headless: `gemini --yolo -p <prompt>` — `-p/--prompt` "Run in
//     non-interactive (headless) mode with the given prompt", and `-y/--yolo`
//     "Automatically accept all actions" bypasses every approval, gemini's
//     equivalent of claude's --dangerously-skip-permissions for a no-TTY cron run
//     with no human to confirm.
//
// gemini has no separate system-prompt flag, so the interactive builder reuses
// the interactive `-i` form and seeds the session with the prompt. `--yolo` is
// intentionally absent from the interactive/system-prompt modes: a human is
// present to answer approval prompts there, exactly as opencode omits `--auto`,
// hermes omits `--yolo`, and codex omits its bypass.
var geminiProfile = Profile{
	Name:             "gemini",
	Binary:           "gemini",
	DisplayName:      "Gemini CLI",
	InstallURL:       "https://github.com/google-gemini/gemini-cli",
	InteractiveArgv:  []string{"-i"},
	SystemPromptArgv: []string{"-i"},
	HeadlessArgv:     []string{"--yolo", "-p"},
	InstallTarget:    "gemini",
}

const defaultHarness = "claude-code"

// HarnessEnvVar is the environment variable that selects the harness when no
// --harness flag is given.
const HarnessEnvVar = "SPEKK_HARNESS"

// harnessProfiles maps canonical harness names to their profiles.
var harnessProfiles = map[string]Profile{
	claudeCodeProfile.Name: claudeCodeProfile,
	opencodeProfile.Name:   opencodeProfile,
	hermesProfile.Name:     hermesProfile,
	codexProfile.Name:      codexProfile,
	geminiProfile.Name:     geminiProfile,
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

// knownHarnessNames returns one entry per canonical harness, sorted, for use in
// error messages. A harness with aliases carries them as an annotation on its
// single entry (e.g. "claude-code (alias: claude)") rather than appearing again
// as a bare alias entry, so the list reads as one line per real harness.
func knownHarnessNames() []string {
	aliasesByCanonical := make(map[string][]string)
	for alias, canonical := range harnessAliases {
		aliasesByCanonical[canonical] = append(aliasesByCanonical[canonical], alias)
	}
	names := make([]string, 0, len(harnessProfiles))
	for name := range harnessProfiles {
		entry := name
		if aliases := aliasesByCanonical[name]; len(aliases) > 0 {
			sort.Strings(aliases)
			entry = fmt.Sprintf("%s (alias: %s)", name, strings.Join(aliases, ", "))
		}
		names = append(names, entry)
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
