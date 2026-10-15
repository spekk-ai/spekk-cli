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

func TestBelowMinimum(t *testing.T) {
	for version, want := range map[string]bool{
		"2.1.197":  true,
		"2.1.291":  true,
		"2.1.292":  false,
		"2.1.1000": false,
		"2.10.0":   false,
		"3.0.0":    false,
	} {
		if got := BelowMinimum(version); got != want {
			t.Errorf("BelowMinimum(%q) = %v, want %v", version, got, want)
		}
	}
}
