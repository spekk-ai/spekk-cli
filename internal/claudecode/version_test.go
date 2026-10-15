package claudecode

import "testing"

// A later Claude Code can print its version in another form, so the parse
// knows the likely forms. Text with no version, and an IP address in an ssh
// warning, are not versions.
func TestParseVersion(t *testing.T) {
	for _, out := range []string{
		"2.1.301 (Claude Code)\n",
		"Claude Code v2.1.301\n",
		"claude 2.1.301\n",
		"Warning: Permanently added '10.1.2.3' (ED25519) to the list of known hosts.\n2.1.301 (Claude Code)\n",
	} {
		if got, err := ParseVersion(out); err != nil || got != "2.1.301" {
			t.Errorf("ParseVersion(%q) = %q, %v; want 2.1.301", out, got, err)
		}
	}
	for _, out := range []string{
		"bash: line 1: claude: command not found\n",
		"Warning: Permanently added '10.1.2.3' (ED25519) to the list of known hosts.\n",
		"",
	} {
		if got, err := ParseVersion(out); err == nil {
			t.Errorf("ParseVersion(%q) = %q, want an error", out, got)
		}
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
