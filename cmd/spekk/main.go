// Command spekk is the CLI entry point for spec-driven development workflows.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	spekk "github.com/spekk-ai/spekk-cli"
	"github.com/spekk-ai/spekk-cli/internal/agent"
	"github.com/spekk-ai/spekk-cli/internal/cli"
	"github.com/spekk-ai/spekk-cli/internal/crossbranch"
	"github.com/spekk-ai/spekk-cli/internal/formatter"
	"github.com/spekk-ai/spekk-cli/internal/index"
	"github.com/spekk-ai/spekk-cli/internal/install"
	"github.com/spekk-ai/spekk-cli/internal/parser"
	"github.com/spekk-ai/spekk-cli/internal/sandbox"
	"github.com/spekk-ai/spekk-cli/internal/serve"
	"github.com/spekk-ai/spekk-cli/internal/show"
	"github.com/spekk-ai/spekk-cli/internal/status"
	"github.com/spekk-ai/spekk-cli/internal/update"
	"github.com/spekk-ai/spekk-cli/internal/validate"
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
	install.DefaultSkillFS = spekk.EmbeddedFS

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

	case "list":
		runList(args[1:])

	case "index":
		runIndex(args[1:])

	case "query":
		runQuery(args[1:])

	case "next":
		runParser(args[1:])

	case "coach":
		launchCoachAgent(args[1:])

	case "builder":
		launchBuilderAgent(args[1:])

	case "observer":
		// Go-native observer subcommands (digest, scan-check, announce)
		// route to deterministic code; anything else launches the agent.
		if len(args) > 1 {
			if run, ok := observerGoSubcommands[args[1]]; ok {
				run(args[2:])
				return
			}
		}
		launchObserverAgent(args[1:])

	case "loop":
		runLoop(args[1:])

	case "status":
		showStatus()

	case "validate":
		runValidate(args[1:])

	case "show":
		showSpekk(args[1:])

	case "serve":
		launchServe(args[1:])

	case "sandbox":
		launchSandbox(args[1:])

	case "conversation":
		runConversation(args[1:])

	case "prompt":
		runPrompt(args[1:])

	case "skill":
		runSkill(args[1:])

	case "install":
		runInstall(args[1:])

	case "uninstall":
		runUninstall(args[1:])

	case "skills":
		runSkills(args[1:])

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

	// Auto-rebuild the SQLite index silently if it is stale, absent, or built
	// against an older schema. EnsureFresh is the single freshness gate shared
	// with `spekk query`.
	repoRoot := filepath.Dir(specsDir)
	if rebuilt, err := index.EnsureFresh(specsDir, index.DBPath(repoRoot)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", index.FormatError(err))
		os.Exit(1)
	} else if rebuilt {
		_ = index.EnsureGitignored(repoRoot)
	}

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		out, _ := parser.FormatError(err.Error())
		fmt.Println(string(out))
		os.Exit(1)
	}
	if summary := result.WarningSummary(); summary != "" {
		fmt.Fprintln(os.Stderr, summary)
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

// runList implements the `spekk list` subcommand.
// Flags: --status <value>, --assertions-only, --specs-dir <path>,
//
//	--json, --tsv, --csv, --long / -l
func runList(args []string) {
	code := execList(args, os.Stdout, os.Stderr, "")
	if code != 0 {
		os.Exit(code)
	}
}

// execList is the testable core of runList.
// It writes to stdout/stderr and returns an exit code (0 = success).
// specsDir: if non-empty, skip findSpecsDir() and use this directly.
func execList(args []string, stdout, stderr io.Writer, specsDir string) int {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"status":         {Names: []string{"--status"}, Type: cli.StringFlag},
		"assertionsOnly": {Names: []string{"--assertions-only"}, Type: cli.BoolFlag},
		"specsDir":       {Names: []string{"--specs-dir"}, Type: cli.StringFlag},
		"json":           {Names: []string{"--json"}, Type: cli.BoolFlag},
		"tsv":            {Names: []string{"--tsv"}, Type: cli.BoolFlag},
		"csv":            {Names: []string{"--csv"}, Type: cli.BoolFlag},
		"long":           {Names: []string{"--long", "-l"}, Type: cli.BoolFlag},
		"crossBranch":    {Names: []string{"--cross-branch"}, Type: cli.BoolFlag},
		"branchFilter":   {Names: []string{"--branch-filter"}, Type: cli.StringFlag},
		"help":           {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Fprint(stdout, `
spekk list - List specs and assertions with optional filtering

USAGE:
  spekk list [OPTIONS]

OUTPUT FORMAT (default: table):
  --json        JSON output (flat assertion list; for jq pipelines)
  --tsv         Tab-separated values with lowercase header (for awk/cut/sort)
  --csv         RFC 4180 CSV with header row
  --long, -l    Add FILE column to table/TSV/CSV output (JSON always includes file)

FILTER OPTIONS:
  --status <value>      Filter by assertion status. Valid values:
                          not_started, in_progress, done, draft, failed
  --assertions-only     Accepted for backward compatibility (now a no-op; assertions are the default)
  --specs-dir <path>    Read specs from a specific directory (default: git root specs/)
  --help, -h            Show this help message

CROSS-BRANCH MODE:
  --cross-branch           List spec/assertion state across all branches
                           instead of assertions: one row per changed
                           (file, branch) pair with columns path, branch,
                           state (incoming_add, incoming_mod, conflict,
                           incoming_del), degraded, old_status, new_status.
                           Read-only; never touches the working tree.
  --branch-filter <glob>   Only include branches matching the glob (e.g. 'feat/*')

NOTES:
  The default output shows all assertions. Hierarchy output is available via spekk next --all.

EXAMPLES:
  spekk list
  spekk list --json
  spekk list --tsv
  spekk list --csv
  spekk list --long
  spekk list --status draft
  spekk list --status draft --tsv
  spekk list --assertions-only --csv
  spekk list --specs-dir /path/to/specs
  spekk list --cross-branch --json
  spekk list --cross-branch --branch-filter 'feat/*' --json
`)
		return 0
	}

	// Assertions are always the default; --assertions-only is accepted for
	// backward compatibility but is now a no-op.
	assertionsOnly := true
	_ = flags.Bool("assertionsOnly") // accepted but redundant
	useJSON := flags.Bool("json")
	useTSV := flags.Bool("tsv")
	useCSV := flags.Bool("csv")
	showFile := flags.Bool("long")
	statusVal := flags.String("status")

	// Reject mutually exclusive format flags.
	formatCount := 0
	for _, f := range []bool{useJSON, useTSV, useCSV} {
		if f {
			formatCount++
		}
	}
	if formatCount > 1 {
		fmt.Fprintln(stderr, "Error: --json, --tsv, and --csv are mutually exclusive; use at most one")
		return 1
	}

	if flags.Bool("crossBranch") {
		if statusVal != "" {
			fmt.Fprintln(stderr, "Error: --status does not apply to --cross-branch output")
			return 1
		}
		if specsDir != "" || flags.String("specsDir") != "" {
			// The cross-branch engine reads the git object store across
			// branches, not one specs directory, so it cannot honour a
			// different one. Refuse rather than return git-root results
			// under a path the caller chose: a silent substitution reports
			// success for something the command did not do.
			fmt.Fprintln(stderr, "Error: --specs-dir does not apply to --cross-branch output")
			return 1
		}
		return execListCrossBranch(flags.String("branchFilter"), stdout, stderr, useJSON, useTSV, useCSV)
	}
	if flags.String("branchFilter") != "" {
		fmt.Fprintln(stderr, "Error: --branch-filter requires --cross-branch")
		return 1
	}

	// Build opts before empty check — opts determines which header columns appear.
	opts := formatter.Options{
		ShowParent: assertionsOnly,
		ShowFile:   showFile,
	}

	if specsDir == "" {
		specsDir = flags.String("specsDir")
	}
	if specsDir == "" {
		specsDir = findSpecsDir()
	}

	result, err := parser.ParseAllSpecs(specsDir)
	if err != nil {
		out, _ := parser.FormatError(err.Error())
		fmt.Fprintln(stdout, string(out))
		return 1
	}
	if summary := result.WarningSummary(); summary != "" {
		fmt.Fprintln(stderr, summary)
	}

	// Apply status filter if requested.
	if statusVal != "" {
		filtered, filterErr := parser.FilterByStatus(result, statusVal)
		if filterErr != nil {
			// Use FormatError so machine-readable callers get consistent JSON output.
			out, _ := parser.FormatError(filterErr.Error())
			fmt.Fprintln(stdout, string(out))
			return 1
		}
		result = filtered
	}

	// Handle empty results with format-aware output.
	if len(result.Assertions) == 0 {
		switch {
		case useTSV:
			fmt.Fprint(stdout, formatter.FormatTSVHeader(opts))
		case useCSV:
			fmt.Fprint(stdout, formatter.FormatCSVHeader(opts))
		default:
			var out []byte
			if statusVal != "" {
				out, _ = parser.FormatEmptyFiltered(statusVal)
			} else {
				out, _ = parser.FormatEmpty()
			}
			fmt.Fprintln(stdout, string(out))
		}
		return 0
	}

	// --json: flat assertion JSON, in the same order as the default table.
	// The JSON is a superset: it also carries branch and depends_on, which the
	// table, TSV, and CSV columns do not show.
	if useJSON {
		out, err := parser.FormatAssertionsFlat(result)
		if err != nil {
			out2, _ := parser.FormatError(err.Error())
			fmt.Fprintln(stdout, string(out2))
			return 1
		}
		fmt.Fprintln(stdout, string(out))
		return 0
	}

	// Build rows for table/TSV/CSV formats.
	rows := listRows(result, assertionsOnly)

	switch {
	case useTSV:
		// FormatTSV owns its trailing newline; use Fprint to avoid a duplicate.
		fmt.Fprint(stdout, formatter.FormatTSV(rows, opts))
	case useCSV:
		// FormatCSV owns its trailing CRLF per RFC 4180; use Fprint to avoid
		// appending a bare LF after the final CRLF.
		fmt.Fprint(stdout, formatter.FormatCSV(rows, opts))
	default:
		// Default: human-readable table.
		fmt.Fprintln(stdout, formatter.FormatTable(rows, opts))
	}
	return 0
}

// execListCrossBranch renders the cross-branch classification — the same
// read-only engine behind `spekk show --cross-branch` — as machine-readable
// rows, one per changed (file, branch) pair. This is the surface an observer
// agent consumes; the HTML explorer stays the human surface.
func execListCrossBranch(branchFilter string, stdout, stderr io.Writer, useJSON, useTSV, useCSV bool) int {
	states, _, err := crossbranch.Classify(branchFilter)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %s\n", err)
		return 1
	}

	switch {
	case useJSON:
		out, jerr := crossbranch.FormatJSON(states)
		if jerr != nil {
			fmt.Fprintf(stderr, "Error: %s\n", jerr)
			return 1
		}
		fmt.Fprintln(stdout, out)
	case useTSV:
		fmt.Fprint(stdout, formatter.RenderTSV(crossbranch.OutputColumns, crossbranch.OutputRows(states)))
	case useCSV:
		fmt.Fprint(stdout, formatter.RenderCSV(crossbranch.OutputColumns, crossbranch.OutputRows(states)))
	default:
		if len(states) == 0 {
			fmt.Fprintln(stdout, "No cross-branch spec changes.")
			return 0
		}
		hdr := make([]string, len(crossbranch.OutputColumns))
		for i, c := range crossbranch.OutputColumns {
			hdr[i] = strings.ToUpper(c)
		}
		matrix := append([][]string{hdr}, crossbranch.OutputRows(states)...)
		fmt.Fprintln(stdout, formatter.RenderTable(matrix))
	}
	return 0
}

