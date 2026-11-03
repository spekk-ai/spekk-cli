package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIEntryPoint(t *testing.T) {
	if os.Getenv("SPEKK_TEST_CLI_ENTRY") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"spekk"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(2)
}

func runCLIEntry(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^TestCLIEntryPoint$", "--"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SPEKK_TEST_CLI_ENTRY=1", "PWD="+dir)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), stdout.String(), stderr.String()
	}
	t.Fatalf("run spekk %v: %v", args, err)
	return 0, "", ""
}

func TestConstraintsFromSymlinkedWorkingDirectory(t *testing.T) {
	root := constraintProject(t)
	link := filepath.Join(filepath.Dir(root), "plink")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"constraints", "code.go"}, {"constraints", "./code.go", "--json"}} {
		code, output, diagnostic := runCLIEntry(t, link, args...)
		if code != 0 || diagnostic != "" || !strings.Contains(output, "alpha") || !strings.Contains(output, "beta") {
			t.Fatalf("symlinked working directory %v: code %d, stdout %q, stderr %q", args, code, output, diagnostic)
		}
	}
	code, output, diagnostic := runCLIEntry(t, link, "constraints", "missing/new.go")
	if code != 0 || output != "" || diagnostic != "no properties anchor on missing/new.go\n" {
		t.Fatalf("missing file: code %d, stdout %q, stderr %q", code, output, diagnostic)
	}
	outside := filepath.Join(filepath.Dir(root), "outside", "x.go")
	writeConstraintFile(t, filepath.Dir(root), "outside/x.go", "package outside\n")
	if err := os.Symlink(outside, filepath.Join(root, "linkfile.go")); err != nil {
		t.Fatal(err)
	}
	code, _, diagnostic = runCLIEntry(t, link, "constraints", "linkfile.go")
	if code != 1 || !strings.Contains(diagnostic, "outside project root") {
		t.Fatalf("outside file symlink: code %d, stderr %q", code, diagnostic)
	}
}

func TestListIgnoresIndexFiles(t *testing.T) {
	for _, oldEntry := range []bool{false, true} {
		root := constraintProject(t)
		if oldEntry {
			writeConstraintFile(t, root, ".gitignore", ".spekk/index.db\n")
			cmd := exec.Command("git", "add", ".gitignore")
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git add: %v: %s", err, out)
			}
			cmd = exec.Command("git", "commit", "-q", "-m", "Add old ignore entry")
			cmd.Dir = root
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git commit: %v: %s", err, out)
			}
		}
		for run := 0; run < 2; run++ {
			code, _, diagnostic := runCLIEntry(t, root, "list", "--tsv")
			if code != 0 {
				t.Fatalf("list run %d, old entry %t: code %d, stderr %q", run, oldEntry, code, diagnostic)
			}
		}
		cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=all")
		cmd.Dir = root
		status, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(status), ".spekk/") {
			t.Fatalf("index files are untracked, old entry %t: %s", oldEntry, status)
		}
		ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(ignore), ".spekk/index.db*\n") != 1 {
			t.Fatalf("new pattern must appear once, old entry %t: %q", oldEntry, ignore)
		}
	}
}
