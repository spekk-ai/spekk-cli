package docs

import (
	"errors"
	"strings"
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

func TestLineWindow(t *testing.T) {
	filler := strings.Repeat("we talked about the weather and lunch plans for a while. ", 6)
	turn := "**Rowan Pike:** " + filler + "The harbor survey quote came to 7,400 for the north pier. " + filler
	terms := []string{"harbo", "surve", "quote"}

	got := lineWindow(turn, terms, 160)
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "…") || !strings.Contains(got, "7,400 for the north pier") {
		t.Errorf("deep match: got %q, want the part with the quote, marked at both ends", got)
	}
	if n := len([]rune(got)); n > 160 {
		t.Errorf("deep match: %d runes, want at most 160", n)
	}

	// Both parts hold the two terms; the second holds them more often.
	tie := "harbor quote early. " + filler + "harbor quote, harbor quote again: the late part. " + filler
	if got := lineWindow(tie, []string{"harbo", "quote"}, 160); !strings.Contains(got, "the late part") {
		t.Errorf("tie: got %q, want the part with more matches", got)
	}

	for _, line := range []string{"a short harbor line", filler + filler} {
		want := Truncate(line, 160)
		if got := lineWindow(line, terms, 160); got != want {
			t.Errorf("lineWindow(%.30q...) = %q, want %q", line, got, want)
		}
	}
}
