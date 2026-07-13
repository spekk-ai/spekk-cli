package agent

import (
	"strings"
	"testing"
)

// TestParseInstallCronFlags_Defaults checks that defaults are 30 and 360 minutes.
func TestParseInstallCronFlags_Defaults(t *testing.T) {
	cfg, err := ParseInstallCronFlags([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LoopInterval != 30 {
		t.Errorf("expected LoopInterval=30, got %d", cfg.LoopInterval)
	}
	if cfg.ConsolidateInterval != 360 {
		t.Errorf("expected ConsolidateInterval=360, got %d", cfg.ConsolidateInterval)
	}
}

func TestParseInstallCronFlags_Custom(t *testing.T) {
	cfg, err := ParseInstallCronFlags([]string{"--loop-interval", "15", "--consolidate-interval", "120"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LoopInterval != 15 {
		t.Errorf("expected 15, got %d", cfg.LoopInterval)
	}
	if cfg.ConsolidateInterval != 120 {
		t.Errorf("expected 120, got %d", cfg.ConsolidateInterval)
	}
}

func TestParseInstallCronFlags_InvalidLoop(t *testing.T) {
	_, err := ParseInstallCronFlags([]string{"--loop-interval", "notanumber"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "loop-interval") {
		t.Errorf("error should mention loop-interval: %v", err)
	}
}

func TestParseInstallCronFlags_ZeroConsolidate(t *testing.T) {
	_, err := ParseInstallCronFlags([]string{"--consolidate-interval", "0"})
	if err == nil {
		t.Fatal("expected error for zero interval")
	}
}

// TestParseInstallCronFlags_IntervalValidation checks that non-cron-expressible
// intervals are rejected at parse time and valid ones are accepted.
func TestParseInstallCronFlags_IntervalValidation(t *testing.T) {
	rejectCases := []struct {
		flag  string
		value string
	}{
		{"--loop-interval", "90"},  // >60, not a multiple of 60
		{"--loop-interval", "150"}, // >60, not a multiple of 60
		{"--loop-interval", "75"},  // >60, not a multiple of 60
		{"--consolidate-interval", "90"},
		{"--consolidate-interval", "150"},
	}
	for _, tc := range rejectCases {
		_, err := ParseInstallCronFlags([]string{tc.flag, tc.value})
		if err == nil {
			t.Errorf("expected error for %s %s (non-expressible cron interval)", tc.flag, tc.value)
		}
	}

	acceptCases := []struct {
		flag  string
		value string
	}{
		{"--loop-interval", "30"},
		{"--loop-interval", "60"},
		{"--loop-interval", "1"},
		{"--consolidate-interval", "120"},
		{"--consolidate-interval", "360"},
	}
	for _, tc := range acceptCases {
		_, err := ParseInstallCronFlags([]string{tc.flag, tc.value})
		if err != nil {
			t.Errorf("unexpected error for %s %s: %v", tc.flag, tc.value, err)
		}
	}
}

// TestMinutesToCron covers sub-hour and exact-hour cases.
func TestMinutesToCron(t *testing.T) {
	cases := []struct {
		minutes int
		want    string
	}{
		{30, "*/30 * * * *"},
		{15, "*/15 * * * *"},
		{60, "*/60 * * * *"},
		{360, "0 */6 * * *"},
		{120, "0 */2 * * *"},
	}
	for _, tc := range cases {
		got := minutesToCron(tc.minutes)
		if got != tc.want {
			t.Errorf("minutesToCron(%d) = %q, want %q", tc.minutes, got, tc.want)
		}
	}
}

// TestBuildCronLines verifies that generated lines contain the required elements:
// quoted binary path, absolute claude path via --claude-path, cd into project dir,
// headless flag, log file redirect, cron marker, and no shell flock.
func TestBuildCronLines(t *testing.T) {
	cfg := InstallCronConfig{LoopInterval: 30, ConsolidateInterval: 360}
	projectDir := "/home/user/my project"           // space in path intentional
	claudePath := "/home/user/.claude/local/claude" // absolute path with no PATH dependency
	loop, consolidate := buildCronLines("/usr/local/bin/spekk", claudePath, projectDir, cfg)

	// Schedule
	if !strings.Contains(loop, "*/30 * * * *") {
		t.Errorf("loop line missing schedule: %q", loop)
	}
	if !strings.Contains(consolidate, "0 */6 * * *") {
		t.Errorf("consolidate line missing schedule: %q", consolidate)
	}

	// cd into quoted project dir
	if !strings.Contains(loop, "cd '/home/user/my project'") {
		t.Errorf("loop line missing cd into project dir: %q", loop)
	}
	if !strings.Contains(consolidate, "cd '/home/user/my project'") {
		t.Errorf("consolidate line missing cd into project dir: %q", consolidate)
	}

	// Quoted binary path
	if !strings.Contains(loop, "'/usr/local/bin/spekk'") {
		t.Errorf("loop line binary path not quoted: %q", loop)
	}
	if !strings.Contains(consolidate, "'/usr/local/bin/spekk'") {
		t.Errorf("consolidate line binary path not quoted: %q", consolidate)
	}

	// Absolute claude path passed via --claude-path (no shell flock)
	if !strings.Contains(loop, "--claude-path '/home/user/.claude/local/claude'") {
		t.Errorf("loop line missing --claude-path: %q", loop)
	}
	if !strings.Contains(consolidate, "--claude-path '/home/user/.claude/local/claude'") {
		t.Errorf("consolidate line missing --claude-path: %q", consolidate)
	}
	if strings.Contains(loop, "flock") {
		t.Errorf("loop line must not contain shell flock (overlap guard is in Go): %q", loop)
	}
	if strings.Contains(consolidate, "flock") {
		t.Errorf("consolidate line must not contain shell flock: %q", consolidate)
	}

	// headless flag
	if !strings.Contains(loop, "--headless") {
		t.Errorf("loop line missing --headless flag: %q", loop)
	}
	if !strings.Contains(consolidate, "--headless") {
		t.Errorf("consolidate line missing --headless flag: %q", consolidate)
	}

	// log file redirect (append)
	if !strings.Contains(loop, ">> '/home/user/my project/.spekk/observer.log'") {
		t.Errorf("loop line missing log file redirect: %q", loop)
	}
	if !strings.Contains(consolidate, ">> '/home/user/my project/.spekk/observer-consolidate.log'") {
		t.Errorf("consolidate line missing log file redirect: %q", consolidate)
	}

	// cron marker
	if !strings.Contains(loop, cronMarker) {
		t.Errorf("loop line missing marker: %q", loop)
	}
	if !strings.Contains(consolidate, cronMarker) {
		t.Errorf("consolidate line missing marker: %q", consolidate)
	}

	// consolidate line contains the consolidate subcommand; loop does not
	if !strings.Contains(consolidate, "observer consolidate") {
		t.Errorf("consolidate line missing 'observer consolidate' subcommand: %q", consolidate)
	}
	if strings.Contains(loop, "observer consolidate") {
		t.Errorf("loop line should not include consolidate subcommand: %q", loop)
	}
}

// TestDoInstallCron_ClaudeNotFound verifies that doInstallCron returns an error
// when the claude binary cannot be found on PATH, and installs nothing.
func TestDoInstallCron_ClaudeNotFound(t *testing.T) {
	// Override PATH to an empty value so exec.LookPath("claude") fails.
	t.Setenv("PATH", "")

	err := doInstallCron([]string{})
	if err == nil {
		t.Fatal("expected error when claude is not on PATH")
	}
	if !strings.Contains(err.Error(), "claude") {
		t.Errorf("error should mention 'claude': %v", err)
	}
}

// TestRemoveCronMarkerLines verifies that only spekk-observer lines are removed.
func TestRemoveCronMarkerLines(t *testing.T) {
	input := strings.Join([]string{
		"0 * * * * /usr/local/bin/something-else",
		"*/30 * * * * /usr/local/bin/spekk observer " + cronMarker,
		"0 */6 * * * /usr/local/bin/spekk observer consolidate " + cronMarker,
		"30 2 * * * /usr/local/bin/backup",
	}, "\n") + "\n"

	result := removeCronMarkerLines(input)

	if strings.Contains(result, cronMarker) {
		t.Error("result should not contain the spekk marker")
	}
	if !strings.Contains(result, "something-else") {
		t.Error("unrelated entry should be preserved")
	}
	if !strings.Contains(result, "backup") {
		t.Error("unrelated entry should be preserved")
	}
}

// TestRemoveCronMarkerLines_Empty handles an empty / no-crontab case.
func TestRemoveCronMarkerLines_Empty(t *testing.T) {
	result := removeCronMarkerLines("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// TestCronMarkerLines returns only the spekk-managed lines.
func TestCronMarkerLines(t *testing.T) {
	input := "*/30 * * * * spekk observer " + cronMarker + "\n" +
		"0 2 * * * backup\n" +
		"0 */6 * * * spekk observer consolidate " + cronMarker + "\n"

	found := cronMarkerLines(input)
	if len(found) != 2 {
		t.Fatalf("expected 2 marker lines, got %d: %v", len(found), found)
	}
}

// TestRemoveCronMarkerLines_Idempotent confirms that running removal twice is safe.
func TestRemoveCronMarkerLines_Idempotent(t *testing.T) {
	withEntries := "*/30 * * * * spekk observer " + cronMarker + "\n" +
		"0 2 * * * backup\n"
	once := removeCronMarkerLines(withEntries)
	twice := removeCronMarkerLines(once)
	if once != twice {
		t.Errorf("removal is not idempotent:\nonce=%q\ntwice=%q", once, twice)
	}
}
