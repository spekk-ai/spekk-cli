package docs

import (
	"errors"
	"testing"
)

func TestMatchExpression(t *testing.T) {
	cases := []struct {
		query, want string
	}{
		{`index rebuild`, `"index" OR "rebuild"`},
		{`snake_case dotted.name kebab-case`, `"snake_case" OR "dotted.name" OR "kebab-case"`},
		{`"exact phrase" word`, `"exact phrase" OR "word"`},
		// FTS5 operators and syntax are plain text.
		{`NOT a* (b) col:c -d`, `"NOT" OR "a*" OR "(b)" OR "col:c" OR "-d"`},
		{`say"hi`, `"say" OR "hi"`},
		// A term with no letter or digit is dropped.
		{`-- * fix`, `"fix"`},
	}
	for _, tc := range cases {
		got, err := MatchExpression(tc.query)
		if err != nil || got != tc.want {
			t.Errorf("MatchExpression(%q) = %q, %v; want %q", tc.query, got, err, tc.want)
		}
	}
	for _, q := range []string{"", "  ", "*** --"} {
		if _, err := MatchExpression(q); !errors.Is(err, ErrEmptyQuery) {
			t.Errorf("MatchExpression(%q) err = %v, want ErrEmptyQuery", q, err)
		}
	}
}

func TestIdentifierTerms(t *testing.T) {
	got := IdentifierTerms("call parseAllSpecs, then HTTPServer.Start and EnsureFresh; plain words and snake_case stay out")
	want := "parse all specs http server ensure fresh"
	if got != want {
		t.Errorf("IdentifierTerms = %q, want %q", got, want)
	}
}
