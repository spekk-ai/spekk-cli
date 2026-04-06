// Command spekk is the CLI entry point for spec-driven development workflows.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spekk-dev/spekk-cli/internal/agent"
	"github.com/spekk-dev/spekk-cli/internal/cli"
	"github.com/spekk-dev/spekk-cli/internal/parser"
	"github.com/spekk-dev/spekk-cli/internal/show"
	"github.com/spekk-dev/spekk-cli/internal/status"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		runParser(args)
		return
	}

	command := args[0]

	switch command {
	case "next":
		runParser(args[1:])

	case "coach":
		launchCoachAgent(args[1:])

	case "builder":
		launchBuilderAgent(args[1:])

	case "observer":
		launchObserverAgent(args[1:])

	case "loop":
		runLoop(args[1:])

	case "status":
		showStatus()

	case "show":
		showSpekk(args[1:])

	case "serve":
		launchServe(args[1:])

	case "sandbox":
		launchSandbox(args[1:])

	case "help", "--help", "-h":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, `Run "spekk help" for available commands.`)
		os.Exit(1)
	}
}

// runParser runs the spec parser to find the next assertion.
// Accepts flags: --all, --spec <name>, --assertion <name>, --all-branches, --raw, --specs-dir
func runParser(args []string) {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"all":        {Names: []string{"--all"}, Type: cli.BoolFlag},
		"spec":       {Names: []string{"--spec", "-s"}, Type: cli.StringFlag},
		"assertion":  {Names: []string{"--assertion"}, Type: cli.StringFlag},
		"allBranch":  {Names: []string{"--all-branches"}, Type: cli.BoolFlag},
		"raw":        {Names: []string{"--raw"}, Type: cli.BoolFlag},
		"specsDir":   {Names: []string{"--specs-dir"}, Type: cli.StringFlag},
	})

	specsDir := flags.String("specsDir")
	if specsDir == "" {
		specsDir = findSpecsDir()
	}

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		out, _ := parser.FormatError(err.Error())
		fmt.Println(string(out))
		os.Exit(1)
	}

	if len(result.Specs) == 0 {
		out, _ := parser.FormatEmpty()
		fmt.Println(string(out))
		return
	}

	// --raw mode: output everything for downstream Node shims
	if flags.Bool("raw") {
		out, _ := parser.FormatRaw(result)
		fmt.Println(string(out))
		return
	}

	// --all mode: output full hierarchy
	if flags.Bool("all") {
		out, _ := parser.FormatHierarchy(result)
		fmt.Println(string(out))
		return
	}

	// Default: find next assertion
	opts := parser.FindOptions{
		AssertionID:   flags.String("assertion"),
		SpecID:        flags.String("spec"),
		AllBranches:   flags.Bool("allBranch"),
		CurrentBranch: currentBranch(),
	}

	next := parser.FindNextAssertion(result.Assertions, result.Specs, opts)
	if next == nil {
		out, _ := parser.FormatComplete()
		fmt.Println(string(out))
		return
	}

	out, _ := parser.FormatNextAssertion(next, result.Specs)
	fmt.Println(string(out))
}

// findSpecsDir locates the specs/ directory using git root.
func findSpecsDir() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return filepath.Join(strings.TrimSpace(string(out)), "specs")
	}
	// Fallback: assume cwd
	wd, _ := os.Getwd()
	return filepath.Join(wd, "specs")
}

// currentBranch returns the current git branch name.
func currentBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// findInstallDir returns the spekk installation directory
// (the directory containing the running binary).
func findInstallDir() string {
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		return filepath.Dir(filepath.Dir(exe)) // bin/spekk-go -> project root
	}
	// Fallback: use compile-time caller info
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

