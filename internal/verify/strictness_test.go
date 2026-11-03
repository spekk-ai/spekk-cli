package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func strictProject(t *testing.T, marker, outcome string) string {
	t.Helper()
	root := project(t)
	for id, status := range map[string]string{"done": "done", "later": "not_started"} {
		write(t, filepath.Join(root, "specs", "sample", "assertions", id+".md"), "---\nid: "+id+"\nparent: sample\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nkind: invariant\nstatus: "+status+"\n---\n# "+id+"\n")
	}
	if marker != "" {
		write(t, filepath.Join(root, "check_test.go"), "// spekk: done evidence="+marker+"\nfunc TestCheck(t *testing.T) {}\n")
	}
	commit(t, root, "Add properties")
	caseXML := ""
	if marker != "" {
		caseXML = `<testcase classname="example.test/acme" name="TestCheck">` + outcome + `</testcase>`
	}
	write(t, filepath.Join(root, "source.xml"), "<testsuite>"+caseXML+"</testsuite>")
	return root
}

func strictConfig(t *testing.T, root, rules string) {
	t.Helper()
	config(t, root, "cp source.xml result.xml", "result.xml")
	f, err := os.OpenFile(filepath.Join(root, "spekk.toml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(rules); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestStrictUncheckedAndUnconfirmed(t *testing.T) {
	root := strictProject(t, "", "")
	for _, tc := range []struct {
		rules, want string
		fails       bool
	}{
		{"", "warning: strict: done is unchecked\n", false},
		{"[strict]\nunchecked = \"warn\"\n", "warning: strict: done is unchecked\n", false},
		{"[strict]\nunchecked = \"error\"\n", "strict: done is unchecked\n", true},
	} {
		strictConfig(t, root, tc.rules)
		out, diagnostic, err := run(t, root)
		if (err != nil) != tc.fails || diagnostic != tc.want || !strings.Contains(out, "2 unchecked") {
			t.Fatalf("rules=%q stdout=%q stderr=%q err=%v", tc.rules, out, diagnostic, err)
		}
	}
	root = strictProject(t, "", "")
	path := filepath.Join(root, "specs", "sample", "assertions", "done.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, strings.Replace(string(data), "status: done\n", "status: done\nanchors: [code.go]\n", 1))
	write(t, filepath.Join(root, "code.go"), "package acme\n")
	commit(t, root, "Name source")
	write(t, filepath.Join(root, "code.go"), "package acme\n// Changed source.\n")
	commit(t, root, "Change source")
	strictConfig(t, root, "[strict]\nunchecked = \"error\"\n")
	out, diagnostic, err := run(t, root)
	if err == nil || diagnostic != "strict: done is unconfirmed\n" || !strings.Contains(out, "1 unconfirmed") {
		t.Fatalf("unconfirmed: stdout=%q stderr=%q err=%v", out, diagnostic, err)
	}
}

func TestStrictMinimumLevelAndFailingTruth(t *testing.T) {
	root := strictProject(t, "property", "")
	for _, tc := range []struct {
		minimum, want string
		fails         bool
	}{
		{"property", "", false},
		{"contract", "strict: done is property, below contract for invariant\n", true},
	} {
		strictConfig(t, root, "[strict.min-level]\ninvariant = \""+tc.minimum+"\"\n")
		out, diagnostic, err := run(t, root)
		if (err != nil) != tc.fails || diagnostic != tc.want || !strings.Contains(out, "1 verified") {
			t.Fatalf("minimum=%s stdout=%q stderr=%q err=%v", tc.minimum, out, diagnostic, err)
		}
	}
	root = strictProject(t, "example", "<failure/>")
	for _, rules := range []string{"", "[strict]\nunchecked = \"error\"\n"} {
		strictConfig(t, root, rules)
		out, diagnostic, err := run(t, root)
		if err == nil || !strings.Contains(out, "done: failing") || strings.Contains(diagnostic, "strict: done is unchecked") {
			t.Fatalf("failing rules=%q stdout=%q stderr=%q err=%v", rules, out, diagnostic, err)
		}
	}
	root = strictProject(t, "", "")
	strictConfig(t, root, "[strict.min-level]\ninvariant = \"example\"\n")
	_, diagnostic, err := run(t, root)
	if err == nil || !strings.Contains(diagnostic, "strict: done is no level, below example for invariant") {
		t.Fatalf("empty level: stderr=%q err=%v", diagnostic, err)
	}
}
