// Command spekk is the CLI entry point for spec-driven development workflows.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	spekk "github.com/spekk-ai/spekk-cli"
	"github.com/spekk-ai/spekk-cli/internal/agent"
	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/parser"
	"github.com/spekk-ai/spekk-cli/internal/sandbox"
	"github.com/spekk-ai/spekk-cli/internal/serve"
	"github.com/spekk-ai/spekk-cli/internal/show"
	"github.com/spekk-ai/spekk-cli/internal/status"
	"github.com/spekk-ai/spekk-cli/internal/update"
	pkgversion "github.com/spekk-ai/spekk-cli/internal/version"
)

// version is set at build time via: go build -ldflags "-X main.version=1.2.3"
var version = "dev"

func main() {
	// Propagate build-time version to shared package for use by other packages
	pkgversion.Version = version

	// Set embedded assets so agents and skills work when binary is installed outside source tree
	cli.DefaultEmbeddedFS = spekk.EmbeddedFS
	cli.DefaultEmbeddedSkillFS = spekk.EmbeddedFS

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

	case "update":
		runUpdate(args[1:])

	case "version", "--version":
		fmt.Println(version)

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

// findInstallDir returns the spekk installation directory.
// Used for resolving skills and other on-disk assets (not prompts, which are embedded).
func findInstallDir() string {
	// Check cwd first (handles go run, go build, and running from project root)
	cwd, _ := os.Getwd()
	if cwd != "" && hasSpecsDir(cwd) {
		return cwd
	}

	// Check relative to executable location
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		dir := filepath.Dir(exe)
		if hasSpecsDir(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if hasSpecsDir(parent) {
			return parent
		}
	}

	// Fallback to cwd
	if cwd != "" {
		return cwd
	}
	return "."
}

func hasSpecsDir(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "specs"))
	return err == nil && info.IsDir()
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
		runBuilderLoop(findInstallDir())
	case "coach":
		runCoachLoop(findInstallDir())
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
func runBuilderLoop(installDir string) {
	agent.RunBuilderLoop(installDir)
}

