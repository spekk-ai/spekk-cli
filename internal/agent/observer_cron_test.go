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

// TestMinutesToCron covers the three cases: sub-hour, exact-hour, and other.
func TestMinutesToCron(t *testing.T) {
	cases := []struct {
		minutes int
		want    string
	}{
		{30, "*/30 * * * *"},
		{15, "*/15 * * * *"},
		{60, "0 */1 * * *"},
		{360, "0 */6 * * *"},
		{120, "0 */2 * * *"},
		// Non-round: falls back to minute-field form
		{90, "*/90 * * * *"},
	}
	for _, tc := range cases {
		got := minutesToCron(tc.minutes)
		if got != tc.want {
			t.Errorf("minutesToCron(%d) = %q, want %q", tc.minutes, got, tc.want)
		}
	}
}

// TestBuildCronLines verifies the two generated lines contain the binary, marker,
// and correct cron schedule.
func TestBuildCronLines(t *testing.T) {
	cfg := InstallCronConfig{LoopInterval: 30, ConsolidateInterval: 360}
	loop, consolidate := buildCronLines("/usr/local/bin/spekk", cfg)

	if !strings.Contains(loop, "*/30 * * * *") {
		t.Errorf("loop line missing schedule: %q", loop)
	}
	if !strings.Contains(loop, "/usr/local/bin/spekk observer") {
		t.Errorf("loop line missing binary/command: %q", loop)
	}
	if !strings.Contains(loop, cronMarker) {
		t.Errorf("loop line missing marker: %q", loop)
	}
	// loop line must NOT include "consolidate"
	if strings.Contains(loop, "consolidate") {
		t.Errorf("loop line should not include consolidate: %q", loop)
	}

	if !strings.Contains(consolidate, "0 */6 * * *") {
		t.Errorf("consolidate line missing schedule: %q", consolidate)
	}
	if !strings.Contains(consolidate, "/usr/local/bin/spekk observer consolidate") {
		t.Errorf("consolidate line missing command: %q", consolidate)
	}
	if !strings.Contains(consolidate, cronMarker) {
		t.Errorf("consolidate line missing marker: %q", consolidate)
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
