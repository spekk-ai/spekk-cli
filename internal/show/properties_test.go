package show

import (
	"os/exec"
	"strings"
	"testing"
)

// TestTemplateDetailProperties runs the template's own propertiesHtml in node
// (skipped when node is absent) and checks the "Properties" block, then checks
// the template wires it into both detail panels and wraps long values.
func TestTemplateDetailProperties(t *testing.T) {
	start := strings.Index(templateHTML, "function propertiesHtml(")
	end := strings.Index(templateHTML, "// Status emoji for detail badges")
	if start < 0 || end < start {
		t.Fatal("template has no propertiesHtml function")
	}
	if got := strings.Count(templateHTML, "html += propertiesHtml("); got != 2 {
		t.Errorf("propertiesHtml is wired into %d detail panels, want 2 (spec and assertion)", got)
	}
	if !strings.Contains(templateHTML, "overflow-wrap: anywhere") {
		t.Error("property lines must wrap long values")
	}

	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	script := `
function escapeHtml(s) { return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
` + templateHTML[start:end] + `
var full = propertiesHtml({customFields: {zeta: ['b','c'], '10': ['x'], alpha: ['<script>alert(1)</script>'], '<k>': ['v']}});
var none = propertiesHtml({});
var empty = propertiesHtml({customFields: {}});
console.log(JSON.stringify({full: full, none: none, empty: empty}));
`
	out, err := exec.Command(node, "-e", script).CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}
	s := string(out)
	if strings.Contains(s, "<script>") {
		t.Errorf("value was not escaped: %s", s)
	}
	if !strings.Contains(s, "&lt;script&gt;alert(1)&lt;/script&gt;") || !strings.Contains(s, "&lt;k&gt;") {
		t.Errorf("escaped key or value missing: %s", s)
	}
	if !strings.Contains(s, ">zeta</span>: b, c<") {
		t.Errorf("values are not joined with \", \": %s", s)
	}
	// Byte order: "10" < "<k>" < "alpha" < "zeta".
	last := -1
	for _, k := range []string{">10<", ">&lt;k&gt;<", ">alpha<", ">zeta<"} {
		i := strings.Index(s, k)
		if i <= last {
			t.Errorf("key %s out of order in %s", k, s)
		}
		last = i
	}
	if !strings.Contains(s, `"none":""`) || !strings.Contains(s, `"empty":""`) {
		t.Errorf("an item without custom fields must render nothing: %s", s)
	}
	if strings.Count(s, "Properties") != 1 {
		t.Errorf("want exactly one Properties heading: %s", s)
	}
}