// runCoachLoop runs the interactive coach loop.
func runCoachLoop(installDir string) {
	agent.RunCoachLoop(installDir)
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
// Supports --watch / -w flag for live reload.
func showSpekk(args []string) {
	specsDir := findSpecsDir()

	// Check for --watch / -w flag
	watch := false
	for _, a := range args {
		if a == "--watch" || a == "-w" {
			watch = true
			break
		}
	}

	if watch {
		if err := show.RunWatch(specsDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		return
	}

	if err := show.Run(specsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// launchServe starts the WebSocket server for the browser extension.
// Supports --port, --host, and --verbose flags.
func launchServe(args []string) {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"port":    {Names: []string{"--port", "-p"}, Type: cli.StringFlag},
		"host":    {Names: []string{"--host"}, Type: cli.StringFlag},
		"verbose": {Names: []string{"--verbose", "-v"}, Type: cli.BoolFlag},
		"help":    {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Print(`
spekk serve - Start WebSocket server for browser extension

USAGE:
  spekk serve [OPTIONS]

OPTIONS:
  --port, -p <port>   Port to listen on (default: 3118)
  --host <host>       Host to bind to (default: localhost)
  --verbose, -v       Enable debug logging for WebSocket messages
  --help, -h          Show this help message
`)
		return
	}

	opts := serve.Options{
		Verbose: flags.Bool("verbose"),
	}

	if p := flags.String("port"); p != "" {
		fmt.Sscanf(p, "%d", &opts.Port)
	}
	if h := flags.String("host"); h != "" {
		opts.Host = h
	}

	installDir := findInstallDir()
	if err := serve.Run(opts, installDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// launchSandbox manages cloud sandbox environments.
// Subcommands: create, list, status, ssh, destroy, deploy.
func launchSandbox(args []string) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Print(`
spekk sandbox - Manage cloud sandbox environments

USAGE:
  spekk sandbox <subcommand> [options]

SUBCOMMANDS:
  create      Create a new sandbox droplet
  list        List all sandbox droplets
  status      Show status of a sandbox
  ssh         SSH into a sandbox
  destroy     Destroy a sandbox droplet
  deploy      Deploy agent client to a sandbox

OPTIONS:
  --help, -h  Show this help message

Use "spekk sandbox <subcommand> --help" for more information about a subcommand.
`)
		return
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "create":
		createSandbox(subArgs)
	case "list":
		if err := sandbox.List(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	case "status":
		if len(subArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: spekk sandbox status <name>")
			os.Exit(1)
		}
		if err := sandbox.Status(subArgs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	case "ssh":
		if len(subArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: spekk sandbox ssh <name> [ssh-flags...]")
			os.Exit(1)
		}
		if err := sandbox.SSH(subArgs[0], subArgs[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	case "destroy":
		name := ""
		force := false
		for _, a := range subArgs {
			if a == "--force" || a == "-f" {
				force = true
			} else if !strings.HasPrefix(a, "-") && name == "" {
				name = a
			}
		}
		if name == "" {
			fmt.Fprintln(os.Stderr, "Usage: spekk sandbox destroy <name> [--force]")
			os.Exit(1)
		}
		if err := sandbox.Destroy(name, force); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	case "deploy":
		if len(subArgs) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: spekk sandbox deploy <name>")
			os.Exit(1)
		}
		if err := sandbox.Deploy(subArgs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown sandbox command: %s\n", subcommand)
		fmt.Fprintln(os.Stderr, `Run "spekk sandbox --help" for available subcommands.`)
		os.Exit(1)
	}
}

func createSandbox(args []string) {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"name":    {Names: []string{"--name"}, Type: cli.StringFlag},
		"region":  {Names: []string{"--region"}, Type: cli.StringFlag},
		"size":    {Names: []string{"--size"}, Type: cli.StringFlag},
		"project": {Names: []string{"--project"}, Type: cli.StringFlag},
		"vpc":     {Names: []string{"--vpc"}, Type: cli.StringFlag},
		"help":    {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Print(`
spekk sandbox create - Create a new sandbox droplet

USAGE:
  spekk sandbox create --name <name> [options]

OPTIONS:
  --name <name>        Sandbox name (required)
  --region <region>    DigitalOcean region (default: nyc1)
  --size <size>        Droplet size slug (default: s-2vcpu-4gb)
  --project <project>  Assign to a DigitalOcean project (name or UUID)
  --vpc <uuid>         Place droplet in a specific DigitalOcean VPC
`)
		return
	}

	if flags.String("name") == "" {
		fmt.Fprintln(os.Stderr, "Error: --name is required for sandbox create")
		os.Exit(1)
	}

	opts := sandbox.CreateOptions{
		Name:    flags.String("name"),
		Region:  flags.String("region"),
		Size:    flags.String("size"),
		Project: flags.String("project"),
		VPC:     flags.String("vpc"),
	}
	if err := sandbox.Create(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// runUpdate performs a self-update check and optional install.
func runUpdate(args []string) {
	checkOnly := false
	for _, a := range args {
		if a == "--check" || a == "-c" {
			checkOnly = true
		}
		if a == "--help" || a == "-h" {
			fmt.Print(`
spekk update - Self-update the spekk CLI binary

USAGE:
  spekk update [OPTIONS]

OPTIONS:
  --check, -c   Check for available updates without installing
  --help, -h    Show this help message

ENVIRONMENT:
  GEMFURY_TOKEN     API token for Gemfury authentication (required)
  GEMFURY_ACCOUNT   Gemfury account name (default: spekk)
`)
			return
		}
	}

	if err := update.Run(checkOnly); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
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
  update    Self-update the spekk CLI to the latest version (--check to preview)
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
