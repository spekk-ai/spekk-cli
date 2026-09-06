// Package agent provides shared agent launching functionality.
package agent

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/install"
)

// CoachFlags defines the flag set for the coach CLI. The coach takes an
// optional skill as a positional argument plus the shared --harness selector.
var CoachFlags = cli.FlagSet{
	"harness": {Names: []string{"--harness"}, Type: cli.StringFlag},
	"help":    {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

// LaunchOptions configures agent launching behavior.
type LaunchOptions struct {
	Agent      string
	InstallDir string
	SkillName  string
	SkillArgs  []string
	// ExtraMessage is appended to the activation message (e.g., skill content).
	ExtraMessage string
}

// BuildActivationMessage constructs the full message for the agent,
// including optional skill activation.
func BuildActivationMessage(opts LaunchOptions) (string, error) {
	resolver := &cli.PromptResolver{
		Cwd: cwd(),
	}

	message, err := resolver.CreateActivationMessage(opts.Agent)
	if err != nil {
		return "", err
	}

	if opts.ExtraMessage != "" {
		message += opts.ExtraMessage
	}

	return message, nil
}

// BuildSkillMessage builds skill activation content to append to the base message.
func BuildSkillMessage(installDir, agent, subcommand string, args []string) (string, error) {
	sr := &cli.SkillResolver{
		Cwd:        cwd(),
		InstallDir: installDir,
	}

	skill := sr.ResolveSkill(agent, subcommand)
	if skill == nil {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("\n\n---\n\n**Skill Activation: `spekk " + agent + " " + subcommand + "`**\n\n")
	sb.WriteString("The user has launched you with a skill active via `spekk " + agent + " " + subcommand + "`.\n")
	sb.WriteString("Follow the inlined skill workflow below immediately — do not wait for trigger detection.\n")
	sb.WriteString("\n<skill-content>\n" + sanitizeSkillContent(skill.Content) + "\n</skill-content>\n")

	// Handle meeting-specific transcript argument
	if subcommand == "meeting" && len(args) > 1 {
		transcriptFile := args[1]
		resolvedPath := resolvePath(transcriptFile)
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return "", fmt.Errorf("Transcript file not found: %s", resolvedPath)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("path is not a regular file: %s", resolvedPath)
		}
		data, err := os.ReadFile(resolvedPath)
		if err != nil {
			return "", fmt.Errorf("Transcript file not found: %s", resolvedPath)
		}
		sb.WriteString(fmt.Sprintf("\nThe user provided a transcript file: %s\n", transcriptFile))
		sb.WriteString("\n<transcript>\n" + string(data) + "\n</transcript>\n")
		sb.WriteString("\nProcess this transcript now.\n")
	} else if subcommand == "meeting" {
		sb.WriteString("\nNo transcript file was provided. Ask the user to paste or provide their meeting transcript.\n")
	}

	return sb.String(), nil
}

// Launch spawns the harness binary with message as the interactive prompt and
// inherited stdio. It forwards SIGINT and preserves the exit code.
func Launch(profile Profile, message string) error {
	return spawn(profile, profile.InteractiveArgs(message))
}

// ensureInteractiveSkill installs (or refreshes) the coach/builder skill for a
// skill-delivery harness before its interactive session opens, so the session
// is governed by an installed skill rather than an inline prompt. target is the
// harness's install target; an empty target (claude-code takes an inline system
// prompt) is a no-op. The install is idempotent — an up-to-date skill is not
// rewritten — and writes only the one role skill, no agent shim.
func ensureInteractiveSkill(target, role string) error {
	if target == "" {
		return nil
	}
	_, err := install.EnsureRoleSkill(install.Options{Target: target}, role)
	return err
}

// runInteractivePlan ensures the skill (for a skill-delivery harness) and then
// launches, in that order: a skill-governed session must not open before its
// skill is on disk. ensure and launch are injected so the ordering is testable
// without spawning a process. For an inline-prompt harness (empty InstallTarget)
// ensure is skipped and only launch runs.
func runInteractivePlan(plan InteractivePlan, role string, ensure func(target, role string) error, launch func(argv []string) error) error {
	if plan.InstallTarget != "" {
		if err := ensure(plan.InstallTarget, role); err != nil {
			return fmt.Errorf("ensuring the spekk-%s skill for %s: %w", role, plan.InstallTarget, err)
		}
	}
	return launch(plan.Argv)
}

// LaunchInteractive opens an interactive coach or builder session under profile.
// role is "coach" or "builder". inlineArgv is the argv used only for a harness
// that accepts an inline system prompt (claude-code) — the caller builds it from
// the full agent prompt (InteractiveArgs or SystemPromptArgs). Every other
// harness executes any message it receives, so the full prompt is never passed:
// its role skill is ensured installed and the session opens governed by that
// skill, seeded only with a short activation message.
func LaunchInteractive(profile Profile, role string, inlineArgv []string) error {
	plan := profile.InteractiveLaunch(role, SkillActivationMessage(role), inlineArgv)
	return runInteractivePlan(plan, role, ensureInteractiveSkill, func(argv []string) error {
		return spawn(profile, argv)
	})
}

// spawn runs the harness binary with the given argv and inherited stdio. It
// forwards SIGINT and preserves the exit code.
func spawn(profile Profile, argv []string) error {
	cmd := exec.Command(profile.Binary, argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			l1, l2 := profile.notFoundLines()
			fmt.Fprintln(os.Stderr, "Error: "+l1)
			fmt.Fprintln(os.Stderr, l2)
			os.Exit(1)
		}
		return fmt.Errorf("Error launching %s: %w", profile.DisplayName, err)
	}

	// Forward SIGINT to child process
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		for range sigCh {
			if cmd.Process != nil {
				cmd.Process.Signal(syscall.SIGINT)
			}
		}
	}()

	err := cmd.Wait()
	signal.Stop(sigCh)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}

