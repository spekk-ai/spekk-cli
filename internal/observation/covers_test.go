package observation

import "testing"

// An affected path is a string an agent writes, not one the code derives,
// so the same file arrives spelled several ways across runs. Every spelling
// that carries no meaning must reduce to the same thing, or a dont-flag
// entry stops matching the drift it was written to suppress.
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

// The dedup key is the finding, not the file. Two findings that name one
// file are two findings — the affected list of a code_spec_misalignment
// names the code as well as the assertion, so the code file is what
// unrelated findings share.
func TestCoversKeysOnTypeAndSlug(t *testing.T) {
	filed := &Observation{
		Slug:     "parser-drops-draft-status",
		Type:     "code_spec_misalignment",
		Affected: []string{"internal/parser/parser.go"},
	}

	if !filed.Covers("code_spec_misalignment", "parser-drops-draft-status") {
		t.Error("the same finding must be covered, or run 2 files it again")
	}
	if filed.Covers("code_spec_misalignment", "parser-ignores-priority") {
		t.Error("a different finding in the same file must still be filed")
	}
	if filed.Covers("outdated_specs", "parser-drops-draft-status") {
		t.Error("one file can carry drift of both types; neither covers the other")
	}
}
