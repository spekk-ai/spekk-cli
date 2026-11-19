package show

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// customFieldsTree writes a fixture tree whose items carry custom fields in
// every spelling, and returns its specs directory.
func customFieldsTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	writeFile(t, filepath.Join(specsDir, "alpha", "alpha.md"),
		"---\nid: alpha\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nowner: platform\nworkflows: [checkout, signup]\n---\n\n# Alpha\n")
	writeFile(t, filepath.Join(specsDir, "alpha", "assertions", "one.md"),
		"---\nid: one\nparent: alpha\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: done\nbranch: main\ndepends-on: two\n"+
			"zeta: scalar\nworkflows: checkout, signup\nlabels:\n  - \"a, b\"\n  - c\ndup: x, y, x\nnotes: |\n  prose here\nowner: [\"platform\"]\nalpha-key: <script>alert(1)</script>\n---\n\n# One\n")
	writeFile(t, filepath.Join(specsDir, "alpha", "assertions", "two.md"),
		"---\nid: two\nparent: alpha\ncreated: 2026-01-01T00:00:00Z\npriority: 2\n---\n\n# Two\n")
	return specsDir
}

func TestShowDataCustomFields(t *testing.T) {
	specsDir := customFieldsTree(t)

	var out bytes.Buffer
	if err := WriteJSON(&out, specsDir, Options{}); err != nil {
		t.Fatal(err)
	}
	var data showData
	if err := json.Unmarshal(out.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	byID := map[string]showAssertion{}
	for _, a := range data.Assertions {
		byID[a.ID] = a
	}

	wantSpec := map[string][]string{"owner": {"platform"}, "workflows": {"checkout", "signup"}}
	if !reflect.DeepEqual(data.Specs[0].CustomFields, wantSpec) {
		t.Errorf("spec customFields = %v, want %v", data.Specs[0].CustomFields, wantSpec)
	}
	want := map[string][]string{
		"zeta":      {"scalar"},
		"workflows": {"checkout", "signup"}, // scalar spelling == flow spelling
		"labels":    {"a, b", "c"},          // block list items are not re-split
		"dup":       {"x", "y"},             // repeated value once
		"owner":     {"platform"},           // quoted flow item
		"alpha-key": {"<script>alert(1)</script>"},
	}
	if got := byID["one"].CustomFields; !reflect.DeepEqual(got, want) {
		t.Errorf("assertion customFields = %v, want %v", got, want)
	}
	for _, known := range []string{"id", "status", "priority", "depends-on", "branch", "parent", "created", "notes"} {
		if _, ok := byID["one"].CustomFields[known]; ok {
			t.Errorf("known or block-scalar key %q leaked into customFields", known)
		}
	}

	// An item without custom fields has no customFields key in the JSON.
	var raw struct {
		Assertions []map[string]json.RawMessage `json:"assertions"`
	}
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, a := range raw.Assertions {
		var id string
		_ = json.Unmarshal(a["id"], &id)
		if _, has := a["customFields"]; id == "two" && has {
			t.Error("assertion without custom fields carries a customFields key")
		}
	}

	// The --json data and the data embedded in the HTML are the same bytes.
	if err := Run(specsDir, Options{}); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(filepath.Dir(specsDir), ".spekk", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), strings.TrimSuffix(out.String(), "\n")) {
		t.Error("HTML does not embed the same data as --json")
	}
}

func TestForeignItemCarriesCustomFields(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "commit.gpgsign", "false")
	specsDir := filepath.Join(dir, "specs")
	writeFile(t, filepath.Join(specsDir, "local", "local.md"),
		"---\nid: local\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n\n# Local\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "base")
	git(t, dir, "checkout", "-q", "-b", "other")
	writeFile(t, filepath.Join(specsDir, "shiny", "shiny.md"),
		"---\nid: shiny\ncreated: 2026-01-01T00:00:00Z\npriority: 2\nowner: [core]\n---\n\n# Shiny\n")
	writeFile(t, filepath.Join(specsDir, "shiny", "assertions", "works.md"),
		"---\nid: works\nparent: shiny\ncreated: 2026-01-01T00:00:00Z\npriority: 3\nstatus: done\nworkflows: a, b\n---\n\n# Works\n")
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-qm", "foreign")
	git(t, dir, "checkout", "-q", "main")
	chdir(t, dir)

	var out bytes.Buffer
	if err := WriteJSON(&out, specsDir, Options{CrossBranch: true}); err != nil {
		t.Fatal(err)
	}
	var data showData
	if err := json.Unmarshal(out.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	var sawSpec, sawAssertion bool
	for _, s := range data.Specs {
		if s.ID == "shiny" && s.Foreign {
			sawSpec = true
			if want := map[string][]string{"owner": {"core"}}; !reflect.DeepEqual(s.CustomFields, want) {
				t.Errorf("foreign spec customFields = %v, want %v", s.CustomFields, want)
			}
		}
	}
	for _, a := range data.Assertions {
		if a.ID == "works" && a.Foreign {
			sawAssertion = true
			if want := map[string][]string{"workflows": {"a", "b"}}; !reflect.DeepEqual(a.CustomFields, want) {
				t.Errorf("foreign assertion customFields = %v, want %v", a.CustomFields, want)
			}
		}
	}
	if !sawSpec || !sawAssertion {
		t.Fatalf("foreign items missing: spec=%v assertion=%v", sawSpec, sawAssertion)
	}
}

// TestShowOutputDeterministicWithCustomFields runs both outputs many times:
// Go randomizes map iteration, so one run could pass by chance.
func TestShowOutputDeterministicWithCustomFields(t *testing.T) {
	specsDir := customFieldsTree(t)
	htmlPath := filepath.Join(filepath.Dir(specsDir), ".spekk", "index.html")

	jsonOutputs := map[string]bool{}
	htmlOutputs := map[string]bool{}
	var first bytes.Buffer
	for i := 0; i < 25; i++ {
		var out bytes.Buffer
		if err := WriteJSON(&out, specsDir, Options{}); err != nil {
			t.Fatal(err)
		}
		jsonOutputs[out.String()] = true
		if i == 0 {
			first = out
		}
		if _, err := writeExplorer(specsDir, Options{}); err != nil {
			t.Fatal(err)
		}
		html, err := os.ReadFile(htmlPath)
		if err != nil {
			t.Fatal(err)
		}
		htmlOutputs[string(html)] = true
	}
	if len(jsonOutputs) != 1 {
		t.Errorf("--json gave %d distinct outputs over 25 runs, want 1", len(jsonOutputs))
	}
	if len(htmlOutputs) != 1 {
		t.Errorf("index.html had %d distinct contents over 25 runs, want 1", len(htmlOutputs))
	}

	// Keys are in byte order; values keep frontmatter order, duplicates removed.
	s := first.String()
	i := strings.Index(s, `"customFields":{"alpha-key"`)
	if i < 0 {
		t.Fatalf("assertion customFields do not start with alpha-key: %s", s)
	}
	order := []string{`"alpha-key"`, `"dup"`, `"labels"`, `"owner"`, `"workflows"`, `"zeta"`}
	last := i
	for _, k := range order {
		j := strings.Index(s[last:], k)
		if j < 0 {
			t.Fatalf("key %s out of order in %s", k, s)
		}
		last += j
	}
	if !strings.Contains(s, `"dup":["x","y"]`) {
		t.Errorf("dup values not in frontmatter order without duplicates: %s", s)
	}
}
