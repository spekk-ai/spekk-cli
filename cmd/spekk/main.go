// Command spekk is the CLI entry point for spec-driven development workflows.
package main

import (
	"fmt"
	"os"
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
// Accepts flags: --all, --spec <name>, --assertion <name>, --all-branches
func runParser(args []string) {
	// TODO: implement via parser package
	fmt.Fprintln(os.Stderr, "parser: not implemented yet")
	os.Exit(0)
}

// launchCoachAgent launches the Coach Agent to create and refine specs.
func launchCoachAgent(args []string) {
	// TODO: implement via coach package
	fmt.Fprintln(os.Stderr, "coach agent: not implemented yet")
	os.Exit(0)
}

// launchBuilderAgent launches the Builder Agent to implement specs.
func launchBuilderAgent(args []string) {
	// TODO: implement via builder package
	fmt.Fprintln(os.Stderr, "builder agent: not implemented yet")
	os.Exit(0)
}

// launchObserverAgent launches the Observer Agent to monitor spec-code drift.
func launchObserverAgent(args []string) {
	// TODO: implement via observer package
	fmt.Fprintln(os.Stderr, "observer agent: not implemented yet")
	os.Exit(0)
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
	// TODO: implement builder loop
	fmt.Fprintln(os.Stderr, "builder loop: not implemented yet")
	os.Exit(0)
}

// runCoachLoop runs the interactive coach loop.
func runCoachLoop(args []string) {
	// TODO: implement coach loop
	fmt.Fprintln(os.Stderr, "coach loop: not implemented yet")
	os.Exit(0)
}

// showStatus displays a comprehensive overview of all specs and assertions.
func showStatus() {
	// TODO: implement status display
	fmt.Fprintln(os.Stderr, "status: not implemented yet")
	os.Exit(0)
}

// showSpekk generates and displays the spec explorer web interface.
// Supports --watch / -w flag.
func showSpekk(args []string) {
	// TODO: implement show handler
	fmt.Fprintln(os.Stderr, "show: not implemented yet")
	os.Exit(0)
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
