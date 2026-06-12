package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// specIDRe matches valid spec/assertion IDs: lowercase alphanumeric and hyphens.
var specIDRe = regexp.MustCompile(`^[a-z0-9-]+$`)

// ValidateSpecOrAssertionID checks that a spec or assertion ID contains only
// valid characters (lowercase letters, digits, hyphens). This prevents prompt
// injection when IDs are interpolated into activation messages.
func ValidateSpecOrAssertionID(flagName, value string) error {
	if value == "" {
		return nil
	}
	if !specIDRe.MatchString(value) {
		return fmt.Errorf("invalid %s value %q: must contain only lowercase letters, digits, and hyphens (a-z, 0-9, -)", flagName, value)
	}
	return nil
}

// ANSI color codes for console output.
const (
	colorReset   = "\033[0m"
	colorBright  = "\033[1m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
)

func colorLog(color, message string) {
	fmt.Printf("%s%s%s\n", color, message, colorReset)
}

// BuilderFlags defines the flag set for the builder CLI.
var BuilderFlags = cli.FlagSet{
	"once":        {Names: []string{"--once"}, Type: cli.BoolFlag},
	"dryRun":      {Names: []string{"--dry-run", "-d"}, Type: cli.BoolFlag},
	"confirm":     {Names: []string{"--confirm", "-c"}, Type: cli.BoolFlag},
	"interactive": {Names: []string{"--interactive", "-i"}, Type: cli.BoolFlag},
	"spec":        {Names: []string{"--spec", "-s"}, Type: cli.StringFlag},
	"assertion":   {Names: []string{"--assertion"}, Type: cli.StringFlag},
	"help":        {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

// BuilderConfig holds parsed builder options.
type BuilderConfig struct {
	Once        bool
	DryRun      bool
	Confirm     bool
	Interactive bool
	Spec        string
	Assertion   string
	InstallDir  string
}

// ParseBuilderFlags parses args into a BuilderConfig.
func ParseBuilderFlags(args []string) BuilderConfig {
	parsed := cli.ParseFlags(args, BuilderFlags)
	return BuilderConfig{
		Once:        parsed.Bool("once"),
		DryRun:      parsed.Bool("dryRun"),
		Confirm:     parsed.Bool("confirm"),
		Interactive: parsed.Bool("interactive"),
		Spec:        parsed.String("spec"),
		Assertion:   parsed.String("assertion"),
	}
}

// ExtractSkillArg returns the first positional (non-flag) argument
// from the args list, skipping known builder flags and their values.
func ExtractSkillArg(args []string) string {
	flagStrings := make(map[string]bool)
	stringFlags := make(map[string]bool)

	for _, def := range BuilderFlags {
		for _, f := range def.Names {
			flagStrings[f] = true
			if def.Type == cli.StringFlag {
				stringFlags[f] = true
			}
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if flagStrings[arg] {
			if stringFlags[arg] {
				i++ // skip value
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}

// AssertionResult represents a parsed assertion from the parser.
type AssertionResult struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	File     string `json:"file"`
	Priority int    `json:"priority"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Spec     *struct {
		ID string `json:"id"`
	} `json:"spec"`
}

// BuildSpekkNextCommand builds the command args for getting the next assertion.
func BuildSpekkNextCommand(spekkBin string, cfg BuilderConfig) []string {
	args := []string{spekkBin, "next"}
	if cfg.Spec != "" {
		args = append(args, "--spec", cfg.Spec)
	}
	if cfg.Assertion != "" {
		args = append(args, "--assertion", cfg.Assertion)
	}
	return args
}

// getNextAssertion calls the parser to get the next assertion.
func getNextAssertion(spekkBin string, cfg BuilderConfig) (*AssertionResult, error) {
	cmdArgs := BuildSpekkNextCommand(spekkBin, cfg)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	out, err := cmd.Output()
	if err != nil {
		// Try to parse stdout from the error
		if exitErr, ok := err.(*exec.ExitError); ok {
			if len(out) > 0 {
				var result AssertionResult
				if jsonErr := json.Unmarshal(out, &result); jsonErr == nil {
					return &result, nil
				}
			}
			_ = exitErr
		}
		return nil, fmt.Errorf("failed to get next assertion: %w", err)
	}

	var result AssertionResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse assertion JSON: %w", err)
	}
	return &result, nil
}

// displayAssertion prints assertion details with colored output.
func displayAssertion(a *AssertionResult) {
	colorLog(colorGreen, fmt.Sprintf("Assertion: %s", a.ID))
	colorLog(colorBlue, fmt.Sprintf("   Title: %s", a.Title))
	colorLog(colorBlue, fmt.Sprintf("   File: %s", a.File))
	colorLog(colorBlue, fmt.Sprintf("   Priority: %d", a.Priority))
	colorLog(colorBlue, fmt.Sprintf("   Status: %s", a.Status))
	if a.Spec != nil {
		colorLog(colorBlue, fmt.Sprintf("   Spec: %s", a.Spec.ID))
	}
}

// askConfirmation prompts the user for y/n/q and returns "proceed", "skip", or "quit".
func askConfirmation(message string) string {
	fmt.Printf("%s [y/n/q]: ", message)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		switch answer {
		case "q":
			return "quit"
		case "n":
			return "skip"
		default:
			return "proceed"
		}
	}
	return "quit"
}

// findSpekkBin returns the path to the spekk binary (self).
func findSpekkBin() string {
	exe, err := os.Executable()
	if err != nil {
		return "spekk"
	}
	return exe
}

type processHolder struct {
	mu      sync.Mutex
	process *os.Process
}

func (ph *processHolder) set(p *os.Process) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.process = p
}

func (ph *processHolder) signal(sig os.Signal) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	if ph.process != nil {
		ph.process.Signal(sig)
	}
}

func (ph *processHolder) isSet() bool {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	return ph.process != nil
}

// launchClaude spawns the claude CLI with the given args and inherited stdio.
// Returns true if claude exited successfully.
// holder receives the started process for SIGINT forwarding.
func launchClaude(claudeArgs []string, holder *processHolder) (bool, error) {
	cmd := exec.Command("claude", claudeArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		if isNotFound(err) {
			colorLog(colorRed, "Error: Claude Code CLI not found. Please install Claude Code first.")
			colorLog(colorBlue, "Visit: https://claude.ai/code for installation instructions.")
			os.Exit(1)
		}
		return false, fmt.Errorf("error launching Claude Code: %w", err)
	}

	if holder != nil {
		holder.set(cmd.Process)
	}

	err := cmd.Wait()

	if holder != nil {
		holder.set(nil)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			colorLog(colorYellow, fmt.Sprintf("Claude Code exited with code %d", exitErr.ExitCode()))
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// RunBuilder is the main entry point for the builder agent.
func RunBuilder(args []string, installDir string) {
	cfg := ParseBuilderFlags(args)
	cfg.InstallDir = installDir

	// Validate spec/assertion flags before any interpolation
	if err := ValidateSpecOrAssertionID("--spec", cfg.Spec); err != nil {
		colorLog(colorRed, fmt.Sprintf("Error: %s", err))
		os.Exit(1)
	}
	if err := ValidateSpecOrAssertionID("--assertion", cfg.Assertion); err != nil {
		colorLog(colorRed, fmt.Sprintf("Error: %s", err))
		os.Exit(1)
	}

	// Handle help
	if hasHelp(args) {
		ShowHelp(installDir, "builder")
		return
	}

	// Skill subcommand
	skillName := ExtractSkillArg(args)
	if skillName != "" {
		sr := &cli.SkillResolver{
			Cwd:        cwd(),
			InstallDir: installDir,
		}
		if sr.ResolveSkill("builder", skillName) != nil {
			colorLog(colorCyan, fmt.Sprintf("Starting Builder Agent with skill: %s", skillName))
			opts := LaunchOptions{
				Agent:      "builder",
				InstallDir: installDir,
			}
			message, err := BuildActivationMessage(opts)
			if err != nil {
				colorLog(colorRed, fmt.Sprintf("Error: %s", err))
				os.Exit(1)
			}
			skillMsg, err := BuildSkillMessage(installDir, "builder", skillName, args)
			if err != nil {
				colorLog(colorRed, fmt.Sprintf("Error: %s", err))
				os.Exit(1)
			}
			message += skillMsg

			success, err := launchClaude([]string{"--dangerously-skip-permissions", message}, nil)
			if err != nil {
				colorLog(colorRed, fmt.Sprintf("Error: %s", err))
				os.Exit(1)
			}
			if success {
				colorLog(colorGreen, "Builder agent completed skill work")
			}
			return
		}
	}

	// Interactive mode: suppress SIGINT, let claude handle it
	if cfg.Interactive {
		signal.Ignore(syscall.SIGINT)
		launchInteractiveBuilder(cfg)
		return
	}

	// Non-interactive modes: manage SIGINT
	active := &processHolder{}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		for range sigCh {
			if active.isSet() {
				colorLog(colorYellow, "\nInterrupting current build...")
				active.signal(syscall.SIGINT)
			} else {
				colorLog(colorYellow, "\nReceived SIGINT. Exiting gracefully...")
				os.Exit(0)
			}
		}
	}()

	// Mode banner
	if cfg.DryRun {
		colorLog(colorCyan, "Dry run - showing what would be built...")
	} else if cfg.Once {
		colorLog(colorCyan, "Starting Builder Agent (single build)...")
	} else {
		colorLog(colorCyan, "Starting Builder Agent (continuous mode)...")
		colorLog(colorYellow, "Press Ctrl+C to exit gracefully.")
	}

	spekkBin := findSpekkBin()
	iteration := 0

	for {
		iteration++

		if !cfg.Once {
			colorLog(colorBright, fmt.Sprintf("\n--- Iteration %d ---", iteration))
		}

		colorLog(colorBlue, "Getting next assertion...")

		result, err := getNextAssertion(spekkBin, cfg)
		if err != nil {
			if cfg.Once {
				colorLog(colorRed, err.Error())
				os.Exit(1)
			}
			colorLog(colorYellow, "Parser error (transient). Retrying in 5s...")
			colorLog(colorYellow, "   "+err.Error())
			time.Sleep(5 * time.Second)
			continue
		}

		// Handle result types
		if result.Type == "complete" {
			if cfg.Once {
				colorLog(colorGreen, "No assertions to work on.")
				os.Exit(0)
			}
			colorLog(colorGreen, "All assertions completed. Waiting for new work...")
			time.Sleep(5 * time.Second)
			continue
		}

		if result.Type == "error" {
			if cfg.Once {
				colorLog(colorRed, result.Message)
				os.Exit(1)
			}
			colorLog(colorYellow, "Parser returned error (transient). Retrying in 5s...")
			colorLog(colorYellow, "   "+result.Message)
			time.Sleep(5 * time.Second)
			continue
		}

		if result.Type != "assertion" {
			if cfg.Once {
				colorLog(colorRed, "Unexpected result from parser")
				os.Exit(1)
			}
			colorLog(colorYellow, "Unexpected result type from parser. Retrying in 5s...")
			time.Sleep(5 * time.Second)
			continue
		}

		displayAssertion(result)

		// Dry run: display and exit
		if cfg.DryRun {
			colorLog(colorCyan, "\n(Dry run - no build performed)")
			os.Exit(0)
		}

		// Confirm before building
		if cfg.Confirm {
			answer := askConfirmation("Build this assertion?")
			switch answer {
			case "quit":
				colorLog(colorYellow, "Exiting builder.")
				os.Exit(0)
			case "skip":
				colorLog(colorBlue, "Skipping assertion...")
				if cfg.Once {
					os.Exit(0)
				}
				continue
			}
		}

		// Build the assertion
		colorLog(colorMagenta, "Launching Claude Code Builder Agent...")

		opts := LaunchOptions{
			Agent:      "builder",
			InstallDir: installDir,
		}
		message, err := BuildActivationMessage(opts)
		if err != nil {
			colorLog(colorRed, fmt.Sprintf("Error: %s", err))
			os.Exit(1)
		}

		// Add filter override if spec/assertion flags are set
		if cfg.Spec != "" || cfg.Assertion != "" {
			cmdArgs := BuildSpekkNextCommand(spekkBin, cfg)
			override := fmt.Sprintf("**IMPORTANT: Use this command instead of the default `spekk next`:**\n\n```bash\n%s\n```\n\nThis ensures you work on the correct assertion based on the user's filter.\n\n---\n\n",
				strings.Join(cmdArgs, " "))
			message = override + message
		}

		success, launchErr := launchClaude(
			[]string{"--dangerously-skip-permissions", message},
			active,
		)

		if launchErr != nil {
			colorLog(colorRed, "Build failed: "+launchErr.Error())
			if cfg.Once {
				os.Exit(1)
			}
			colorLog(colorYellow, "Continuing to next assertion...")
		} else if success {
			colorLog(colorGreen, "Builder agent completed work")
		} else {
			colorLog(colorYellow, "Build did not succeed.")
		}

		if cfg.Once {
			if success {
				os.Exit(0)
			}
			os.Exit(1)
		}

		// Brief pause before next iteration
		colorLog(colorBlue, "Checking for more work...")
		time.Sleep(1 * time.Second)
	}
}

// launchInteractiveBuilder launches claude in interactive mode with --system-prompt.
func launchInteractiveBuilder(cfg BuilderConfig) {
	colorLog(colorCyan, "Starting Builder Agent (interactive mode)...")
	colorLog(colorYellow, "Ctrl+C interrupts the current action. Use /exit to end the session.")

	opts := LaunchOptions{
		Agent:      "builder",
		InstallDir: cfg.InstallDir,
	}
	message, err := BuildActivationMessage(opts)
	if err != nil {
		colorLog(colorRed, fmt.Sprintf("Error: %s", err))
		os.Exit(1)
	}

	// Add filter hint if provided
	if cfg.Spec != "" || cfg.Assertion != "" {
		spekkBin := findSpekkBin()
		cmdArgs := BuildSpekkNextCommand(spekkBin, cfg)
		hint := fmt.Sprintf("**Note: When you run the spekk next command, use these filters:**\n\n```bash\n%s\n```\n\n---\n\n",
			strings.Join(cmdArgs, " "))
		message = hint + message
	}

	// Interactive mode: use --system-prompt so claude waits for user input
	success, launchErr := launchClaude(
		[]string{"--dangerously-skip-permissions", "--system-prompt", message},
		nil,
	)

	if launchErr != nil {
		colorLog(colorRed, fmt.Sprintf("Error: %s", launchErr))
		os.Exit(1)
	}
	if success {
		colorLog(colorGreen, "Builder agent session ended")
	}
}

func hasHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" || a == "help" {
			return true
		}
	}
	return false
}
