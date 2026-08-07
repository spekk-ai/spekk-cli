package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// lockNameSafe collapses every character that is not safe in a single file
// name segment, so a skill name can never place the lock outside .spekk/.
var lockNameSafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// ObserverLockFile returns the flock path for a headless observer run.
// Each skill locks its own file (`.spekk/observer-<skill>.lock`), so a
// scheduled skill run does not exit only because the default loop is
// active. The default loop (empty skillName) keeps
// `.spekk/observer-loop.lock`, and `consolidate` keeps
// `.spekk/observer-consolidate.lock` under the same rule.
func ObserverLockFile(wd, skillName string) string {
	name := "observer-loop.lock"
	if skillName != "" {
		name = "observer-" + lockNameSafe.ReplaceAllString(skillName, "-") + ".lock"
	}
	return filepath.Join(wd, ".spekk", name)
}

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
		if skill := sr.ResolveSkill("observer", skillName); skill != nil {
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
				lockFile := ObserverLockFile(wd, skill.Name)
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
		lockFile := ObserverLockFile(wd, "")
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
