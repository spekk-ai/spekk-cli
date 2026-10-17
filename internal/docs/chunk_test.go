package docs

import (
	"fmt"
	"strings"
	"testing"
)

type span struct {
	heading    string
	start, end int
}

func spans(ps []Passage) []span {
	out := make([]span, len(ps))
	for i, p := range ps {
		out[i] = span{p.Heading, p.Start, p.End}
	}
	return out
}

func TestChunkBoundaries(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []span
	}{
		{
			name: "frontmatter lines are counted but in no passage",
			text: "---\ntitle: x\n---\n\nIntro line.\n\n# One\n\nBody one.\n\n## Two\nBody two.\n",
			want: []span{{"", 5, 5}, {"One", 7, 9}, {"Two", 11, 12}},
		},
		{
			name: "setext headings with = and -",
			text: "Title\n=====\n\nText.\n\nPart\n----\nMore.\n",
			want: []span{{"Title", 1, 4}, {"Part", 6, 8}},
		},
		{
			name: "a heading inside a code fence is text",
			text: "# Real\n\n```sh\n# not a heading\n```\n\n~~~\nx\n===\n~~~\n",
			want: []span{{"Real", 1, 10}},
		},
		{
			name: "a heading-only section joins the next",
			text: "# Part A\n\n## Detail\nText.\n",
			want: []span{{"Part A / Detail", 1, 4}},
		},
		{
			name: "seven hashes and no space are not headings",
			text: "####### seven\n#tag\nText.\n",
			want: []span{{"", 1, 3}},
		},
		{
			name: "CRLF gives the same line numbers",
			text: "# A\r\nx\r\n\r\n# B\r\ny\r\n",
			want: []span{{"A", 1, 2}, {"B", 4, 5}},
		},
		{
			name: "blank text gives no passages",
			text: "\n\n  \n",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := spans(Chunk(tc.text))
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestChunkWindowsLongSections(t *testing.T) {
	var b strings.Builder
	b.WriteString("# Long\n")
	for i := 0; i < 99; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	got := spans(Chunk(b.String()))
	want := []span{{"Long", 1, 40}, {"Long", 41, 80}, {"Long", 81, 100}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("100-line section: got %v, want %v", got, want)
	}

	// 80 lines is the limit: a section of exactly 80 lines stays whole.
	b.Reset()
	b.WriteString("# Edge\n")
	for i := 0; i < 79; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	if got := spans(Chunk(b.String())); len(got) != 1 || got[0].end != 80 {
		t.Errorf("80-line section: got %v, want one passage 1-80", got)
	}
}
