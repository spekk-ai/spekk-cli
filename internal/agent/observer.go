package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
// There is no scan-interval flag. It named a cadence inside a session that
// ran until it was stopped, and the observer no longer works that way: a run
// files one observation and ends. Cadence is now the schedule's
// business, which is `install-cron` or whatever the operator dispatches with.
var ObserverFlags = cli.FlagSet{
	"quiet":      {Names: []string{"--quiet"}, Type: cli.BoolFlag},
	"headless":   {Names: []string{"--headless"}, Type: cli.BoolFlag},
	"claudePath": {Names: []string{"--claude-path"}, Type: cli.StringFlag},
	"help":       {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

// ObserverConfig holds parsed observer options.
type ObserverConfig struct {
	Quiet      bool
	Headless   bool
	ClaudePath string
	InstallDir string
}

// removedObserverFlags names flags that were accepted once, mapped to what to
// tell someone still passing them.
//
// Dropping a flag from the set is not enough on its own. The first bare
// positional argument is read as a skill name, so `spekk observer --interval
// 60` would survive the parse and launch a skill called "60" -- the wrong
// thing, quietly, for anyone whose script still carries the old flag. A
// removed flag has to fail loudly and say what replaced it.
var removedObserverFlags = map[string]string{
	"--interval": "cadence is now set by the schedule that runs the observer, not by the run itself.\n" +
		"A run files one observation and ends. Use 'spekk observer install-cron --loop-interval <minutes>',\n" +
		"or set the cadence in whatever dispatches the observer.",
}

// RemovedObserverFlag returns the first removed flag present in args, with the
// message explaining it, or empty strings when there is none.
func RemovedObserverFlag(args []string) (flag, message string) {
	for _, a := range args {
		name, _, _ := strings.Cut(a, "=")
		if msg, removed := removedObserverFlags[name]; removed {
			return name, msg
		}
	}
	return "", ""
}

// ParseObserverFlags parses args into an ObserverConfig. No flag can fail to
// parse, so there is no error to return.
func ParseObserverFlags(args []string) ObserverConfig {
	parsed := cli.ParseFlags(args, ObserverFlags)
	return ObserverConfig{
		Quiet:      parsed.Bool("quiet"),
		Headless:   parsed.Bool("headless"),
		ClaudePath: parsed.String("claudePath"),
	}
}

// BuildObserverOptionsMessage builds the CLI options suffix for the activation message.
func BuildObserverOptionsMessage(cfg ObserverConfig) string {
	if !cfg.Quiet {
		return ""
	}
	return "\n\nCLI Options provided:\n- Quiet mode: enabled" +
		"\n\nYou can use this preference in how you report."
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

	// Before anything reads a positional argument as a skill name.
	if flag, message := RemovedObserverFlag(args); flag != "" {
		fmt.Fprintf(os.Stderr, "Error: %s is no longer a flag.\n%s\n", flag, message)
		os.Exit(1)
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
			cfg := ParseObserverFlags(args)

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

	cfg := ParseObserverFlags(args)
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
