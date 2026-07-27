package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/observation"
)

// observerGoSubcommands lists the Go-native observer subcommands that must
// never be treated as skill names by the agent launcher. main.go routes them
// here before falling through to the observer agent.
var observerGoSubcommands = map[string]func(args []string){
	"digest":     runObserverDigest,
	"scan-check": runObserverScanCheck,
}

// observerDigestUsage is the help text for `spekk observer digest --help`.
const observerDigestUsage = `
spekk observer digest - Render the observation digest view

USAGE:
  spekk observer digest [--json]

OPTIONS:
  --json      Emit the digest as a JSON array
  --help, -h  Show this help message

The digest is a rendered view, not a committed file: the open observations
across all visible observer/* branches plus main, ranked by severity
(high > medium > low, oldest first within a severity), capped at 5. A slug
whose observation is already on main counts as not open.

The view reads committed git state only (no checkout, no remote call). Run
"git fetch" first when you want remote-tracking observer branches current.
`

// runObserverDigest implements `spekk observer digest`.
func runObserverDigest(args []string) {
	if code := execObserverDigest(args, os.Stdout, os.Stderr); code != 0 {
		os.Exit(code)
	}
}

// execObserverDigest is the testable core of runObserverDigest.
func execObserverDigest(args []string, stdout, stderr io.Writer) int {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"json": {Names: []string{"--json"}, Type: cli.BoolFlag},
		"help": {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})
	if flags.Bool("help") {
		fmt.Fprint(stdout, observerDigestUsage)
		return 0
	}

	u, err := observation.LoadUnion()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}
	warn(stderr, u.Warnings)

	digest := u.Digest()
	if flags.Bool("json") {
		printDigestJSON(stdout, digest)
		return 0
	}
	printDigestTable(stdout, digest)
	return 0
}

// printDigestTable renders the digest for humans.
func printDigestTable(w io.Writer, digest []*observation.Observation) {
	if len(digest) == 0 {
		fmt.Fprintln(w, "No open observations.")
		return
	}
	fmt.Fprintf(w, "Observation digest — %d open item(s), severity-ranked (cap %d):\n\n",
		len(digest), observation.DigestCap)
	for i, o := range digest {
		fmt.Fprintf(w, "%d. [%s] %s — %s\n", i+1, o.Severity, o.Slug, o.Title)
		fmt.Fprintf(w, "   branch %s, created %s\n", observation.BranchName(o.Slug), o.Created)
		fmt.Fprintf(w, "   affected: %s\n", strings.Join(o.Affected, ", "))
	}
}

// printDigestJSON renders the digest for tooling.
func printDigestJSON(w io.Writer, digest []*observation.Observation) {
	type entry struct {
		Slug     string   `json:"slug"`
		Severity string   `json:"severity"`
		Title    string   `json:"title"`
		Branch   string   `json:"branch"`
		Created  string   `json:"created"`
		Affected []string `json:"affected"`
	}
	entries := make([]entry, 0, len(digest))
	for _, o := range digest {
		entries = append(entries, entry{
			Slug: o.Slug, Severity: o.Severity, Title: o.Title,
			Branch: observation.BranchName(o.Slug), Created: o.Created, Affected: o.Affected,
		})
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(w, string(data))
}

// observerScanCheckUsage is the help text for `spekk observer scan-check --help`.
const observerScanCheckUsage = `
spekk observer scan-check - Check drift against the cross-branch
observation union before creating an observation

USAGE:
  spekk observer scan-check --type <type> --slug <slug> --affected <paths>

OPTIONS:
  --type <type>       code_spec_misalignment or outdated_specs (required)
  --slug <slug>       The slug the observation would be given (required)
  --affected <paths>  Comma-separated evidence file paths (required)
  --help, -h          Show this help message

The observer runs this before creating any observation. The result is JSON:

  {"result":"covered", ...}     an observation on a visible branch already
                                covers the drift (same type, overlapping
                                affected paths); create nothing
  {"result":"clear","slug":..}  no coverage; create the observation with the
                                returned slug (a -YYYYMMDD suffix is added
                                when the plain slug is taken by an
                                observation already on main)

The check reads committed git state only. Run "git fetch" first so
remote-tracking observer/* branches are current.
`

// scanCheckResult is the JSON shape scan-check prints.
type scanCheckResult struct {
	Result string `json:"result"` // suppressed | covered | clear
	Slug   string `json:"slug,omitempty"`
	Branch string `json:"branch,omitempty"`
	Ref    string `json:"ref,omitempty"`
	Match  string `json:"match,omitempty"`
	Reason string `json:"reason,omitempty"`
	By     string `json:"by,omitempty"`
}

// runObserverScanCheck implements `spekk observer scan-check`.
func runObserverScanCheck(args []string) {
	code := execObserverScanCheck(args, os.Stdout, os.Stderr, time.Now())
	if code != 0 {
		os.Exit(code)
	}
}

// execObserverScanCheck is the testable core of runObserverScanCheck.
func execObserverScanCheck(args []string, stdout, stderr io.Writer, now time.Time) int {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"type":     {Names: []string{"--type"}, Type: cli.StringFlag},
		"slug":     {Names: []string{"--slug"}, Type: cli.StringFlag},
		"affected": {Names: []string{"--affected"}, Type: cli.StringFlag},
		"help":     {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})
	if flags.Bool("help") {
		fmt.Fprint(stdout, observerScanCheckUsage)
		return 0
	}

	typ := flags.String("type")
	slug := flags.String("slug")
	affected := splitPaths(flags.String("affected"))

	var missing []string
	if typ == "" {
		missing = append(missing, "--type")
	}
	if slug == "" {
		missing = append(missing, "--slug")
	}
	if len(affected) == 0 {
		missing = append(missing, "--affected")
	}
	if len(missing) > 0 {
		fmt.Fprintf(stderr, "Error: missing required flag(s): %s\n", strings.Join(missing, ", "))
		return 1
	}

	u, err := observation.LoadUnion()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}
	warn(stderr, u.Warnings)

	if covering := u.FindCovering(typ, affected); covering != nil {
		printJSON(stdout, scanCheckResult{
			Result: "covered", Slug: covering.Slug,
			Branch: observation.BranchName(covering.Slug), Ref: covering.Ref,
		})
		return 0
	}

	resolved := observation.ResolveSlug(slug, u.OnMain, now)
	printJSON(stdout, scanCheckResult{
		Result: "clear", Slug: resolved, Branch: observation.BranchName(resolved),
	})
	return 0
}

// splitPaths splits a comma-separated path list, dropping empty entries.
func splitPaths(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// printJSON writes v as a single JSON line.
func printJSON(w io.Writer, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(w, string(data))
}

// warn prints skipped-file warnings to stderr.
func warn(w io.Writer, warnings []string) {
	for _, msg := range warnings {
		fmt.Fprintf(w, "warning: %s\n", msg)
	}
}
