package conversation

import (
	"encoding/json"
	"testing"
)

func TestRequestJSONRoundTrip(t *testing.T) {
	req := Request{Title: "t", Body: "b", Severity: SeverityWarning}

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("Unmarshal into map: %v", err)
	}
	if len(asMap) != 3 {
		t.Fatalf("expected exactly 3 keys, got %d: %v", len(asMap), asMap)
	}
	for _, key := range []string{"title", "body", "severity"} {
		if _, ok := asMap[key]; !ok {
			t.Errorf("expected key %q in marshalled JSON, got %v", key, asMap)
		}
	}
	if _, ok := asMap["session_id"]; ok {
		t.Errorf("session_id must not appear in the request file, got %v", asMap)
	}

	var roundTripped Request
	if err := json.Unmarshal(raw, &roundTripped); err != nil {
		t.Fatalf("Unmarshal into Request: %v", err)
	}
	if roundTripped != req {
		t.Errorf("round-trip mismatch: got %+v, want %+v", roundTripped, req)
	}
}

func TestRequestAbsentSeverityDefault(t *testing.T) {
	req := Request{Title: "t", Body: "b"}
	if req.Severity != "" {
		t.Fatalf("expected zero-value Severity to be empty, got %q", req.Severity)
	}
	if DefaultSeverity != SeverityInfo {
		t.Fatalf("expected DefaultSeverity to be SeverityInfo, got %q", DefaultSeverity)
	}
}

func TestIsValidSeverity(t *testing.T) {
	cases := map[string]bool{
		"info":     true,
		"warning":  true,
		"critical": true,
		"":         false,
		"debug":    false,
		"INFO":     false,
	}
	for severity, want := range cases {
		if got := IsValidSeverity(severity); got != want {
			t.Errorf("IsValidSeverity(%q) = %v, want %v", severity, got, want)
		}
	}
}
