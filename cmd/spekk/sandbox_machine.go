package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/sandbox"
)

// isMachineMode reports whether a sandbox subcommand runs in machine mode,
// the interface for a server worker (specs/sandbox-machine-mode/).
func isMachineMode(args []string) bool {
	return slices.Contains(args, "--machine") && !slices.Contains(args, "--help") && !slices.Contains(args, "-h")
}

// createMachineSandbox is `spekk sandbox create <name> --machine`. Stdout
// carries the stage lines and nothing else, so each other write to stdout
// in this process goes to stderr.
func createMachineSandbox(args []string) {
	stages := os.Stdout
	os.Stdout = os.Stderr
	opts, err := parseMachineCreateArgs(args, time.Now())
	if err != nil {
		sandbox.WriteMachineFailure(stages, err)
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if err := sandbox.CreateMachine(os.Stdin, stages, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// parseMachineCreateArgs reads the flags of a machine-mode create. It takes
// only the flags of the contract, so a flag that would change what the
// server asked for is an error, not a surprise.
func parseMachineCreateArgs(args []string, now time.Time) (sandbox.CreateOptions, error) {
	var opts sandbox.CreateOptions
	var name, expires, timeout string
	var oneOff bool
	var labels []string
	values := map[string]*string{
		"--name":              &name,
		"--size":              &opts.Size,
		"--region":            &opts.Region,
		"--release":           &opts.Release,
		"--expires":           &expires,
		"--provision-timeout": &timeout,
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		target, takesValue := values[a]
		switch {
		case a == "--machine":
		case a == "--one-off":
			oneOff = true
		case a == "--label" || takesValue:
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") || args[i+1] == "" {
				return opts, fmt.Errorf("%s requires a value", a)
			}
			i++
			if a == "--label" {
				labels = append(labels, args[i])
			} else if *target != "" {
				return opts, fmt.Errorf("%s must be supplied only once", a)
			} else {
				*target = args[i]
			}
		case !strings.HasPrefix(a, "-") && name == "":
			name = a
		default:
			return opts, fmt.Errorf("machine mode does not take %q: it takes <name>, --size, --region, --one-off, --expires, --label, --release, and --provision-timeout", a)
		}
	}
	if name == "" {
		return opts, fmt.Errorf("a sandbox name is required: spekk sandbox create <name> --machine")
	}
	if err := sandbox.ValidateSandboxName(name); err != nil {
		return opts, err
	}
	if opts.Size == "" || opts.Region == "" {
		return opts, fmt.Errorf("machine mode needs --size and --region")
	}
	opts.Name = name

	var err error
	if opts.Kind, opts.ExpiresAt, err = sandbox.ResolveKind(oneOff, expires, now); err != nil {
		return opts, err
	}
	if opts.ProvisionTimeout, err = sandbox.ParseProvisionTimeout(timeout); err != nil {
		return opts, err
	}
	if opts.Labels, err = sandbox.ParseLabels(labels); err != nil {
		return opts, err
	}
	return opts, nil
}

// destroyMachineSandbox is `spekk sandbox destroy <name> --machine --force`.
// The DigitalOcean token comes on stdin, so stdin cannot answer a prompt,
// and --force is required.
func destroyMachineSandbox(args []string) {
	os.Stdout = os.Stderr
	name, force := "", false
	for _, a := range args {
		switch {
		case a == "--machine":
		case a == "--force" || a == "-f":
			force = true
		case !strings.HasPrefix(a, "-") && name == "":
			name = a
		default:
			fmt.Fprintf(os.Stderr, "Error: machine mode does not take %q: use spekk sandbox destroy <name> --machine --force\n", a)
			os.Exit(1)
		}
	}
	if name == "" || !force {
		fmt.Fprintln(os.Stderr, "Error: use spekk sandbox destroy <name> --machine --force; stdin carries the token, so it cannot answer the prompt")
		os.Exit(1)
	}
	if err := sandbox.ValidateSandboxName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if err := sandbox.DestroyMachine(os.Stdin, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
