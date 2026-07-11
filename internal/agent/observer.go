package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// ObserverFlags defines the flag set for the observer CLI.
var ObserverFlags = cli.FlagSet{
	"interval":   {Names: []string{"--interval"}, Type: cli.StringFlag},
	"quiet":      {Names: []string{"--quiet"}, Type: cli.BoolFlag},
	"headless":   {Names: []string{"--headless"}, Type: cli.BoolFlag},
	"claudePath": {Names: []string{"--claude-path"}, Type: cli.StringFlag},
	"help":       {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

// ObserverConfig holds parsed observer options.
type ObserverConfig struct {
	Interval   int
	Quiet      bool
	Headless   bool
	ClaudePath string
	InstallDir string
}

// ParseObserverFlags parses args into an ObserverConfig.
func ParseObserverFlags(args []string) (ObserverConfig, error) {
	parsed := cli.ParseFlags(args, ObserverFlags)
	cfg := ObserverConfig{
		Quiet:      parsed.Bool("quiet"),
		Headless:   parsed.Bool("headless"),
		ClaudePath: parsed.String("claudePath"),
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
	// Operational subcommands (Go-native, do not launch Claude Code).
	// Checked before hasHelp so "install-cron --help" routes to the
	// subcommand's own help text rather than the observer-level help.
	if len(args) > 0 {
		switch args[0] {
		case "install-cron":
			RunObserverInstallCron(args[1:])
			return
		case "uninstall-cron":
			RunObserverUninstallCron(args[1:])
			return
		}
	}

	// Handle help
	if hasHelp(args) {
		ShowHelp(installDir, "observer")
		return
	}

	// Skill subcommand: check the first positional arg against the observer skill resolver
	// before parsing flags as monitoring options.
	skillName := ExtractSkillArgFromFlagSet(args, ObserverFlags)
	if skillName != "" {
		sr := &cli.SkillResolver{
			Cwd:        cwd(),
			InstallDir: installDir,
		}
		if sr.ResolveSkill("observer", skillName) != nil {
			fmt.Println("Launching Observer Agent with skill:", skillName)
			wd, _ := os.Getwd()
			fmt.Println("Working directory:", wd)
			fmt.Println()

			// Flags still apply in skill mode (`spekk observer <skill> [flags]`),
			// so parse them and pass the preferences along with the skill content.
			cfg, err := ParseObserverFlags(args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}

			skillMsg, err := BuildSkillMessage(installDir, "observer", skillName, args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}

			opts := LaunchOptions{
				Agent:        "observer",
				InstallDir:   installDir,
				ExtraMessage: skillMsg + BuildObserverOptionsMessage(cfg),
			}
			message, err := BuildActivationMessage(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}
			if cfg.Headless {
				wd, _ := os.Getwd()
				lockFile := filepath.Join(wd, ".spekk", "observer-loop.lock")
				if skillName == "consolidate" {
					lockFile = filepath.Join(wd, ".spekk", "observer-consolidate.lock")
				}
				if err := LaunchHeadless(cfg.ClaudePath, lockFile, message); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %s\n", err)
					os.Exit(1)
				}
			} else {
				if err := Launch(message); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %s\n", err)
					os.Exit(1)
				}
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

	if !cfg.Headless {
		fmt.Println("Launching Observer Agent with Claude Code...")
		wd, _ := os.Getwd()
		fmt.Println("Working directory:", wd)
		fmt.Println("Press Ctrl+C to exit the observation session.")
		fmt.Println()
	}

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

	if cfg.Headless {
		wd, _ := os.Getwd()
		lockFile := filepath.Join(wd, ".spekk", "observer-loop.lock")
		if err := LaunchHeadless(cfg.ClaudePath, lockFile, message); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	} else {
		if err := Launch(message); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	}
}
