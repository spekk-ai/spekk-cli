package docs

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseRange(t *testing.T) {
	for in, want := range map[string][2]int{"3:9": {3, 9}, "7:": {7, 0}, "5:5": {5, 5}} {
		a, b, err := ParseRange(in)
		if err != nil || a != want[0] || b != want[1] {
			t.Errorf("ParseRange(%q) = %d, %d, %v; want %v", in, a, b, err, want)
		}
	}
	for _, in := range []string{"", "5", "0:3", "9:3", "a:b", ":4"} {
		if _, _, err := ParseRange(in); err == nil {
			t.Errorf("ParseRange(%q) = nil error, want ErrBadRange", in)
		}
	}
}

func TestRenderShowCapsLinesAndCleansText(t *testing.T) {
	var b strings.Builder
	for i := 1; i <= 450; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	d := Document{Kind: KindRaw, ID: "manual", TextPath: "formatted/manual.txt", Title: "Manual"}

	out := RenderShow(d, b.String(), 1, 0)
	if !strings.Contains(out, "lines 1-200 of 450") || !strings.Contains(out, "200  line 200\n") || strings.Contains(out, "line 201\n") {
		t.Errorf("first call is not lines 1-200:\n%s", out[:200])
	}
	if !strings.Contains(out, "--lines 201:400") {
		t.Errorf("no next-range hint 201:400 in:\n%s", out[len(out)-200:])
	}
	if out := RenderShow(d, b.String(), 401, 0); strings.Contains(out, "Printed") || !strings.Contains(out, "lines 401-450 of 450") {
		t.Errorf("a range under the cap must print whole, without a hint:\n%s", out)
	}

	out = RenderShow(d, "ok\x1b[2J\x07 text\n", 1, 0)
	if strings.ContainsAny(out, "\x1b\x07") || !strings.Contains(out, "ok[2J text") {
		t.Errorf("control characters not removed: %q", out)
	}
	if !strings.Contains(out, "begin untrusted document text") || !strings.Contains(out, "end untrusted document text") {
		t.Errorf("raw text is not between untrusted markers: %q", out)
	}
}
