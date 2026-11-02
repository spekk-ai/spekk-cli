package parser

import (
	"reflect"
	"testing"
)

func TestPropertyKindAndAnchors(t *testing.T) {
	for _, tc := range []struct {
		frontmatter string
		kind        string
		anchors     []string
	}{
		{"", "example", nil},
		{"kind: invariant  # note\nanchors: [a.py, b.py]\n", "invariant", []string{"a.py", "b.py"}},
		{"kind: ordering\nanchors: a.py, b.py\n", "ordering", []string{"a.py", "b.py"}},
		{"kind: response\nanchors:\n  - a.py\n  - b.py\n", "response", []string{"a.py", "b.py"}},
		{"kind: authorization\n", "authorization", nil},
		{"kind: example\n", "example", nil},
		{"kind: invarient\n", "example", nil},
		{"kind: [invariant]\n", "example", nil},
	} {
		content := "---\nid: property\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: not_started\n" + tc.frontmatter + "---\n# Property\n"
		a, err := ParseAssertionContent("specs/sample/assertions/property.md", content)
		if err != nil {
			t.Fatal(err)
		}
		if a.Kind != tc.kind || !reflect.DeepEqual(a.Anchors, tc.anchors) {
			t.Errorf("frontmatter %q: kind %q, anchors %v", tc.frontmatter, a.Kind, a.Anchors)
		}
	}
}

func TestPropertySections(t *testing.T) {
	content := "---\nid: property\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Property\n\n## TERMS\n\n  balance: value\n### Detail\nMore text\n\n## Other\nIgnore\n## Decisions\n\nCommit only.\n\n# End\n## Rationale\nReason.\n"
	a, err := ParseAssertionContent("property.md", content)
	if err != nil {
		t.Fatal(err)
	}
	want := []PropertySection{{"terms", "  balance: value\n### Detail\nMore text"}, {"decisions", "Commit only."}, {"rationale", "Reason."}}
	if !reflect.DeepEqual(a.Sections, want) {
		t.Fatalf("sections = %#v, want %#v", a.Sections, want)
	}
}
