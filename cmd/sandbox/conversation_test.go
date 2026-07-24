package main

import (
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/conversation"
)

// TestNewConversationOpenFrameShape verifies the exact key set produced with
// and without metadata, and that type/severity round-trip correctly.
func TestNewConversationOpenFrameShape(t *testing.T) {
	t.Run("with metadata", func(t *testing.T) {
		frame, err := NewConversationOpenFrame("sess-1", "Title", "Body", conversation.SeverityWarning, map[string]any{"k": "v"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		wantKeys := []string{"type", "session_id", "title", "body", "severity", "metadata"}
		if len(frame) != len(wantKeys) {
			t.Fatalf("expected %d keys, got %d: %v", len(wantKeys), len(frame), frame)
		}
		for _, k := range wantKeys {
			if _, ok := frame[k]; !ok {
				t.Errorf("expected key %q in frame, got %v", k, frame)
			}
		}

		if frame["type"] != MessageTypeConversationOpen {
			t.Errorf("type = %v, want %v", frame["type"], MessageTypeConversationOpen)
		}
		if frame["severity"] != conversation.SeverityWarning {
			t.Errorf("severity = %v, want %v", frame["severity"], conversation.SeverityWarning)
		}
	})

	t.Run("without metadata", func(t *testing.T) {
		frame, err := NewConversationOpenFrame("sess-1", "Title", "Body", conversation.SeverityInfo, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := frame["metadata"]; ok {
			t.Errorf("expected metadata key to be omitted when empty, got %v", frame)
		}

		wantKeys := []string{"type", "session_id", "title", "body", "severity"}
		if len(frame) != len(wantKeys) {
			t.Fatalf("expected %d keys, got %d: %v", len(wantKeys), len(frame), frame)
		}
	})
}

// TestNewConversationOpenFrameEmptySessionID verifies that an empty
// session_id is rejected rather than sent.
func TestNewConversationOpenFrameEmptySessionID(t *testing.T) {
	_, err := NewConversationOpenFrame("", "Title", "Body", conversation.SeverityInfo, nil)
	if err == nil {
		t.Fatal("expected error for empty session_id, got nil")
	}
}

// TestNewConversationOpenFrameSeverityDefaulting covers the three severity
// rules: empty defaults to info, a valid value round-trips, and an invalid
// value is rejected rather than passed through verbatim.
func TestNewConversationOpenFrameSeverityDefaulting(t *testing.T) {
	t.Run("empty severity defaults to info", func(t *testing.T) {
		frame, err := NewConversationOpenFrame("sess-1", "Title", "Body", "", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if frame["severity"] != conversation.DefaultSeverity {
			t.Errorf("severity = %v, want default %v", frame["severity"], conversation.DefaultSeverity)
		}
	})

	t.Run("invalid severity is rejected", func(t *testing.T) {
		_, err := NewConversationOpenFrame("sess-1", "Title", "Body", conversation.Severity("urgent"), nil)
		if err == nil {
			t.Fatal("expected error for invalid severity, got nil")
		}
	})
}
