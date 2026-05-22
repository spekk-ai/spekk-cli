package agent

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// ObserverFlags defines the flag set for the observer CLI.
var ObserverFlags = cli.FlagSet{
	"interval": {Names: []string{"--interval"}, Type: cli.StringFlag},
	"quiet":    {Names: []string{"--quiet"}, Type: cli.BoolFlag},
	"help":     {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

// ObserverConfig holds parsed observer options.
type ObserverConfig struct {
	Interval   int
	Quiet      bool
	InstallDir string
}

// ParseObserverFlags parses args into an ObserverConfig.
func ParseObserverFlags(args []string) (ObserverConfig, error) {
	parsed := cli.ParseFlags(args, ObserverFlags)
	cfg := ObserverConfig{
		Quiet: parsed.Bool("quiet"),
	}

	if v := parsed.String("interval"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("--interval must be a positive number")
		}
		cfg.Interval = n
	}

	return cfg, nil
}

// BuildObserverOptionsMessage builds the CLI options suffix for the activation message.
func BuildObserverOptionsMessage(cfg ObserverConfig) string {
	var parts []string

	if cfg.Interval > 0 {
		parts = append(parts, fmt.Sprintf("- Scan interval: %d seconds", cfg.Interval))
	}
	if cfg.Quiet {
		parts = append(parts, "- Quiet mode: enabled")
	}

	if len(parts) == 0 {
		return ""
	}

	return "\n\nCLI Options provided:\n" + strings.Join(parts, "\n") +
		"\n\nYou can use these preferences in your monitoring approach."
}

// RunObserver is the main entry point for the observer agent.
func RunObserver(args []string, installDir string) {
	// Handle help
	if hasHelp(args) {
		showObserverHelp()
		return
	}

	// Skill subcommand: check the first positional arg against the observer skill resolver
	// before parsing flags as monitoring options.
	skillName := ExtractSkillArgFromFlagSet(args, ObserverFlags)
	if skillName != "" {
		sr := &cli.SkillResolver{
			HomeDir:    homeDir(),
			Cwd:        cwd(),
			InstallDir: installDir,
		}
		if sr.ResolveSkill("observer", skillName) != nil {
			fmt.Println("Launching Observer Agent with skill:", skillName)
			wd, _ := os.Getwd()
			fmt.Println("Working directory:", wd)
			fmt.Println()

			skillMsg, err := BuildSkillMessage(installDir, "observer", skillName, args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}

			opts := LaunchOptions{
				Agent:        "observer",
				InstallDir:   installDir,
				ExtraMessage: skillMsg,
			}
			message, err := BuildActivationMessage(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}
			if err := Launch(message); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}
			return
		}
	}

	cfg, err := ParseObserverFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	cfg.InstallDir = installDir

	fmt.Println("Launching Observer Agent with Claude Code...")
	wd, _ := os.Getwd()
	fmt.Println("Working directory:", wd)
	fmt.Println("Press Ctrl+C to exit the observation session.")
	fmt.Println()

	opts := LaunchOptions{
		Agent:        "observer",
		InstallDir:   installDir,
		ExtraMessage: BuildObserverOptionsMessage(cfg),
	}

	message, err := BuildActivationMessage(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if err := Launch(message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func showObserverHelp() {
	fmt.Print(`
spekk observer - Intelligent spec-code drift monitoring with Claude

USAGE:
  spekk observer [OPTIONS]

OPTIONS:
  --interval <seconds>   Preferred scan interval (Claude agent can adjust)
  --quiet                Preference for minimal output (Claude agent decides)
  --help, -h             Show this help message

EXAMPLES:
  spekk observer                    # Launch Claude agent observer
  spekk observer --interval 60      # Observer with 60s interval preference
  spekk observer --quiet            # Observer with quiet preference
`)
}
