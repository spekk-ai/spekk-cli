package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/show"
	"github.com/spekk-ai/spekk-cli/internal/status"
	"github.com/spekk-ai/spekk-cli/internal/verify"
)

func truthProject(t *testing.T, record bool) string {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	write := func(path, value string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q")
	git("config", "user.email", "test@example.test")
	git("config", "user.name", "Test")
	write(".gitignore", ".spekk/\n")
	write("go.mod", "module example.test/fixture\n\ngo 1.25\n")
	write("specs/demo/demo.md", "---\nid: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\n---\n# Demo\n")
	for id, state := range map[string]string{"pass": "done", "fail": "done", "plain": "not_started", "draft": "draft"} {
		write("specs/demo/assertions/"+id+".md", "---\nid: "+id+"\nparent: demo\ncreated: 2026-01-01T00:00:00Z\npriority: 1\nstatus: "+state+"\n---\n# "+id+"\n")
	}
	write("checks_test.go", "package fixture\n\nimport \"testing\"\n\n// spekk: pass evidence=property\nfunc TestPass(t *testing.T) {}\n// spekk: fail evidence=example\nfunc TestFail(t *testing.T) {}\n// spekk: draft evidence=contract\nfunc TestDraft(t *testing.T) {}\n")
	git("add", ".")
	git("commit", "-q", "-m", "Add fixture")
	if record {
		write("source.xml", `<testsuite><testcase classname="example.test/fixture" name="TestPass"/><testcase classname="example.test/fixture" name="TestFail"><failure/></testcase><testcase classname="example.test/fixture" name="TestDraft"/></testsuite>`)
		write("spekk.toml", "[[verify.run]]\ncommand = \"cp source.xml result.xml\"\njunit = \"result.xml\"\n")
		var out, diagnostic bytes.Buffer
		if err := verify.Run(filepath.Join(root, "specs"), nil, &out, &diagnostic); err == nil || !strings.Contains(out.String(), "1 failing") {
			t.Fatalf("verify: %v, %s, %s", err, out.String(), diagnostic.String())
		}
	}
	return filepath.Join(root, "specs")
}

func statusOutput(t *testing.T, specsDir string) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	if err := status.Show(specsDir); err != nil {
		t.Fatal(err)
	}
	w.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTruthStatusDisplay(t *testing.T) {
	output := statusOutput(t, truthProject(t, true))
	for _, want := range []string{"pass · property · verified", "fail · failing", "plain · unchecked", "draft · contract · verified", "2/3 assertions complete · 1 draft · 1 verified · 1 failing · 1 unchecked", "Total: 3 assertions", "🔎 Truth: 1 verified · 1 failing · 1 unchecked · 0 unconfirmed", "Results are from a run with uncommitted changes"} {
		if !strings.Contains(output, want) {
			t.Errorf("missing %q in status output", want)
		}
	}
	if got := statusOutput(t, truthProject(t, false)); !strings.Contains(got, "🔎 Truth: 0 verified · 0 failing · 3 unchecked · 0 unconfirmed") {
		t.Fatal("no run must show unchecked")
	}
	proseSpecs := truthProject(t, false)
	root := filepath.Dir(proseSpecs)
	plain := filepath.Join(proseSpecs, "demo", "assertions", "plain.md")
	b, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plain, []byte(strings.Replace(string(b), "status: not_started\n", "status: not_started\nanchors: [code.go]\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	code := filepath.Join(root, "code.go")
	if err := os.WriteFile(code, []byte("package fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "Anchor prose"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	if err := os.WriteFile(code, []byte("package fixture\nvar changed = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", "Change code"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	if got := statusOutput(t, proseSpecs); !strings.Contains(got, "🔎 Truth: 0 verified · 0 failing · 2 unchecked · 1 unconfirmed") || !strings.Contains(got, "plain · unconfirmed") {
		t.Fatal("changed prose anchor must be unconfirmed", got)
	}
}

func TestTruthListFormats(t *testing.T) {
	specsDir := truthProject(t, true)
	output := func(args ...string) string {
		t.Helper()
		var out, diagnostic bytes.Buffer
		if code := execList(args, &out, &diagnostic, specsDir); code != 0 {
			t.Fatalf("list %v: exit %d: %s", args, code, diagnostic.String())
		}
		return out.String()
	}
	table := output()
	headerLine := strings.Split(table, "\n")[0]
	if strings.Index(headerLine, "STATUS") >= strings.Index(headerLine, "TRUTH") || !strings.Contains(table, "property · verified") || !strings.Contains(table, "unchecked") {
		t.Fatal(table)
	}
	for _, args := range [][]string{{"--tsv"}, {"--csv"}, {"--long", "--status", "done", "--csv"}, {"--sort-by", "title", "--tsv"}} {
		out := output(args...)
		sep := "\t"
		if strings.Contains(out, ",") && !strings.Contains(out, "\t") {
			sep = ","
		}
		header := strings.Split(strings.TrimSuffix(strings.Split(out, "\n")[0], "\r"), sep)
		if strings.Join(header[:5], ",") != "id,status,pri,parent,title" || strings.Join(header[len(header)-2:], ",") != "truth,level" {
			t.Fatalf("changed v1 column positions: %v", header)
		}
		if args[0] == "--long" && header[5] != "file" {
			t.Fatalf("changed v1 FILE column position: %v", header)
		}
	}
	records, err := csv.NewReader(strings.NewReader(output("--csv"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 5 {
		t.Fatal(records)
	}
	var flat struct {
		Assertions []map[string]any `json:"assertions"`
	}
	if err := json.Unmarshal([]byte(output("--json")), &flat); err != nil {
		t.Fatal(err)
	}
	for _, a := range flat.Assertions {
		if a["truth"] == nil || a["level"] == nil {
			t.Fatal(a)
		}
	}
	var noRun, diagnostic bytes.Buffer
	if code := execList([]string{"--json"}, &noRun, &diagnostic, truthProject(t, false)); code != 0 || !strings.Contains(noRun.String(), `"truth": "unchecked"`) {
		t.Fatalf("list without a run: exit %d: %s %s", code, noRun.String(), diagnostic.String())
	}
}

func TestTruthShowData(t *testing.T) {
	specsDir := truthProject(t, true)
	var out bytes.Buffer
	if err := show.WriteJSON(&out, specsDir, show.Options{}); err != nil {
		t.Fatal(err)
	}
	var data struct {
		Assertions []struct{ ID, Status, Truth, Level string }
	}
	if err := json.Unmarshal(out.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"pass": "property:verified", "fail": ":failing", "plain": ":unchecked", "draft": "contract:verified"}
	for _, a := range data.Assertions {
		if got := a.Level + ":" + a.Truth; got != want[a.ID] {
			t.Errorf("%s: got %s, want %s", a.ID, got, want[a.ID])
		}
	}
	var again bytes.Buffer
	if err := show.WriteJSON(&again, specsDir, show.Options{}); err != nil || out.String() != again.String() {
		t.Fatal("show output changed between runs", err)
	}
	root := filepath.Dir(specsDir)
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("new HEAD\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "new.txt"}, {"commit", "-q", "-m", "Advance HEAD"}} {
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(output))
		}
	}
	var nextHead bytes.Buffer
	if err := show.WriteJSON(&nextHead, specsDir, show.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(nextHead.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	for _, a := range data.Assertions {
		if a.Truth != "unchecked" || a.Level != "" {
			t.Fatalf("old verify result leaked to new HEAD: %+v", a)
		}
	}
	var noRun bytes.Buffer
	if err := show.WriteJSON(&noRun, truthProject(t, false), show.Options{}); err != nil || !strings.Contains(noRun.String(), `"truth":"unchecked"`) {
		t.Fatal("show without a run must show unchecked", err)
	}
}