// agentHelpExtras supplies agent-specific OPTIONS and EXAMPLES blocks
// that get inserted into the shared help template. Agents not listed here
// fall back to the default skill-only template.
var agentHelpExtras = map[string]struct {
	Options  string
	Examples string
}{
	"observer": {
		Options: `  --quiet                Preference for minimal output (Claude agent decides)
`,
		Examples: `  spekk observer                          # Scan recent change; file at most one observation
  spekk observer coverage-gap             # Launch observer with coverage-gap skill
  spekk observer --quiet                  # Observer with quiet preference
`,
	},
}

// ShowHelp displays agent help with available skills.
func ShowHelp(installDir, agent string) {
	fmt.Print(buildHelpText(installDir, agent))
}

// buildHelpText constructs the help string for an agent: dynamic skill listing
// (from all resolver layers, deduped) plus agent-specific options/examples
// when registered in agentHelpExtras.
func buildHelpText(installDir, agent string) string {
	sr := &cli.SkillResolver{
		Cwd:        cwd(),
		InstallDir: installDir,
	}

	skills := sr.ListSkills(agent)
	aliases := sr.ListAliases(agent)

	reverseAliases := make(map[string]string)
	for alias, stem := range aliases {
		reverseAliases[stem] = alias
	}

	var skillLines string
	if len(skills) > 0 {
		lines := make([]string, len(skills))
		for i, s := range skills {
			name := s.Name
			if alias, ok := reverseAliases[s.Name]; ok {
				name = alias
			}
			lines[i] = "  " + name
		}
		skillLines = strings.Join(lines, "\n")
	} else {
		skillLines = "  (none found)"
	}

	displayName := agent
	if agent != "" {
		displayName = strings.ToUpper(agent[:1]) + agent[1:]
	}

	extras := agentHelpExtras[agent]
	optionsBlock := extras.Options + `  --harness <name>       Agent harness: claude-code (default, alias claude),
                         opencode, hermes, codex, or gemini.
                         Overrides the ` + HarnessEnvVar + ` env var.
  --help, -h             Show this help message`

	examplesBlock := extras.Examples
	if examplesBlock == "" {
		examplesBlock = fmt.Sprintf(`  spekk %s                          # Launch interactive %s
  spekk %s meeting                  # Launch %s with meeting skill active
  spekk %s meeting notes.txt        # Process a transcript file
`, agent, agent, agent, agent, agent)
	}

	return fmt.Sprintf(`
spekk %s - Launch the %s Agent

USAGE:
  spekk %s [SKILL] [OPTIONS]

AVAILABLE SKILLS:
%s

OPTIONS:
%s

EXAMPLES:
%s`, agent, displayName, agent, skillLines, optionsBlock, examplesBlock)
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func cwd() string {
	d, _ := os.Getwd()
	return d
}

func resolvePath(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		home := homeDir()
		if home != "" {
			p = filepath.Join(home, p[1:])
		}
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, p)
}

// sanitizeSkillContent strips any closing </skill-content> tags (case-insensitive)
// from skill markdown to prevent content from breaking out of the wrapper boundary.
func sanitizeSkillContent(content string) string {
	lower := strings.ToLower(content)
	var result strings.Builder
	result.Grow(len(content))
	i := 0
	for i < len(content) {
		idx := strings.Index(lower[i:], "</skill-content")
		if idx == -1 {
			result.WriteString(content[i:])
			break
		}
		result.WriteString(content[i : i+idx])
		// Find the end of this tag (closing >)
		tagEnd := strings.IndexByte(content[i+idx:], '>')
		if tagEnd == -1 {
			// No closing >, skip the rest as a partial tag
			break
		}
		// Skip the entire tag
		i = i + idx + tagEnd + 1
	}
	return result.String()
}

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "executable file not found") ||
		strings.Contains(err.Error(), "no such file or directory")
}
