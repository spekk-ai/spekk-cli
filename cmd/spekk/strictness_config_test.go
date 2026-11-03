package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/show"
	"github.com/spekk-ai/spekk-cli/internal/verify"
)

func TestValidateAndVerifyRejectInvalidConfig(t *testing.T) {
	cases := map[string]struct{ config, want string }{
		"unknown table":      {"[stric]\nunchecked = \"error\"\n", "unknown table or key stric"},
		"unknown key":        {"[strict]\nunknow = \"error\"\n", "unknown key strict.unknow"},
		"unknown kind":       {"[strict.min-level]\ninvarient = \"property\"\n", "unknown property kind strict.min-level.invarient"},
		"unknown level":      {"[strict.min-level]\ninvariant = \"strong\"\n", "invalid evidence level strict.min-level.invariant"},
		"bad unchecked":      {"[strict]\nunchecked = \"ignore\"\n", "strict.unchecked must be warn or error"},
		"invalid TOML":       {"[strict\nunchecked = \"error\"\n", ""},
		"unknown verify key": {"[[verify.run]]\ncommand = \"true\"\njunit = \"result.xml\"\nextra = 1\n", "unknown key verify.run[1].extra"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			specsDir := makeCleanValidateFixture(t)
			root := filepath.Dir(specsDir)
			if err := os.WriteFile(filepath.Join(root, "spekk.toml"), []byte(tc.config), 0o644); err != nil {
				t.Fatal(err)
			}
			var out, diagnostic bytes.Buffer
			if code := execValidate(nil, &out, &diagnostic, specsDir); code != 1 || out.Len() != 0 || strings.Count(diagnostic.String(), "spekk.toml:") != 1 || strings.Count(strings.TrimSpace(diagnostic.String()), "\n") != 0 || !strings.Contains(diagnostic.String(), tc.want) {
				t.Fatalf("exit=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
			}
		})
	}
	specsDir := makeCleanValidateFixture(t)
	root := filepath.Dir(specsDir)
	path := filepath.Join(root, "spekk.toml")
	config := "[[verify.run]]\ncommand = \"touch ran\"\njunit = \"result.xml\"\n[strict]\nunchecked = \"ignore\"\nextra = true\n[strict.min-level]\ninvarient = \"strong\"\n"
	if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	if code := execValidate(nil, &out, &diagnostic, specsDir); code != 1 || strings.Count(diagnostic.String(), "spekk.toml:") != 3 {
		t.Fatalf("multiple problems: exit=%d stderr=%q", code, diagnostic.String())
	}
	out.Reset()
	diagnostic.Reset()
	if err := verify.Run(specsDir, nil, &out, &diagnostic); err != verify.ErrInvalidConfig || out.Len() != 0 || strings.Count(diagnostic.String(), "spekk.toml:") != 3 {
		t.Fatalf("verify invalid: %v %q %q", err, out.String(), diagnostic.String())
	}
	if _, err := os.Stat(filepath.Join(root, "ran")); !os.IsNotExist(err) {
		t.Fatal("invalid config ran a command")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	diagnostic.Reset()
	if code := execValidate(nil, &out, &diagnostic, specsDir); code != 0 || !strings.Contains(out.String(), "assertions OK") {
		t.Fatalf("missing config: exit=%d stdout=%q stderr=%q", code, out.String(), diagnostic.String())
	}
}

func TestStrictnessLeavesDisplayUnchanged(t *testing.T) {
	specsDir := truthProject(t, true)
	root := filepath.Dir(specsDir)
	for _, id := range []string{"pass", "plain"} {
		path := filepath.Join(specsDir, "demo", "assertions", id+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(data), "---\n#", "anchors: [checks_test.go]\n---\n#", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commitFixture := func(message string) {
		t.Helper()
		for _, args := range [][]string{{"add", "."}, {"commit", "-q", "-m", message}} {
			if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v: %s", args, err, out)
			}
		}
	}
	commitFixture("Name source")
	checkPath := filepath.Join(root, "checks_test.go")
	check, err := os.ReadFile(checkPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(checkPath, append(check, []byte("// New line.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	commitFixture("Change source")
	var verifyOut, verifyDiagnostic bytes.Buffer
	if err := verify.Run(specsDir, nil, &verifyOut, &verifyDiagnostic); err == nil || !strings.Contains(verifyOut.String(), "1 unconfirmed") {
		t.Fatalf("verify: %v %q %q", err, verifyOut.String(), verifyDiagnostic.String())
	}
	outputs := func() map[string]string {
		t.Helper()
		result := map[string]string{"status": statusOutput(t, specsDir)}
		for _, mode := range []struct {
			name string
			args []string
		}{
			{"table", nil}, {"tsv", []string{"--tsv"}}, {"csv", []string{"--csv"}}, {"json", []string{"--json"}},
		} {
			var out, diagnostic bytes.Buffer
			if code := execList(mode.args, &out, &diagnostic, specsDir); code != 0 {
				t.Fatalf("list %s: exit=%d stderr=%q", mode.name, code, diagnostic.String())
			}
			result["list "+mode.name] = out.String()
		}
		var out bytes.Buffer
		if err := show.WriteJSON(&out, specsDir, show.Options{}); err != nil {
			t.Fatal(err)
		}
		result["show json"] = out.String()
		for _, mode := range []struct {
			name string
			args []string
		}{
			{"table", []string{filepath.Join(root, "checks_test.go")}},
			{"json", []string{filepath.Join(root, "checks_test.go"), "--json"}},
		} {
			var out, diagnostic bytes.Buffer
			if code := execConstraints(mode.args, &out, &diagnostic, specsDir); code != 0 {
				t.Fatalf("constraints %s: exit=%d stderr=%q", mode.name, code, diagnostic.String())
			}
			result["constraints "+mode.name] = out.String()
		}
		return result
	}
	before := outputs()
	if !strings.Contains(before["list table"], "pass") || !strings.Contains(before["show json"], `"truth":"verified"`) || !strings.Contains(before["show json"], `"truth":"unconfirmed"`) || !strings.Contains(before["constraints table"], "pass") {
		t.Fatalf("missing data row: list=%q show=%q constraints=%q", before["list table"], before["show json"], before["constraints table"])
	}
	f, err := os.OpenFile(filepath.Join(root, "spekk.toml"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\n[strict]\nunchecked = \"error\"\n[strict.min-level]\nexample = \"proof\"\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	after := outputs()
	for command, want := range before {
		if got := after[command]; got != want {
			t.Errorf("%s changed with strictness:\nbefore=%q\nafter=%q", command, want, got)
		}
	}
}
