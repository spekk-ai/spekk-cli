package claudecode

import "testing"

func TestParseVersion(t *testing.T) {
	got, err := ParseVersion("** WARNING: connection is not using a post-quantum key exchange algorithm.\n2.1.301 (Claude Code)\n")
	if err != nil || got != "2.1.301" {
		t.Errorf("ParseVersion = %q, %v; want 2.1.301", got, err)
	}
	if got, err := ParseVersion("bash: line 1: claude: command not found\n"); err == nil {
		t.Errorf("ParseVersion of a missing claude = %q, want an error", got)
	}
}

// The comparison is numeric for each part. A comparison of the text puts
// 2.1.1000 before 2.1.292.
func TestCompare(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"2.1.291", "2.1.292", -1},
		{"2.1.292", "2.1.292", 0},
		{"2.1.1000", "2.1.292", 1},
		{"2.10.0", "2.9.9", 1},
		{"3.0.0", "2.99.99", 1},
	} {
		if got := compare(tc.a, tc.b); got != tc.want {
			t.Errorf("compare(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
	if BelowMinimum(MinVersion) {
		t.Errorf("BelowMinimum(MinVersion) = true, want false")
	}
}
