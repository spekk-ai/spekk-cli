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

// InstallCronConfig holds parsed install-cron options.
type InstallCronConfig struct {
	// LoopInterval is the observer default loop interval in minutes (default 30).
	LoopInterval int
	// ConsolidateInterval is the consolidation interval in minutes (default 360 = 6 h).
	ConsolidateInterval int
}

// isValidCronInterval reports whether minutes can be expressed exactly as a cron
// schedule. Only values ≤ 60 (sub-hourly */N) or exact multiples of 60 (hourly
// 0 */H) are accepted; values like 90 would produce a syntactically invalid or
// misinterpreted cron expression.
func isValidCronInterval(minutes int) bool {
	return minutes <= 60 || minutes%60 == 0
}

// ParseInstallCronFlags parses args into an InstallCronConfig.
func ParseInstallCronFlags(args []string) (InstallCronConfig, error) {
	cfg := InstallCronConfig{
		LoopInterval:        30,
		ConsolidateInterval: 360,
	}

	parsed := cli.ParseFlags(args, InstallCronFlags)

	if v := parsed.String("loopInterval"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("--loop-interval must be a positive number of minutes")
		}
		if !isValidCronInterval(n) {
			return cfg, fmt.Errorf("--loop-interval %d cannot be expressed as a valid cron schedule; use a value ≤ 60 or an exact multiple of 60 (e.g. 30, 60, 120, 360)", n)
		}
		cfg.LoopInterval = n
	}

	if v := parsed.String("consolidateInterval"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("--consolidate-interval must be a positive number of minutes")
		}
		if !isValidCronInterval(n) {
			return cfg, fmt.Errorf("--consolidate-interval %d cannot be expressed as a valid cron schedule; use a value ≤ 60 or an exact multiple of 60 (e.g. 60, 120, 360)", n)
		}
		cfg.ConsolidateInterval = n
	}

	return cfg, nil
}

// minutesToCron converts a positive interval in minutes to a cron schedule expression.
//
//   - Intervals ≤ 60 m:               */N * * * *
//   - Intervals that are exact multiples of 60:  0 */H * * *
func minutesToCron(minutes int) string {
	if minutes <= 60 {
		return fmt.Sprintf("*/%d * * * *", minutes)
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
  --loop-interval <minutes>        How often to run the observer loop (default: 30)
  --consolidate-interval <minutes> How often to consolidate observations (default: 360)
  --help, -h                       Show this help message

Installs two crontab entries:
  1. spekk observer              (default loop, runs every --loop-interval minutes)
  2. spekk observer consolidate  (consolidation, runs every --consolidate-interval minutes)

Intervals must be ≤ 60 or an exact multiple of 60 (e.g. 30, 60, 120, 360).

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
