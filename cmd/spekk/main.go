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
	"github.com/spekk-ai/spekk-cli/internal/install"
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
	// Set embedded assets so agents and skills work when binary is installed outside source tree
	cli.DefaultEmbeddedFS = spekk.EmbeddedFS
	cli.DefaultEmbeddedSkillFS = spekk.EmbeddedFS

	// Propagate build-time version to shared package for use by other packages (e.g., self-update).
	pkgversion.Version = version

	args := os.Args[1:]

	if len(args) == 0 {
		runParser(args)
		return
	}

	command := args[0]

	switch command {
	case "init":
		runInit(args[1:])

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

	case "prompt":
		runPrompt(args[1:])

	case "skill":
		runSkill(args[1:])

	case "install":
		runInstall(args[1:])

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
		"all":       {Names: []string{"--all"}, Type: cli.BoolFlag},
		"spec":      {Names: []string{"--spec", "-s"}, Type: cli.StringFlag},
		"assertion": {Names: []string{"--assertion"}, Type: cli.StringFlag},
		"allBranch": {Names: []string{"--all-branches"}, Type: cli.BoolFlag},
		"raw":       {Names: []string{"--raw"}, Type: cli.BoolFlag},
		"specsDir":  {Names: []string{"--specs-dir"}, Type: cli.StringFlag},
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
// Supports --watch / -w for live reload and --cross-branch (with an optional
// --branch-filter glob) for merge-preview mode.
func showSpekk(args []string) {
	specsDir := findSpecsDir()

	flags := cli.ParseFlags(args, cli.FlagSet{
		"watch":         {Names: []string{"--watch", "-w"}, Type: cli.BoolFlag},
		"cross-branch":  {Names: []string{"--cross-branch"}, Type: cli.BoolFlag},
		"branch-filter": {Names: []string{"--branch-filter"}, Type: cli.StringFlag},
	})

	opts := show.Options{
		CrossBranch:  flags.Bool("cross-branch"),
		BranchFilter: flags.String("branch-filter"),
	}

	if flags.Bool("watch") {
		if err := show.RunWatch(specsDir, opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		return
	}

	if err := show.Run(specsDir, opts); err != nil {
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
		if err := sandbox.ValidateSandboxName(subArgs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
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
		if err := sandbox.ValidateSandboxName(subArgs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
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
		if err := sandbox.ValidateSandboxName(name); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
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
		if err := sandbox.ValidateSandboxName(subArgs[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
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

	if err := sandbox.ValidateSandboxName(flags.String("name")); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
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
`)
			return
		}
	}

	if err := update.Run(checkOnly); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// specsReadme is written by spekk init so the new specs/ directory is
// non-empty (git tracks it) and explains itself to readers.
const specsReadme = `# Specs

This directory is a work queue for AI agents, managed with
[spekk](https://github.com/spekk-ai/spekk-cli).

Each spec is a folder containing a markdown file that states what must be
true, plus an assertions/ folder breaking that down into small, testable
assertions:

    specs/
      my-feature/
        my-feature.md          # what must be true, and why
        assertions/
          first-assertion.md   # one small, verifiable step

Common commands:

    spekk coach      # draft and refine specs with the coach agent
    spekk builder    # implement the next ready assertion
    spekk next       # print the next ready assertion
    spekk status     # overview of all specs and assertions
`

// runInit creates the specs/ directory so a project can start using spekk.
func runInit(args []string) {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Print(`
spekk init - Set up a project for spec-driven development

USAGE:
  spekk init

Creates a specs/ directory (at the git root if in a repository, otherwise
in the current directory) with a short README explaining the format.
Does nothing if specs/ already exists.
`)
			return
		}
	}

	specsDir := findSpecsDir()
	if info, err := os.Stat(specsDir); err == nil && info.IsDir() {
		fmt.Printf("specs/ already exists at %s — you're set.\n", specsDir)
		fmt.Println(`Run "spekk coach" to draft a spec, or "spekk next" to see what's ready.`)
		return
	}

	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating %s: %s\n", specsDir, err)
		os.Exit(1)
	}
	readmePath := filepath.Join(specsDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(specsReadme), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: writing %s: %s\n", readmePath, err)
		os.Exit(1)
	}

	fmt.Printf("Created %s\n", specsDir)
	fmt.Println(`
Next steps:
  spekk coach      # draft your first spec with the coach agent
  spekk builder    # implement the next ready assertion

Using a different coding assistant? Register the agents with it:
  spekk install --target claude-code|copilot|cursor|opencode|codex`)
}

// runPrompt prints the layered-resolved prompt for an agent to stdout.
func runPrompt(args []string) {
	usage := `
spekk prompt - Print an agent's resolved prompt to stdout

USAGE:
  spekk prompt <agent>

AGENTS:
  coach, builder, observer

The prompt is resolved through the standard layers (.spekk/ overrides and
extensions, then the embedded base), so the output is exactly what
"spekk <agent>" would launch with. Useful for piping into other tools or
for host-assistant shims installed via "spekk install".
`
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
	if args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage)
		return
	}

	resolver := cli.NewPromptResolver()
	content, err := resolver.GetPromptContent(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s (valid agents: coach, builder, observer)\n", err)
		os.Exit(1)
	}
	fmt.Println(content)
}

// runSkill exposes layered skill discovery: list and show.
func runSkill(args []string) {
	usage := `
spekk skill - Discover and print agent skills

USAGE:
  spekk skill list <agent>          List available skills and their source layer
  spekk skill show <agent> <name>   Print a skill's content to stdout

AGENTS:
  coach, builder

Skills resolve through layers: .spekk/skills/<agent>/ (project), then
~/.config/spekk/skills/<agent>/ (user), then the skills built into the binary.
`
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		if len(args) == 0 {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(1)
		}
		fmt.Print(usage)
		return
	}

	resolver := cli.NewSkillResolver(findInstallDir())

	switch args[0] {
	case "list":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: spekk skill list <agent>")
			os.Exit(1)
		}
		for _, entry := range resolver.ListSkills(args[1]) {
			fmt.Printf("%-40s %s\n", entry.Name, entry.Source)
		}

	case "show":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: spekk skill show <agent> <name>")
			os.Exit(1)
		}
		skill := resolver.ResolveSkill(args[1], args[2])
		if skill == nil {
			fmt.Fprintf(os.Stderr, "Error: skill %q not found for agent %q (try \"spekk skill list %s\")\n", args[2], args[1], args[1])
			os.Exit(1)
		}
		fmt.Println(skill.Content)

	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

// runInstall writes thin shim subagents into a host coding assistant.
func runInstall(args []string) {
	usage := `
spekk install - Install spekk agents into a coding assistant

USAGE:
  spekk install --target <tool> [--project]

TARGETS:
  claude-code (alias: claude)   ~/.claude/agents/
  copilot                       ~/.copilot/agents/ (project: .github/agents/)
  cursor                        ~/.cursor/agents/
  opencode                      ~/.config/opencode/agents/
  codex                         ~/.codex/prompts/ (global only)

OPTIONS:
  --target <tool>   Host tool to install into (required)
  --project         Install into the current project instead of globally
  --help, -h        Show this help message

Installs thin shims for the coach, builder, and observer agents. Shims
fetch their full instructions from this binary at session start via
"spekk prompt <agent>", so they never go stale — updating spekk updates
every installed agent.

OTHER TOOLS:
  Any assistant that can run shell commands can use spekk without an
  installer. Point it at the prompt directly:

    spekk prompt coach        # print the coach prompt
    spekk prompt builder      # print the builder prompt

  Wire that into your tool's custom-agent or rules mechanism (many tools
  also read AGENTS.md — paste a line there telling the agent to run
  "spekk prompt <agent>" when doing spec-driven work).
`
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Print(usage)
			return
		}
	}

	flags := cli.ParseFlags(args, cli.FlagSet{
		"target":  {Names: []string{"--target", "-t"}, Type: cli.StringFlag},
		"project": {Names: []string{"--project"}, Type: cli.BoolFlag},
	})

	if flags.String("target") == "" {
		fmt.Fprintf(os.Stderr, "Error: --target is required (valid targets: %s)\n", strings.Join(install.ValidTargets(), ", "))
		os.Exit(1)
	}

	written, err := install.Install(install.Options{
		Target:  flags.String("target"),
		Project: flags.Bool("project"),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	for _, path := range written {
		fmt.Println("installed:", path)
	}
}

// printHelp displays the help text with all available commands.
func printHelp() {
	fmt.Print(`
spekk - Spec-driven development CLI

USAGE:
  spekk [COMMAND]

COMMANDS:
  init      Set up a project for spec-driven development (creates specs/)
  show      Generate and display spec explorer web interface (-w to watch)
  status    Show comprehensive overview of all specs and assertions
  serve     Start WebSocket server for browser extension (--port, --host)
  coach     Launch the Coach Agent to create and refine specs
  builder   Launch the Builder Agent to implement specs
  observer  Launch the Observer Agent to monitor spec-code drift
  sandbox   Manage cloud sandbox environments (create, list, status, ssh, destroy, deploy)
  loop      Run orchestration workflows (builder/coach loops)
  prompt    Print an agent's resolved prompt to stdout
  skill     List and print agent skills (list, show)
  install   Install spekk agents into a coding assistant (--target)
  update    Self-update the spekk CLI to the latest version (--check to preview)
  version   Print the current version
  help      Show this help message

DEFAULT:
  When no command is provided, spekk runs the spec parser to find the next assertion.

FLAGS for "show":
  --watch, -w              Watch specs and live-reload the explorer
  --cross-branch           Merge-preview mode: show spec/assertion state across all branches
  --branch-filter <glob>   In cross-branch mode, only include branches matching the glob (e.g. 'feat/*')

FLAGS for "next":
  --all             Show all assertions
  --spec <name>     Filter by spec name
  --assertion <id>  Filter by assertion ID
  --all-branches    Include assertions from all branches
`)
}
