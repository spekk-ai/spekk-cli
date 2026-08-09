package agent

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/cli"
)

// cronMarker is appended to every crontab line managed by spekk observer.
// uninstall-cron identifies lines to remove by looking for this string.
const cronMarker = "# spekk-observer"

// InstallCronFlags defines flags accepted by install-cron.
var InstallCronFlags = cli.FlagSet{
	"loopInterval":        {Names: []string{"--loop-interval"}, Type: cli.StringFlag},
	"consolidateInterval": {Names: []string{"--consolidate-interval"}, Type: cli.StringFlag},
	"help":                {Names: []string{"--help", "-h"}, Type: cli.BoolFlag},
}

// Default schedule: one scan a day, and one consolidation a day after it.
//
// A scan used to run every 30 minutes because a session did not end on its
// own — the schedule was what bounded the run. A run now ends by itself, so
// the schedule does one job only: it sets how often the observer looks, and
// therefore how many observations arrive. Once a day is a rate a person can
// keep up with; every 30 minutes is not.
//
// Consolidation follows the scan. Curating several times a day over a set
// that changes once a day spends an agent session on nothing.
const (
	defaultLoopInterval        = 1440 // 24 h
	defaultConsolidateInterval = 1440 // 24 h
)

// maxCronInterval is one day. Above it, the hourly form 0 */H * * * runs off
// the end of the hour field (0-23) and either fails or silently collapses to
// midnight, so a longer interval cannot be expressed and is refused instead.
const maxCronInterval = 1440

// InstallCronConfig holds parsed install-cron options.
type InstallCronConfig struct {
	// LoopInterval is how often the observer scans, in minutes.
	LoopInterval int
	// ConsolidateInterval is how often consolidation runs, in minutes.
	ConsolidateInterval int
}

// isValidCronInterval reports whether minutes can be expressed exactly as a cron
// schedule. Only values ≤ 60 (sub-hourly */N), exact multiples of 60 up to a
// day (hourly 0 */H), and a day itself are accepted; values like 90 would
// produce a syntactically invalid or misinterpreted cron expression.
func isValidCronInterval(minutes int) bool {
	if minutes > maxCronInterval {
		return false
	}
	return minutes <= 60 || minutes%60 == 0
}

// ParseInstallCronFlags parses args into an InstallCronConfig.
func ParseInstallCronFlags(args []string) (InstallCronConfig, error) {
	cfg := InstallCronConfig{
		LoopInterval:        defaultLoopInterval,
		ConsolidateInterval: defaultConsolidateInterval,
	}

	parsed := cli.ParseFlags(args, InstallCronFlags)

	if v := parsed.String("loopInterval"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("--loop-interval must be a positive number of minutes")
		}
		if !isValidCronInterval(n) {
			return cfg, fmt.Errorf("--loop-interval %d cannot be expressed as a valid cron schedule; use a value ≤ 60, or an exact multiple of 60 up to 1440 (e.g. 30, 60, 360, 1440)", n)
		}
		cfg.LoopInterval = n
	}

	if v := parsed.String("consolidateInterval"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("--consolidate-interval must be a positive number of minutes")
		}
		if !isValidCronInterval(n) {
			return cfg, fmt.Errorf("--consolidate-interval %d cannot be expressed as a valid cron schedule; use a value ≤ 60, or an exact multiple of 60 up to 1440 (e.g. 60, 360, 1440)", n)
		}
		cfg.ConsolidateInterval = n
	}

	return cfg, nil
}

// minutesToCron converts a positive interval in minutes to a cron schedule expression.
//
//   - Intervals < 60 m:                          */N * * * *
//   - One hour:                                  0 * * * *
//   - One day:                                   0 0 * * *
//   - Other exact multiples of 60:               0 */H * * *
//
// An hour and a day are written out rather than left to the stepped forms.
// `*/60 * * * *` steps by 60 across a minute field that only reaches 59, and
// `0 */24 * * *` steps by 24 across an hour field that only reaches 23:
// strict crons reject both, and lax crons quietly reduce them to the first
// value — the right time by accident, from an expression that does not mean
// it. An hour matters as much as a day here, because the error message for a
// rejected interval offers 60 as an example.
func minutesToCron(minutes int) string {
	if minutes == 60 {
		return "0 * * * *"
	}
	if minutes < 60 {
		return fmt.Sprintf("*/%d * * * *", minutes)
	}
	if minutes == maxCronInterval {
		return "0 0 * * *"
	}
	// Only exact multiples of 60 reach here (validated by ParseInstallCronFlags).
	hours := minutes / 60
	return fmt.Sprintf("0 */%d * * *", hours)
}

// spekkBinaryPath returns the absolute path of the running spekk binary,
// following symlinks. Falls back to "spekk" (relies on PATH) on error.
func spekkBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "spekk"
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return resolved
}

// readCrontab reads the current user crontab. Returns an empty string if no
// crontab exists yet (crontab -l exits non-zero on empty crontab on many
// systems and prints "no crontab for <user>").
// LC_ALL=C is set so the "no crontab" sentinel is matched regardless of locale.
func readCrontab() (string, error) {
	cmd := exec.Command("crontab", "-l")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		// Treat "no crontab" as an empty one.
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "no crontab") {
				return "", nil
			}
		}
		return "", fmt.Errorf("reading crontab: %w", err)
	}
	return string(out), nil
}

// writeCrontab replaces the user crontab with content.
func writeCrontab(content string) error {
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = bytes.NewBufferString(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("writing crontab: %w\n%s", err, out)
	}
	return nil
}

