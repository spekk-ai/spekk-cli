//go:build !windows

package agent

import "testing"

func TestMarkerAtLineStart(t *testing.T) {
	marker := []byte(doneMarker)
	tests := []struct {
		name          string
		data          string
		atStreamStart bool
		want          bool
	}{
		{"on its own line", "working...\n[SPEKK_DONE]\n", false, true},
		{"at stream start", "[SPEKK_DONE]\n", true, true},
		{"at position zero but not stream start", "[SPEKK_DONE]\n", false, false},
		{"inline within a line", "see [SPEKK_DONE] token\n", false, false},
		{"echoed after prose", "the marker is [SPEKK_DONE]", false, false},
		{"absent", "no marker here\n", false, false},
		{"line start after other lines", "a\nb\n[SPEKK_DONE]", false, true},
	}
	for _, tt := range tests {
		got := markerAtLineStart([]byte(tt.data), marker, tt.atStreamStart)
		if got != tt.want {
			t.Errorf("%s: markerAtLineStart(%q, atStreamStart=%v) = %v, want %v",
				tt.name, tt.data, tt.atStreamStart, got, tt.want)
		}
	}
}
