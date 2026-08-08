package crossbranch

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatJSON_EmptyIsArray(t *testing.T) {
	out, err := FormatJSON(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Errorf("expected [], got %q", out)
	}
}

func TestFormatJSON_Rows(t *testing.T) {
	states := []FileState{
		{
			Path:      "specs/demo/assertions/drifts.md",
			Branch:    "other",
			State:     StateIncomingMod,
			OldStatus: "not_started",
			NewStatus: "done",
		},
		{
			Path:     "specs/demo/assertions/conflicted.md",
			Branch:   "other",
			State:    StateConflict,
			Degraded: true,
			// Meta must never leak into the machine surface.
			Meta: &FileMeta{Title: "secret", Content: "full file body"},
		},
	}
	out, err := FormatJSON(states)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["state"] != "incoming_mod" || rows[0]["old_status"] != "not_started" || rows[0]["new_status"] != "done" {
		t.Errorf("unexpected drift row: %v", rows[0])
	}
	if rows[0]["degraded"] != false {
		t.Errorf("expected degraded=false as a JSON bool, got %v", rows[0]["degraded"])
	}
	if rows[1]["degraded"] != true {
		t.Errorf("expected degraded=true as a JSON bool, got %v", rows[1]["degraded"])
	}
	// Every row carries every key, so a consumer never has to tell a missing
	// key apart from an empty value -- and TSV/CSV, which always emit the
	// column, agree with JSON.
	if v, ok := rows[1]["old_status"]; !ok || v != "" {
		t.Errorf("expected an empty old_status key, got %v (present=%v)", v, ok)
	}
	if v, ok := rows[1]["new_status"]; !ok || v != "" {
		t.Errorf("expected an empty new_status key, got %v (present=%v)", v, ok)
	}
	if strings.Contains(out, "full file body") || strings.Contains(out, "Meta") {
		t.Errorf("Meta content leaked into JSON output:\n%s", out)
	}
}

func TestOutputRows(t *testing.T) {
	states := []FileState{
		{Path: "specs/a/a.md", Branch: "feat/x", State: StateIncomingAdd},
	}
	rows := OutputRows(states)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	want := []string{"specs/a/a.md", "feat/x", "incoming_add", "false", "", ""}
	for i, cell := range want {
		if rows[0][i] != cell {
			t.Errorf("cell %d (%s): expected %q, got %q", i, OutputColumns[i], cell, rows[0][i])
		}
	}
	if len(rows[0]) != len(OutputColumns) {
		t.Errorf("row width %d does not match OutputColumns %d", len(rows[0]), len(OutputColumns))
	}
}