// launchCoachAgent launches the Coach Agent to create and refine specs.
func launchCoachAgent(args []string) {
	installDir := findInstallDir()

	// Handle help
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		agent.ShowHelp(installDir, "coach")
		return
	}

	// Build activation message
	opts := agent.LaunchOptions{
		Agent:      "coach",
		InstallDir: installDir,
	}

	// Check for skill subcommand
	if len(args) > 0 {
		sr := &cli.SkillResolver{
			HomeDir:    homeDir(),
			Cwd:        cwdStr(),
			InstallDir: installDir,
		}
		if sr.ResolveSkill("coach", args[0]) != nil {
			skillMsg, err := agent.BuildSkillMessage(installDir, "coach", args[0], args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				os.Exit(1)
			}
			opts.ExtraMessage = skillMsg
		}
	}

	message, err := agent.BuildActivationMessage(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if err := agent.Launch(message); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func cwdStr() string {
	d, _ := os.Getwd()
	return d
}

// launchBuilderAgent launches the Builder Agent to implement specs.
func launchBuilderAgent(args []string) {
	agent.RunBuilder(args, findInstallDir())
}

// launchObserverAgent launches the Observer Agent to monitor spec-code drift.
func launchObserverAgent(args []string) {
	agent.RunObserver(args, findInstallDir())
}

// runLoop routes to the loop orchestration handlers.
func runLoop(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, `Usage: spekk loop <command>`)
		fmt.Fprintln(os.Stderr, `Commands:`)
		fmt.Fprintln(os.Stderr, `  builder   Run the automated builder loop`)
		fmt.Fprintln(os.Stderr, `  coach     Run the interactive coach loop`)
		os.Exit(1)
	}

	switch args[0] {
	case "builder":
		runBuilderLoop(args[1:])
	case "coach":
		runCoachLoop(args[1:])
	case "help", "--help", "-h":
		fmt.Println(`
spekk loop - Orchestration workflows for spec-driven development

USAGE:
  spekk loop [COMMAND]

COMMANDS:
  builder   Run the automated builder loop (gets next assertion, implements, commits, repeats)
  coach     Run the interactive coach loop (create specs, commit, repeat)
  help      Show this help message`)
	default:
		fmt.Fprintf(os.Stderr, "unknown loop command: %s\n", args[0])
		fmt.Fprintln(os.Stderr, `Run "spekk loop help" for available commands.`)
		os.Exit(1)
	}
}

// runBuilderLoop runs the automated builder loop.
func runBuilderLoop(args []string) {
	agent.RunBuilderLoop(args, findInstallDir())
}

// runCoachLoop runs the interactive coach loop.
func runCoachLoop(args []string) {
	agent.RunCoachLoop(args, findInstallDir())
}

// showStatus displays a comprehensive overview of all specs and assertions.
func showStatus() {
	specsDir := findSpecsDir()
	if err := status.Show(specsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// showSpekk generates and displays the spec explorer web interface.
// Supports --watch / -w flag.
func showSpekk(args []string) {
	specsDir := findSpecsDir()
	if err := show.Run(specsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// launchServe starts the WebSocket server for the browser extension.
// Supports --port and --host flags.
func launchServe(args []string) {
	// TODO: implement serve handler
	fmt.Fprintln(os.Stderr, "serve: not implemented yet")
	os.Exit(0)
}

// launchSandbox manages cloud sandbox environments.
// Subcommands: create, list, status, ssh, destroy, deploy.
func launchSandbox(args []string) {
	// TODO: implement sandbox handler
	fmt.Fprintln(os.Stderr, "sandbox: not implemented yet")
	os.Exit(0)
}

// printHelp displays the help text with all available commands.
func printHelp() {
	fmt.Print(`
spekk - Spec-driven development CLI

USAGE:
  spekk [COMMAND]

COMMANDS:
  show      Generate and display spec explorer web interface (-w to watch)
  status    Show comprehensive overview of all specs and assertions
  serve     Start WebSocket server for browser extension (--port, --host)
  coach     Launch the Coach Agent to create and refine specs
  builder   Launch the Builder Agent to implement specs
  observer  Launch the Observer Agent to monitor spec-code drift
  sandbox   Manage cloud sandbox environments (create, list, status, ssh, destroy, deploy)
  loop      Run orchestration workflows (builder/coach loops)
  help      Show this help message

DEFAULT:
  When no command is provided, spekk runs the spec parser to find the next assertion.

FLAGS for "next":
  --all             Show all assertions
  --spec <name>     Filter by spec name
  --assertion <id>  Filter by assertion ID
  --all-branches    Include assertions from all branches
`)
}
