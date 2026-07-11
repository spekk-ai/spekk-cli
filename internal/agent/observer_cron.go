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
		cfg.LoopInterval = n
	}

	if v := parsed.String("consolidateInterval"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("--consolidate-interval must be a positive number of minutes")
		}
		cfg.ConsolidateInterval = n
	}

	return cfg, nil
}

// minutesToCron converts a positive interval in minutes to a cron schedule expression.
//
//   - Intervals < 60 m:   */N * * * *
//   - Intervals that are exact multiples of 60:  0 */H * * *
//   - All other intervals (not an exact hour multiple):  */N * * * *
//     (cron implementations cap field values at 59; callers should prefer
//     round-number intervals)
func minutesToCron(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("*/%d * * * *", minutes)
	}
	if minutes%60 == 0 {
		hours := minutes / 60
		return fmt.Sprintf("0 */%d * * *", hours)
	}
	// Non-round interval: fall back to minute-field expression.
	return fmt.Sprintf("*/%d * * * *", minutes)
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
func readCrontab() (string, error) {
	out, err := exec.Command("crontab", "-l").Output()
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

// buildCronLines returns the two cron lines that install-cron would add, given
// the binary path and config. Exported for testing.
func buildCronLines(binary string, cfg InstallCronConfig) (loopLine, consolidateLine string) {
	loopLine = fmt.Sprintf("%s %s observer %s",
		minutesToCron(cfg.LoopInterval), binary, cronMarker)
	consolidateLine = fmt.Sprintf("%s %s observer consolidate %s",
		minutesToCron(cfg.ConsolidateInterval), binary, cronMarker)
	return
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

Lines are tagged with a comment so uninstall-cron can find and remove them later.
Run "spekk observer uninstall-cron" to remove the installed entries.
`)
		return
	}

	cfg, err := ParseInstallCronFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	binary := spekkBinaryPath()
	loopLine, consolidateLine := buildCronLines(binary, cfg)

	existing, err := readCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
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
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Installed crontab entries:")
	fmt.Println(" ", loopLine)
	fmt.Println(" ", consolidateLine)
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
