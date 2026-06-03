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
)

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
		HomeDir: homeDir(),
		Cwd:     cwd(),
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
		HomeDir:    homeDir(),
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
		wd, _ := os.Getwd()
		if wd != "" && !strings.HasPrefix(resolvedPath, wd+string(filepath.Separator)) {
			return "", fmt.Errorf("path %q resolves outside working directory", transcriptFile)
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

// Launch spawns the `claude` CLI with the given message and inherited stdio.
// It forwards SIGINT and preserves the exit code.
func Launch(message string) error {
	cmd := exec.Command("claude", "--dangerously-skip-permissions", message)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			fmt.Fprintln(os.Stderr, "Error: Claude Code CLI not found. Please install Claude Code first.")
			fmt.Fprintln(os.Stderr, "Visit: https://claude.ai/code for installation instructions.")
			os.Exit(1)
		}
		return fmt.Errorf("Error launching Claude Code: %w", err)
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
		Options: `  --interval <seconds>   Preferred scan interval (Claude agent can adjust)
  --quiet                Preference for minimal output (Claude agent decides)
`,
		Examples: `  spekk observer                          # Launch interactive observer
  spekk observer coverage-gap             # Launch observer with coverage-gap skill
  spekk observer --interval 60            # Observer with 60s interval preference
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
		HomeDir:    homeDir(),
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
	optionsBlock := extras.Options + "  --help, -h             Show this help message"

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
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	wd, _ := os.Getwd()
	return filepath.Join(wd, p)
}

// sanitizeSkillContent strips any closing </skill-content> tags (case-insensitive)
// from skill markdown to prevent content from breaking out of the wrapper boundary.
func sanitizeSkillContent(content string) string {
	// Case-insensitive match for </skill-content> with optional whitespace
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
