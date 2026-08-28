package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/conversation"
	"github.com/spekk-ai/spekk-cli/internal/dontflag"
	"github.com/spekk-ai/spekk-cli/internal/observation"
	"github.com/spekk-ai/spekk-cli/internal/observer"
)

// observerGoSubcommands lists the Go-native observer subcommands that must
// never be treated as skill names by the agent launcher. main.go routes them
// here before falling through to the observer agent.
var observerGoSubcommands = map[string]func(args []string){
	"digest":     runObserverDigest,
	"scan-check": runObserverScanCheck,
	"announce":   runObserverAnnounce,
}

// observerAnnounceUsage is the help text for `spekk observer announce --help`.
const observerAnnounceUsage = `
spekk observer announce - Announce the top unannounced observations

USAGE:
  spekk observer announce

OPTIONS:
  --help, -h  Show this help message

One invocation, in order: run git fetch (the only remote read); refresh the
index; pick the top unannounced open observations — severity high or medium
only (low never announces), high first, oldest first within a severity, and
only from observer/* branches visible on origin; open ONE conversation that
carries them via the sandbox conversation spool; then commit the announced:
frontmatter flip to each observer branch and push.

Hard caps enforced in code: at most ONE message per invocation, at most
THREE findings inside it (the rest wait for the next run), and an
observation without affected evidence paths never announces. With nothing
eligible the command prints "nothing to announce" and exits 0.

This command must run inside a sandbox session: delivery writes to the spool
directory named by the ` + conversation.SpoolEnvVar + ` environment
variable. When that variable is unset the command fails with a clear error,
appends to ` + observer.LogFileName + `, and exits non-zero — it never
pretends to announce. Every other failure follows the same rule: log line,
non-zero exit, and no announced: flip, so the next run retries.
`

// runObserverAnnounce implements `spekk observer announce`.
func runObserverAnnounce(args []string) {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"help": {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})
	if flags.Bool("help") {
		fmt.Print(observerAnnounceUsage)
		return
	}
	if code := observer.Announce(observer.AnnounceOptions{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}); code != 0 {
		os.Exit(code)
	}
}

// observerDigestUsage is the help text for `spekk observer digest --help`.
const observerDigestUsage = `
spekk observer digest - Render the observation digest view

USAGE:
  spekk observer digest [--json]

OPTIONS:
  --json      Emit the digest as a JSON array
  --help, -h  Show this help message

The digest is a rendered view, not a committed file: the open findings that
are live claims, ranked by severity (high > medium > low, oldest first
within a severity), capped at 5. A live claim is the observation read from
the branch named after it, whose slug has not reached main — a copy another
branch inherited is not a claim, and a slug already on main is resolved.

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
spekk observer scan-check - Check drift against suppressions and the
cross-branch observation union before creating an observation

USAGE:
  spekk observer scan-check --type <type> --slug <slug> --affected <paths>

OPTIONS:
  --type <type>       code_spec_misalignment or outdated_specs (required)
  --slug <slug>       The slug the observation would be given (required)
  --affected <paths>  Comma-separated evidence file paths (required)
  --help, -h          Show this help message

The observer runs this before creating any observation. The result is JSON:

  {"result":"suppressed", ...}  an active .spekk/dont-flag.yaml entry (as
                                committed on main) matches an evidence path
                                or the slug; create nothing
  {"result":"covered", ...}     a live claim already covers the drift: the
                                same type and the same slug, read from the
                                branch named after it, with that slug not
                                yet on main; create nothing
  {"result":"clear","slug":..}  no coverage; create the observation with the
                                returned slug (a -YYYYMMDD suffix is added
                                when the plain slug is taken by an
                                observation already on main). A dated slug
                                and its plain form are the same finding, so
                                a recurrence still under review is covered.

--slug must be kebab-case, because it is the identity a later scan dedups
on and the name the observation file must carry. A slug the observation
format rejects would file an observation the index skips forever.

A malformed .spekk/dont-flag.yaml fails the check with a message naming the
offending entry — a broken suppression file is never treated as empty.

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

	// The slug is the dedup identity and the name the observation file must
	// carry, so it is checked here rather than at the file. A slug the
	// observation format rejects would pass this gate, file an observation
	// the union skips forever, and never dedup or announce — invisible,
	// with only a warning on a later command's stderr.
	if !observation.ValidSlug(slug) {
		fmt.Fprintf(stderr, "Error: --slug must be kebab-case (lowercase letters and digits, single hyphens), got %q\n", slug)
		return 1
	}

	// Suppression first: suppressed drift is invisible to the entire
	// downstream lifecycle — no observation, no branch, no index row, no
	// announcement — so it wins over dedup.
	entries, err := dontflag.LoadFromMain()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}
	if e := dontflag.Suppressed(entries, slug, affected, now); e != nil {
		printJSON(stdout, scanCheckResult{
			Result: "suppressed", Match: e.Match, Reason: e.Reason, By: e.By,
		})
		return 0
	}

	u, err := observation.LoadUnion()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}
	warn(stderr, u.Warnings)

	if covering := u.FindCovering(typ, slug); covering != nil {
		printJSON(stdout, scanCheckResult{
			Result: "covered", Slug: covering.Slug,
			Branch: observation.BranchFromRef(covering.Ref), Ref: covering.Ref,
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
