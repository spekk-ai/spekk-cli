package crossbranch

import "testing"

func TestParseGitVersion(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantMajor int
		wantMinor int
		wantErr   bool
	}{
		{name: "standard", in: "git version 2.43.0", wantMajor: 2, wantMinor: 43},
		{name: "apple vendor suffix", in: "git version 2.39.3 (Apple Git-145)", wantMajor: 2, wantMinor: 39},
		{name: "windows suffix", in: "git version 2.30.windows.1", wantMajor: 2, wantMinor: 30},
		{name: "floor exactly", in: "git version 2.38.0", wantMajor: 2, wantMinor: 38},
		{name: "minor without patch", in: "git version 2.38", wantMajor: 2, wantMinor: 38},
		{name: "rc suffix on minor", in: "git version 2.38-rc0", wantMajor: 2, wantMinor: 38},
		{name: "extra leading whitespace", in: "  git version 2.41.1  ", wantMajor: 2, wantMinor: 41},
		{name: "future major", in: "git version 3.0.0", wantMajor: 3, wantMinor: 0},

		{name: "no version token", in: "git version unknown", wantErr: true},
		{name: "empty", in: "", wantErr: true},
		{name: "garbage", in: "not a version string", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := parseGitVersion(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseGitVersion(%q) = (%d, %d, nil); want error", tt.in, major, minor)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGitVersion(%q) returned unexpected error: %v", tt.in, err)
			}
			if major != tt.wantMajor || minor != tt.wantMinor {
				t.Errorf("parseGitVersion(%q) = (%d, %d); want (%d, %d)",
					tt.in, major, minor, tt.wantMajor, tt.wantMinor)
			}
		})
	}
}

func TestAtLeastMergeTree(t *testing.T) {
	tests := []struct {
		name  string
		major int
		minor int
		want  bool
	}{
		{name: "exactly floor 2.38", major: 2, minor: 38, want: true},
		{name: "just below floor 2.37", major: 2, minor: 37, want: false},
		{name: "above floor same major 2.43", major: 2, minor: 43, want: true},
		{name: "well below 2.30", major: 2, minor: 30, want: false},
		{name: "older major 1.99", major: 1, minor: 99, want: false},
		{name: "newer major 3.0", major: 3, minor: 0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := atLeastMergeTree(tt.major, tt.minor); got != tt.want {
				t.Errorf("atLeastMergeTree(%d, %d) = %v; want %v",
					tt.major, tt.minor, got, tt.want)
			}
		})
	}
}