// listRows converts a ParseResult into formatter.Row slices for table/TSV/CSV,
// sorted by priority (ascending) then ID (alphabetical) — same order as
// FormatAssertionsFlat, so all output formats agree.
// When assertionsOnly is true, rows come from result.Assertions (with Parent set).
// Otherwise rows come from result.Specs (Parent not set).
func listRows(result *parser.ParseResult, assertionsOnly bool) []formatter.Row {
	var rows []formatter.Row
	if assertionsOnly {
		rows = make([]formatter.Row, 0, len(result.Assertions))
		for _, a := range result.Assertions {
			rows = append(rows, formatter.Row{
				ID:       a.ID,
				Status:   a.Status,
				Priority: a.Priority,
				Title:    a.Title,
				Parent:   a.Parent,
				File:     a.File,
			})
		}
	} else {
		rows = make([]formatter.Row, 0, len(result.Specs))
		for _, s := range result.Specs {
			rows = append(rows, formatter.Row{
				ID:       s.ID,
				Status:   s.Status,
				Priority: s.Priority,
				Title:    s.Title,
				File:     s.File,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Priority != rows[j].Priority {
			return rows[i].Priority < rows[j].Priority
		}
		return rows[i].ID < rows[j].ID
	})
	return rows
}

// runValidate implements the `spekk validate` subcommand.
func runValidate(args []string) {
	code := execValidate(args, os.Stdout, os.Stderr, "")
	if code != 0 {
		os.Exit(code)
	}
}

// validateUsage is the help text for `spekk validate`.
const validateUsage = `
spekk validate - Check every spec and assertion under specs/ against a fixed
set of invariants and exit non-zero if any is violated.

USAGE:
  spekk validate [OPTIONS]

OPTIONS:
  --specs-dir <path>   Read specs from a specific directory (default: git root specs/)
  --help, -h           Show this help message

Checks (all in one pass): frontmatter well-formedness (no silent skips — a
malformed spec or assertion file is a failure here, unlike "spekk next"),
parent resolution, depends-on validity (kebab-case, exists, no self-reference,
no cycles), no duplicate spec or assertion ids, lock state (only in_progress
may carry a locked-by, and it need not carry one), and parent specs carrying
no rolled-up status field (absent, or the literal value "draft").

Exit 0 and a one-line summary on stdout when everything is valid. Exit 1 and
one plain-text failure line per violation (file + problem), sorted by file
then message, when anything is invalid.

Warnings go to stderr and never change the exit code: a branch value that
matches no branch on an assertion that is not done, and a builder lock that is
stale.
`

// execValidate is the testable core of runValidate. It writes the report to
// stdout and returns an exit code (0 = pass, 1 = at least one failure).
// Warnings go to stderr, so stdout stays clean and diffable for CI and they
// never change the exit code.
// specsDir: if non-empty, skip findSpecsDir() and use this directly.
func execValidate(args []string, stdout, stderr io.Writer, specsDir string) int {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"specsDir": {Names: []string{"--specs-dir"}, Type: cli.StringFlag},
		"help":     {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Fprint(stdout, validateUsage)
		return 0
	}

	if specsDir == "" {
		specsDir = flags.String("specsDir")
	}
	if specsDir == "" {
		specsDir = findSpecsDir()
	}

	result, err := validate.Run(specsDir)
	if err != nil {
		fmt.Fprintf(stdout, "validate: %s\n", err.Error())
		return 1
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(stderr, "Warning: %s\n", w)
	}

	if result.Passed() {
		fmt.Fprintf(stdout, "validate: %d specs, %d assertions OK\n", result.SpecCount, result.AssertionCount)
		return 0
	}

	for _, f := range result.Failures {
		fmt.Fprintln(stdout, f.String())
	}
	return 1
}

// indexUsage is the help text for `spekk index`.
const indexUsage = `
spekk index - Build .spekk/index.db, a SQLite index of the spec tree, for use
by "spekk query". The Markdown files remain the source of truth; the database
is a derived artifact and is added to .gitignore automatically.

USAGE:
  spekk index [OPTIONS]

OPTIONS:
  --force              Drop and recreate all tables from scratch
  --specs-dir <path>   Read specs from a specific directory (default: git root specs/)
  --help, -h           Show this help message

The index is rebuilt automatically when it is stale, absent, or built against an
older schema, so running this by hand is rarely necessary.
`

// queryUsage is the help text for `spekk query`.
const queryUsage = `
spekk query "<sql>" - Run a read-only SELECT against .spekk/index.db and print
the result. The index is refreshed automatically first, so results always
reflect the current specs.

USAGE:
  spekk query "<sql>" [OPTIONS]

OPTIONS:
  --json               JSON array of objects
  --tsv                Tab-separated values with a lowercase header
  --csv                RFC 4180 CSV with a header row
  --help, -h           Show this help message

Only SELECT (and WITH ... SELECT) statements are permitted; the database is
opened read-only. Tables: specs(id, title, status, priority, branch, file),
assertions(id, parent_id, title, status, priority, branch, file),
depends_on(assertion_id, depends_on_id).
`

// runIndex implements the `spekk index` subcommand.
// It builds (or rebuilds) .spekk/index.db from the specs directory.
// Flags: --specs-dir <path>, --force
func runIndex(args []string) {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"specsDir": {Names: []string{"--specs-dir"}, Type: cli.StringFlag},
		"force":    {Names: []string{"--force"}, Type: cli.BoolFlag},
		"help":     {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Print(indexUsage)
		return
	}

	specsDir := flags.String("specsDir")
	if specsDir == "" {
		specsDir = findSpecsDir()
	}

	// Derive repo root from the specs directory.
	repoRoot := filepath.Dir(specsDir)
	dbPath := index.DBPath(repoRoot)

	stats, err := index.BuildIndex(specsDir, dbPath, flags.Bool("force"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	warn(os.Stderr, stats.Warnings)

	// Ensure .spekk/index.db is in .gitignore.
	if err := index.EnsureGitignored(repoRoot); err != nil {
		// Non-fatal: warn but continue.
		fmt.Fprintf(os.Stderr, "warning: could not update .gitignore: %s\n", err)
	}

	fmt.Printf("Indexed %d specs, %d assertions, %d observations.\n", stats.Specs, stats.Assertions, stats.Observations)
}

// runQuery implements the `spekk query "<sql>"` subcommand.
// Executes a SELECT-only SQL query against .spekk/index.db and prints results.
// Flags: --json, --tsv, --csv
func runQuery(args []string) {
	flags := cli.ParseFlags(args, cli.FlagSet{
		"json": {Names: []string{"--json"}, Type: cli.BoolFlag},
		"tsv":  {Names: []string{"--tsv"}, Type: cli.BoolFlag},
		"csv":  {Names: []string{"--csv"}, Type: cli.BoolFlag},
		"help": {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Print(queryUsage)
		return
	}

	// Extract the SQL positional argument (first non-flag arg).
	knownFlags := map[string]bool{
		"--json": true, "--tsv": true, "--csv": true, "--help": true, "-h": true,
	}
	var sqlStr string
	for _, arg := range args {
		if !knownFlags[arg] {
			sqlStr = arg
			break
		}
	}

	if sqlStr == "" {
		fmt.Fprintln(os.Stderr, `error: missing SQL argument`)
		fmt.Fprintln(os.Stderr, `usage: spekk query "<sql>"`)
		os.Exit(1)
	}

	// Locate the repo root and the index DB.
	specsDir := findSpecsDir()
	repoRoot := filepath.Dir(specsDir)
	dbPath := index.DBPath(repoRoot)

	// Refresh the index before querying so results reflect the current specs
	// (and rebuild if it was built against an older schema). Same freshness gate
	// as `spekk next`.
	if rebuilt, err := index.EnsureFresh(specsDir, dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: could not refresh index: %s\n", err)
		os.Exit(1)
	} else if rebuilt {
		_ = index.EnsureGitignored(repoRoot)
	}

	result, err := index.RunQuery(dbPath, sqlStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	// Format and print output.
	var out string
	switch {
	case flags.Bool("json"):
		out = index.FormatQueryJSON(result)
	case flags.Bool("tsv"):
		out = index.FormatQueryTSV(result)
	case flags.Bool("csv"):
		out = index.FormatQueryCSV(result)
	default:
		out = index.FormatQueryTable(result)
	}

	fmt.Println(out)
}

// repoRoot returns the root of the repository that holds the working directory,
// and the working directory itself when there is no repository. A project is the
// repository, not the directory the user happens to stand in, so a command run
// from a subdirectory must find the same project files as one run from the root.
func repoRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return root
		}
	}
	wd, _ := os.Getwd()
	return wd
}

// findSpecsDir locates the specs/ directory at the repository root.
func findSpecsDir() string {
	return filepath.Join(repoRoot(), "specs")
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
  create      Create a sandbox, or register a machine you already have
  list        List all sandboxes
  status      Show status of a sandbox
  ssh         SSH into a sandbox
  destroy     Destroy a sandbox, or deregister a machine you own
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
		// Status reads more than the provider knows, so a provider it
		// cannot build is a warning, not a failure. Before this command
		// needed an API token at all, it still printed the stored fields
		// and the SSH checks; it keeps doing that.
		// Status reports a missing sandbox itself, so only a real
		// provider problem is worth a warning here.
		p, err := sandbox.ProviderForName(subArgs[0])
		if err != nil && !errors.Is(err, sandbox.ErrSandboxNotFound) {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", err)
		}
		if err := sandbox.Status(p, subArgs[0]); err != nil {
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
		p, err := sandbox.ProviderForName(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		if err := sandbox.Destroy(p, name, force); err != nil {
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
		"name":     {Names: []string{"--name"}, Type: cli.StringFlag},
		"provider": {Names: []string{"--provider"}, Type: cli.StringFlag},
		"region":   {Names: []string{"--region"}, Type: cli.StringFlag},
		"size":     {Names: []string{"--size"}, Type: cli.StringFlag},
		"project":  {Names: []string{"--project"}, Type: cli.StringFlag},
		"vpc":      {Names: []string{"--vpc"}, Type: cli.StringFlag},
		"ip":       {Names: []string{"--ip"}, Type: cli.StringFlag},
		"ssh-key":  {Names: []string{"--ssh-key"}, Type: cli.StringFlag},
		"ssh-user": {Names: []string{"--ssh-user"}, Type: cli.StringFlag},
		"auth":     {Names: []string{"--auth"}, Type: cli.StringFlag},
		"help":     {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
	})

	if flags.Bool("help") {
		fmt.Print(`
spekk sandbox create - Create a new sandbox

USAGE:
  spekk sandbox create --name <name> [options]

OPTIONS:
  --name <name>          Sandbox name (required)
  --provider <provider>  digitalocean, or none for a machine you already
                         have (inferred from --ip / --ssh-key)

  DigitalOcean options:
  --region <region>      DigitalOcean region (default: nyc1)
  --size <size>          Droplet size slug (default: s-2vcpu-4gb)
  --project <project>    Assign to a DigitalOcean project (name or UUID)
  --vpc <uuid>           Place droplet in a specific DigitalOcean VPC

  Options for a machine you already have:
  --ip <address>         Address of the machine
  --ssh-key <path>       Private key that reaches it as the login user
  --ssh-user <user>      SSH login user (default: root); a non-root user must
                         have passwordless sudo, as AWS Ubuntu AMIs do

  Model credential:
  --auth <mode>          How the agent authenticates Claude: bedrock (default,
                         bills through the AWS Bedrock API) or subscription
                         (uses CLAUDE_CODE_OAUTH_TOKEN from your environment,
                         minted by "claude setup-token")

  spekk does not provision a machine it did not create. The machine must
  already carry /opt/spekk/.provisioned; spekk then injects credentials
  and deploys the agent onto it.
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

	auth, err := sandbox.ParseAuthMode(flags.String("auth"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	setFlags := map[string]bool{
		"--region":   flags.String("region") != "",
		"--size":     flags.String("size") != "",
		"--vpc":      flags.String("vpc") != "",
		"--project":  flags.String("project") != "",
		"--ip":       flags.String("ip") != "",
		"--ssh-key":  flags.String("ssh-key") != "",
		"--ssh-user": flags.String("ssh-user") != "",
	}

	// Naming an existing machine is what says there is nothing to create.
	providerName, err := sandbox.ResolveProviderName(flags.String("provider"), sandbox.NamesExistingMachine(setFlags))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Validate provider-specific flags.
	if err := sandbox.ValidateProviderFlags(providerName, setFlags); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	p, err := sandbox.ProviderByName(providerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	opts := sandbox.CreateOptions{
		Name:    flags.String("name"),
		Region:  flags.String("region"),
		Size:    flags.String("size"),
		Project: flags.String("project"),
		VPC:     flags.String("vpc"),
		IP:      flags.String("ip"),
		SSHKey:  flags.String("ssh-key"),
		SSHUser: flags.String("ssh-user"),
		Auth:    auth,
	}
	if err := sandbox.Create(p, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// runInstallSkill parses arguments for `spekk install <agent> <skill>`.
func runInstallSkill(args []string) {
	opts, err := install.ParseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		fmt.Fprint(os.Stderr, install.UsageText)
		os.Exit(1)
	}
	if opts.Help {
		fmt.Print(install.UsageText)
		return
	}

	home, _ := os.UserHomeDir()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if opts.List != "" {
		skills, err := install.ListRemoteSkills(opts.List, cwd, home, install.FetchListRaw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(install.FormatList(opts.List, skills))
		return
	}

	skill := opts.Skill
	if opts.Source != "" {
		skill, err = install.ResolveSourceSkill(opts.Source, opts.Skill)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	} else if skill == "" {
		fmt.Fprintf(os.Stderr, "Error: missing <skill> argument\n\n")
		fmt.Fprint(os.Stderr, install.UsageText)
		os.Exit(1)
	}

	msg, err := install.PerformInstall(install.InstallRequest{
		Cwd:      cwd,
		HomeDir:  home,
		Scope:    opts.Scope,
		Agent:    opts.Agent,
		Skill:    skill,
		Source:   opts.Source,
		Force:    opts.Force,
		FetchFn:  install.FetchSkill,
		FetchURL: install.FetchURL,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Println(msg)
}

// runUninstall handles `spekk uninstall <agent> <skill> [--global|--local]`.
func runUninstall(args []string) {
	opts, err := install.ParseUninstallArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		fmt.Fprint(os.Stderr, install.UninstallUsageText)
		os.Exit(1)
	}
	if opts.Help {
		fmt.Print(install.UninstallUsageText)
		return
	}

	home, _ := os.UserHomeDir()
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	removed, err := install.Uninstall(cwd, home, opts.Scope, opts.Agent, opts.Skill)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("removed %s/%s ← %s\n", opts.Agent, opts.Skill, removed)
}

// runSkills handles `spekk skills <subcommand>`. Today the only subcommand
// is `list`, which wraps SkillResolver.ListSkills and prints each entry's
// name and source directory (or "(embedded)").
func runSkills(args []string) {
	if len(args) == 0 {
		fmt.Print(install.SkillsUsageText)
		return
	}

	switch args[0] {
	case "help", "--help", "-h":
		fmt.Print(install.SkillsUsageText)
		return
	case "list":
		runSkillsList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown skills subcommand: %s\n\n", args[0])
		fmt.Fprint(os.Stderr, install.SkillsUsageText)
		os.Exit(1)
	}
}

// runSkillsList handles `spekk skills list <agent>`.
func runSkillsList(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: missing <agent> argument\n\n")
		fmt.Fprint(os.Stderr, install.SkillsUsageText)
		os.Exit(1)
	}
	agent := args[0]
	if err := install.ValidateSkillsAgent(agent); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	r := cli.NewSkillResolver(findInstallDir())
	skills := r.ListSkills(agent)
	fmt.Print(install.FormatSkillsList(agent, skills))
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

	replaced, err := update.Run(checkOnly)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	home, project := scopeForInstall()
	reportAfterUpdate(os.Stderr, replaced, home, project)
}

// reportAfterUpdate says which report follows a self-update attempt. A stale
// check after a real replacement would compare against this process's embedded
// content, which the new binary has already replaced, so that case names the
// install commands instead.
func reportAfterUpdate(w io.Writer, replaced bool, home, cwd string) {
	if replaced {
		reportReinstall(w, home, cwd)
		return
	}
	reportStale(w, home, cwd)
}

// warnCheckFailed reports that a scan of the install locations did not finish.
// A check that cannot run must say so rather than read as "nothing is stale".
func warnCheckFailed(w io.Writer, err error) {
	fmt.Fprintf(w, "\nwarning: could not check the installed spekk files: %s\n", err)
}

// reportReinstall names the install targets that have spekk files on disk and
// asks the user to run the install again. A role shim reads its instructions
// from the binary at session start. The spekk-dev-loop skill carries its content
// in the file, so a new binary can hold content the installed file does not.
func reportReinstall(w io.Writer, home, cwd string) {
	reportReinstallWith(w, func() ([]string, error) {
		return install.InstalledTargets(home, cwd)
	})
}

// reportReinstallWith is reportReinstall with the scan as a parameter, so a test
// can drive the case where the scan does not finish.
func reportReinstallWith(w io.Writer, scan func() ([]string, error)) {
	installed, err := scan()
	if err != nil {
		warnCheckFailed(w, err)
		return
	}
	if len(installed) == 0 {
		return
	}
	fmt.Fprintln(w, "\nThe installed spekk files come from the binary. Re-run:")
	for _, cmd := range installed {
		fmt.Fprintln(w, "  "+cmd)
	}
}

// reportStale reports every installed file that this binary no longer writes or
// no longer matches. It changes no file.
func reportStale(w io.Writer, home, cwd string) {
	reportStaleWith(w, func() ([]install.StaleFile, error) {
		return install.CheckStale(home, cwd, spekk.EmbeddedFS)
	})
}

// reportStaleWith is reportStale with the check as a parameter, so a test can
// drive the case where the check does not finish.
func reportStaleWith(w io.Writer, check func() ([]install.StaleFile, error)) {
	stale, err := check()
	if err != nil {
		warnCheckFailed(w, err)
		return
	}
	if len(stale) == 0 {
		return
	}
	fmt.Fprintln(w, "\nwarning: some installed spekk files do not match this version of spekk:")
	for _, s := range stale {
		if s.Reason == install.StaleSymlink {
			fmt.Fprintf(w, "  %s %s (%s; %s)\n", s.Path, s.Reason, s.LinkTarget, s.Remedy())
			continue
		}
		fmt.Fprintf(w, "  %s %s (%s)\n", s.Path, s.Reason, s.Remedy())
	}
}

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
If specs/README.md already exists, its managed block is regenerated in
place (well-formed fence), appended (no fence yet — e.g. a legacy or
hand-written README), or recovered (a corrupt fence). Human prose outside
the managed block is always left untouched.
`)
			return
		}
	}

	specsDir := findSpecsDir()
	if info, err := os.Stat(specsDir); err == nil && info.IsDir() {
		fmt.Printf("specs/ already exists at %s — you're set.\n", specsDir)

		readmePath := filepath.Join(specsDir, "README.md")
		switch existing, err := os.ReadFile(readmePath); {
		case err == nil:
			if updated, changed := regenerateReadmeContent(string(existing)); changed {
				if err := os.WriteFile(readmePath, []byte(updated), 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "Error: writing %s: %s\n", readmePath, err)
					os.Exit(1)
				}
				fmt.Println("Refreshed the managed block in specs/README.md.")
			}
		case os.IsNotExist(err):
			// specs/ predates this feature (or the README was removed): create it.
			if err := os.WriteFile(readmePath, []byte(renderSpecsReadme()), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: writing %s: %s\n", readmePath, err)
				os.Exit(1)
			}
			fmt.Println("Created specs/README.md.")
		}

		fmt.Println(`Run "spekk coach" to draft a spec, or "spekk next" to see what's ready.`)
		return
	}

	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: creating %s: %s\n", specsDir, err)
		os.Exit(1)
	}
	readmePath := filepath.Join(specsDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(renderSpecsReadme()), 0o644); err != nil {
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

// runInstallTargets installs the observer agent and the spekk skills into a host
// coding assistant.
func runInstallTargets(args []string) {
	usage := `
spekk install - Install spekk into a coding assistant

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

This writes the observer as an agent, and the coach, the builder, and the
dev-loop as skills. Each role file is thin: it fetches its full instructions
from this binary with "spekk prompt <role>", so it does not go stale.
Updating spekk updates every installed file. "spekk install" also removes an
old coach or builder agent shim from a previous version.

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

	res, err := install.Install(targetInstallOptions(flags.String("target"), flags.Bool("project")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}
	for _, path := range res.Written {
		fmt.Println("installed:", path)
	}
	for _, path := range res.Removed {
		fmt.Println("removed:", path)
	}
}

// scopeForInstall returns the two directories a target install and its report
// both work in: the user's home, and the repository that holds the working
// directory. One function so the install and the report cannot disagree about
// where the project is.
func scopeForInstall() (home, project string) {
	home, _ = os.UserHomeDir()
	return home, repoRoot()
}

// targetInstallOptions builds the options for one target install. A project
// install works on the repository, so the scope is the repository root.
func targetInstallOptions(target string, project bool) install.Options {
	home, projectDir := scopeForInstall()
	return install.Options{
		Target:  target,
		Project: project,
		HomeDir: home,
		Cwd:     projectDir,
	}
}

// runInstall dispatches to the skill installer or the agent-shim installer
// depending on whether --target is present.
func runInstall(args []string) {
	for _, a := range args {
		if a == "--target" || a == "-t" {
			runInstallTargets(args)
			return
		}
		if strings.HasPrefix(a, "--target=") || strings.HasPrefix(a, "-t=") {
			runInstallTargets(args)
			return
		}
	}
	runInstallSkill(args)
}

// helpText is the top-level help printed for `spekk help`. Exposed as a
// constant so tests can verify the commands table without exec'ing the binary.
const helpText = `
spekk - Spec-driven development CLI

USAGE:
  spekk [COMMAND]

COMMANDS:
  init      Set up a project for spec-driven development (creates specs/)
  list      List specs and assertions with optional --status filter and --assertions-only flat output
  index     Build .spekk/index.db SQLite index from specs/ (use --force to rebuild from scratch)
  query     Execute a SELECT query against .spekk/index.db (supports --json, --tsv, --csv)
  show      Generate and display spec explorer web interface (-w to watch)
  status    Show comprehensive overview of all specs and assertions
  validate  Check specs/ against a fixed set of invariants; exit non-zero on any violation
  serve     Start WebSocket server for browser extension (--port, --host)
  coach     Launch the Coach Agent to create and refine specs
  builder   Launch the Builder Agent to implement specs
  observer  Launch the Observer Agent to find spec-code drift
  sandbox   Manage cloud sandbox environments (create, list, status, ssh, destroy, deploy)
  conversation  Request a conversation on the connected chat surface (open)
  install   Install a skill for an agent (coach/builder/observer)
  uninstall Remove an installed skill from local or global scope
  skills    Inspect skills available to an agent (list)
  loop      Run orchestration workflows (builder/coach loops)
  prompt    Print an agent's resolved prompt to stdout
  skill     List and print agent skills (list, show)
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
`

// printHelp displays the help text with all available commands.
func printHelp() {
	fmt.Print(helpText)
}
