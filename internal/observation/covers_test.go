package observation

import "testing"

// The dedup key is a string an agent writes, not one the code derives, so
// the same file arrives spelled several ways across runs. Every spelling
// that carries no meaning must reduce to the same thing, or a second run
// files a finding a first run already filed.
func TestNormalizePathStripsSpellingThatCarriesNoMeaning(t *testing.T) {
	const want = "internal/parser/parser.go"

	for _, spelling := range []string{
		"internal/parser/parser.go",
		"./internal/parser/parser.go",
		".//internal/parser/parser.go",
		"internal/parser/parser.go:42",
		"internal/parser/parser.go:42:7",
		"  internal/parser/parser.go  ",
		"./internal/parser/parser.go:42",
	} {
		if got := NormalizePath(spelling); got != want {
			t.Errorf("NormalizePath(%q) = %q, want %q", spelling, got, want)
		}
	}
}

// Normalization must not invent equality. A directory is not the files under
// it, and a colon that is not a location is part of the name.
func TestNormalizePathKeepsWhatIsMeaningful(t *testing.T) {
	cases := map[string]string{
		"internal/parser/":       "internal/parser",
		"internal/parser":        "internal/parser",
		"weird:name.go":          "weird:name.go",
		"path/to/file.go:notnum": "path/to/file.go:notnum",
		"":                       "",
		// At most two suffixes are removed, because that is all a location
		// can be. A name is not eaten segment by segment.
		"a:1:2:3": "a:1",
	}
	for in, want := range cases {
		if got := NormalizePath(in); got != want {
			t.Errorf("NormalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

// The compounding contract, stated as the table it was measured as: run 1
// filed internal/parser/parser.go, and run 2 must be told the drift is
// already covered however it spells the same file.
func TestCoversAcrossSpellingsOfTheSameFile(t *testing.T) {
	filed := &Observation{
		Type:     "code_spec_misalignment",
		Affected: []string{"internal/parser/parser.go"},
	}

	covered := []struct {
		name     string
		typ      string
		affected []string
	}{
		{"identical", "code_spec_misalignment", []string{"internal/parser/parser.go"}},
		{"one of several", "code_spec_misalignment", []string{"internal/parser/parser.go", "other.go"}},
		{"with a line", "code_spec_misalignment", []string{"internal/parser/parser.go:42"}},
		{"with line and column", "code_spec_misalignment", []string{"internal/parser/parser.go:42:7"}},
		{"dot-slash prefixed", "code_spec_misalignment", []string{"./internal/parser/parser.go"}},
	}
	for _, tc := range covered {
		if !filed.Covers(tc.typ, tc.affected) {
			t.Errorf("%s: expected covered, so run 2 files nothing", tc.name)
		}
	}

	notCovered := []struct {
		name     string
		typ      string
		affected []string
		why      string
	}{
		{
			"a different file", "code_spec_misalignment",
			[]string{"internal/parser/output.go"},
			"distinct drift must still be filed",
		},
		{
			"the containing directory", "code_spec_misalignment",
			[]string{"internal/parser/"},
			"a directory-level finding must not swallow every file beneath it",
		},
		{
			"a different type", "outdated_specs",
			[]string{"internal/parser/parser.go"},
			"the type is part of what the finding is",
		},
	}
	for _, tc := range notCovered {
		if filed.Covers(tc.typ, tc.affected) {
			t.Errorf("%s: expected not covered — %s", tc.name, tc.why)
		}
	}
}

// Normalization applies to what was filed as well as to what is offered. A
// first run that recorded a line number must still cover a second run that
// does not.
func TestCoversNormalizesTheFiledSideToo(t *testing.T) {
	filed := &Observation{
		Type:     "code_spec_misalignment",
		Affected: []string{"./internal/parser/parser.go:42"},
	}
	if !filed.Covers("code_spec_misalignment", []string{"internal/parser/parser.go"}) {
		t.Error("a filed path with a location suffix must still cover the plain path")
	}
}