// buildCronLines returns the two cron lines that install-cron would add. The
// binary and claude paths are single-quoted so paths containing spaces survive.
// Each line changes into the project directory before running, passes the
// absolute claude path via --claude-path (so cron's limited PATH is irrelevant),
// runs the observer in headless mode (--headless → claude -p, no TTY required),
// and redirects output to a log file under the project directory.
// Overlap prevention is handled in Go (syscall.Flock inside LaunchHeadless),
// not via a shell flock wrapper, so no flock appears in the cron line.
func buildCronLines(binary, claudePath, projectDir string, cfg InstallCronConfig) (loopLine, consolidateLine string) {
	loopLine = fmt.Sprintf(
		"%s cd '%s' && '%s' observer --headless --claude-path '%s' >> '%s/.spekk/observer.log' 2>&1 %s",
		minutesToCron(cfg.LoopInterval),
		projectDir,
		binary,
		claudePath,
		projectDir,
		cronMarker,
	)
	consolidateLine = fmt.Sprintf(
		"%s cd '%s' && '%s' observer consolidate --headless --claude-path '%s' >> '%s/.spekk/observer-consolidate.log' 2>&1 %s",
		minutesToCron(cfg.ConsolidateInterval),
		projectDir,
		binary,
		claudePath,
		projectDir,
		cronMarker,
	)
	return
}

// doInstallCron implements the core install-cron logic and returns an error
// instead of calling os.Exit, making it testable.
func doInstallCron(args []string) error {
	cfg, err := ParseInstallCronFlags(args)
	if err != nil {
		return err
	}

	// Resolve claude's absolute path at install time. Cron runs with a minimal
	// PATH that typically does not include claude's install location, so a bare
	// "claude" lookup would fail silently. Baking the absolute path into the
	// cron line ensures the entry is functional.
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("cannot find 'claude' binary: %w\nInstall Claude Code first: https://claude.ai/code", err)
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	// Create .spekk/ before writing the crontab. The installed lines redirect
	// into <project-dir>/.spekk/ and lock files live there too; the shell
	// redirect would fail if the directory does not exist.
	if err := os.MkdirAll(filepath.Join(projectDir, ".spekk"), 0o755); err != nil {
		return fmt.Errorf("creating .spekk directory: %w", err)
	}

	binary := spekkBinaryPath()
	loopLine, consolidateLine := buildCronLines(binary, claudePath, projectDir, cfg)

	existing, err := readCrontab()
	if err != nil {
		return err
	}

	// Remove any previously installed spekk-observer lines so re-running
	// install-cron is idempotent (it replaces old entries, not duplicates).
	cleaned := removeCronMarkerLines(existing)

	// Ensure file ends with a newline before appending.
	if cleaned != "" && !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}

	newContent := cleaned + loopLine + "\n" + consolidateLine + "\n"

	if err := writeCrontab(newContent); err != nil {
		return err
	}

	fmt.Println("Installed crontab entries:")
	fmt.Println(" ", loopLine)
	fmt.Println(" ", consolidateLine)
	return nil
}

// RunObserverInstallCron implements `spekk observer install-cron`.
func RunObserverInstallCron(args []string) {
	if hasHelp(args) {
		fmt.Print(`
spekk observer install-cron - Install crontab entries for automatic observation

USAGE:
  spekk observer install-cron [OPTIONS]

OPTIONS:
  --loop-interval <minutes>        How often to scan (default: 1440, once a day)
  --consolidate-interval <minutes> How often to consolidate observations (default: 1440)
  --help, -h                       Show this help message

Installs two crontab entries:
  1. spekk observer              (a scan, every --loop-interval minutes)
  2. spekk observer consolidate  (consolidation, every --consolidate-interval minutes)

A run files one observation and then stops, so the schedule sets how many
arrive: one a day by default. Set --loop-interval for a different rate.

Intervals must be ≤ 60, or an exact multiple of 60 up to 1440 (one day).

Lines are tagged with a comment so uninstall-cron can find and remove them later.
Run "spekk observer uninstall-cron" to remove the installed entries.
`)
		return
	}

	if err := doInstallCron(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

// RunObserverUninstallCron implements `spekk observer uninstall-cron`.
func RunObserverUninstallCron(args []string) {
	if hasHelp(args) {
		fmt.Print(`
spekk observer uninstall-cron - Remove crontab entries installed by install-cron

USAGE:
  spekk observer uninstall-cron [OPTIONS]

OPTIONS:
  --help, -h  Show this help message

Removes only the crontab lines that "spekk observer install-cron" added.
The rest of your crontab is left untouched.
`)
		return
	}

	existing, err := readCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	removed := cronMarkerLines(existing)
	cleaned := removeCronMarkerLines(existing)

	if len(removed) == 0 {
		fmt.Println("No spekk observer crontab entries found.")
		return
	}

	if err := writeCrontab(cleaned); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Removed crontab entries:")
	for _, line := range removed {
		fmt.Println(" ", line)
	}
}

// removeCronMarkerLines returns the crontab content with all spekk-observer
// lines stripped. Trailing blank lines are also cleaned up.
func removeCronMarkerLines(crontab string) string {
	var kept []string
	for _, line := range strings.Split(crontab, "\n") {
		if strings.Contains(line, cronMarker) {
			continue
		}
		kept = append(kept, line)
	}
	result := strings.Join(kept, "\n")
	// Trim trailing whitespace/newlines, then add a single trailing newline
	// if there's still content — so the file stays clean.
	result = strings.TrimRight(result, "\n ")
	if result != "" {
		result += "\n"
	}
	return result
}

// cronMarkerLines returns all lines in crontab that contain the spekk marker.
func cronMarkerLines(crontab string) []string {
	var found []string
	for _, line := range strings.Split(crontab, "\n") {
		if strings.Contains(line, cronMarker) {
			found = append(found, line)
		}
	}
	return found
}
