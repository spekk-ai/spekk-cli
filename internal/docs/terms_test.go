package docs

import (
	"errors"
	"testing"
)

func TestMatchExpression(t *testing.T) {
	cases := []struct {
		query, want string
	}{
		// Each pair of adjacent terms is also a phrase.
		{`index rebuild`, `"index" OR "rebuild" OR "index rebuild"`},
		{`snake_case dotted.name`, `"snake_case" OR "dotted.name" OR "snake_case dotted.name"`},
		{`"exact phrase" word`, `"exact phrase" OR "word" OR "exact phrase word"`},
		// FTS5 operators and syntax are plain text.
		{`NOT a*`, `"NOT" OR "a*" OR "NOT a*"`},
		{`(b) col:c -d`, `"(b)" OR "col:c" OR "-d" OR "(b) col:c" OR "col:c -d"`},
		{`say"hi`, `"say" OR "hi" OR "say hi"`},
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
